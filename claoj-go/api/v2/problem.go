package v2

import (
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/CLAOJ/claoj-go/db"
	"github.com/CLAOJ/claoj-go/models"
	"github.com/gin-gonic/gin"
)

func ProblemList(c *gin.Context) {
	page, pageSize := parsePagination(c)
	search := c.Query("search")
	group := c.Query("group")
	pointsMin := c.Query("points_min")
	pointsMax := c.Query("points_max")
	status := c.Query("status") // solved, unsolved
	sortField := c.DefaultQuery("sort", "code")
	order := c.DefaultQuery("order", "asc")

	userID := uint(0)
	if uid, exists := c.Get("user_id"); exists {
		userID = uid.(uint)
	}

	q := db.DB.Preload("Group").
		Select("judge_problem.id, judge_problem.code, judge_problem.name, judge_problem.points, judge_problem.partial, judge_problem.is_public, judge_problem.user_count, judge_problem.ac_rate, judge_problem.group_id, judge_problem.date").
		Where("is_public = ?", true)

	if search != "" {
		q = q.Where("code LIKE ? OR name LIKE ?", "%"+search+"%", "%"+search+"%")
	}
	if group != "" {
		q = q.Joins("JOIN judge_problemgroup ON judge_problemgroup.id = judge_problem.group_id").
			Where("judge_problemgroup.full_name LIKE ?", "%"+group+"%")
	}
	if pointsMin != "" {
		q = q.Where("points >= ?", pointsMin)
	}
	if pointsMax != "" {
		q = q.Where("points <= ?", pointsMax)
	}

	if status == "solved" && userID > 0 {
		q = q.Where("judge_problem.id IN (SELECT problem_id FROM judge_submission WHERE user_id = ? AND result = 'AC')", userID)
	} else if status == "unsolved" && userID > 0 {
		q = q.Where("judge_problem.id NOT IN (SELECT problem_id FROM judge_submission WHERE user_id = ? AND result = 'AC')", userID)
	}

	// Validate sort field
	validSorts := map[string]string{
		"code":       "code",
		"name":       "name",
		"points":     "points",
		"ac_rate":    "ac_rate",
		"user_count": "user_count",
	}
	dbField, ok := validSorts[sortField]
	if !ok {
		dbField = "code"
	}
	if order != "asc" && order != "desc" {
		order = "asc"
	}

	var problems []models.Problem
	q = q.Order("judge_problem." + dbField + " " + order).
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&problems)

	if q.Error != nil {
		c.JSON(http.StatusInternalServerError, apiError(q.Error.Error()))
		return
	}

	// Get solved problems for the current user
	solvedIDs := make(map[uint]bool)
	if userID > 0 {
		var solved []uint
		db.DB.Table("judge_submission").
			Where("user_id = ? AND result = 'AC'", userID).
			Distinct("problem_id").
			Pluck("problem_id", &solved)
		for _, id := range solved {
			solvedIDs[id] = true
		}
	}

	type ProblemItem struct {
		Code      string  `json:"code"`
		Name      string  `json:"name"`
		Points    float64 `json:"points"`
		Partial   bool    `json:"partial"`
		UserCount int     `json:"user_count"`
		AcRate    float64 `json:"ac_rate"`
		Group     string  `json:"group"`
		IsSolved  bool    `json:"is_solved"`
	}

	items := make([]ProblemItem, len(problems))
	for i, p := range problems {
		items[i] = ProblemItem{
			Code:      p.Code,
			Name:      p.Name,
			Points:    p.Points,
			Partial:   p.Partial,
			UserCount: p.UserCount,
			AcRate:    p.AcRate,
			Group:     p.Group.FullName,
			IsSolved:  solvedIDs[p.ID],
		}
	}
	c.JSON(http.StatusOK, apiList(items))
}

// RandomProblem – GET /api/v2/problems/random
func RandomProblem(c *gin.Context) {
	var p models.Problem
	if err := db.DB.Where("is_public = ?", true).Order("RAND()").First(&p).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError("failed to find a random problem"))
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": p.Code})
}

