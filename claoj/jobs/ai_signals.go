package jobs

import (
	"sort"
	"strings"
	"time"
)

// normaliseLines returns the non-empty lines, each trimmed with inner
// whitespace collapsed to a single space, in their original order.
func normaliseLines(s string) []string {
	var out []string
	for _, line := range strings.Split(s, "\n") {
		if n := strings.Join(strings.Fields(line), " "); n != "" {
			out = append(out, n)
		}
	}
	return out
}

func lineSet(lines []string) map[string]struct{} {
	set := make(map[string]struct{}, len(lines))
	for _, l := range lines {
		set[l] = struct{}{}
	}
	return set
}

// jaccard returns |a ∩ b| / |a ∪ b|. Two empty sets are identical (1).
func jaccard(a, b map[string]struct{}) float64 {
	if len(a) == 0 && len(b) == 0 {
		return 1
	}
	inter := 0
	for l := range a {
		if _, ok := b[l]; ok {
			inter++
		}
	}
	return float64(inter) / float64(len(a)+len(b)-inter)
}

// LineSimilarity: Jaccard over the sets of normalised lines (trimmed, inner
// whitespace collapsed to one space, empty lines dropped). Two empty inputs → 1.
func LineSimilarity(a, b string) float64 {
	return jaccard(lineSet(normaliseLines(a)), lineSet(normaliseLines(b)))
}

// CountRewrites: number of consecutive pairs (attempts in time order) where both
// have ≥ rewriteMinLines normalised lines and LineSimilarity < rewriteMaxSimilarity.
func CountRewrites(attempts []string) int {
	count := 0
	var prevLines []string
	var prevSet map[string]struct{}
	for i, a := range attempts {
		lines := normaliseLines(a)
		set := lineSet(lines)
		if i > 0 && len(prevLines) >= rewriteMinLines && len(lines) >= rewriteMinLines &&
			jaccard(prevSet, set) < rewriteMaxSimilarity {
			count++
		}
		prevLines, prevSet = lines, set
	}
	return count
}

type SolvedProblem struct {
	ProblemID uint
	Points    float64   // judge_problem.points
	ACAt      time.Time // first AC in this contest
}

// AboveHistory: nil when prevMax is nil (no AC before the contest);
// otherwise points > *prevMax * historyMargin.
func AboveHistory(points float64, prevMax *float64) *bool {
	if prevMax == nil {
		return nil
	}
	above := points > *prevMax*historyMargin
	return &above
}

// SpeedInversions: problem IDs solved faster than some lower-points problem.
// Solve time of a problem = its ACAt minus the previous ACAt (or start for the first).
func SpeedInversions(start time.Time, solved []SolvedProblem) map[uint]bool {
	sorted := append([]SolvedProblem(nil), solved...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].ACAt.Before(sorted[j].ACAt) })

	solve := make([]time.Duration, len(sorted))
	prev := start
	for i, p := range sorted {
		solve[i] = p.ACAt.Sub(prev)
		prev = p.ACAt
	}

	out := make(map[uint]bool)
	for i, p := range sorted {
		for j, q := range sorted {
			if q.Points < p.Points && solve[i] < solve[j] {
				out[p.ProblemID] = true
				break
			}
		}
	}
	return out
}
