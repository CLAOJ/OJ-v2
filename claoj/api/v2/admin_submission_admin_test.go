package v2

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func setupAdminSubmissionDB(t *testing.T) {
	t.Helper()
	old := db.DB
	t.Cleanup(func() { db.DB = old })
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)
	sqlDB, err := database.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, database.AutoMigrate(
		&models.AuthUser{}, &models.Profile{}, &models.Organization{}, &models.Language{},
		&models.ProblemGroup{}, &models.ProblemType{}, &models.Problem{}, &models.Judge{},
		&models.Contest{}, &models.ContestParticipation{}, &models.ContestSubmission{},
		&models.Submission{}, &models.SubmissionSource{}, &models.SubmissionTestCase{},
		&models.DjangoContentType{}, &models.AuthPermission{}, &models.AuthUserPermission{},
		&models.AuthGroup{}, &models.AuthUserGroup{}, &models.AuthGroupPermission{},
	))
	db.DB = database
	submissionService = nil
}

func newSubmissionUser(t *testing.T, username string, superuser bool) models.AuthUser {
	t.Helper()
	u := models.AuthUser{Username: username, IsActive: true, IsStaff: true, IsSuperuser: superuser}
	require.NoError(t, db.DB.Create(&u).Error)
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&models.Profile{UserID: u.ID, Timezone: "UTC"}).Error)
	return u
}

func grantSubmissionPerm(t *testing.T, userID uint, codename string) {
	t.Helper()
	ct := models.DjangoContentType{AppLabel: "judge", Model: "submission"}
	require.NoError(t, db.DB.Create(&ct).Error)
	perm := models.AuthPermission{Name: codename, ContentTypeID: ct.ID, Codename: codename}
	require.NoError(t, db.DB.Create(&perm).Error)
	require.NoError(t, db.DB.Create(&models.AuthUserPermission{UserID: userID, PermissionID: perm.ID}).Error)
}

type submissionFixture struct {
	sub     models.Submission
	owner   models.Profile
	problem models.Problem
	lang    models.Language
	judge   models.Judge
	contest models.Contest
}

func seedAdminSubmission(t *testing.T) submissionFixture {
	t.Helper()
	ownerUser := models.AuthUser{Username: "alice", IsActive: true}
	require.NoError(t, db.DB.Create(&ownerUser).Error)
	owner := models.Profile{UserID: ownerUser.ID, Timezone: "UTC"}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&owner).Error)

	problem := models.Problem{Code: "aplusb", Name: "A+B"}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&problem).Error)
	lang := models.Language{Key: "CPP17", Name: "C++17", CommonName: "C++"}
	require.NoError(t, db.DB.Create(&lang).Error)
	judge := models.Judge{Name: "judge-1", AuthKey: "secret", Created: time.Now()}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&judge).Error)
	contest := models.Contest{Key: "tik25", Name: "TiK25", StartTime: time.Now().Add(-time.Hour), EndTime: time.Now().Add(time.Hour)}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&contest).Error)

	ac := "AC"
	pts := 100.0
	locked := time.Now().Add(-time.Hour)
	sub := models.Submission{
		UserID: owner.ID, ProblemID: problem.ID, LanguageID: lang.ID, Date: time.Now(),
		Status: "D", Result: &ac, Points: &pts, CasePoints: 100, CaseTotal: 100,
		JudgedOnID: &judge.ID, JudgedDate: &locked, ContestObjectID: &contest.ID,
		LockedAfter: &locked, IsArchived: true, Memory: &pts, Time: &pts,
	}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&sub).Error)
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&models.SubmissionSource{SubmissionID: sub.ID, Source: "int main() {}"}).Error)
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&models.SubmissionTestCase{
		SubmissionID: sub.ID, Case: 1, Status: "AC", Feedback: "ok", Points: &pts, Total: &pts,
	}).Error)

	return submissionFixture{sub: sub, owner: owner, problem: problem, lang: lang, judge: judge, contest: contest}
}

