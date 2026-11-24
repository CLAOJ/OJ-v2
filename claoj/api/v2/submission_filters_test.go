package v2

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm/clause"
)

// ============================================================
// /submissions filters: multi-value verdict/language (v1's
// `result__in` / `language__in`) and the per-user organization filter.
// ============================================================

// listSubmissions calls SubmissionList anonymously and returns the rows/total.
func listSubmissions(t *testing.T, query string) (int, []map[string]any, float64) {
	t.Helper()
	router := authedRouter(0, http.MethodGet, "/submissions", SubmissionList)
	w := doRequest(router, http.MethodGet, "/submissions"+query, "")
	if w.Code != http.StatusOK {
		return w.Code, nil, 0
	}
	var body struct {
		Data  []map[string]any `json:"data"`
		Total float64          `json:"total"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	return w.Code, body.Data, body.Total
}

func TestSubmissionList_MultiValueAndOrganizationFilters(t *testing.T) {
	setupAdminSubmissionDB(t)

	_, alice := seedAccount(t, "alice", false, false)
	_, bob := seedAccount(t, "bob", false, false)

	org := models.Organization{Name: "acme", Slug: "acme", ShortName: "ACME", CreationDate: time.Now()}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&org).Error)
	require.NoError(t, db.DB.Model(&org).Association("Members").Append(&alice))

	group := models.ProblemGroup{Name: "g", FullName: "Group"}
	require.NoError(t, db.DB.Create(&group).Error)
	problem := models.Problem{Code: "p1", Name: "Problem", GroupID: group.ID, IsPublic: true, TimeLimit: 1, MemoryLimit: 256}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&problem).Error)

	cpp := models.Language{Key: "CPP17", Name: "C++17", Extension: "cpp"}
	py := models.Language{Key: "PY3", Name: "Python 3", Extension: "py"}
	require.NoError(t, db.DB.Create(&cpp).Error)
	require.NoError(t, db.DB.Create(&py).Error)

	now := time.Now()
	seq := 0
	newSub := func(user models.Profile, lang models.Language, result string) {
		r := result
		seq++
		s := models.Submission{
			UserID: user.ID, ProblemID: problem.ID, LanguageID: lang.ID,
			Date: now.Add(time.Duration(seq) * time.Second), Status: "D", Result: &r,
		}
		require.NoError(t, db.DB.Omit(clause.Associations).Create(&s).Error)
	}
	newSub(alice, cpp, "AC")
	newSub(alice, py, "WA")
	newSub(bob, cpp, "WA")

	cases := []struct {
		name  string
		query string
		total float64
	}{
		{"no filter", "", 3},
		{"single verdict", "?result=AC", 1},
		{"repeated verdict", "?result=AC&result=WA", 3},
		{"comma verdict", "?result=AC,WA", 3},
		{"unknown verdict", "?result=CE", 0},
		{"single language", "?language=PY3", 1},
		{"comma language", "?language=CPP17,PY3", 3},
		{"verdict and language", "?result=WA&language=CPP17", 1},
		{"organization membership", "?organization=" + strconv.Itoa(int(org.ID)), 2},
		{"organization and verdict", "?organization=" + strconv.Itoa(int(org.ID)) + "&result=AC", 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			code, rows, total := listSubmissions(t, tc.query)
			require.Equal(t, http.StatusOK, code)
			require.Equal(t, tc.total, total, "total must match the filtered count")
			require.Equal(t, int(tc.total), len(rows))
		})
	}

	code, _, _ := listSubmissions(t, "?organization=abc")
	require.Equal(t, http.StatusBadRequest, code)
}

// queryValues is the shared parser behind both filters.
func TestQueryValues_RepeatedAndCommaSeparated(t *testing.T) {
	router := authedRouter(0, http.MethodGet, "/probe", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"values": queryValues(c, "x")})
	})
	w := doRequest(router, http.MethodGet, "/probe?x=a&x=b,c&x=&x=%20d%20", "")
	require.Equal(t, http.StatusOK, w.Code)
	var body struct {
		Values []string `json:"values"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, []string{"a", "b", "c", "d"}, body.Values)
}
