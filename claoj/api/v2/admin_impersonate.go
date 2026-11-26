package v2

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	authHandlers "github.com/CLAOJ/claoj/api/v2/auth"
	"github.com/CLAOJ/claoj/auditlog"
	"github.com/CLAOJ/claoj/auth"
	"github.com/CLAOJ/claoj/auth/tokenstore"
	"github.com/CLAOJ/claoj/cookie"
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// AdminImpersonateStart - POST /api/admin/impersonate/:userId
// Opens a session as judge_profile :userId (same id space as /admin/users).
// Superuser-only, matching v1's django-impersonate. The actor's own tokens
// are left intact: the browser's cookies are replaced, not revoked, so the
// original session survives until it expires.
func AdminImpersonateStart(c *gin.Context) {
	actor := auth.GetAccess(c)
	if c.GetUint("impersonator_id") != 0 {
		// Checked before the superuser gate so an impersonated staff session
		// can never chain into a second impersonation.
		c.JSON(http.StatusBadRequest, apiUserError("already impersonating"))
		return
	}
	// A deactivated account keeps a usable access token until it expires, so
	// the active flag is checked here too, not only at login.
	if !actor.IsSuperuser || !actor.IsActive {
		c.JSON(http.StatusForbidden, apiUserError("superuser access required"))
		return
	}

	profileID, err := strconv.ParseUint(c.Param("userId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, apiUserError("invalid user ID"))
		return
	}

	var profile models.Profile
	if err := db.DB.Preload("User").First(&profile, uint(profileID)).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, apiUserError("user not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}
	target := profile.User

	// The self case is technically subsumed by the superuser-target case
	// (the actor must be a superuser to reach this point), but it gets its own
	// message so the UI can explain what happened.
	switch {
	case target.ID == actor.UserID:
		c.JSON(http.StatusBadRequest, apiUserError("cannot impersonate yourself"))
		return
	case target.IsSuperuser:
		c.JSON(http.StatusBadRequest, apiUserError("cannot impersonate a superuser"))
		return
	case !target.IsActive:
		c.JSON(http.StatusBadRequest, apiUserError("cannot impersonate an inactive user"))
		return
	}

	if err := startSession(c, target, actor.UserID); err != nil {
		c.JSON(http.StatusInternalServerError, apiUserError("failed to generate tokens"))
		return
	}

	actorName := c.GetString("username")
	if actorName == "" {
		var u models.AuthUser
		if err := db.DB.Select("username").First(&u, actor.UserID).Error; err == nil {
			actorName = u.Username
		}
	}

	writeImpersonationAudit(c, actor.UserID, actorName, "impersonate_start", profile.ID, gin.H{
		"impersonator_id":   actor.UserID,
		"impersonated_id":   target.ID,
		"impersonated_name": target.Username,
	})

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"user": gin.H{
			"id":       profile.ID,
			"username": target.Username,
		},
	})
}

// AdminImpersonateExit - POST /api/admin/impersonate/exit
// Requires an impersonating session. Revokes the impersonated token family
// and reissues tokens for the superuser who started it.
//
// Revocation happens BEFORE the actor is re-checked: a demoted, deactivated
// or deleted actor must not be able to keep a live impersonated session just
// because the exit request is refused. In that case the request answers 403
// and the family is already dead.
//
// It sits outside the admin route group on purpose: an impersonated account
// is usually not staff, so it cannot pass AdminRequiredMiddleware — and the
// exit button must stay reachable from that session.
func AdminImpersonateExit(c *gin.Context) {
	impersonatorID := c.GetUint("impersonator_id")
	if impersonatorID == 0 {
		c.JSON(http.StatusBadRequest, apiUserError("not impersonating"))
		return
	}

	if familyID := c.GetString("family_id"); familyID != "" && authHandlers.RefreshStore != nil {
		if err := authHandlers.RefreshStore.RevokeFamily(familyID); err != nil {
			log.Printf("impersonate: revoke family %s: %v", familyID, err)
		}
	}

	var actor models.AuthUser
	if err := db.DB.Select("id", "username", "is_active", "is_superuser").First(&actor, impersonatorID).Error; err != nil {
		c.JSON(http.StatusUnauthorized, apiUserError("impersonator not found"))
		return
	}
	// A demoted or deactivated actor must not get their session back; the
	// impersonated family is already revoked above.
	if !actor.IsActive || !actor.IsSuperuser {
		c.JSON(http.StatusForbidden, apiUserError("impersonator is no longer a superuser"))
		return
	}

	if err := startSession(c, actor, 0); err != nil {
		c.JSON(http.StatusInternalServerError, apiUserError("failed to generate tokens"))
		return
	}

	writeImpersonationAudit(c, actor.ID, actor.Username, "impersonate_exit", actor.ID, gin.H{
		"impersonator_id": actor.ID,
	})

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"user": gin.H{
			"id":       actor.ID,
			"username": actor.Username,
		},
	})
}

// startSession issues a fresh token pair for user (impersonatorID 0 for a
// normal session) and installs the auth cookies.
func startSession(c *gin.Context, user models.AuthUser, impersonatorID uint) error {
	accessToken, refreshToken, familyID, err := auth.GenerateTokensAs(
		user.ID, user.Username, user.IsSuperuser, "", false, impersonatorID)
	if err != nil {
		return err
	}
	if authHandlers.RefreshStore != nil {
		ttl := 7 * 24 * time.Hour
		if impersonatorID != 0 {
			ttl = auth.ImpersonationRefreshTTL
		}
		if err := authHandlers.RefreshStore.Save(refreshToken, tokenstore.Entry{
			UserID:    user.ID,
			FamilyID:  familyID,
			ExpiresAt: time.Now().Add(ttl),
			CreatedAt: time.Now(),
			UserAgent: c.Request.UserAgent(),
			ClientIP:  c.ClientIP(),
		}); err != nil {
			log.Printf("impersonate: save refresh token for user %d: %v", user.ID, err)
		}
	}
	cookie.Helper().SetAuthTokens(c, accessToken, refreshToken, cookie.RefreshTokenDuration)
	return nil
}

// writeImpersonationAudit records a start/exit event explicitly, so the actor
// and target survive in the trail even though the request audit entry only
// sees the session's current identity.
func writeImpersonationAudit(c *gin.Context, userID uint, username, action string, resourceID uint, details gin.H) {
	store := auditlog.Default
	if store == nil {
		log.Printf("impersonate: audit store unavailable, dropping %s entry for user %d", action, userID)
		return
	}
	raw, _ := json.Marshal(details)
	entry := auditlog.Entry{
		UserID:     userID,
		Username:   username,
		Action:     action,
		Resource:   "User",
		ResourceID: strconv.FormatUint(uint64(resourceID), 10),
		IPAddress:  c.ClientIP(),
		UserAgent:  c.Request.UserAgent(),
		Details:    string(raw),
		Status:     "success",
		CreatedAt:  time.Now(),
	}
	if err := store.Write(entry); err != nil {
		log.Printf("impersonate: audit write %s failed: %v", action, err)
	}
}
