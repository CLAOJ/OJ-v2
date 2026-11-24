package v2

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
)

// Submission files for file-only languages. v1 stored them in the media tree
// and put the resulting URL in SubmissionSource.source; the judge packet then
// sends SiteFullURL + source (jobs/judge.go), so the file must be reachable
// through the public site. Files live beside problem data in the v2 data
// volume and are served by SubmissionFileServe.
var submissionFileDir = filepath.Join("data", "submissions")

// Files are named by the server (<16 hex>.<ext>), so the serve route can
// require exactly that shape: no slashes, dots or traversal in the name. The
// extension alternation is built from the whitelist, so a name like
// "<hex>.html" never reaches the filesystem.
var submissionFileName = func() *regexp.Regexp {
	exts := make([]string, 0, len(submissionFileContentTypes))
	for ext := range submissionFileContentTypes {
		exts = append(exts, regexp.QuoteMeta(ext))
	}
	return regexp.MustCompile(`^[0-9a-f]{16}\.(` + strings.Join(exts, "|") + `)$`)
}()

// submissionFileContentTypes is both the extension whitelist and the exact
// Content-Type each one is served with. User files are always served as an
// attachment, never inline: serving an uploaded .html/.svg/.js from our own
// origin would be stored XSS, so those types are simply not accepted.
var submissionFileContentTypes = map[string]string{
	"zip":   "application/zip",
	"txt":   textPlain,
	"md":    textPlain,
	"csv":   textPlain,
	"json":  "application/json",
	"yaml":  textPlain,
	"yml":   textPlain,
	"c":     textPlain,
	"cc":    textPlain,
	"cpp":   textPlain,
	"cxx":   textPlain,
	"h":     textPlain,
	"hh":    textPlain,
	"hpp":   textPlain,
	"cs":    textPlain,
	"java":  textPlain,
	"kt":    textPlain,
	"py":    textPlain,
	"rb":    textPlain,
	"php":   textPlain,
	"js":    textPlain,
	"ts":    textPlain,
	"go":    textPlain,
	"rs":    textPlain,
	"hs":    textPlain,
	"lua":   textPlain,
	"r":     textPlain,
	"swift": textPlain,
	"pas":   textPlain,
	"vb":    textPlain,
	"sh":    textPlain,
	"sql":   textPlain,
	"in":    textPlain,
	"out":   textPlain,
	"ans":   textPlain,
}

const textPlain = "text/plain; charset=utf-8"

// submissionFileError is a validation failure whose message is safe to show
// the user (a 400, not a 500).
type submissionFileError string

func (e submissionFileError) Error() string { return string(e) }

// publicSubmissionFilePath is the URL stored in SubmissionSource.source. The
// judge prepends SiteFullURL, which the public site proxies to /api/.
func publicSubmissionFilePath(name string) string { return "/api/submission-file/" + name }

// saveSubmissionFile validates and stores one uploaded submission file,
// returning its public path. v1 parity (judge/forms.py:327-338): the extension
// must match the language's own extension and the size must fit
// language.file_size_limit (MB).
func saveSubmissionFile(c *gin.Context, file *multipart.FileHeader, lang *models.Language) (string, error) {
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(file.Filename)), ".")
	want := strings.TrimPrefix(strings.ToLower(lang.Extension), ".")
	if want == "" || ext != want {
		return "", submissionFileError(fmt.Sprintf("wrong file type for language %s: expected .%s, got .%s", lang.Name, want, ext))
	}
	if _, ok := submissionFileContentTypes[ext]; !ok {
		return "", submissionFileError(fmt.Sprintf("files of type .%s cannot be submitted", ext))
	}
	if lang.FileSizeLimit <= 0 {
		return "", submissionFileError("file submissions are not enabled for this language")
	}
	if file.Size <= 0 {
		return "", submissionFileError("the submitted file is empty")
	}
	if file.Size > int64(lang.FileSizeLimit)<<20 {
		return "", submissionFileError(fmt.Sprintf("the submitted file is too large: the limit is %d MB", lang.FileSizeLimit))
	}

	random := make([]byte, 8)
	if _, err := rand.Read(random); err != nil {
		return "", err
	}
	name := hex.EncodeToString(random) + "." + ext

	if err := os.MkdirAll(submissionFileDir, 0o755); err != nil {
		return "", err
	}
	if err := c.SaveUploadedFile(file, filepath.Join(submissionFileDir, name)); err != nil {
		return "", err
	}
	return publicSubmissionFilePath(name), nil
}

// SubmissionFileServe - GET /api/submission-file/:name
// Public read-only serving of uploaded submission files. The judge agent
// downloads through this route; browsers only ever get the file as an
// attachment.
func SubmissionFileServe(c *gin.Context) {
	name := c.Param("name")
	if !submissionFileName.MatchString(name) {
		c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		return
	}
	ext := strings.TrimPrefix(filepath.Ext(name), ".")
	contentType, ok := submissionFileContentTypes[ext]
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		return
	}

	path := filepath.Join(submissionFileDir, name)
	if info, err := os.Stat(path); err != nil || info.IsDir() {
		c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		return
	}

	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", `attachment; filename="submission.`+ext+`"`)
	c.Header("Cache-Control", "public, max-age=31536000, immutable")
	c.Header("X-Content-Type-Options", "nosniff")
	c.File(path)
}
