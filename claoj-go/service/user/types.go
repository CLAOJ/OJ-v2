// Package user provides user management services.
package user

import (
	"time"
)

// UserProfile represents a user profile with associated user data.
type UserProfile struct {
	ID                 uint
	Username           string
	Email              string
	DisplayName        string
	About              string
	Points             float64
	PerformancePoints  float64
	ContributionPoints int
	Rating             *int
	ProblemCount       int
	IsStaff            bool
	IsSuperuser        bool
	IsActive           bool
	IsUnlisted         bool
	IsMuted            bool
	IsTotpEnabled      bool
	IsWebauthnEnabled  bool
	DateJoined         time.Time
	LastAccess         time.Time
	DisplayRank        string
	BanReason          *string
	OrganizationIDs    []uint
}

// BanUserRequest holds the parameters for banning a user.
type BanUserRequest struct {
	UserID uint
	Reason string
	Day    int
}

// UnbanUserRequest holds the parameters for unbanning a user.
type UnbanUserRequest struct {
	UserID uint
}

// UpdateUserRequest holds the parameters for updating a user.
type UpdateUserRequest struct {
	UserID                  uint
	Email                   *string
	DisplayName             *string
	About                   *string
	IsActive                *bool
	IsUnlisted              *bool
	IsMuted                 *bool
	DisplayRank             *string
	BanReason               *string
	RemoveOrganizationIDs   []uint
	AddOrganizationIDs      []uint
	RemoveOrganizationAdmin []uint
	AddOrganizationAdmin    []uint
}

// DeleteUserRequest holds the parameters for deleting a user.
type DeleteUserRequest struct {
	UserID uint
}

// GetUserRequest holds the parameters for getting a user.
type GetUserRequest struct {
	UserID uint
}

// ListUsersRequest holds the parameters for listing users.
type ListUsersRequest struct {
	Page     int
	PageSize int
}

// ListUsersResponse holds the response for listing users.
type ListUsersResponse struct {
	Users    []UserProfile
	Total    int64
	Page     int
	PageSize int
}
