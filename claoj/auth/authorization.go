package auth

import (
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
)

// UserPermissions is a helper struct for permission checking
type UserPermissions struct {
	UserID      uint
	ProfileID   uint
	IsSuperuser bool
	IsStaff     bool
	Roles       []models.Role
	Permissions map[string]bool
}

// GetPermissionsFromContext extracts user permissions from gin context
func GetPermissionsFromContext(c *gin.Context) (*UserPermissions, bool) {
	userID, exists := c.Get("user_id")
	if !exists {
		return nil, false
	}

	isAdmin, _ := c.Get("is_admin")
	isSuperuser := isAdmin.(bool)

	return &UserPermissions{
		UserID:      userID.(uint),
		IsSuperuser: isSuperuser,
		IsStaff:     isSuperuser, // For backward compatibility
		Permissions: make(map[string]bool),
	}, true
}

// HasPermission checks if the user has a specific permission
func (p *UserPermissions) HasPermission(permission string) bool {
	// Superusers have all permissions
	if p.IsSuperuser {
		return true
	}

	// Load permissions if not already loaded
	if p.Permissions == nil || len(p.Permissions) == 0 {
		p.loadPermissions()
	}

	return p.Permissions[permission]
}

// HasRole checks if the user has a specific role
func (p *UserPermissions) HasRole(roleName string) bool {
	// Superusers are considered to have all roles
	if p.IsSuperuser {
		return true
	}

	// Load roles if not already loaded
	if p.Roles == nil {
		p.loadPermissions()
	}

	for _, role := range p.Roles {
		if role.Name == roleName {
			return true
		}
	}
	return false
}

// loadPermissions loads user permissions from the database
func (p *UserPermissions) loadPermissions() {
	p.Permissions = make(map[string]bool)
	p.Roles = []models.Role{}

	// Get profile with roles
	var profile models.Profile
	if err := db.DB.Preload("Roles.Permissions").First(&profile, p.UserID).Error; err != nil {
		return
	}

	p.Roles = profile.Roles

	// Collect all permissions from all roles
	for _, role := range profile.Roles {
		for _, perm := range role.Permissions {
			p.Permissions[perm.Code] = true
		}
	}
}

// HasPermission is a convenience function to check if a user has a permission
func HasPermission(c *gin.Context, permission string) bool {
	perms, ok := GetPermissionsFromContext(c)
	if !ok {
		return false
	}
	return perms.HasPermission(permission)
}

// HasRole is a convenience function to check if a user has a role
func HasRole(c *gin.Context, roleName string) bool {
	perms, ok := GetPermissionsFromContext(c)
	if !ok {
		return false
	}
	return perms.HasRole(roleName)
}

// CanEditProblem checks if user can edit a specific problem
func CanEditProblem(c *gin.Context, problemID uint) bool {
	perms, ok := GetPermissionsFromContext(c)
	if !ok {
		return false
	}

	// Superusers can edit everything
	if perms.IsSuperuser {
		return true
	}

	// Check permission
	if !perms.HasPermission(PermEditProblem) {
		return false
	}

	// Check if user is an author or curator of the problem
	var problem models.Problem
	if err := db.DB.Preload("Authors").Preload("Curators").First(&problem, problemID).Error; err != nil {
		return false
	}

	for _, author := range problem.Authors {
		if author.UserID == perms.UserID {
			return true
		}
	}

	for _, curator := range problem.Curators {
		if curator.UserID == perms.UserID {
			return true
		}
	}

	return false
}

// CanEditContest checks if user can edit a specific contest
func CanEditContest(c *gin.Context, contestID uint) bool {
	perms, ok := GetPermissionsFromContext(c)
	if !ok {
		return false
	}

	// Superusers can edit everything
	if perms.IsSuperuser {
		return true
	}

	// Check permission
	if !perms.HasPermission(PermEditContest) {
		return false
	}

	// Check if user is an author, curator, or tester of the contest
	var contest models.Contest
	if err := db.DB.Preload("Authors").Preload("Curators").Preload("Testers").First(&contest, contestID).Error; err != nil {
		return false
	}

	for _, author := range contest.Authors {
		if author.UserID == perms.UserID {
			return true
		}
	}

	for _, curator := range contest.Curators {
		if curator.UserID == perms.UserID {
			return true
		}
	}

	for _, tester := range contest.Testers {
		if tester.UserID == perms.UserID {
			return true
		}
	}

	return false
}

// CanViewProblem checks if user can view a specific problem
func CanViewProblem(c *gin.Context, problem *models.Problem) bool {
	perms, ok := GetPermissionsFromContext(c)
	if !ok {
		return false
	}

	// Public problems are visible to everyone
	if problem.IsPublic {
		return true
	}

	// Superusers can see everything
	if perms.IsSuperuser {
		return true
	}

	// Users with view_hidden permission can see hidden problems
	if perms.HasPermission(PermViewHiddenProblem) {
		return true
	}

	return false
}

// CanViewContest checks if user can view a specific contest
func CanViewContest(c *gin.Context, contest *models.Contest) bool {
	perms, ok := GetPermissionsFromContext(c)
	if !ok {
		return false
	}

	// Visible contests are visible to everyone
	if contest.IsVisible {
		return true
	}

	// Superusers can see everything
	if perms.IsSuperuser {
		return true
	}

	// Users with view_hidden permission can see hidden contests
	if perms.HasPermission(PermViewHiddenContest) {
		return true
	}

	return false
}

// CanBanUser checks if user can ban another user
func CanBanUser(c *gin.Context, targetUserID uint) bool {
	perms, ok := GetPermissionsFromContext(c)
	if !ok {
		return false
	}

	// Must have ban permission
	if !perms.HasPermission(PermBanUser) {
		return false
	}

	// Can't ban superusers (unless you're a superuser)
	var target models.AuthUser
	if err := db.DB.First(&target, targetUserID).Error; err != nil {
		return false
	}

	if target.IsSuperuser && !perms.IsSuperuser {
		return false
	}

	return true
}

// LoadUserPermissions loads and returns permissions for a user
func LoadUserPermissions(userID uint) *UserPermissions {
	perms := &UserPermissions{
		UserID:      userID,
		Permissions: make(map[string]bool),
	}
	perms.loadPermissions()
	return perms
}

// GetUserPermissionsFromProfileID gets permissions using profile ID
func GetUserPermissionsFromProfileID(profileID uint) *UserPermissions {
	var profile models.Profile
	if err := db.DB.Preload("User").Preload("Roles.Permissions").First(&profile, profileID).Error; err != nil {
		return &UserPermissions{Permissions: make(map[string]bool)}
	}

	perms := &UserPermissions{
		UserID:      profile.UserID,
		ProfileID:   profile.ID,
		IsSuperuser: profile.User.IsSuperuser,
		IsStaff:     profile.User.IsStaff,
		Roles:       profile.Roles,
		Permissions: make(map[string]bool),
	}

	// Collect permissions from roles
	for _, role := range profile.Roles {
		for _, perm := range role.Permissions {
			perms.Permissions[perm.Code] = true
		}
	}

	// Superusers have all permissions
	if perms.IsSuperuser {
		for _, code := range DefaultPermissionSets["admin"] {
			perms.Permissions[code] = true
		}
	}

	return perms
}
