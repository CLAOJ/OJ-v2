package v2

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	authHandlers "github.com/CLAOJ/claoj/api/v2/auth"
	"github.com/CLAOJ/claoj/config"
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
	"github.com/pquerna/otp/totp"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// enableTotpForUser creates a confirmed TOTP device for the user and returns the
// plaintext base32 secret so tests can generate valid codes.
func enableTotpForUser(t *testing.T, database *gorm.DB, user models.AuthUser) string {
	t.Helper()

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      config.C.Email.FromName,
		AccountName: user.Username,
	})
	if err != nil {
		t.Fatalf("Failed to generate TOTP secret: %v", err)
	}

	encSecret, err := encryptSecret(key.Secret())
	if err != nil {
		t.Fatalf("Failed to encrypt TOTP secret: %v", err)
	}

	device := models.TotpDevice{
		UserID:    user.ID,
		Secret:    encSecret,
		Confirmed: true,
		CreatedAt: time.Now(),
	}
	if err := database.Create(&device).Error; err != nil {
		t.Fatalf("Failed to create TOTP device: %v", err)
	}

	return key.Secret()
}

// assertAuthCookiesSet verifies both httpOnly auth cookies were planted on the
// response. The v2 frontend authenticates solely via these cookies, so any
// login-completion path that omits them leaves the user effectively logged out.
func assertAuthCookiesSet(t *testing.T, w *httptest.ResponseRecorder) {
	t.Helper()

	var hasAccessToken, hasRefreshToken bool
	for _, c := range w.Result().Cookies() {
		if c.Name == "access_token" {
			hasAccessToken = true
			assert.True(t, c.HttpOnly, "access_token cookie must be HttpOnly")
		}
		if c.Name == "refresh_token" {
			hasRefreshToken = true
			assert.True(t, c.HttpOnly, "refresh_token cookie must be HttpOnly")
		}
	}
	assert.True(t, hasAccessToken, "access_token cookie not set")
	assert.True(t, hasRefreshToken, "refresh_token cookie not set")
}

// TestIssueAuthSession_SetsCookiesAndStoresRefreshToken locks the shared
// login-finalization contract used by every second-factor login path
// (TotpVerify, TotpBackupVerify, and WebAuthnFinishLogin): a finalised session
// MUST plant the httpOnly auth cookies AND persist the refresh token so
// /auth/refresh rotation keeps working after the access token expires.
// WebAuthnFinishLogin can't be exercised end-to-end without a WebAuthn signing
// harness, so this guards the helper it delegates to.
func TestIssueAuthSession_SetsCookiesAndStoresRefreshToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	database := setupLoginTestDB(t)
	db.DB = database

	user := createTestUser(t, database, "sessionuser", "password123", true)

	router := gin.New()
	router.POST("/issue", func(c *gin.Context) {
		access, refresh, err := issueAuthSession(c, user)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"access_token": access, "refresh_token": refresh})
	})

	req := httptest.NewRequest(http.MethodPost, "/issue", nil)
	req.Header.Set("User-Agent", "ContractTest/1.0")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	refreshToken, _ := response["refresh_token"].(string)
	assert.NotEmpty(t, refreshToken)

	assertAuthCookiesSet(t, w)

	entry, found, err := authHandlers.RefreshStore.Get(refreshToken)
	assert.NoError(t, err)
	assert.True(t, found, "refresh token not saved in RefreshStore")
	assert.NotEmpty(t, entry.FamilyID)
	assert.Equal(t, user.ID, entry.UserID)
	assert.Contains(t, entry.UserAgent, "ContractTest")
}

func TestTotpVerify_SetsAuthCookiesAndStoresRefreshToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	database := setupLoginTestDB(t)
	db.DB = database

	user := createTestUser(t, database, "totpuser", "password123", true)
	secret := enableTotpForUser(t, database, user)

	code, err := totp.GenerateCode(secret, time.Now())
	assert.NoError(t, err)

	router := gin.New()
	router.POST("/auth/totp/verify", TotpVerify)

	body := map[string]interface{}{"username": user.Username, "code": code}
	jsonBody, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/auth/totp/verify", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	refreshToken, _ := response["refresh_token"].(string)
	assert.NotEmpty(t, refreshToken)

	// The auth cookies must be set so the browser is actually authenticated.
	assertAuthCookiesSet(t, w)

	// The refresh token must be persisted so /auth/refresh rotation works.
	entry, found, err := authHandlers.RefreshStore.Get(refreshToken)
	assert.NoError(t, err)
	assert.True(t, found, "refresh token not saved in RefreshStore after TOTP verify")
	assert.NotEmpty(t, entry.FamilyID)
	assert.Equal(t, user.ID, entry.UserID)
}

func TestTotpBackupVerify_SetsAuthCookiesAndStoresRefreshToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	database := setupLoginTestDB(t)
	assert.NoError(t, database.AutoMigrate(&models.BackupCode{}))
	db.DB = database

	user := createTestUser(t, database, "backupuser", "password123", true)
	enableTotpForUser(t, database, user)

	// Store a known backup code (hashed the same way the handler expects).
	plainCode := "1234-5678"
	hash := sha256.Sum256([]byte(plainCode))
	backupCode := models.BackupCode{
		UserID:    user.ID,
		Code:      hex.EncodeToString(hash[:]),
		Used:      false,
		CreatedAt: time.Now(),
	}
	assert.NoError(t, database.Create(&backupCode).Error)

	router := gin.New()
	router.POST("/auth/totp/verify-backup", TotpBackupVerify)

	body := map[string]interface{}{"username": user.Username, "code": plainCode}
	jsonBody, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/auth/totp/verify-backup", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	refreshToken, _ := response["refresh_token"].(string)
	assert.NotEmpty(t, refreshToken)

	assertAuthCookiesSet(t, w)

	entry, found, err := authHandlers.RefreshStore.Get(refreshToken)
	assert.NoError(t, err)
	assert.True(t, found, "refresh token not saved in RefreshStore after backup-code verify")
	assert.NotEmpty(t, entry.FamilyID)
	assert.Equal(t, user.ID, entry.UserID)
}
