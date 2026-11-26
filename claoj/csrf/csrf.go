// Package csrf provides CSRF protection middleware using the double-submit cookie pattern
package csrf

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	// CSRFTokenCookie is the name of the CSRF token cookie
	CSRFTokenCookie = "csrf_token"

	// CSRFTokenHeader is the name of the CSRF token header
	CSRFTokenHeader = "X-CSRF-Token"

	// TokenLength is the length of the CSRF token in bytes
	TokenLength = 32
)

// generateToken creates a new random CSRF token
func generateToken() (string, error) {
	bytes := make([]byte, TokenLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// Config holds CSRF middleware configuration
type Config struct {
	// Secure sets cookie Secure flag
	Secure bool

	// Domain sets cookie domain
	Domain string

	// Path sets cookie path
	Path string

	// ExcludedPaths are paths that don't require CSRF validation
	ExcludedPaths []string
}

// DefaultConfig returns a default configuration
func DefaultConfig() Config {
	return Config{
		Secure: true,
		Domain: "",
		Path:   "/",
		ExcludedPaths: []string{
			"/api/auth/login",
			"/api/auth/register",
			"/api/auth/password/reset",
			"/api/auth/password/reset/confirm",
			"/api/auth/verify-email",
			"/api/auth/resend-verification",
			"/api/auth/oauth/",
			"/health",
		},
	}
}

func safeMethod(m string) bool { return m == "GET" || m == "HEAD" || m == "OPTIONS" }

// plant makes sure the browser holds a token cookie (once per request).
func plant(c *gin.Context, config Config) {
	if _, done := c.Get("csrf_token"); done {
		return
	}
	token, err := c.Cookie(CSRFTokenCookie)
	if err != nil {
		if token, err = generateToken(); err == nil {
			setCSRFCookie(c, token, config)
		}
	}
	c.Set("csrf_token", token)
}

// Plant sets the token cookie on every safe request. Without it the cookie
// only came from GETs to the protected groups, so a page that reads public
// endpoints only (e.g. admin name styles) had no token for its first write.
func Plant(config Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		if safeMethod(c.Request.Method) {
			plant(c, config)
		}
		c.Next()
	}
}

// Middleware returns the CSRF protection middleware
func Middleware(config Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip CSRF for safe methods
		if safeMethod(c.Request.Method) {
			plant(c, config)
			c.Next()
			return
		}

		// Check if path is excluded
		for _, excluded := range config.ExcludedPaths {
			if strings.HasPrefix(c.Request.URL.Path, excluded) {
				c.Next()
				return
			}
		}

		// Get token from cookie
		cookieToken, err := c.Cookie(CSRFTokenCookie)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":   "csrf_token_missing",
				"message": "CSRF token cookie not found",
			})
			return
		}

		// Get token from header
		headerToken := c.GetHeader(CSRFTokenHeader)
		if headerToken == "" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":   "csrf_token_missing",
				"message": "CSRF token header not found",
			})
			return
		}

		// Validate tokens match
		if cookieToken != headerToken {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":   "csrf_token_invalid",
				"message": "CSRF token validation failed",
			})
			return
		}

		c.Set("csrf_token", cookieToken)
		c.Next()
	}
}

// setCSRFCookie sets the CSRF token cookie
func setCSRFCookie(c *gin.Context, token string, config Config) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:   CSRFTokenCookie,
		Value:  token,
		MaxAge: 86400, // 24 hours
		Path:   config.Path,
		Domain: config.Domain,
		Secure: config.Secure,
		// Double-submit pattern: the client must read this cookie to echo
		// it in the X-CSRF-Token header, so it cannot be HttpOnly. Its
		// value is not a secret from same-origin JS — the protection comes
		// from cross-origin pages being unable to read it.
		HttpOnly: false,
		SameSite: http.SameSiteStrictMode,
	})
}

// GetToken returns the current CSRF token from context
func GetToken(c *gin.Context) string {
	token, _ := c.Get("csrf_token")
	if t, ok := token.(string); ok {
		return t
	}
	return ""
}

// GetCookieName returns the CSRF cookie name
func GetCookieName() string {
	return CSRFTokenCookie
}

// GetHeaderName returns the CSRF header name
func GetHeaderName() string {
	return CSRFTokenHeader
}
