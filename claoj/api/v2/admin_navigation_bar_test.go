package v2

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupNavigationDB(t *testing.T) {
	t.Helper()
	old := db.DB
	t.Cleanup(func() { db.DB = old })
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)
	sqlDB, err := database.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, database.AutoMigrate(&models.NavigationBar{}))
	db.DB = database
}

// seedNav creates an entry through the admin API so the tree is built exactly
// as production builds it, then returns the stored row.
func seedNav(t *testing.T, key string, parentID *uint, order int) models.NavigationBar {
	t.Helper()
	payload := fmt.Sprintf(`{"key":%q,"label":%q,"path":%q,"order":%d}`, key, key, "/"+key, order)
	if parentID != nil {
		payload = fmt.Sprintf(`{"key":%q,"label":%q,"path":%q,"order":%d,"parent_id":%d}`,
			key, key, "/"+key, order, *parentID)
	}
	router := authedRouter(1, http.MethodPost, "/admin/navigation-bars", AdminNavigationBarCreate)
	w := doRequest(router, http.MethodPost, "/admin/navigation-bars", payload)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	var resp struct {
		NavigationBar struct {
			ID uint `json:"id"`
		} `json:"navigation_bar"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	var nav models.NavigationBar
	require.NoError(t, db.DB.First(&nav, resp.NavigationBar.ID).Error)
	return nav
}

func fetchNav(t *testing.T, id uint) models.NavigationBar {
	t.Helper()
	var nav models.NavigationBar
	require.NoError(t, db.DB.First(&nav, id).Error)
	return nav
}

// assertNavTree checks the nested-set invariants for the whole table.
func assertNavTree(t *testing.T) {
	t.Helper()
	var nodes []models.NavigationBar
	require.NoError(t, db.DB.Find(&nodes).Error)
	byID := make(map[uint]models.NavigationBar, len(nodes))
	for _, n := range nodes {
		byID[n.ID] = n
		require.Less(t, n.Lft, n.Rght, "node %s: lft must be < rght", n.Key)
		// 1-based, like Django MPTT: every tree spans 1..2*len.
		require.Greater(t, n.Lft, 0, "node %s: lft must be >= 1", n.Key)
	}
	lfts := map[int]bool{}
	for _, n := range nodes {
		require.False(t, lfts[n.Lft], "duplicate lft %d", n.Lft)
		lfts[n.Lft] = true
		if n.ParentID == nil {
			require.Equal(t, 0, n.Level, "root %s level", n.Key)
			continue
		}
		parent, ok := byID[*n.ParentID]
		require.True(t, ok, "node %s parent missing", n.Key)
		require.Equal(t, parent.Level+1, n.Level, "node %s level", n.Key)
		require.Equal(t, parent.TreeID, n.TreeID, "node %s tree_id", n.Key)
		require.Greater(t, n.Lft, parent.Lft, "node %s after parent lft", n.Key)
		require.Less(t, n.Rght, parent.Rght, "node %s inside parent rght", n.Key)
	}
	// Distinct roots get distinct tree ids, and each tree occupies a
	// contiguous nested-set range.
	seenTrees := map[int]models.NavigationBar{}
	for _, n := range nodes {
		if n.ParentID != nil {
			continue
		}
		if prev, ok := seenTrees[n.TreeID]; ok {
			require.Greater(t, n.Lft, prev.Rght, "tree %d ranges overlap", n.TreeID)
		}
		seenTrees[n.TreeID] = n
	}
}

func TestAdminNavigationBarCreate_PersistsRegex(t *testing.T) {
	setupNavigationDB(t)
	router := authedRouter(1, http.MethodPost, "/admin/navigation-bars", AdminNavigationBarCreate)

	w := doRequest(router, http.MethodPost, "/admin/navigation-bars",
		`{"key":"problems","label":"Problems","path":"/problems","regex":"^/problems(?:/|$)","order":1}`)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	nav := fetchNav(t, 1)
	require.Equal(t, "^/problems(?:/|$)", nav.Regex)
	require.Equal(t, 0, nav.Level)
	assertNavTree(t)

	// Empty regex is allowed (v1 stores NOT NULL with an empty value).
	w = doRequest(router, http.MethodPost, "/admin/navigation-bars",
		`{"key":"home","label":"Home","path":"/"}`)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	home := fetchNav(t, 2)
	require.Equal(t, "", home.Regex)
	require.Equal(t, 2, home.Rght-home.Lft+1)
}

func TestAdminNavigationBarCreate_RejectsInvalidRegex(t *testing.T) {
	setupNavigationDB(t)
	router := authedRouter(1, http.MethodPost, "/admin/navigation-bars", AdminNavigationBarCreate)

	w := doRequest(router, http.MethodPost, "/admin/navigation-bars",
		`{"key":"bad","label":"Bad","path":"/bad","regex":"("}`)
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	require.Contains(t, w.Body.String(), "navigation bar regex is invalid")
	// The compile detail stays in the server log, not the user-facing toast.
	require.NotContains(t, w.Body.String(), "error parsing regexp")

	var count int64
	require.NoError(t, db.DB.Model(&models.NavigationBar{}).Count(&count).Error)
	require.Zero(t, count)
}

func TestAdminNavigationBarCreate_UnknownParent(t *testing.T) {
	setupNavigationDB(t)
	router := authedRouter(1, http.MethodPost, "/admin/navigation-bars", AdminNavigationBarCreate)

	w := doRequest(router, http.MethodPost, "/admin/navigation-bars",
		`{"key":"orphan","label":"Orphan","path":"/orphan","parent_id":999}`)
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	require.Contains(t, w.Body.String(), "parent navigation bar not found")
}

func TestAdminNavigationBarUpdate_RejectsParentCycle(t *testing.T) {
	setupNavigationDB(t)
	a := seedNav(t, "a", nil, 1)
	b := seedNav(t, "b", &a.ID, 1)
	c := seedNav(t, "c", &b.ID, 1)

	router := authedRouter(1, http.MethodPatch, "/admin/navigation-bar/:id", AdminNavigationBarUpdate)

	// Self parent.
	w := doRequest(router, http.MethodPatch, "/admin/navigation-bar/1", `{"parent_id":1}`)
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	require.Contains(t, w.Body.String(), "descendants")

	// Descendant parent.
	w = doRequest(router, http.MethodPatch, "/admin/navigation-bar/1", `{"parent_id":3}`)
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())

	// Unknown parent.
	w = doRequest(router, http.MethodPatch, "/admin/navigation-bar/3", `{"parent_id":42}`)
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	require.Contains(t, w.Body.String(), "parent navigation bar not found")

	// Still intact.
	got := fetchNav(t, c.ID)
	require.Equal(t, b.ID, *got.ParentID)
	assertNavTree(t)
}

func TestAdminNavigationBarUpdate_MoveSubtreeRebuildsTree(t *testing.T) {
	setupNavigationDB(t)
	a := seedNav(t, "a", nil, 1)
	b := seedNav(t, "b", &a.ID, 1)
	c := seedNav(t, "c", &b.ID, 1)
	d := seedNav(t, "d", &a.ID, 2)

	router := authedRouter(1, http.MethodPatch, "/admin/navigation-bar/:id", AdminNavigationBarUpdate)

	// Move b (with child c) under d.
	w := doRequest(router, http.MethodPatch, "/admin/navigation-bar/2", `{"parent_id":4}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assertNavTree(t)

	movedB, movedC, movedD := fetchNav(t, b.ID), fetchNav(t, c.ID), fetchNav(t, d.ID)
	require.Equal(t, d.ID, *movedB.ParentID)
	require.Equal(t, 2, movedB.Level)
	require.Equal(t, 3, movedC.Level)
	require.Equal(t, b.ID, *movedC.ParentID)
	require.True(t, movedC.Lft > movedB.Lft && movedC.Rght < movedB.Rght)
	require.True(t, movedB.Lft > movedD.Lft && movedB.Rght < movedD.Rght)

	// Reorder b to the front of the roots and detach it (parent_id: null).
	w = doRequest(router, http.MethodPatch, "/admin/navigation-bar/2", `{"parent_id":null,"order":0}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assertNavTree(t)

	movedB = fetchNav(t, b.ID)
	rootA := fetchNav(t, a.ID)
	require.Nil(t, movedB.ParentID)
	require.Equal(t, 0, movedB.Level)
	require.Equal(t, 4, movedB.Rght-movedB.Lft+1)    // b keeps its subtree (c)
	require.NotEqual(t, movedB.TreeID, rootA.TreeID) // it is its own root tree now
	require.Less(t, movedB.Lft, rootA.Lft)           // order 0 sorts first
}

func TestAdminNavigationBarUpdate_PersistsRegexAndRejectsInvalid(t *testing.T) {
	setupNavigationDB(t)
	a := seedNav(t, "a", nil, 1)
	router := authedRouter(1, http.MethodPatch, "/admin/navigation-bar/:id", AdminNavigationBarUpdate)

	w := doRequest(router, http.MethodPatch, "/admin/navigation-bar/1", `{"regex":"^/a"}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Equal(t, "^/a", fetchNav(t, a.ID).Regex)

	w = doRequest(router, http.MethodPatch, "/admin/navigation-bar/1", `{"regex":"[z-a]"}`)
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	require.Equal(t, "^/a", fetchNav(t, a.ID).Regex)
}

func TestAdminNavigationBarDelete_CascadesAndKeepsTreeConsistent(t *testing.T) {
	setupNavigationDB(t)
	a := seedNav(t, "a", nil, 1)
	b := seedNav(t, "b", &a.ID, 1)
	c := seedNav(t, "c", &b.ID, 1)
	d := seedNav(t, "d", &a.ID, 2)

	router := authedRouter(1, http.MethodDelete, "/admin/navigation-bar/:id", AdminNavigationBarDelete)
	w := doRequest(router, http.MethodDelete, "/admin/navigation-bar/2", "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var remaining []models.NavigationBar
	require.NoError(t, db.DB.Find(&remaining).Error)
	require.Len(t, remaining, 2) // a and d; b and c are gone
	for _, n := range remaining {
		require.NotEqual(t, b.ID, n.ID)
		require.NotEqual(t, c.ID, n.ID)
	}
	assertNavTree(t)

	rootA := fetchNav(t, a.ID)
	keptD := fetchNav(t, d.ID)
	require.Equal(t, 4, rootA.Rght-rootA.Lft+1)
	require.Equal(t, 2, keptD.Rght-keptD.Lft+1)
	require.Equal(t, 1, keptD.Level)

	w = doRequest(router, http.MethodDelete, "/admin/navigation-bar/2", "")
	require.Equal(t, http.StatusNotFound, w.Code, w.Body.String())
}

func TestAdminNavigationBarListAndDetail_IncludeRegexAndParent(t *testing.T) {
	setupNavigationDB(t)
	a := seedNav(t, "a", nil, 1)
	require.NoError(t, db.DB.Model(&models.NavigationBar{}).Where("id = ?", a.ID).Update("regex", "^/a").Error)
	seedNav(t, "b", &a.ID, 1)

	listRouter := authedRouter(1, http.MethodGet, "/admin/navigation-bars", AdminNavigationBarList)
	w := doRequest(listRouter, http.MethodGet, "/admin/navigation-bars", "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var listResp struct {
		Data []struct {
			Key      string `json:"key"`
			Regex    string `json:"regex"`
			ParentID *uint  `json:"parent_id"`
			Parent   *struct {
				ID    uint   `json:"id"`
				Label string `json:"label"`
			} `json:"parent"`
		} `json:"data"`
		Total int64 `json:"total"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &listResp))
	require.Equal(t, int64(2), listResp.Total)
	require.Equal(t, "a", listResp.Data[0].Key)
	require.Equal(t, "^/a", listResp.Data[0].Regex)
	require.Nil(t, listResp.Data[0].ParentID)
	require.Nil(t, listResp.Data[0].Parent)
	require.Equal(t, "b", listResp.Data[1].Key)
	require.NotNil(t, listResp.Data[1].ParentID, "list must expose parent_id: the admin parent picker excludes descendants from it")
	require.Equal(t, a.ID, *listResp.Data[1].ParentID)
	require.NotNil(t, listResp.Data[1].Parent)
	require.Equal(t, a.ID, listResp.Data[1].Parent.ID)
	require.Equal(t, "a", listResp.Data[1].Parent.Label)

	detailRouter := authedRouter(1, http.MethodGet, "/admin/navigation-bar/:id", AdminNavigationBarDetail)
	w = doRequest(detailRouter, http.MethodGet, "/admin/navigation-bar/2", "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var detail struct {
		Regex       string `json:"regex"`
		ParentID    *uint  `json:"parent_id"`
		ParentLabel string `json:"parent_label"`
		Lft         int    `json:"lft"`
		Rght        int    `json:"rght"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &detail))
	require.Equal(t, "", detail.Regex)
	require.NotNil(t, detail.ParentID)
	require.Equal(t, a.ID, *detail.ParentID)
	require.Equal(t, "a", detail.ParentLabel)
	require.Less(t, detail.Lft, detail.Rght)
}
