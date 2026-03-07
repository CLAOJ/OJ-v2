package v2

import (
	"net/http"
	"strconv"

	"github.com/CLAOJ/claoj-go/service/language"
	"github.com/gin-gonic/gin"
)

// ============================================================
// ADMIN LANGUAGE MANAGEMENT API
// ============================================================

// AdminLanguageList - GET /api/v2/admin/languages
// List all languages
func AdminLanguageList(c *gin.Context) {
	page, pageSize := parsePagination(c)

	resp, err := getLanguageService().ListLanguages(language.ListLanguagesRequest{
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	type LanguageItem struct {
		ID               uint    `json:"id"`
		Key              string  `json:"key"`
		Name             string  `json:"name"`
		ShortName        *string `json:"short_name"`
		CommonName       string  `json:"common_name"`
		Ace              string  `json:"ace"`
		Pygments         string  `json:"pygments"`
		Extension        string  `json:"extension"`
		FileOnly         bool    `json:"file_only"`
		FileSizeLimit    int     `json:"file_size_limit"`
		IncludeInProblem bool    `json:"include_in_problem"`
		Info             string  `json:"info"`
	}

	items := make([]LanguageItem, len(resp.Languages))
	for i, lang := range resp.Languages {
		items[i] = LanguageItem{
			ID:               lang.ID,
			Key:              lang.Key,
			Name:             lang.Name,
			ShortName:        lang.ShortName,
			CommonName:       lang.CommonName,
			Ace:              lang.Ace,
			Pygments:         lang.Pygments,
			Extension:        lang.Extension,
			FileOnly:         lang.FileOnly,
			FileSizeLimit:    lang.FileSizeLimit,
			IncludeInProblem: lang.IncludeInProblem,
			Info:             lang.Info,
		}
	}

	c.JSON(http.StatusOK, apiListWithTotal(items, resp.Total))
}

// AdminLanguageDetail - GET /api/v2/admin/language/:id
// Get language detail
func AdminLanguageDetail(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid language id"))
		return
	}

	lang, err := getLanguageService().GetLanguage(language.GetLanguageRequest{
		LanguageID: uint(id),
	})
	if err != nil {
		if err == language.ErrLanguageNotFound {
			c.JSON(http.StatusNotFound, apiError("language not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":                 lang.ID,
		"key":                lang.Key,
		"name":               lang.Name,
		"short_name":         lang.ShortName,
		"common_name":        lang.CommonName,
		"ace":                lang.Ace,
		"pygments":           lang.Pygments,
		"template":           lang.Template,
		"description":        lang.Description,
		"extension":          lang.Extension,
		"file_only":          lang.FileOnly,
		"file_size_limit":    lang.FileSizeLimit,
		"include_in_problem": lang.IncludeInProblem,
		"info":               lang.Info,
	})
}

// AdminLanguageCreateRequest - POST /api/v2/admin/languages
type AdminLanguageCreateRequest struct {
	Key              string  `json:"key" binding:"required"`
	Name             string  `json:"name" binding:"required"`
	ShortName        *string `json:"short_name"`
	CommonName       string  `json:"common_name" binding:"required"`
	Ace              string  `json:"ace" binding:"required"`
	Pygments         string  `json:"pygments" binding:"required"`
	Template         string  `json:"template"`
	Description      string  `json:"description"`
	Extension        string  `json:"extension" binding:"required"`
	FileOnly         bool    `json:"file_only"`
	FileSizeLimit    int     `json:"file_size_limit"`
	IncludeInProblem bool    `json:"include_in_problem"`
	Info             string  `json:"info"`
}

// AdminLanguageCreate - POST /api/v2/admin/languages
// Create a new language
func AdminLanguageCreate(c *gin.Context) {
	user, _, ok := resolveUserProfile(c)
	if !ok {
		return
	}

	if !user.IsSuperuser {
		c.JSON(http.StatusForbidden, apiError("admin access required"))
		return
	}

	var req AdminLanguageCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	lang, err := getLanguageService().CreateLanguage(language.CreateLanguageRequest{
		Key:              req.Key,
		Name:             req.Name,
		ShortName:        req.ShortName,
		CommonName:       req.CommonName,
		Ace:              req.Ace,
		Pygments:         req.Pygments,
		Template:         req.Template,
		Description:      req.Description,
		Extension:        req.Extension,
		FileOnly:         req.FileOnly,
		FileSizeLimit:    req.FileSizeLimit,
		IncludeInProblem: req.IncludeInProblem,
		Info:             req.Info,
	})
	if err != nil {
		if err == language.ErrLanguageKeyExists {
			c.JSON(http.StatusBadRequest, apiError("language key already exists"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":  "language created",
		"language": gin.H{"id": lang.ID, "key": lang.Key},
	})
}

// AdminLanguageUpdateRequest - PATCH /api/v2/admin/language/:id
type AdminLanguageUpdateRequest struct {
	Name             *string `json:"name"`
	ShortName        *string `json:"short_name"`
	CommonName       *string `json:"common_name"`
	Ace              *string `json:"ace"`
	Pygments         *string `json:"pygments"`
	Template         *string `json:"template"`
	Description      *string `json:"description"`
	Extension        *string `json:"extension"`
	FileOnly         *bool   `json:"file_only"`
	FileSizeLimit    *int    `json:"file_size_limit"`
	IncludeInProblem *bool   `json:"include_in_problem"`
	Info             *string `json:"info"`
}

// AdminLanguageUpdate - PATCH /api/v2/admin/language/:id
// Update a language
func AdminLanguageUpdate(c *gin.Context) {
	user, _, ok := resolveUserProfile(c)
	if !ok {
		return
	}

	if !user.IsSuperuser {
		c.JSON(http.StatusForbidden, apiError("admin access required"))
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid language id"))
		return
	}

	var req AdminLanguageUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	lang, err := getLanguageService().UpdateLanguage(language.UpdateLanguageRequest{
		LanguageID:       uint(id),
		Name:             req.Name,
		ShortName:        req.ShortName,
		CommonName:       req.CommonName,
		Ace:              req.Ace,
		Pygments:         req.Pygments,
		Template:         req.Template,
		Description:      req.Description,
		Extension:        req.Extension,
		FileOnly:         req.FileOnly,
		FileSizeLimit:    req.FileSizeLimit,
		IncludeInProblem: req.IncludeInProblem,
		Info:             req.Info,
	})
	if err != nil {
		if err == language.ErrLanguageNotFound {
			c.JSON(http.StatusNotFound, apiError("language not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "language updated",
		"language": gin.H{"id": lang.ID, "key": lang.Key},
	})
}

// AdminLanguageDelete - DELETE /api/v2/admin/language/:id
// Delete a language
func AdminLanguageDelete(c *gin.Context) {
	user, _, ok := resolveUserProfile(c)
	if !ok {
		return
	}

	if !user.IsSuperuser {
		c.JSON(http.StatusForbidden, apiError("admin access required"))
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid language id"))
		return
	}

	if err := getLanguageService().DeleteLanguage(language.DeleteLanguageRequest{
		LanguageID: uint(id),
	}); err != nil {
		if err == language.ErrLanguageNotFound {
			c.JSON(http.StatusNotFound, apiError("language not found"))
			return
		}
		if err == language.ErrLanguageInUse {
			c.JSON(http.StatusBadRequest, apiError("cannot delete language with existing submissions"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "language deleted",
	})
}
