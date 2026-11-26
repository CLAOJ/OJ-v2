package v2

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	authHandlers "github.com/CLAOJ/claoj/api/v2/auth"
	"github.com/CLAOJ/claoj/auth"
	"github.com/CLAOJ/claoj/auth/tokenstore"
	"github.com/CLAOJ/claoj/cache"
	"github.com/CLAOJ/claoj/config"
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/email"
	"github.com/CLAOJ/claoj/models"
	"github.com/CLAOJ/claoj/ratelimit"
	"github.com/gin-gonic/gin"
)

// Self-service email change (v1 EmailChangeRequestView / EmailChangeActivateView).
//
// v1 signs a TimestampSigner payload with its own key; v2 reuses the one-time
// token store instead (single-use + TTL, no new table). The token is
// "<random>.<base64url(new email)>"; only a hash of the whole string is stored,
// so the address is not persisted outside the user row until confirmation.

const emailChangeTTL = time.Hour

// emailChangeLimit mirrors v1 DMOJ_EMAIL_CHANGE_LIMIT_COUNT/WINDOW (10/hour).
var emailChangeLimit = ratelimit.Limit{Rate: 10, Window: 3600}

// emailChangeLimiter is the Redis-less fallback (tests, single-process dev).
var emailChangeLimiter ratelimit.Repository = ratelimit.NewInMemoryRepository(3600)

// Seams so tests can observe notifications without SMTP.
var (
	sendEmailChangeActivation = func(to, username, newEmail, link string) error {
		return email.SendEmailChangeEmail(to, username, newEmail, link, true)
	}
	sendEmailChangeNotice = func(to, username, newEmail string) error {
		return email.SendEmailChangeEmail(to, username, newEmail, "", false)
	}
)

func emailChangeAllowed(userID uint) bool {
	var repo ratelimit.Repository = emailChangeLimiter
	if cache.Client != nil {
		repo = ratelimit.NewRedisRepository(cache.Client)
	}
	allowed, err := repo.IsAllowed(context.Background(),
		"emailchange:"+strconv.FormatUint(uint64(userID), 10), emailChangeLimit)
	// Fail open on a storage error, like ratelimit.Middleware: the password
	// check still guards the endpoint.
	return err != nil || allowed
}

func newEmailChangeToken(userID uint, newEmail string) (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	payload := base64.RawURLEncoding.EncodeToString(
		[]byte(strconv.FormatUint(uint64(userID), 10) + ":" + newEmail))
	return hex.EncodeToString(buf) + "." + payload, nil
}

// emailChangeTokenPayload returns the account id and new address encoded in a
// token, so a mismatched account is rejected before the single-use token is
// consumed (v1 checked the signer payload first too).
func emailChangeTokenPayload(token string) (uint, string, bool) {
	i := strings.IndexByte(token, '.')
	if i < 0 {
		return 0, "", false
	}
	raw, err := base64.RawURLEncoding.DecodeString(token[i+1:])
	if err != nil {
		return 0, "", false
	}
	idStr, email, ok := strings.Cut(string(raw), ":")
	if !ok {
		return 0, "", false
	}
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return 0, "", false
	}
	return uint(id), email, true
}

