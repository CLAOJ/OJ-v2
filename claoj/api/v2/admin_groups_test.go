package v2

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// ============================================================
// Regression tests for the privilege-escalation gap where Django
// group/permission MUTATION endpoints were only behind is_staff
// (AdminRequiredMiddleware). A staff-but-not-superuser user could
// create/edit groups, grant permissions, or add themselves to a
// powerful group. Django treats group/permission mutation as
// superuser-level, so these handlers now require IsSuperuser.
// ============================================================

// setupAdminGroupsTestDB points db.DB at a fresh in-memory sqlite database
// migrated with the Django auth models these handlers read/write, plus
// judge_profile (needed by resolveAuthUserID for the /admin/user/:id/groups
// family), mirroring the pattern in service/group/group_service_test.go's
// setupGroupDB and auth/perms_test.go's setupPermsDB.
func setupAdminGroupsTestDB(t *testing.T) {
	t.Helper()
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, database.AutoMigrate(
		&models.AuthUser{}, &models.AuthGroup{}, &models.DjangoContentType{},
		&models.AuthPermission{}, &models.AuthUserGroup{}, &models.AuthGroupPermission{},
		&models.AuthUserPermission{}, &models.Profile{},
	))
	db.DB = database
}

// newStaffUser creates an active, staff-but-not-superuser AuthUser -- the
// exact identity the finding says could escalate its own privileges.
func newStaffUser(t *testing.T) models.AuthUser {
	t.Helper()
	u := models.AuthUser{Username: "staff", IsActive: true, IsStaff: true, IsSuperuser: false}
	require.NoError(t, db.DB.Create(&u).Error)
	return u
}

// newSuperuserUser creates an active superuser AuthUser.
func newSuperuserUser(t *testing.T) models.AuthUser {
	t.Helper()
	u := models.AuthUser{Username: "root", IsActive: true, IsStaff: true, IsSuperuser: true}
	require.NoError(t, db.DB.Create(&u).Error)
	return u
}

// newProfileForAdminGroups creates a judge_profile row for the given
// auth_user, needed because AdminUserAddGroup/AdminUserRemoveGroup take a
// judge_profile.id in :id and resolve it to the auth_user.id internally.
func newProfileForAdminGroups(t *testing.T, userID uint) models.Profile {
	t.Helper()
	p := models.Profile{UserID: userID, Timezone: "UTC"}
	require.NoError(t, db.DB.Create(&p).Error)
	return p
}

// authedRouter builds a single-route gin router that stamps
// c.Set("user_id", userID) on the context before dispatching to handler --
// cache.Client is nil in tests, so auth.GetAccess -> LoadUserAccess resolves
// IsSuperuser straight from the seeded AuthUser row. Mirrors the pattern in
// verify_test.go's TestResendVerification_AuthenticatedUser.
func authedRouter(userID uint, method, path string, handler gin.HandlerFunc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Handle(method, path, func(c *gin.Context) {
		c.Set("user_id", userID)
		handler(c)
	})
	return router
}

func doRequest(router *gin.Engine, method, target, body string) *httptest.ResponseRecorder {
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, target, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, target, nil)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// TestAdminGroupMutations_StaffNotSuperuser_Forbidden proves the gate: a
// staff-but-not-superuser user hitting any Django group/permission mutation
// endpoint gets 403, before any group/user lookup or DB write happens.
func TestAdminGroupMutations_StaffNotSuperuser_Forbidden(t *testing.T) {
	cases := []struct {
		name    string
		method  string
		route   string
		target  string
		handler gin.HandlerFunc
	}{
		{"AdminGroupCreate", http.MethodPost, "/admin/groups", "/admin/groups", AdminGroupCreate},
		{"AdminGroupUpdate", http.MethodPatch, "/admin/group/:id", "/admin/group/1", AdminGroupUpdate},
		{"AdminGroupDelete", http.MethodDelete, "/admin/group/:id", "/admin/group/1", AdminGroupDelete},
		{"AdminUserAddGroup", http.MethodPost, "/admin/user/:id/groups", "/admin/user/1/groups", AdminUserAddGroup},
		{"AdminUserRemoveGroup", http.MethodDelete, "/admin/user/:id/groups/:groupId", "/admin/user/1/groups/1", AdminUserRemoveGroup},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			setupAdminGroupsTestDB(t)
			staff := newStaffUser(t)

			router := authedRouter(staff.ID, tc.method, tc.route, tc.handler)
			w := doRequest(router, tc.method, tc.target, "")

			require.Equal(t, http.StatusForbidden, w.Code, "body: %s", w.Body.String())
		})
	}
}

