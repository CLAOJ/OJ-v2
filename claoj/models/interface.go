package models

import "time"

// BlogPost mirrors judge_blogpost.
type BlogPost struct {
	ID             uint      `gorm:"primaryKey;column:id"`
	Title          string    `gorm:"column:title;size:100;not null"`
	Slug           string    `gorm:"column:slug;size:100;not null"`
	AuthorID       uint      `gorm:"column:author_id;not null;index"`
	PublishOn      time.Time `gorm:"column:publish_on;not null"`
	Content        string    `gorm:"column:content;type:longtext;not null"`
	Summary        string    `gorm:"column:summary;type:longtext;not null"`
	Visible        bool      `gorm:"column:visible;not null;default:0"`
	Sticky         bool      `gorm:"column:sticky;not null;default:0"`
	Score          int       `gorm:"column:score;not null;default:0"`
	GlobalPost     bool      `gorm:"column:global_post;not null;default:0"`
	OgImage        string    `gorm:"column:og_image;size:150;not null;default:''"`
	OrganizationID *uint     `gorm:"column:organization_id;index"`

	Author       Profile       `gorm:"foreignKey:AuthorID"`
	Organization *Organization `gorm:"foreignKey:OrganizationID"`
	Authors      []Profile     `gorm:"many2many:judge_blogpost_authors;joinForeignKey:blogpost_id;joinReferences:profile_id"`
}

func (BlogPost) TableName() string { return "judge_blogpost" }

// BlogVote mirrors judge_blogvote.
type BlogVote struct {
	ID      uint     `gorm:"primaryKey;column:id"`
	BlogID  uint     `gorm:"column:blog_id;not null;index:idx_blog_voter,unique"`
	VoterID uint     `gorm:"column:voter_id;not null;index:idx_blog_voter,unique"`
	Score   int      `gorm:"column:score;not null"`
	Blog    BlogPost `gorm:"foreignKey:BlogID"`
	Voter   Profile  `gorm:"foreignKey:VoterID"`
}

func (BlogVote) TableName() string { return "judge_blogvote" }

// MiscConfig mirrors judge_miscconfig (key-value config store).
type MiscConfig struct {
	ID    uint   `gorm:"primaryKey;column:id"`
	Key   string `gorm:"column:key;size:30;not null;uniqueIndex"`
	Value string `gorm:"column:value;type:longtext;not null"`
}

func (MiscConfig) TableName() string { return "judge_miscconfig" }

// NavigationBar mirrors judge_navigationbar.
type NavigationBar struct {
	ID       uint   `gorm:"primaryKey;column:id"`
	Key      string `gorm:"column:key;size:10;not null"`
	Label    string `gorm:"column:label;size:20;not null"`
	Path     string `gorm:"column:path;size:255;not null"`
	ParentID *uint  `gorm:"column:parent_id;index"`
	Order    int    `gorm:"column:order;not null;default:0"`

	// MPTT
	Lft    int `gorm:"column:lft;not null;index"`
	Rght   int `gorm:"column:rght;not null;index"`
	TreeID int `gorm:"column:tree_id;not null;index"`
	Level  int `gorm:"column:level;not null;index"`

	Parent *NavigationBar `gorm:"foreignKey:ParentID"`
}

func (NavigationBar) TableName() string { return "judge_navigationbar" }
