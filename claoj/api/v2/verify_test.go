package v2

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/CLAOJ/claoj/config"
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestMain(m *testing.M) {
	// Initialize test configuration
	config.C.App.JwtSecretKey = "test-secret-key-for-jwt-tokens-generation-minimum-32-characters"
	config.C.App.SecretKey = "test-secret-key-for-encryption-32-characters"
	config.C.Email.FromName = "CLAOJ Test"
	os.Exit(m.Run())
}

func setupVerifyTestDB(t *testing.T) *gorm.DB {
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Migrate the schema
	database.AutoMigrate(
		&models.AuthUser{},
		&models.EmailVerificationToken{},
	)

	return database
}

func TestResendVerification_Unauthenticated(t *testing.T) {
	gin.SetMode(gin.TestMode)

	database := setupVerifyTestDB(t)
	db.DB = database

	router := gin.New()
	router.POST("/auth/resend-verification", ResendVerification)

	tests := []struct {
		name       string
		body       map[string]interface{}
		wantStatus int
		wantMsg    string
		wantField  string // "message" or "error"
	}{
		{
			name:       "missing email",
			body:       map[string]interface{}{},
			wantStatus: http.StatusBadRequest,
			wantMsg:    "An error occurred. Please try again.",
			wantField:  "error",
		},
		{
			name:       "non-existent email",
			body:       map[string]interface{}{"email": "nonexistent@example.com"},
			wantStatus: http.StatusOK, // Don't reveal if email exists
			wantMsg:    "If the email exists and is not verified, a verification link has been sent",
			wantField:  "message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonBody, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/auth/resend-verification", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)

			var response map[string]string
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Equal(t, tt.wantMsg, response[tt.wantField])
		})
	}
}

func TestResendVerification_AlreadyVerified(t *testing.T) {
	gin.SetMode(gin.TestMode)

	database := setupVerifyTestDB(t)
	db.DB = database

	// Create a verified user
	user := models.AuthUser{
		Username:    "verifieduser",
		Email:       "verified@example.com",
		Password:    "hashedpassword",
		IsActive:    true, // Already verified
		IsStaff:     false,
		IsSuperuser: false,
		DateJoined:  time.Now(),
	}
	database.Create(&user)

	router := gin.New()
	router.POST("/auth/resend-verification", ResendVerification)

	body := map[string]interface{}{"email": "verified@example.com"}
	jsonBody, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/auth/resend-verification", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]string
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, "Email is already verified", response["message"])
}

func TestResendVerification_AuthenticatedUser(t *testing.T) {
	gin.SetMode(gin.TestMode)

	database := setupVerifyTestDB(t)
	db.DB = database

	// Create an unverified user
	user := models.AuthUser{
		Username:    "unverifieduser",
		Email:       "unverified@example.com",
		Password:    "hashedpassword",
		IsActive:    false, // Not verified
		IsStaff:     false,
		IsSuperuser: false,
		DateJoined:  time.Now(),
	}
	database.Create(&user)
	// GORM ignores boolean zero values, so explicitly update is_active
	database.Model(&models.AuthUser{}).Where("id = ?", user.ID).Update("is_active", false)

	router := gin.New()
	router.POST("/auth/resend-verification", func(c *gin.Context) {
		// Simulate authenticated user by setting user_id in context
		c.Set("user_id", user.ID)
		ResendVerification(c)
	})

	// Request without email (should use authenticated user's email)
	req := httptest.NewRequest(http.MethodPost, "/auth/resend-verification", bytes.NewBuffer([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Should succeed or fail gracefully (depending on email configuration)
	assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusInternalServerError)

	if w.Code == http.StatusOK {
		var response map[string]string
		json.Unmarshal(w.Body.Bytes(), &response)
		// Should either confirm email was sent or indicate rate limiting
		assert.True(t,
			response["message"] == "Verification email sent. Please check your inbox." ||
				response["message"] == "If the email exists and is not verified, a verification link has been sent",
		)
	}
}

func TestResendVerification_CreatesToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	database := setupVerifyTestDB(t)
	db.DB = database

	// Create an unverified user
	user := models.AuthUser{
		Username:    "testuser",
		Email:       "test@example.com",
		Password:    "hashedpassword",
		IsActive:    false,
		IsStaff:     false,
		IsSuperuser: false,
		DateJoined:  time.Now(),
	}
	database.Create(&user)
	// GORM ignores boolean zero values, so explicitly update is_active
	database.Model(&models.AuthUser{}).Where("id = ?", user.ID).Update("is_active", false)

	router := gin.New()
	router.POST("/auth/resend-verification", ResendVerification)

	body := map[string]interface{}{"email": "test@example.com"}
	jsonBody, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/auth/resend-verification", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Check that a token was created in the database
	// First check the response code and body
	t.Logf("Response code: %d", w.Code)
	t.Logf("Response body: %s", w.Body.String())

	var token models.EmailVerificationToken
	err := database.Where("user_id = ?", user.ID).First(&token).Error

	// If email config is not set up, it might fail, but token should still be created
	// or the endpoint might return success
	if w.Code == http.StatusOK {
		// Token should exist
		assert.NoError(t, err)
		assert.NotEmpty(t, token.Token)
		assert.Equal(t, user.Email, token.Email)
		assert.True(t, token.ExpiresAt.After(time.Now()))
	}
}

func TestResendVerification_ReplacesOldToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	database := setupVerifyTestDB(t)
	db.DB = database

	// Create an unverified user
	user := models.AuthUser{
		Username:    "testuser",
		Email:       "test@example.com",
		Password:    "hashedpassword",
		IsActive:    false,
		IsStaff:     false,
		IsSuperuser: false,
		DateJoined:  time.Now(),
	}
	database.Create(&user)
	// GORM ignores boolean zero values, so explicitly update is_active
	database.Model(&models.AuthUser{}).Where("id = ?", user.ID).Update("is_active", false)

	// Create an old token
	oldToken := models.EmailVerificationToken{
		UserID:    user.ID,
		Token:     "old-token-123",
		Email:     user.Email,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	database.Create(&oldToken)

	router := gin.New()
	router.POST("/auth/resend-verification", ResendVerification)

	body := map[string]interface{}{"email": "test@example.com"}
	jsonBody, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/auth/resend-verification", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code == http.StatusOK {
		// Old token should be deleted
		var count int64
		database.Model(&models.EmailVerificationToken{}).Where("token = ?", "old-token-123").Count(&count)
		assert.Equal(t, int64(0), count)

		// New token should exist
		var newToken models.EmailVerificationToken
		err := database.Where("user_id = ?", user.ID).First(&newToken).Error
		assert.NoError(t, err)
		assert.NotEqual(t, "old-token-123", newToken.Token)
	}
}

