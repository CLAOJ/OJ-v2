package scoring

import (
	"testing"
)

func TestRecalculateRatings(t *testing.T) {
	// Simple test case: 2 players
	// Player 1: Rank 1 (score 100), old mean 1500, 0 times ranked
	// Player 2: Rank 2 (score 50), old mean 1500, 0 times ranked
	ranking := []float64{1.0, 2.0}
	oldMeans := []float64{1500.0, 1500.0}
	timesRanked := []int{0, 0}
	historicalPs := [][]float64{{}, {}}

	newRatings, newMeans, newPerformances := RecalculateRatings(ranking, oldMeans, timesRanked, historicalPs)

	if len(newRatings) != 2 {
		t.Fatalf("Expected 2 results, got %d", len(newRatings))
	}

	t.Logf("Player 1: Rating=%d, Mean=%.2f, Perf=%.2f\n", newRatings[0], newMeans[0], newPerformances[0])
	t.Logf("Player 2: Rating=%d, Mean=%.2f, Perf=%.2f\n", newRatings[1], newMeans[1], newPerformances[1])

	// Basic assertions
	if newRatings[0] <= newRatings[1] {
		t.Errorf("Higher rank should have higher rating: %d <= %d", newRatings[0], newRatings[1])
	}
	if newMeans[0] <= newMeans[1] {
		t.Errorf("Higher rank should have higher mean: %.2f <= %.2f", newMeans[0], newMeans[1])
	}
}

func TestRecalculateRatingsTied(t *testing.T) {
	// Tied case: 2 players
	ranking := []float64{0.5, 0.5}
	oldMeans := []float64{1500.0, 1500.0}
	timesRanked := []int{0, 0}
	historicalPs := [][]float64{{}, {}}

	newRatings, newMeans, _ := RecalculateRatings(ranking, oldMeans, timesRanked, historicalPs)

	if newRatings[0] != newRatings[1] {
		t.Errorf("Tied players should have same rating: %d != %d", newRatings[0], newRatings[1])
	}
	if newMeans[0] != newMeans[1] {
		t.Errorf("Tied players should have same mean: %.2f != %.2f", newMeans[0], newMeans[1])
	}
}
