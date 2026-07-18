package models

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupAuthTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, database.AutoMigrate(
		&AuthUser{}, &AuthGroup{}, &DjangoContentType{}, &AuthPermission{},
		&AuthUserGroup{}, &AuthGroupPermission{}, &AuthUserPermission{},
	))
	return database
}

func TestDjangoAuthTableNames(t *testing.T) {
	require.Equal(t, "auth_group", AuthGroup{}.TableName())
	require.Equal(t, "auth_permission", AuthPermission{}.TableName())
	require.Equal(t, "django_content_type", DjangoContentType{}.TableName())
	require.Equal(t, "auth_user_groups", AuthUserGroup{}.TableName())
	require.Equal(t, "auth_group_permissions", AuthGroupPermission{}.TableName())
	require.Equal(t, "auth_user_user_permissions", AuthUserPermission{}.TableName())
}

func TestGroupPermissionAssociations(t *testing.T) {
	database := setupAuthTestDB(t)
	ct := DjangoContentType{AppLabel: "judge", Model: "problem"}
	require.NoError(t, database.Create(&ct).Error)
	perm := AuthPermission{Name: "See hidden problems", Codename: "see_private_problem", ContentTypeID: ct.ID}
	require.NoError(t, database.Create(&perm).Error)
	group := AuthGroup{Name: "Editors"}
	require.NoError(t, database.Create(&group).Error)
	require.NoError(t, database.Model(&group).Association("Permissions").Append(&perm))

	var loaded AuthGroup
	require.NoError(t, database.Preload("Permissions").First(&loaded, group.ID).Error)
	require.Len(t, loaded.Permissions, 1)
	require.Equal(t, "see_private_problem", loaded.Permissions[0].Codename)
}
