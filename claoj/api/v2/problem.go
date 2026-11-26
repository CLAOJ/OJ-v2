package v2

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/CLAOJ/claoj/auth"
	"github.com/CLAOJ/claoj/config"
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/CLAOJ/claoj/storage"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ProblemList – GET /api/v2/problems
// @Description List all public problems with pagination, filtering, and sorting.
// @Tags Problems
// @Summary List problems
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Items per page" default(20)
// @Param search query string false "Search by problem code or name"
// @Param group query string false "Filter by problem group (id or full name)"
// @Param type query []int false "Filter by problem type ids (repeated or comma-separated)"
// @Param has_editorial query string false "Filter by a public editorial: 1/true or 0/false"
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
	// Filters are applied to a bare Model query so the same statement can be
	// branched (via Session) into both a COUNT and the page fetch. Counting a
	// query that carries an explicit multi-column Select produces invalid SQL.
	q := db.DB.Model(&models.Problem{}).Where(visExpr, visArgs...)

	if search != "" {
		q = q.Where("code LIKE ? OR name LIKE ?", "%"+search+"%", "%"+search+"%")
	}
	if group != "" {
		// The list sends a group id; a name substring is still accepted for
		// bookmarked v1-style URLs.
		if gid, err := strconv.ParseUint(group, 10, 64); err == nil {
			q = q.Where("judge_problem.group_id = ?", gid)
		} else {
			q = q.Joins("JOIN judge_problemgroup ON judge_problemgroup.id = judge_problem.group_id").
				Where("judge_problemgroup.full_name LIKE ?", "%"+group+"%")
		}
	}
	// A problem matches when it carries any of the selected types. The
	// subquery keeps one row per problem, so COUNT stays correct.
	if typeIDs, requested := problemTypeIDs(c); requested {
		if len(typeIDs) == 0 {
			// v1 narrows on an all-invalid type filter; it never widens it
			// back to every problem.
			q = q.Where("1 = 0")
		} else {
			q = q.Where("judge_problem.id IN (SELECT problem_id FROM judge_problem_types WHERE problemtype_id IN ?)", typeIDs)
		}
	}
	switch c.Query("has_editorial") {
	case "1", "true":
		q = q.Where(hasPublicEditorialSQL, time.Now())
	case "0", "false":
		q = q.Where("NOT "+hasPublicEditorialSQL, time.Now())
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
		// v1's problem list opens newest first.
		"date": "date",
	}
	dbField, ok := validSorts[sortField]
	if !ok {
		dbField = "code"
	}
	if order != "asc" && order != "desc" {
		order = "asc"
	}

	var total int64
	if err := q.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	var problems []models.Problem
	if err := q.Session(&gorm.Session{}).
		Preload("Group").
		Select("judge_problem.id, judge_problem.code, judge_problem.name, judge_problem.points, judge_problem.partial, judge_problem.is_public, judge_problem.user_count, judge_problem.ac_rate, judge_problem.group_id, judge_problem.date").
		Order("judge_problem." + dbField + " " + order).
		Order("judge_problem.id " + order). // stable pages when dates tie or are NULL
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&problems).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, apiListWithTotal(problemListItems(problems, profileID), total))
}

// problemListItem is the public problem-list row shape, shared with the
// organization problems tab so both lists always look the same.
type problemListItem struct {
	ID        uint    `json:"id"`
	Code      string  `json:"code"`
	Name      string  `json:"name"`
	Points    float64 `json:"points"`
	Partial   bool    `json:"partial"`
	UserCount int     `json:"user_count"`
	AcRate    float64 `json:"ac_rate"`
	Group     string  `json:"group"`
	IsSolved  bool    `json:"is_solved"`
}

