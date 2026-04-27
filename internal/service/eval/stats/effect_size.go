package stats

import "math"

// CohensD calculates Cohen's d effect size between two groups.
// Cohen's d = (meanA - meanB) / pooled_stddev
func CohensD(groupA, groupB []float64) float64 {
	if len(groupA) == 0 || len(groupB) == 0 {
		return 0
	}

	meanA := mean(groupA)
	meanB := mean(groupB)

	// Calculate pooled standard deviation
	nA := float64(len(groupA))
	nB := float64(len(groupB))

	var sumSqA, sumSqB float64
	for _, v := range groupA {
		diff := v - meanA
		sumSqA += diff * diff
	}
	for _, v := range groupB {
		diff := v - meanB
		sumSqB += diff * diff
	}

	pooledVar := (sumSqA + sumSqB) / (nA + nB - 2)
	pooledStd := math.Sqrt(pooledVar)

	if pooledStd == 0 {
		return 0
	}

	return (meanA - meanB) / pooledStd
}

// InterpretCohensD provides a human-readable interpretation of Cohen's d.
func InterpretCohensD(d float64) string {
	absD := math.Abs(d)
	switch {
	case absD < 0.2:
		return "negligible"
	case absD < 0.5:
		return "small"
	case absD < 0.8:
		return "medium"
	default:
		return "large"
	}
}
