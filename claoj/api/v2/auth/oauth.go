package auth

import (
	"net/http"

	"github.com/CLAOJ/claoj/auth"
	"github.com/CLAOJ/claoj/config"
	"github.com/CLAOJ/claoj/cookie"
	"github.com/CLAOJ/claoj/oauth"
	"github.com/gin-gonic/gin"
)

// OAuthCallbackRequest - POST /api/v2/auth/oauth/:provider/callback
type OAuthCallbackRequest struct {
	Code     string `json:"code" binding:"required"`
	State    string `json:"state" binding:"required"`
	Password string `json:"password,omitempty"` // For linking to existing account
}

// OAuthStart - GET /api/v2/auth/oauth/:provider
// Starts OAuth flow by redirecting to provider
func OAuthStart(c *gin.Context) {
	provider := c.Param("provider")

	if provider != "google" && provider != "github" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported provider"})
		return
	}

	// Generate state token for CSRF protection
	state, err := oauth.GenerateStateToken(config.C.App.SecretKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate state token"})
		return
	}

	// Store state in cookie with SameSite attribute for CSRF protection
	cookieHelper := cookie.Helper()
	cookieHelper.SetOAuthState(c, state)

	// Get auth URL
	authURL, err := oauth.GetAuthURL(oauth.Provider(provider), state)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "OAuth provider not configured"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"auth_url": authURL})
}

// OAuthCallback - POST /api/v2/auth/oauth/:provider/callback
// Handles OAuth callback and returns JWT tokens
func OAuthCallback(c *gin.Context) {
	provider := c.Param("provider")

	if provider != "google" && provider != "github" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported provider"})
		return
	}

	var req OAuthCallbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify state token
	storedState, err := c.Cookie("oauth_state")
	if err != nil || storedState != req.State {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid state parameter"})
		return
	}
	// Clear state cookie
	cookieHelper := cookie.Helper()
	cookieHelper.ClearOAuthState(c)

	ctx := c.Request.Context()

	// Exchange code for token
	token, err := oauth.ExchangeCode(ctx, oauth.Provider(provider), req.Code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to exchange code"})
		return
	}

	// Get user info
	userInfo, err := oauth.GetUserInfo(ctx, oauth.Provider(provider), token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get user info"})
		return
	}

	// Find or create user - may require password confirmation
	userID, requiresLinking, err := findOrCreateOAuthUser(userInfo, req.Password)
	if err != nil {
		if err.Error() == "password_required_for_linking" {
			c.JSON(http.StatusConflict, gin.H{
				"error":   "account_linking_required",
				"message": "An account with this email already exists. Please provide your password to link accounts.",
				"email":   userInfo.Email,
			})
			return
		}
		if err.Error() == "invalid_password_for_linking" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "invalid_password",
				"message": "Invalid password. Please try again or use a different email.",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Generate JWT tokens (OAuth logins don't have remember_me, use default 7 days)
	accessToken, refreshToken, _, err := auth.GenerateTokens(userID, userInfo.Email, false, "", false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate tokens"})
		return
	}

	// Set httpOnly cookies
	cookieHelper.SetAuthTokens(c, accessToken, refreshToken, cookie.RefreshTokenDuration)

	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"user": gin.H{
			"id":       userID,
			"username": userInfo.Email,
			"email":    userInfo.Email,
			"name":     userInfo.Name,
			"avatar":   userInfo.AvatarURL,
			"provider": userInfo.Provider,
			"linked":   !requiresLinking,
		},
	})
}
