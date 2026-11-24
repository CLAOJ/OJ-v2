package v2

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func setupLanguageDB(t *testing.T) {
	t.Helper()
	old := db.DB
	t.Cleanup(func() { db.DB = old })
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)
	sqlDB, err := database.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, database.AutoMigrate(
		&models.AuthUser{}, &models.Profile{}, &models.Language{},
		&models.ProblemGroup{}, &models.ProblemType{}, &models.Problem{},
		&models.Submission{},
		&models.DjangoContentType{}, &models.AuthPermission{}, &models.AuthUserPermission{},
		&models.AuthGroup{}, &models.AuthUserGroup{}, &models.AuthGroupPermission{},
	))
	db.DB = database
}

func newLanguageUser(t *testing.T, username string, superuser bool) models.AuthUser {
	t.Helper()
	u := models.AuthUser{Username: username, IsActive: true, IsStaff: true, IsSuperuser: superuser}
	require.NoError(t, db.DB.Create(&u).Error)
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&models.Profile{UserID: u.ID, Timezone: "UTC"}).Error)
	return u
}

func grantLanguagePerm(t *testing.T, userID uint, codename string) {
	t.Helper()
	ct := models.DjangoContentType{AppLabel: "judge", Model: "language"}
	require.NoError(t, db.DB.Create(&ct).Error)
	perm := models.AuthPermission{Name: codename, ContentTypeID: ct.ID, Codename: codename}
	require.NoError(t, db.DB.Create(&perm).Error)
	require.NoError(t, db.DB.Create(&models.AuthUserPermission{UserID: userID, PermissionID: perm.ID}).Error)
}

func seedLanguage(t *testing.T, key string) models.Language {
	t.Helper()
	lang := models.Language{Key: key, Name: key, CommonName: key, Template: "// " + key, Description: key + " description"}
	require.NoError(t, db.DB.Create(&lang).Error)
	return lang
}

func seedProblems(t *testing.T, codes ...string) {
	t.Helper()
	for _, code := range codes {
		p := models.Problem{Code: code, Name: code}
		require.NoError(t, db.DB.Omit(clause.Associations).Create(&p).Error)
	}
}

// allowedCount counts problems the language is an allowed language of.
func allowedCount(t *testing.T, languageID uint) int64 {
	t.Helper()
	var n int64
	require.NoError(t, db.DB.Table("judge_problem_allowed_languages").
		Where("language_id = ?", languageID).Count(&n).Error)
	return n
}

