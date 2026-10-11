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

func setupJudgeCrudDB(t *testing.T) {
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
		&models.ProblemGroup{}, &models.ProblemType{}, &models.Problem{},
		&models.Judge{}, &models.RuntimeVersion{}, &models.Submission{},
	))
	db.DB = database
}

func seedJudge(t *testing.T, name string, online bool) models.Judge {
	t.Helper()
	j := models.Judge{Name: name, AuthKey: "key-" + name, Created: time.Now(), Online: online}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&j).Error)
	return j
}

func TestAdminJudgeCreate_GeneratesKey(t *testing.T) {
	setupJudgeCrudDB(t)
	router := authedRouter(1, http.MethodPost, "/admin/judges", AdminJudgeCreate)
	w := doRequest(router, http.MethodPost, "/admin/judges", `{"name":"judge1","description":"rack 3"}`)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, "judge1", body["name"])
	key, _ := body["auth_key"].(string)
	require.Len(t, key, 64)

	var j models.Judge
	require.NoError(t, db.DB.Where("name = ?", "judge1").First(&j).Error)
	require.Equal(t, key, j.AuthKey)
	require.Equal(t, "rack 3", j.Description)
	require.False(t, j.Created.IsZero())
	require.False(t, j.Online)
	require.Equal(t, float64(j.ID), body["id"])
}

func TestAdminJudgeCreate_KeepsSuppliedKey(t *testing.T) {
	setupJudgeCrudDB(t)
	router := authedRouter(1, http.MethodPost, "/admin/judges", AdminJudgeCreate)
	w := doRequest(router, http.MethodPost, "/admin/judges", `{"name":"judge2","auth_key":"my-secret-key"}`)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	var j models.Judge
	require.NoError(t, db.DB.Where("name = ?", "judge2").First(&j).Error)
	require.Equal(t, "my-secret-key", j.AuthKey)
}

func TestAdminJudgeCreate_Rejects(t *testing.T) {
	setupJudgeCrudDB(t)
	seedJudge(t, "taken", false)
	router := authedRouter(1, http.MethodPost, "/admin/judges", AdminJudgeCreate)

	cases := []struct {
		name string
		body string
		want int
	}{
		{"missing name", `{}`, http.StatusBadRequest},
		{"blank name", `{"name":"   "}`, http.StatusBadRequest},
		{"name too long", `{"name":"` + strings.Repeat("a", 51) + `"}`, http.StatusBadRequest},
		{"key too long", `{"name":"ok","auth_key":"` + strings.Repeat("k", 101) + `"}`, http.StatusBadRequest},
		{"duplicate name", `{"name":"taken"}`, http.StatusConflict},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := doRequest(router, http.MethodPost, "/admin/judges", tc.body)
			require.Equal(t, tc.want, w.Code, w.Body.String())
		})
	}

	var n int64
	require.NoError(t, db.DB.Model(&models.Judge{}).Count(&n).Error)
	require.Equal(t, int64(1), n, "a rejected create must not insert a row")
}