func TestAdminSubmissionDetail_Shape(t *testing.T) {
	setupAdminSubmissionDB(t)
	seedAdminSubmission(t)
	root := newSubmissionUser(t, "root", true)

	router := authedRouter(root.ID, http.MethodGet, "/admin/submission/:id", AdminSubmissionDetail)
	w := doRequest(router, http.MethodGet, "/admin/submission/1", "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, "alice", body["user"])
	require.Equal(t, "aplusb", body["problem"])
	require.Equal(t, "A+B", body["problem_name"])
	require.Equal(t, "C++17", body["language_name"])
	require.Equal(t, "judge-1", body["judged_on"])
	require.Equal(t, "tik25", body["contest_key"])
	require.Equal(t, "int main() {}", body["source"])
	require.Equal(t, true, body["is_locked"])
	require.Equal(t, true, body["is_archived"])
	require.Equal(t, float64(100), body["case_points"])
	tcs := body["test_cases"].([]any)
	require.Len(t, tcs, 1)
	require.Equal(t, "ok", tcs[0].(map[string]any)["feedback"])
	require.NotContains(t, body, "judge", "detail must not nest relations")
}

func TestAdminSubmissionUpdate_HappyPathAndSource(t *testing.T) {
	setupAdminSubmissionDB(t)
	f := seedAdminSubmission(t)
	root := newSubmissionUser(t, "root", true)

	router := authedRouter(root.ID, http.MethodPatch, "/admin/submission/:id", AdminSubmissionUpdate)
	body := `{
		"language_id": ` + itoa(f.lang.ID) + `,
		"status": "CE",
		"result": "CE",
		"points": 0,
		"case_points": 1.5,
		"case_total": 200,
		"judged_on_id": ` + itoa(f.judge.ID) + `,
		"judged_date": "2026-11-20T10:00",
		"error": "compile failed",
		"contest_object_id": null,
		"source": "print(1)",
		"locked_after": null,
		"is_archived": false
	}`
	w := doRequest(router, http.MethodPatch, "/admin/submission/1", body)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var got models.Submission
	require.NoError(t, db.DB.First(&got, f.sub.ID).Error)
	require.Equal(t, "CE", got.Status)
	require.Equal(t, "CE", *got.Result)
	require.Equal(t, 1.5, got.CasePoints)
	require.Equal(t, 200.0, got.CaseTotal)
	require.Equal(t, "compile failed", *got.Error)
	require.Nil(t, got.ContestObjectID)
	require.Nil(t, got.LockedAfter)
	require.False(t, got.IsArchived)
	require.Equal(t, 2026, got.JudgedDate.Year())
	require.Equal(t, f.lang.ID, got.LanguageID)
	require.Equal(t, 0.0, *got.Points)
	require.Equal(t, f.judge.ID, *got.JudgedOnID)

	var src models.SubmissionSource
	require.NoError(t, db.DB.Where("submission_id = ?", f.sub.ID).First(&src).Error)
	require.Equal(t, "print(1)", src.Source)
}

func TestAdminSubmissionUpdate_SourceByteCap(t *testing.T) {
	setupAdminSubmissionDB(t)
	f := seedAdminSubmission(t)
	root := newSubmissionUser(t, "root", true)
	router := authedRouter(root.ID, http.MethodPatch, "/admin/submission/:id", AdminSubmissionUpdate)

	tooLong := `{"source":"` + strings.Repeat("a", maxSourceBytes+1) + `"}`
	require.Equal(t, http.StatusBadRequest, doRequest(router, http.MethodPatch, "/admin/submission/1", tooLong).Code)

	var src models.SubmissionSource
	require.NoError(t, db.DB.Where("submission_id = ?", f.sub.ID).First(&src).Error)
	require.Equal(t, "int main() {}", src.Source, "an oversized source must not be written")

	// Exactly at the cap is allowed, matching the submit path.
	atCap := `{"source":"` + strings.Repeat("b", maxSourceBytes) + `"}`
	require.Equal(t, http.StatusOK, doRequest(router, http.MethodPatch, "/admin/submission/1", atCap).Code)
	require.NoError(t, db.DB.Where("submission_id = ?", f.sub.ID).First(&src).Error)
	require.Len(t, src.Source, maxSourceBytes)
}

// A second PATCH must update the existing judge_submissionsource row instead of
// inserting a duplicate.
func TestAdminSubmissionUpdate_SecondPatchDoesNotDuplicateSource(t *testing.T) {
	setupAdminSubmissionDB(t)
	f := seedAdminSubmission(t)
	root := newSubmissionUser(t, "root", true)
	router := authedRouter(root.ID, http.MethodPatch, "/admin/submission/:id", AdminSubmissionUpdate)

	require.Equal(t, http.StatusOK, doRequest(router, http.MethodPatch, "/admin/submission/1", `{"source":"first"}`).Code)
	require.Equal(t, http.StatusOK, doRequest(router, http.MethodPatch, "/admin/submission/1", `{"source":"second"}`).Code)

	require.Equal(t, int64(1), countSubmissionRows(t, "judge_submissionsource", f.sub.ID))
	var src models.SubmissionSource
	require.NoError(t, db.DB.Where("submission_id = ?", f.sub.ID).First(&src).Error)
	require.Equal(t, "second", src.Source)
}

func TestAdminSubmissionUpdate_GatesAndValidation(t *testing.T) {
	setupAdminSubmissionDB(t)
	f := seedAdminSubmission(t)

	// Can change ordinary fields but not lock or archive.
	editor := newSubmissionUser(t, "editor", false)
	grantSubmissionPerm(t, editor.ID, "change_submission")
	router := authedRouter(editor.ID, http.MethodPatch, "/admin/submission/:id", AdminSubmissionUpdate)

	require.Equal(t, http.StatusOK, doRequest(router, http.MethodPatch, "/admin/submission/1", `{"points": 42}`).Code)
	require.Equal(t, http.StatusForbidden, doRequest(router, http.MethodPatch, "/admin/submission/1", `{"locked_after": "2026-11-20T10:00"}`).Code)
	require.Equal(t, http.StatusForbidden, doRequest(router, http.MethodPatch, "/admin/submission/1", `{"is_archived": false}`).Code)

	// No change permission at all.
	plain := newSubmissionUser(t, "plain", false)
	plainRouter := authedRouter(plain.ID, http.MethodPatch, "/admin/submission/:id", AdminSubmissionUpdate)
	require.Equal(t, http.StatusForbidden, doRequest(plainRouter, http.MethodPatch, "/admin/submission/1", `{"points": 1}`).Code)

	root := newSubmissionUser(t, "root", true)
	rootRouter := authedRouter(root.ID, http.MethodPatch, "/admin/submission/:id", AdminSubmissionUpdate)
	for _, body := range []string{
		`{"status": "ZZ"}`,
		`{"result": "ZZ"}`,
		`{"language_id": 999}`,
		`{"judged_on_id": 999}`,
		`{"contest_object_id": 999}`,
		`{"judged_date": "not-a-date"}`,
		`{"points": -1}`,
		`{"points": 100001}`,
		`{"case_points": -0.5}`,
		`{"case_total": 100001}`,
	} {
		require.Equal(t, http.StatusBadRequest, doRequest(rootRouter, http.MethodPatch, "/admin/submission/1", body).Code, body)
	}

	// The bound itself is allowed.
	require.Equal(t, http.StatusOK, doRequest(rootRouter, http.MethodPatch, "/admin/submission/1", `{"points": 100000}`).Code)

	var got models.Submission
	require.NoError(t, db.DB.First(&got, f.sub.ID).Error)
	require.Equal(t, "D", got.Status, "rejected patches must not write")

	// Identity and date fields are not part of the request struct.
	require.Equal(t, http.StatusOK, doRequest(rootRouter, http.MethodPatch, "/admin/submission/1",
		`{"user_id": 999, "problem_id": 999, "date": "2000-01-01T00:00:00Z"}`).Code)
	var after models.Submission
	require.NoError(t, db.DB.First(&after, f.sub.ID).Error)
	require.Equal(t, f.owner.ID, after.UserID)
	require.Equal(t, f.problem.ID, after.ProblemID)
	require.Equal(t, got.Date.Unix(), after.Date.Unix())
}

// Archiving toggles what the submission contributes, so the stored profile and
// organization points must be recalculated (v1 recalculates on archive; the
// delete path does the same work).
func TestAdminSubmissionUpdate_ArchiveRecalculatesPoints(t *testing.T) {
	setupAdminSubmissionDB(t)
	f := seedAdminSubmission(t)
	require.NoError(t, db.DB.AutoMigrate(&models.Organization{}))
	require.NoError(t, db.DB.Model(&models.Problem{}).Where("id = ?", f.problem.ID).
		Updates(map[string]interface{}{"is_public": true, "partial": false, "points": 100}).Error)

	// An unarchived, fully accepted submission in a public problem: the stored
	// columns start at what it contributes.
	require.NoError(t, db.DB.Model(&models.Submission{}).Where("id = ?", f.sub.ID).
		Update("is_archived", false).Error)
	require.NoError(t, db.DB.Model(&models.Profile{}).Where("id = ?", f.owner.ID).
		Updates(map[string]interface{}{"points": 100.0, "performance_points": 100.0, "problem_count": 1}).Error)

	root := newSubmissionUser(t, "root", true)
	router := authedRouter(root.ID, http.MethodPatch, "/admin/submission/:id", AdminSubmissionUpdate)

	require.Equal(t, http.StatusOK, doRequest(router, http.MethodPatch, "/admin/submission/1", `{"is_archived": true}`).Code)
	var p models.Profile
	require.NoError(t, db.DB.First(&p, f.owner.ID).Error)
	require.InDelta(t, 0.0, p.Points, 1e-9, "an archived submission must stop counting")
	require.Equal(t, 0, p.ProblemCount)

	// Unarchiving restores it.
	require.Equal(t, http.StatusOK, doRequest(router, http.MethodPatch, "/admin/submission/1", `{"is_archived": false}`).Code)
	require.NoError(t, db.DB.First(&p, f.owner.ID).Error)
	require.InDelta(t, 100.0, p.Points, 1e-9)
	require.Equal(t, 1, p.ProblemCount)

	// A patch that does not touch is_archived leaves the points alone.
	require.NoError(t, db.DB.Model(&models.Profile{}).Where("id = ?", f.owner.ID).
		Update("points", 55.0).Error)
	require.Equal(t, http.StatusOK, doRequest(router, http.MethodPatch, "/admin/submission/1", `{"points": 10}`).Code)
	require.NoError(t, db.DB.First(&p, f.owner.ID).Error)
	require.InDelta(t, 55.0, p.Points, 1e-9)
}

func TestAdminSubmissionList_FilteredTotalAndProblemName(t *testing.T) {
	setupAdminSubmissionDB(t)
	seedAdminSubmission(t)
	root := newSubmissionUser(t, "root", true)

	router := authedRouter(root.ID, http.MethodGet, "/admin/submissions", AdminSubmissionList)
	w := doRequest(router, http.MethodGet, "/admin/submissions?status=D", "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var body struct {
		Data  []map[string]any `json:"data"`
		Total int64            `json:"total"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, int64(1), body.Total)
	require.Len(t, body.Data, 1)
	require.Equal(t, "A+B", body.Data[0]["problem_name"])

	w = doRequest(router, http.MethodGet, "/admin/submissions?status=QU", "")
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, int64(0), body.Total)
	require.Len(t, body.Data, 0)

	// Combined filters must intersect, not either-or.
	w = doRequest(router, http.MethodGet, "/admin/submissions?status=D&result=AC", "")
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, int64(1), body.Total)
	require.Len(t, body.Data, 1)

	w = doRequest(router, http.MethodGet, "/admin/submissions?status=D&result=WA", "")
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, int64(0), body.Total)
	require.Len(t, body.Data, 0)
}

func TestAdminSubmissionBulkDelete(t *testing.T) {
	setupAdminSubmissionDB(t)
	f := seedAdminSubmission(t)
	queued := models.Submission{UserID: f.owner.ID, ProblemID: f.problem.ID, LanguageID: f.lang.ID, Date: time.Now(), Status: "QU"}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&queued).Error)

	root := newSubmissionUser(t, "root", true)
	router := authedRouter(root.ID, http.MethodPost, "/admin/submissions/bulk-delete", AdminSubmissionBulkDelete)

	require.Equal(t, http.StatusBadRequest, doRequest(router, http.MethodPost, "/admin/submissions/bulk-delete", `{"ids":[]}`).Code)
	require.Equal(t, http.StatusBadRequest, doRequest(router, http.MethodPost, "/admin/submissions/bulk-delete", `{"ids":[0]}`).Code)

	many := make([]uint, 201)
	for i := range many {
		many[i] = uint(i + 1)
	}
	encoded, _ := json.Marshal(map[string]any{"ids": many})
	require.Equal(t, http.StatusBadRequest, doRequest(router, http.MethodPost, "/admin/submissions/bulk-delete", string(encoded)).Code)

	// Unknown ids are ignored, known ones are deleted.
	body := `{"ids":[` + itoa(f.sub.ID) + `,9999]}`
	w := doRequest(router, http.MethodPost, "/admin/submissions/bulk-delete", body)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Contains(t, w.Body.String(), `"deleted":1`)
	require.Equal(t, int64(0), countSubmissions(t, f.sub.ID))
	require.Equal(t, int64(0), countSubmissionRows(t, "judge_submissionsource", f.sub.ID))
	require.Equal(t, int64(1), countSubmissions(t, queued.ID), "queued submissions are untouched")

	// A batch containing a queued submission is refused as a whole.
	require.Equal(t, http.StatusConflict, doRequest(router, http.MethodPost, "/admin/submissions/bulk-delete", `{"ids":[`+itoa(queued.ID)+`]}`).Code)

	// Missing permission.
	plain := newSubmissionUser(t, "plain", false)
	plainRouter := authedRouter(plain.ID, http.MethodPost, "/admin/submissions/bulk-delete", AdminSubmissionBulkDelete)
	require.Equal(t, http.StatusForbidden, doRequest(plainRouter, http.MethodPost, "/admin/submissions/bulk-delete", `{"ids":[`+itoa(queued.ID)+`]}`).Code)
}

func countSubmissions(t *testing.T, id uint) int64 {
	t.Helper()
	var n int64
	require.NoError(t, db.DB.Model(&models.Submission{}).Where("id = ?", id).Count(&n).Error)
	return n
}

func countSubmissionRows(t *testing.T, table string, submissionID uint) int64 {
	t.Helper()
	var n int64
	require.NoError(t, db.DB.Table(table).Where("submission_id = ?", submissionID).Count(&n).Error)
	return n
}
