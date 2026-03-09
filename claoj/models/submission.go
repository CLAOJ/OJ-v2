package models

import "time"

// Submission mirrors judge_submission.
type Submission struct {
	ID              uint       `gorm:"primaryKey;column:id"`
	UserID          uint       `gorm:"column:user_id;not null;index"`
	ProblemID       uint       `gorm:"column:problem_id;not null;index"`
	Date            time.Time  `gorm:"column:date;not null;index"`
	Time            *float64   `gorm:"column:time;index"`
	Memory          *float64   `gorm:"column:memory"`
	Points          *float64   `gorm:"column:points;index"`
	LanguageID      uint       `gorm:"column:language_id;not null"`
	Status          string     `gorm:"column:status;size:2;not null;index;default:'QU'"`
	Result          *string    `gorm:"column:result;size:3;index"`
	Error           *string    `gorm:"column:error;type:longtext"`
	CurrentTestcase int        `gorm:"column:current_testcase;not null;default:0"`
	Batch           bool       `gorm:"column:batch;not null;default:0"`
	CasePoints      float64    `gorm:"column:case_points;not null;default:0"`
	CaseTotal       float64    `gorm:"column:case_total;not null;default:0"`
	JudgedOnID      *uint      `gorm:"column:judged_on_id"`
	JudgedDate      *time.Time `gorm:"column:judged_date"`
	RejudgedDate    *time.Time `gorm:"column:rejudged_date"`
	IsPretested     bool       `gorm:"column:is_pretested;not null;default:0"`
	ContestObjectID *uint      `gorm:"column:contest_object_id;index"`
	LockedAfter     *time.Time `gorm:"column:locked_after"`

	// eager-loadable relations
	User      Profile              `gorm:"foreignKey:UserID"`
	Problem   Problem              `gorm:"foreignKey:ProblemID"`
	Language  Language             `gorm:"foreignKey:LanguageID"`
	JudgedOn  *Judge               `gorm:"foreignKey:JudgedOnID"`
	Source    *SubmissionSource    `gorm:"foreignKey:SubmissionID"`
	TestCases []SubmissionTestCase `gorm:"foreignKey:SubmissionID"`
}

func (Submission) TableName() string { return "judge_submission" }

// SubmissionSource mirrors judge_submissionsource.
type SubmissionSource struct {
	ID           uint       `gorm:"primaryKey;column:id"`
	SubmissionID uint       `gorm:"column:submission_id;not null;uniqueIndex"`
	Source       string     `gorm:"column:source;type:longtext;not null"`
	Submission   Submission `gorm:"foreignKey:SubmissionID"`
}

func (SubmissionSource) TableName() string { return "judge_submissionsource" }

// SubmissionTestCase mirrors judge_submissiontestcase.
type SubmissionTestCase struct {
	ID               uint       `gorm:"primaryKey;column:id"`
	SubmissionID     uint       `gorm:"column:submission_id;not null;index"`
	Case             int        `gorm:"column:case;not null"`
	Status           string     `gorm:"column:status;size:3;not null"`
	Time             *float64   `gorm:"column:time"`
	Memory           *float64   `gorm:"column:memory"`
	Points           *float64   `gorm:"column:points"`
	Total            *float64   `gorm:"column:total"`
	Batch            *int       `gorm:"column:batch"`
	Feedback         string     `gorm:"column:feedback;size:50;not null;default:''"`
	ExtendedFeedback string     `gorm:"column:extended_feedback;type:longtext;not null"`
	Output           string     `gorm:"column:output;type:longtext;not null"`
	Submission       Submission `gorm:"foreignKey:SubmissionID"`
}

func (SubmissionTestCase) TableName() string { return "judge_submissiontestcase" }

// SubmissionResult holds all possible submission result codes.
var SubmissionResult = map[string]string{
	"AC":  "Accepted",
	"WA":  "Wrong Answer",
	"TLE": "Time Limit Exceeded",
	"MLE": "Memory Limit Exceeded",
	"OLE": "Output Limit Exceeded",
	"IR":  "Invalid Return",
	"RTE": "Runtime Error",
	"CE":  "Compile Error",
	"IE":  "Internal Error",
	"SC":  "Short Circuited",
	"AB":  "Aborted",
}
