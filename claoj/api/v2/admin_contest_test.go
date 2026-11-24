package v2

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm/clause"
)

func createContestTestUser(t *testing.T, username string) models.Profile {
	t.Helper()
	u := models.AuthUser{Username: username, IsActive: true}
	require.NoError(t, db.DB.Create(&u).Error)
	p := models.Profile{UserID: u.ID, Timezone: "UTC"}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&p).Error)
	return p
}

func createContestTestOrg(t *testing.T, name string) models.Organization {
	t.Helper()
	o := models.Organization{Name: name, Slug: name, ShortName: name, CreationDate: time.Now()}
	require.NoError(t, db.DB.Omit("Admins", "Members").Create(&o).Error)
	return o
}

func intp(v int) *int { return &v }

// grantContestPerm seeds judge.<codename> and attaches it to the auth user,
// the same rows v1's Django admin reads.
func grantContestPerm(t *testing.T, userID uint, codename string) {
	t.Helper()
	var ct models.DjangoContentType
	require.NoError(t, db.DB.FirstOrCreate(&ct, models.DjangoContentType{AppLabel: "judge", Model: "contest"}).Error)
	perm := models.AuthPermission{Name: codename, Codename: codename, ContentTypeID: ct.ID}
	require.NoError(t, db.DB.Create(&perm).Error)
	require.NoError(t, db.DB.Create(&models.AuthUserPermission{UserID: userID, PermissionID: perm.ID}).Error)
}

