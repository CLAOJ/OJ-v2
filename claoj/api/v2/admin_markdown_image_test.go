package v2

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// 1x1 transparent PNG.
const tinyPNGBase64 = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg=="

func useTempMediaDir(t *testing.T) {
	t.Helper()
	old := markdownMediaDir
	markdownMediaDir = t.TempDir()
	t.Cleanup(func() { markdownMediaDir = old })
}

func multipartUpload(t *testing.T, field, filename string, content []byte) (*bytes.Buffer, string) {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, err := w.CreateFormFile(field, filename)
	require.NoError(t, err)
	_, err = part.Write(content)
	require.NoError(t, err)
	require.NoError(t, w.Close())
	return &buf, w.FormDataContentType()
}

func uploadImage(t *testing.T, filename string, content []byte) *httptest.ResponseRecorder {
	t.Helper()
	body, contentType := multipartUpload(t, "image", filename, content)
	req := httptest.NewRequest(http.MethodPost, "/admin/markdown/image", body)
	req.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()
	authedRouter(1, http.MethodPost, "/admin/markdown/image", AdminMarkdownImageUpload).ServeHTTP(w, req)
	return w
}

func TestAdminMarkdownImageUpload_SavesAndServes(t *testing.T) {
	useTempMediaDir(t)
	png, err := base64.StdEncoding.DecodeString(tinyPNGBase64)
	require.NoError(t, err)

	w := uploadImage(t, "screenshot.PNG", png)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var body struct {
		URL string `json:"url"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Regexp(t, `^/api/media/[0-9a-f]{16}\.png$`, body.URL)

	name := strings.TrimPrefix(body.URL, "/api/media/")
	onDisk, err := os.ReadFile(filepath.Join(markdownMediaDir, name))
	require.NoError(t, err)
	require.Equal(t, png, onDisk, "extension and contents come from the uploaded bytes")

	// Serve it back through the public route.
	router := authedRouter(0, http.MethodGet, "/media/:name", MarkdownMediaServe)
	got := doRequest(router, http.MethodGet, "/media/"+name, "")
	require.Equal(t, http.StatusOK, got.Code, got.Body.String())
	require.Equal(t, "image/png", got.Header().Get("Content-Type"))
	require.Contains(t, got.Header().Get("Cache-Control"), "immutable")
	require.Equal(t, png, got.Body.Bytes())
}

func TestAdminMarkdownImageUpload_RejectsBadInput(t *testing.T) {
	useTempMediaDir(t)

	t.Run("wrong content type", func(t *testing.T) {
		w := uploadImage(t, "notes.png", []byte("this is not an image"))
		require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	})

	t.Run("oversize", func(t *testing.T) {
		oversize := make([]byte, maxMarkdownImageBytes+1)
		copy(oversize, []byte("\x89PNG\r\n\x1a\n"))
		w := uploadImage(t, "big.png", oversize)
		require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	})

	t.Run("missing field", func(t *testing.T) {
		body, contentType := multipartUpload(t, "other", "x.png", []byte("\x89PNG\r\n\x1a\n"))
		req := httptest.NewRequest(http.MethodPost, "/admin/markdown/image", body)
		req.Header.Set("Content-Type", contentType)
		w := httptest.NewRecorder()
		authedRouter(1, http.MethodPost, "/admin/markdown/image", AdminMarkdownImageUpload).ServeHTTP(w, req)
		require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	})

	entries, err := os.ReadDir(markdownMediaDir)
	require.NoError(t, err)
	require.Empty(t, entries, "rejected uploads must not leave files behind")
}

func TestMarkdownMediaServe_RejectsNonWhitelistedNames(t *testing.T) {
	useTempMediaDir(t)
	router := authedRouter(0, http.MethodGet, "/media/:name", MarkdownMediaServe)

	for _, name := range []string{"abc.png", "0123456789ABCDEF.png", "0123456789abcdef.exe", "0123456789abcdef.svg"} {
		w := doRequest(router, http.MethodGet, "/media/"+name, "")
		require.Equal(t, http.StatusNotFound, w.Code, name)
	}

	// A whitelisted-but-missing file is also a 404, not a directory listing.
	w := doRequest(router, http.MethodGet, "/media/0123456789abcdef.png", "")
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestMarkdownMediaName_Whitelist(t *testing.T) {
	valid := []string{"0123456789abcdef.png", "0123456789abcdef.jpg", "0123456789abcdef.gif", "0123456789abcdef.webp"}
	for _, name := range valid {
		require.True(t, markdownMediaName.MatchString(name), name)
	}
	invalid := []string{
		"", ".", "..", "../etc/passwd", "/etc/passwd", `..\windows.png`,
		"0123456789abcdef.png/../x", "0123456789abcdef.svg", "0123456789abcdef.png\x00",
	}
	for _, name := range invalid {
		require.False(t, markdownMediaName.MatchString(name), name)
	}
}