func TestAdminJudgeUpdate_RenameAndRekey(t *testing.T) {
	setupJudgeCrudDB(t)
	a := seedJudge(t, "a", false)
	seedJudge(t, "b", false)
	router := authedRouter(1, http.MethodPatch, "/admin/judge/:id", AdminJudgeUpdate)
	target := "/admin/judge/1"
	require.Equal(t, uint(1), a.ID)

	w := doRequest(router, http.MethodPatch, target, `{"name":"a2","auth_key":"newkey"}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var got models.Judge
	require.NoError(t, db.DB.First(&got, a.ID).Error)
	require.Equal(t, "a2", got.Name)
	require.Equal(t, "newkey", got.AuthKey)

	w = doRequest(router, http.MethodPatch, target, `{"name":"b"}`)
	require.Equal(t, http.StatusConflict, w.Code, w.Body.String())

	w = doRequest(router, http.MethodPatch, target, `{"auth_key":""}`)
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())

	w = doRequest(router, http.MethodPatch, target, `{"description":"only this"}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NoError(t, db.DB.First(&got, a.ID).Error)
	require.Equal(t, "a2", got.Name, "omitted fields stay unchanged")
	require.Equal(t, "newkey", got.AuthKey, "omitted fields stay unchanged")
	require.Equal(t, "only this", got.Description)
}

func TestAdminJudgeDelete_CascadesLikeDjango(t *testing.T) {
	setupJudgeCrudDB(t)
	j := seedJudge(t, "doomed", false)
	keep := seedJudge(t, "keep", false)

	lang := models.Language{Key: "py3", Name: "Python 3"}
	require.NoError(t, db.DB.Create(&lang).Error)
	p := models.Problem{Code: "p1", Name: "P1"}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&p).Error)
	for _, judge := range []models.Judge{j, keep} {
		judge := judge
		require.NoError(t, db.DB.Model(&judge).Association("Problems").Append(&p))
		require.NoError(t, db.DB.Model(&judge).Association("Runtimes").Append(&lang))
		require.NoError(t, db.DB.Omit(clause.Associations).Create(&models.RuntimeVersion{
			LanguageID: lang.ID, JudgeID: judge.ID, Name: "python", Version: "3.12",
		}).Error)
	}
	sub := models.Submission{UserID: 1, ProblemID: p.ID, LanguageID: lang.ID, Date: time.Now(), Status: "D", JudgedOnID: &j.ID}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&sub).Error)

	router := authedRouter(1, http.MethodDelete, "/admin/judge/:id", AdminJudgeDelete)
	w := doRequest(router, http.MethodDelete, "/admin/judge/1", "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	count := func(table, where string, arg any) int64 {
		var n int64
		require.NoError(t, db.DB.Table(table).Where(where, arg).Count(&n).Error)
		return n
	}
	require.Equal(t, int64(0), count("judge_judge", "id = ?", j.ID))
	require.Equal(t, int64(0), count("judge_runtimeversion", "judge_id = ?", j.ID))
	require.Equal(t, int64(0), count("judge_judge_problems", "judge_id = ?", j.ID))
	require.Equal(t, int64(0), count("judge_judge_runtimes", "judge_id = ?", j.ID))

	// The other judge keeps everything.
	require.Equal(t, int64(1), count("judge_judge", "id = ?", keep.ID))
	require.Equal(t, int64(1), count("judge_runtimeversion", "judge_id = ?", keep.ID))
	require.Equal(t, int64(1), count("judge_judge_problems", "judge_id = ?", keep.ID))
	require.Equal(t, int64(1), count("judge_judge_runtimes", "judge_id = ?", keep.ID))

	// Django: Submission.judged_on is SET_NULL, the submission survives.
	var gotSub models.Submission
	require.NoError(t, db.DB.First(&gotSub, sub.ID).Error)
	require.Nil(t, gotSub.JudgedOnID)
}

func TestAdminJudgeDelete_OnlineIs409(t *testing.T) {
	setupJudgeCrudDB(t)
	j := seedJudge(t, "busy", true)
	router := authedRouter(1, http.MethodDelete, "/admin/judge/:id", AdminJudgeDelete)
	w := doRequest(router, http.MethodDelete, "/admin/judge/1", "")
	require.Equal(t, http.StatusConflict, w.Code, w.Body.String())

	var n int64
	require.NoError(t, db.DB.Model(&models.Judge{}).Where("id = ?", j.ID).Count(&n).Error)
	require.Equal(t, int64(1), n)
}

func TestAdminJudgeDelete_Errors(t *testing.T) {
	setupJudgeCrudDB(t)
	router := authedRouter(1, http.MethodDelete, "/admin/judge/:id", AdminJudgeDelete)
	require.Equal(t, http.StatusNotFound, doRequest(router, http.MethodDelete, "/admin/judge/99", "").Code)
	require.Equal(t, http.StatusBadRequest, doRequest(router, http.MethodDelete, "/admin/judge/abc", "").Code)
}
