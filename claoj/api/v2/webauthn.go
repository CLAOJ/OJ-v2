package v2

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/CLAOJ/claoj/auth"
	"github.com/CLAOJ/claoj/config"
	"github.com/CLAOJ/claoj/cookie"
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"gorm.io/gorm"
)

// WebAuthnConfig holds the WebAuthn configuration
var WebAuthnConfig *webauthn.WebAuthn

// InitWebAuthn initializes the WebAuthn configuration
func InitWebAuthn() error {
	siteURL, err := url.Parse(config.C.App.SiteFullURL)
	if err != nil {
		return fmt.Errorf("failed to parse site URL: %w", err)
	}

	webauthnConfig := &webauthn.Config{
		RPDisplayName:         config.C.Email.FromName,
		RPID:                  siteURL.Host,
		RPOrigins:             []string{config.C.App.SiteFullURL},
		AttestationPreference: protocol.PreferDirectAttestation,
		AuthenticatorSelection: protocol.AuthenticatorSelection{
			AuthenticatorAttachment: protocol.CrossPlatform,
			UserVerification:        protocol.VerificationPreferred,
		},
	}

	if config.C.App.SecretKey == "" {
		return fmt.Errorf("secret key is required for WebAuthn")
	}

	WebAuthnConfig, err = webauthn.New(webauthnConfig)
	if err != nil {
		return fmt.Errorf("failed to initialize WebAuthn: %w", err)
	}

	return nil
}

// webauthnUser implements the webauthn.User interface
type webauthnUser struct {
	userID    uint
	username  string
	creds     []webauthn.Credential
}

func (u *webauthnUser) WebAuthnID() []byte {
	return []byte(fmt.Sprintf("%d", u.userID))
}

func (u *webauthnUser) WebAuthnName() string {
	return u.username
}

func (u *webauthnUser) WebAuthnDisplayName() string {
	return u.username
}

func (u *webauthnUser) WebAuthnCredentials() []webauthn.Credential {
	return u.creds
}

func (u *webauthnUser) WebAuthnIcon() string {
	return ""
}

// loadWebAuthnCredentials loads WebAuthn credentials for a user from the database
func loadWebAuthnCredentials(userID uint) ([]webauthn.Credential, error) {
	var creds []models.WebAuthnCredential
	if err := db.DB.Where("user_id = ?", userID).Find(&creds).Error; err != nil {
		return nil, err
	}

	result := make([]webauthn.Credential, 0, len(creds))
	for _, cred := range creds {
		// Decode credential ID from base64
		credID, err := base64.URLEncoding.DecodeString(cred.CredID)
		if err != nil {
			continue // Skip invalid credentials
		}

		// Decode public key from base64
		publicKey, err := base64.URLEncoding.DecodeString(cred.PublicKey)
		if err != nil {
			continue // Skip invalid credentials
		}

		result = append(result, webauthn.Credential{
			ID:              credID,
			PublicKey:       publicKey,
			AttestationType: "none",
			Transport:       []protocol.AuthenticatorTransport{protocol.USB, protocol.NFC, protocol.BLE, protocol.Internal},
			Authenticator: webauthn.Authenticator{
				SignCount: uint32(cred.Counter),
			},
		})
	}

	return result, nil
}

// WebAuthnBeginRegistrationRequest - POST /api/v2/auth/webauthn/register/begin
type WebAuthnBeginRegistrationRequest struct {
	Password string `json:"password" binding:"required"`
}

