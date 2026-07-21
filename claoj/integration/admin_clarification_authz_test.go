package integration_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	v2 "github.com/CLAOJ/claoj/api/v2"
	authHandlers "github.com/CLAOJ/claoj/api/v2/auth"
	"github.com/CLAOJ/claoj/auth"
	"github.com/CLAOJ/claoj/integration"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func adminClarificationDeleteRouter() *gin.Engine {
	g := integration.TestRouter()
	g.POST("/auth/login", authHandlers.Login)
	g.Use(auth.RequiredMiddleware())
	g.Use(auth.AdminRequiredMiddleware())
	g.DELETE("/admin/problem/clarification/:id", v2.ProblemClarificationDelete)
	return g
}

func TestClarificationDelete_Authz(t *testing.T) {
	testDB := integration.SetupIntegrationDB(t)
	defer integration.CleanupDB(t, testDB)
	// ProblemClarification is not in the harness AutoMigrate list — add it here.
	require.NoError(t, testDB.DB.AutoMigrate(&models.ProblemClarification{}))

	prob := models.Problem{Code: "pc", Name: "PC", IsPublic: true}
	require.NoError(t, testDB.DB.Create(&prob).Error)
	clar := models.ProblemClarification{ProblemID: prob.ID, Description: "d", Date: time.Now()}
	require.NoError(t, testDB.DB.Create(&clar).Error)

	staff := integration.CreateTestUser(testDB.DB, "clstaff", "Password123!", true)
	makeStaff(t, testDB.DB, staff.ID, true, false)
	staffToken := loginToken(t, "clstaff", "Password123!")

	g := adminClarificationDeleteRouter()
	resp := integration.MakeRequest(t, g, integration.HTTPRequest{
		Method:  "DELETE",
		Path:    fmt.Sprintf("/admin/problem/clarification/%d", clar.ID),
		Headers: map[string]string{"Authorization": "Bearer " + staffToken},
	})
	require.Equal(t, http.StatusForbidden, resp.Code)

	var count int64
	testDB.DB.Model(&models.ProblemClarification{}).Where("id = ?", clar.ID).Count(&count)
	require.Equal(t, int64(1), count) // not deleted
}
