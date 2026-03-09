// Package integration_test provides integration tests for registration flow.
package integration_test

import (
	"net/http"
	"testing"
	"time"

	v2 "github.com/CLAOJ/claoj/api/v2"
	authHandlers "github.com/CLAOJ/claoj/api/v2/auth"
	"github.com/CLAOJ/claoj/integration"
	"github.com/CLAOJ/claoj/models"
	"github.com/stretchr/testify/assert"
)

// TestRegistrationFlow_FullCycle_Success tests successful user registration
func TestRegistrationFlow_FullCycle_Success(t *testing.T) {
	testDB := integration.SetupIntegrationDB(t)
	defer func() {
		// Give async email verification goroutine time to complete
		time.Sleep(200 * time.Millisecond)
		integration.CleanupDB(t, testDB)
	}()

	// Setup router
	gin := integration.TestRouter()
	gin.POST("/auth/register", v2.Register)

	// Register new user
	resp := integration.MakeRequest(t, gin, integration.HTTPRequest{
		Method: "POST",
		Path:   "/auth/register",
		Body: map[string]interface{}{
			"username":  "newuser",
			"email":     "newuser@example.com",
			"password":  "Password123!",
			"full_name": "New User",
			"language":  "go",
			"timezone":  "UTC",
		},
	})

	assert.Equal(t, http.StatusCreated, resp.Code)
	assert.Equal(t, "account created successfully. Please check your email to verify your account.", resp.JSONBody["message"])
	assert.True(t, resp.JSONBody["requires_verification"].(bool))

	// Wait for async email verification token creation
	time.Sleep(200 * time.Millisecond)

	// Verify user was created in database
	var user models.AuthUser
	err := testDB.DB.Where("username = ?", "newuser").First(&user).Error
	assert.NoError(t, err)
	assert.Equal(t, "newuser@example.com", user.Email)
	assert.False(t, user.IsActive) // Should be inactive until verified

	// Verify verification token was created
	var token models.EmailVerificationToken
	err = testDB.DB.Where("user_id = ?", user.ID).First(&token).Error
	assert.NoError(t, err)
	assert.NotEmpty(t, token.Token)
}

// TestRegistrationFlow_DuplicateUsername tests registration with existing username
func TestRegistrationFlow_DuplicateUsername(t *testing.T) {
	testDB := integration.SetupIntegrationDB(t)
	defer integration.CleanupDB(t, testDB)

	// Create existing user
	integration.CreateTestUser(testDB.DB, "existinguser", "Password123!", true)

	// Setup router
	gin := integration.TestRouter()
	gin.POST("/auth/register", v2.Register)

	// Try to register with same username
	resp := integration.MakeRequest(t, gin, integration.HTTPRequest{
		Method: "POST",
		Path:   "/auth/register",
		Body: map[string]interface{}{
			"username": "existinguser",
			"email":    "different@example.com",
			"password": "Password123!",
		},
	})

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	// Note: apiError() returns generic message for security
	assert.Equal(t, "An error occurred. Please try again.", resp.JSONBody["error"])
}

// TestRegistrationFlow_DuplicateEmail tests registration with existing email
func TestRegistrationFlow_DuplicateEmail(t *testing.T) {
	testDB := integration.SetupIntegrationDB(t)
	defer integration.CleanupDB(t, testDB)

	// Create existing user
	integration.CreateTestUser(testDB.DB, "user1", "Password123!", true)

	// Setup router
	gin := integration.TestRouter()
	gin.POST("/auth/register", v2.Register)

	// Try to register with same email
	resp := integration.MakeRequest(t, gin, integration.HTTPRequest{
		Method: "POST",
		Path:   "/auth/register",
		Body: map[string]interface{}{
			"username": "user2",
			"email":    "user1@example.com",
			"password": "Password123!",
		},
	})

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	// Note: apiError() returns generic message for security
	assert.Equal(t, "An error occurred. Please try again.", resp.JSONBody["error"])
}