// WebAuthnBeginRegistration starts WebAuthn registration
func WebAuthnBeginRegistration(c *gin.Context) {
	userID := c.GetUint("userID")
	username := c.GetString("username")
	cookieHelper := cookie.Helper()

	var req WebAuthnBeginRegistrationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	// Verify password
	var user models.AuthUser
	if err := db.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, apiError("user not found"))
		return
	}

	match, err := auth.CheckPassword(req.Password, user.Password)
	if err != nil || !match {
		c.JSON(http.StatusUnauthorized, apiError("invalid password"))
		return
	}

	// Check if WebAuthn is already enabled
	var profile models.Profile
	if err := db.DB.Where("user_id = ?", userID).First(&profile).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError("database error"))
		return
	}

	if profile.IsWebauthnEnabled {
		c.JSON(http.StatusBadRequest, apiError("WebAuthn is already enabled. Disable it first to reconfigure."))
		return
	}

	// Load existing credentials
	creds, err := loadWebAuthnCredentials(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, apiError("failed to load credentials"))
		return
	}

	u := &webauthnUser{
		userID:   userID,
		username: username,
		creds:    creds,
	}

	// Generate registration options
	options, session, err := WebAuthnConfig.BeginRegistration(u)
	if err != nil {
		c.JSON(http.StatusInternalServerError, apiError(fmt.Sprintf("failed to begin registration: %v", err)))
		return
	}

	// Store session in cookie
	sessionData, _ := json.Marshal(session)
	cookieHelper.SetWebAuthnRegistrationSession(c, base64.StdEncoding.EncodeToString(sessionData))

	c.JSON(http.StatusOK, gin.H{
		"options": options,
	})
}

// WebAuthnFinishRegistrationRequest - POST /api/v2/auth/webauthn/register/finish
type WebAuthnFinishRegistrationRequest struct {
	Response protocol.CredentialCreationResponse `json:"response"`
	Name     string                              `json:"name"`
}

// WebAuthnFinishRegistration completes WebAuthn registration
func WebAuthnFinishRegistration(c *gin.Context) {
	userID := c.GetUint("userID")
	username := c.GetString("username")
	cookieHelper := cookie.Helper()

	var req WebAuthnFinishRegistrationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	// Get session from cookie
	sessionCookie, err := c.Cookie("webauthn_registration_session")
	if err != nil {
		c.JSON(http.StatusBadRequest, apiError("registration session not found"))
		return
	}

	sessionData, err := base64.StdEncoding.DecodeString(sessionCookie)
	if err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid registration session"))
		return
	}

	var session webauthn.SessionData
	if err := json.Unmarshal(sessionData, &session); err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid registration session data"))
		return
	}

	// Load existing credentials
	creds, err := loadWebAuthnCredentials(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, apiError("failed to load credentials"))
		return
	}

	u := &webauthnUser{
		userID:   userID,
		username: username,
		creds:    creds,
	}

	// Parse the response bytes
	responseJSON, _ := json.Marshal(req.Response)
	parsedResponse, err := protocol.ParseCredentialCreationResponseBytes(responseJSON)
	if err != nil {
		c.JSON(http.StatusBadRequest, apiError(fmt.Sprintf("failed to parse response: %v", err)))
		return
	}

	// Validate the response
	credential, err := WebAuthnConfig.CreateCredential(u, session, parsedResponse)
	if err != nil {
		c.JSON(http.StatusBadRequest, apiError(fmt.Sprintf("registration failed: %v", err)))
		return
	}

	// Store credential in database
	newCred := models.WebAuthnCredential{
		UserID:    userID,
		Name:      req.Name,
		CredID:    base64.URLEncoding.EncodeToString(credential.ID),
		PublicKey: base64.URLEncoding.EncodeToString(credential.PublicKey),
		Counter:   int64(credential.Authenticator.SignCount),
	}

	err = db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&newCred).Error; err != nil {
			return err
		}

		// Enable WebAuthn on profile
		if err := tx.Model(&models.Profile{}).Where("user_id = ?", userID).Update("is_webauthn_enabled", true).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, apiError("failed to store credential"))
		return
	}

	// Clear session cookie
	cookieHelper.ClearWebAuthnRegistrationSession(c)

	c.JSON(http.StatusOK, gin.H{
		"message":   "WebAuthn credential registered successfully",
		"credential": gin.H{
			"id":   newCred.ID,
			"name": newCred.Name,
		},
	})
}

