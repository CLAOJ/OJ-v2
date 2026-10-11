package v2

import (
	"net/http"
	"testing"
	"time"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func setupSubmissionDeleteDB(t *testing.T) {
	t.Helper()
	old := db.DB
	t.Cleanup(func() { db.DB = old })
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)
	sqlDB, err := database.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, database.AutoMigrate(&models.Submission{}))
	db.DB = database
}

func TestAdminSubmissionDelete_MapsServiceErrors(t *testing.T) {
	setupSubmissionDeleteDB(t)
	queued := models.Submission{UserID: 1, ProblemID: 1, LanguageID: 1, Date: time.Now(), Status: "QU"}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&queued).Error)

	router := authedRouter(1, http.MethodDelete, "/admin/submission/:id", AdminSubmissionDelete)

	require.Equal(t, http.StatusNotFound, doRequest(router, http.MethodDelete, "/admin/submission/999", "").Code)
	require.Equal(t, http.StatusBadRequest, doRequest(router, http.MethodDelete, "/admin/submission/abc", "").Code)

	w := doRequest(router, http.MethodDelete, "/admin/submission/1", "")
	require.Equal(t, http.StatusConflict, w.Code, w.Body.String())
	require.Contains(t, w.Body.String(), "abort it first")

	var n int64
	require.NoError(t, db.DB.Model(&models.Submission{}).Count(&n).Error)
	require.Equal(t, int64(1), n)
}
