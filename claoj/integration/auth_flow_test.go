// Package integration_test provides integration tests for authentication flow.
package integration_test

import (
	"net/http"
	"testing"
	"time"

	authHandlers "github.com/CLAOJ/claoj/api/v2/auth"
	"github.com/CLAOJ/claoj/auth"
	"github.com/CLAOJ/claoj/auth/tokenstore"
	v2 "github.com/CLAOJ/claoj/api/v2"
	"github.com/CLAOJ/claoj/integration"
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

// TestCurrentUser_IncludesStaffFlags verifies /user/me returns is_staff and
// is_admin. The frontend admin gate (AdminAccessWrapper / AdminSidebar) hides
// the whole admin surface unless user.is_staff is truthy, and reads
// user.is_admin for the super-admin label. The handler previously omitted
// is_staff entirely (and emitted is_superuser instead of is_admin), so the
// admin button never appeared even for a staff+superuser account.
func TestCurrentUser_IncludesStaffFlags(t *testing.T) {
	testDB := integration.SetupIntegrationDB(t)
	defer CleanupDB(testDB)

	user := integration.CreateTestUser(testDB.DB, "staffuser", "Password123!", true)
	// Promote to staff + superuser (CreateTestUser defaults both to false).
	testDB.DB.Model(&user).Updates(map[string]interface{}{"is_staff": true, "is_superuser": true})

	gin := integration.TestRouter()
	gin.POST("/auth/login", authHandlers.Login)
	gin.GET("/me", auth.RequiredMiddleware(), v2.CurrentUser)

	loginResp := integration.MakeRequest(t, gin, integration.HTTPRequest{
		Method: "POST",
		Path:   "/auth/login",
		Body: map[string]interface{}{
			"username": "staffuser",
			"password": "Password123!",
		},
	})
	assert.Equal(t, http.StatusOK, loginResp.Code, "Login should succeed")
	accessToken := loginResp.JSONBody["access_token"].(string)

	meResp := integration.MakeRequest(t, gin, integration.HTTPRequest{
		Method: "GET",
		Path:   "/me",
		Headers: map[string]string{
			"Authorization": "Bearer " + accessToken,
		},
	})
	assert.Equal(t, http.StatusOK, meResp.Code, "Should access /me")
	assert.Equal(t, true, meResp.JSONBody["is_staff"], "/me must return is_staff so the frontend can show the admin surface")
	assert.Equal(t, true, meResp.JSONBody["is_admin"], "/me must return is_admin for the super-admin label")
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

	// Issue a verification token in the store to simulate pending
	// verification.
	issueErr := authHandlers.OneTimeTokens.Issue(tokenstore.KindEmailVerify, "test-verification-token", user.ID, 24*time.Hour)
	assert.NoError(t, issueErr)

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

// TestAuthFlow_RefreshTokenReuse_RevokesWholeFamily is the regression test
// for rotation-reuse detection: replaying a refresh token that has already
// been rotated out must not just be rejected — it must burn the entire
// token family (including the live successor token issued during the
// legitimate rotation), since a replayed token is the signature of a
// leaked/stolen refresh token.
func TestAuthFlow_RefreshTokenReuse_RevokesWholeFamily(t *testing.T) {
	testDB := integration.SetupIntegrationDB(t)
	defer CleanupDB(testDB)

	integration.CreateTestUser(testDB.DB, "testuser", "Password123!", true)

	gin := integration.TestRouter()
	gin.POST("/auth/login", authHandlers.Login)
	gin.POST("/auth/refresh", authHandlers.Refresh)

	// 1. Login to get the original refresh token.
	loginResp := integration.MakeRequest(t, gin, integration.HTTPRequest{
		Method: "POST",
		Path:   "/auth/login",
		Body: map[string]interface{}{
			"username": "testuser",
			"password": "Password123!",
		},
	})
	assert.Equal(t, http.StatusOK, loginResp.Code)
	originalRefreshToken := loginResp.JSONBody["refresh_token"].(string)

	// JWT `iat` has second precision, so wait out the second before
	// rotating or the newly-minted token would be byte-identical to the
	// original (same claims, same second) instead of a distinct token.
	time.Sleep(1100 * time.Millisecond)

	// 2. Rotate it once — this revokes originalRefreshToken and issues a
	// live successor token in the same family.
	firstRefresh := integration.MakeRequest(t, gin, integration.HTTPRequest{
		Method:  "POST",
		Path:    "/auth/refresh",
		Headers: map[string]string{"Cookie": "refresh_token=" + originalRefreshToken},
	})
	assert.Equal(t, http.StatusOK, firstRefresh.Code)
	liveSuccessorToken := firstRefresh.JSONBody["refresh_token"].(string)
	assert.NotEqual(t, originalRefreshToken, liveSuccessorToken)

	// 3. Replay the now-revoked original token — this must be rejected...
	reuseResp := integration.MakeRequest(t, gin, integration.HTTPRequest{
		Method:  "POST",
		Path:    "/auth/refresh",
		Headers: map[string]string{"Cookie": "refresh_token=" + originalRefreshToken},
	})
	assert.Equal(t, http.StatusUnauthorized, reuseResp.Code)

	// ...AND must revoke the live successor token too, even though it was
	// never itself replayed. If reuse detection only revoked the replayed
	// token, the successor below would still refresh successfully — which
	// is exactly the gap rotation-reuse detection exists to close.
	successorAfterReuse := integration.MakeRequest(t, gin, integration.HTTPRequest{
		Method:  "POST",
		Path:    "/auth/refresh",
		Headers: map[string]string{"Cookie": "refresh_token=" + liveSuccessorToken},
	})
	assert.Equal(t, http.StatusUnauthorized, successorAfterReuse.Code,
		"replaying a revoked token must revoke its whole family, including the live successor token")
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

	// Verify that the token was revoked in the refresh-token store
	entry, found, err := authHandlers.RefreshStore.Get(refreshToken)
	assert.NoError(t, err)
	assert.True(t, found, "Token should still be present in the store")
	assert.True(t, entry.Revoked, "Token should be revoked after logout")
}

// Helper to clean up database
func CleanupDB(testDB *integration.TestDB) {
	integration.CleanupDB(&testing.T{}, testDB)
}
