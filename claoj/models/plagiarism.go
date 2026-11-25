package models

import "time"

// PlagiarismResult is a v2-only table (scripts/v2_runtime_tables.sql): one
// compare50 plagiarism check per contest problem x language. v1 MOSS uses
// judge_contestmoss (ContestMoss); this table is not shared with it.
type PlagiarismResult struct {
	ID              uint      `gorm:"primaryKey;column:id"`
	ContestID       uint      `gorm:"column:contest_id;not null;index"`
	ProblemID       uint      `gorm:"column:problem_id;not null"`
	Language        string    `gorm:"column:language;size:20;not null"`
	SubmissionCount uint      `gorm:"column:submission_count;not null;default:0"`
	ReportPath      *string   `gorm:"column:report_path;size:255"` // relative to the contest's report dir, e.g. knap/cpp
	CreatedAt       time.Time `gorm:"column:created_at;not null"`
}

func (PlagiarismResult) TableName() string { return "plagiarism_result" }
