package v2

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/CLAOJ/claoj/auth"
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/CLAOJ/claoj/storage"
	"github.com/gin-gonic/gin"
)

// Direct browser uploads to R2. The browser asks for a signed PUT URL into
// tmp/<purpose>/<profile>/, sends the file straight to R2, then calls the
// purpose's commit endpoint. Commit re-checks the stored object (size, and
// type sniffed from its first bytes; the browser's claims are not trusted)
// and moves it to its final key. R2 has no signed POST policy, so this check
// is what enforces the limits. Without R2 the sign endpoints answer 501 and
// the client falls back to the multipart endpoints.

type uploadPurpose struct {
	name     string
	bucket   storage.Bucket
	maxBytes int64
	types    map[string]string // sniffed MIME type -> extension
	prefix   string            // final key prefix
}

var (
	purposeImage  = uploadPurpose{"image", storage.Public, maxMarkdownImageBytes, markdownImageExts, "media/"}
	purposeAvatar = uploadPurpose{"avatar", storage.Public, 1 << 20, markdownImageExts, "avatars/"}
	purposePDF    = uploadPurpose{"pdf", storage.Private, 20 << 20, map[string]string{"application/pdf": ".pdf"}, ""}
)

func (p uploadPurpose) tmpPrefix(profileID uint) string {
	return storage.TmpPrefix + p.name + "/" + strconv.FormatUint(uint64(profileID), 10) + "/"
}

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// signUpload answers {upload_url, key, content_type} for one file.
func signUpload(c *gin.Context, p uploadPurpose) {
	if !p.bucket.Enabled() {
		c.JSON(http.StatusNotImplemented, apiUserError("direct upload is not configured"))
		return
	}
	profileID, ok := auth.CurrentProfileID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, apiUserError("login required"))
		return
	}
	var req struct {
		ContentType string `json:"content_type"`
		Size        int64  `json:"size"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiUserError("invalid request"))
		return
	}
	ext, allowed := p.types[req.ContentType]
	if !allowed {
		c.JSON(http.StatusBadRequest, apiUserError("file type not allowed"))
		return
	}
	if req.Size <= 0 || req.Size > p.maxBytes {
		c.JSON(http.StatusBadRequest, apiUserError("file is too large (max "+strconv.FormatInt(p.maxBytes>>20, 10)+" MB)"))
		return
	}
	id, err := randomHex(12)
	if err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}
	key := p.tmpPrefix(profileID) + id + ext
	url, err := storage.PresignPut(c.Request.Context(), p.bucket, key, req.ContentType)
	if err != nil {
		c.JSON(http.StatusBadGateway, apiUserError("could not sign upload"))
		return
	}
	c.JSON(http.StatusOK, gin.H{"upload_url": url, "key": key, "content_type": req.ContentType})
}

// commitUpload verifies the tmp object named in the JSON body and moves it
// to <prefix><finalName><ext>. It writes the error response itself and
// returns ok=false on failure; on success it returns the final key.
func commitUpload(c *gin.Context, p uploadPurpose, finalName string) (string, bool) {
	profileID, ok := auth.CurrentProfileID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, apiUserError("login required"))
		return "", false
	}
	var req struct {
		Key string `json:"key"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || !strings.HasPrefix(req.Key, p.tmpPrefix(profileID)) || strings.Contains(req.Key, "..") {
		c.JSON(http.StatusBadRequest, apiUserError("invalid upload key"))
		return "", false
	}
	ctx := c.Request.Context()
	size, head, err := storage.Inspect(ctx, p.bucket, req.Key, 512)
	if err != nil {
		c.JSON(http.StatusBadRequest, apiUserError("upload not found; please try again"))
		return "", false
	}
	mime := http.DetectContentType(head)
	ext, allowed := p.types[mime]
	if !allowed || size > p.maxBytes || size == 0 {
		_ = storage.Delete(ctx, p.bucket, req.Key)
		c.JSON(http.StatusBadRequest, apiUserError("file is not an allowed type or is too large"))
		return "", false
	}
	final := p.prefix + finalName + ext
	if err := storage.Promote(ctx, p.bucket, req.Key, final, mime); err != nil {
		c.JSON(http.StatusBadGateway, apiUserError("could not store upload"))
		return "", false
	}
	return final, true
}

// AdminUploadSign - POST /api/admin/uploads/sign?purpose=image|pdf[&problem=<code>] {content_type, size}
func AdminUploadSign(c *gin.Context) {
	switch c.Query("purpose") {
	case "image":
		signUpload(c, purposeImage)
	case "pdf":
		var problem models.Problem
		if err := db.DB.Preload("Authors").Preload("Curators").Where("code = ?", c.Query("problem")).First(&problem).Error; err != nil {
			c.JSON(http.StatusNotFound, apiUserError("problem not found"))
			return
		}
		if !auth.CanEditProblem(c, &problem) {
			c.JSON(http.StatusForbidden, apiUserError("you do not have permission to edit this problem"))
			return
		}
		signUpload(c, purposePDF)
	default:
		c.JSON(http.StatusBadRequest, apiUserError("unknown upload purpose"))
	}
}

