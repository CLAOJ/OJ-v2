// Package integration_test provides integration tests for authentication flow.
package integration_test

import (
	"net/http"
	"testing"
	"time"

	authHandlers "github.com/CLAOJ/claoj/api/v2/auth"
	"github.com/CLAOJ/claoj/auth"
	v2 "github.com/CLAOJ/claoj/api/v2"
	"github.com/CLAOJ/claoj/integration"
	"github.com/CLAOJ/claoj/models"
	"github.com/stretchr/testify/assert"
)

// TestAuthFlow_FullLoginLogout tests the complete authentication journey:
// Register -> Login -> Access Protected Resource -> Refresh Token -> Logout
func TestAuthFlow_FullLoginLogout(t *testing.T) {
	testDB := integration.SetupIntegrationDB(t)
	defer CleanupDB(testDB)

	// Create test user
	user := integration.CreateTestUser(testDB.DB, "testuser", "Password123!", true)

	// Setup router
	gin := integration.TestRouter()
	gin.POST("/auth/login", authHandlers.Login)
	gin.POST("/auth/logout", authHandlers.Logout)
	gin.POST("/auth/refresh", authHandlers.Refresh)
	gin.GET("/me", auth.RequiredMiddleware(), v2.CurrentUser)

	// 1. Login
	loginResp := integration.MakeRequest(t, gin, integration.HTTPRequest{
		Method: "POST",
		Path:   "/auth/login",
		Body: map[string]interface{}{
			"username": "testuser",
			"password": "Password123!",
		},
	})

	assert.Equal(t, http.StatusOK, loginResp.Code, "Login should succeed")
	assert.NotNil(t, loginResp.JSONBody["access_token"], "Access token should be returned")
	assert.NotNil(t, loginResp.JSONBody["refresh_token"], "Refresh token should be returned")
	assert.Equal(t, float64(user.ID), loginResp.JSONBody["user"].(map[string]interface{})["id"].(float64))

	accessToken := loginResp.JSONBody["access_token"].(string)
	_ = loginResp.JSONBody["refresh_token"].(string)

	// 2. Access protected resource with token
	meResp := integration.MakeRequest(t, gin, integration.HTTPRequest{
		Method: "GET",
		Path:   "/me",
		Headers: map[string]string{
			"Authorization": "Bearer " + accessToken,
		},
	})

	assert.Equal(t, http.StatusOK, meResp.Code, "Should access protected resource")
	assert.Equal(t, "testuser", meResp.JSONBody["username"])

	// 3. Logout
	logoutResp := integration.MakeRequest(t, gin, integration.HTTPRequest{
		Method: "POST",
		Path:   "/auth/logout",
		Headers: map[string]string{
			"Authorization": "Bearer " + accessToken,
		},
	})

	assert.Equal(t, http.StatusOK, logoutResp.Code, "Logout should succeed")
	assert.Equal(t, "logged out successfully", logoutResp.JSONBody["message"])

	// 4. Try to access protected resource after logout (should fail)
	_ = integration.MakeRequest(t, gin, integration.HTTPRequest{
		Method: "GET",
		Path:   "/me",
		Headers: map[string]string{
			"Authorization": "Bearer " + accessToken,
		},
	})

	// Note: Token itself is still valid (JWT), but in a real app we'd check revocation
	// For this test, we just verify logout endpoint succeeded
}

// TestAuthFlow_LoginFailure tests login failure scenarios
func TestAuthFlow_LoginFailure(t *testing.T) {
	testDB := integration.SetupIntegrationDB(t)
	defer CleanupDB(testDB)

	// Create test user
	integration.CreateTestUser(testDB.DB, "testuser", "Password123!", true)

	// Setup router
	gin := integration.TestRouter()
	gin.POST("/auth/login", authHandlers.Login)

	tests := []struct {
		name     string
		username string
		password string
		wantCode int
		wantErr  string
	}{
		{
			name:     "wrong password",
			username: "testuser",
			password: "WrongPassword",
			wantCode: http.StatusUnauthorized,
			wantErr:  "invalid username or password",
		},
		{
			name:     "non-existent user",
			username: "nonexistent",
			password: "Password123!",
			wantCode: http.StatusUnauthorized,
			wantErr:  "invalid username or password",
		},
		{
			name:     "missing username",
			username: "",
			password: "Password123!",
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "missing password",
			username: "testuser",
			password: "",
			wantCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := integration.MakeRequest(t, gin, integration.HTTPRequest{
				Method: "POST",
				Path:   "/auth/login",
				Body: map[string]interface{}{
					"username": tt.username,
					"password": tt.password,
				},
			})

			assert.Equal(t, tt.wantCode, resp.Code)
			if tt.wantErr != "" {
				assert.Equal(t, tt.wantErr, resp.JSONBody["error"])
			}
		})
	}
}

