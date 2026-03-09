// Package role provides role and permission management services.
package role

import (
	"errors"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"gorm.io/gorm"
)

// RoleService provides role and permission management operations.
type RoleService struct{}

// NewRoleService creates a new RoleService instance.
func NewRoleService() *RoleService {
	return &RoleService{}
}

// ListRoles retrieves a list of all roles.
func (s *RoleService) ListRoles(req ListRolesRequest) (*ListRolesResponse, error) {
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 50
	}
	if req.PageSize > 100 {
		req.PageSize = 100
	}

	var roles []models.Role
	query := db.DB.Model(&models.Role{}).Preload("Permissions")

	// Get total count
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// Get paginated results
	if err := query.
		Order("name ASC").
		Offset((req.Page - 1) * req.PageSize).
		Limit(req.PageSize).
		Find(&roles).Error; err != nil {
		return nil, err
	}

	result := make([]Role, len(roles))
	for i, r := range roles {
		result[i] = roleToModel(r)
	}

	return &ListRolesResponse{
		Roles:      result,
		Total:      total,
		Page:       req.Page,
		PageSize:   req.PageSize,
	}, nil
}

// GetRole retrieves a role by ID with permissions.
func (s *RoleService) GetRole(req GetRoleRequest) (*RoleDetail, error) {
	if req.RoleID == 0 {
		return nil, ErrInvalidRoleID
	}

	var role models.Role
	if err := db.DB.Preload("Permissions").First(&role, req.RoleID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrRoleNotFound
		}
		return nil, err
	}

	detail := &RoleDetail{
		Role: roleToModel(role),
	}

	permissions := make([]Permission, len(role.Permissions))
	for i, p := range role.Permissions {
		permissions[i] = permissionToModel(p)
	}
	detail.Permissions = permissions

	return detail, nil
}

// CreateRole creates a new role.
func (s *RoleService) CreateRole(req CreateRoleRequest) (*Role, error) {
	if req.Name == "" {
		return nil, ErrEmptyRoleName
	}
	if req.DisplayName == "" {
		return nil, ErrEmptyDisplayName
	}

	// Check if name already exists
	var existing models.Role
	if err := db.DB.Where("name = ?", req.Name).First(&existing).Error; err == nil {
		return nil, ErrRoleNameExists
	}

	role := models.Role{
		Name:        req.Name,
		DisplayName: req.DisplayName,
		Description: req.Description,
		Color:       req.Color,
		IsDefault:   req.IsDefault,
	}

	if err := db.DB.Create(&role).Error; err != nil {
		return nil, err
	}

	// Associate permissions if provided
	if len(req.PermissionIDs) > 0 {
		var permissions []models.Permission
		if err := db.DB.Where("id IN ?", req.PermissionIDs).Find(&permissions).Error; err == nil {
			db.DB.Model(&role).Association("Permissions").Append(&permissions)
		}
	}

	// Reload role with permissions
	db.DB.Preload("Permissions").First(&role, role.ID)

	result := roleToModel(role)
	return &result, nil
}

// UpdateRole updates an existing role.
func (s *RoleService) UpdateRole(req UpdateRoleRequest) (*Role, error) {
	if req.RoleID == 0 {
		return nil, ErrInvalidRoleID
	}

	var role models.Role
	if err := db.DB.First(&role, req.RoleID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrRoleNotFound
		}
		return nil, err
	}

	updates := make(map[string]interface{})
	if req.DisplayName != nil {
		updates["display_name"] = *req.DisplayName
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.Color != nil {
		updates["color"] = *req.Color
	}
	if req.IsDefault != nil {
		updates["is_default"] = *req.IsDefault
	}

	if len(updates) > 0 {
		if err := db.DB.Model(&role).Updates(updates).Error; err != nil {
			return nil, err
		}
	}

	// Update permissions if provided
	if req.PermissionIDs != nil {
		// Clear existing permissions
		db.DB.Model(&role).Association("Permissions").Clear()

		// Add new permissions
		if len(req.PermissionIDs) > 0 {
			var permissions []models.Permission
			if err := db.DB.Where("id IN ?", req.PermissionIDs).Find(&permissions).Error; err == nil {
				db.DB.Model(&role).Association("Permissions").Append(&permissions)
			}
		}
	}

	// Reload role with permissions
	db.DB.Preload("Permissions").First(&role, role.ID)

	result := roleToModel(role)
	return &result, nil
}

// DeleteRole deletes a role (prevents deletion of default roles).
func (s *RoleService) DeleteRole(req DeleteRoleRequest) error {
	if req.RoleID == 0 {
		return ErrInvalidRoleID
	}

	var role models.Role
	if err := db.DB.First(&role, req.RoleID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrRoleNotFound
		}
		return err
	}

	// Prevent deletion of default roles
	if role.IsDefault {
		return ErrCannotDeleteDefault
	}

	return db.DB.Delete(&role).Error
}

// AssignRole assigns a role to a profile.
func (s *RoleService) AssignRole(req AssignRoleRequest) error {
	if req.ProfileID == 0 || req.RoleID == 0 {
		return ErrInvalidRoleID
	}

	var role models.Role
	if err := db.DB.First(&role, req.RoleID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrRoleNotFound
		}
		return err
	}

	var profile models.Profile
	if err := db.DB.First(&profile, req.ProfileID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.New("role: profile not found")
		}
		return err
	}

	return db.DB.Model(&profile).Association("Roles").Append(&role)
}

// RemoveRole removes a role from a profile.
func (s *RoleService) RemoveRole(req RemoveRoleRequest) error {
	if req.ProfileID == 0 || req.RoleID == 0 {
		return ErrInvalidRoleID
	}

	var role models.Role
	if err := db.DB.First(&role, req.RoleID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrRoleNotFound
		}
		return err
	}

	var profile models.Profile
	if err := db.DB.First(&profile, req.ProfileID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.New("role: profile not found")
		}
		return err
	}

	return db.DB.Model(&profile).Association("Roles").Delete(&role)
}

// ListPermissions retrieves a list of permissions.
func (s *RoleService) ListPermissions(req ListPermissionsRequest) (*ListPermissionsResponse, error) {
	query := db.DB.Model(&models.Permission{})

	if req.Category != "" {
		query = query.Where("category = ?", req.Category)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	var permissions []models.Permission
	if err := query.Order("category ASC, name ASC").Find(&permissions).Error; err != nil {
		return nil, err
	}

	result := make([]Permission, len(permissions))
	for i, p := range permissions {
		result[i] = permissionToModel(p)
	}

	return &ListPermissionsResponse{
		Permissions: result,
		Total:       total,
	}, nil
}

// Helper functions

func roleToModel(r models.Role) Role {
	permissionIDs := make([]uint, len(r.Permissions))
	for i, p := range r.Permissions {
		permissionIDs[i] = p.ID
	}

	return Role{
		ID:            r.ID,
		Name:          r.Name,
		DisplayName:   r.DisplayName,
		Description:   r.Description,
		Color:         r.Color,
		IsDefault:     r.IsDefault,
		CreatedAt:     r.CreatedAt,
		UpdatedAt:     r.UpdatedAt,
		PermissionIDs: permissionIDs,
	}
}

func permissionToModel(p models.Permission) Permission {
	return Permission{
		ID:          p.ID,
		Code:        p.Code,
		Name:        p.Name,
		Description: p.Description,
		Category:    p.Category,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}