// TestAdminGroupMutations_Superuser_NotBlockedByGate proves the gate does
// not false-positive on a genuine superuser: each handler is exercised
// end-to-end and must not return the gate's 403, and in fact succeeds.
func TestAdminGroupMutations_Superuser_NotBlockedByGate(t *testing.T) {
	t.Run("AdminGroupCreate", func(t *testing.T) {
		setupAdminGroupsTestDB(t)
		root := newSuperuserUser(t)

		router := authedRouter(root.ID, http.MethodPost, "/admin/groups", AdminGroupCreate)
		w := doRequest(router, http.MethodPost, "/admin/groups", `{"name":"Judges"}`)

		require.NotEqual(t, http.StatusForbidden, w.Code, "body: %s", w.Body.String())
		require.Equal(t, http.StatusCreated, w.Code, "body: %s", w.Body.String())
	})

	t.Run("AdminGroupUpdate", func(t *testing.T) {
		setupAdminGroupsTestDB(t)
		root := newSuperuserUser(t)
		group := models.AuthGroup{Name: "Judges"}
		require.NoError(t, db.DB.Create(&group).Error)

		router := authedRouter(root.ID, http.MethodPatch, "/admin/group/:id", AdminGroupUpdate)
		w := doRequest(router, http.MethodPatch, "/admin/group/1", `{"name":"Senior Judges"}`)

		require.NotEqual(t, http.StatusForbidden, w.Code, "body: %s", w.Body.String())
		require.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())
	})

	t.Run("AdminGroupDelete", func(t *testing.T) {
		setupAdminGroupsTestDB(t)
		root := newSuperuserUser(t)
		group := models.AuthGroup{Name: "Judges"}
		require.NoError(t, db.DB.Create(&group).Error)

		router := authedRouter(root.ID, http.MethodDelete, "/admin/group/:id", AdminGroupDelete)
		w := doRequest(router, http.MethodDelete, "/admin/group/1", "")

		require.NotEqual(t, http.StatusForbidden, w.Code, "body: %s", w.Body.String())
		require.Equal(t, http.StatusNoContent, w.Code, "body: %s", w.Body.String())
	})

	t.Run("AdminUserAddGroup", func(t *testing.T) {
		setupAdminGroupsTestDB(t)
		root := newSuperuserUser(t)
		group := models.AuthGroup{Name: "Judges"}
		require.NoError(t, db.DB.Create(&group).Error)
		target := newStaffUser(t) // the user being granted the group
		profile := newProfileForAdminGroups(t, target.ID)

		router := authedRouter(root.ID, http.MethodPost, "/admin/user/:id/groups", AdminUserAddGroup)
		w := doRequest(router, http.MethodPost, "/admin/user/"+strconv.Itoa(int(profile.ID))+"/groups", `{"group_id":1}`)

		require.NotEqual(t, http.StatusForbidden, w.Code, "body: %s", w.Body.String())
		require.Equal(t, http.StatusNoContent, w.Code, "body: %s", w.Body.String())
	})

	t.Run("AdminUserRemoveGroup", func(t *testing.T) {
		setupAdminGroupsTestDB(t)
		root := newSuperuserUser(t)
		group := models.AuthGroup{Name: "Judges"}
		require.NoError(t, db.DB.Create(&group).Error)
		target := newStaffUser(t)
		profile := newProfileForAdminGroups(t, target.ID)
		require.NoError(t, db.DB.Create(&models.AuthUserGroup{UserID: target.ID, GroupID: group.ID}).Error)

		router := authedRouter(root.ID, http.MethodDelete, "/admin/user/:id/groups/:groupId", AdminUserRemoveGroup)
		w := doRequest(router, http.MethodDelete, "/admin/user/"+strconv.Itoa(int(profile.ID))+"/groups/"+strconv.Itoa(int(group.ID)), "")

		require.NotEqual(t, http.StatusForbidden, w.Code, "body: %s", w.Body.String())
		require.Equal(t, http.StatusNoContent, w.Code, "body: %s", w.Body.String())
	})
}

