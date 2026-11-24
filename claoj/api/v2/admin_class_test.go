package v2

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/CLAOJ/claoj/auth"
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// classGuardRouter wires the real admin gate so the test proves the route
// cannot be read by a non-staff user.
func classGuardRouter(userID uint) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	})
	r.Use(auth.AdminRequiredMiddleware())
	r.GET("/admin/classes", AdminClassList)
	return r
}

func TestAdminClassList_ShapeAndSearch(t *testing.T) {
	setupAdminDetailDB(t)
	org := createContestTestOrg(t, "org-a")
	alpha := models.Class{OrganizationID: org.ID, Name: "Alpha", Slug: "alpha"}
	beta := models.Class{OrganizationID: org.ID, Name: "Beta", Slug: "beta"}
	require.NoError(t, db.DB.Create(&alpha).Error)
	require.NoError(t, db.DB.Create(&beta).Error)

	w := doRequest(authedRouter(1, http.MethodGet, "/admin/classes", AdminClassList),
		http.MethodGet, "/admin/classes?search=alp", "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var body struct {
		Data     []map[string]any `json:"data"`
		Total    int64            `json:"total"`
		Page     int              `json:"page"`
		PageSize int              `json:"page_size"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.EqualValues(t, 1, body.Total)
	require.Equal(t, 1, body.Page)
	require.NotZero(t, body.PageSize)
	require.Len(t, body.Data, 1)
	require.Equal(t, "Alpha", body.Data[0]["name"])
	require.Equal(t, "alpha", body.Data[0]["slug"])
	require.EqualValues(t, org.ID, body.Data[0]["organization_id"])
	require.Equal(t, "org-a", body.Data[0]["organization_name"])

	w = doRequest(authedRouter(1, http.MethodGet, "/admin/classes", AdminClassList),
		http.MethodGet, "/admin/classes", "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.EqualValues(t, 2, body.Total)
}

func TestAdminClassList_RequiresAdmin(t *testing.T) {
	setupAdminDetailDB(t)
	plain := models.AuthUser{Username: "plain", IsActive: true}
	require.NoError(t, db.DB.Create(&plain).Error)
	staff := models.AuthUser{Username: "staffer", IsActive: true, IsStaff: true}
	require.NoError(t, db.DB.Create(&staff).Error)

	w := doRequest(classGuardRouter(plain.ID), http.MethodGet, "/admin/classes", "")
	require.Equal(t, http.StatusForbidden, w.Code, w.Body.String())

	w = doRequest(classGuardRouter(staff.ID), http.MethodGet, "/admin/classes", "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
}