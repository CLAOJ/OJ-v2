package csrf

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestPlantSetsOneCookieOnGET(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Plant(DefaultConfig()), Middleware(DefaultConfig()))
	r.GET("/x", func(c *gin.Context) { c.Status(204) })
	r.PUT("/x", func(c *gin.Context) { c.Status(204) })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/x", nil))
	cookies := w.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != CSRFTokenCookie || cookies[0].Value == "" {
		t.Fatalf("want one csrf cookie, got %v", cookies)
	}

	req := httptest.NewRequest("PUT", "/x", nil)
	req.AddCookie(cookies[0])
	req.Header.Set(CSRFTokenHeader, cookies[0].Value)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 204 {
		t.Fatalf("PUT with planted token: %d", w.Code)
	}

	// An existing cookie is re-sent with the same value.
	req = httptest.NewRequest("GET", "/x", nil)
	req.AddCookie(cookies[0])
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if again := w.Result().Cookies(); len(again) != 1 || again[0].Value != cookies[0].Value {
		t.Fatalf("want the same cookie re-sent, got %v", again)
	}

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("PUT", "/x", nil))
	if w.Code != http.StatusForbidden {
		t.Fatalf("PUT without token: %d", w.Code)
	}
}
