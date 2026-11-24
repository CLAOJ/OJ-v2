package v2

import (
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// problemTranslationLanguages mirrors dmoj/settings.py LANGUAGES: ProblemTranslation
// .language has choices=settings.LANGUAGES and unique_together = (problem, language).
var problemTranslationLanguages = map[string]bool{"en": true, "vi": true}

type problemTranslationItem struct {
	ID          uint   `json:"id"`
	ProblemID   uint   `json:"problem_id"`
	Language    string `json:"language"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

func newProblemTranslationItem(t models.ProblemTranslation) problemTranslationItem {
	return problemTranslationItem{
		ID:          t.ID,
		ProblemID:   t.ProblemID,
		Language:    t.Language,
		Name:        t.Name,
		Description: t.Description,
	}
}

type problemTranslationRequest struct {
	Language    string `json:"language"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// validate returns a user-facing message, or "" when the request matches the
// v1 model constraints (language choice, non-blank name <= 100, non-blank description).
func (r *problemTranslationRequest) validate() string {
	r.Language = strings.TrimSpace(r.Language)
	r.Name = strings.TrimSpace(r.Name)
	if !problemTranslationLanguages[r.Language] {
		return "language must be one of: en, vi"
	}
	if r.Name == "" {
		return "name is required"
	}
	if utf8.RuneCountInString(r.Name) > 100 {
		return "name must be at most 100 characters"
	}
	if strings.TrimSpace(r.Description) == "" {
		return "description is required"
	}
	return ""
}

// translationProblemID resolves :code; the route middleware already checked
// existence and edit permission.
func translationProblemID(c *gin.Context) (uint, bool) {
	var problem models.Problem
	if err := db.DB.Select("id").Where("code = ?", c.Param("code")).First(&problem).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, apiUserError("problem not found"))
			return 0, false
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return 0, false
	}
	return problem.ID, true
}

func translationLanguageTaken(problemID uint, language string, exceptID uint) bool {
	var n int64
	db.DB.Model(&models.ProblemTranslation{}).
		Where("problem_id = ? AND language = ? AND id <> ?", problemID, language, exceptID).
		Count(&n)
	return n > 0
}

// ProblemTranslationList – GET /api/v2/admin/problem/:code/translations
func ProblemTranslationList(c *gin.Context) {
	problemID, ok := translationProblemID(c)
	if !ok {
		return
	}

	var translations []models.ProblemTranslation
	if err := db.DB.Where("problem_id = ?", problemID).Order("language").Find(&translations).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	items := make([]problemTranslationItem, len(translations))
	for i, tr := range translations {
		items[i] = newProblemTranslationItem(tr)
	}

	c.JSON(http.StatusOK, apiList(items))
}

// ProblemTranslationCreate – POST /api/v2/admin/problem/:code/translations
func ProblemTranslationCreate(c *gin.Context) {
	problemID, ok := translationProblemID(c)
	if !ok {
		return
	}

	var req problemTranslationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiUserError("invalid request body"))
		return
	}
	if msg := req.validate(); msg != "" {
		c.JSON(http.StatusBadRequest, apiUserError(msg))
		return
	}
	// v1 keeps one translation per (problem, language): the admin inline has no
	// way to add a second row for the same language.
	if translationLanguageTaken(problemID, req.Language, 0) {
		c.JSON(http.StatusConflict, apiUserError("a translation for this language already exists"))
		return
	}

	translation := models.ProblemTranslation{
		ProblemID:   problemID,
		Language:    req.Language,
		Name:        req.Name,
		Description: req.Description,
	}
	if err := db.DB.Omit("Problem").Create(&translation).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusCreated, newProblemTranslationItem(translation))
}

// ProblemTranslationUpdate – PATCH /api/v2/admin/problem/:code/translations/:id
func ProblemTranslationUpdate(c *gin.Context) {
	problemID, ok := translationProblemID(c)
	if !ok {
		return
	}
	id, ok := numericParam(c, "id")
	if !ok {
		return
	}

	var translation models.ProblemTranslation
	if err := db.DB.Where("id = ? AND problem_id = ?", id, problemID).First(&translation).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, apiUserError("translation not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	var req problemTranslationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiUserError("invalid request body"))
		return
	}
	if msg := req.validate(); msg != "" {
		c.JSON(http.StatusBadRequest, apiUserError(msg))
		return
	}
	if translationLanguageTaken(problemID, req.Language, translation.ID) {
		c.JSON(http.StatusConflict, apiUserError("a translation for this language already exists"))
		return
	}

	translation.Language = req.Language
	translation.Name = req.Name
	translation.Description = req.Description
	if err := db.DB.Model(&translation).
		Select("language", "name", "description").
		Updates(&translation).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, newProblemTranslationItem(translation))
}

// ProblemTranslationDelete – DELETE /api/v2/admin/problem/:code/translations/:id
func ProblemTranslationDelete(c *gin.Context) {
	problemID, ok := translationProblemID(c)
	if !ok {
		return
	}
	id, ok := numericParam(c, "id")
	if !ok {
		return
	}

	res := db.DB.Where("id = ? AND problem_id = ?", id, problemID).Delete(&models.ProblemTranslation{})
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, apiError(res.Error.Error()))
		return
	}
	if res.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, apiUserError("translation not found"))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "translation deleted"})
}