type EmailChangeStartRequest struct {
	NewEmail string `json:"new_email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// EmailChangeStart - POST /api/v2/auth/email/change
// @Description Start an email change: checks the password, mails a one-time
// @Description confirmation link to the new address and a notice to the old one.
// @Tags Authentication
// @Summary Request an email change
// @Accept json
// @Produce json
// @Param request body EmailChangeStartRequest true "New email and current password"
// @Success 200 {object} map[string]interface{} "Request accepted"
// @Failure 400 {object} map[string]interface{} "Invalid input or taken email"
// @Failure 401 {object} map[string]interface{} "Wrong password"
// @Failure 429 {object} map[string]interface{} "Too many requests"
// @Router /auth/email/change [post]
func EmailChangeStart(c *gin.Context) {
	var req EmailChangeStartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiUserError(err.Error()))
		return
	}
	user, _, ok := resolveUserProfile(c)
	if !ok {
		return
	}

	// v1 EmailChangeForm requires the current password.
	valid, err := auth.CheckPassword(req.Password, user.Password)
	if err != nil || !valid {
		c.JSON(http.StatusUnauthorized, apiUserError("invalid password"))
		return
	}

	if !emailChangeAllowed(user.ID) {
		c.JSON(http.StatusTooManyRequests, apiUserError("you have sent too many email change requests. Please try again later."))
		return
	}

	var taken int64
	if err := db.DB.Model(&models.AuthUser{}).Where("email = ?", req.NewEmail).Count(&taken).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}
	if taken > 0 {
		c.JSON(http.StatusBadRequest, apiUserError("this email address is already taken"))
		return
	}

	token, err := newEmailChangeToken(user.ID, req.NewEmail)
	if err != nil {
		c.JSON(http.StatusInternalServerError, apiUserError("failed to generate token"))
		return
	}
	// Only one live request per user: a new one revokes the previous link.
	if err := authHandlers.OneTimeTokens.Invalidate(tokenstore.KindEmailChange, user.ID); err != nil {
		// Best-effort; Issue would still overwrite the value below.
	}
	if err := authHandlers.OneTimeTokens.Issue(tokenstore.KindEmailChange, token, user.ID, emailChangeTTL); err != nil {
		c.JSON(http.StatusInternalServerError, apiUserError("failed to create email change token"))
		return
	}

	link := config.C.App.SiteFullURL + "/email/change/confirm?token=" + url.QueryEscape(token)
	// Delivery failures must not reveal anything or lose the token: the link
	// stays valid and either mailbox can retry.
	if err := sendEmailChangeActivation(req.NewEmail, user.Username, req.NewEmail, link); err != nil {
		log.Printf("email change: activation mail to %s failed: %v", req.NewEmail, err)
	}
	if err := sendEmailChangeNotice(user.Email, user.Username, req.NewEmail); err != nil {
		log.Printf("email change: notice mail to %s failed: %v", user.Email, err)
	}

	c.JSON(http.StatusOK, gin.H{"message": "please click on the link sent to " + req.NewEmail})
}

type EmailChangeConfirmRequest struct {
	Token string `json:"token" binding:"required"`
}

// EmailChangeConfirm - POST /api/v2/auth/email/change/confirm
// @Description Redeem the one-time email change token for the logged-in user.
// @Tags Authentication
// @Summary Confirm an email change
// @Accept json
// @Produce json
// @Param request body EmailChangeConfirmRequest true "Token from the email"
// @Success 200 {object} map[string]interface{} "Email changed"
// @Failure 400 {object} map[string]interface{} "Invalid/expired token or taken email"
// @Failure 403 {object} map[string]interface{} "Token belongs to another account"
// @Router /auth/email/change/confirm [post]
func EmailChangeConfirm(c *gin.Context) {
	var req EmailChangeConfirmRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiUserError(err.Error()))
		return
	}
	userID := c.GetUint("user_id")

	// The account id travels in the token so a wrong-account click is rejected
	// before the single-use token is burned (v1 checked the signed payload too).
	tokenUserID, newEmail, ok := emailChangeTokenPayload(req.Token)
	if !ok || tokenUserID != userID {
		c.JSON(http.StatusForbidden, apiUserError("please try again while logged in to the account this email change was originally requested from"))
		return
	}

	// Single-use: GETDEL under the hood, so a replay is "invalid or expired".
	if _, ok, err := authHandlers.OneTimeTokens.Consume(tokenstore.KindEmailChange, req.Token); err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	} else if !ok {
		c.JSON(http.StatusBadRequest, apiUserError("invalid or expired activation key. Please try again."))
		return
	}

	// Re-check uniqueness: the address may have been registered since.
	var taken int64
	if err := db.DB.Model(&models.AuthUser{}).Where("email = ?", newEmail).Count(&taken).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}
	if taken > 0 {
		c.JSON(http.StatusBadRequest, apiUserError("the email you originally requested has since been registered by another user. Please try again with a new email."))
		return
	}

	if err := db.DB.Model(&models.AuthUser{}).Where("id = ?", userID).Update("email", newEmail).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiUserError("failed to change email"))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "the email attached to your account has been changed to " + newEmail,
		"email":   newEmail,
	})
}
