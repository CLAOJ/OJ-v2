package v2

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm/clause"
)

// ============================================================
// /users list rows carry the display name and the user's organizations,
// without a query per row.
// ============================================================

func TestUserList_DisplayNameAndOrganizations(t *testing.T) {
	setupAdminUserCreateDB(t)

	_, named := seedAccount(t, "named", false, false)
	_, plain := seedAccount(t, "plain", false, false)
	require.NoError(t, db.DB.Model(&named).Update("username_display_override", "Nguyen Van A").Error)
	require.NoError(t, db.DB.Model(&named).Updates(map[string]any{
		"performance_points": 300.0, "contribution_points": 1,
	}).Error)
	require.NoError(t, db.DB.Model(&plain).Updates(map[string]any{
		"performance_points": 100.0, "contribution_points": 9,
	}).Error)

	org := models.Organization{Name: "acme", Slug: "acme", ShortName: "ACME", CreationDate: time.Now()}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&org).Error)
	require.NoError(t, db.DB.Model(&org).Association("Members").Append(&named))

	fetch := func(query string) []map[string]any {
		router := authedRouter(0, http.MethodGet, "/users", UserList)
		w := doRequest(router, http.MethodGet, "/users"+query, "")
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		var body struct {
			Data []map[string]any `json:"data"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		return body.Data
	}

	rows := fetch("")
	require.Len(t, rows, 2)
	byName := map[string]map[string]any{}
	for _, r := range rows {
		byName[r["username"].(string)] = r
	}

	require.Equal(t, "Nguyen Van A", byName["named"]["display_name"])
	require.Equal(t, "", byName["plain"]["display_name"], "no override is an empty string, the caller falls back")

	orgs := byName["named"]["organizations"].([]any)
	require.Len(t, orgs, 1)
	require.Equal(t, float64(org.ID), orgs[0].(map[string]any)["id"])
	require.Equal(t, "ACME", orgs[0].(map[string]any)["short_name"])

	plainOrgs, ok := byName["plain"]["organizations"].([]any)
	require.True(t, ok, "organizations is a JSON array, never null")
	require.Empty(t, plainOrgs)

	// Home uses sort=contribution for the top-contributors widget.
	contrib := fetch("?sort=contribution")
	require.Equal(t, "plain", contrib[0]["username"])
	require.Equal(t, "named", contrib[1]["username"])

	// /contributors is the same list with the contribution ranking: unlisted
	// profiles stay out of it even when they lead on contribution points.
	_, hidden := seedAccount(t, "hidden", false, false)
	require.NoError(t, db.DB.Model(&hidden).Updates(map[string]any{
		"contribution_points": 50, "is_unlisted": true,
	}).Error)
	contrib = fetch("?sort=contribution")
	require.Len(t, contrib, 2, "unlisted profiles are excluded")
	require.Equal(t, "plain", contrib[0]["username"])

	// Home's top-rating widget uses sort=rating; unrated users sort last.
	require.NoError(t, db.DB.Model(&named).Update("rating", 1500).Error)
	rated := fetch("?sort=rating")
	require.Len(t, rated, 2)
	require.Equal(t, "named", rated[0]["username"])
	require.Equal(t, "plain", rated[1]["username"])
}
