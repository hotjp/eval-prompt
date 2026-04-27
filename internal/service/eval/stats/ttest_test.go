package stats

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPairedTTest(t *testing.T) {
	t.Run("same arrays returns t≈0, p≈1", func(t *testing.T) {
		before := []float64{1.0, 2.0, 3.0, 4.0, 5.0}
		after := []float64{1.0, 2.0, 3.0, 4.0, 5.0}

		tStat, pValue := PairedTTest(before, after)

		assert.InDelta(t, 0.0, tStat, 0.001, "t-statistic should be ~0 for identical arrays")
		assert.InDelta(t, 1.0, pValue, 0.001, "p-value should be ~1 for identical arrays")
	})

	t.Run("empty arrays returns 0,1", func(t *testing.T) {
		tStat, pValue := PairedTTest([]float64{}, []float64{})
		assert.Equal(t, 0.0, tStat)
		assert.Equal(t, 1.0, pValue)
	})

	t.Run("mismatched length returns 0,1", func(t *testing.T) {
		tStat, pValue := PairedTTest([]float64{1, 2, 3}, []float64{1, 2})
		assert.Equal(t, 0.0, tStat)
		assert.Equal(t, 1.0, pValue)
	})

	t.Run("明显差异数组 returns t≠0", func(t *testing.T) {
		// Use arrays where differences have variance (not constant differences)
		before := []float64{1.0, 2.0, 3.0, 4.0, 5.0}
		after := []float64{2.0, 4.0, 4.0, 6.0, 7.0}
		// differences: [1.0, 2.0, 1.0, 2.0, 2.0] - has variance

		tStat, pValue := PairedTTest(before, after)

		assert.NotEqual(t, 0.0, tStat, "t-statistic should not be 0 for different arrays")
		assert.True(t, tStat > 0, "t-statistic should be positive when after > before")
		assert.True(t, pValue < 1.0, "p-value should be less than 1 for different arrays")
	})

	t.Run("large sample with difference", func(t *testing.T) {
		// Create larger samples where differences have variance
		before := make([]float64, 50)
		after := make([]float64, 50)
		for i := 0; i < 50; i++ {
			before[i] = float64(i)
			// Add some noise to make differences variable
			after[i] = float64(i) + 1.0 + float64(i%3)*0.1
		}

		tStat, pValue := PairedTTest(before, after)

		assert.NotEqual(t, 0.0, tStat)
		assert.True(t, tStat > 0)
		assert.True(t, pValue < 0.05, "p-value should be significant for consistent difference")
	})
}
