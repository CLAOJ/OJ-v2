package cookie

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CLAOJ/claoj/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func withSiteURL(t *testing.T, url string) {
	t.Helper()
	orig := config.C.App.SiteFullURL
	t.Cleanup(func() { config.C.App.SiteFullURL = orig })
	config.C.App.SiteFullURL = url
}

// run invokes fn against a throwaway gin context and returns the cookies it set.
func run(t *testing.T, fn func(c *gin.Context)) []*http.Cookie {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	fn(c)
	return w.Result().Cookies()
}

func find(cookies []*http.Cookie, name string) *http.Cookie {
	for _, ck := range cookies {
		if ck.Name == name {
			return ck
		}
	}
	return nil
}

// Browsers reject `SameSite=None` unless `Secure` is also set. clearCookie used
// to hardcode SameSite=None while deriving Secure from the site URL, so over a
// plain-http deployment the deletion header was thrown away and logging out
// left a perfectly valid session cookie in the browser — the server had revoked
// the refresh token, but the access token kept working until it expired.
func TestClearAuthTokensNeverEmitsSameSiteNoneWithoutSecure(t *testing.T) {
	withSiteURL(t, "http://localhost:3000")

	cookies := run(t, func(c *gin.Context) { Helper().ClearAuthTokens(c) })

	for _, name := range []string{AccessTokenName, RefreshTokenName} {
		ck := find(cookies, name)
		require.NotNil(t, ck, "%s deletion cookie must be emitted", name)
		assert.False(t, ck.Secure, "insecure site URL should not mark the cookie Secure")
		assert.NotEqual(t, http.SameSiteNoneMode, ck.SameSite,
			"%s: SameSite=None without Secure is rejected by browsers, so the cookie would never be cleared", name)
		assert.Equal(t, http.SameSiteLaxMode, ck.SameSite, "%s should fall back to Lax", name)
	}
}

// The deletion cookie has to carry the same attributes as the one it replaces,
// otherwise the browser keeps the original and stores a second cookie.
func TestClearAuthTokensMirrorsSetAuthTokensAttributes(t *testing.T) {
	for _, siteURL := range []string{"http://localhost:3000", "https://claoj.edu.vn"} {
		t.Run(siteURL, func(t *testing.T) {
			withSiteURL(t, siteURL)

			set := run(t, func(c *gin.Context) {
				Helper().SetAuthTokens(c, "access", "refresh", RefreshTokenDuration)
			})
			cleared := run(t, func(c *gin.Context) { Helper().ClearAuthTokens(c) })

			for _, name := range []string{AccessTokenName, RefreshTokenName} {
				s, d := find(set, name), find(cleared, name)
				require.NotNil(t, s)
				require.NotNil(t, d)
				assert.Equal(t, s.Path, d.Path, "%s: Path must match", name)
				assert.Equal(t, s.Domain, d.Domain, "%s: Domain must match", name)
				assert.Equal(t, s.Secure, d.Secure, "%s: Secure must match", name)
				assert.Equal(t, s.SameSite, d.SameSite, "%s: SameSite must match", name)
				assert.Equal(t, s.HttpOnly, d.HttpOnly, "%s: HttpOnly must match", name)
			}
		})
	}
}

func TestClearAuthTokensExpiresTheCookies(t *testing.T) {
	withSiteURL(t, "https://claoj.edu.vn")

	cookies := run(t, func(c *gin.Context) { Helper().ClearAuthTokens(c) })

	for _, name := range []string{AccessTokenName, RefreshTokenName} {
		ck := find(cookies, name)
		require.NotNil(t, ck)
		assert.Empty(t, ck.Value, "%s value must be blanked", name)
		assert.Less(t, ck.MaxAge, 0, "%s must be sent with a negative Max-Age to delete it", name)
	}
}

// Over HTTPS the cross-site mode is both valid and wanted.
func TestSecureSiteStillUsesSameSiteNone(t *testing.T) {
	withSiteURL(t, "https://claoj.edu.vn")

	cookies := run(t, func(c *gin.Context) {
		Helper().SetAuthTokens(c, "access", "refresh", RefreshTokenDuration)
	})

	for _, name := range []string{AccessTokenName, RefreshTokenName} {
		ck := find(cookies, name)
		require.NotNil(t, ck)
		assert.True(t, ck.Secure)
		assert.Equal(t, http.SameSiteNoneMode, ck.SameSite)
	}
}
