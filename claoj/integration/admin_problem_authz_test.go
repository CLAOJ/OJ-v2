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
	"gorm.io/gorm"
)

// makeStaff flips is_staff / is_superuser on a user row.
func makeStaff(t *testing.T, db *gorm.DB, userID uint, staff, super bool) {
	t.Helper()
	require.NoError(t, db.Model(&models.AuthUser{}).Where("id = ?", userID).
		Updates(map[string]interface{}{"is_staff": staff, "is_superuser": super}).Error)
}

// adminProblemRouter builds the real admin chain for the problem-update route.
func adminProblemRouter() *gin.Engine {
	g := integration.TestRouter()
	g.POST("/auth/login", authHandlers.Login)
	g.Use(auth.RequiredMiddleware())
	g.Use(auth.AdminRequiredMiddleware())
	g.PATCH("/admin/problem/:code", auth.RequireProblemEdit(), v2.AdminProblemUpdate)
	return g
}

func TestAdminProblemUpdate_Authz(t *testing.T) {
	testDB := integration.SetupIntegrationDB(t)
	defer integration.CleanupDB(t, testDB)

	prob := models.Problem{Code: "px", Name: "PX", IsPublic: true}
	require.NoError(t, testDB.DB.Create(&prob).Error)

	// Plain staff, not an author -> 403.
	staff := integration.CreateTestUser(testDB.DB, "pstaff", "Password123!", true)
	makeStaff(t, testDB.DB, staff.ID, true, false)
	staffToken := loginToken(t, "pstaff", "Password123!")

	// Superuser -> 200.
	su := integration.CreateTestUser(testDB.DB, "psuper", "Password123!", true)
	makeStaff(t, testDB.DB, su.ID, true, true)
	suToken := loginToken(t, "psuper", "Password123!")

	g := adminProblemRouter()

	resp := integration.MakeRequest(t, g, integration.HTTPRequest{
		Method:  "PATCH",
		Path:    fmt.Sprintf("/admin/problem/%s", prob.Code),
		Headers: map[string]string{"Authorization": "Bearer " + staffToken},
		Body:    map[string]interface{}{"is_public": true},
	})
	require.Equal(t, http.StatusForbidden, resp.Code)

	resp = integration.MakeRequest(t, g, integration.HTTPRequest{
		Method:  "PATCH",
		Path:    fmt.Sprintf("/admin/problem/%s", prob.Code),
		Headers: map[string]string{"Authorization": "Bearer " + suToken},
		Body:    map[string]interface{}{"is_public": true},
	})
	// Not 403: the guard admits the superuser. (Assert NotEqual rather than
	// ==200 to isolate the guard from the pre-existing update service.)
	require.NotEqual(t, http.StatusForbidden, resp.Code)
}
