package contest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// The list endpoint used to return no row count, so a client asking for
// page_size=1000 silently got the first 100 rows and could not page further.
func TestContestList_ReturnsFilteredTotal(t *testing.T) {
	old := db.DB
	t.Cleanup(func() { db.DB = old })
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)
	sqlDB, err := database.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, database.AutoMigrate(&models.Contest{}, &models.ContestTag{}))
	db.DB = database

	now := time.Now()
	for i, key := range []string{"pub1", "pub2", "pub3"} {
		ct := models.Contest{
			Key: key, Name: "Public " + key,
			StartTime: now.Add(-time.Duration(i) * time.Hour), EndTime: now.Add(time.Hour),
			IsVisible: true,
		}
		require.NoError(t, db.DB.Omit(clause.Associations).Create(&ct).Error)
	}
	hidden := models.Contest{Key: "hidden", Name: "Hidden", StartTime: now, EndTime: now.Add(time.Hour), IsVisible: false}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&hidden).Error)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Handle(http.MethodGet, "/contests", ContestList)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/contests?page_size=2", nil))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var body struct {
		Results []struct {
			Key string `json:"key"`
		} `json:"results"`
		Total int `json:"total"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	// Three visible contests, one page of two rows: the total still counts all.
	require.Equal(t, 3, body.Total)
	require.Len(t, body.Results, 2)

	w = httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/contests?search=pub2", nil))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, 1, body.Total)
	require.Equal(t, "pub2", body.Results[0].Key)
}