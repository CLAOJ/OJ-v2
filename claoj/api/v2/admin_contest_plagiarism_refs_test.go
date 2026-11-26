package v2

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm/clause"
)

// setupRefsDB extends setupContestMossDB with the judge_language table, the
// caller's judge_profile row (created_by), two contest problems in tik25 and
// one problem that belongs to a different contest.
func setupRefsDB(t *testing.T) (root, staff models.AuthUser, contest models.Contest, foreign models.Problem) {
	t.Helper()
	root, staff, contest = setupContestMossDB(t)
	require.NoError(t, db.DB.AutoMigrate(&models.Language{}))
	require.NoError(t, db.DB.Create(&models.Language{Key: "CPP11", Name: "C++11", CommonName: "C++", Ace: "c_cpp", Pygments: "cpp", Extension: "cpp"}).Error)
	require.NoError(t, db.DB.Create(&models.Language{Key: "PY3", Name: "Python 3", CommonName: "PY3", Ace: "python", Pygments: "python", Extension: "py"}).Error)
	require.NoError(t, db.DB.Create(&models.Profile{UserID: root.ID, Timezone: "UTC", LastAccess: time.Now()}).Error)

	knap := addRefsProblem(t, "knap", "Knapsack")
	sort := addRefsProblem(t, "sort", "Sort")
	require.NoError(t, db.DB.Create(&models.ContestProblem{ContestID: contest.ID, ProblemID: sort.ID, Points: 10, Order: 2}).Error)
	require.NoError(t, db.DB.Create(&models.ContestProblem{ContestID: contest.ID, ProblemID: knap.ID, Points: 10, Order: 1}).Error)

	foreign = addRefsProblem(t, "foreign", "Foreign")
	other := models.Contest{Key: "other", Name: "Other", StartTime: time.Now(), EndTime: time.Now().Add(time.Hour)}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&other).Error)
	require.NoError(t, db.DB.Create(&models.ContestProblem{ContestID: other.ID, ProblemID: foreign.ID, Points: 10, Order: 1}).Error)
	return
}

func addRefsProblem(t *testing.T, code, name string) models.Problem {
	t.Helper()
	p := models.Problem{Code: code, Name: name}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&p).Error)
	return p
}

func seedRef(t *testing.T, problemID uint, language, origin, source string) uint {
	t.Helper()
	ref := models.PlagiarismAIReference{ProblemID: problemID, Language: language, Source: source, Origin: origin, CreatedAt: time.Now()}
	require.NoError(t, db.DB.Create(&ref).Error)
	return ref.ID
}