func TestAdminContestCreate_AndUpdateV1Fields(t *testing.T) {
	setupAdminDetailDB(t)
	viewer := createContestTestUser(t, "viewer")
	troll := createContestTestUser(t, "troll")
	private := createContestTestUser(t, "private")
	org := createContestTestOrg(t, "org")
	joinOrg := createContestTestOrg(t, "joinorg")
	class := models.Class{OrganizationID: org.ID, Name: "Class A", Slug: "class-a"}
	require.NoError(t, db.DB.Create(&class).Error)
	// The two v1-gated fields below need their Django permission.
	grantContestPerm(t, viewer.UserID, "contest_problem_label")
	grantContestPerm(t, viewer.UserID, "override_performance_ceiling")

	body := `{
		"key":"full1","name":"Full","description":"d",
		"start_time":"2026-11-01T09:00:00Z","end_time":"2026-11-01T12:00:00Z",
		"is_rated":true,"scoreboard_visibility":"C","scoreboard_cache_timeout":30,
		"use_clarifications":false,"push_announcements":true,
		"rating_floor":1200,"rating_ceiling":2000,"performance_ceiling_override":2100,
		"rate_all":true,"is_private":true,"hide_problem_authors":true,"show_short_display":true,
		"limit_join_organizations":true,
		"og_image":"https://example.com/og.png","logo_override_image":"https://example.com/logo.png",
		"problem_label_script":"return 'A'","points_precision":5,
		"organization_ids":[` + itoa(org.ID) + `],
		"banned_user_ids":[` + itoa(troll.ID) + `],
		"private_contestant_ids":[` + itoa(private.ID) + `],
		"view_scoreboard_user_ids":[` + itoa(viewer.ID) + `],
		"join_organization_ids":[` + itoa(joinOrg.ID) + `],
		"class_ids":[` + itoa(class.ID) + `]
	}`
	w := doRequest(authedRouter(1, http.MethodPost, "/admin/contests", AdminContestCreate),
		http.MethodPost, "/admin/contests", body)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	w = doRequest(authedRouter(1, http.MethodGet, "/admin/contest/:key", AdminContestDetail),
		http.MethodGet, "/admin/contest/full1", "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var got map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	require.Equal(t, "C", got["scoreboard_visibility"])
	require.EqualValues(t, 30, got["scoreboard_cache_timeout"])
	require.Equal(t, false, got["use_clarifications"])
	require.Equal(t, true, got["push_announcements"])
	require.EqualValues(t, 1200, got["rating_floor"])
	require.EqualValues(t, 2000, got["rating_ceiling"])
	require.EqualValues(t, 2100, got["performance_ceiling_override"])
	require.Equal(t, true, got["rate_all"])
	require.Equal(t, true, got["is_private"])
	require.Equal(t, true, got["hide_problem_authors"])
	require.Equal(t, true, got["show_short_display"])
	require.Equal(t, true, got["limit_join_organizations"])
	require.Equal(t, "https://example.com/og.png", got["og_image"])
	require.Equal(t, "https://example.com/logo.png", got["logo_override_image"])
	require.Equal(t, "return 'A'", got["problem_label_script"])
	require.EqualValues(t, 5, got["points_precision"])
	require.Equal(t, []any{float64(org.ID)}, got["organization_ids"])
	require.Equal(t, []any{float64(troll.ID)}, got["banned_user_ids"])
	require.Equal(t, []any{float64(private.ID)}, got["private_contestant_ids"])
	require.Equal(t, []any{float64(viewer.ID)}, got["view_scoreboard_user_ids"])
	require.Equal(t, []any{float64(joinOrg.ID)}, got["join_organization_ids"])
	require.Equal(t, []any{float64(class.ID)}, got["class_ids"])

	refs := func(key, name string) {
		list, ok := got[key].([]any)
		require.True(t, ok, key)
		require.Len(t, list, 1, key)
		require.Equal(t, name, list[0].(map[string]any)["name"], key)
	}
	refs("join_organizations", "joinorg")
	refs("classes", "Class A")
	users := func(key, username string) {
		list, ok := got[key].([]any)
		require.True(t, ok, key)
		require.Len(t, list, 1, key)
		require.Equal(t, username, list[0].(map[string]any)["username"], key)
	}
	users("banned_users", "troll")
	users("private_contestants", "private")
	users("view_scoreboard_users", "viewer")

	// Clear the relations and the nullable ints.
	patch := `{"scoreboard_visibility":"P","use_clarifications":true,
		"is_private":false,"scoreboard_cache_timeout":0,"limit_join_organizations":false,
		"organization_ids":[],"banned_user_ids":[],"private_contestant_ids":[],
		"view_scoreboard_user_ids":[],"join_organization_ids":[],"class_ids":[],
		"clear_rating_floor":true,"clear_rating_ceiling":true,"clear_performance_ceiling_override":true}`
	w = doRequest(authedRouter(1, http.MethodPatch, "/admin/contest/:key", AdminContestUpdate),
		http.MethodPatch, "/admin/contest/full1", patch)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	w = doRequest(authedRouter(1, http.MethodGet, "/admin/contest/:key", AdminContestDetail),
		http.MethodGet, "/admin/contest/full1", "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	require.Equal(t, "P", got["scoreboard_visibility"])
	require.Equal(t, true, got["use_clarifications"])
	require.Equal(t, false, got["is_private"])
	require.EqualValues(t, 0, got["scoreboard_cache_timeout"])
	require.Equal(t, false, got["limit_join_organizations"])
	require.Nil(t, got["rating_floor"])
	require.Nil(t, got["rating_ceiling"])
	require.Nil(t, got["performance_ceiling_override"])
	require.Empty(t, got["organization_ids"])
	require.Empty(t, got["banned_user_ids"])
	require.Empty(t, got["private_contestant_ids"])
	require.Empty(t, got["view_scoreboard_user_ids"])
	require.Empty(t, got["join_organization_ids"])
	require.Empty(t, got["class_ids"])
	require.Empty(t, got["classes"])
	require.Empty(t, got["join_organizations"])
}

func TestAdminContestCreate_InvalidScoreboardVisibilityIs400(t *testing.T) {
	setupAdminDetailDB(t)
	body := `{"key":"badvis","name":"Bad","description":"d",
		"start_time":"2026-11-01T09:00:00Z","end_time":"2026-11-01T12:00:00Z",
		"scoreboard_visibility":"X"}`
	w := doRequest(authedRouter(1, http.MethodPost, "/admin/contests", AdminContestCreate),
		http.MethodPost, "/admin/contests", body)
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())

	var count int64
	require.NoError(t, db.DB.Model(&models.Contest{}).Where("`key` = ?", "badvis").Count(&count).Error)
	require.Zero(t, count)
}

func TestAdminContestUpdate_InvalidScoreboardVisibilityKeepsStoredValue(t *testing.T) {
	setupAdminDetailDB(t)
	c := models.Contest{Key: "vis", Name: "Vis", StartTime: time.Now(), EndTime: time.Now().Add(time.Hour), FormatName: "default"}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&c).Error)

	w := doRequest(authedRouter(1, http.MethodPatch, "/admin/contest/:key", AdminContestUpdate),
		http.MethodPatch, "/admin/contest/vis", `{"scoreboard_visibility":"Q"}`)
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())

	var got models.Contest
	require.NoError(t, db.DB.First(&got, c.ID).Error)
	require.Equal(t, "V", got.ScoreboardVisibility)
}

