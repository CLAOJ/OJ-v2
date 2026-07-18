package auth

import (
	"testing"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupPermsDB(t *testing.T) {
	t.Helper()
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, database.AutoMigrate(
		&models.AuthUser{}, &models.AuthGroup{}, &models.DjangoContentType{},
		&models.AuthPermission{}, &models.AuthUserGroup{},
		&models.AuthGroupPermission{}, &models.AuthUserPermission{},
	))
	db.DB = database
}

// seedPerm creates judge.<codename> and returns its ID.
func seedPerm(t *testing.T, codename string) uint {
	t.Helper()
	var ct models.DjangoContentType
	require.NoError(t, db.DB.FirstOrCreate(&ct, models.DjangoContentType{AppLabel: "judge", Model: "problem"}).Error)
	perm := models.AuthPermission{Name: codename, Codename: codename, ContentTypeID: ct.ID}
	require.NoError(t, db.DB.Create(&perm).Error)
	return perm.ID
}

func TestResolve_GroupAndDirectPerms(t *testing.T) {
	setupPermsDB(t)
	user := models.AuthUser{Username: "alice", IsActive: true}
	require.NoError(t, db.DB.Create(&user).Error)

	directID := seedPerm(t, "see_private_problem")
	groupPermID := seedPerm(t, "rejudge_submission")
	group := models.AuthGroup{Name: "Judges"}
	require.NoError(t, db.DB.Create(&group).Error)
	require.NoError(t, db.DB.Create(&models.AuthUserPermission{UserID: user.ID, PermissionID: directID}).Error)
	require.NoError(t, db.DB.Create(&models.AuthGroupPermission{GroupID: group.ID, PermissionID: groupPermID}).Error)
	require.NoError(t, db.DB.Create(&models.AuthUserGroup{UserID: user.ID, GroupID: group.ID}).Error)

	access, err := LoadUserAccess(user.ID)
	require.NoError(t, err)
	require.True(t, access.Perms.Has("judge.see_private_problem"))
	require.True(t, access.Perms.Has("judge.rejudge_submission"))
	require.False(t, access.Perms.Has("judge.edit_all_problem"))
}

func TestResolve_InactiveDeniesAll_SuperuserAllowsAll(t *testing.T) {
	setupPermsDB(t)
	inactive := models.AuthUser{Username: "banned", IsActive: false}
	super := models.AuthUser{Username: "root", IsActive: true, IsSuperuser: true}
	require.NoError(t, db.DB.Create(&inactive).Error)
	require.NoError(t, db.DB.Create(&super).Error)

	a1, err := LoadUserAccess(inactive.ID)
	require.NoError(t, err)
	require.False(t, a1.HasPerm("judge.see_private_problem"))

	a2, err := LoadUserAccess(super.ID)
	require.NoError(t, err)
	require.True(t, a2.HasPerm("judge.anything_at_all"))
}

func TestAnonymousAccessDeniesAll(t *testing.T) {
	a := AnonymousAccess()
	require.False(t, a.HasPerm("judge.see_private_problem"))
	require.False(t, a.IsStaff)
}