func TestRefs_RequiresPermission(t *testing.T) {
	_, staff, _, _ := setupRefsDB(t)
	list := authedRouter(staff.ID, http.MethodGet, "/admin/contest/:key/plagiarism/references", AdminContestRefList)
	w := doRequest(list, http.MethodGet, "/admin/contest/tik25/plagiarism/references", "")
	require.Equal(t, http.StatusForbidden, w.Code)

	add := authedRouter(staff.ID, http.MethodPost, "/admin/contest/:key/plagiarism/references", AdminContestRefAdd)
	w = doRequest(add, http.MethodPost, "/admin/contest/tik25/plagiarism/references", `{"problem_code":"knap","language":"C++","source":"x"}`)
	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestRefs_AddListDelete(t *testing.T) {
	root, _, _, _ := setupRefsDB(t)
	var rootProfile models.Profile
	require.NoError(t, db.DB.Where("user_id = ?", root.ID).First(&rootProfile).Error)

	add := authedRouter(root.ID, http.MethodPost, "/admin/contest/:key/plagiarism/references", AdminContestRefAdd)
	w := doRequest(add, http.MethodPost, "/admin/contest/tik25/plagiarism/references", `{"problem_code":"knap","language":"C++","source":"int main() {}"}`)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	var created struct {
		ID uint `json:"id"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))
	require.NotZero(t, created.ID)

	var stored models.PlagiarismAIReference
	require.NoError(t, db.DB.First(&stored, created.ID).Error)
	require.Equal(t, "admin", stored.Origin)
	require.Equal(t, "C++", stored.Language)
	require.Equal(t, "int main() {}", stored.Source)
	require.NotNil(t, stored.CreatedBy)
	require.Equal(t, rootProfile.ID, *stored.CreatedBy)
	require.Nil(t, stored.Model)

	list := authedRouter(root.ID, http.MethodGet, "/admin/contest/:key/plagiarism/references", AdminContestRefList)
	w = doRequest(list, http.MethodGet, "/admin/contest/tik25/plagiarism/references", "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var listed struct {
		References []struct {
			ID          uint    `json:"id"`
			ProblemCode string  `json:"problem_code"`
			Language    string  `json:"language"`
			Origin      string  `json:"origin"`
			Model       *string `json:"model"`
			Source      string  `json:"source"`
		} `json:"references"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &listed))
	require.Len(t, listed.References, 1)
	require.Equal(t, created.ID, listed.References[0].ID)
	require.Equal(t, "knap", listed.References[0].ProblemCode)
	require.Equal(t, "admin", listed.References[0].Origin)
	require.Nil(t, listed.References[0].Model)
	require.Equal(t, "int main() {}", listed.References[0].Source)

	del := authedRouter(root.ID, http.MethodDelete, "/admin/contest/:key/plagiarism/references/:id", AdminContestRefDelete)
	w = doRequest(del, http.MethodDelete, "/admin/contest/tik25/plagiarism/references/"+uitoa(created.ID), "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var n int64
	db.DB.Model(&models.PlagiarismAIReference{}).Count(&n)
	require.Zero(t, n)
}

func TestRefs_ListOrdersByProblemOrderLanguageID(t *testing.T) {
	root, _, _, _ := setupRefsDB(t)
	seedRef(t, mustProblemID(t, "sort"), "C++", "admin", "s1")
	seedRef(t, mustProblemID(t, "knap"), "PY3", "admin", "k1")
	seedRef(t, mustProblemID(t, "knap"), "C++", "generated", "k2")
	seedRef(t, mustProblemID(t, "knap"), "C++", "admin", "k3")

	list := authedRouter(root.ID, http.MethodGet, "/admin/contest/:key/plagiarism/references", AdminContestRefList)
	w := doRequest(list, http.MethodGet, "/admin/contest/tik25/plagiarism/references", "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var listed struct {
		References []struct {
			ProblemCode string `json:"problem_code"`
			Language    string `json:"language"`
			Source      string `json:"source"`
		} `json:"references"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &listed))
	var got []string
	for _, r := range listed.References {
		got = append(got, r.Source)
	}
	require.Equal(t, []string{"k2", "k3", "k1", "s1"}, got)
}

func TestRefs_RejectsProblemOutsideContest(t *testing.T) {
	root, _, _, foreign := setupRefsDB(t)
	add := authedRouter(root.ID, http.MethodPost, "/admin/contest/:key/plagiarism/references", AdminContestRefAdd)
	w := doRequest(add, http.MethodPost, "/admin/contest/tik25/plagiarism/references", `{"problem_code":"foreign","language":"C++","source":"x"}`)
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	w = doRequest(add, http.MethodPost, "/admin/contest/tik25/plagiarism/references", `{"problem_code":"nope","language":"C++","source":"x"}`)
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())

	id := seedRef(t, foreign.ID, "C++", "admin", "f")
	del := authedRouter(root.ID, http.MethodDelete, "/admin/contest/:key/plagiarism/references/:id", AdminContestRefDelete)
	w = doRequest(del, http.MethodDelete, "/admin/contest/tik25/plagiarism/references/"+uitoa(id), "")
	require.Equal(t, http.StatusNotFound, w.Code, w.Body.String())
	var n int64
	db.DB.Model(&models.PlagiarismAIReference{}).Where("id = ?", id).Count(&n)
	require.Equal(t, int64(1), n, "reference of another contest must survive")
}

func TestRefs_RejectsUnknownLanguageAndEmptySource(t *testing.T) {
	root, _, _, _ := setupRefsDB(t)
	add := authedRouter(root.ID, http.MethodPost, "/admin/contest/:key/plagiarism/references", AdminContestRefAdd)
	for _, body := range []string{
		`{"problem_code":"knap","language":"Klingon","source":"x"}`,
		`{"problem_code":"knap","language":"C++","source":"   \n\t "}`,
		`{"problem_code":"knap","language":"C++","source":""}`,
		`{"problem_code":"knap","language":"C++","source":"` + strings.Repeat("a", 65537) + `"}`,
	} {
		w := doRequest(add, http.MethodPost, "/admin/contest/tik25/plagiarism/references", body)
		require.Equal(t, http.StatusBadRequest, w.Code, "body %.60s: %s", body, w.Body.String())
	}

	w := doRequest(add, http.MethodPost, "/admin/contest/tik25/plagiarism/references", `{"problem_code":"knap","language":"C++","source":"`+strings.Repeat("a", 65536)+`"}`)
	require.Equal(t, http.StatusCreated, w.Code, "65536 bytes is the limit and must be accepted")
}

func TestRefs_RegenerateDeletesOnlyGenerated(t *testing.T) {
	root, _, _, _ := setupRefsDB(t)
	seedRef(t, mustProblemID(t, "knap"), "C++", "generated", "g1")
	seedRef(t, mustProblemID(t, "knap"), "PY3", "generated", "g2")
	adminID := seedRef(t, mustProblemID(t, "knap"), "C++", "admin", "a1")
	sortGen := seedRef(t, mustProblemID(t, "sort"), "C++", "generated", "g3")

	del := authedRouter(root.ID, http.MethodDelete, "/admin/contest/:key/plagiarism/references", AdminContestRefRegenerate)
	for _, target := range []string{
		"/admin/contest/tik25/plagiarism/references?problem_code=knap&origin=admin",
		"/admin/contest/tik25/plagiarism/references?problem_code=knap",
		"/admin/contest/tik25/plagiarism/references?problem_code=foreign&origin=generated",
	} {
		w := doRequest(del, http.MethodDelete, target, "")
		require.Equal(t, http.StatusBadRequest, w.Code, "target %s: %s", target, w.Body.String())
	}

	w := doRequest(del, http.MethodDelete, "/admin/contest/tik25/plagiarism/references?problem_code=knap&origin=generated", "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var body struct {
		Deleted int64 `json:"deleted"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, int64(2), body.Deleted)

	var left []models.PlagiarismAIReference
	require.NoError(t, db.DB.Order("id ASC").Find(&left).Error)
	require.Len(t, left, 2)
	require.Equal(t, adminID, left[0].ID)
	require.Equal(t, sortGen, left[1].ID)
}

func TestContestMoss_DeleteKeepsReferences(t *testing.T) {
	useTempReportDir(t)
	root, _, c, _ := setupRefsDB(t)
	knap := mustProblemID(t, "knap")
	reportPath := "knap/cpp"
	require.NoError(t, db.DB.Create(&models.PlagiarismResult{ContestID: c.ID, ProblemID: knap, Language: "C++", ReportPath: &reportPath, CreatedAt: time.Now()}).Error)
	seedRef(t, knap, "C++", "generated", "g")
	seedRef(t, knap, "C++", "admin", "a")

	router := authedRouter(root.ID, http.MethodDelete, "/admin/contest/:key/plagiarism", AdminContestMossDelete)
	w := doRequest(router, http.MethodDelete, "/admin/contest/tik25/plagiarism", "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var n int64
	db.DB.Model(&models.PlagiarismResult{}).Count(&n)
	require.Zero(t, n)
	db.DB.Model(&models.PlagiarismAIReference{}).Count(&n)
	require.Equal(t, int64(2), n, "contest Clear must not touch reference solutions")
}

// mustProblemID returns the id of a problem seeded by setupRefsDB.
func mustProblemID(t *testing.T, code string) uint {
	t.Helper()
	var p models.Problem
	require.NoError(t, db.DB.Where("code = ?", code).First(&p).Error)
	return p.ID
}

func uitoa(v uint) string {
	return strconv.FormatUint(uint64(v), 10)
}
