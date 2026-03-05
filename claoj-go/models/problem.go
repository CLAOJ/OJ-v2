package models

import "time"

const (
	SubmissionSourceAccessAlways  = "A"
	SubmissionSourceAccessSolved  = "S"
	SubmissionSourceAccessOnlyOwn = "O"
	SubmissionSourceAccessFollow  = "F"

	ProblemTestcaseAccessAlways = "A"
)

// ProblemGroup mirrors judge_problemgroup.
type ProblemGroup struct {
	ID       uint   `gorm:"primaryKey;column:id"`
	Name     string `gorm:"column:name;size:20;not null;uniqueIndex"`
	FullName string `gorm:"column:full_name;size:100;not null"`
}

func (ProblemGroup) TableName() string { return "judge_problemgroup" }

// ProblemType mirrors judge_problemtype.
type ProblemType struct {
	ID       uint   `gorm:"primaryKey;column:id"`
	Name     string `gorm:"column:name;size:20;not null;uniqueIndex"`
	FullName string `gorm:"column:full_name;size:100;not null"`
}

func (ProblemType) TableName() string { return "judge_problemtype" }

// License mirrors judge_license.
type License struct {
	ID      uint   `gorm:"primaryKey;column:id"`
	Key     string `gorm:"column:key;size:20;not null;uniqueIndex"`
	Link    string `gorm:"column:link;size:256;not null"`
	Name    string `gorm:"column:name;size:256;not null"`
	Display string `gorm:"column:display;size:256;not null;default:''"`
	Icon    string `gorm:"column:icon;size:256;not null;default:''"`
	Text    string `gorm:"column:text;type:longtext;not null"`
}

func (License) TableName() string { return "judge_license" }

// Problem mirrors judge_problem.
type Problem struct {
	ID                             uint       `gorm:"primaryKey;column:id"`
	Code                           string     `gorm:"column:code;size:32;not null;uniqueIndex"`
	Name                           string     `gorm:"column:name;size:100;not null;index"`
	Source                         string     `gorm:"column:source;size:200;not null;index;default:''"`
	Description                    string     `gorm:"column:description;type:longtext;not null"`
	PdfURL                         string     `gorm:"column:pdf_url;size:200;not null;default:''"`
	GroupID                        uint       `gorm:"column:group_id;not null"`
	TimeLimit                      float64    `gorm:"column:time_limit;not null"`
	MemoryLimit                    uint       `gorm:"column:memory_limit;not null"`
	ShortCircuit                   bool       `gorm:"column:short_circuit;not null;default:0"`
	Points                         float64    `gorm:"column:points;not null"`
	Partial                        bool       `gorm:"column:partial;not null;default:0"`
	IsPublic                       bool       `gorm:"column:is_public;not null;index;default:0"`
	IsManuallyManaged              bool       `gorm:"column:is_manually_managed;not null;index;default:0"`
	Date                           *time.Time `gorm:"column:date;index"`
	LicenseID                      *uint      `gorm:"column:license_id"`
	OgImage                        string     `gorm:"column:og_image;size:150;not null;default:''"`
	Summary                        string     `gorm:"column:summary;type:longtext;not null"`
	UserCount                      int        `gorm:"column:user_count;not null;default:0"`
	AcRate                         float64    `gorm:"column:ac_rate;not null;default:0"`
	IsFullMarkup                   bool       `gorm:"column:is_full_markup;not null;default:0"`
	SubmissionSourceVisibilityMode string     `gorm:"column:submission_source_visibility_mode;size:1;not null;default:'F'"`
	TestcaseVisibilityMode         string     `gorm:"column:testcase_visibility_mode;size:1;not null;default:'C'"`
	IsOrganizationPrivate          bool       `gorm:"column:is_organization_private;not null;default:0"`
	SuggesterID                    *uint      `gorm:"column:suggester_id"`

	// relations
	Group         ProblemGroup   `gorm:"foreignKey:GroupID"`
	License       *License       `gorm:"foreignKey:LicenseID"`
	Authors       []Profile      `gorm:"many2many:judge_problem_authors;joinForeignKey:problem_id;joinReferences:profile_id"`
	Curators      []Profile      `gorm:"many2many:judge_problem_curators;joinForeignKey:problem_id;joinReferences:profile_id"`
	Testers       []Profile      `gorm:"many2many:judge_problem_testers;joinForeignKey:problem_id;joinReferences:profile_id"`
	Types         []ProblemType  `gorm:"many2many:judge_problem_types;joinForeignKey:problem_id;joinReferences:problemtype_id"`
	AllowedLangs  []Language     `gorm:"many2many:judge_problem_allowed_languages;joinForeignKey:problem_id;joinReferences:language_id"`
	Organizations []Organization `gorm:"many2many:judge_problem_organizations;joinForeignKey:problem_id;joinReferences:organization_id"`
	BannedUsers   []Profile      `gorm:"many2many:judge_problem_banned_users;joinForeignKey:problem_id;joinReferences:profile_id"`
	Judges        []Judge        `gorm:"many2many:judge_judge_problems;joinForeignKey:problem_id;joinReferences:judge_id"`
}

