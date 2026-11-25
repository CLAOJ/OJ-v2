package v2

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type publicNavNode struct {
	Key      string          `json:"key"`
	Label    string          `json:"label"`
	Path     string          `json:"path"`
	Regex    string          `json:"regex"`
	Children []publicNavNode `json:"children"`
}

// publicNavRouter exposes the handler with no auth middleware: the route must
// work for anonymous visitors, exactly as v1 rendered it for everyone.
func publicNavRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/navigation", PublicNavigation)
	return router
}

func createNavRow(t *testing.T, key, path, regex string, order int, parentID *uint) models.NavigationBar {
	t.Helper()
	row := models.NavigationBar{Key: key, Label: key, Path: path, Regex: regex, Order: order, ParentID: parentID}
	require.NoError(t, db.DB.Create(&row).Error)
	return row
}

func TestPublicNavigation_OrdersByOrderAndNestsChildren(t *testing.T) {
	setupNavigationDB(t)

	// Inserted out of display order on purpose: the endpoint must sort by
	// `order`, like v1's MPTT (order_insertion_by = ['order']).
	createNavRow(t, "contest", "/contests/", "^/contests/", 30, nil)
	problems := createNavRow(t, "problem", "/problems/", "^/problems/", 20, nil)
	sub := createNavRow(t, "problem_tag", "/problems/tag/", "^/problems/tag/", 1, &problems.ID)
	createNavRow(t, "user", "/users/", "^/users/", 10, nil)

	w := doRequest(publicNavRouter(), http.MethodGet, "/navigation", "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var resp struct {
		Data []publicNavNode `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Len(t, resp.Data, 3)

	require.Equal(t, []string{"user", "problem", "contest"},
		[]string{resp.Data[0].Key, resp.Data[1].Key, resp.Data[2].Key})
	require.Equal(t, "/problems/", resp.Data[1].Path)
	require.Equal(t, "^/problems/", resp.Data[1].Regex)
	require.Len(t, resp.Data[1].Children, 1)
	require.Equal(t, sub.Key, resp.Data[1].Children[0].Key)
	require.Empty(t, resp.Data[0].Children)
}

func TestPublicNavigation_EmptyTableReturnsEmptyList(t *testing.T) {
	setupNavigationDB(t)

	w := doRequest(publicNavRouter(), http.MethodGet, "/navigation", "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.JSONEq(t, `{"data":[]}`, w.Body.String())
}