package models

// ProblemData mirrors judge_problemdata.
type ProblemData struct {
	ID            uint   `gorm:"primaryKey;column:id"`
	ProblemID     uint   `gorm:"column:problem_id;not null;uniqueIndex"`
	Zipfile       string `gorm:"column:zipfile;size:100;not null;default:''"`
	Generator     string `gorm:"column:generator;size:100;not null;default:''"`
	OutputPrefix  *int   `gorm:"column:output_prefix"`
	OutputLimit   *int   `gorm:"column:output_limit"`
	Feedback      string `gorm:"column:feedback;type:longtext;not null;default:''"`
	Checker       string `gorm:"column:checker;size:10;not null;default:'standard'"`
	Grader        string `gorm:"column:grader;size:30;not null;default:'standard'"`
	CheckerArgs   string `gorm:"column:checker_args;type:longtext;not null;default:''"`
	CustomChecker string `gorm:"column:custom_checker;size:100;not null;default:''"`
	CustomGrader  string `gorm:"column:custom_grader;size:100;not null;default:''"`
	CustomHeader  string `gorm:"column:custom_header;size:100;not null;default:''"`
	GraderArgs    string `gorm:"column:grader_args;type:longtext;not null;default:''"`

	Problem Problem `gorm:"foreignKey:ProblemID"`
}

func (ProblemData) TableName() string { return "judge_problemdata" }

// ProblemTestCase mirrors judge_problemtestcase.
type ProblemTestCase struct {
	ID            uint   `gorm:"primaryKey;column:id"`
	DatasetID     uint   `gorm:"column:dataset_id;not null;index"`
	Order         int    `gorm:"column:order;not null"`
	Type          string `gorm:"column:type;size:1;not null;default:'C'"`
	InputFile     string `gorm:"column:input_file;size:100;not null;default:''"`
	OutputFile    string `gorm:"column:output_file;size:100;not null;default:''"`
	GeneratorArgs string `gorm:"column:generator_args;type:longtext;not null;default:''"`
	Points        *int   `gorm:"column:points"`
	IsPretest     bool   `gorm:"column:is_pretest;not null;default:0"`
	OutputPrefix  *int   `gorm:"column:output_prefix"`
	OutputLimit   *int   `gorm:"column:output_limit"`
	Checker       string `gorm:"column:checker;size:10;not null;default:''"`
	CheckerArgs   string `gorm:"column:checker_args;type:longtext;not null;default:''"`

	Dataset ProblemData `gorm:"foreignKey:DatasetID"`
}

func (ProblemTestCase) TableName() string { return "judge_problemtestcase" }
