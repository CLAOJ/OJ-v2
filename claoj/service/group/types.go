// Package group provides Django-group (role) management services: it reads
// and writes ROWS in Django's own auth_group / auth_group_permissions /
// auth_user_groups / auth_permission tables, exactly like Django admin does.
package group

// GroupSummary is the list-view summary of a Django group (auth_group row).
type GroupSummary struct {
	ID              uint
	Name            string
	UserCount       int
	PermissionCount int
}

// GroupUser is a minimal user reference used in GroupDetail.Users.
type GroupUser struct {
	// ID is judge_profile.id, not auth_user.id.
	ID       uint
	Username string
}

// GroupDetail is the full detail view of a single Django group.
type GroupDetail struct {
	ID            uint
	Name          string
	PermissionIDs []uint
	Users         []GroupUser
}

// GroupMember is one row of the paginated group member list.
type GroupMember struct {
	// ID is judge_profile.id -- the same id space as GroupUser.ID and the
	// /admin/user/:id route family.
	ID       uint
	Username string
	Email    string
	IsActive bool
	Points   float64
}

// ListMembersRequest selects a page of a group's members.
type ListMembersRequest struct {
	GroupID  uint
	Page     int
	PageSize int
	// Search, when non-empty, filters to members whose username or email
	// contains the term (case-insensitive substring match).
	Search string
}

// ListMembersResponse is one page of a group's members.
type ListMembersResponse struct {
	GroupID   uint
	GroupName string
	Members   []GroupMember
	Total     int64
	Page      int
	PageSize  int
}

// PermissionInfo describes a single Django permission (auth_permission row),
// including its full wire identity "{app_label}.{codename}".
type PermissionInfo struct {
	ID       uint
	Codename string // "{app_label}.{codename}", e.g. "judge.edit_all_problem"
	Name     string
	AppLabel string
	Model    string
}
