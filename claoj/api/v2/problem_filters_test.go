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
	group2 models.ProblemGroup
	ptype  models.ProblemType
	ptype2 models.ProblemType
	admin  models.AuthUser
}

// setupProblemListDB seeds three public problems in one group (published,
// future-dated and undated editorials), one public problem in a second group,
// an accepted suggestion and an unsuggested hidden problem.
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
	group2 := models.ProblemGroup{Name: "geom", FullName: "Geometry"}
	require.NoError(t, db.DB.Create(&group2).Error)
	ptype := models.ProblemType{Name: "dp", FullName: "Dynamic Programming"}
	require.NoError(t, db.DB.Create(&ptype).Error)
	ptype2 := models.ProblemType{Name: "greedy", FullName: "Greedy"}
	require.NoError(t, db.DB.Create(&ptype2).Error)

	newProblem := func(code string, groupID uint, public bool, suggesterID *uint) models.Problem {
		p := models.Problem{
			Code: code, Name: code, Description: "d", GroupID: groupID,
			TimeLimit: 1, MemoryLimit: 256, Points: 100, IsPublic: public,
			SuggesterID: suggesterID,
		}
		require.NoError(t, db.DB.Omit(clause.Associations).Create(&p).Error)
		return p
	}

	// "typed" carries two types and is the only problem with a published
	// editorial; "plain" publishes tomorrow and "undated" has a NULL
	// publish_on, which the v1 rule treats as unpublished too.
	typed := newProblem("typed", group.ID, true, nil)
	require.NoError(t, db.DB.Model(&typed).Association("Types").Append([]models.ProblemType{ptype, ptype2}))
	plain := newProblem("plain", group.ID, true, nil)
	undated := newProblem("undated", group.ID, true, nil)
	newProblem("geo", group2.ID, true, nil)

	past := time.Now().Add(-time.Hour)
	future := time.Now().Add(24 * time.Hour)
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&models.Solution{
		ProblemID: typed.ID, Content: "", IsPublic: true, PublishOn: &past,
	}).Error)
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&models.Solution{
		ProblemID: plain.ID, Content: "", IsPublic: true, PublishOn: &future,
	}).Error)
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&models.Solution{
		ProblemID: undated.ID, Content: "", IsPublic: true,
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
	newProblem("sugg", group.ID, false, &sp.ID)
	newProblem("hidden", group.ID, false, nil)

	return problemListFixture{group: group, group2: group2, ptype: ptype, ptype2: ptype2, admin: admin}
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

	// Repeated params over a problem carrying both types: one row, total 1.
	_, body = getProblemListJSON(t, ProblemList, "/problems?type="+itoa(fx.ptype.ID)+"&type="+itoa(fx.ptype2.ID))
	require.Equal(t, []string{"typed"}, codes(t, body))
	require.EqualValues(t, 1, body["total"])

	// An all-invalid type filter narrows to nothing instead of widening.
	_, body = getProblemListJSON(t, ProblemList, "/problems?type=abc")
	require.Empty(t, codes(t, body))
	require.EqualValues(t, 0, body["total"])
}

func TestProblemList_HasEditorialFilter(t *testing.T) {
	fx := setupProblemListDB(t)

	// Only the past-dated editorial counts: a future publish_on and a NULL
	// publish_on are both unpublished.
	_, body := getProblemListJSON(t, ProblemList, "/problems?has_editorial=1")
	require.Equal(t, []string{"typed"}, codes(t, body))
	require.EqualValues(t, 1, body["total"])

	_, body = getProblemListJSON(t, ProblemList, "/problems?has_editorial=0")
	require.Equal(t, []string{"geo", "plain", "undated"}, codes(t, body))
	require.EqualValues(t, 3, body["total"])

	// Filters combine, and the total reflects the combination.
	_, body = getProblemListJSON(t, ProblemList, "/problems?type="+itoa(fx.ptype.ID)+"&has_editorial=0")
	require.Empty(t, codes(t, body))
	require.EqualValues(t, 0, body["total"])
}

// The editorial tab (exists) and the list filter must apply the same rule.
func TestProblemSolutionExists_MatchesEditorialFilter(t *testing.T) {
	setupProblemListDB(t)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/problem/:code/solution/exists", ProblemSolutionExists)
	exists := func(code string) bool {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/problem/"+code+"/solution/exists", nil))
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		var body map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		return body["exists"].(bool)
	}

	require.True(t, exists("typed"))
	require.False(t, exists("plain"), "a future publish_on is not published")
	require.False(t, exists("undated"), "a NULL publish_on is not published (v1 rule)")
}

func TestProblemList_GroupFilterByID(t *testing.T) {
	fx := setupProblemListDB(t)

	_, body := getProblemListJSON(t, ProblemList, "/problems?group="+itoa(fx.group.ID))
	require.ElementsMatch(t, []string{"typed", "plain", "undated"}, codes(t, body))
	require.EqualValues(t, 3, body["total"])

	// A group filter excludes every other group's problems.
	_, body = getProblemListJSON(t, ProblemList, "/problems?group="+itoa(fx.group2.ID))
	require.Equal(t, []string{"geo"}, codes(t, body))
	require.EqualValues(t, 1, body["total"])
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
	require.Len(t, rows, 2)
	require.Equal(t, map[string]any{
		"id":        float64(fx.group2.ID),
		"name":      "geom",
		"full_name": "Geometry",
	}, rows[0])
	require.Equal(t, map[string]any{
		"id":        float64(fx.group.ID),
		"name":      "graphs",
		"full_name": "Graph Theory",
	}, rows[1])
}