// problemListItems builds those rows, flagging what profileID already solved
// (0 = anonymous, nothing is flagged).
func problemListItems(problems []models.Problem, profileID uint) []problemListItem {
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

	items := make([]problemListItem, len(problems))
	for i, p := range problems {
		items[i] = problemListItem{
			ID:        p.ID,
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
	return items
}

// publicEditorialCondition is the v1 editorial visibility rule
// (Solution.is_accessible_by): public and already published. A NULL publish_on
// is unpublished, so it does not count. Shared with ProblemSolutionExists so
// the editorial tab and the has_editorial filter always agree.
const publicEditorialCondition = `is_public = 1 AND publish_on <= ?`

const hasPublicEditorialSQL = `EXISTS (SELECT 1 FROM judge_solution s` +
	` WHERE s.problem_id = judge_problem.id AND ` + publicEditorialCondition + `)`

// problemTypeIDs reads the `type` filter as repeated params (?type=1&type=2)
// or as one comma-separated value (?type=1,2) and reports whether the filter
// was sent at all. Bad values are dropped; a filter with no valid id stays
// requested so the caller can narrow to nothing instead of widening.
func problemTypeIDs(c *gin.Context) (ids []uint, requested bool) {
	raws := c.QueryArray("type")
	for _, raw := range raws {
		for _, part := range strings.Split(raw, ",") {
			if id, err := strconv.ParseUint(strings.TrimSpace(part), 10, 64); err == nil {
				ids = append(ids, uint(id))
			}
		}
	}
	return ids, len(raws) > 0
}

// SuggestProblemList - GET /api/v2/problems/suggest
// @Description Problems still hidden but carrying a suggester, i.e.
// @Description suggestions awaiting review (v1 SuggestList parity). Metadata
// @Description only: no statement content is returned.
// @Tags Problems
// @Summary List suggested problems
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Items per page" default(100)
// @Success 200 {object} map[string]interface{} "Paginated list of suggestions"
// @Failure 403 {object} map[string]interface{} "judge.suggest_new_problem required"
// @Router /problems/suggest [get]
func SuggestProblemList(c *gin.Context) {
	if !auth.HasPerm(c, "judge.suggest_new_problem") {
		c.JSON(http.StatusForbidden, apiUserError("you do not have permission to view problem suggestions"))
		return
	}
	page, pageSize := parsePagination(c)

	// v1: Q(is_public=False) & ~Q(suggester=None).
	q := db.DB.Model(&models.Problem{}).
		Where("is_public = ? AND suggester_id IS NOT NULL", false)

	var total int64
	if err := q.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	var problems []models.Problem
	if err := q.Session(&gorm.Session{}).
		Preload("Suggester.User").
		Select("judge_problem.id, judge_problem.code, judge_problem.name, judge_problem.points, judge_problem.date, judge_problem.suggester_id").
		Order("judge_problem.date DESC, judge_problem.id DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&problems).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	type SuggestItem struct {
		Code      string     `json:"code"`
		Name      string     `json:"name"`
		Points    float64    `json:"points"`
		Date      *time.Time `json:"date"`
		Suggester string     `json:"suggester"`
	}

	items := make([]SuggestItem, len(problems))
	for i, p := range problems {
		items[i] = SuggestItem{Code: p.Code, Name: p.Name, Points: p.Points, Date: p.Date}
		if p.Suggester != nil {
			items[i].Suggester = p.Suggester.User.Username
		}
	}
	// Per-user and permission-gated: no shared cache may keep it.
	c.Header("Cache-Control", "private, no-store")
	c.JSON(http.StatusOK, apiListWithTotal(items, total))
}

// RandomProblem – GET /api/v2/problems/random
func RandomProblem(c *gin.Context) {
	var p models.Problem
	visExpr, visArgs := auth.VisibleProblemFilter(c)
	if err := db.DB.Where(visExpr, visArgs...).Order("RAND()").First(&p).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiUserError("failed to find a random problem"))
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": p.Code})
}

// Nested DTOs for the problem detail payload. The json tags are load-bearing:
// claoj-web reads these as lowercase (`languages: {key, name}[]`, `types:
// {name}[]`, `authors: {username}[]` in claoj-web/src/types/index.ts). Without
// tags, encoding/json emits the Go field names and every field reads as
// undefined on the client — which blanked out the submit-language <select>.
type problemLangItem struct {
	Key           string `json:"key"`
	Name          string `json:"name"`
	Extension     string `json:"extension"`
	FileOnly      bool   `json:"file_only"`
	FileSizeLimit int    `json:"file_size_limit"`
}

type problemTypeItem struct {
	Name string `json:"name"`
}

type problemAuthorItem struct {
	Username string `json:"username"`
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
		Preload("License").
		Where("code = ?", code).
		First(&p).Error; err != nil {
		c.JSON(http.StatusNotFound, apiUserError("problem not found"))
		return
	}
	// Django parity (Problem.is_accessible_by): hidden problems stay 404
	// unless the viewer is privileged (superuser, see_private_problem,
	// editor, or tester).
	if !auth.CanViewProblem(c, &p) {
		c.JSON(http.StatusNotFound, apiUserError("problem not found"))
		return
	}

	// v1 problem.html shows the ticket link to editors ("Manage tickets") and to
	// users with tickets of their own ("My tickets"); the page decides which
	// one from these two flags.
	canEdit := auth.CanEditProblem(c, &p)
	hasTickets := problemHasTickets(c, &p, canEdit)

	langs := make([]problemLangItem, len(p.AllowedLangs))
	for i, l := range p.AllowedLangs {
		langs[i] = problemLangItem{
			Key:           l.Key,
			Name:          l.Name,
			Extension:     l.Extension,
			FileOnly:      l.FileOnly,
			FileSizeLimit: l.FileSizeLimit,
		}
	}

	types := make([]problemTypeItem, len(p.Types))
	for i, t := range p.Types {
		types[i] = problemTypeItem{Name: t.FullName}
	}

	authors := make([]problemAuthorItem, len(p.Authors))
	for i, a := range p.Authors {
		authors[i] = problemAuthorItem{Username: a.User.Username}
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

	// v1 problem.html shows the license as a link to /license/<key>, labelled
	// with display (falling back to name). nil when the problem has none.
	var licensePayload interface{}
	if p.License != nil {
		licensePayload = gin.H{
			"id":      p.License.ID,
			"key":     p.License.Key,
			"name":    p.License.Name,
			"link":    p.License.Link,
			"display": p.License.Display,
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
		"license":        licensePayload,
		"can_edit":       canEdit,
		"has_tickets":    hasTickets,
	})
}

// problemHasTickets reports whether the request user has any ticket to see on
// this problem: all of them for editors, otherwise only the ones they opened or
// are assigned to (v1 Problem.is_editable_by + own_ticket_filter). Anonymous
// visitors have none.
func problemHasTickets(c *gin.Context, problem *models.Problem, canEdit bool) bool {
	profileID, ok := auth.CurrentProfileID(c)
	if !ok {
		return false
	}
	contentTypeID, ok := ticketContentTypeID("problem")
	if !ok {
		return false
	}
	q := db.DB.Model(&models.Ticket{}).
		Where("content_type_id = ? AND object_id = ?", contentTypeID, problem.ID)
	if !canEdit {
		q = q.Where("(user_id = ? OR EXISTS (SELECT 1 FROM judge_ticket_assignees a "+
			"WHERE a.ticket_id = judge_ticket.id AND a.profile_id = ?))", profileID, profileID)
	}
	var count int64
	return q.Count(&count).Error == nil && count > 0
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
		// Must resolve to a file strictly under the media root — reject the root
		// directory itself (e.g. pdf_url "/") as well as anything that escapes it.
		if clean == root || !strings.HasPrefix(clean, root+string(os.PathSeparator)) {
			return "", errPDFInvalidPath
		}
		return clean, nil
	}
	base := filepath.Clean(filepath.Join("data", "problems", code))
	clean := filepath.Clean(filepath.Join(base, pdfURL))
	if clean == base || !strings.HasPrefix(clean, base+string(os.PathSeparator)) {
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

	// In R2: hand out a short-lived signed link (the permission check above is
	// what guards it). ?url=1 answers JSON for the in-page viewer, which fetches
	// the link itself; otherwise redirect, for plain download links.
	if link, ok, err := storage.PDFLink(c.Request.Context(), storage.PDFKey(problem.PdfURL, code), code+".pdf"); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "PDF storage unavailable"})
		return
	} else if ok {
		c.Header("Cache-Control", "private, no-store")
		if c.Query("url") == "1" {
			c.JSON(http.StatusOK, gin.H{"url": link})
		} else {
			c.Redirect(http.StatusFound, link)
		}
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
