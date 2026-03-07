// Package problemgroup provides problem group management services.
package problemgroup

// ProblemGroup represents a problem group.
type ProblemGroup struct {
	ID       uint
	Name     string
	FullName string
}

// ListGroupsResponse holds the response for listing groups.
type ListGroupsResponse struct {
	Groups []ProblemGroup
	Total  int64
}

// GetGroupRequest holds parameters for getting a group.
type GetGroupRequest struct {
	GroupID uint
}

// CreateGroupRequest holds parameters for creating a group.
type CreateGroupRequest struct {
	Name     string
	FullName string
}

// UpdateGroupRequest holds parameters for updating a group.
type UpdateGroupRequest struct {
	GroupID  uint
	Name     *string
	FullName *string
}

// DeleteGroupRequest holds parameters for deleting a group.
type DeleteGroupRequest struct {
	GroupID uint
}
