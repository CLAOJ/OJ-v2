package v2

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/CLAOJ/claoj/scoring"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm/clause"
)

// ============================================================
// Public profile: solved rows, analytics heatmap, blogs by author,
// and the PP header/breakdown agreement (Phase 5 ruling 4).
// Reuses setupAdminUserCreateDB / seedAccount / doRequest.
// ============================================================

type profileFixture struct {
	user    models.AuthUser
	profile models.Profile

	visibleA, visibleB models.Problem
	orgPrivate         models.Problem
	archivedProblem    models.Problem
}

func setupProfileDB(t *testing.T) profileFixture {
	t.Helper()
	setupAdminUserCreateDB(t)
	require.NoError(t, db.DB.AutoMigrate(
		&models.ProblemGroup{}, &models.Problem{}, &models.Submission{},
		&models.Contest{}, &models.ContestParticipation{}, &models.BlogPost{},
	))

	user, profile := seedAccount(t, "solver", false, false)
	group := models.ProblemGroup{Name: "g", FullName: "Group"}
	require.NoError(t, db.DB.Create(&group).Error)

	date := time.Now().Add(-48 * time.Hour)
	newProblem := func(code string, public, orgPrivate bool) models.Problem {
		date = date.Add(time.Minute)
		p := models.Problem{
			Code: code, Name: code + " name", GroupID: group.ID, TimeLimit: 1, MemoryLimit: 256,
			Points: 100, IsPublic: public, IsOrganizationPrivate: orgPrivate, Date: &date,
		}
		require.NoError(t, db.DB.Omit(clause.Associations).Create(&p).Error)
		return p
	}
	return profileFixture{
		user: user, profile: profile,
		visibleA:        newProblem("visa", true, false),
		visibleB:        newProblem("visb", true, false),
		orgPrivate:      newProblem("orgpriv", true, true),
		archivedProblem: newProblem("arch", true, false),
	}
}

func addSubmission(t *testing.T, p models.Problem, profile models.Profile, points float64, result string, archived bool, when time.Time) models.Submission {
	t.Helper()
	s := models.Submission{
		UserID: profile.ID, ProblemID: p.ID, LanguageID: 1, Date: when,
		Status: "D", Result: &result, Points: &points,
		CasePoints: 100, CaseTotal: 100, IsArchived: archived,
	}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&s).Error)
	return s
}

func profileGet(t *testing.T, userID uint, path, route string, handler gin.HandlerFunc) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	if userID == 0 {
		router.GET(route, handler)
	} else {
		router = authedRouter(userID, http.MethodGet, route, handler)
	}
	w := doRequest(router, http.MethodGet, path, "")
	body := map[string]any{}
	if w.Code == http.StatusOK {
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	}
	return w, body
}

// The stored header columns and the breakdown endpoint run one calculation
// (scoring.CalculatePP): org-private problems and archived submissions are
// excluded from both, so the two numbers agree exactly.
func TestProfilePP_HeaderAgreesWithBreakdown(t *testing.T) {
	fx := setupProfileDB(t)
	now := time.Now()
	addSubmission(t, fx.visibleA, fx.profile, 100, "AC", false, now.Add(-3*time.Hour))
	addSubmission(t, fx.visibleB, fx.profile, 50, "AC", false, now.Add(-2*time.Hour))
	addSubmission(t, fx.orgPrivate, fx.profile, 200, "AC", false, now.Add(-time.Hour))
	addSubmission(t, fx.archivedProblem, fx.profile, 300, "AC", true, now)

	// The existing recalc path fills the stored columns.
	require.NoError(t, scoring.CalculateProfilePoints(db.DB, fx.profile.ID))

	w, detail := profileGet(t, 0, "/user/solver", "/user/:user", UserDetail)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	headerPP := detail["performance_points"].(float64)

	w, breakdown := profileGet(t, 0, "/user/solver/pp-breakdown", "/user/:user/pp-breakdown", UserPPBreakdown)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	expected := 100 + 0.95*50 + scoring.PpBonusFunction(2)
	require.InDelta(t, expected, headerPP, 1e-9, "header excludes org-private and archived solves")
	require.InDelta(t, headerPP, breakdown["total"].(float64), 1e-9, "breakdown must equal the header")
	require.InDelta(t, headerPP, breakdown["performance_points"].(float64), 1e-9)
	require.InDelta(t, 150.0, detail["points"].(float64), 1e-9)
	require.InDelta(t, float64(2), detail["problem_count"].(float64), 1e-9)

	problems := breakdown["problems"].([]any)
	require.Len(t, problems, 2)
	codes := []string{problems[0].(map[string]any)["code"].(string), problems[1].(map[string]any)["code"].(string)}
	require.Equal(t, []string{"visa", "visb"}, codes)
}