// TestRegistrationFlow_InvalidUsername tests registration with invalid username formats
func TestRegistrationFlow_InvalidUsername(t *testing.T) {
	testDB := integration.SetupIntegrationDB(t)
	defer integration.CleanupDB(t, testDB)

	// Setup router
	gin := integration.TestRouter()
	gin.POST("/auth/register", v2.Register)

	tests := []struct {
		name     string
		username string
		wantCode int
	}{
		{
			name:     "username with spaces",
			username: "user name",
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "username too long",
			username: "verylongusernamethatexceedsthirtycharacters",
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "username with special chars",
			username: "user@name",
			wantCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := integration.MakeRequest(t, gin, integration.HTTPRequest{
				Method: "POST",
				Path:   "/auth/register",
				Body: map[string]interface{}{
					"username": tt.username,
					"email":    "test@example.com",
					"password": "Password123!",
				},
			})

			assert.Equal(t, tt.wantCode, resp.Code)
		})
	}
}

// TestRegistrationFlow_WeakPassword tests registration with weak passwords
func TestRegistrationFlow_WeakPassword(t *testing.T) {
	testDB := integration.SetupIntegrationDB(t)
	defer integration.CleanupDB(t, testDB)

	// Setup router
	gin := integration.TestRouter()
	gin.POST("/auth/register", v2.Register)

	tests := []struct {
		name     string
		password string
		wantErr  string
	}{
		{
			name:     "password too short",
			password: "Pass1",
			wantErr:  "password must be at least 8 characters long",
		},
		{
			name:     "password missing uppercase",
			password: "password123",
			wantErr:  "password must contain at least one uppercase letter",
		},
		{
			name:     "password missing lowercase",
			password: "PASSWORD123",
			wantErr:  "password must contain at least one lowercase letter",
		},
		{
			name:     "password missing number",
			password: "Password",
			wantErr:  "password must contain at least one number",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := integration.MakeRequest(t, gin, integration.HTTPRequest{
				Method: "POST",
				Path:   "/auth/register",
				Body: map[string]interface{}{
					"username": "testuser",
					"email":    "test@example.com",
					"password": tt.password,
				},
			})

			assert.Equal(t, http.StatusBadRequest, resp.Code)
			// Note: apiError() returns generic message for security
			assert.Equal(t, "An error occurred. Please try again.", resp.JSONBody["error"])
		})
	}
}

// TestRegistrationFlow_EmailVerification tests the email verification flow
func TestRegistrationFlow_EmailVerification(t *testing.T) {
	testDB := integration.SetupIntegrationDB(t)
	defer integration.CleanupDB(t, testDB)

	// Setup router
	gin := integration.TestRouter()
	gin.POST("/auth/register", v2.Register)
	gin.POST("/auth/verify-email", v2.VerifyEmail)
	gin.POST("/auth/login", authHandlers.Login)

	// 1. Register new user
	resp := integration.MakeRequest(t, gin, integration.HTTPRequest{
		Method: "POST",
		Path:   "/auth/register",
		Body: map[string]interface{}{
			"username": "verifyuser",
			"email":    "verify@example.com",
			"password": "Password123!",
		},
	})

	assert.Equal(t, http.StatusCreated, resp.Code)

	// Wait for async email verification token creation
	time.Sleep(100 * time.Millisecond)

	// Get verification token from database
	var user models.AuthUser
	testDB.DB.Where("username = ?", "verifyuser").First(&user)
	assert.False(t, user.IsActive) // Should be inactive

	var token models.EmailVerificationToken
	testDB.DB.Where("user_id = ?", user.ID).First(&token)
	assert.NotEmpty(t, token.Token)

	// 2. Try to login before verification (should fail)
	loginResp := integration.MakeRequest(t, gin, integration.HTTPRequest{
		Method: "POST",
		Path:   "/auth/login",
		Body: map[string]interface{}{
			"username": "verifyuser",
			"password": "Password123!",
		},
	})

	assert.Equal(t, http.StatusForbidden, loginResp.Code)
	assert.Equal(t, "email not verified", loginResp.JSONBody["error"])

	// 3. Verify email with token
	verifyResp := integration.MakeRequest(t, gin, integration.HTTPRequest{
		Method: "POST",
		Path:   "/auth/verify-email",
		Body: map[string]interface{}{
			"token": token.Token,
		},
	})

	assert.Equal(t, http.StatusOK, verifyResp.Code)
	assert.Equal(t, "Email verified successfully", verifyResp.JSONBody["message"])

	// 4. Verify user is now active
	testDB.DB.First(&user, user.ID)
	assert.True(t, user.IsActive)

	// 5. Login should now succeed
	loginResp2 := integration.MakeRequest(t, gin, integration.HTTPRequest{
		Method: "POST",
		Path:   "/auth/login",
		Body: map[string]interface{}{
			"username": "verifyuser",
			"password": "Password123!",
		},
	})

	assert.Equal(t, http.StatusOK, loginResp2.Code)
	assert.NotNil(t, loginResp2.JSONBody["access_token"])
}

