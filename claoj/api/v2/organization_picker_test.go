package v2

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// ============================================================
// Picker sources for the non-staff organization create pages: the org
// problem search, the public license list, the public tag list and the
// public user search (staff pickers). Reuses setupOrgTabsDB /
// setupAdminUserCreateDB / seedAccount / authedRouter / doRequest.
// ============================================================

// The org problem list doubles as the problem picker: ?search= filters by
// code/name, rows carry the id the contest form submits, and the existing
// member gate still applies.
func TestOrganizationProblemSearch_GateAndFilter(t *testing.T) {
	fx := setupOrgTabsDB(t)

	search := func(userID uint, query string) (int, []map[string]any) {
		router := authedRouter(userID, http.MethodGet, "/organization/:id/problems", OrganizationProblemList)
		w := doRequest(router, http.MethodGet,
			"/organization/"+strconv.Itoa(int(fx.org.ID))+"/problems"+query, "")
		if w.Code != http.StatusOK {
			return w.Code, nil
		}
		var body struct {
			Data []map[string]any `json:"data"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		return w.Code, body.Data
	}

	code, rows := search(fx.memberUser.ID, "?search=orgpub")
	require.Equal(t, http.StatusOK, code)
	require.Len(t, rows, 1)
	require.Equal(t, "orgpub", rows[0]["code"])
	require.Equal(t, fx.orgPublic.ID, uint(rows[0]["id"].(float64)))

	code, rows = search(fx.memberUser.ID, "?search=nothing-matches")
	require.Equal(t, http.StatusOK, code)
	require.Empty(t, rows)

	code, _ = search(fx.outsiderUser.ID, "?search=orgpub")
	require.Equal(t, http.StatusForbidden, code, "the picker keeps the org membership gate")

	code, _ = search(fx.memberUser.ID, "?search=orgpub&page_size=1")
	require.Equal(t, http.StatusOK, code)
}

// Public read-only lists the create forms need without the staff gate.
func TestPublicPickerLists(t *testing.T) {
	setupAdminUserCreateDB(t)
	require.NoError(t, db.DB.AutoMigrate(&models.License{}, &models.ContestTag{}))

	require.NoError(t, db.DB.Create(&models.License{
		Key: "cc-by", Name: "CC BY", Link: "https://example.com/cc", Text: "body",
	}).Error)
	require.NoError(t, db.DB.Create(&models.ContestTag{Name: "dp", Color: "#ff0000", Description: "dynamic"}).Error)

	_, bob := seedAccount(t, "bob", false, false)
	_, unlisted := seedAccount(t, "charlie", false, false)
	require.NoError(t, db.DB.Model(&models.Profile{}).Where("id = ?", unlisted.ID).
		Update("is_unlisted", true).Error)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/licenses", LicenseList)
	router.GET("/contest-tags", ContestTagList)
	router.GET("/users", UserList)

	get := func(path string) map[string]any {
		w := doRequest(router, http.MethodGet, path, "")
		require.Equal(t, http.StatusOK, w.Code, path+": "+w.Body.String())
		var body map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		return body
	}

	licenses := get("/licenses")["data"].([]any)
	require.Len(t, licenses, 1)
	require.Equal(t, "cc-by", licenses[0].(map[string]any)["key"])
	require.Equal(t, "CC BY", licenses[0].(map[string]any)["name"])

	tags := get("/contest-tags")["data"].([]any)
	require.Len(t, tags, 1)
	require.Equal(t, "dp", tags[0].(map[string]any)["name"])

	users := get("/users?search=bo")["data"].([]any)
	require.Len(t, users, 1)
	require.Equal(t, "bob", users[0].(map[string]any)["username"])
	require.Equal(t, bob.ID, uint(users[0].(map[string]any)["id"].(float64)))

	users = get("/users?search=charlie")["data"].([]any)
	require.Empty(t, users, "unlisted profiles stay out of the public picker")

	// The picker list is a plain paginated list, so the form's page_size works.
	users = get("/users?search=bo&page_size=1")["data"].([]any)
	require.Len(t, users, 1)
}
