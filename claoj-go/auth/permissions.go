package auth

// Permission constants for the CLAOJ permission system
// These permissions are used for fine-grained access control
const (
	// =========================================
	// PROBLEM PERMISSIONS
	// =========================================

	// PermCreateProblem - Create new problems
	PermCreateProblem = "problems.create"

	// PermEditProblem - Edit existing problems
	PermEditProblem = "problems.edit"

	// PermDeleteProblem - Delete/hide problems
	PermDeleteProblem = "problems.delete"

	// PermViewHiddenProblem - View hidden/unlisted problems
	PermViewHiddenProblem = "problems.view_hidden"

	// PermEditProblemData - Edit problem test data
	PermEditProblemData = "problems.edit_data"

	// =========================================
	// CONTEST PERMISSIONS
	// =========================================

	// PermCreateContest - Create new contests
	PermCreateContest = "contests.create"

	// PermEditContest - Edit existing contests
	PermEditContest = "contests.edit"

	// PermDeleteContest - Delete/hide contests
	PermDeleteContest = "contests.delete"

	// PermViewHiddenContest - View hidden contests
	PermViewHiddenContest = "contests.view_hidden"

	// PermManageContestProblems - Add/remove contest problems
	PermManageContestProblems = "contests.manage_problems"

	// =========================================
	// SUBMISSION PERMISSIONS
	// =========================================

	// PermRejudgeSubmission - Rejudge submissions
	PermRejudgeSubmission = "submissions.rejudge"

	// PermViewAllSubmissions - View all submissions (not just own)
	PermViewAllSubmissions = "submissions.view_all"

	// PermAccessContestSubmission - Submit in contests
	PermAccessContestSubmission = "submissions.contest_access"

	// =========================================
	// USER PERMISSIONS
	// =========================================

	// PermBanUser - Ban/unban users
	PermBanUser = "users.ban"

	// PermEditUser - Edit user profiles
	PermEditUser = "users.edit"

	// PermDeleteUser - Delete/deactivate users
	PermDeleteUser = "users.delete"

	// PermViewUserEmail - View user email addresses
	PermViewUserEmail = "users.view_email"

	// =========================================
	// ORGANIZATION PERMISSIONS
	// =========================================

	// PermCreateOrganization - Create organizations
	PermCreateOrganization = "organizations.create"

	// PermEditOrganization - Edit organizations
	PermEditOrganization = "organizations.edit"

	// PermDeleteOrganization - Delete organizations
	PermDeleteOrganization = "organizations.delete"

	// PermManageOrganizationMembers - Manage organization members
	PermManageOrganizationMembers = "organizations.manage_members"

	// =========================================
	// COMMENT PERMISSIONS
	// =========================================

	// PermEditComment - Edit any comment
	PermEditComment = "comments.edit"

	// PermDeleteComment - Delete comments
	PermDeleteComment = "comments.delete"

	// PermPinComment - Pin/unpin comments
	PermPinComment = "comments.pin"

	// =========================================
	// TICKET PERMISSIONS
	// =========================================

	// PermViewTicket - View support tickets
	PermViewTicket = "tickets.view"

	// PermReplyTicket - Reply to tickets
	PermReplyTicket = "tickets.reply"

	// PermCloseTicket - Close tickets
	PermCloseTicket = "tickets.close"

	// =========================================
	// BLOG PERMISSIONS
	// =========================================

	// PermEditBlog - Edit any blog post
	PermEditBlog = "blogs.edit"

	// PermDeleteBlog - Delete blog posts
	PermDeleteBlog = "blogs.delete"

	// =========================================
	// SYSTEM PERMISSIONS
	// =========================================

	// PermAccessAdminPanel - Access admin panel
	PermAccessAdminPanel = "system.admin_panel"

	// PermUseMoss - Use MOSS plagiarism detection
	PermUseMOSS = "system.moss"

	// PermViewStats - View site statistics
	PermViewStats = "system.stats"

	// PermManageJudges - Manage judge servers
	PermManageJudges = "system.manage_judges"

	// PermManageLanguages - Manage programming languages
	PermManageLanguages = "system.manage_languages"

	// PermManageAnnouncements - Create/edit announcements
	PermManageAnnouncements = "system.manage_announcements"

	// PermSuggestProblem - Suggest new problems
	PermSuggestProblem = "problems.suggest"

	// PermManageProblemSuggestions - Manage problem suggestions (approve/reject)
	PermManageProblemSuggestions = "problems.manage_suggestions"
)

// DefaultPermissionSets defines which permissions each default role has
var DefaultPermissionSets = map[string][]string{
	"user": {
		// Basic user permissions
		PermAccessContestSubmission,
	},
	"helper": {
		// Helper permissions (can help with tickets and comments)
		PermReplyTicket,
		PermEditComment,
	},
	"moderator": {
		// Moderator permissions (can moderate content and users)
		PermBanUser,
		PermDeleteComment,
		PermDeleteBlog,
		PermViewTicket,
		PermReplyTicket,
		PermCloseTicket,
		PermRejudgeSubmission,
		PermViewAllSubmissions,
		PermEditComment,
		PermPinComment,
	},
	"admin": {
		// All permissions
		PermCreateProblem,
		PermEditProblem,
		PermDeleteProblem,
		PermViewHiddenProblem,
		PermEditProblemData,
		PermCreateContest,
		PermEditContest,
		PermDeleteContest,
		PermViewHiddenContest,
		PermManageContestProblems,
		PermRejudgeSubmission,
		PermViewAllSubmissions,
		PermBanUser,
		PermEditUser,
		PermDeleteUser,
		PermViewUserEmail,
		PermCreateOrganization,
		PermEditOrganization,
		PermDeleteOrganization,
		PermManageOrganizationMembers,
		PermEditComment,
		PermDeleteComment,
		PermPinComment,
		PermViewTicket,
		PermReplyTicket,
		PermCloseTicket,
		PermEditBlog,
		PermDeleteBlog,
		PermAccessAdminPanel,
		PermUseMOSS,
		PermViewStats,
		PermManageJudges,
		PermManageLanguages,
		PermManageAnnouncements,
		PermManageProblemSuggestions,
	},
}
