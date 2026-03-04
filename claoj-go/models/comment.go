package models

import "time"

// Comment mirrors judge_comment (using django-mptt tree structure).
type Comment struct {
	ID       uint      `gorm:"primaryKey;column:id"`
	AuthorID uint      `gorm:"column:author_id;not null;index"`
	Time     time.Time `gorm:"column:time;not null"`
	Page     string    `gorm:"column:page;size:30;not null;index"`
	Score    int       `gorm:"column:score;not null;index;default:0"`
	Body     string    `gorm:"column:body;type:longtext;not null"`
	Hidden   bool      `gorm:"column:hidden;not null;default:0"`
	ParentID *uint     `gorm:"column:parent_id;index"`

	// MPTT tree fields (managed by django-mptt)
	Lft    int `gorm:"column:lft;not null;index"`
	Rght   int `gorm:"column:rght;not null;index"`
	TreeID int `gorm:"column:tree_id;not null;index"`
	Level  int `gorm:"column:level;not null;index"`

	Author Profile  `gorm:"foreignKey:AuthorID"`
	Parent *Comment `gorm:"foreignKey:ParentID"`
}

func (Comment) TableName() string { return "judge_comment" }

// CommentVote mirrors judge_commentvote.
type CommentVote struct {
	ID        uint    `gorm:"primaryKey;column:id"`
	CommentID uint    `gorm:"column:comment_id;not null;index"`
	VoterID   uint    `gorm:"column:voter_id;not null;index"`
	Score     int     `gorm:"column:score;not null"`
	Comment   Comment `gorm:"foreignKey:CommentID"`
	Voter     Profile `gorm:"foreignKey:VoterID"`
}

func (CommentVote) TableName() string { return "judge_commentvote" }

// CommentLock mirrors judge_commentlock.
type CommentLock struct {
	ID   uint   `gorm:"primaryKey;column:id"`
	Page string `gorm:"column:page;size:30;not null;uniqueIndex"`
}

func (CommentLock) TableName() string { return "judge_commentlock" }
