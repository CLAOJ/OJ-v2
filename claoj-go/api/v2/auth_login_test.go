package v2

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	authHandlers "github.com/CLAOJ/claoj-go/api/v2/auth"
	"github.com/CLAOJ/claoj-go/auth"
	"github.com/CLAOJ/claoj-go/config"
	"github.com/CLAOJ/claoj-go/db"
	"github.com/CLAOJ/claoj-go/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func init() {
	// Initialize test configuration
	config.C.App.JwtSecretKey = "test-secret-key-for-jwt-tokens-generation-minimum-32-characters"
	config.C.App.SecretKey = "test-secret-key-for-encryption-32-characters"
	config.C.App.SiteFullURL = "http://localhost:3000"
	config.C.Email.FromName = "CLAOJ Test"
}

func setupLoginTestDB(t *testing.T) *gorm.DB {
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Migrate the schema
	database.AutoMigrate(
		&models.AuthUser{},
		&models.Profile{},
		&models.RefreshToken{},
		&models.TotpDevice{},
		&models.EmailVerificationToken{},
	)

	return database
}

func createTestUser(t *testing.T, database *gorm.DB, username, password string, isActive bool) models.AuthUser {
	// Hash password using Django-compatible hasher
	hashedPassword, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	user := models.AuthUser{
		Username:    username,
		Email:       username + "@example.com",
		Password:    hashedPassword,
		IsActive:    true, // Set to true first to work around GORM default value issue
		IsStaff:     false,
		IsSuperuser: false,
		DateJoined:  time.Now(),
	}

	if err := database.Create(&user).Error; err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Update IsActive if it should be false (GORM ignores bool zero values with defaults)
	if !isActive {
		database.Model(&user).Update("is_active", false)
		// Refresh the user object
		database.First(&user, user.ID)
	}

	// Create profile
	profile := models.Profile{
		UserID:      user.ID,
		Timezone:    "UTC",
		LanguageID:  1,
		LastAccess:  time.Now(),
		DisplayRank: "user",
		AceTheme:    "auto",
		SiteTheme:   "auto",
		MathEngine:  "TeX",
	}
	database.Create(&profile)

	return user
}

func TestLogin_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	database := setupLoginTestDB(t)
	db.DB = database

	// Create active user
	createTestUser(t, database, "testuser", "password123", true)

	router := gin.New()
	router.POST("/auth/login", authHandlers.Login)

	body := map[string]interface{}{
		"username": "testuser",
		"password": "password123",
	}
	jsonBody, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.NotEmpty(t, response["access_token"])
	assert.NotEmpty(t, response["refresh_token"])
	assert.NotNil(t, response["user"])

	// Check cookies were set
	cookies := w.Result().Cookies()
	var hasAccessToken, hasRefreshToken bool
	for _, cookie := range cookies {
		if cookie.Name == "access_token" {
			hasAccessToken = true
			assert.True(t, cookie.HttpOnly)
			// Access token should be ~15 minutes
			assert.True(t, cookie.MaxAge == 900 || cookie.MaxAge == 0) // 0 means session cookie
		}
		if cookie.Name == "refresh_token" {
			hasRefreshToken = true
			assert.True(t, cookie.HttpOnly)
			// Default refresh token is 7 days = 604800 seconds
			assert.True(t, cookie.MaxAge == 7*24*60*60 || cookie.MaxAge == 0)
		}
	}
	assert.True(t, hasAccessToken, "access_token cookie not set")
	assert.True(t, hasRefreshToken, "refresh_token cookie not set")
}

func TestLogin_WithRememberMe(t *testing.T) {
	gin.SetMode(gin.TestMode)

	database := setupLoginTestDB(t)
	db.DB = database

	// Create active user
	createTestUser(t, database, "testuser", "password123", true)

	router := gin.New()
	router.POST("/auth/login", authHandlers.Login)

	body := map[string]interface{}{
		"username":     "testuser",
		"password":     "password123",
		"remember_me":  true,
	}
	jsonBody, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Check refresh token cookie has extended expiry (30 days)
	cookies := w.Result().Cookies()
	var refreshTokenCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == "refresh_token" {
			refreshTokenCookie = cookie
			break
		}
	}

	assert.NotNil(t, refreshTokenCookie, "refresh_token cookie not set")
	// With remember_me, cookie should be 30 days = 2592000 seconds
	expectedMaxAge := 30 * 24 * 60 * 60
	assert.Equal(t, expectedMaxAge, refreshTokenCookie.MaxAge, "Refresh token cookie should have 30-day expiry with remember_me")
}

func TestLogin_WithoutRememberMe(t *testing.T) {
	gin.SetMode(gin.TestMode)

	database := setupLoginTestDB(t)
	db.DB = database

	// Create active user
	createTestUser(t, database, "testuser", "password123", true)

	router := gin.New()
	router.POST("/auth/login", authHandlers.Login)

	body := map[string]interface{}{
		"username":     "testuser",
		"password":     "password123",
		"remember_me":  false,
	}
	jsonBody, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Check refresh token cookie has standard expiry (7 days)
	cookies := w.Result().Cookies()
	var refreshTokenCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == "refresh_token" {
			refreshTokenCookie = cookie
			break
		}
	}

	assert.NotNil(t, refreshTokenCookie, "refresh_token cookie not set")
	// Without remember_me, cookie should be 7 days = 604800 seconds
	expectedMaxAge := 7 * 24 * 60 * 60
	assert.Equal(t, expectedMaxAge, refreshTokenCookie.MaxAge, "Refresh token cookie should have 7-day expiry without remember_me")
}