// TestRegistrationFlow_VerifyEmail_InvalidToken tests verification with invalid token
func TestRegistrationFlow_VerifyEmail_InvalidToken(t *testing.T) {
	testDB := integration.SetupIntegrationDB(t)
	defer integration.CleanupDB(t, testDB)

	// Setup router
	gin := integration.TestRouter()
	gin.POST("/auth/verify-email", v2.VerifyEmail)

	// Try to verify with invalid token
	resp := integration.MakeRequest(t, gin, integration.HTTPRequest{
		Method: "POST",
		Path:   "/auth/verify-email",
		Body: map[string]interface{}{
			"token": "invalid-token",
		},
	})

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	// Note: apiError() returns generic message for security
	assert.Equal(t, "An error occurred. Please try again.", resp.JSONBody["error"])
}

// TestRegistrationFlow_ResendVerification tests resending verification email
func TestRegistrationFlow_ResendVerification(t *testing.T) {
	testDB := integration.SetupIntegrationDB(t)
	defer integration.CleanupDB(t, testDB)

	// Create unverified user
	user := integration.CreateTestUser(testDB.DB, "unverified", "Password123!", false)

	// Setup router
	gin := integration.TestRouter()
	gin.POST("/auth/resend-verification", v2.ResendVerification)

	// Resend verification
	resp := integration.MakeRequest(t, gin, integration.HTTPRequest{
		Method: "POST",
		Path:   "/auth/resend-verification",
		Body: map[string]interface{}{
			"email": "unverified@example.com",
		},
	})

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.JSONBody["message"], "verification")

	// Verify new token was created
	var tokenCount int64
	testDB.DB.Model(&models.EmailVerificationToken{}).Where("user_id = ?", user.ID).Count(&tokenCount)
	assert.Equal(t, int64(1), tokenCount)
}

// TestRegistrationFlow_LoginAfterVerification tests the full flow: register -> verify -> login
func TestRegistrationFlow_LoginAfterVerification(t *testing.T) {
	testDB := integration.SetupIntegrationDB(t)
	defer integration.CleanupDB(t, testDB)

	// Setup router
	gin := integration.TestRouter()
	gin.POST("/auth/register", v2.Register)
	gin.POST("/auth/verify-email", v2.VerifyEmail)
	gin.POST("/auth/login", authHandlers.Login)

	// 1. Register
	registerResp := integration.MakeRequest(t, gin, integration.HTTPRequest{
		Method: "POST",
		Path:   "/auth/register",
		Body: map[string]interface{}{
			"username": "fullflowuser",
			"email":    "fullflow@example.com",
			"password": "Password123!",
		},
	})

	assert.Equal(t, http.StatusCreated, registerResp.Code)

	// Wait for async email verification token creation
	time.Sleep(100 * time.Millisecond)

	// Get user and token
	var user models.AuthUser
	testDB.DB.Where("username = ?", "fullflowuser").First(&user)

	var token models.EmailVerificationToken
	testDB.DB.Where("user_id = ?", user.ID).First(&token)
	assert.NotEmpty(t, token.Token)

	// 2. Verify email
	verifyResp := integration.MakeRequest(t, gin, integration.HTTPRequest{
		Method: "POST",
		Path:   "/auth/verify-email",
		Body: map[string]interface{}{
			"token": token.Token,
		},
	})

	assert.Equal(t, http.StatusOK, verifyResp.Code)

	// 3. Login should succeed
	loginResp := integration.MakeRequest(t, gin, integration.HTTPRequest{
		Method: "POST",
		Path:   "/auth/login",
		Body: map[string]interface{}{
			"username": "fullflowuser",
			"password": "Password123!",
		},
	})

	assert.Equal(t, http.StatusOK, loginResp.Code)
	assert.NotNil(t, loginResp.JSONBody["access_token"])
	assert.NotNil(t, loginResp.JSONBody["refresh_token"])
}
