package group

import (
	"testing"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupGroupDB points db.DB at a fresh in-memory sqlite database migrated
// with the Django auth models this service reads and writes, mirroring the
// pattern in claoj/auth/perms_test.go's setupPermsDB.
func setupGroupDB(t *testing.T) {
	t.Helper()
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, database.AutoMigrate(
		&models.AuthUser{}, &models.AuthGroup{}, &models.DjangoContentType{},
		&models.AuthPermission{}, &models.AuthUserGroup{}, &models.AuthGroupPermission{},
		&models.Profile{},
	))
	db.DB = database
}

// seedPermission creates an {appLabel}.{codename} permission (model always
// "problem", mirroring auth/perms_test.go's seedPerm) and returns its ID.
func seedPermission(t *testing.T, appLabel, codename string) uint {
	t.Helper()
	var ct models.DjangoContentType
	require.NoError(t, db.DB.FirstOrCreate(&ct, models.DjangoContentType{AppLabel: appLabel, Model: "problem"}).Error)
	perm := models.AuthPermission{Name: codename, Codename: codename, ContentTypeID: ct.ID}
	require.NoError(t, db.DB.Create(&perm).Error)
	return perm.ID
}

func newUser(t *testing.T, username string) models.AuthUser {
	t.Helper()
	u := models.AuthUser{Username: username, IsActive: true}
	require.NoError(t, db.DB.Create(&u).Error)
	return u
}

// newProfile creates a judge_profile row for the given auth_user, mirroring
// the pattern in api/v2/admin_groups_test.go's newProfileForAdminGroups.
// GetGroup's member list must return judge_profile.id (not auth_user.id) so
// its output is consistent with the judge_profile.id the /admin/user/:id
// route family expects.
func newProfile(t *testing.T, userID uint) models.Profile {
	t.Helper()
	p := models.Profile{UserID: userID, Timezone: "UTC"}
	require.NoError(t, db.DB.Create(&p).Error)
	return p
}

func TestCreateGroup_WithPermissions_ListShowsCounts(t *testing.T) {
	setupGroupDB(t)
	svc := NewGroupService()

	p1 := seedPermission(t, "judge", "edit_all_problem")
	p2 := seedPermission(t, "judge", "rejudge_submission")

	created, err := svc.CreateGroup("Judges", []uint{p1, p2})
	require.NoError(t, err)
	require.NotZero(t, created.ID)
	require.Equal(t, "Judges", created.Name)

	list, err := svc.ListGroups()
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, "Judges", list[0].Name)
	require.Equal(t, 2, list[0].PermissionCount)
	require.Equal(t, 0, list[0].UserCount)
}

func TestCreateGroup_EmptyName_ReturnsErrEmptyGroupName(t *testing.T) {
	setupGroupDB(t)
	svc := NewGroupService()

	_, err := svc.CreateGroup("", nil)
	require.ErrorIs(t, err, ErrEmptyGroupName)
}

func TestCreateGroup_DuplicateName_ReturnsErrGroupNameExists(t *testing.T) {
	setupGroupDB(t)
	svc := NewGroupService()

	_, err := svc.CreateGroup("Judges", nil)
	require.NoError(t, err)

	_, err = svc.CreateGroup("Judges", nil)
	require.ErrorIs(t, err, ErrGroupNameExists)
}

func TestCreateGroup_UnknownPermissionID_ReturnsErrUnknownPermissionID(t *testing.T) {
	setupGroupDB(t)
	svc := NewGroupService()

	p1 := seedPermission(t, "judge", "edit_all_problem")

	_, err := svc.CreateGroup("Judges", []uint{p1, 9999})
	require.ErrorIs(t, err, ErrUnknownPermissionID)

	var groupCount int64
	require.NoError(t, db.DB.Model(&models.AuthGroup{}).Count(&groupCount).Error)
	require.Equal(t, int64(0), groupCount, "no group row should have been created")

	var permCount int64
	require.NoError(t, db.DB.Model(&models.AuthPermission{}).Count(&permCount).Error)
	require.Equal(t, int64(1), permCount, "no phantom auth_permission row should have been created")
}

