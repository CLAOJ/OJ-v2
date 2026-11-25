package v2

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// The per-problem editorial editor finds its row through ?problem=<code>; the
// list must also fill problem_code/problem_name (they used to scan empty
// because the struct tags did not match the SELECT aliases).
func TestAdminSolutionList_ProblemFilterAndNames(t *testing.T) {
	fx := setupProblemListDB(t)
	router := authedRouter(fx.admin.ID, http.MethodGet, "/admin/solutions", AdminSolutionList)

	type item struct {
		ProblemCode string `json:"problem_code"`
		ProblemName string `json:"problem_name"`
	}
	type resp struct {
		Data  []item `json:"data"`
		Total int64  `json:"total"`
	}
	get := func(target string) resp {
		w := doRequest(router, http.MethodGet, target, "")
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		var r resp
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &r))
		return r
	}

	all := get("/admin/solutions")
	require.EqualValues(t, 3, all.Total)
	require.Len(t, all.Data, 3)
	for _, it := range all.Data {
		require.NotEmpty(t, it.ProblemCode)
		require.Equal(t, it.ProblemCode, it.ProblemName) // fixture names equal codes
	}

	one := get("/admin/solutions?problem=plain")
	require.EqualValues(t, 1, one.Total)
	require.Len(t, one.Data, 1)
	require.Equal(t, "plain", one.Data[0].ProblemCode)

	none := get("/admin/solutions?problem=geo")
	require.EqualValues(t, 0, none.Total)
	require.Empty(t, none.Data)
}
