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

func adminUserUpdateRouter() *gin.Engine {
	g := integration.TestRouter()
	g.POST("/auth/login", authHandlers.Login)
	g.Use(auth.RequiredMiddleware())
	g.Use(auth.AdminRequiredMiddleware())
	g.PATCH("/admin/user/:id", v2.AdminUserUpdate)
	return g
}

func TestAppointOrgAdmin_RequiresOrgAuthority(t *testing.T) {
	testDB := integration.SetupIntegrationDB(t)
	defer integration.CleanupDB(t, testDB)

	org := models.Organization{Name: "OA", Slug: "oa", ShortName: "OA"}
	require.NoError(t, testDB.DB.Create(&org).Error)

	// Target whose profile we will (attempt to) appoint as org admin.
	target := integration.CreateTestUser(testDB.DB, "target", "Password123!", true)
	targetProfile := profileForUser(t, testDB.DB, target.ID)

	// Staff who does NOT administer org -> appointment forbidden.
	staff := integration.CreateTestUser(testDB.DB, "plainstaff", "Password123!", true)
	makeStaff(t, testDB.DB, staff.ID, true, false)
	staffToken := loginToken(t, "plainstaff", "Password123!")

	g := adminUserUpdateRouter()
	resp := integration.MakeRequest(t, g, integration.HTTPRequest{
		Method:  "PATCH",
		Path:    fmt.Sprintf("/admin/user/%d", targetProfile.ID),
		Headers: map[string]string{"Authorization": "Bearer " + staffToken},
		Body:    map[string]interface{}{"add_organization_admin": []uint{org.ID}},
	})
	require.Equal(t, http.StatusForbidden, resp.Code)

	// The appointment must NOT have happened.
	var count int64
	testDB.DB.Table("judge_organization_admins").
		Where("organization_id = ? AND profile_id = ?", org.ID, targetProfile.ID).Count(&count)
	require.Equal(t, int64(0), count)
}