func TestLogin_InvalidCredentials(t *testing.T) {
	gin.SetMode(gin.TestMode)

	database := setupLoginTestDB(t)
	db.DB = database

	// Create active user
	createTestUser(t, database, "testuser", "password123", true)

	router := gin.New()
	router.POST("/auth/login", authHandlers.Login)

	tests := []struct {
		name     string
		username string
		password string
		wantMsg  string
	}{
		{
			name:     "wrong password",
			username: "testuser",
			password: "wrongpassword",
			wantMsg:  "invalid username or password",
		},
		{
			name:     "non-existent user",
			username: "nonexistent",
			password: "password123",
			wantMsg:  "invalid username or password",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := map[string]interface{}{
				"username": tt.username,
				"password": tt.password,
			}
			jsonBody, _ := json.Marshal(body)
			req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusUnauthorized, w.Code)

			var response map[string]string
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Equal(t, tt.wantMsg, response["error"])
		})
	}
}

func TestLogin_InactiveUser(t *testing.T) {
	gin.SetMode(gin.TestMode)

	database := setupLoginTestDB(t)
	db.DB = database

	// Create inactive user (email not verified)
	user := createTestUser(t, database, "inactiveuser", "password123", false)

	router := gin.New()
	router.POST("/auth/login", authHandlers.Login)

	body := map[string]interface{}{
		"username": "inactiveuser",
		"password": "password123",
	}
	jsonBody, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "email not verified", response["error"])
	assert.True(t, response["requires_email_verification"].(bool))

	// Create a verification token for this user
	token := models.EmailVerificationToken{
		UserID:    user.ID,
		Token:     "test-token",
		Email:     user.Email,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	database.Create(&token)

	// Try login again - should still get verification required
	req2 := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer(jsonBody))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()

	router.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusForbidden, w2.Code)
}

func TestLogin_MissingFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	database := setupLoginTestDB(t)
	db.DB = database

	router := gin.New()
	router.POST("/auth/login", authHandlers.Login)

	tests := []struct {
		name string
		body map[string]interface{}
	}{
		{
			name: "missing username",
			body: map[string]interface{}{"password": "password123"},
		},
		{
			name: "missing password",
			body: map[string]interface{}{"username": "testuser"},
		},
		{
			name: "empty body",
			body: map[string]interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonBody, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code)
		})
	}
}

func TestLogin_StoresRefreshToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	database := setupLoginTestDB(t)
	db.DB = database

	// Create active user
	user := createTestUser(t, database, "testuser", "password123", true)

	router := gin.New()
	router.POST("/auth/login", authHandlers.Login)

	body := map[string]interface{}{
		"username":    "testuser",
		"password":    "password123",
		"remember_me": true,
	}
	jsonBody, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Check that refresh token was stored in database
	var refreshToken models.RefreshToken
	err := database.Where("user_id = ?", user.ID).First(&refreshToken).Error
	assert.NoError(t, err)
	assert.NotEmpty(t, refreshToken.Token)
	assert.NotEmpty(t, refreshToken.FamilyID)

	// With remember_me, expiry should be ~30 days from now
	expectedExpiryMin := time.Now().Add(29 * 24 * time.Hour)
	expectedExpiryMax := time.Now().Add(31 * 24 * time.Hour)
	assert.True(t, refreshToken.ExpiresAt.After(expectedExpiryMin) && refreshToken.ExpiresAt.Before(expectedExpiryMax),
		"Refresh token expiry should be ~30 days from now")
}

func TestLogin_TOTPRequired(t *testing.T) {
	gin.SetMode(gin.TestMode)

	database := setupLoginTestDB(t)
	db.DB = database

	// Create active user
	user := createTestUser(t, database, "testuser", "password123", true)

	// Enable TOTP for user
	totpDevice := models.TotpDevice{
		UserID:    user.ID,
		Secret:    "encrypted-secret",
		Confirmed: true,
		CreatedAt: time.Now(),
	}
	database.Create(&totpDevice)

	router := gin.New()
	router.POST("/auth/login", authHandlers.Login)

	body := map[string]interface{}{
		"username": "testuser",
		"password": "password123",
	}
	jsonBody, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response["requires_totp"].(bool))
	assert.Equal(t, "testuser", response["username"])

	// No tokens should be issued yet
	assert.Empty(t, response["access_token"])
	assert.Empty(t, response["refresh_token"])
}

func TestLogin_UserAgentAndIP(t *testing.T) {
	gin.SetMode(gin.TestMode)

	database := setupLoginTestDB(t)
	db.DB = database

	// Create active user
	user := createTestUser(t, database, "testuser", "password123", true)

	router := gin.New()
	router.POST("/auth/login", authHandlers.Login)

	body := map[string]interface{}{
		"username": "testuser",
		"password": "password123",
	}
	jsonBody, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "TestBrowser/1.0")
	// Note: httptest doesn't set RemoteAddr in a way that's easily testable
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Check that user agent was stored
	var refreshToken models.RefreshToken
	err := database.Where("user_id = ?", user.ID).First(&refreshToken).Error
	assert.NoError(t, err)
	assert.NotNil(t, refreshToken.UserAgent)
	assert.True(t, strings.Contains(*refreshToken.UserAgent, "TestBrowser"))
}
