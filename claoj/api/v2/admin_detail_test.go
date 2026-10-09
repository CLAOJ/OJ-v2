package v2

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// setupAdminDetailDB migrates everything the problem and contest admin
// detail/update handlers touch. judge_contestproblem is recreated because
// Contest's many2many leaves a bare join table without id/points.
func setupAdminDetailDB(t *testing.T) {
	t.Helper()
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := database.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, database.AutoMigrate(
		&models.AuthUser{}, &models.Profile{}, &models.Organization{}, &models.Language{},
		&models.ProblemGroup{}, &models.ProblemType{}, &models.Problem{}, &models.ContestTag{},
		&models.Contest{}, &models.ContestParticipation{}, &models.Submission{}, &models.ContestSubmission{},
	))
	require.NoError(t, database.Migrator().DropTable("judge_contestproblem"))
	require.NoError(t, database.Migrator().CreateTable(&models.ContestProblem{}))
	db.DB = database
	problemService = nil
	contestService = nil
}

func TestAdminProblemDetail_IsFlatSnakeCase(t *testing.T) {
	setupAdminDetailDB(t)
	u := models.AuthUser{Username: "alice", IsActive: true}
	require.NoError(t, db.DB.Create(&u).Error)
	prof := models.Profile{UserID: u.ID, Timezone: "UTC"}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&prof).Error)
	p := models.Problem{Code: "aplusb", Name: "A+B", PdfURL: "/pdf/aplusb.pdf"}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&p).Error)
	require.NoError(t, db.DB.Model(&p).Association("Authors").Append(&prof))

	router := authedRouter(u.ID, http.MethodGet, "/admin/problem/:code", AdminProblemDetail)
	w := doRequest(router, http.MethodGet, "/admin/problem/aplusb", "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, "aplusb", body["code"])
	require.Equal(t, "/pdf/aplusb.pdf", body["pdf_url"])
	require.Equal(t, []any{float64(prof.ID)}, body["author_ids"])
	require.Equal(t, []any{map[string]any{"id": float64(prof.ID), "username": "alice"}}, body["authors"])
	require.Equal(t, []any{}, body["type_ids"])
	require.NotContains(t, body, "problem", "detail must not be nested")
}

func TestAdminProblemUpdate_UnknownAuthorIs400(t *testing.T) {
	setupAdminDetailDB(t)
	p := models.Problem{Code: "p2", Name: "P2"}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&p).Error)

	router := authedRouter(1, http.MethodPatch, "/admin/problem/:code", AdminProblemUpdate)
	w := doRequest(router, http.MethodPatch, "/admin/problem/p2", `{"author_ids":[777]}`)
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
}

var _ gin.HandlerFunc = AdminProblemDetail