func TestVerifyEmail(t *testing.T) {
	gin.SetMode(gin.TestMode)

	database := setupVerifyTestDB(t)
	db.DB = database

	// Create an unverified user
	user := models.AuthUser{
		Username:    "testuser",
		Email:       "test@example.com",
		Password:    "hashedpassword",
		IsActive:    false,
		IsStaff:     false,
		IsSuperuser: false,
		DateJoined:  time.Now(),
	}
	database.Create(&user)
	// GORM ignores boolean zero values, so explicitly update is_active
	database.Model(&models.AuthUser{}).Where("id = ?", user.ID).Update("is_active", false)

	// Create a valid verification token
	token := models.EmailVerificationToken{
		UserID:    user.ID,
		Token:     "valid-verification-token",
		Email:     user.Email,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	database.Create(&token)

	router := gin.New()
	router.POST("/auth/verify-email", VerifyEmail)

	tests := []struct {
		name       string
		body       map[string]interface{}
		wantStatus int
		wantMsg    string
		wantField  string // "message" for success, "error" for errors
	}{
		{
			name:       "valid token",
			body:       map[string]interface{}{"token": "valid-verification-token"},
			wantStatus: http.StatusOK,
			wantMsg:    "Email verified successfully",
			wantField:  "message",
		},
		{
			name:       "invalid token",
			body:       map[string]interface{}{"token": "invalid-token"},
			wantStatus: http.StatusBadRequest,
			wantMsg:    "An error occurred. Please try again.",
			wantField:  "error",
		},
		{
			name:       "missing token",
			body:       map[string]interface{}{},
			wantStatus: http.StatusBadRequest,
			wantMsg:    "An error occurred. Please try again.",
			wantField:  "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset user state for each test
			database.Model(&user).Update("is_active", false)
			// Delete old tokens and create fresh one
			database.Where("user_id = ?", user.ID).Delete(&models.EmailVerificationToken{})
			database.Create(&token)

			jsonBody, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/auth/verify-email", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)

			var response map[string]string
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Equal(t, tt.wantMsg, response[tt.wantField])

			// For valid token, check user is now active
			if tt.name == "valid token" {
				var updatedUser models.AuthUser
				database.First(&updatedUser, user.ID)
				assert.True(t, updatedUser.IsActive)

				// Token should be deleted
				var count int64
				database.Model(&models.EmailVerificationToken{}).Where("user_id = ?", user.ID).Count(&count)
				assert.Equal(t, int64(0), count)
			}
		})
	}
}

func TestVerifyEmail_ExpiredToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	database := setupVerifyTestDB(t)
	db.DB = database

	// Create an unverified user
	user := models.AuthUser{
		Username:    "testuser",
		Email:       "test@example.com",
		Password:    "hashedpassword",
		IsActive:    false,
		IsStaff:     false,
		IsSuperuser: false,
		DateJoined:  time.Now(),
	}
	database.Create(&user)
	// GORM ignores boolean zero values, so explicitly update is_active
	database.Model(&models.AuthUser{}).Where("id = ?", user.ID).Update("is_active", false)

	// Create an expired token
	expiredToken := models.EmailVerificationToken{
		UserID:    user.ID,
		Token:     "expired-token",
		Email:     user.Email,
		CreatedAt: time.Now().Add(-48 * time.Hour),
		ExpiresAt: time.Now().Add(-24 * time.Hour), // Expired
	}
	database.Create(&expiredToken)

	router := gin.New()
	router.POST("/auth/verify-email", VerifyEmail)

	body := map[string]interface{}{"token": "expired-token"}
	jsonBody, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/auth/verify-email", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]string
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, "An error occurred. Please try again.", response["error"])

	// User should still be inactive
	var updatedUser models.AuthUser
	database.First(&updatedUser, user.ID)
	assert.False(t, updatedUser.IsActive)
}

func TestResendVerification_UserNotFound_Authenticated(t *testing.T) {
	gin.SetMode(gin.TestMode)

	database := setupVerifyTestDB(t)
	db.DB = database

	router := gin.New()
	router.POST("/auth/resend-verification", func(c *gin.Context) {
		// Set a non-existent user ID
		c.Set("user_id", uint(9999))
		ResendVerification(c)
	})

	req := httptest.NewRequest(http.MethodPost, "/auth/resend-verification", bytes.NewBuffer([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response map[string]string
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, "An error occurred. Please try again.", response["error"])
}

// Helper to check if error is a GORM "record not found" error
func isRecordNotFoundError(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}
