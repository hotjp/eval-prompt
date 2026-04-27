package stats

import "math"

// UpdateELO calculates new ELO ratings after a match.
// ratingA, ratingB: current ratings
// outcome: 1.0 if A wins, 0.0 if B wins, 0.5 for draw
// Returns (newRatingA, newRatingB)
func UpdateELO(ratingA, ratingB, outcome float64) (newA, newB float64) {
	// ELO K-factor (can be made configurable)
	const K = 32.0

	// Calculate expected scores
	expectedA := 1.0 / (1.0 + math.Pow(10, (ratingB-ratingA)/400))
	expectedB := 1.0 - expectedA

	// Update ratings
	newA = ratingA + K*(outcome-expectedA)
	newB = ratingB + K*((1-outcome)-expectedB)

	// Clamp to reasonable range (optional)
	if newA < 100 {
		newA = 100
	}
	if newB < 100 {
		newB = 100
	}

	return newA, newB
}