func TestAdminContest_UnknownClassIdIs400(t *testing.T) {
	setupAdminDetailDB(t)
	body := `{"key":"badclass","name":"Bad","description":"d",
		"start_time":"2026-11-01T09:00:00Z","end_time":"2026-11-01T12:00:00Z",
		"class_ids":[424242]}`
	w := doRequest(authedRouter(1, http.MethodPost, "/admin/contests", AdminContestCreate),
		http.MethodPost, "/admin/contests", body)
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())

	var count int64
	require.NoError(t, db.DB.Model(&models.Contest{}).Where("`key` = ?", "badclass").Count(&count).Error)
	require.Zero(t, count, "the rejected relation must not leave an orphan row")
}

func TestAdminContest_ImageLengthCapsAre400(t *testing.T) {
	setupAdminDetailDB(t)
	long := strings.Repeat("o", 151)

	body := `{"key":"longog","name":"L","description":"d",
		"start_time":"2026-11-01T09:00:00Z","end_time":"2026-11-01T12:00:00Z",
		"og_image":"` + long + `"}`
	w := doRequest(authedRouter(1, http.MethodPost, "/admin/contests", AdminContestCreate),
		http.MethodPost, "/admin/contests", body)
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())

	c := models.Contest{Key: "lenlogo", Name: "L", StartTime: time.Now(), EndTime: time.Now().Add(time.Hour), FormatName: "default"}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&c).Error)
	w = doRequest(authedRouter(1, http.MethodPatch, "/admin/contest/:key", AdminContestUpdate),
		http.MethodPatch, "/admin/contest/lenlogo", `{"logo_override_image":"`+long+`"}`)
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())

	var got models.Contest
	require.NoError(t, db.DB.First(&got, c.ID).Error)
	require.Empty(t, got.LogoOverrideImage)
}

// TestAdminContest_ElevatedFieldsForbiddenWithoutPerm proves the v1
// readonly_fields gates: an author without judge.contest_problem_label /
// judge.override_performance_ceiling cannot write those fields, and nothing
// is stored.
func TestAdminContest_ElevatedFieldsForbiddenWithoutPerm(t *testing.T) {
	setupAdminDetailDB(t)
	author := createContestTestUser(t, "author")
	c := models.Contest{
		Key: "gated", Name: "Gated", StartTime: time.Now(), EndTime: time.Now().Add(time.Hour),
		FormatName: "default", ProblemLabelScript: "keep", PerformanceCeilingOverride: intp(2100),
	}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&c).Error)

	cases := []struct{ name, createBody, updateBody string }{
		{
			name: "problem_label_script",
			createBody: `{"key":"gated2","name":"G","description":"d",
				"start_time":"2026-11-01T09:00:00Z","end_time":"2026-11-01T12:00:00Z",
				"problem_label_script":"return 'A'"}`,
			updateBody: `{"problem_label_script":"return 'A'"}`,
		},
		{
			name: "performance_ceiling_override",
			createBody: `{"key":"gated2","name":"G","description":"d",
				"start_time":"2026-11-01T09:00:00Z","end_time":"2026-11-01T12:00:00Z",
				"performance_ceiling_override":2100}`,
			updateBody: `{"performance_ceiling_override":2200}`,
		},
		{
			name:       "clear_performance_ceiling_override",
			updateBody: `{"clear_performance_ceiling_override":true}`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.createBody != "" {
				w := doRequest(authedRouter(author.UserID, http.MethodPost, "/admin/contests", AdminContestCreate),
					http.MethodPost, "/admin/contests", tc.createBody)
				require.Equal(t, http.StatusForbidden, w.Code, w.Body.String())

				var count int64
				require.NoError(t, db.DB.Model(&models.Contest{}).Where("`key` = ?", "gated2").Count(&count).Error)
				require.Zero(t, count, "a denied field must not leave a contest row")
			}

			w := doRequest(authedRouter(author.UserID, http.MethodPatch, "/admin/contest/:key", AdminContestUpdate),
				http.MethodPatch, "/admin/contest/gated", tc.updateBody)
			require.Equal(t, http.StatusForbidden, w.Code, w.Body.String())
		})
	}

	var got models.Contest
	require.NoError(t, db.DB.First(&got, c.ID).Error)
	require.Equal(t, "keep", got.ProblemLabelScript, "a denied update must not write")
	require.Equal(t, intp(2100), got.PerformanceCeilingOverride, "a denied clear must not write")
}

