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

func adminContestRouter() *gin.Engine {
	g := integration.TestRouter()
	g.POST("/auth/login", authHandlers.Login)
	g.Use(auth.RequiredMiddleware())
	g.Use(auth.AdminRequiredMiddleware())
	g.PATCH("/admin/contest/:key", auth.RequireContestEdit(), v2.AdminContestUpdate)
	return g
}

func TestAdminContestUpdate_Authz(t *testing.T) {
	testDB := integration.SetupIntegrationDB(t)
	defer integration.CleanupDB(t, testDB)

	ct := models.Contest{Key: "cx", Name: "CX"}
	require.NoError(t, testDB.DB.Create(&ct).Error)

	staff := integration.CreateTestUser(testDB.DB, "cstaff", "Password123!", true)
	makeStaff(t, testDB.DB, staff.ID, true, false)
	staffToken := loginToken(t, "cstaff", "Password123!")

	su := integration.CreateTestUser(testDB.DB, "csuper", "Password123!", true)
	makeStaff(t, testDB.DB, su.ID, true, true)
	suToken := loginToken(t, "csuper", "Password123!")

	g := adminContestRouter()

	resp := integration.MakeRequest(t, g, integration.HTTPRequest{
		Method:  "PATCH",
		Path:    fmt.Sprintf("/admin/contest/%s", ct.Key),
		Headers: map[string]string{"Authorization": "Bearer " + staffToken},
		Body:    map[string]interface{}{"is_visible": true},
	})
	require.Equal(t, http.StatusForbidden, resp.Code)

	resp = integration.MakeRequest(t, g, integration.HTTPRequest{
		Method:  "PATCH",
		Path:    fmt.Sprintf("/admin/contest/%s", ct.Key),
		Headers: map[string]string{"Authorization": "Bearer " + suToken},
		Body:    map[string]interface{}{"is_visible": true},
	})
	// Not 403: the guard admits the superuser (isolates guard from the service).
	require.NotEqual(t, http.StatusForbidden, resp.Code)
}
