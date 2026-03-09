package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// JSONField is a helper type for columns containing JSON (mirrors Django's JSONField).
type JSONField map[string]interface{}

func (j JSONField) Value() (driver.Value, error) {
	b, err := json.Marshal(j)
	return string(b), err
}

func (j *JSONField) Scan(value interface{}) error {
	var bytes []byte
	switch v := value.(type) {
	case string:
		bytes = []byte(v)
	case []byte:
		bytes = v
	default:
		return fmt.Errorf("JSONField: unsupported type %T", value)
	}
	return json.Unmarshal(bytes, j)
}

// ContestTag mirrors judge_contesttag.
type ContestTag struct {
	ID          uint   `gorm:"primaryKey;column:id"`
	Name        string `gorm:"column:name;size:20;not null;uniqueIndex"`
	Color       string `gorm:"column:color;size:7;not null"`
	Description string `gorm:"column:description;type:longtext;not null"`
}

func (ContestTag) TableName() string { return "judge_contesttag" }

// Contest mirrors judge_contest.
type Contest struct {
	ID                     uint       `gorm:"primaryKey;column:id"`
	Key                    string     `gorm:"column:key;size:32;not null;uniqueIndex"`
	Name                   string     `gorm:"column:name;size:100;not null;index"`
	Description            string     `gorm:"column:description;type:longtext;not null"`
	StartTime              time.Time  `gorm:"column:start_time;not null;index"`
	EndTime                time.Time  `gorm:"column:end_time;not null;index"`
	TimeLimit              *int64     `gorm:"column:time_limit"` // microseconds (Django DurationField)
	Summary                string     `gorm:"column:summary;type:longtext;not null"`
	IsVisible              bool       `gorm:"column:is_visible;not null;default:0"`
	IsRated                bool       `gorm:"column:is_rated;not null;default:0"`
	ScoreboardVisibility   string     `gorm:"column:scoreboard_visibility;size:1;not null;default:'V'"`
	ScoreboardCacheTimeout uint       `gorm:"column:scoreboard_cache_timeout;not null;default:0"`
	UseClarifications      bool       `gorm:"column:use_clarifications;not null;default:1"`
	PushAnnouncements      bool       `gorm:"column:push_announcements;not null;default:0"`
	RatingFloor            *int       `gorm:"column:rating_floor"`
	RatingCeiling          *int       `gorm:"column:rating_ceiling"`
	RateAll                bool       `gorm:"column:rate_all;not null;default:0"`
	IsPrivate              bool       `gorm:"column:is_private;not null;default:0"`
	HideProblemTags        bool       `gorm:"column:hide_problem_tags;not null;default:0"`
	HideProblemAuthors     bool       `gorm:"column:hide_problem_authors;not null;default:0"`
	RunPretestsOnly        bool       `gorm:"column:run_pretests_only;not null;default:0"`
	ShowShortDisplay       bool       `gorm:"column:show_short_display;not null;default:0"`
	IsOrganizationPrivate  bool       `gorm:"column:is_organization_private;not null;default:0"`
	OgImage                string     `gorm:"column:og_image;size:150;not null;default:''"`
	LogoOverrideImage      string     `gorm:"column:logo_override_image;size:150;not null;default:''"`
	UserCount              int        `gorm:"column:user_count;not null;default:0"`
	VirtualCount           int        `gorm:"column:virtual_count;not null;default:0"`
	AccessCode             string     `gorm:"column:access_code;size:255;not null;default:''"`
	FormatName             string     `gorm:"column:format_name;size:32;not null;default:'default'"`
	FormatConfig           JSONField  `gorm:"column:format_config;type:longtext"`
	ProblemLabelScript     string     `gorm:"column:problem_label_script;type:longtext;not null"`
	LockedAfter            *time.Time `gorm:"column:locked_after"`
	PointsPrecision        int        `gorm:"column:points_precision;not null;default:3"`
	MaxSubmissions         *int       `gorm:"column:max_submissions"`

	// relations
	Authors            []Profile        `gorm:"many2many:judge_contest_authors;joinForeignKey:contest_id;joinReferences:profile_id"`
	Curators           []Profile        `gorm:"many2many:judge_contest_curators;joinForeignKey:contest_id;joinReferences:profile_id"`
	Testers            []Profile        `gorm:"many2many:judge_contest_testers;joinForeignKey:contest_id;joinReferences:profile_id"`
	Problems           []Problem        `gorm:"many2many:judge_contestproblem;joinForeignKey:contest_id;joinReferences:problem_id"`
	ContestProblems    []ContestProblem `gorm:"foreignKey:ContestID"`
	Tags               []ContestTag     `gorm:"many2many:judge_contest_tags;joinForeignKey:contest_id;joinReferences:contesttag_id"`
	Organizations      []Organization   `gorm:"many2many:judge_contest_organizations;joinForeignKey:contest_id;joinReferences:organization_id"`
	BannedUsers        []Profile        `gorm:"many2many:judge_contest_banned_users;joinForeignKey:contest_id;joinReferences:profile_id"`
	PrivateContestants []Profile        `gorm:"many2many:judge_contest_private_contestants;joinForeignKey:contest_id;joinReferences:profile_id"`
}

func (Contest) TableName() string { return "judge_contest" }

