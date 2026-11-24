package v2

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

type problemListFixture struct {
	group  models.ProblemGroup
	ptype  models.ProblemType
	admin  models.AuthUser
	editor models.Solution
}

// setupProblemListDB seeds two public problems (one typed, one with a still
// unpublished editorial), an accepted suggestion and an unsuggested hidden
// problem.
func setupProblemListDB(t *testing.T) problemListFixture {
	t.Helper()
	old := db.DB
	t.Cleanup(func() { db.DB = old })
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)
	sqlDB, err := database.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, database.AutoMigrate(
		&models.AuthUser{}, &models.AuthGroup{}, &models.DjangoContentType{},
		&models.AuthPermission{}, &models.AuthUserGroup{}, &models.AuthGroupPermission{},
		&models.AuthUserPermission{}, &models.Profile{}, &models.Organization{},
		&models.ProblemGroup{}, &models.ProblemType{}, &models.Problem{},
		&models.Solution{},
	))
	db.DB = database

	group := models.ProblemGroup{Name: "graphs", FullName: "Graph Theory"}
	require.NoError(t, db.DB.Create(&group).Error)
	ptype := models.ProblemType{Name: "dp", FullName: "Dynamic Programming"}
	require.NoError(t, db.DB.Create(&ptype).Error)

	newProblem := func(code string, public bool, suggesterID *uint) models.Problem {
		p := models.Problem{
			Code: code, Name: code, Description: "d", GroupID: group.ID,
			TimeLimit: 1, MemoryLimit: 256, Points: 100, IsPublic: public,
			SuggesterID: suggesterID,
		}
		require.NoError(t, db.DB.Omit(clause.Associations).Create(&p).Error)
		return p
	}

	typed := newProblem("typed", true, nil)
	require.NoError(t, db.DB.Model(&typed).Association("Types").Append(&ptype))
	newProblem("plain", true, nil)

	// Publishes now (NULL publish_on = published immediately, like
	// ProblemSolutionExists); "plain" keeps a future-dated editorial.
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&models.Solution{
		ProblemID: typed.ID, Content: "", IsPublic: true,
	}).Error)
	future := time.Now().Add(24 * time.Hour)
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&models.Solution{
		ProblemID: 2, Content: "", IsPublic: true, PublishOn: &future,
	}).Error)

	// Admin: superusers hold every permission, so no perm rows are needed.
	admin := models.AuthUser{Username: "root", IsActive: true, IsSuperuser: true}
	require.NoError(t, db.DB.Create(&admin).Error)
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&models.Profile{UserID: admin.ID, Timezone: "UTC"}).Error)

	// Suggester profile, then one accepted suggestion + one unsuggested hidden
	// problem (must never show up in either list).
	su := models.AuthUser{Username: "sug", IsActive: true}
	require.NoError(t, db.DB.Create(&su).Error)
	sp := models.Profile{UserID: su.ID, Timezone: "UTC"}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&sp).Error)
	newProblem("sugg", false, &sp.ID)
	newProblem("hidden", false, nil)

	return problemListFixture{group: group, ptype: ptype, admin: admin}
}

func getProblemListJSON(t *testing.T, handler gin.HandlerFunc, target string, userID ...uint) (int, map[string]any) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	if len(userID) > 0 {
		id := userID[0]
		router.Use(func(c *gin.Context) {
			c.Set("user_id", id)
			c.Next()
		})
	}
	router.GET("/problems", handler)
	router.GET("/problem-groups", ProblemGroupList)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, target, nil))
	var body map[string]any
	if w.Code == http.StatusOK {
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	}
	return w.Code, body
}

func codes(t *testing.T, body map[string]any) []string {
	t.Helper()
	rows, ok := body["data"].([]any)
	require.True(t, ok)
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i] = r.(map[string]any)["code"].(string)
	}
	return out
}

func TestProblemList_TypeFilter(t *testing.T) {
	fx := setupProblemListDB(t)

	code, body := getProblemListJSON(t, ProblemList, "/problems?type="+itoa(fx.ptype.ID))
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, []string{"typed"}, codes(t, body))
	require.EqualValues(t, 1, body["total"])

	// Comma-separated ids are equivalent to repeated params.
	_, body = getProblemListJSON(t, ProblemList, "/problems?type="+itoa(fx.ptype.ID)+",999")
	require.Equal(t, []string{"typed"}, codes(t, body))
}

func TestProblemList_HasEditorialFilter(t *testing.T) {
	fx := setupProblemListDB(t)

	_, body := getProblemListJSON(t, ProblemList, "/problems?has_editorial=1")
	require.Equal(t, []string{"typed"}, codes(t, body))

	_, body = getProblemListJSON(t, ProblemList, "/problems?has_editorial=0")
	require.Equal(t, []string{"plain"}, codes(t, body))

	// Filters combine, and the total reflects the combination.
	_, body = getProblemListJSON(t, ProblemList, "/problems?type="+itoa(fx.ptype.ID)+"&has_editorial=0")
	require.Empty(t, codes(t, body))
	require.EqualValues(t, 0, body["total"])
}

func TestProblemList_GroupFilterByID(t *testing.T) {
	fx := setupProblemListDB(t)

	_, body := getProblemListJSON(t, ProblemList, "/problems?group="+itoa(fx.group.ID))
	require.ElementsMatch(t, []string{"typed", "plain"}, codes(t, body))
	require.EqualValues(t, 2, body["total"])
}

func TestSuggestProblemList_PermissionGate(t *testing.T) {
	fx := setupProblemListDB(t)

	code, _ := getProblemListJSON(t, SuggestProblemList, "/problems")
	require.Equal(t, http.StatusForbidden, code)

	code, body := getProblemListJSON(t, SuggestProblemList, "/problems", fx.admin.ID)
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, []string{"sugg"}, codes(t, body))
	require.EqualValues(t, 1, body["total"])
	require.Equal(t, "sug", body["data"].([]any)[0].(map[string]any)["suggester"])
}

func TestProblemGroupList_PublicShape(t *testing.T) {
	fx := setupProblemListDB(t)

	code, body := getProblemListJSON(t, ProblemGroupList, "/problem-groups")
	require.Equal(t, http.StatusOK, code)
	rows := body["data"].([]any)
	require.Len(t, rows, 1)
	require.Equal(t, map[string]any{
		"id":        float64(fx.group.ID),
		"name":      "graphs",
		"full_name": "Graph Theory",
	}, rows[0])
}
