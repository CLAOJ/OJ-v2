package v2

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/CLAOJ/claoj/auth"
	"github.com/CLAOJ/claoj/config"
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
)

// ProblemList – GET /api/v2/problems
// @Description List all public problems with pagination, filtering, and sorting.
// @Tags Problems
// @Summary List problems
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Items per page" default(20)
// @Param search query string false "Search by problem code or name"
// @Param group query string false "Filter by problem group"
// @Param points_min query float64 false "Minimum points"
// @Param points_max query float64 false "Maximum points"
// @Param status query string false "Filter by status: solved, unsolved (requires login)"
// @Param sort query string false "Sort field: code, name, points, ac_rate, user_count" default(code)
// @Param order query string false "Sort order: asc, desc" default(asc)
// @Success 200 {object} map[string]interface{} "Paginated list of problems"
// @Router /problems [get]
func ProblemList(c *gin.Context) {
	page, pageSize := parsePagination(c)
	search := c.Query("search")
	group := c.Query("group")
	pointsMin := c.Query("points_min")
	pointsMax := c.Query("points_max")
	status := c.Query("status") // solved, unsolved
	sortField := c.DefaultQuery("sort", "code")
	order := c.DefaultQuery("order", "asc")

	// judge_submission.user_id is a judge_profile.id FK — resolve the
	// request user's profile id, not the auth_user id from the context.
	profileID := uint(0)
	if pid, ok := auth.CurrentProfileID(c); ok {
		profileID = pid
	}

	// Restrict to problems the requesting user may see. Without this,
	// organization-private problems (is_public + is_organization_private) leak
	// to every user (Django parity: Problem.get_visible_problems).
	visExpr, visArgs := auth.VisibleProblemFilter(c)
	q := db.DB.Preload("Group").
		Select("judge_problem.id, judge_problem.code, judge_problem.name, judge_problem.points, judge_problem.partial, judge_problem.is_public, judge_problem.user_count, judge_problem.ac_rate, judge_problem.group_id, judge_problem.date").
		Where(visExpr, visArgs...)

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

	if status == "solved" && profileID > 0 {
		q = q.Where("judge_problem.id IN (SELECT problem_id FROM judge_submission WHERE user_id = ? AND result = 'AC')", profileID)
	} else if status == "unsolved" && profileID > 0 {
		q = q.Where("judge_problem.id NOT IN (SELECT problem_id FROM judge_submission WHERE user_id = ? AND result = 'AC')", profileID)
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
	if profileID > 0 {
		var solved []uint
		db.DB.Table("judge_submission").
			Where("user_id = ? AND result = 'AC'", profileID).
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
	visExpr, visArgs := auth.VisibleProblemFilter(c)
	if err := db.DB.Where(visExpr, visArgs...).Order("RAND()").First(&p).Error; err != nil {
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
		Preload("Curators").
		Preload("Testers").
		Where("code = ?", code).
		First(&p).Error; err != nil {
		c.JSON(http.StatusNotFound, apiError("problem not found"))
		return
	}
	// Django parity (Problem.is_accessible_by): hidden problems stay 404
	// unless the viewer is privileged (superuser, see_private_problem,
	// editor, or tester).
	if !auth.CanViewProblem(c, &p) {
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
	// judge_submission.user_id is a judge_profile.id FK
	if profileID, ok := auth.CurrentProfileID(c); ok {
		var result string
		err := db.DB.Table("judge_submission").
			Select("result").
			Where("user_id = ? AND problem_id = ?", profileID, p.ID).
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

var (
	errPDFMediaUnavailable = errors.New("pdf media root not configured")
	errPDFInvalidPath      = errors.New("invalid PDF path")
)

// resolveStatementPDFPath maps a problem's stored pdf_url to an on-disk file path.
//
// Two shapes are supported:
//   - v2-native: a bare filename stored under the problem's data directory,
//     e.g. "statement.pdf" -> data/problems/<code>/statement.pdf
//   - v1-migrated: a site-relative media path served by the v1 stack,
//     e.g. "/pdf/<uuid>.pdf" -> <v1MediaRoot>/pdf/<uuid>.pdf
//
// v1MediaRoot is the read-only mount of the v1 Django media directory
// (config app.v1_media_root / env V1_MEDIA_ROOT). A v1-style path requested
// with no media root configured yields errPDFMediaUnavailable (the caller 404s).
// Any resolved path escaping its intended root yields errPDFInvalidPath.
func resolveStatementPDFPath(pdfURL, code, v1MediaRoot string) (string, error) {
	if strings.HasPrefix(pdfURL, "/") {
		if v1MediaRoot == "" {
			return "", errPDFMediaUnavailable
		}
		root := filepath.Clean(v1MediaRoot)
		clean := filepath.Clean(filepath.Join(root, filepath.FromSlash(pdfURL)))
		if clean != root && !strings.HasPrefix(clean, root+string(os.PathSeparator)) {
			return "", errPDFInvalidPath
		}
		return clean, nil
	}
	base := filepath.Clean(filepath.Join("data", "problems", code))
	clean := filepath.Clean(filepath.Join(base, pdfURL))
	if clean != base && !strings.HasPrefix(clean, base+string(os.PathSeparator)) {
		return "", errPDFInvalidPath
	}
	return clean, nil
}

// ProblemStatementPDF - GET /api/v2/problem/:code/pdf
// Serves the PDF statement file for a problem
func ProblemStatementPDF(c *gin.Context) {
	code := c.Param("code")

	var problem models.Problem
	if err := db.DB.Preload("Authors").Preload("Curators").Preload("Testers").Where("code = ?", code).First(&problem).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "problem not found"})
		return
	}

	// Check if problem has a PDF URL configured
	if problem.PdfURL == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "no PDF statement available"})
		return
	}

	// Check if problem is public or the current user may view the hidden problem
	// (Django parity: Problem.is_accessible_by).
	if !auth.CanViewProblem(c, &problem) {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	// Resolve the stored pdf_url to an on-disk file. v2-native problems store a
	// bare filename under data/problems/<code>/; v1-migrated problems store a
	// site-relative media path like /pdf/<uuid>.pdf served from the v1 Django
	// media directory (mounted read-only at config app.v1_media_root).
	cleanPath, err := resolveStatementPDFPath(problem.PdfURL, code, config.C.App.V1MediaRoot)
	if err != nil {
		if errors.Is(err, errPDFMediaUnavailable) {
			c.JSON(http.StatusNotFound, gin.H{"error": "PDF statement not available"})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid PDF path"})
		}
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
