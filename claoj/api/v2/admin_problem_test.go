package v2

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func itoa(id uint) string { return strconv.Itoa(int(id)) }

// setupAdminProblemDB seeds the tables the admin problem create/update/detail
// handlers touch, including the v1 fields Task 1 exposes.
func setupAdminProblemDB(t *testing.T) models.ProblemGroup {
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
		&models.License{}, &models.ProblemGroup{}, &models.ProblemType{}, &models.Problem{},
	))
	db.DB = database

	group := models.ProblemGroup{Name: "g1"}
	require.NoError(t, db.DB.Create(&group).Error)
	return group
}

func createAdminProblemTestUser(t *testing.T, username string) models.Profile {
	t.Helper()
	user := models.AuthUser{Username: username, IsActive: true}
	require.NoError(t, db.DB.Create(&user).Error)
	profile := models.Profile{UserID: user.ID, Timezone: "UTC"}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&profile).Error)
	return profile
}

func TestAdminProblemCreate_V1Fields(t *testing.T) {
	group := setupAdminProblemDB(t)
	author := createAdminProblemTestUser(t, "author")
	curator := createAdminProblemTestUser(t, "curator")
	tester := createAdminProblemTestUser(t, "tester")
	banned := createAdminProblemTestUser(t, "banned")
	suggester := createAdminProblemTestUser(t, "suggester")

	org := models.Organization{Name: "org", Slug: "org", ShortName: "org", CreationDate: time.Now()}
	require.NoError(t, db.DB.Omit("Admins", "Members").Create(&org).Error)
	license := models.License{Key: "cc", Link: "https://example.com/cc", Name: "CC"}
	require.NoError(t, db.DB.Create(&license).Error)

	body := `{
		"code":"full1","name":"Full Problem","description":"<p>body</p>",
		"points":100,"time_limit":1.5,"memory_limit":262144,"group_id":` + itoa(group.ID) + `,
		"source":"VOI 2026","date":"2026-03-04T05:06:07Z","license_id":` + itoa(license.ID) + `,
		"og_image":"https://example.com/og.png","summary":"short summary",
		"short_circuit":false,"is_full_markup":true,
		"submission_source_visibility_mode":"S","testcase_visibility_mode":"O",
		"is_organization_private":true,
		"organization_ids":[` + itoa(org.ID) + `],"banned_user_ids":[` + itoa(banned.ID) + `],
		"suggester_id":` + itoa(suggester.ID) + `,
		"author_ids":[` + itoa(author.ID) + `],"curator_ids":[` + itoa(curator.ID) + `],
		"tester_ids":[` + itoa(tester.ID) + `]
	}`

	w := doRequest(authedRouter(1, http.MethodPost, "/admin/problems", AdminProblemCreate),
		http.MethodPost, "/admin/problems", body)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	w = doRequest(authedRouter(1, http.MethodGet, "/admin/problem/:code", AdminProblemDetail),
		http.MethodGet, "/admin/problem/full1", "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var got map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	require.Equal(t, "VOI 2026", got["source"])
	require.Equal(t, "https://example.com/og.png", got["og_image"])
	require.Equal(t, "short summary", got["summary"])
	require.EqualValues(t, license.ID, got["license_id"])
	require.Equal(t, true, got["short_circuit"], "derived from partial=false; the client's false is ignored")
	require.Equal(t, true, got["is_full_markup"])
	require.Equal(t, "S", got["submission_source_visibility_mode"])
	require.Equal(t, "O", got["testcase_visibility_mode"])
	require.Equal(t, true, got["is_organization_private"])
	require.Equal(t, []any{float64(org.ID)}, got["organization_ids"])
	require.Equal(t, []any{float64(banned.ID)}, got["banned_user_ids"])
	require.EqualValues(t, suggester.ID, got["suggester_id"])
	require.Equal(t, []any{float64(author.ID)}, got["author_ids"])
	require.Equal(t, []any{float64(curator.ID)}, got["curator_ids"])
	require.Equal(t, []any{float64(tester.ID)}, got["tester_ids"])

	bannedUsers, ok := got["banned_users"].([]any)
	require.True(t, ok)
	require.Len(t, bannedUsers, 1)
	require.Equal(t, "banned", bannedUsers[0].(map[string]any)["username"])
}

func TestAdminProblemCreate_InvalidVisibilityMode(t *testing.T) {
	group := setupAdminProblemDB(t)

	body := `{"code":"bad1","name":"Bad","description":"d","points":100,"time_limit":1,
		"memory_limit":262144,"group_id":` + itoa(group.ID) + `,"testcase_visibility_mode":"B"}`
	w := doRequest(authedRouter(1, http.MethodPost, "/admin/problems", AdminProblemCreate),
		http.MethodPost, "/admin/problems", body)
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())

	var count int64
	require.NoError(t, db.DB.Model(&models.Problem{}).Count(&count).Error)
	require.Equal(t, int64(0), count)
}

