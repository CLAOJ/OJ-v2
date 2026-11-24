package v2

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"

	"github.com/gin-gonic/gin"
)

// Shared markdown image upload (blog posts + license text). Files live in the
// v2 data volume next to problem data and are served by MarkdownMediaServe.

const maxMarkdownImageBytes = 2 << 20 // 2 MiB

// Cap on the whole multipart body, so FormFile never buffers a huge upload.
// The slack covers the multipart envelope of a legal 2 MiB image; anything
// bigger is rejected before the file-size check runs.
const maxMarkdownUploadBytes = maxMarkdownImageBytes + (64 << 10)

// Files are named by the server (<16 hex>.<ext>), so the serve route can
// require exactly that shape — anything else (".", "..", slashes) is a 404.
var markdownMediaName = regexp.MustCompile(`^[0-9a-f]{16}\.(png|jpg|gif|webp)$`)

// Upload sniffing decides the extension from the bytes, never the filename.
var markdownImageExts = map[string]string{
	"image/png":  ".png",
	"image/jpeg": ".jpg",
	"image/gif":  ".gif",
	"image/webp": ".webp",
}

var markdownMediaContentTypes = map[string]string{
	".png":  "image/png",
	".jpg":  "image/jpeg",
	".gif":  "image/gif",
	".webp": "image/webp",
}

var markdownMediaDir = filepath.Join("data", "media")

// AdminMarkdownImageUpload - POST /api/admin/markdown/image
// Accepts multipart field "image" and returns the public URL of the saved file.
func AdminMarkdownImageUpload(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxMarkdownUploadBytes)
	file, err := c.FormFile("image")
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			c.JSON(http.StatusRequestEntityTooLarge, apiUserError("image must be at most 2 MB"))
			return
		}
		c.JSON(http.StatusBadRequest, apiUserError("no image uploaded"))
		return
	}
	if file.Size > maxMarkdownImageBytes {
		c.JSON(http.StatusBadRequest, apiUserError("image must be at most 2 MB"))
		return
	}

	opened, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}
	header := make([]byte, 512)
	n, err := io.ReadFull(opened, header)
	opened.Close()
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}
	ext, ok := markdownImageExts[http.DetectContentType(header[:n])]
	if !ok {
		c.JSON(http.StatusBadRequest, apiUserError("only PNG, JPEG, GIF or WebP images are allowed"))
		return
	}

	random := make([]byte, 8)
	if _, err := rand.Read(random); err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}
	name := hex.EncodeToString(random) + ext

	if err := os.MkdirAll(markdownMediaDir, 0o755); err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}
	if err := c.SaveUploadedFile(file, filepath.Join(markdownMediaDir, name)); err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{"url": "/api/media/" + name})
}

// MarkdownMediaServe - GET /api/media/:name
// Public read-only serving of uploaded markdown images.
func MarkdownMediaServe(c *gin.Context) {
	name := c.Param("name")
	if !markdownMediaName.MatchString(name) {
		c.JSON(http.StatusNotFound, gin.H{"error": "media not found"})
		return
	}

	path := filepath.Join(markdownMediaDir, name)
	if info, err := os.Stat(path); err != nil || info.IsDir() {
		c.JSON(http.StatusNotFound, gin.H{"error": "media not found"})
		return
	}

	c.Header("Content-Type", markdownMediaContentTypes[filepath.Ext(name)])
	c.Header("Cache-Control", "public, max-age=31536000, immutable")
	c.Header("X-Content-Type-Options", "nosniff")
	c.File(path)
}
