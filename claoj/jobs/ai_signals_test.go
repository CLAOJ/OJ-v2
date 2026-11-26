package jobs

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func f64Ptr(v float64) *float64 { return &v }

func TestLineSimilarity(t *testing.T) {
	tests := []struct {
		name string
		a, b string
		want float64
	}{
		{"normalised lines match", "a\n b \n\n", "a\nb", 1},
		{"one shared of three", "x\ny", "x\nz", 1.0 / 3.0},
		{"both empty", "", "", 1},
		{"one empty", "x", "", 0},
		{"inner whitespace collapsed", "a   b\n", "a b", 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.InDelta(t, tt.want, LineSimilarity(tt.a, tt.b), 1e-9)
		})
	}
}

// sixLines builds a 6-line program whose lines are unique per tag.
func sixLines(tag string) string {
	var sb strings.Builder
	for i := 0; i < 6; i++ {
		sb.WriteString(tag + "_line_" + string(rune('a'+i)) + "\n")
	}
	return sb.String()
}

func TestCountRewrites(t *testing.T) {
	original := sixLines("orig")
	edited := strings.Replace(original, "orig_line_c", "edited", 1)
	rewritten := sixLines("new")
	short := "x\ny\n"

	tests := []struct {
		name     string
		attempts []string
		want     int
	}{
		{"single attempt", []string{original}, 0},
		{"small edit is not a rewrite", []string{original, edited}, 0},
		{"full rewrite counts", []string{original, edited, rewritten}, 1},
		{"short program in between counts for nothing", []string{original, short, rewritten}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, CountRewrites(tt.attempts))
		})
	}
}

func TestAboveHistory(t *testing.T) {
	tests := []struct {
		name    string
		points  float64
		prevMax *float64
		want    *bool
	}{
		{"no prior AC", 900, nil, nil},
		{"well above history", 900, f64Ptr(500), boolPtr(true)},
		{"within margin", 700, f64Ptr(500), boolPtr(false)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AboveHistory(tt.points, tt.prevMax)
			if tt.want == nil {
				require.Nil(t, got)
				return
			}
			require.NotNil(t, got)
			assert.Equal(t, *tt.want, *got)
		})
	}
}

func boolPtr(v bool) *bool { return &v }

func TestSpeedInversions(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	at := func(m int) time.Time { return start.Add(time.Duration(m) * time.Minute) }

	tests := []struct {
		name   string
		solved []SolvedProblem
		want   map[uint]bool
	}{
		{
			// A solved at 40 min (40 min solve time); B at 50 min is 10 min after A.
			name: "higher-point problem solved faster than lower-point one",
			solved: []SolvedProblem{
				{ProblemID: 1, Points: 100, ACAt: at(40)},
				{ProblemID: 2, Points: 1800, ACAt: at(50)},
			},
			want: map[uint]bool{2: true},
		},
		{
			// B at 2:00 is 80 min after A, slower than A's 40 min.
			name: "higher-point problem solved slower is not flagged",
			solved: []SolvedProblem{
				{ProblemID: 1, Points: 100, ACAt: at(40)},
				{ProblemID: 2, Points: 1800, ACAt: at(120)},
			},
			want: map[uint]bool{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SpeedInversions(start, tt.solved)
			require.NotNil(t, got)
			assert.Equal(t, tt.want, got)
		})
	}
}
