package stats

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCohensD(t *testing.T) {
	t.Run("empty groupA returns 0", func(t *testing.T) {
		result := CohensD([]float64{}, []float64{1, 2, 3})
		assert.Equal(t, 0.0, result)
	})

	t.Run("empty groupB returns 0", func(t *testing.T) {
		result := CohensD([]float64{1, 2, 3}, []float64{})
		assert.Equal(t, 0.0, result)
	})

	t.Run("same means returns 0", func(t *testing.T) {
		groupA := []float64{1.0, 2.0, 3.0}
		groupB := []float64{1.0, 2.0, 3.0}
		result := CohensD(groupA, groupB)
		assert.Equal(t, 0.0, result)
	})

	t.Run("different means returns non-zero", func(t *testing.T) {
		groupA := []float64{1.0, 2.0, 3.0, 4.0, 5.0}
		groupB := []float64{5.0, 6.0, 7.0, 8.0, 9.0}
		result := CohensD(groupA, groupB)
		assert.NotEqual(t, 0.0, result)
		assert.True(t, result < 0, "groupA mean < groupB mean should give negative d")
	})

	t.Run("large difference returns large effect size", func(t *testing.T) {
		// Use groups with variance within groups to get meaningful effect size
		groupA := []float64{1.0, 2.0, 3.0, 2.0, 1.0}
		groupB := []float64{10.0, 11.0, 12.0, 11.0, 10.0}
		result := CohensD(groupA, groupB)
		// Mean difference is about 9, pooled std should be ~1.4, giving |d| > 2
		assert.True(t, math.Abs(result) > 2.0, "Large difference should give |Cohen's d| > 2, got %v", result)
	})
}

func TestInterpretCohensD(t *testing.T) {
	tests := []struct {
		d    float64
		want string
	}{
		{0.1, "negligible"},
		{0.19, "negligible"},
		{0.2, "small"},
		{0.49, "small"},
		{0.5, "medium"},
		{0.79, "medium"},
		{0.8, "large"},
		{1.5, "large"},
		{-0.3, "small"},  // absolute value used
		{-0.6, "medium"}, // absolute value used
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			assert.Equal(t, tt.want, InterpretCohensD(tt.d))
		})
	}
}
