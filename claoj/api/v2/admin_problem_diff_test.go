package v2

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/CLAOJ/claoj/auth"
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm/clause"
)

// problemDiffRouter mounts the matrix route behind the real admin guard
// (auth.RequireProblemEdit), so permission and :code handling are exercised.
func problemDiffRouter(userID uint) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Handle(http.MethodGet, "/admin/problem/:code/diff",
		func(c *gin.Context) { c.Set("user_id", userID) },
		auth.RequireProblemEdit(),
		AdminProblemSubmissionDiff,
	)
	return router
}

func TestAdminProblemSubmissionDiff(t *testing.T) {
	setupAdminSubmissionDB(t)
	fx := seedAdminSubmission(t)
	root := newSubmissionUser(t, "root", true)
	staff := newSubmissionUser(t, "staff", false)

	// A second submission for the same problem, with two test cases, so the
	// response has something to order and to put in the matrix columns.
	other := models.Submission{
		UserID: fx.owner.ID, ProblemID: fx.problem.ID, LanguageID: fx.lang.ID,
		Date: time.Now().Add(time.Minute), Status: "D",
	}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&other).Error)
	for _, tc := range []models.SubmissionTestCase{
		{SubmissionID: other.ID, Case: 1, Status: "WA", Feedback: ""},
		{SubmissionID: other.ID, Case: 2, Status: "AC", Feedback: ""},
	} {
		require.NoError(t, db.DB.Omit(clause.Associations).Create(&tc).Error)
	}

	type caseItem struct {
		Case   int     `json:"case"`
		Status string  `json:"status"`
		Points float64 `json:"points"`
	}
	type subItem struct {
		ID       uint       `json:"id"`
		User     string     `json:"user"`
		Language string     `json:"language"`
		Result   string     `json:"result"`
		Cases    []caseItem `json:"cases"`
	}
	type body struct {
		Problem     string    `json:"problem"`
		ProblemName string    `json:"problem_name"`
		Submissions []subItem `json:"submissions"`
	}

	t.Run("non-editor is forbidden", func(t *testing.T) {
		w := doRequest(problemDiffRouter(staff.ID), http.MethodGet, "/admin/problem/aplusb/diff", "")
		require.Equal(t, http.StatusForbidden, w.Code, w.Body.String())
	})

	t.Run("unknown problem is not found", func(t *testing.T) {
		w := doRequest(problemDiffRouter(root.ID), http.MethodGet, "/admin/problem/nope/diff", "")
		require.Equal(t, http.StatusNotFound, w.Code, w.Body.String())
	})

	t.Run("editor sees rows and per-case results", func(t *testing.T) {
		w := doRequest(problemDiffRouter(root.ID), http.MethodGet, "/admin/problem/aplusb/diff", "")
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())

		var got body
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
		require.Equal(t, "aplusb", got.Problem)
		require.Equal(t, "A+B", got.ProblemName)
		require.Len(t, got.Submissions, 2)

		// Newest first, matching the admin submissions list.
		require.Equal(t, other.ID, got.Submissions[0].ID)
		require.Equal(t, "alice", got.Submissions[0].User)
		require.Equal(t, "CPP17", got.Submissions[0].Language)
		require.Len(t, got.Submissions[0].Cases, 2)
		require.Equal(t, "WA", got.Submissions[0].Cases[0].Status)
		require.Equal(t, "AC", got.Submissions[0].Cases[1].Status)

		require.Equal(t, fx.sub.ID, got.Submissions[1].ID)
		require.Equal(t, "AC", got.Submissions[1].Result)
		require.Len(t, got.Submissions[1].Cases, 1)
	})

	t.Run("user filter narrows the rows", func(t *testing.T) {
		w := doRequest(problemDiffRouter(root.ID), http.MethodGet, "/admin/problem/aplusb/diff?user=nobody", "")
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		var got body
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
		require.Empty(t, got.Submissions)
	})
}