// ContestAnnouncement mirrors judge_contestannouncement.
type ContestAnnouncement struct {
	ID          uint      `gorm:"primaryKey;column:id"`
	ContestID   uint      `gorm:"column:contest_id;not null;index"`
	Title       string    `gorm:"column:title;size:100;not null"`
	Description string    `gorm:"column:description;type:longtext;not null"`
	Date        time.Time `gorm:"column:date;not null"`
	Contest     Contest   `gorm:"foreignKey:ContestID"`
}

func (ContestAnnouncement) TableName() string { return "judge_contestannouncement" }

// ContestClarification mirrors judge_contestclarification
type ContestClarification struct {
	ID            uint      `gorm:"primaryKey;column:id"`
	ContestID     uint      `gorm:"column:contest_id;not null;index"`
	Question      string    `gorm:"column:question;type:longtext;not null"`
	Answer        *string   `gorm:"column:answer;type:longtext"`
	CreateTime    time.Time `gorm:"column:time;not null"`
	IsAnswered    bool      `gorm:"column:is_answered;not null;default:0"`
	IsInlined     bool      `gorm:"column:is_inlined;not null;default:0"`
	AuthorID      uint      `gorm:"column:author_id;not null;index"`
	Contest       Contest   `gorm:"foreignKey:ContestID"`
	Author        Profile   `gorm:"foreignKey:AuthorID"`
}

func (ContestClarification) TableName() string { return "judge_contestclarification" }

// ContestParticipation mirrors judge_contestparticipation.
type ContestParticipation struct {
	ID             uint      `gorm:"primaryKey;column:id"`
	ContestID      uint      `gorm:"column:contest_id;not null;index"`
	UserID         uint      `gorm:"column:user_id;not null;index"`
	RealStart      time.Time `gorm:"column:start;not null"` // db_column='start'
	Score          float64   `gorm:"column:score;not null;index;default:0"`
	Cumtime        uint      `gorm:"column:cumtime;not null;default:0"`
	IsDisqualified bool      `gorm:"column:is_disqualified;not null;default:0"`
	Tiebreaker     float64   `gorm:"column:tiebreaker;not null;default:0"`
	Virtual        int       `gorm:"column:virtual;not null;default:0"` // 0=LIVE, -1=SPECTATE
	FormatData     JSONField `gorm:"column:format_data;type:longtext"`
	Contest        Contest   `gorm:"foreignKey:ContestID"`
	User           Profile   `gorm:"foreignKey:UserID"`
}

func (ContestParticipation) TableName() string { return "judge_contestparticipation" }

// ContestProblem mirrors judge_contestproblem.
type ContestProblem struct {
	ID                   uint    `gorm:"primaryKey;column:id"`
	ProblemID            uint    `gorm:"column:problem_id;not null;index"`
	ContestID            uint    `gorm:"column:contest_id;not null;index"`
	Points               int     `gorm:"column:points;not null"`
	Partial              bool    `gorm:"column:partial;not null;default:1"`
	IsPretested          bool    `gorm:"column:is_pretested;not null;default:0"`
	Order                uint    `gorm:"column:order;not null;index"`
	OutputPrefixOverride *int    `gorm:"column:output_prefix_override;default:0"`
	MaxSubmissions       *int    `gorm:"column:max_submissions"`
	Problem              Problem `gorm:"foreignKey:ProblemID"`
	Contest              Contest `gorm:"foreignKey:ContestID"`
}

func (ContestProblem) TableName() string { return "judge_contestproblem" }

// ContestSubmission mirrors judge_contestsubmission.
type ContestSubmission struct {
	ID              uint                 `gorm:"primaryKey;column:id"`
	SubmissionID    uint                 `gorm:"column:submission_id;not null;uniqueIndex"`
	ProblemID       uint                 `gorm:"column:problem_id;not null;index"`
	ParticipationID uint                 `gorm:"column:participation_id;not null;index"`
	Points          float64              `gorm:"column:points;not null;default:0"`
	IsPretest       bool                 `gorm:"column:is_pretest;not null;default:0"`
	Submission      Submission           `gorm:"foreignKey:SubmissionID"`
	Problem         ContestProblem       `gorm:"foreignKey:ProblemID"`
	Participation   ContestParticipation `gorm:"foreignKey:ParticipationID"`
}

func (ContestSubmission) TableName() string { return "judge_contestsubmission" }

// Rating mirrors judge_rating.
type Rating struct {
	ID              uint                 `gorm:"primaryKey;column:id"`
	UserID          uint                 `gorm:"column:user_id;not null;index"`
	ContestID       uint                 `gorm:"column:contest_id;not null;index"`
	ParticipationID uint                 `gorm:"column:participation_id;not null;uniqueIndex"`
	Rank            int                  `gorm:"column:rank;not null"`
	RatingVal       int                  `gorm:"column:rating;not null"`
	Mean            float64              `gorm:"column:mean;not null"`
	Performance     float64              `gorm:"column:performance;not null"`
	LastRated       time.Time            `gorm:"column:last_rated;not null;index"`
	User            Profile              `gorm:"foreignKey:UserID"`
	Contest         Contest              `gorm:"foreignKey:ContestID"`
	Participation   ContestParticipation `gorm:"foreignKey:ParticipationID"`
}

func (Rating) TableName() string { return "judge_rating" }
