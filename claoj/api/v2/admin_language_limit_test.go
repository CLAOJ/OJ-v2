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

func setupLanguageLimitDB(t *testing.T) {
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
		&models.ProblemGroup{}, &models.ProblemType{}, &models.Problem{}, &models.LanguageLimit{},
	))
	db.DB = database
}

// The admin table reads flat snake_case fields, so the list must not hand back
// the raw model (Go field names) nor the whole Problem incl. its description.
func TestAdminLanguageLimitList_ReturnsSlimSnakeCaseRows(t *testing.T) {
	setupLanguageLimitDB(t)

	problem := models.Problem{
		Code: "sum", Name: "A Plus B", Description: "a very long statement",
		TimeLimit: 1, MemoryLimit: 262144, Points: 100,
	}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&problem).Error)
	lang := models.Language{Key: "CPP17", Name: "C++17"}
	require.NoError(t, db.DB.Create(&lang).Error)
	require.NoError(t, db.DB.Create(&models.LanguageLimit{
		ProblemID: problem.ID, LanguageID: lang.ID, TimeLimit: 1.5, MemoryLimit: 262144,
	}).Error)

	router := authedRouter(1, http.MethodGet, "/admin/language-limits", AdminLanguageLimitList)
	w := doRequest(router, http.MethodGet, "/admin/language-limits?page=1&page_size=20", "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var body struct {
		Data  []map[string]any `json:"data"`
		Total int64            `json:"total"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, int64(1), body.Total)
	require.Len(t, body.Data, 1)

	row := body.Data[0]
	require.EqualValues(t, problem.ID, row["problem_id"])
	require.Equal(t, "sum", row["problem_code"])
	require.Equal(t, "A Plus B", row["problem_name"])
	require.EqualValues(t, lang.ID, row["language_id"])
	require.Equal(t, "CPP17", row["language_key"])
	require.Equal(t, "C++17", row["language_name"])
	require.EqualValues(t, 1.5, row["time_limit"])
	require.EqualValues(t, 262144, row["memory_limit"])

	_, hasProblem := row["problem"]
	_, hasDescription := row["description"]
	require.False(t, hasProblem, "the full Problem object must not be embedded")
	require.False(t, hasDescription, "problem description must not leak into the list")
}