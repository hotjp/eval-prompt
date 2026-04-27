package stats

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBootstrapCI(t *testing.T) {
	t.Run("empty array returns 0,0", func(t *testing.T) {
		low, high := BootstrapCI([]float64{}, 0.95, 1000)
		assert.Equal(t, 0.0, low)
		assert.Equal(t, 0.0, high)
	})

	t.Run("single element returns same value", func(t *testing.T) {
		val := 5.0
		low, high := BootstrapCI([]float64{val}, 0.95, 1000)
		assert.Equal(t, val, low)
		assert.Equal(t, val, high)
	})

	t.Run("normal data has reasonable CI range", func(t *testing.T) {
		// Use a larger sample and known random seed for reproducibility
		values := []float64{1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0, 9.0, 10.0}
		low, high := BootstrapCI(values, 0.95, 1000)

		// CI should be ordered
		assert.True(t, low <= high, "low should be <= high")

		// The mean of the data is 5.5, CI should contain it
		mean := 5.5
		assert.True(t, low <= mean, "CI lower bound should be <= mean")
		assert.True(t, high >= mean, "CI upper bound should be >= mean")

		// CI should be within reasonable range of the data
		assert.True(t, low >= 0 && low <= 10, "CI lower bound should be within data range")
		assert.True(t, high >= 0 && high <= 10, "CI upper bound should be within data range")
	})
}

func TestMean(t *testing.T) {
	t.Run("empty slice returns 0", func(t *testing.T) {
		assert.Equal(t, 0.0, mean([]float64{}))
	})

	t.Run("single element returns that element", func(t *testing.T) {
		assert.Equal(t, 5.0, mean([]float64{5.0}))
	})

	t.Run("multiple elements returns correct mean", func(t *testing.T) {
		assert.InDelta(t, 5.5, mean([]float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}), 0.001)
	})
}

func TestStdDev(t *testing.T) {
	t.Run("empty slice returns 0", func(t *testing.T) {
		assert.Equal(t, 0.0, StdDev([]float64{}))
	})

	t.Run("single element returns 0", func(t *testing.T) {
		assert.Equal(t, 0.0, StdDev([]float64{5.0}))
	})

	t.Run("known data returns correct stddev", func(t *testing.T) {
		// stddev of [1,2,3,4,5] is sqrt(2) ≈ 1.414
		values := []float64{1.0, 2.0, 3.0, 4.0, 5.0}
		result := StdDev(values)
		assert.InDelta(t, math.Sqrt(2.0), result, 0.001)
	})
}