func (Problem) TableName() string { return "judge_problem" }

// ProblemTranslation mirrors judge_problemtranslation.
type ProblemTranslation struct {
	ID          uint    `gorm:"primaryKey;column:id"`
	ProblemID   uint    `gorm:"column:problem_id;not null;index"`
	Language    string  `gorm:"column:language;size:7;not null"`
	Name        string  `gorm:"column:name;size:100;not null;index"`
	Description string  `gorm:"column:description;type:longtext;not null"`
	Problem     Problem `gorm:"foreignKey:ProblemID"`
}

func (ProblemTranslation) TableName() string { return "judge_problemtranslation" }

// ProblemClarification mirrors judge_problemclarification.
type ProblemClarification struct {
	ID          uint      `gorm:"primaryKey;column:id"`
	ProblemID   uint      `gorm:"column:problem_id;not null;index"`
	Description string    `gorm:"column:description;type:longtext;not null"`
	Date        time.Time `gorm:"column:date;not null"`
	Problem     Problem   `gorm:"foreignKey:ProblemID"`
}

func (ProblemClarification) TableName() string { return "judge_problemclarification" }

// LanguageLimit mirrors judge_languagelimit.
type LanguageLimit struct {
	ID          uint     `gorm:"primaryKey;column:id"`
	ProblemID   uint     `gorm:"column:problem_id;not null;index"`
	LanguageID  uint     `gorm:"column:language_id;not null;index"`
	TimeLimit   float64  `gorm:"column:time_limit;not null"`
	MemoryLimit int      `gorm:"column:memory_limit;not null"`
	Problem     Problem  `gorm:"foreignKey:ProblemID"`
	Language    Language `gorm:"foreignKey:LanguageID"`
}

func (LanguageLimit) TableName() string { return "judge_languagelimit" }

// Solution mirrors judge_solution.
type Solution struct {
	ID         uint       `gorm:"primaryKey;column:id"`
	ProblemID  uint       `gorm:"column:problem_id;not null;uniqueIndex"`
	Problem    Problem    `gorm:"foreignKey:ProblemID"`
	PdfURL     string     `gorm:"column:pdf_url;size:200;not null;default:''"`
	Content    string     `gorm:"column:content;type:longtext;not null"`
	Authors    []Profile  `gorm:"many2many:judge_solution_authors;joinForeignKey:solution_id;joinReferences:profile_id"`
	IsPublic   bool       `gorm:"column:is_public;not null;default:0;index"`
	IsOfficial bool       `gorm:"column:is_official;not null;default:0"`
	PublishOn  *time.Time `gorm:"column:publish_on;index"`
	ValidUntil *time.Time `gorm:"column:valid_until"`
	Summary    string     `gorm:"column:summary;type:longtext"`
	Language   string     `gorm:"column:language;size:7;not null;default:'en'"`
}

func (Solution) TableName() string { return "judge_solution" }
