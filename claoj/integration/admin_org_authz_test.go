package integration_test

import (
	"fmt"
	"net/http"
	"testing"

	v2 "github.com/CLAOJ/claoj/api/v2"
	authHandlers "github.com/CLAOJ/claoj/api/v2/auth"
	"github.com/CLAOJ/claoj/auth"
	"github.com/CLAOJ/claoj/integration"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func adminOrgUpdateRouter() *gin.Engine {
	g := integration.TestRouter()
	g.POST("/auth/login", authHandlers.Login)
	g.Use(auth.RequiredMiddleware())
	g.Use(auth.AdminRequiredMiddleware())
	g.PATCH("/admin/organization/:id", auth.RequireOrgEdit(), v2.AdminOrganizationUpdate)
	return g
}

func TestAdminOrgUpdate_Authz(t *testing.T) {
	testDB := integration.SetupIntegrationDB(t)
	defer integration.CleanupDB(t, testDB)

	org := models.Organization{Name: "OX", Slug: "ox", ShortName: "OX", IsOpen: true}
	require.NoError(t, testDB.DB.Create(&org).Error)

	// Staff who administers this org -> 200.
	orgAdmin := integration.CreateTestUser(testDB.DB, "oxadmin", "Password123!", true)
	makeStaff(t, testDB.DB, orgAdmin.ID, true, false)
	adminProfile := profileForUser(t, testDB.DB, orgAdmin.ID)
	require.NoError(t, testDB.DB.Model(&org).Association("Admins").Append(&adminProfile))
	adminToken := loginToken(t, "oxadmin", "Password123!")

	// Staff who does NOT administer this org -> 403.
	other := integration.CreateTestUser(testDB.DB, "oxother", "Password123!", true)
	makeStaff(t, testDB.DB, other.ID, true, false)
	otherToken := loginToken(t, "oxother", "Password123!")

	g := adminOrgUpdateRouter()

	resp := integration.MakeRequest(t, g, integration.HTTPRequest{
		Method:  "PATCH",
		Path:    fmt.Sprintf("/admin/organization/%d", org.ID),
		Headers: map[string]string{"Authorization": "Bearer " + otherToken},
		Body:    map[string]interface{}{"about": "x"},
	})
	require.Equal(t, http.StatusForbidden, resp.Code)

	resp = integration.MakeRequest(t, g, integration.HTTPRequest{
		Method:  "PATCH",
		Path:    fmt.Sprintf("/admin/organization/%d", org.ID),
		Headers: map[string]string{"Authorization": "Bearer " + adminToken},
		Body:    map[string]interface{}{"about": "x"},
	})
	// Not 403: the guard admits the org admin (isolates guard from the service).
	require.NotEqual(t, http.StatusForbidden, resp.Code)
}
