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
// extension is any short alphanumeric token — uploads validate it against the
// language's configured extension, and serving pins the Content-Type itself,
// so the set of servable extensions is not a closed list.
var submissionFileName = regexp.MustCompile(`^[0-9a-f]{16}\.[a-z0-9]{1,16}$`)

// submissionFileExtension is the shape a language-configured extension must
// have to become part of a stored file name.
var submissionFileExtension = regexp.MustCompile(`^[a-z0-9]{1,16}$`)

// submissionFileContentTypes pins the Content-Type for known extensions; the
// extension is part of the name the server generated, never a browser-supplied
// header. Anything unlisted is served as application/octet-stream. User files
// are always attachments — serving an uploaded .html/.svg inline from our own
// origin would be stored XSS — and every response carries nosniff.
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
	"html":  textPlain,
	"css":   textPlain,
	"xml":   textPlain,
	"svg":   textPlain,
	"ino":   textPlain,
	"asm":   textPlain,
	"s":     textPlain,
}

const textPlain = "text/plain; charset=utf-8"

// submissionFileServeType is the forced Content-Type for a stored file.
func submissionFileServeType(ext string) string {
	if ct, ok := submissionFileContentTypes[ext]; ok {
		return ct
	}
	return "application/octet-stream"
}

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
// language.file_size_limit (MB). The language configuration is the only
// extension gate — inventing a second list here would silently reject a
// legitimately configured .sb3/.ino/.asm upload.
func saveSubmissionFile(c *gin.Context, file *multipart.FileHeader, lang *models.Language) (string, error) {
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(file.Filename)), ".")
	want := strings.TrimPrefix(strings.ToLower(lang.Extension), ".")
	if want == "" || ext != want {
		return "", submissionFileError(fmt.Sprintf("wrong file type for language %s: expected .%s, got .%s", lang.Name, want, ext))
	}
	if !submissionFileExtension.MatchString(ext) {
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
// attachment with a Content-Type this server picked.
func SubmissionFileServe(c *gin.Context) {
	name := c.Param("name")
	if !submissionFileName.MatchString(name) {
		c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		return
	}
	ext := strings.TrimPrefix(filepath.Ext(name), ".")

	path := filepath.Join(submissionFileDir, name)
	if info, err := os.Stat(path); err != nil || info.IsDir() {
		c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		return
	}

	c.Header("Content-Type", submissionFileServeType(ext))
	c.Header("Content-Disposition", `attachment; filename="submission.`+ext+`"`)
	c.Header("Cache-Control", "public, max-age=31536000, immutable")
	c.Header("X-Content-Type-Options", "nosniff")
	c.File(path)
}