// AdminMarkdownImageCommit - POST /api/admin/markdown/image/commit {key}
func AdminMarkdownImageCommit(c *gin.Context) {
	name, err := randomHex(8)
	if err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}
	if key, ok := commitUpload(c, purposeImage, name); ok {
		c.JSON(http.StatusOK, gin.H{"url": storage.PublicURL(key)})
	}
}

// AdminProblemPdfCommit - POST /api/admin/problem/:code/pdf/commit {key}
// (behind RequireProblemEdit). Stores the PDF as problems/<code>/statement-<id>.pdf
// and points pdf_url at it; a fresh name per upload means no stale copies.
func AdminProblemPdfCommit(c *gin.Context) {
	code := c.Param("code")
	var problem models.Problem
	if err := db.DB.Where("code = ?", code).First(&problem).Error; err != nil {
		c.JSON(http.StatusNotFound, apiUserError("problem not found"))
		return
	}
	id, err := randomHex(8)
	if err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}
	name := "statement-" + id
	key, ok := commitUpload(c, uploadPurpose{purposePDF.name, purposePDF.bucket, purposePDF.maxBytes, purposePDF.types, "problems/" + code + "/"}, name)
	if !ok {
		return
	}
	pdfURL := name + ".pdf"
	if err := db.DB.Model(&problem).Update("pdf_url", pdfURL).Error; err != nil {
		_ = storage.Delete(c.Request.Context(), storage.Private, key)
		c.JSON(http.StatusInternalServerError, apiUserError("failed to update problem"))
		return
	}
	// The previous v2 upload is now unreferenced (v1-migrated /pdf/ files are
	// left alone: v1 may still point at them while both sites run).
	if old := problem.PdfURL; old != "" && !strings.HasPrefix(old, "/") && old != pdfURL {
		_ = storage.Delete(c.Request.Context(), storage.Private, storage.PDFKey(old, code))
	}
	c.JSON(http.StatusOK, gin.H{"message": "PDF uploaded successfully", "pdf_url": pdfURL})
}

// AvatarUploadSign - POST /api/user/avatar/sign {content_type, size}
func AvatarUploadSign(c *gin.Context) { signUpload(c, purposeAvatar) }

// UserImageUploadSign - POST /api/user/uploads/image/sign {content_type, size}:
// a markdown image for any signed-in user (committed by AdminMarkdownImageCommit).
func UserImageUploadSign(c *gin.Context) { signUpload(c, purposeImage) }

// AvatarSet - PUT /api/user/avatar {key}: use an uploaded image as the avatar.
func AvatarSet(c *gin.Context) {
	profileID, _ := auth.CurrentProfileID(c)
	name, err := randomHex(8)
	if err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}
	key, ok := commitUpload(c, purposeAvatar, strconv.FormatUint(uint64(profileID), 10)+"-"+name)
	if !ok {
		return
	}
	url := storage.PublicURL(key)
	var prev models.ProfileAvatar
	hadPrev := db.DB.Where("profile_id = ?", profileID).First(&prev).Error == nil
	row := models.ProfileAvatar{ProfileID: profileID, URL: url, UpdatedAt: time.Now().UTC()}
	if err := db.DB.Save(&row).Error; err != nil {
		_ = storage.Delete(c.Request.Context(), storage.Public, key)
		c.JSON(http.StatusInternalServerError, apiUserError("failed to save avatar"))
		return
	}
	if hadPrev {
		deleteAvatarObject(c, prev.URL)
	}
	c.JSON(http.StatusOK, gin.H{"avatar_url": url})
}

// AvatarDelete - DELETE /api/user/avatar: back to the Gravatar identicon.
func AvatarDelete(c *gin.Context) {
	profileID, _ := auth.CurrentProfileID(c)
	var prev models.ProfileAvatar
	if db.DB.Where("profile_id = ?", profileID).First(&prev).Error == nil {
		if err := db.DB.Delete(&prev).Error; err != nil {
			c.JSON(http.StatusInternalServerError, apiUserError("failed to remove avatar"))
			return
		}
		deleteAvatarObject(c, prev.URL)
	}
	var profile models.Profile
	db.DB.Preload("User").First(&profile, profileID)
	c.JSON(http.StatusOK, gin.H{"avatar_url": getGravatarURL(profile.User.Email, 80)})
}

func deleteAvatarObject(c *gin.Context, url string) {
	if key, found := strings.CutPrefix(url, storage.PublicURL("")); found && strings.HasPrefix(key, purposeAvatar.prefix) {
		_ = storage.Delete(c.Request.Context(), storage.Public, key)
	}
}
