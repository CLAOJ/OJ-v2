// Package pdf provides PDF generation utilities for CLAOJ
package pdf

import (
	"bytes"
	"fmt"
	"time"

	"github.com/CLAOJ/claoj-go/models"
	"github.com/jung-kurt/gofpdf"
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

// GenerateContestScoreboard creates a PDF scoreboard for a contest
func GenerateContestScoreboard(data *ContestScoreboardData) ([]byte, error) {
	pdf := gofpdf.New("L", "mm", "A4", "")
	pdf.AddPage()

	// Set up fonts - use built-in fonts instead of external files
	pdf.SetFont("Arial", "", 10)

	// Title
	pdf.SetFont("Arial", "B", 16)
	pdf.Cell(0, 15, data.ContestName)
	pdf.Ln(12)

	// Contest info
	pdf.SetFont("Arial", "", 11)
	pdf.Cell(70, 8, fmt.Sprintf("Contest: %s", data.ContestKey))
	pdf.Cell(70, 8, fmt.Sprintf("Format: %s", data.FormatName))
	pdf.Ln(6)
	pdf.Cell(70, 8, fmt.Sprintf("Start: %s", data.StartTime.Format("2006-01-02 15:04:05")))
	pdf.Cell(70, 8, fmt.Sprintf("End: %s", data.EndTime.Format("2006-01-02 15:04:05")))
	pdf.Ln(10)

	// Table header
	pdf.SetFont("Arial", "B", 10)

	// Column widths (in mm)
	colRank := 12.0
	colUser := 45.0
	colScore := 20.0
	colCumtime := 20.0
	colProb := 15.0

	// Calculate total width for centering
	totalWidth := colRank + colUser + colScore + colCumtime + (float64(len(data.Problems)) * colProb)
	startX := (297.0 - totalWidth) / 2.0 // A4 landscape width is 297mm

	pdf.SetX(startX)

	// Rank header
	pdf.CellFormat(colRank, 10, "Rank", "1", 0, "C", false, 0, "")
	// User header
	pdf.CellFormat(colUser, 10, "Username", "1", 0, "C", false, 0, "")
	// Score header
	pdf.CellFormat(colScore, 10, "Score", "1", 0, "C", false, 0, "")
	// Cumtime header
	pdf.CellFormat(colCumtime, 10, "Time", "1", 0, "C", false, 0, "")

	// Problem headers
	for _, prob := range data.Problems {
		pdf.CellFormat(colProb, 10, prob.Label, "1", 0, "C", false, 0, "")
	}
	pdf.Ln(10)

	// Table rows
	pdf.SetFont("Arial", "", 9)
	for _, row := range data.Rankings {
		pdf.SetX(startX)

		// Rank
		rankStr := fmt.Sprintf("%d", row.Rank)
		pdf.CellFormat(colRank, 8, rankStr, "1", 0, "C", false, 0, "")

		// Username
		pdf.CellFormat(colUser, 8, row.Username, "1", 0, "L", false, 0, "")

		// Score
		scoreStr := fmt.Sprintf("%.1f", row.Score)
		pdf.CellFormat(colScore, 8, scoreStr, "1", 0, "R", false, 0, "")

		// Cumtime (convert to minutes)
		cumtimeMin := row.Cumtime / 60
		timeStr := fmt.Sprintf("%d", cumtimeMin)
		pdf.CellFormat(colCumtime, 8, timeStr, "1", 0, "R", false, 0, "")

		// Problem breakdown - reset text color
		pdf.SetTextColor(0, 0, 0)

		// Only render first N breakdowns based on problem count
		for i := 0; i < len(data.Problems) && i < len(row.Breakdown); i++ {
			bd := row.Breakdown[i]
			cellStr := ""
			if bd.Solved {
				cellStr = fmt.Sprintf("+%.0f", bd.Points)
				pdf.SetTextColor(0, 128, 0) // Green for solved
			} else if bd.Points > 0 {
				cellStr = fmt.Sprintf("%.0f", bd.Points)
				pdf.SetTextColor(255, 128, 0) // Orange for partial
			} else {
				pdf.SetTextColor(0, 0, 0) // Black for unsolved
			}
			pdf.CellFormat(colProb, 8, cellStr, "1", 0, "R", false, 0, "")
		}

		pdf.Ln(8)
	}

	// Footer
	pdf.SetY(-20)
	pdf.SetFont("Arial", "I", 9)
	pdf.SetTextColor(128, 128, 128)
	pdf.Cell(0, 10, fmt.Sprintf("Generated at %s", data.GeneratedAt.Format("2006-01-02 15:04:05")))

	// Get PDF as byte slice
	var buf bytes.Buffer
	err := pdf.Output(&buf)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
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