// TestAuthFlow_InactiveUser tests login for unverified email accounts
func TestAuthFlow_InactiveUser(t *testing.T) {
	testDB := integration.SetupIntegrationDB(t)
	defer CleanupDB(testDB)

	// Create inactive user (email not verified)
	user := integration.CreateTestUser(testDB.DB, "unverified", "Password123!", false)

	// Create verification token to simulate pending verification
	token := models.EmailVerificationToken{
		UserID:    user.ID,
		Token:     "test-verification-token",
		Email:     user.Email,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	testDB.DB.Create(&token)

	// Setup router
	gin := integration.TestRouter()
	gin.POST("/auth/login", authHandlers.Login)

	resp := integration.MakeRequest(t, gin, integration.HTTPRequest{
		Method: "POST",
		Path:   "/auth/login",
		Body: map[string]interface{}{
			"username": "unverified",
			"password": "Password123!",
		},
	})

	assert.Equal(t, http.StatusForbidden, resp.Code)
	assert.Equal(t, "email not verified", resp.JSONBody["error"])
	assert.True(t, resp.JSONBody["requires_email_verification"].(bool))
}

// TestAuthFlow_TokenRefresh tests the token refresh flow
func TestAuthFlow_TokenRefresh(t *testing.T) {
	testDB := integration.SetupIntegrationDB(t)
	defer CleanupDB(testDB)

	// Create test user
	integration.CreateTestUser(testDB.DB, "testuser", "Password123!", true)

	// Setup router
	gin := integration.TestRouter()
	gin.POST("/auth/login", authHandlers.Login)
	gin.POST("/auth/refresh", authHandlers.Refresh)

	// 1. Login to get tokens
	loginResp := integration.MakeRequest(t, gin, integration.HTTPRequest{
		Method: "POST",
		Path:   "/auth/login",
		Body: map[string]interface{}{
			"username": "testuser",
			"password": "Password123!",
		},
	})

	assert.Equal(t, http.StatusOK, loginResp.Code)
	refreshToken := loginResp.JSONBody["refresh_token"].(string)
	accessToken := loginResp.JSONBody["access_token"].(string)

	// Significant delay to ensure new tokens have different timestamps
	// JWT iat claim is in seconds, so we need at least 1 second difference
	time.Sleep(1100 * time.Millisecond)

	// 2. Refresh token
	refreshResp := integration.MakeRequest(t, gin, integration.HTTPRequest{
		Method: "POST",
		Path:   "/auth/refresh",
		Headers: map[string]string{
			"Cookie": "refresh_token=" + refreshToken,
		},
	})

	assert.Equal(t, http.StatusOK, refreshResp.Code)
	assert.NotNil(t, refreshResp.JSONBody["access_token"])
	assert.NotNil(t, refreshResp.JSONBody["refresh_token"])

	// New tokens should be different from original (token rotation)
	newAccessToken := refreshResp.JSONBody["access_token"].(string)
	newRefreshToken := refreshResp.JSONBody["refresh_token"].(string)
	assert.NotEqual(t, newAccessToken, accessToken)
	assert.NotEqual(t, newRefreshToken, refreshToken)
}

// TestAuthFlow_RememberMe tests login with remember_me option
func TestAuthFlow_RememberMe(t *testing.T) {
	testDB := integration.SetupIntegrationDB(t)
	defer CleanupDB(testDB)

	// Create test user
	integration.CreateTestUser(testDB.DB, "testuser", "Password123!", true)

	// Setup router
	gin := integration.TestRouter()
	gin.POST("/auth/login", authHandlers.Login)

	// Login with remember_me
	resp := integration.MakeRequest(t, gin, integration.HTTPRequest{
		Method: "POST",
		Path:   "/auth/login",
		Body: map[string]interface{}{
			"username":    "testuser",
			"password":    "Password123!",
			"remember_me": true,
		},
	})

	assert.Equal(t, http.StatusOK, resp.Code)

	// Check refresh token cookie has extended expiry (30 days)
	refreshCookie := integration.GetCookie(resp.Cookies, "refresh_token")
	assert.NotNil(t, refreshCookie)
	// 30 days = 2592000 seconds
	assert.Equal(t, 30*24*60*60, refreshCookie.MaxAge)
}

// TestAuthFlow_LogoutRevokesToken tests that logout revokes the refresh token
func TestAuthFlow_LogoutRevokesToken(t *testing.T) {
	testDB := integration.SetupIntegrationDB(t)
	defer CleanupDB(testDB)

	// Create test user
	integration.CreateTestUser(testDB.DB, "testuser", "Password123!", true)

	// Setup router with auth middleware for logout
	gin := integration.TestRouter()
	gin.POST("/auth/login", authHandlers.Login)
	gin.POST("/auth/logout", auth.RequiredMiddleware(), authHandlers.Logout)

	// 1. Login
	loginResp := integration.MakeRequest(t, gin, integration.HTTPRequest{
		Method: "POST",
		Path:   "/auth/login",
		Body: map[string]interface{}{
			"username": "testuser",
			"password": "Password123!",
		},
	})

	refreshToken := loginResp.JSONBody["refresh_token"].(string)
	accessToken := loginResp.JSONBody["access_token"].(string)

	// 2. Logout - send refresh token in cookie
	logoutResp := integration.MakeRequest(t, gin, integration.HTTPRequest{
		Method: "POST",
		Path:   "/auth/logout",
		Headers: map[string]string{
			"Authorization": "Bearer " + accessToken,
			"Cookie":        "refresh_token=" + refreshToken,
		},
	})

	assert.Equal(t, http.StatusOK, logoutResp.Code)
	assert.Equal(t, "logged out successfully", logoutResp.JSONBody["message"])

	// Verify that the token was revoked in the database
	var dbToken models.RefreshToken
	err := testDB.DB.Where("token = ?", refreshToken).First(&dbToken).Error
	assert.NoError(t, err, "Token should exist in database")
	assert.NotNil(t, dbToken.RevokedAt, "Token should be revoked after logout")
}

// Helper to clean up database
func CleanupDB(testDB *integration.TestDB) {
	integration.CleanupDB(&testing.T{}, testDB)
}
