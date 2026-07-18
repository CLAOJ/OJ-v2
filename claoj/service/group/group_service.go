// Package group provides Django-group (role) management services.
package group

import (
	"github.com/CLAOJ/claoj/auth"
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"gorm.io/gorm"
)

// GroupService manages Django groups (roles) and their permissions by
// writing rows into Django's own auth_group / auth_group_permissions /
// auth_user_groups tables -- exactly like Django admin does.
type GroupService struct{}

// NewGroupService creates a new GroupService instance.
func NewGroupService() *GroupService {
	return &GroupService{}
}

// ListGroups returns a summary of every Django group, with user and
// permission counts.
func (s *GroupService) ListGroups() ([]GroupSummary, error) {
	var groups []models.AuthGroup
	if err := db.DB.Order("name ASC").Find(&groups).Error; err != nil {
		return nil, err
	}

	result := make([]GroupSummary, len(groups))
	for i, g := range groups {
		var permCount int64
		if err := db.DB.Model(&models.AuthGroupPermission{}).Where("group_id = ?", g.ID).Count(&permCount).Error; err != nil {
			return nil, err
		}
		var userCount int64
		if err := db.DB.Model(&models.AuthUserGroup{}).Where("group_id = ?", g.ID).Count(&userCount).Error; err != nil {
			return nil, err
		}
		result[i] = GroupSummary{
			ID:              g.ID,
			Name:            g.Name,
			UserCount:       int(userCount),
			PermissionCount: int(permCount),
		}
	}
	return result, nil
}

// GetGroup returns full detail for a single group: its permission IDs and
// member users (id, username).
func (s *GroupService) GetGroup(id uint) (*GroupDetail, error) {
	var g models.AuthGroup
	if err := db.DB.First(&g, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrGroupNotFound
		}
		return nil, err
	}

	var permIDs []uint
	if err := db.DB.Model(&models.AuthGroupPermission{}).
		Where("group_id = ?", id).
		Pluck("permission_id", &permIDs).Error; err != nil {
		return nil, err
	}
	if permIDs == nil {
		permIDs = []uint{}
	}

	var users []GroupUser
	if err := db.DB.Table("auth_user").
		Select("auth_user.id AS id, auth_user.username AS username").
		Joins("JOIN auth_user_groups ON auth_user_groups.user_id = auth_user.id").
		Where("auth_user_groups.group_id = ?", id).
		Order("auth_user.username ASC").
		Scan(&users).Error; err != nil {
		return nil, err
	}
	if users == nil {
		users = []GroupUser{}
	}

	return &GroupDetail{
		ID:            g.ID,
		Name:          g.Name,
		PermissionIDs: permIDs,
		Users:         users,
	}, nil
}

// CreateGroup creates a new Django group with the given permission set.
func (s *GroupService) CreateGroup(name string, permissionIDs []uint) (*GroupSummary, error) {
	if name == "" {
		return nil, ErrEmptyGroupName
	}

	var existing models.AuthGroup
	if err := db.DB.Where("name = ?", name).First(&existing).Error; err == nil {
		return nil, ErrGroupNameExists
	}

	g := models.AuthGroup{Name: name}
	if err := db.DB.Create(&g).Error; err != nil {
		return nil, err
	}

	if len(permissionIDs) > 0 {
		if err := db.DB.Model(&g).Association("Permissions").Replace(idsToPermissions(permissionIDs)); err != nil {
			return nil, err
		}
	}

	auth.BumpPermVersion()

	return &GroupSummary{
		ID:              g.ID,
		Name:            g.Name,
		UserCount:       0,
		PermissionCount: len(permissionIDs),
	}, nil
}