// TestAdminContest_UnchangedElevatedFieldsNeedNoPerm covers the form
// round-trip: the edit page PATCHes every field, so an author without the
// elevated permissions must still save while those two values stay as stored.
func TestAdminContest_UnchangedElevatedFieldsNeedNoPerm(t *testing.T) {
	setupAdminDetailDB(t)
	author := createContestTestUser(t, "author")
	c := models.Contest{
		Key: "roundtrip", Name: "Roundtrip", StartTime: time.Now(), EndTime: time.Now().Add(time.Hour),
		FormatName: "default", ProblemLabelScript: "keep", PerformanceCeilingOverride: intp(2100),
	}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&c).Error)

	patch := func(body string) *httptest.ResponseRecorder {
		return doRequest(authedRouter(author.UserID, http.MethodPatch, "/admin/contest/:key", AdminContestUpdate),
			http.MethodPatch, "/admin/contest/roundtrip", body)
	}
	stored := func() models.Contest {
		var got models.Contest
		require.NoError(t, db.DB.First(&got, c.ID).Error)
		return got
	}

	w := patch(`{"name":"Renamed","problem_label_script":"keep","performance_ceiling_override":2100}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	got := stored()
	require.Equal(t, "Renamed", got.Name)
	require.Equal(t, "keep", got.ProblemLabelScript)
	require.Equal(t, intp(2100), got.PerformanceCeilingOverride)

	w = patch(`{"summary":"just a summary"}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// A real change still needs the permission and writes nothing.
	for _, body := range []string{
		`{"name":"Nope","problem_label_script":"changed"}`,
		`{"performance_ceiling_override":2200}`,
		`{"clear_performance_ceiling_override":true}`,
	} {
		w = patch(body)
		require.Equal(t, http.StatusForbidden, w.Code, body)
	}
	got = stored()
	require.Equal(t, "Renamed", got.Name)
	require.Equal(t, "keep", got.ProblemLabelScript)
	require.Equal(t, intp(2100), got.PerformanceCeilingOverride)
}

// TestAdminContest_ElevatedFieldsAllowedWithPerm is the positive half: the
// same fields succeed once the permissions are granted.
func TestAdminContest_ElevatedFieldsAllowedWithPerm(t *testing.T) {
	setupAdminDetailDB(t)
	author := createContestTestUser(t, "labeler")
	grantContestPerm(t, author.UserID, "contest_problem_label")
	grantContestPerm(t, author.UserID, "override_performance_ceiling")

	body := `{"key":"allowed","name":"A","description":"d",
		"start_time":"2026-11-01T09:00:00Z","end_time":"2026-11-01T12:00:00Z",
		"problem_label_script":"return 'A'","performance_ceiling_override":2100}`
	w := doRequest(authedRouter(author.UserID, http.MethodPost, "/admin/contests", AdminContestCreate),
		http.MethodPost, "/admin/contests", body)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	detail := func() map[string]any {
		w := doRequest(authedRouter(author.UserID, http.MethodGet, "/admin/contest/:key", AdminContestDetail),
			http.MethodGet, "/admin/contest/allowed", "")
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		var got map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
		return got
	}
	got := detail()
	require.Equal(t, "return 'A'", got["problem_label_script"])
	require.EqualValues(t, 2100, got["performance_ceiling_override"])

	w = doRequest(authedRouter(author.UserID, http.MethodPatch, "/admin/contest/:key", AdminContestUpdate),
		http.MethodPatch, "/admin/contest/allowed",
		`{"problem_label_script":"return 'B'","performance_ceiling_override":2200}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	got = detail()
	require.Equal(t, "return 'B'", got["problem_label_script"])
	require.EqualValues(t, 2200, got["performance_ceiling_override"])

	w = doRequest(authedRouter(author.UserID, http.MethodPatch, "/admin/contest/:key", AdminContestUpdate),
		http.MethodPatch, "/admin/contest/allowed", `{"clear_performance_ceiling_override":true}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Nil(t, detail()["performance_ceiling_override"])
}
