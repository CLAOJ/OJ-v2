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

type navRow struct {
	key, label, path       string
	tree, lft, rght, level int
	order                  int
	parent                 *uint
}

func createNavRow(t *testing.T, row navRow) models.NavigationBar {
	t.Helper()
	nav := models.NavigationBar{
		Key:      row.key,
		Label:    row.label,
		Path:     row.path,
		TreeID:   row.tree,
		Lft:      row.lft,
		Rght:     row.rght,
		Level:    row.level,
		Order:    row.order,
		ParentID: row.parent,
	}
	require.NoError(t, db.DB.Create(&nav).Error)
	return nav
}

// TestPublicNavigation_OrdersByMpttTree seeds the live judge_navigationbar rows
// (2026-11-25, read-only copy) in a scrambled insertion order and with the
// `order` column deliberately reversed: v1 renders mptt_tree, so the response
// must follow tree_id/lft, never `order`.
func TestPublicNavigation_OrdersByMpttTree(t *testing.T) {
	setupNavigationDB(t)

	// "About" owns the only dropdown: Server Status, Judges, devlog.
	about := createNavRow(t, navRow{key: "about", label: "About", path: "/about/", tree: 8, lft: 1, rght: 8, order: 99})

	createNavRow(t, navRow{key: "devlog", label: "devlog", path: "/devlog/", tree: 8, lft: 6, rght: 7, level: 1, order: 2, parent: &about.ID})
	createNavRow(t, navRow{key: "wiki", label: "Wiki", path: "/wiki/", tree: 7, lft: 1, rght: 2, order: 3})
	createNavRow(t, navRow{key: "status", label: "Judges", path: "/status/", tree: 8, lft: 4, rght: 5, level: 1, order: 1, parent: &about.ID})
	createNavRow(t, navRow{key: "contest", label: "Contests", path: "/contests/", tree: 6, lft: 1, rght: 2, order: 4})
	createNavRow(t, navRow{key: "ServerStat", label: "Server Status", path: "https://status.claoj.edu.vn", tree: 8, lft: 2, rght: 3, level: 1, order: 3, parent: &about.ID})
	createNavRow(t, navRow{key: "org", label: "Organizations", path: "/organizations/", tree: 5, lft: 1, rght: 2, order: 5})
	createNavRow(t, navRow{key: "user", label: "Leaderboard", path: "/users/", tree: 3, lft: 1, rght: 2, order: 6})
	createNavRow(t, navRow{key: "submit", label: "Submissions", path: "/submissions/", tree: 2, lft: 1, rght: 2, order: 7})
	createNavRow(t, navRow{key: "bugaboo", label: "Problems", path: "/problems/", tree: 1, lft: 1, rght: 2, order: 8})

	w := doRequest(publicNavRouter(), http.MethodGet, "/navigation", "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var resp struct {
		Data []publicNavNode `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

	keys := make([]string, len(resp.Data))
	for i, node := range resp.Data {
		keys[i] = node.Key
	}
	require.Equal(t, []string{"bugaboo", "submit", "user", "org", "contest", "wiki", "about"}, keys)

	aboutNode := resp.Data[len(resp.Data)-1]
	require.Equal(t, "/about/", aboutNode.Path)
	require.Len(t, aboutNode.Children, 3)
	require.Equal(t, []string{"ServerStat", "status", "devlog"},
		[]string{aboutNode.Children[0].Key, aboutNode.Children[1].Key, aboutNode.Children[2].Key})
	require.Equal(t, "https://status.claoj.edu.vn", aboutNode.Children[0].Path)
	require.Equal(t, "/status/", aboutNode.Children[1].Path)
	require.Equal(t, "/devlog/", aboutNode.Children[2].Path)
	require.Empty(t, resp.Data[0].Children)
}

func TestPublicNavigation_EmptyTableReturnsEmptyList(t *testing.T) {
	setupNavigationDB(t)

	w := doRequest(publicNavRouter(), http.MethodGet, "/navigation", "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.JSONEq(t, `{"data":[]}`, w.Body.String())
}
