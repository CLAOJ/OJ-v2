package contest_format

import (
	"testing"
)

func TestDefaultContestFormat_GetLabelForProblem(t *testing.T) {
	cf := &DefaultContestFormat{}
	labels := []string{"1", "2", "3"}
	for i, expected := range labels {
		if got := cf.GetLabelForProblem(i); got != expected {
			t.Errorf("GetLabelForProblem(%d) = %s; want %s", i, got, expected)
		}
	}
}

func TestICPCContestFormat_GetLabelForProblem(t *testing.T) {
	cf := &ICPCContestFormat{}
	tests := []struct {
		index    int
		expected string
	}{
		{0, "A"},
		{1, "B"},
		{25, "Z"},
		{26, "AA"},
		{27, "AB"},
		{701, "ZZ"},
		{702, "AAA"},
	}
	for _, tt := range tests {
		if got := cf.GetLabelForProblem(tt.index); got != tt.expected {
			t.Errorf("GetLabelForProblem(%d) = %s; want %s", tt.index, got, tt.expected)
		}
	}
}

func TestBaseFormat_BestSolutionState(t *testing.T) {
	bf := &BaseFormat{}
	tests := []struct {
		points   float64
		total    float64
		expected string
	}{
		{0, 100, "failed-score"},
		{50, 100, "partial-score"},
		{100, 100, "full-score"},
		{110, 100, "full-score"},
	}
	for _, tt := range tests {
		if got := bf.BestSolutionState(tt.points, tt.total); got != tt.expected {
			t.Errorf("BestSolutionState(%f, %f) = %s; want %s", tt.points, tt.total, got, tt.expected)
		}
	}
}
