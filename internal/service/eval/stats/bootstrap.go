// Package stats provides statistical functions for evaluation.
package stats

import (
	"math"
	"math/rand"
	"sort"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// BootstrapCI computes the bootstrap confidence interval for a set of values.
// Uses percentile method with n bootstrap iterations.
func BootstrapCI(values []float64, confidence float64, n int) (low, high float64) {
	if len(values) == 0 {
		return 0, 0
	}
	if len(values) == 1 {
		return values[0], values[0]
	}

	// Generate bootstrap sample means
	sampleMeans := make([]float64, n)
	for i := 0; i < n; i++ {
		// Resample with replacement
		sum := 0.0
		for j := 0; j < len(values); j++ {
			idx := rand.Intn(len(values))
			sum += values[idx]
		}
		sampleMeans[i] = sum / float64(len(values))
	}

	// Sort sample means
	sort.Float64s(sampleMeans)

	// Calculate percentile bounds
	alpha := 1 - confidence
	lowerIdx := int(alpha / 2 * float64(n))
	upperIdx := int((1-alpha/2) * float64(n))

	// Clamp indices
	if lowerIdx < 0 {
		lowerIdx = 0
	}
	if upperIdx >= n {
		upperIdx = n - 1
	}

	return sampleMeans[lowerIdx], sampleMeans[upperIdx]
}

func mean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

// StdDev calculates the standard deviation.
func StdDev(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	m := mean(values)
	sumSquares := 0.0
	for _, v := range values {
		diff := v - m
		sumSquares += diff * diff
	}
	return math.Sqrt(sumSquares / float64(len(values)))
}