func TestCreateGroup_DuplicateValidPermissionIDs_DoesNotError(t *testing.T) {
	setupGroupDB(t)
	svc := NewGroupService()

	p1 := seedPermission(t, "judge", "edit_all_problem")

	created, err := svc.CreateGroup("Judges", []uint{p1, p1})
	require.NoError(t, err)

	detail, err := svc.GetGroup(created.ID)
	require.NoError(t, err)
	require.ElementsMatch(t, []uint{p1}, detail.PermissionIDs)
}

func TestUpdateGroup_UnknownPermissionID_ReturnsErrUnknownPermissionIDAndLeavesPermissionsUnchanged(t *testing.T) {
	setupGroupDB(t)
	svc := NewGroupService()

	p1 := seedPermission(t, "judge", "edit_all_problem")

	created, err := svc.CreateGroup("Judges", []uint{p1})
	require.NoError(t, err)

	badPerms := []uint{9999}
	err = svc.UpdateGroup(created.ID, nil, &badPerms)
	require.ErrorIs(t, err, ErrUnknownPermissionID)

	detail, err := svc.GetGroup(created.ID)
	require.NoError(t, err)
	require.ElementsMatch(t, []uint{p1}, detail.PermissionIDs, "existing permission set must be unchanged")

	var permCount int64
	require.NoError(t, db.DB.Model(&models.AuthPermission{}).Count(&permCount).Error)
	require.Equal(t, int64(1), permCount, "no phantom auth_permission row should have been created")
}

func TestUpdateGroup_DuplicateValidPermissionIDs_DoesNotError(t *testing.T) {
	setupGroupDB(t)
	svc := NewGroupService()

	p1 := seedPermission(t, "judge", "edit_all_problem")

	created, err := svc.CreateGroup("Judges", nil)
	require.NoError(t, err)

	dupPerms := []uint{p1, p1}
	require.NoError(t, svc.UpdateGroup(created.ID, nil, &dupPerms))

	detail, err := svc.GetGroup(created.ID)
	require.NoError(t, err)
	require.ElementsMatch(t, []uint{p1}, detail.PermissionIDs)
}

func TestUpdateGroup_ReplacesPermissionSet(t *testing.T) {
	setupGroupDB(t)
	svc := NewGroupService()

	p1 := seedPermission(t, "judge", "edit_all_problem")
	p2 := seedPermission(t, "judge", "rejudge_submission")
	p3 := seedPermission(t, "judge", "see_private_problem")

	created, err := svc.CreateGroup("Judges", []uint{p1, p2})
	require.NoError(t, err)

	newName := "Senior Judges"
	newPerms := []uint{p3}
	require.NoError(t, svc.UpdateGroup(created.ID, &newName, &newPerms))

	detail, err := svc.GetGroup(created.ID)
	require.NoError(t, err)
	require.Equal(t, "Senior Judges", detail.Name)
	require.ElementsMatch(t, []uint{p3}, detail.PermissionIDs)
}

func TestUpdateGroup_DuplicateName_ReturnsErrGroupNameExists(t *testing.T) {
	setupGroupDB(t)
	svc := NewGroupService()

	_, err := svc.CreateGroup("Judges", nil)
	require.NoError(t, err)
	other, err := svc.CreateGroup("Testers", nil)
	require.NoError(t, err)

	dup := "Judges"
	err = svc.UpdateGroup(other.ID, &dup, nil)
	require.ErrorIs(t, err, ErrGroupNameExists)
}

func TestUpdateGroup_NotFound(t *testing.T) {
	setupGroupDB(t)
	svc := NewGroupService()

	name := "Ghost"
	err := svc.UpdateGroup(999, &name, nil)
	require.ErrorIs(t, err, ErrGroupNotFound)
}