func TestAdminLanguageList_IncludesTemplateAndDescription(t *testing.T) {
	setupLanguageDB(t)
	seedLanguage(t, "CPP17")
	router := authedRouter(1, http.MethodGet, "/admin/languages", AdminLanguageList)

	w := doRequest(router, http.MethodGet, "/admin/languages", "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Contains(t, w.Body.String(), `"template":"// CPP17"`)
	require.Contains(t, w.Body.String(), `"description":"CPP17 description"`)
}

func TestAdminLanguageUpdate_NeedsChangePermission(t *testing.T) {
	setupLanguageDB(t)
	lang := seedLanguage(t, "PY3")
	staff := newLanguageUser(t, "staff", false)
	granted := newLanguageUser(t, "granted", false)
	grantLanguagePerm(t, granted.ID, "change_language")

	denied := authedRouter(staff.ID, http.MethodPatch, "/admin/language/:id", AdminLanguageUpdate)
	w := doRequest(denied, http.MethodPatch, "/admin/language/1", `{"name":"Python"}`)
	require.Equal(t, http.StatusForbidden, w.Code, w.Body.String())

	allowed := authedRouter(granted.ID, http.MethodPatch, "/admin/language/:id", AdminLanguageUpdate)
	w = doRequest(allowed, http.MethodPatch, "/admin/language/1", `{"name":"Python"}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var got models.Language
	require.NoError(t, db.DB.First(&got, lang.ID).Error)
	require.Equal(t, "Python", got.Name)

	// Superuser bypass still holds.
	root := newLanguageUser(t, "root", true)
	super := authedRouter(root.ID, http.MethodPatch, "/admin/language/:id", AdminLanguageUpdate)
	w = doRequest(super, http.MethodPatch, "/admin/language/1", `{"name":"Python 3"}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
}

func TestAdminLanguageDelete_NeedsDeletePermission(t *testing.T) {
	setupLanguageDB(t)
	seedLanguage(t, "PY3")
	staff := newLanguageUser(t, "staff", false)
	granted := newLanguageUser(t, "granted", false)
	grantLanguagePerm(t, granted.ID, "delete_language")

	denied := authedRouter(staff.ID, http.MethodDelete, "/admin/language/:id", AdminLanguageDelete)
	require.Equal(t, http.StatusForbidden, doRequest(denied, http.MethodDelete, "/admin/language/1", "").Code)

	allowed := authedRouter(granted.ID, http.MethodDelete, "/admin/language/:id", AdminLanguageDelete)
	require.Equal(t, http.StatusOK, doRequest(allowed, http.MethodDelete, "/admin/language/1", "").Code)

	var n int64
	require.NoError(t, db.DB.Model(&models.Language{}).Count(&n).Error)
	require.Equal(t, int64(0), n)
}

func TestAdminLanguageCreate_IncludeInProblemAttachesToAllProblems(t *testing.T) {
	setupLanguageDB(t)
	seedProblems(t, "aplusb", "bplusc", "cplusd")
	root := newLanguageUser(t, "root", true)
	router := authedRouter(root.ID, http.MethodPost, "/admin/languages", AdminLanguageCreate)

	w := doRequest(router, http.MethodPost, "/admin/languages",
		`{"key":"PY3","name":"Python 3","common_name":"Python","ace":"python","pygments":"python","extension":"py","include_in_problem":true}`)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	var lang models.Language
	require.NoError(t, db.DB.Where("`key` = ?", "PY3").First(&lang).Error)
	require.True(t, lang.IncludeInProblem)
	require.Equal(t, int64(3), allowedCount(t, lang.ID))
}

func TestAdminLanguageCreate_WithoutIncludeInProblemAttachesNothing(t *testing.T) {
	setupLanguageDB(t)
	seedProblems(t, "aplusb")
	root := newLanguageUser(t, "root", true)
	router := authedRouter(root.ID, http.MethodPost, "/admin/languages", AdminLanguageCreate)

	w := doRequest(router, http.MethodPost, "/admin/languages",
		`{"key":"PY3","name":"Python 3","common_name":"Python","ace":"python","pygments":"python","extension":"py"}`)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	var lang models.Language
	require.NoError(t, db.DB.Where("`key` = ?", "PY3").First(&lang).Error)
	require.Equal(t, int64(0), allowedCount(t, lang.ID))
}

func TestAdminLanguageUpdate_IncludeInProblemIsIdempotentAndNeverDetaches(t *testing.T) {
	setupLanguageDB(t)
	seedProblems(t, "aplusb", "bplusc")
	lang := seedLanguage(t, "PY3")
	root := newLanguageUser(t, "root", true)
	router := authedRouter(root.ID, http.MethodPatch, "/admin/language/:id", AdminLanguageUpdate)

	for i := 0; i < 2; i++ {
		w := doRequest(router, http.MethodPatch, "/admin/language/1", `{"include_in_problem":true}`)
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		require.Equal(t, int64(2), allowedCount(t, lang.ID), "re-applying must not duplicate rows")
	}

	// Clearing the flag leaves existing rows alone (v1 only adds).
	w := doRequest(router, http.MethodPatch, "/admin/language/1", `{"include_in_problem":false}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Equal(t, int64(2), allowedCount(t, lang.ID))

	var got models.Language
	require.NoError(t, db.DB.First(&got, lang.ID).Error)
	require.False(t, got.IncludeInProblem)
}

// The row update and the attach are one unit of work: a failed attach must
// leave the language exactly as it was.
func TestAdminLanguageUpdate_AttachFailureRollsBackUpdate(t *testing.T) {
	setupLanguageDB(t)
	seedProblems(t, "aplusb")
	lang := seedLanguage(t, "PY3")
	root := newLanguageUser(t, "root", true)
	router := authedRouter(root.ID, http.MethodPatch, "/admin/language/:id", AdminLanguageUpdate)
	require.NoError(t, db.DB.Migrator().DropTable("judge_problem_allowed_languages"))

	w := doRequest(router, http.MethodPatch, "/admin/language/1", `{"name":"Renamed","include_in_problem":true}`)
	require.Equal(t, http.StatusInternalServerError, w.Code, w.Body.String())

	var got models.Language
	require.NoError(t, db.DB.First(&got, lang.ID).Error)
	require.Equal(t, "PY3", got.Name, "the update rolls back with the failed attach")
	require.False(t, got.IncludeInProblem)
}

func TestAdminLanguageUpdate_ReturnsLanguageKey(t *testing.T) {
	setupLanguageDB(t)
	seedLanguage(t, "PY3")
	root := newLanguageUser(t, "root", true)
	router := authedRouter(root.ID, http.MethodPatch, "/admin/language/:id", AdminLanguageUpdate)

	w := doRequest(router, http.MethodPatch, "/admin/language/1", `{"template":"print(1)"}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, "PY3", body["language"].(map[string]any)["key"])

	var got models.Language
	require.NoError(t, db.DB.First(&got, 1).Error)
	require.Equal(t, "print(1)", got.Template)
}
