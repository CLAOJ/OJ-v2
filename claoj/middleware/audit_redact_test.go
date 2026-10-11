package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/CLAOJ/claoj/auditlog"
	"github.com/gin-gonic/gin"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRedactSecrets(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"password", `{"username":"bob","password":"hunter2"}`, `{"username":"bob","password":"[redacted]"}`},
		{"auth_key", `{"name":"j1","auth_key":"abc"}`, `{"name":"j1","auth_key":"[redacted]"}`},
		{"mixed case", `{"New_Password":"x","Auth_Key":"y"}`, `{"New_Password":"[redacted]","Auth_Key":"[redacted]"}`},
		{"secret and token", `{"client_secret":"s","api_token":"t","email":"a@b.c"}`, `{"client_secret":"[redacted]","api_token":"[redacted]","email":"a@b.c"}`},
		{"nested object", `{"a":{"b":{"password":"p","keep":1}}}`, `{"a":{"b":{"password":"[redacted]","keep":1}}}`},
		{"array of objects", `{"users":[{"password":"p","n":1},{"n":2}]}`, `{"users":[{"password":"[redacted]","n":1},{"n":2}]}`},
		{"non-secret unchanged", `{"name":"x","description":"authentic","auth":"keep"}`, `{"name":"x","description":"authentic","auth":"keep"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var v any
			require.NoError(t, json.Unmarshal([]byte(tc.in), &v))
			got, err := json.Marshal(redactSecrets(v))
			require.NoError(t, err)
			require.JSONEq(t, tc.want, string(got))
		})
	}
}

func TestAudit_Middleware_StoresRedactedBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := auditlog.NewMemoryStore()
	prev := auditlog.Default
	auditlog.Default = store
	t.Cleanup(func() { auditlog.Default = prev })

	router := gin.New()
	router.Use(Audit())
	var seen string
	router.POST("/admin/users", func(c *gin.Context) {
		b, _ := io.ReadAll(c.Request.Body)
		seen = string(b)
		c.JSON(http.StatusOK, gin.H{})
	})

	body := `{"username":"bob","password":"hunter2"}`
	req := httptest.NewRequest(http.MethodPost, "/admin/users", strings.NewReader(body))
	router.ServeHTTP(httptest.NewRecorder(), req)
	require.Equal(t, body, seen, "downstream handler must still see the real body")

	entries := waitForEntries(t, store)
	require.NotContains(t, entries[0].Details, "hunter2")
	require.Contains(t, entries[0].Details, "[redacted]")
	require.Contains(t, entries[0].Details, "bob")
}
