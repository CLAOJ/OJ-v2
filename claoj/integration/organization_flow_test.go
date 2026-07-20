// Package integration_test provides integration tests for the organization flow.
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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// profileForUser returns the judge_profile row for a user.
func profileForUser(t *testing.T, db *gorm.DB, userID uint) models.Profile {
	t.Helper()
	var p models.Profile
	require.NoError(t, db.Where("user_id = ?", userID).First(&p).Error)
	return p
}

// loginToken logs a user in and returns their access token.
func loginToken(t *testing.T, username, password string) string {
	t.Helper()
	loginGin := integration.TestRouter()
	loginGin.POST("/auth/login", authHandlers.Login)
	resp := integration.MakeRequest(t, loginGin, integration.HTTPRequest{
		Method: "POST",
		Path:   "/auth/login",
		Body:   map[string]interface{}{"username": username, "password": password},
	})
	require.Equal(t, http.StatusOK, resp.Code)
	token, ok := resp.JSONBody["access_token"].(string)
	require.True(t, ok, "login response missing access_token")
	return token
}

// TestOrganizationDetail_ReturnsMembersAndAdmins verifies the detail endpoint
// exposes the member and admin lists the frontend requires (regression: it used
// to omit them, breaking every organization page).
func TestOrganizationDetail_ReturnsMembersAndAdmins(t *testing.T) {
	testDB := integration.SetupIntegrationDB(t)
	defer integration.CleanupDB(t, testDB)

	org := models.Organization{Name: "Test Org", Slug: "test-org", ShortName: "TORG", IsOpen: true}
	require.NoError(t, testDB.DB.Create(&org).Error)

	memberUser := integration.CreateTestUser(testDB.DB, "member1", "Password123!", true)
	adminUser := integration.CreateTestUser(testDB.DB, "admin1", "Password123!", true)
	memberProfile := profileForUser(t, testDB.DB, memberUser.ID)
	adminProfile := profileForUser(t, testDB.DB, adminUser.ID)

	// The admin is also a member; the plain member is not an admin.
	require.NoError(t, testDB.DB.Model(&org).Association("Members").Append(&memberProfile, &adminProfile))
	require.NoError(t, testDB.DB.Model(&org).Association("Admins").Append(&adminProfile))

	gin := integration.TestRouter()
	gin.GET("/organization/:id", v2.OrganizationDetail)

	resp := integration.MakeRequest(t, gin, integration.HTTPRequest{
		Method: "GET",
		Path:   fmt.Sprintf("/organization/%d", org.ID),
	})

	require.Equal(t, http.StatusOK, resp.Code)

	members, ok := resp.JSONBody["members"].([]interface{})
	require.True(t, ok, "response missing members array")
	assert.Len(t, members, 2)

	admins, ok := resp.JSONBody["admins"].([]interface{})
	require.True(t, ok, "response missing admins array")
	assert.Len(t, admins, 1)

	// user_id is present (0 for this anonymous request) so the client can run
	// its "am I an admin" check without crashing.
	_, hasUserID := resp.JSONBody["user_id"]
	assert.True(t, hasUserID, "response missing user_id")

	// The admin appears in the members list with the "admin" role.
	adminRoleSeen := false
	for _, m := range members {
		mm := m.(map[string]interface{})
		if mm["username"] == "admin1" {
			assert.Equal(t, "admin", mm["role"])
			adminRoleSeen = true
		}
	}
	assert.True(t, adminRoleSeen, "admin member not found in members list")
}

// TestUpdateOrganization_AdminAllowed_NonAdminForbidden verifies the new
// org-admin update endpoint authorizes correctly.
func TestUpdateOrganization_AdminAllowed_NonAdminForbidden(t *testing.T) {
	testDB := integration.SetupIntegrationDB(t)
	defer integration.CleanupDB(t, testDB)

	org := models.Organization{Name: "Test Org", Slug: "test-org", ShortName: "TORG", IsOpen: true}
	require.NoError(t, testDB.DB.Create(&org).Error)

	admin := integration.CreateTestUser(testDB.DB, "orgadmin", "Password123!", true)
	integration.CreateTestUser(testDB.DB, "outsider", "Password123!", true)
	adminProfile := profileForUser(t, testDB.DB, admin.ID)
	require.NoError(t, testDB.DB.Model(&org).Association("Admins").Append(&adminProfile))

	// Admin can update.
	adminToken := loginToken(t, "orgadmin", "Password123!")
	ginAdmin := integration.TestRouter()
	ginAdmin.Use(auth.RequiredMiddleware())
	ginAdmin.PATCH("/organization/:id", v2.UpdateOrganization)

	resp := integration.MakeRequest(t, ginAdmin, integration.HTTPRequest{
		Method:  "PATCH",
		Path:    fmt.Sprintf("/organization/%d", org.ID),
		Headers: map[string]string{"Authorization": "Bearer " + adminToken},
		Body:    map[string]interface{}{"is_open": false, "about": "Updated about"},
	})
	require.Equal(t, http.StatusOK, resp.Code)

	var updated models.Organization
	require.NoError(t, testDB.DB.First(&updated, org.ID).Error)
	assert.False(t, updated.IsOpen)
	assert.Equal(t, "Updated about", updated.About)

	// A non-admin is forbidden.
	outsiderToken := loginToken(t, "outsider", "Password123!")
	ginOut := integration.TestRouter()
	ginOut.Use(auth.RequiredMiddleware())
	ginOut.PATCH("/organization/:id", v2.UpdateOrganization)

	respForbidden := integration.MakeRequest(t, ginOut, integration.HTTPRequest{
		Method:  "PATCH",
		Path:    fmt.Sprintf("/organization/%d", org.ID),
		Headers: map[string]string{"Authorization": "Bearer " + outsiderToken},
		Body:    map[string]interface{}{"is_open": true},
	})
	assert.Equal(t, http.StatusForbidden, respForbidden.Code)

	// The forbidden request did not change anything.
	require.NoError(t, testDB.DB.First(&updated, org.ID).Error)
	assert.False(t, updated.IsOpen)
}