func TestAdminProblemUpdate_V1FieldsAndRelations(t *testing.T) {
	group := setupAdminProblemDB(t)
	problem := models.Problem{Code: "upd1", Name: "Before", GroupID: group.ID}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&problem).Error)
	banned := createAdminProblemTestUser(t, "troll")
	org := models.Organization{Name: "org", Slug: "org", ShortName: "org", CreationDate: time.Now()}
	require.NoError(t, db.DB.Omit("Admins", "Members").Create(&org).Error)

	body := `{"source":"ICPC","og_image":"https://example.com/og.png","summary":"one line",
		"short_circuit":true,"is_full_markup":true,
		"submission_source_visibility_mode":"A","testcase_visibility_mode":"C",
		"is_organization_private":true,
		"organization_ids":[` + itoa(org.ID) + `],"banned_user_ids":[` + itoa(banned.ID) + `]}`

	w := doRequest(authedRouter(1, http.MethodPatch, "/admin/problem/:code", AdminProblemUpdate),
		http.MethodPatch, "/admin/problem/upd1", body)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var got models.Problem
	require.NoError(t, db.DB.First(&got, problem.ID).Error)
	require.Equal(t, "ICPC", got.Source)
	require.Equal(t, "https://example.com/og.png", got.OgImage)
	require.Equal(t, "one line", got.Summary)
	require.False(t, got.ShortCircuit, "short_circuit is derived, never client input")
	require.True(t, got.IsFullMarkup)
	require.Equal(t, "A", got.SubmissionSourceVisibilityMode)
	require.Equal(t, "C", got.TestcaseVisibilityMode)
	require.True(t, got.IsOrganizationPrivate)

	var orgIDs, bannedIDs []uint
	require.NoError(t, db.DB.Table("judge_problem_organizations").Where("problem_id = ?", problem.ID).Pluck("organization_id", &orgIDs).Error)
	require.NoError(t, db.DB.Table("judge_problem_banned_users").Where("problem_id = ?", problem.ID).Pluck("profile_id", &bannedIDs).Error)
	require.Equal(t, []uint{org.ID}, orgIDs)
	require.Equal(t, []uint{banned.ID}, bannedIDs)

	w = doRequest(authedRouter(1, http.MethodPatch, "/admin/problem/:code", AdminProblemUpdate),
		http.MethodPatch, "/admin/problem/upd1", `{"submission_source_visibility_mode":"Q"}`)
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())

	require.NoError(t, db.DB.First(&got, problem.ID).Error)
	require.Equal(t, "A", got.SubmissionSourceVisibilityMode, "a rejected update must not change the stored mode")
}

func TestAdminProblemShortCircuitIsDerivedFromPartial(t *testing.T) {
	group := setupAdminProblemDB(t)
	problem := models.Problem{Code: "sc1", Name: "SC", GroupID: group.ID}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&problem).Error)

	patch := func(body string) models.Problem {
		w := doRequest(authedRouter(1, http.MethodPatch, "/admin/problem/:code", AdminProblemUpdate),
			http.MethodPatch, "/admin/problem/sc1", body)
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		var got models.Problem
		require.NoError(t, db.DB.First(&got, problem.ID).Error)
		return got
	}

	got := patch(`{"partial":true,"short_circuit":true}`)
	require.False(t, got.ShortCircuit, "partial=true forces short_circuit=false")

	got = patch(`{"partial":false,"short_circuit":false}`)
	require.True(t, got.ShortCircuit, "partial=false forces short_circuit=true")
}

func TestAdminProblemCreate_UnknownOrganizationIs400AndAtomic(t *testing.T) {
	group := setupAdminProblemDB(t)

	body := `{"code":"orphan1","name":"Orphan","description":"d","points":100,"time_limit":1,
		"memory_limit":262144,"group_id":` + itoa(group.ID) + `,"organization_ids":[424242]}`
	w := doRequest(authedRouter(1, http.MethodPost, "/admin/problems", AdminProblemCreate),
		http.MethodPost, "/admin/problems", body)
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())

	var count int64
	require.NoError(t, db.DB.Model(&models.Problem{}).Where("code = ?", "orphan1").Count(&count).Error)
	require.Equal(t, int64(0), count, "the rejected relation must not leave an orphan row")
}

func TestAdminProblemCreate_UnknownLicenseIs400(t *testing.T) {
	group := setupAdminProblemDB(t)

	body := `{"code":"lica","name":"L","description":"d","points":100,"time_limit":1,
		"memory_limit":262144,"group_id":` + itoa(group.ID) + `,"license_id":424242}`
	w := doRequest(authedRouter(1, http.MethodPost, "/admin/problems", AdminProblemCreate),
		http.MethodPost, "/admin/problems", body)
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
}

func TestAdminProblemUpdate_ClearNullableFields(t *testing.T) {
	group := setupAdminProblemDB(t)
	date := time.Date(2026, 7, 8, 9, 10, 11, 0, time.UTC)
	license := models.License{Key: "mit", Link: "https://example.com/mit", Name: "MIT"}
	require.NoError(t, db.DB.Create(&license).Error)
	suggester := createAdminProblemTestUser(t, "suggester")
	problem := models.Problem{
		Code: "clr1", Name: "Clear", GroupID: group.ID,
		Date: &date, LicenseID: &license.ID, SuggesterID: &suggester.ID,
	}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&problem).Error)

	w := doRequest(authedRouter(1, http.MethodPatch, "/admin/problem/:code", AdminProblemUpdate),
		http.MethodPatch, "/admin/problem/clr1",
		`{"clear_date":true,"clear_license_id":true,"clear_suggester_id":true}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var got models.Problem
	require.NoError(t, db.DB.First(&got, problem.ID).Error)
	require.Nil(t, got.Date)
	require.Nil(t, got.LicenseID)
	require.Nil(t, got.SuggesterID)
}