func TestDeleteGroup_RemovesJoinRows(t *testing.T) {
	setupGroupDB(t)
	svc := NewGroupService()

	p1 := seedPermission(t, "judge", "edit_all_problem")
	created, err := svc.CreateGroup("Judges", []uint{p1})
	require.NoError(t, err)

	user := newUser(t, "alice")
	require.NoError(t, svc.AddUserToGroup(user.ID, created.ID))

	require.NoError(t, svc.DeleteGroup(created.ID))

	var gpCount int64
	require.NoError(t, db.DB.Model(&models.AuthGroupPermission{}).Where("group_id = ?", created.ID).Count(&gpCount).Error)
	require.Equal(t, int64(0), gpCount)

	var ugCount int64
	require.NoError(t, db.DB.Model(&models.AuthUserGroup{}).Where("group_id = ?", created.ID).Count(&ugCount).Error)
	require.Equal(t, int64(0), ugCount)

	var groupCount int64
	require.NoError(t, db.DB.Model(&models.AuthGroup{}).Where("id = ?", created.ID).Count(&groupCount).Error)
	require.Equal(t, int64(0), groupCount)

	_, err = svc.GetGroup(created.ID)
	require.ErrorIs(t, err, ErrGroupNotFound)
}

func TestDeleteGroup_NotFound(t *testing.T) {
	setupGroupDB(t)
	svc := NewGroupService()

	require.ErrorIs(t, svc.DeleteGroup(999), ErrGroupNotFound)
}

func TestAddRemoveUserFromGroup(t *testing.T) {
	setupGroupDB(t)
	svc := NewGroupService()

	created, err := svc.CreateGroup("Judges", nil)
	require.NoError(t, err)
	// Seed a throwaway auth_user first so alice's auth_user.id and
	// judge_profile.id diverge (both tables auto-increment from 1
	// independently) -- otherwise this assertion would pass trivially
	// against either id space.
	_ = newUser(t, "offset")
	user := newUser(t, "alice")
	profile := newProfile(t, user.ID)
	require.NotEqual(t, user.ID, profile.ID, "test setup must make auth_user.id and judge_profile.id diverge")

	require.NoError(t, svc.AddUserToGroup(user.ID, created.ID))

	detail, err := svc.GetGroup(created.ID)
	require.NoError(t, err)
	require.Len(t, detail.Users, 1)
	require.Equal(t, "alice", detail.Users[0].Username)
	require.Equal(t, profile.ID, detail.Users[0].ID, "GetGroup must return judge_profile.id, matching the id space the /admin/user/:id/groups routes expect")

	list, err := svc.ListGroups()
	require.NoError(t, err)
	require.Equal(t, 1, list[0].UserCount)

	// Adding again is idempotent (no duplicate row, no error).
	require.NoError(t, svc.AddUserToGroup(user.ID, created.ID))
	var ugCount int64
	require.NoError(t, db.DB.Model(&models.AuthUserGroup{}).
		Where("user_id = ? AND group_id = ?", user.ID, created.ID).Count(&ugCount).Error)
	require.Equal(t, int64(1), ugCount)

	require.NoError(t, svc.RemoveUserFromGroup(user.ID, created.ID))

	detail, err = svc.GetGroup(created.ID)
	require.NoError(t, err)
	require.Len(t, detail.Users, 0)

	// Removing again is a no-op success, not an error.
	require.NoError(t, svc.RemoveUserFromGroup(user.ID, created.ID))
}

func TestAddUserToGroup_UnknownUserOrGroup(t *testing.T) {
	setupGroupDB(t)
	svc := NewGroupService()

	created, err := svc.CreateGroup("Judges", nil)
	require.NoError(t, err)
	user := newUser(t, "alice")

	require.ErrorIs(t, svc.AddUserToGroup(999, created.ID), ErrUserNotFound)
	require.ErrorIs(t, svc.AddUserToGroup(user.ID, 999), ErrGroupNotFound)
}

func TestListPermissions_ReturnsAppLabelCodenameForm(t *testing.T) {
	setupGroupDB(t)
	svc := NewGroupService()

	seedPermission(t, "judge", "edit_all_problem")

	perms, err := svc.ListPermissions()
	require.NoError(t, err)
	require.Len(t, perms, 1)
	require.Equal(t, "judge.edit_all_problem", perms[0].Codename)
	require.Equal(t, "edit_all_problem", perms[0].Name)
	require.Equal(t, "judge", perms[0].AppLabel)
	require.Equal(t, "problem", perms[0].Model)
}
