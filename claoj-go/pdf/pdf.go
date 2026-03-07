// Package pdf provides PDF generation utilities for CLAOJ
package pdf

import (
	"fmt"
	"time"

	"github.com/CLAOJ/claoj-go/models"
	"github.com/johnfercher/maroto/v2"
	"github.com/johnfercher/maroto/v2/pkg/components/list"
	"github.com/johnfercher/maroto/v2/pkg/components/row"
	"github.com/johnfercher/maroto/v2/pkg/components/text"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/core"
	"github.com/johnfercher/maroto/v2/pkg/props"
)

// ContestScoreboardData holds all data needed to generate a contest scoreboard PDF
type ContestScoreboardData struct {
	ContestKey      string
	ContestName     string
	StartTime       time.Time
	EndTime         time.Time
	FormatName      string
	Problems        []ProblemHeader
	Rankings        []RankingRow
	GeneratedAt     time.Time
}

// ProblemHeader represents a problem column in the scoreboard
type ProblemHeader struct {
	Label  string
	Points int
}

// RankingRow represents a single row in the scoreboard
type RankingRow struct {
	Username  string
	Score     float64
	Cumtime   uint
	Rank      int
	Breakdown []BreakdownItem
}

// BreakdownItem represents problem-specific stats for a participant
type BreakdownItem struct {
	Points  float64
	Penalty uint
	Solved  bool
}

// ScoreboardRow implements list.Listable interface for scoreboard data
type ScoreboardRow struct {
	data     RankingRow
	problems []ProblemHeader
}

// GetHeader returns the header row for the scoreboard table
func (s ScoreboardRow) GetHeader() core.Row {
	headers := []string{"Rank", "Username", "Score", "Time"}
	for _, prob := range s.problems {
		headers = append(headers, prob.Label)
	}

	r := row.New(10)
	for _, h := range headers {
		r.Add(text.NewCol(1, h, props.Text{Style: fontstyle.Bold}))
	}
	return r
}

// GetContent returns the content row for the scoreboard
func (s ScoreboardRow) GetContent(idx int) core.Row {
	r := row.New(8)

	// Rank
	r.Add(text.NewCol(1, fmt.Sprintf("%d", s.data.Rank)))
	// Username
	r.Add(text.NewCol(1, s.data.Username))
	// Score
	r.Add(text.NewCol(1, fmt.Sprintf("%.1f", s.data.Score)))
	// Cumtime (in minutes)
	r.Add(text.NewCol(1, fmt.Sprintf("%d", s.data.Cumtime/60)))

	// Problem breakdown
	for i := 0; i < len(s.problems) && i < len(s.data.Breakdown); i++ {
		bd := s.data.Breakdown[i]
		var cellStr string
		if bd.Solved {
			cellStr = fmt.Sprintf("+%.0f", bd.Points)
		} else if bd.Points > 0 {
			cellStr = fmt.Sprintf("%.0f", bd.Points)
		} else {
			cellStr = "-"
		}
		r.Add(text.NewCol(1, cellStr))
	}

	return r
}

// GenerateContestScoreboard creates a PDF scoreboard for a contest
func GenerateContestScoreboard(data *ContestScoreboardData) ([]byte, error) {
	m := maroto.New()

	// Title
	m.AddRow(15,
		text.NewCol(12, data.ContestName, props.Text{Size: 16, Style: fontstyle.Bold}),
	)

	// Contest info
	m.AddRow(10,
		text.NewCol(6, fmt.Sprintf("Contest: %s", data.ContestKey)),
		text.NewCol(6, fmt.Sprintf("Format: %s", data.FormatName)),
	)
	m.AddRow(10,
		text.NewCol(6, fmt.Sprintf("Start: %s", data.StartTime.Format("2006-01-02 15:04:05"))),
		text.NewCol(6, fmt.Sprintf("End: %s", data.EndTime.Format("2006-01-02 15:04:05"))),
	)

	// Build scoreboard rows using list component
	if len(data.Rankings) > 0 {
		listableRows := make([]ScoreboardRow, len(data.Rankings))
		for i, ranking := range data.Rankings {
			listableRows[i] = ScoreboardRow{
				data:     ranking,
				problems: data.Problems,
			}
		}

		rows, err := list.Build(listableRows)
		if err == nil {
			m.AddRows(rows...)
		}
	}

	// Footer
	m.AddRow(10,
		text.NewCol(12, fmt.Sprintf("Generated at %s", data.GeneratedAt.Format("2006-01-02 15:04:05")),
			props.Text{Size: 9}),
	)

	// Generate PDF
	document, err := m.Generate()
	if err != nil {
		return nil, err
	}

	return document.GetBytes(), nil
}

// NewContestScoreboardData creates ContestScoreboardData from model data
func NewContestScoreboardData(contest *models.Contest, problems []ProblemHeader, rankings []RankingRow) *ContestScoreboardData {
	return &ContestScoreboardData{
		ContestKey:  contest.Key,
		ContestName: contest.Name,
		StartTime:   contest.StartTime,
		EndTime:     contest.EndTime,
		FormatName:  contest.FormatName,
		Problems:    problems,
		Rankings:    rankings,
		GeneratedAt: time.Now(),
	}
}
