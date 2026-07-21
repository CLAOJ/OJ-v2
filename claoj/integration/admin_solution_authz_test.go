package integration_test

import (
	"fmt"
	"net/http"
	"testing"

	v2 "github.com/CLAOJ/claoj/api/v2"
	"github.com/CLAOJ/claoj/auth"
	"github.com/CLAOJ/claoj/integration"
	"github.com/CLAOJ/claoj/models"
	"github.com/stretchr/testify/require"
)

func TestAdminSolutionAuthz_DeniesPlainStaff(t *testing.T) {
	testDB := integration.SetupIntegrationDB(t)
	defer integration.CleanupDB(t, testDB)
	// Solution is not in the harness AutoMigrate list — add it here.
	require.NoError(t, testDB.DB.AutoMigrate(&models.Solution{}))

	prob := models.Problem{Code: "sol_p", Name: "SolP", IsPublic: true}
	require.NoError(t, testDB.DB.Create(&prob).Error)
	sol := models.Solution{ProblemID: prob.ID, Content: "editorial"}
	require.NoError(t, testDB.DB.Create(&sol).Error)

	staff := integration.CreateTestUser(testDB.DB, "solstaff", "Password123!", true)
	makeStaff(t, testDB.DB, staff.ID, true, false)
	staffToken := loginToken(t, "solstaff", "Password123!")

	g := integration.TestRouter()
	g.Use(auth.RequiredMiddleware())
	g.Use(auth.AdminRequiredMiddleware())
	g.POST("/admin/solutions", v2.AdminSolutionCreate)
	g.PATCH("/admin/solution/:id", v2.AdminSolutionUpdate)
	g.DELETE("/admin/solution/:id", v2.AdminSolutionDelete)

	hdr := map[string]string{"Authorization": "Bearer " + staffToken}

	r := integration.MakeRequest(t, g, integration.HTTPRequest{Method: "PATCH", Path: fmt.Sprintf("/admin/solution/%d", sol.ID), Headers: hdr, Body: map[string]interface{}{"is_public": true}})
	require.Equal(t, http.StatusForbidden, r.Code, "update: non-editor staff must be denied")

	r = integration.MakeRequest(t, g, integration.HTTPRequest{Method: "DELETE", Path: fmt.Sprintf("/admin/solution/%d", sol.ID), Headers: hdr})
	require.Equal(t, http.StatusForbidden, r.Code, "delete: non-editor staff must be denied")
	var cnt int64
	testDB.DB.Model(&models.Solution{}).Where("id = ?", sol.ID).Count(&cnt)
	require.Equal(t, int64(1), cnt, "solution must survive the denied delete")

	r = integration.MakeRequest(t, g, integration.HTTPRequest{Method: "POST", Path: "/admin/solutions", Headers: hdr, Body: map[string]interface{}{"problem_id": prob.ID, "content": "x"}})
	require.Equal(t, http.StatusForbidden, r.Code, "create: non-editor staff must be denied")
}
