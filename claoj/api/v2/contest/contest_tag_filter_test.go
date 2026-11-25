package contest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
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

func setupTagFilterDB(t *testing.T) {
	t.Helper()
	old := db.DB
	t.Cleanup(func() { db.DB = old })
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)
	sqlDB, err := database.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, database.AutoMigrate(&models.Contest{}, &models.ContestTag{}))
	db.DB = database
}

func tagFilterContest(t *testing.T, key string, visible bool) models.Contest {
	t.Helper()
	now := time.Now()
	ct := models.Contest{Key: key, Name: key, StartTime: now, EndTime: now.Add(time.Hour), IsVisible: visible}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&ct).Error)
	return ct
}

func tagKeys(t *testing.T, target string) (keys []string, total int) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Handle(http.MethodGet, "/contests", ContestList)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, target, nil))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var body struct {
		Results []struct {
			Key string `json:"key"`
		} `json:"results"`
		Total int `json:"total"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	for _, r := range body.Results {
		keys = append(keys, r.Key)
	}
	return keys, body.Total
}

// v1's /contests/tag/<name> page lists exactly the tag's contests (and, as with
// every other list filter, never a contest the viewer may not see).
func TestContestList_TagFilterByNameAndID(t *testing.T) {
	setupTagFilterDB(t)

	tagged := tagFilterContest(t, "tagged", true)
	plain := tagFilterContest(t, "plain", true)
	hidden := tagFilterContest(t, "hidden", false)

	tag := models.ContestTag{Name: "open-cup", Color: "#123456"}
	require.NoError(t, db.DB.Create(&tag).Error)
	require.NoError(t, db.DB.Exec("INSERT INTO judge_contest_tags (contest_id, contesttag_id) VALUES (?, ?), (?, ?)",
		tagged.ID, tag.ID, hidden.ID, tag.ID).Error)
	_ = plain

	keys, total := tagKeys(t, "/contests?tag=open-cup")
	require.Equal(t, []string{"tagged"}, keys)
	require.Equal(t, 1, total)

	// Numeric id resolves to the same tag.
	keys, _ = tagKeys(t, "/contests?tag="+strconv.FormatUint(uint64(tag.ID), 10))
	require.Equal(t, []string{"tagged"}, keys)

	// An unknown tag matches nothing instead of silently listing everything.
	keys, total = tagKeys(t, "/contests?tag=nope")
	require.Empty(t, keys)
	require.Zero(t, total)

	// The tag filter composes with search.
	keys, _ = tagKeys(t, "/contests?tag=open-cup&search=tag")
	require.Equal(t, []string{"tagged"}, keys)
	keys, _ = tagKeys(t, "/contests?tag=open-cup&search=plain")
	require.Empty(t, keys)
}