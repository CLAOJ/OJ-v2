package cookie

import (
	"net/http"
	"strings"

	"github.com/CLAOJ/claoj/config"
	"github.com/gin-gonic/gin"
)

// SameSiteMode for cookies
type SameSiteMode string

const (
	SameSiteStrict SameSiteMode = "strict"
	SameSiteLax    SameSiteMode = "lax"
	SameSiteNone   SameSiteMode = "none"
)

// Cookie durations in seconds
const (
	AccessTokenDuration  = 900 // 15 minutes
	RefreshTokenDuration = 7 * 24 * 60 * 60 // 7 days
	ExtendedRefreshTokenDuration = 30 * 24 * 60 * 60 // 30 days
	OAuthStateDuration   = 600 // 10 minutes
	WebAuthnSessionDuration = 300 // 5 minutes
)

// Cookie names
const (
	AccessTokenName  = "access_token"
	RefreshTokenName = "refresh_token"
	OAuthStateName   = "oauth_state"
	WebAuthnRegistrationSession = "webauthn_registration_session"
	WebAuthnLoginSession        = "webauthn_login_session"
)

// Helper returns a new CookieHelper instance
func Helper() *CookieHelper {
	return &CookieHelper{}
}

// CookieHelper provides centralized cookie management
type CookieHelper struct {
	secureCookie *bool
}

// isSecureCookie determines if cookies should use Secure flag
func (h *CookieHelper) isSecureCookie() bool {
	if h.secureCookie != nil {
		return *h.secureCookie
	}
	secure := strings.HasPrefix(config.C.App.SiteFullURL, "https://")
	h.secureCookie = &secure
	return secure
}

// getSameSite returns the SameSite mode based on secure flag
func (h *CookieHelper) getSameSite(mode SameSiteMode) http.SameSite {
	if mode == SameSiteNone {
		return http.SameSiteNoneMode
	}
	if mode == SameSiteStrict {
		return http.SameSiteStrictMode
	}
	return http.SameSiteLaxMode
}

// SetAuthTokens sets both access and refresh token cookies
func (h *CookieHelper) SetAuthTokens(c *gin.Context, accessToken, refreshToken string, maxAge int) {
	h.setTokenCookie(c, AccessTokenName, accessToken, AccessTokenDuration, SameSiteNone)
	h.setTokenCookie(c, RefreshTokenName, refreshToken, maxAge, SameSiteNone)
}

// SetAccessToken sets only the access token cookie
func (h *CookieHelper) SetAccessToken(c *gin.Context, token string) {
	h.setTokenCookie(c, AccessTokenName, token, AccessTokenDuration, SameSiteNone)
}

// SetRefreshToken sets only the refresh token cookie
func (h *CookieHelper) SetRefreshToken(c *gin.Context, token string, maxAge int) {
	h.setTokenCookie(c, RefreshTokenName, token, maxAge, SameSiteNone)
}

// ClearAuthTokens clears both access and refresh token cookies
func (h *CookieHelper) ClearAuthTokens(c *gin.Context) {
	h.clearCookie(c, AccessTokenName)
	h.clearCookie(c, RefreshTokenName)
}

// SetOAuthState sets the OAuth state cookie
func (h *CookieHelper) SetOAuthState(c *gin.Context, state string) {
	secure := h.isSecureCookie()
	cookie := &http.Cookie{
		Name:     OAuthStateName,
		Value:    state,
		MaxAge:   OAuthStateDuration,
		Path:     "/api/v2/auth",
		Domain:   "",
		Secure:   secure,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	}
	http.SetCookie(c.Writer, cookie)
}

// ClearOAuthState clears the OAuth state cookie
func (h *CookieHelper) ClearOAuthState(c *gin.Context) {
	secure := h.isSecureCookie()
	cookie := &http.Cookie{
		Name:     OAuthStateName,
		Value:    "",
		MaxAge:   -1,
		Path:     "/api/v2/auth",
		Domain:   "",
		Secure:   secure,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	}
	http.SetCookie(c.Writer, cookie)
}

// SetWebAuthnRegistrationSession sets the WebAuthn registration session cookie
func (h *CookieHelper) SetWebAuthnRegistrationSession(c *gin.Context, sessionData string) {
	h.setTokenCookie(c, WebAuthnRegistrationSession, sessionData, WebAuthnSessionDuration, SameSiteNone)
}

// ClearWebAuthnRegistrationSession clears the WebAuthn registration session cookie
func (h *CookieHelper) ClearWebAuthnRegistrationSession(c *gin.Context) {
	h.clearCookie(c, WebAuthnRegistrationSession)
}

// SetWebAuthnLoginSession sets the WebAuthn login session cookie
func (h *CookieHelper) SetWebAuthnLoginSession(c *gin.Context, sessionData string) {
	h.setTokenCookie(c, WebAuthnLoginSession, sessionData, WebAuthnSessionDuration, SameSiteNone)
}

// ClearWebAuthnLoginSession clears the WebAuthn login session cookie
func (h *CookieHelper) ClearWebAuthnLoginSession(c *gin.Context) {
	h.clearCookie(c, WebAuthnLoginSession)
}

// setTokenCookie is a helper to set httpOnly, secure cookies with SameSite
func (h *CookieHelper) setTokenCookie(c *gin.Context, name, value string, maxAge int, sameSite SameSiteMode) {
	secure := h.isSecureCookie()
	c.SetSameSite(h.getSameSite(sameSite))
	c.SetCookie(name, value, maxAge, "/", "", secure, true)
}

// clearCookie clears a cookie by setting MaxAge to -1
func (h *CookieHelper) clearCookie(c *gin.Context, name string) {
	secure := h.isSecureCookie()
	c.SetSameSite(http.SameSiteNoneMode)
	c.SetCookie(name, "", -1, "/", "", secure, true)
}

// GetCookie retrieves a cookie value, returns empty string if not found
func (h *CookieHelper) GetCookie(c *gin.Context, name string) string {
	value, err := c.Cookie(name)
	if err != nil {
		return ""
	}
	return value
}