func TestProfileSolved_ReturnsNameDateLanguageAndTests(t *testing.T) {
	fx := setupProfileDB(t)
	now := time.Now()
	addSubmission(t, fx.visibleA, fx.profile, 60, "AC", false, now.Add(-5*time.Hour))
	addSubmission(t, fx.visibleA, fx.profile, 100, "AC", false, now.Add(-4*time.Hour)) // best
	addSubmission(t, fx.visibleB, fx.profile, 50, "AC", false, now.Add(-3*time.Hour))
	addSubmission(t, fx.orgPrivate, fx.profile, 200, "AC", false, now.Add(-2*time.Hour))
	addSubmission(t, fx.archivedProblem, fx.profile, 300, "AC", true, now)

	w, body := profileGet(t, 0, "/user/solver/solved", "/user/:user/solved", UserSolvedProblems)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	rows := body["data"].([]any)
	require.Len(t, rows, 2, "org-private and archived solves are hidden")
	row := rows[0].(map[string]any)
	require.Equal(t, "visb", row["code"])
	require.Equal(t, "visb name", row["name"])
	require.Contains(t, row, "date")
	require.NotEmpty(t, row["language"])
	require.InDelta(t, 100.0, row["case_points"].(float64), 1e-9)
	require.InDelta(t, 100.0, row["case_total"].(float64), 1e-9)
	require.InDelta(t, 50.0, row["points"].(float64), 1e-9)

	// The best accepted submission is the one reported for a problem.
	require.InDelta(t, 100.0, rows[1].(map[string]any)["points"].(float64), 1e-9)
}

func TestProfileAnalytics_YearHeatmap(t *testing.T) {
	fx := setupProfileDB(t)
	// The endpoint keys days in UTC, so the fixture does too. Noon keeps the
	// database DATE() on the same calendar day in sqlite, which stores
	// timezone-aware timestamps (MySQL DATETIME has no offset).
	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, time.UTC)
	addSubmission(t, fx.visibleA, fx.profile, 100, "AC", false, today)
	addSubmission(t, fx.visibleB, fx.profile, 100, "AC", false, today.AddDate(0, 0, -10))
	addSubmission(t, fx.visibleB, fx.profile, 100, "AC", false, today.AddDate(0, 0, -400)) // outside the year

	w, body := profileGet(t, 0, "/user/solver/analytics", "/user/:user/analytics", UserAnalytics)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	year := body["activity_year"].([]any)
	require.Len(t, year, 365, "one dense row per day")
	firstDay := year[0].(map[string]any)
	lastDay := year[364].(map[string]any)
	require.Equal(t, today.Format("2006-01-02"), lastDay["date"])
	require.Equal(t, today.AddDate(0, 0, -364).Format("2006-01-02"), firstDay["date"])
	require.InDelta(t, float64(1), lastDay["count"].(float64), 1e-9)
	require.InDelta(t, float64(1), year[354].(map[string]any)["count"].(float64), 1e-9)

	// The original 30-day sparse field keeps its shape for existing consumers.
	thirty := body["activity"].([]any)
	require.Len(t, thirty, 2)
	require.Equal(t, today.AddDate(0, 0, -10).Format("2006-01-02"), thirty[0].(map[string]any)["date"])
}

func TestBlogList_AuthorFilter(t *testing.T) {
	fx := setupProfileDB(t)
	otherUser, other := seedAccount(t, "otherblogger", false, false)
	_ = otherUser

	newPost := func(title string, visible bool, author models.Profile) models.BlogPost {
		p := models.BlogPost{
			Title: title, Slug: title, PublishOn: time.Now().Add(-time.Hour),
			Content: "body", Summary: "summary", Visible: visible, Sticky: false,
		}
		require.NoError(t, db.DB.Omit(clause.Associations).Create(&p).Error)
		require.NoError(t, db.DB.Model(&p).Association("Authors").Append(&author))
		return p
	}
	newPost("mine", true, fx.profile)
	newPost("draft", false, fx.profile)
	newPost("theirs", true, other)

	_, body := profileGet(t, 0, "/blogs?author=solver", "/blogs", BlogList)
	require.Len(t, body["data"].([]any), 1)
	require.Equal(t, "mine", body["data"].([]any)[0].(map[string]any)["title"])
	require.InDelta(t, float64(1), body["total"].(float64), 1e-9)

	_, byID := profileGet(t, 0, "/blogs?author="+strconv.Itoa(int(fx.profile.ID)), "/blogs", BlogList)
	require.Len(t, byID["data"].([]any), 1)

	w, _ := profileGet(t, 0, "/blogs?author=nobody", "/blogs", BlogList)
	require.Equal(t, http.StatusNotFound, w.Code)
}