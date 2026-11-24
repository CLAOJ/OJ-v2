package v2

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	authHandlers "github.com/CLAOJ/claoj/api/v2/auth"
	"github.com/CLAOJ/claoj/auth"
	"github.com/CLAOJ/claoj/auth/tokenstore"
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Registration (account activation) admin.
//
// v2 verifies email addresses with one-time tokens (tokenstore.KindEmailVerify
// plus auth_user.is_active); the v1 django-registration-redux row in
// registration_registrationprofile is shown for parity and marked activated on
// manual activation, but it is not the source of truth.

// adminRegistrationV1 is the read-only v1 registration row state.
type adminRegistrationV1 struct {
	ID               uint `json:"id"`
	Activated        bool `json:"activated"`
	HasActivationKey bool `json:"has_activation_key"`
}

// AdminRegistrationList - GET /api/v2/admin/registrations
//
// Lists accounts by activation state. Pending means auth_user.is_active = 0,
// which is what v2 registration creates until the email token is consumed.
func AdminRegistrationList(c *gin.Context) {
	page, pageSize := parsePagination(c)

	status := c.DefaultQuery("status", "all")
	switch status {
	case "pending", "active", "all":
	default:
		c.JSON(http.StatusBadRequest, apiUserError("invalid status: must be pending, active or all"))
		return
	}
	order := c.DefaultQuery("order", "desc")
	if order != "asc" && order != "desc" {
		c.JSON(http.StatusBadRequest, apiUserError("invalid order: must be asc or desc"))
		return
	}

	query := db.DB.Table("auth_user").
		Select(`auth_user.id AS user_id, auth_user.username, auth_user.email,
			auth_user.is_active, auth_user.date_joined,
			rp.id AS profile_id, rp.activated AS v1_activated,
			rp.activation_key AS v1_key`).
		Joins("LEFT JOIN registration_registrationprofile rp ON rp.user_id = auth_user.id")

	if status == "pending" {
		query = query.Where("auth_user.is_active = ?", false)
	} else if status == "active" {
		query = query.Where("auth_user.is_active = ?", true)
	}
	if search := c.Query("search"); search != "" {
		pattern := db.ContainsPattern(search)
		query = query.Where("auth_user.username LIKE ? ESCAPE '!' OR auth_user.email LIKE ? ESCAPE '!'", pattern, pattern)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError("database error"))
		return
	}

	var rows []struct {
		UserID      uint      `gorm:"column:user_id"`
		Username    string    `gorm:"column:username"`
		Email       string    `gorm:"column:email"`
		IsActive    bool      `gorm:"column:is_active"`
		DateJoined  time.Time `gorm:"column:date_joined"`
		ProfileID   *uint     `gorm:"column:profile_id"`
		V1Activated *bool     `gorm:"column:v1_activated"`
		V1Key       *string   `gorm:"column:v1_key"`
	}
	if err := query.
		Order("auth_user.date_joined " + order + ", auth_user.id " + order).
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Scan(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError("database error"))
		return
	}

	items := make([]gin.H, len(rows))
	for i, r := range rows {
		var registration *adminRegistrationV1
		if r.ProfileID != nil {
			registration = &adminRegistrationV1{ID: *r.ProfileID}
			if r.V1Activated != nil {
				registration.Activated = *r.V1Activated
			}
			if r.V1Key != nil {
				registration.HasActivationKey = *r.V1Key != ""
			}
		}

		hasToken := false
		if authHandlers.OneTimeTokens != nil {
			ok, err := authHandlers.OneTimeTokens.HasOutstanding(tokenstore.KindEmailVerify, r.UserID)
			if err != nil {
				// Degrade to "no token" rather than failing the whole list.
				log.Printf("admin registrations: outstanding token lookup for user %d: %v", r.UserID, err)
			} else {
				hasToken = ok
			}
		}

		items[i] = gin.H{
			"user_id":           r.UserID,
			"username":          r.Username,
			"email":             r.Email,
			"is_active":         r.IsActive,
			"date_joined":       r.DateJoined,
			"registration":      registration,
			"has_pending_token": hasToken,
		}
	}

	c.JSON(http.StatusOK, apiListWithTotal(items, total))
}

// registrationTarget loads the auth_user behind :userId. Responses are
// written on failure.
func registrationTarget(c *gin.Context) (models.AuthUser, bool) {
	userID, err := strconv.ParseUint(c.Param("userId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid user ID"))
		return models.AuthUser{}, false
	}
	var u models.AuthUser
	if err := db.DB.First(&u, uint(userID)).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, apiError("user not found"))
			return u, false
		}
		c.JSON(http.StatusInternalServerError, apiError("database error"))
		return u, false
	}
	return u, true
}

// AdminRegistrationActivate - POST /api/v2/admin/registration/:userId/activate
//
// Activates the account (is_active = true) and marks the v1 registration row
// activated when it exists. Outstanding verification tokens are revoked
// because the account no longer needs them. Sessions are not touched: this
// only lets the user log in. Idempotent.
func AdminRegistrationActivate(c *gin.Context) {
	if !auth.GetAccess(c).IsSuperuser {
		c.JSON(http.StatusForbidden, apiError("superuser access required"))
		return
	}

	target, ok := registrationTarget(c)
	if !ok {
		return
	}
	if target.IsActive {
		c.JSON(http.StatusOK, gin.H{
			"success":        true,
			"already_active": true,
			"message":        "account is already active",
		})
		return
	}

	err := db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.AuthUser{}).Where("id = ?", target.ID).
			Update("is_active", true).Error; err != nil {
			return err
		}
		// v1 activation blanks the key: the account can log in, so the
		// activation link must stop working.
		return tx.Model(&models.RegistrationProfile{}).
			Where("user_id = ?", target.ID).
			Updates(map[string]any{"activated": true, "activation_key": ""}).Error
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, apiError("failed to activate account"))
		return
	}

	if authHandlers.OneTimeTokens != nil {
		if err := authHandlers.OneTimeTokens.Invalidate(tokenstore.KindEmailVerify, target.ID); err != nil {
			// Best-effort: the account is active, a stale token is harmless.
			log.Printf("admin registrations: invalidate tokens of user %d: %v", target.ID, err)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success":        true,
		"already_active": false,
		"message":        "account activated",
	})
}

// AdminRegistrationResend - POST /api/v2/admin/registration/:userId/resend
//
// Replaces any outstanding verification token with a fresh one and emails the
// verification link through the existing helper. Already-active accounts get
// a 400: there is nothing left to verify.
func AdminRegistrationResend(c *gin.Context) {
	if !auth.GetAccess(c).IsSuperuser {
		c.JSON(http.StatusForbidden, apiError("superuser access required"))
		return
	}

	target, ok := registrationTarget(c)
	if !ok {
		return
	}
	if target.IsActive {
		c.JSON(http.StatusBadRequest, apiUserError("account is already active"))
		return
	}

	// No per-action rate limiter exists in this codebase; the endpoint is
	// superuser-only and the helper replaces (invalidates) the previous token,
	// so a resend cannot be used to fan out links to the same account.
	if err := SendVerificationEmailOnRegistration(target.ID, target.Email, target.Username); err != nil {
		log.Printf("admin registrations: resend verification to user %d: %v", target.ID, err)
		c.JSON(http.StatusInternalServerError, apiError("failed to send verification email"))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "verification email sent",
	})
}