// UpdateGroup updates a group's name and/or its permission set. Nil fields
// are left unchanged; a non-nil PermissionIDs slice fully replaces the
// group's permission set (including an empty slice, which clears it).
func (s *GroupService) UpdateGroup(id uint, name *string, permissionIDs *[]uint) error {
	var g models.AuthGroup
	if err := db.DB.First(&g, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrGroupNotFound
		}
		return err
	}

	if name != nil && *name != g.Name {
		var existing models.AuthGroup
		if err := db.DB.Where("name = ? AND id <> ?", *name, id).First(&existing).Error; err == nil {
			return ErrGroupNameExists
		}
		if err := db.DB.Model(&g).Update("name", *name).Error; err != nil {
			return err
		}
	}

	if permissionIDs != nil {
		if err := db.DB.Model(&g).Association("Permissions").Replace(idsToPermissions(*permissionIDs)); err != nil {
			return err
		}
	}

	auth.BumpPermVersion()
	return nil
}

// DeleteGroup deletes a group along with its join rows in
// auth_group_permissions and auth_user_groups.
func (s *GroupService) DeleteGroup(id uint) error {
	var g models.AuthGroup
	if err := db.DB.First(&g, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrGroupNotFound
		}
		return err
	}

	if err := db.DB.Model(&g).Association("Permissions").Clear(); err != nil {
		return err
	}
	if err := db.DB.Where("group_id = ?", id).Delete(&models.AuthUserGroup{}).Error; err != nil {
		return err
	}
	if err := db.DB.Delete(&g).Error; err != nil {
		return err
	}

	auth.BumpPermVersion()
	return nil
}

// ListPermissions returns every Django permission, with its full
// "{app_label}.{codename}" wire identity.
func (s *GroupService) ListPermissions() ([]PermissionInfo, error) {
	var rows []struct {
		ID       uint
		Name     string
		Codename string
		AppLabel string
		Model    string
	}
	err := db.DB.
		Table("auth_permission").
		Select("auth_permission.id AS id, auth_permission.name AS name, auth_permission.codename AS codename, "+
			"django_content_type.app_label AS app_label, django_content_type.model AS model").
		Joins("JOIN django_content_type ON django_content_type.id = auth_permission.content_type_id").
		Order("django_content_type.app_label ASC, auth_permission.codename ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	result := make([]PermissionInfo, len(rows))
	for i, r := range rows {
		result[i] = PermissionInfo{
			ID:       r.ID,
			Codename: r.AppLabel + "." + r.Codename,
			Name:     r.Name,
			AppLabel: r.AppLabel,
			Model:    r.Model,
		}
	}
	return result, nil
}

// AddUserToGroup adds a user to a Django group by writing an
// auth_user_groups row. userID/groupID are the raw auth_user.id /
// auth_group.id primary keys. Idempotent: adding an already-member user is
// a no-op success.
func (s *GroupService) AddUserToGroup(userID, groupID uint) error {
	var user models.AuthUser
	if err := db.DB.First(&user, userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrUserNotFound
		}
		return err
	}
	var g models.AuthGroup
	if err := db.DB.First(&g, groupID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrGroupNotFound
		}
		return err
	}

	join := models.AuthUserGroup{UserID: userID, GroupID: groupID}
	if err := db.DB.Where(join).FirstOrCreate(&join).Error; err != nil {
		return err
	}

	auth.BumpPermVersion()
	return nil
}

// RemoveUserFromGroup removes a user from a Django group by deleting the
// auth_user_groups row. Idempotent: removing a non-member is a no-op
// success.
func (s *GroupService) RemoveUserFromGroup(userID, groupID uint) error {
	if err := db.DB.Where("user_id = ? AND group_id = ?", userID, groupID).
		Delete(&models.AuthUserGroup{}).Error; err != nil {
		return err
	}

	auth.BumpPermVersion()
	return nil
}

// idsToPermissions builds the []models.AuthPermission slice GORM needs to
// reference existing auth_permission rows by primary key for
// Association("Permissions").Replace.
func idsToPermissions(ids []uint) []models.AuthPermission {
	perms := make([]models.AuthPermission, len(ids))
	for i, id := range ids {
		perms[i] = models.AuthPermission{ID: id}
	}
	return perms
}
