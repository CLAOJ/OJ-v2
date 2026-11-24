// Package user provides user management services.
package user

import (
	"time"
)

// UserProfile represents a user profile with associated user data.
type UserProfile struct {
	ID                 uint
	UserID             uint
	Username           string
	FirstName          string
	LastName           string
	Email              string
	DisplayName        string
	About              string
	Notes              string
	UserScript         string
	Timezone           string
	LanguageID         uint
	AceTheme           string
	SiteTheme          string
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
	LastLogin          *time.Time
	LastAccess         time.Time
	DisplayRank        string
	BanReason          *string
	OrganizationIDs    []uint
	Groups             []GroupRef
}

// GroupRef is the id/name pair the admin API shows for a Django group.
type GroupRef struct {
	ID   uint
	Name string
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
	Username                *string
	FirstName               *string
	LastName                *string
	Email                   *string
	DisplayName             *string
	About                   *string
	Notes                   *string
	UserScript              *string
	Timezone                *string
	LanguageID              *uint
	AceTheme                *string
	SiteTheme               *string
	IsActive                *bool
	IsStaff                 *bool
	IsSuperuser             *bool
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
	// Search, when non-empty, filters results to users whose username or
	// email contains the given term (case-insensitive substring match).
	Search string
	// Staff/Superuser/Active filter on the matching auth_user flag when set.
	Staff     *bool
	Superuser *bool
	Active    *bool
	// GroupID, when non-zero, restricts results to members of that group.
	GroupID uint
	// Sort is one of username|date_joined|last_login|points; anything else
	// falls back to date_joined. Order is asc|desc (default desc).
	Sort  string
	Order string
}

// ListUsersResponse holds the response for listing users.
type ListUsersResponse struct {
	Users    []UserProfile
	Total    int64
	Page     int
	PageSize int
}