// WebAuthnBeginLogin starts WebAuthn login
func WebAuthnBeginLogin(c *gin.Context) {
	cookieHelper := cookie.Helper()
	var req struct {
		Username string `json:"username" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	// Find user by username
	var user models.AuthUser
	if err := db.DB.Where("username = ?", req.Username).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, apiError("invalid credentials"))
		return
	}

	// Check if WebAuthn is enabled for this user
	var profile models.Profile
	if err := db.DB.Where("user_id = ?", user.ID).First(&profile).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError("database error"))
		return
	}

	if !profile.IsWebauthnEnabled {
		c.JSON(http.StatusBadRequest, apiError("WebAuthn is not enabled for this account"))
		return
	}

	// Load credentials
	creds, err := loadWebAuthnCredentials(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, apiError("failed to load credentials"))
		return
	}

	u := &webauthnUser{
		userID:   user.ID,
		username: user.Username,
		creds:    creds,
	}

	// Generate login options
	options, session, err := WebAuthnConfig.BeginLogin(u)
	if err != nil {
		c.JSON(http.StatusInternalServerError, apiError(fmt.Sprintf("failed to begin login: %v", err)))
		return
	}

	// Store session in cookie
	sessionData, _ := json.Marshal(session)
	cookieHelper.SetWebAuthnLoginSession(c, base64.StdEncoding.EncodeToString(sessionData))

	c.JSON(http.StatusOK, gin.H{
		"options":  options,
		"username": user.Username,
	})
}

// WebAuthnFinishLoginRequest - POST /api/v2/auth/webauthn/login/finish
type WebAuthnFinishLoginRequest struct {
	Response protocol.CredentialAssertionResponse `json:"response"`
}

// WebAuthnFinishLogin completes WebAuthn login
func WebAuthnFinishLogin(c *gin.Context) {
	var req WebAuthnFinishLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	// Get session from cookie
	sessionCookie, err := c.Cookie("webauthn_login_session")
	if err != nil {
		c.JSON(http.StatusBadRequest, apiError("login session not found"))
		return
	}

	sessionData, err := base64.StdEncoding.DecodeString(sessionCookie)
	if err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid login session"))
		return
	}

	var session webauthn.SessionData
	if err := json.Unmarshal(sessionData, &session); err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid login session data"))
		return
	}

	// Find user by ID stored in session
	userID := session.UserID
	if len(userID) == 0 {
		c.JSON(http.StatusBadRequest, apiError("invalid session"))
		return
	}

	// Parse user ID from string
	var uid uint
	if _, err := fmt.Sscanf(string(userID), "%d", &uid); err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid user ID"))
		return
	}

	// Get user
	var user models.AuthUser
	if err := db.DB.First(&user, uid).Error; err != nil {
		c.JSON(http.StatusUnauthorized, apiError("user not found"))
		return
	}

	// Load credentials
	creds, err := loadWebAuthnCredentials(uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, apiError("failed to load credentials"))
		return
	}

	u := &webauthnUser{
		userID:   uid,
		username: user.Username,
		creds:    creds,
	}

	// Parse the response bytes
	responseJSON, _ := json.Marshal(req.Response)
	parsedResponse, err := protocol.ParseCredentialRequestResponseBytes(responseJSON)
	if err != nil {
		c.JSON(http.StatusBadRequest, apiError(fmt.Sprintf("failed to parse response: %v", err)))
		return
	}

	// Validate the response
	cred, err := WebAuthnConfig.ValidateLogin(u, session, parsedResponse)
	if err != nil {
		c.JSON(http.StatusUnauthorized, apiError(fmt.Sprintf("login failed: %v", err)))
		return
	}

	// Update counter
	if err := db.DB.Model(&models.WebAuthnCredential{}).
		Where("cred_id = ?", base64.URLEncoding.EncodeToString(cred.ID)).
		Update("counter", int64(cred.Authenticator.SignCount)).Error; err != nil {
		// Log error but don't fail login
	}

	// Generate tokens (WebAuthn login doesn't have remember_me, use default 7 days)
	accessToken, refreshToken, _, err := auth.GenerateTokens(user.ID, user.Username, user.IsSuperuser, "", false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, apiError("failed to generate tokens"))
		return
	}

	cookieHelper := cookie.Helper()

	// Clear session cookie
	cookieHelper.ClearWebAuthnLoginSession(c)

	// Set httpOnly cookies for tokens
	cookieHelper.SetAuthTokens(c, accessToken, refreshToken, cookie.RefreshTokenDuration)

	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
			"is_admin": user.IsSuperuser,
		},
	})
}

// WebAuthnCredentialsList returns list of user's WebAuthn credentials
func WebAuthnCredentialsList(c *gin.Context) {
	userID := c.GetUint("userID")

	var creds []models.WebAuthnCredential
	if err := db.DB.Where("user_id = ?", userID).Find(&creds).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError("database error"))
		return
	}

	result := make([]gin.H, len(creds))
	for i, cred := range creds {
		result[i] = gin.H{
			"id":         cred.ID,
			"name":       cred.Name,
			"cred_id":    cred.CredID,
			"counter":    cred.Counter,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"credentials": result,
	})
}

// WebAuthnCredentialUpdateRequest - PATCH /api/v2/auth/webauthn/credentials/:id
type WebAuthnCredentialUpdateRequest struct {
	Name string `json:"name" binding:"required"`
}

// WebAuthnCredentialUpdate updates a WebAuthn credential
func WebAuthnCredentialUpdate(c *gin.Context) {
	userID := c.GetUint("userID")
	credID := c.Param("id")

	var req WebAuthnCredentialUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	// Find and update credential
	result := db.DB.Model(&models.WebAuthnCredential{}).
		Where("id = ? AND user_id = ?", credID, userID).
		Update("name", req.Name)

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, apiError("credential not found"))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "credential updated successfully",
	})
}

// WebAuthnCredentialDelete deletes a WebAuthn credential
func WebAuthnCredentialDelete(c *gin.Context) {
	userID := c.GetUint("userID")
	credID := c.Param("id")

	var req struct {
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	// Verify password
	var user models.AuthUser
	if err := db.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, apiError("user not found"))
		return
	}

	match, err := auth.CheckPassword(req.Password, user.Password)
	if err != nil || !match {
		c.JSON(http.StatusUnauthorized, apiError("invalid password"))
		return
	}

	err = db.DB.Transaction(func(tx *gorm.DB) error {
		// Delete credential
		result := tx.Where("id = ? AND user_id = ?", credID, userID).Delete(&models.WebAuthnCredential{})
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}

		// Check if this was the last credential
		var count int64
		if err := tx.Model(&models.WebAuthnCredential{}).Where("user_id = ?", userID).Count(&count).Error; err != nil {
			return err
		}

		// If no more credentials, disable WebAuthn
		if count == 0 {
			if err := tx.Model(&models.Profile{}).Where("user_id = ?", userID).Update("is_webauthn_enabled", false).Error; err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, apiError("credential not found"))
		} else {
			c.JSON(http.StatusInternalServerError, apiError("failed to delete credential"))
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "credential deleted successfully",
	})
}

// WebAuthnStatus returns WebAuthn status for current user
func WebAuthnStatus(c *gin.Context) {
	userID := c.GetUint("userID")

	var profile models.Profile
	if err := db.DB.Where("user_id = ?", userID).First(&profile).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError("database error"))
		return
	}

	var credCount int64
	db.DB.Model(&models.WebAuthnCredential{}).Where("user_id = ?", userID).Count(&credCount)

	c.JSON(http.StatusOK, gin.H{
		"enabled":           profile.IsWebauthnEnabled,
		"credentials_count": credCount,
	})
}