// ProblemDetail – GET /api/v2/problem/:code
func ProblemDetail(c *gin.Context) {
	code := c.Param("code")
	var p models.Problem
	if err := db.DB.
		Preload("Types").
		Preload("Group").
		Preload("AllowedLangs").
		Preload("Authors.User").
		Where("code = ? AND is_public = ?", code, true).
		First(&p).Error; err != nil {
		c.JSON(http.StatusNotFound, apiError("problem not found"))
		return
	}

	type LangItem struct{ Key, Name string }
	langs := make([]LangItem, len(p.AllowedLangs))
	for i, l := range p.AllowedLangs {
		langs[i] = LangItem{l.Key, l.Name}
	}

	type TypeItem struct{ Name string }
	types := make([]TypeItem, len(p.Types))
	for i, t := range p.Types {
		types[i] = TypeItem{t.FullName}
	}

	type AuthorItem struct{ Username string }
	authors := make([]AuthorItem, len(p.Authors))
	for i, a := range p.Authors {
		authors[i] = AuthorItem{a.User.Username}
	}

	isSolved := false
	isAttempted := false
	if uid, exists := c.Get("user_id"); exists {
		userID := uid.(uint)
		var result string
		err := db.DB.Table("judge_submission").
			Select("result").
			Where("user_id = ? AND problem_id = ?", userID, p.ID).
			Order("points DESC, date DESC").
			Limit(1).
			Pluck("result", &result).Error
		if err == nil && result != "" {
			isAttempted = true
			if result == "AC" {
				isSolved = true
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"id":             p.ID,
		"code":           p.Code,
		"name":           p.Name,
		"description":    p.Description,
		"is_full_markup": p.IsFullMarkup,
		"points":         p.Points,
		"partial":        p.Partial,
		"time_limit":     p.TimeLimit,
		"memory_limit":   p.MemoryLimit,
		"group":          p.Group.FullName,
		"languages":      langs,
		"types":          types,
		"authors":        authors,
		"user_count":     p.UserCount,
		"ac_rate":        p.AcRate,
		"date":           p.Date,
		"is_solved":      isSolved,
		"is_attempted":   isAttempted,
		"pdf_url":        p.PdfURL,
	})
}

// ProblemStatementPDF - GET /api/v2/problem/:code/pdf
// Serves the PDF statement file for a problem
func ProblemStatementPDF(c *gin.Context) {
	code := c.Param("code")

	var problem models.Problem
	if err := db.DB.Where("code = ?", code).First(&problem).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "problem not found"})
		return
	}

	// Check if problem has a PDF URL configured
	if problem.PdfURL == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "no PDF statement available"})
		return
	}

	// Check if problem is public or user has access
	if !problem.IsPublic {
		userID := uint(0)
		if uid, exists := c.Get("user_id"); exists {
			userID = uid.(uint)
		}
		// Check if user is author, curator, or staff
		var count int64
		db.DB.Table("judge_problem_authors").Where("problem_id = ? AND profile_id IN (SELECT id FROM judge_profile WHERE user_id = ?)", problem.ID, userID).Count(&count)
		if count == 0 {
			db.DB.Table("judge_problem_curators").Where("problem_id = ? AND profile_id IN (SELECT id FROM judge_profile WHERE user_id = ?)", problem.ID, userID).Count(&count)
		}
		if count == 0 {
			var user models.Profile
			if err := db.DB.Preload("User").Where("user_id = ?", userID).First(&user).Error; err != nil || !user.User.IsStaff {
				c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
				return
			}
		}
	}

	// Determine PDF file path
	// pdf_url can be either an absolute path or relative to problem data directory
	pdfPath := problem.PdfURL
	if !filepath.IsAbs(pdfPath) {
		pdfPath = filepath.Join("data", "problems", code, pdfPath)
	}

	// Security: ensure path is within data directory
	cleanPath := filepath.Clean(pdfPath)
	dataPrefix := filepath.Clean("data")
	if !strings.HasPrefix(cleanPath, dataPrefix) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid PDF path"})
		return
	}

	// Check if file exists
	fileInfo, err := os.Stat(cleanPath)
	if err != nil {
		if os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "PDF file not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to access PDF file"})
		}
		return
	}

	// Check file size (limit to 10MB)
	if fileInfo.Size() > 10*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "PDF file too large"})
		return
	}

	// Read and serve file
	content, err := os.ReadFile(cleanPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read PDF file"})
		return
	}

	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", "inline; filename=\""+filepath.Base(cleanPath)+"\"")
	c.Header("Content-Length", strconv.Itoa(len(content)))
	c.Data(http.StatusOK, "application/pdf", content)
}
