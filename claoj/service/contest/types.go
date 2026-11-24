// Package contest provides contest management services.
package contest

import (
	"time"

	"github.com/CLAOJ/claoj/models"
	"github.com/CLAOJ/claoj/service/relations"
)

// ContestProfile is the admin view of a contest (flat snake_case JSON).
// TimeLimit is the stored microseconds and never leaves the service;
// the API exposes TimeLimitSeconds as "time_limit".
type ContestProfile struct {
	ID                     uint                `json:"id"`
	Key                    string              `json:"key"`
	Name                   string              `json:"name"`
	Description            string              `json:"description"`
	Summary                string              `json:"summary"`
	StartTime              time.Time           `json:"start_time"`
	EndTime                time.Time           `json:"end_time"`
	TimeLimit              *int64              `json:"-"`
	TimeLimitSeconds       *int64              `json:"time_limit"`
	IsVisible              bool                `json:"is_visible"`
	IsRated                bool                `json:"is_rated"`
	ScoreboardVisibility   string              `json:"scoreboard_visibility"`
	ScoreboardCacheTimeout uint                `json:"scoreboard_cache_timeout"`
	UseClarifications      bool                `json:"use_clarifications"`
	PushAnnouncements      bool                `json:"push_announcements"`
	RatingFloor            *int                `json:"rating_floor"`
	RatingCeiling          *int                `json:"rating_ceiling"`
	PerformanceCeilingOverride *int            `json:"performance_ceiling_override"`
	RateAll                bool                `json:"rate_all"`
	IsPrivate              bool                `json:"is_private"`
	HideProblemTags        bool                `json:"hide_problem_tags"`
	HideProblemAuthors     bool                `json:"hide_problem_authors"`
	RunPretestsOnly        bool                `json:"run_pretests_only"`
	ShowShortDisplay       bool                `json:"show_short_display"`
	IsOrganizationPrivate  bool                `json:"is_organization_private"`
	LimitJoinOrganizations bool                `json:"limit_join_organizations"`
	OgImage                string              `json:"og_image"`
	LogoOverrideImage      string              `json:"logo_override_image"`
	UserCount              int                 `json:"user_count"`
	VirtualCount           int                 `json:"virtual_count"`
	AccessCode             string              `json:"access_code"`
	FormatName             string              `json:"format_name"`
	FormatConfig           models.JSONField    `json:"format_config"`
	ProblemLabelScript     string              `json:"problem_label_script"`
	LockedAfter            *time.Time          `json:"locked_after"`
	PointsPrecision        int                 `json:"points_precision"`
	AuthorIDs              []uint              `json:"author_ids"`
	CuratorIDs             []uint              `json:"curator_ids"`
	TesterIDs              []uint              `json:"tester_ids"`
	TagIDs                 []uint              `json:"tag_ids"`
	OrganizationIDs        []uint              `json:"organization_ids"`
	BannedUserIDs          []uint              `json:"banned_user_ids"`
	PrivateContestantIDs   []uint              `json:"private_contestant_ids"`
	ViewScoreboardUserIDs  []uint              `json:"view_scoreboard_user_ids"`
	JoinOrganizationIDs    []uint              `json:"join_organization_ids"`
	ClassIDs               []uint              `json:"class_ids"`
	Authors                []relations.UserRef `json:"authors"`
	Curators               []relations.UserRef `json:"curators"`
	Testers                []relations.UserRef `json:"testers"`
	BannedUsers            []relations.UserRef `json:"banned_users"`
	PrivateContestants     []relations.UserRef `json:"private_contestants"`
	ViewScoreboardUsers    []relations.UserRef `json:"view_scoreboard_users"`
	JoinOrganizations      []NamedRef          `json:"join_organizations"`
	Classes                []NamedRef          `json:"classes"`
}

// NamedRef is an id/name pair for organization and class chips in the form.
type NamedRef struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

// CreateContestRequest holds the parameters for creating a contest.
type CreateContestRequest struct {
	Key                        string
	Name                       string
	Description                string
	Summary                    string
	StartTime                  time.Time
	EndTime                    time.Time
	TimeLimit                  *int64
	IsVisible                  bool
	IsRated                    bool
	ScoreboardVisibility       string
	ScoreboardCacheTimeout     uint
	UseClarifications          *bool // nil = v1 default (true)
	PushAnnouncements          bool
	RatingFloor                *int
	RatingCeiling              *int
	PerformanceCeilingOverride *int
	RateAll                    bool
	IsPrivate                  bool
	HideProblemAuthors         bool
	ShowShortDisplay           bool
	OgImage                    string
	LogoOverrideImage          string
	ProblemLabelScript         string
	LimitJoinOrganizations     bool
	PointsPrecision            *int // nil = v1 default (3)
	FormatName                 string
	FormatConfig               string
	AccessCode                 string
	HideProblemTags            bool
	RunPretestsOnly            bool
	IsOrganizationPrivate      bool
	AuthorIDs                  []uint
	CuratorIDs                 []uint
	TesterIDs                  []uint
	ProblemIDs                 []uint
	TagIDs                     []uint
	OrganizationIDs            []uint
	BannedUserIDs              []uint
	PrivateContestantIDs       []uint
	ViewScoreboardUserIDs      []uint
	JoinOrganizationIDs        []uint
	ClassIDs                   []uint
}

// UpdateContestRequest holds the parameters for updating a contest.
// Pointer-to-slice fields are full replacement lists (nil = unchanged).
// TimeLimit is microseconds; a value <= 0 clears it.
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
	HideProblemAuthors    *bool
	RunPretestsOnly       *bool
	ShowShortDisplay      *bool
	UseClarifications     *bool
	PushAnnouncements     *bool
	IsOrganizationPrivate *bool
	LimitJoinOrganizations *bool
	ScoreboardVisibility  *string
	ScoreboardCacheTimeout *uint
	PointsPrecision       *int
	RatingFloor           *int
	RatingCeiling         *int
	PerformanceCeilingOverride *int
	ClearRatingFloor      bool
	ClearRatingCeiling    bool
	ClearPerformanceCeilingOverride bool
	RateAll               *bool
	IsPrivate             *bool
	OgImage               *string
	LogoOverrideImage     *string
	ProblemLabelScript    *string
	TimeLimit             *int64
	FormatName            *string
	FormatConfig          *string
	AuthorIDs             *[]uint
	CuratorIDs            *[]uint
	TesterIDs             *[]uint
	TagIDs                *[]uint
	OrganizationIDs       *[]uint
	BannedUserIDs         *[]uint
	PrivateContestantIDs  *[]uint
	ViewScoreboardUserIDs *[]uint
	JoinOrganizationIDs   *[]uint
	ClassIDs              *[]uint
	Problems              *[]ContestProblemInput
	AddProblemIDs         []uint
	RemoveProblemIDs      []uint
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
	Search   string    // key or name contains (literal)
	Status   string    // "", "upcoming", "ongoing", "past"
	Now      time.Time // zero = time.Now(); tests pin it
}

// ListContestsResponse holds the response for listing contests.
type ListContestsResponse struct {
	Contests []ContestProfile
	Total    int64
	Page     int
	PageSize int
}

// ContestDetailResponse is the flat contest profile plus its problems.
type ContestDetailResponse struct {
	ContestProfile
	Problems []ContestProblemInfo `json:"problems"`
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
