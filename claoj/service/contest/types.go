// Package contest provides contest management services.
package contest

import (
	"time"

	"github.com/CLAOJ/claoj/models"
)

// ContestProfile represents a contest with related data.
type ContestProfile struct {
	ID                     uint
	Key                    string
	Name                   string
	Description            string
	Summary                string
	StartTime              time.Time
	EndTime                time.Time
	TimeLimit              *int64
	IsVisible              bool
	IsRated                bool
	ScoreboardVisibility   string
	ScoreboardCacheTimeout uint
	UseClarifications      bool
	PushAnnouncements      bool
	RatingFloor            *int
	RatingCeiling          *int
	RateAll                bool
	IsPrivate              bool
	HideProblemTags        bool
	HideProblemAuthors     bool
	RunPretestsOnly        bool
	ShowShortDisplay       bool
	IsOrganizationPrivate  bool
	OgImage                string
	LogoOverrideImage      string
	UserCount              int
	VirtualCount           int
	AccessCode             string
	FormatName             string
	FormatConfig           models.JSONField
	ProblemLabelScript     string
	LockedAfter            *time.Time
	PointsPrecision        int
	MaxSubmissions         *int
	AuthorIDs              []uint
	CuratorIDs             []uint
	TesterIDs              []uint
	TagIDs                 []uint
	OrganizationIDs        []uint
}

// CreateContestRequest holds the parameters for creating a contest.
type CreateContestRequest struct {
	Key                   string
	Name                  string
	Description           string
	Summary               string
	StartTime             time.Time
	EndTime               time.Time
	TimeLimit             *int64
	IsVisible             bool
	IsRated               bool
	FormatName            string
	FormatConfig          string
	AccessCode            string
	HideProblemTags       bool
	RunPretestsOnly       bool
	IsOrganizationPrivate bool
	MaxSubmissions        *int
	AuthorIDs             []uint
	CuratorIDs            []uint
	TesterIDs             []uint
	ProblemIDs            []uint
	TagIDs                []uint
	OrganizationIDs       []uint
}

// UpdateContestRequest holds the parameters for updating a contest.
type UpdateContestRequest struct {
	ContestKey            string
	Name                  *string
	Description           *string
	Summary               *string
	StartTime             *string
	EndTime               *string
	IsVisible             *bool
	IsRated               *bool
	AccessCode            *string
	HideProblemTags       *bool
	RunPretestsOnly       *bool
	IsOrganizationPrivate *bool
	AddProblemIDs         []uint
	RemoveProblemIDs      []uint
	TimeLimit             *int64
	MaxSubmissions        *int
	AddTagIDs             []uint
	RemoveTagIDs          []uint
}

// DeleteContestRequest holds the parameters for deleting a contest.
type DeleteContestRequest struct {
	ContestKey string
}

// LockContestRequest holds the parameters for locking a contest.
type LockContestRequest struct {
	ContestKey  string
	LockedAfter *string // ISO 8601 format, or nil to unlock
}

// CloneContestRequest holds the parameters for cloning a contest.
type CloneContestRequest struct {
	SourceKey    string
	NewKey       string
	NewName      string
	CopyProblems bool
	CopySettings bool
	NewStartTime string
	NewEndTime   string
}

// GetContestRequest holds the parameters for getting a contest.
type GetContestRequest struct {
	ContestKey string
}

// ListContestsRequest holds the parameters for listing contests.
type ListContestsRequest struct {
	Page     int
	PageSize int
}

// ListContestsResponse holds the response for listing contests.
type ListContestsResponse struct {
	Contests []ContestProfile
	Total    int64
	Page     int
	PageSize int
}

// ContestDetailResponse holds the full contest detail response.
type ContestDetailResponse struct {
	Contest  ContestProfile
	Problems []ContestProblemInfo
}

// DisqualifyParticipationRequest holds the parameters for disqualifying a participation.
type DisqualifyParticipationRequest struct {
	ContestKey      string
	ParticipationID uint
}

// UndisqualifyParticipationRequest holds the parameters for undisqualifying a participation.
type UndisqualifyParticipationRequest struct {
	ContestKey      string
	ParticipationID uint
}

// AddTagRequest holds the parameters for adding a tag to a contest.
type AddTagRequest struct {
	ContestKey string
	TagID      uint
}

// RemoveTagRequest holds the parameters for removing a tag from a contest.
type RemoveTagRequest struct {
	ContestKey string
	TagID      uint
}
