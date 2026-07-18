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

// PermissionInfo describes a single Django permission (auth_permission row),
// including its full wire identity "{app_label}.{codename}".
type PermissionInfo struct {
	ID       uint
	Codename string // "{app_label}.{codename}", e.g. "judge.edit_all_problem"
	Name     string
	AppLabel string
	Model    string
}
