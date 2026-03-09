package migrations

import (
	"time"

	"github.com/CLAOJ/claoj/auth"
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
)

// Migrate001CreateRolesAndPermissions creates the roles and permissions tables
// and seeds them with default data
func Migrate001CreateRolesAndPermissions() error {
	database := db.DB

	// Create tables
	if err := database.Exec(`
		CREATE TABLE IF NOT EXISTS judge_role (
			id INT AUTO_INCREMENT PRIMARY KEY,
			name VARCHAR(50) NOT NULL UNIQUE,
			display_name VARCHAR(100) NOT NULL,
			description TEXT,
			color VARCHAR(20) DEFAULT '#6b7280',
			is_default BOOLEAN DEFAULT FALSE,
			created_at DATETIME,
			updated_at DATETIME
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
	`).Error; err != nil {
		return err
	}

	if err := database.Exec(`
		CREATE TABLE IF NOT EXISTS judge_permission (
			id INT AUTO_INCREMENT PRIMARY KEY,
			code VARCHAR(100) NOT NULL UNIQUE,
			name VARCHAR(200) NOT NULL,
			description TEXT,
			category VARCHAR(50),
			created_at DATETIME,
			updated_at DATETIME,
			INDEX idx_category (category)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
	`).Error; err != nil {
		return err
	}

	if err := database.Exec(`
		CREATE TABLE IF NOT EXISTS judge_role_permissions (
			role_id INT NOT NULL,
			permission_id INT NOT NULL,
			PRIMARY KEY (role_id, permission_id),
			UNIQUE INDEX idx_role_perm (role_id, permission_id),
			CONSTRAINT fk_role_permissions_role FOREIGN KEY (role_id) REFERENCES judge_role(id) ON DELETE CASCADE,
			CONSTRAINT fk_role_permissions_perm FOREIGN KEY (permission_id) REFERENCES judge_permission(id) ON DELETE CASCADE
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
	`).Error; err != nil {
		return err
	}

	if err := database.Exec(`
		CREATE TABLE IF NOT EXISTS judge_profile_roles (
			profile_id INT NOT NULL,
			role_id INT NOT NULL,
			PRIMARY KEY (profile_id, role_id),
			UNIQUE INDEX idx_profile_role (profile_id, role_id),
			CONSTRAINT fk_profile_roles_profile FOREIGN KEY (profile_id) REFERENCES judge_profile(id) ON DELETE CASCADE,
			CONSTRAINT fk_profile_roles_role FOREIGN KEY (role_id) REFERENCES judge_role(id) ON DELETE CASCADE
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
	`).Error; err != nil {
		return err
	}

	now := time.Now()

	// Seed permissions
	allPermissions := map[string]struct {
		Name string
		Cat  string
		Desc string
	}{
		auth.PermCreateProblem: {"Create Problems", "problems", "Create new problems"},
		auth.PermEditProblem: {"Edit Problems", "problems", "Edit existing problems"},
		auth.PermDeleteProblem: {"Delete Problems", "problems", "Delete/hide problems"},
		auth.PermViewHiddenProblem: {"View Hidden Problems", "problems", "View hidden/unlisted problems"},
		auth.PermEditProblemData: {"Edit Problem Data", "problems", "Edit problem test data"},

		auth.PermCreateContest: {"Create Contests", "contests", "Create new contests"},
		auth.PermEditContest: {"Edit Contests", "contests", "Edit existing contests"},
		auth.PermDeleteContest: {"Delete Contests", "contests", "Delete/hide contests"},
		auth.PermViewHiddenContest: {"View Hidden Contests", "contests", "View hidden contests"},
		auth.PermManageContestProblems: {"Manage Contest Problems", "contests", "Add/remove contest problems"},

		auth.PermRejudgeSubmission: {"Rejudge Submissions", "submissions", "Rejudge submissions"},
		auth.PermViewAllSubmissions: {"View All Submissions", "submissions", "View all submissions"},
		auth.PermAccessContestSubmission: {"Contest Submission Access", "submissions", "Submit in contests"},

		auth.PermBanUser: {"Ban Users", "users", "Ban/unban users"},
		auth.PermEditUser: {"Edit Users", "users", "Edit user profiles"},
		auth.PermDeleteUser: {"Delete Users", "users", "Delete/deactivate users"},
		auth.PermViewUserEmail: {"View User Email", "users", "View user email addresses"},

		auth.PermCreateOrganization: {"Create Organizations", "organizations", "Create organizations"},
		auth.PermEditOrganization: {"Edit Organizations", "organizations", "Edit organizations"},
		auth.PermDeleteOrganization: {"Delete Organizations", "organizations", "Delete organizations"},
		auth.PermManageOrganizationMembers: {"Manage Org Members", "organizations", "Manage organization members"},

		auth.PermEditComment: {"Edit Comments", "comments", "Edit any comment"},
		auth.PermDeleteComment: {"Delete Comments", "comments", "Delete comments"},
		auth.PermPinComment: {"Pin Comments", "comments", "Pin/unpin comments"},

		auth.PermViewTicket: {"View Tickets", "tickets", "View support tickets"},
		auth.PermReplyTicket: {"Reply to Tickets", "tickets", "Reply to tickets"},
		auth.PermCloseTicket: {"Close Tickets", "tickets", "Close tickets"},

		auth.PermEditBlog: {"Edit Blogs", "blogs", "Edit any blog post"},
		auth.PermDeleteBlog: {"Delete Blogs", "blogs", "Delete blog posts"},

		auth.PermAccessAdminPanel: {"Access Admin Panel", "system", "Access admin panel"},
		auth.PermUseMOSS: {"Use MOSS", "system", "Use MOSS plagiarism detection"},
		auth.PermViewStats: {"View Stats", "system", "View site statistics"},
		auth.PermManageJudges: {"Manage Judges", "system", "Manage judge servers"},
		auth.PermManageLanguages: {"Manage Languages", "system", "Manage programming languages"},
		auth.PermManageAnnouncements: {"Manage Announcements", "system", "Create/edit announcements"},
	}

	// Insert permissions
	for code, info := range allPermissions {
		perm := models.Permission{
			Code:        code,
			Name:        info.Name,
			Description: info.Desc,
			Category:    info.Cat,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		if err := database.Create(&perm).Error; err != nil {
			// Skip if already exists
			continue
		}
	}

	// Seed roles
	roles := []struct {
		Name        string
		DisplayName string
		Description string
		Color       string
		IsDefault   bool
		Perms       []string
	}{
		{
			Name:        "user",
			DisplayName: "User",
			Description: "Basic user with minimal permissions",
			Color:       "#6b7280",
			IsDefault:   true,
			Perms:       []string{auth.PermAccessContestSubmission},
		},
		{
			Name:        "helper",
			DisplayName: "Helper",
			Description: "Can help with tickets and comments",
			Color:       "#3b82f6",
			IsDefault:   false,
			Perms:       []string{auth.PermReplyTicket, auth.PermEditComment},
		},
		{
			Name:        "moderator",
			DisplayName: "Moderator",
			Description: "Can moderate content and users",
			Color:       "#8b5cf6",
			IsDefault:   false,
			Perms:       append([]string{
				auth.PermBanUser,
				auth.PermDeleteComment,
				auth.PermDeleteBlog,
				auth.PermViewTicket,
				auth.PermReplyTicket,
				auth.PermCloseTicket,
				auth.PermRejudgeSubmission,
				auth.PermViewAllSubmissions,
				auth.PermEditComment,
				auth.PermPinComment,
			}, auth.DefaultPermissionSets["helper"]...),
		},
		{
			Name:        "admin",
			DisplayName: "Admin",
			Description: "Full administrative access",
			Color:       "#ef4444",
			IsDefault:   false,
			Perms:       getDefaultAdminPermissions(),
		},
	}

	for _, roleInfo := range roles {
		role := models.Role{
			Name:        roleInfo.Name,
			DisplayName: roleInfo.DisplayName,
			Description: roleInfo.Description,
			Color:       roleInfo.Color,
			IsDefault:   roleInfo.IsDefault,
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		if err := database.Create(&role).Error; err != nil {
			// Skip if already exists
			continue
		}

		// Get permission IDs for this role
		var perms []models.Permission
		if err := database.Where("code IN ?", roleInfo.Perms).Find(&perms).Error; err != nil {
			continue
		}

		// Associate permissions with role
		if err := database.Model(&role).Association("Permissions").Append(&perms); err != nil {
			continue
		}
	}

	// Assign "admin" role to existing is_staff users
	var adminRole models.Role
	if err := database.Where("name = ?", "admin").First(&adminRole).Error; err == nil {
		var profiles []models.Profile
		if err := database.Joins("JOIN auth_user ON auth_user.id = judge_profile.user_id").
			Where("auth_user.is_staff = ?", true).
			Find(&profiles).Error; err == nil {

			for _, profile := range profiles {
				if err := database.Model(&profile).Association("Roles").Append(&adminRole); err != nil {
					continue
				}
			}
		}
	}

	// Assign "user" role to all profiles without roles
	var userRole models.Role
	if err := database.Where("name = ?", "user").First(&userRole).Error; err == nil {
		// Find profiles without any roles
		var profilesWithoutRoles []models.Profile
		if err := database.Where("id NOT IN (SELECT DISTINCT profile_id FROM judge_profile_roles)").
			Find(&profilesWithoutRoles).Error; err == nil {

			for _, profile := range profilesWithoutRoles {
				if err := database.Model(&profile).Association("Roles").Append(&userRole); err != nil {
					continue
				}
			}
		}
	}

	return nil
}

func getDefaultAdminPermissions() []string {
	return []string{
		auth.PermCreateProblem,
		auth.PermEditProblem,
		auth.PermDeleteProblem,
		auth.PermViewHiddenProblem,
		auth.PermEditProblemData,
		auth.PermCreateContest,
		auth.PermEditContest,
		auth.PermDeleteContest,
		auth.PermViewHiddenContest,
		auth.PermManageContestProblems,
		auth.PermRejudgeSubmission,
		auth.PermViewAllSubmissions,
		auth.PermBanUser,
		auth.PermEditUser,
		auth.PermDeleteUser,
		auth.PermViewUserEmail,
		auth.PermCreateOrganization,
		auth.PermEditOrganization,
		auth.PermDeleteOrganization,
		auth.PermManageOrganizationMembers,
		auth.PermEditComment,
		auth.PermDeleteComment,
		auth.PermPinComment,
		auth.PermViewTicket,
		auth.PermReplyTicket,
		auth.PermCloseTicket,
		auth.PermEditBlog,
		auth.PermDeleteBlog,
		auth.PermAccessAdminPanel,
		auth.PermUseMOSS,
		auth.PermViewStats,
		auth.PermManageJudges,
		auth.PermManageLanguages,
		auth.PermManageAnnouncements,
	}
}
