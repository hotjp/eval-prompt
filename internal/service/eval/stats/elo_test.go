package stats

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUpdateELO(t *testing.T) {
	t.Run("A wins: A rating上升, B下降", func(t *testing.T) {
		ratingA, ratingB := 1500.0, 1500.0
		outcome := 1.0 // A wins

		newA, newB := UpdateELO(ratingA, ratingB, outcome)

		assert.Greater(t, newA, ratingA, "Winner A's rating should increase")
		assert.Less(t, newB, ratingB, "Loser B's rating should decrease")
	})

	t.Run("B wins: A rating下降, B上升", func(t *testing.T) {
		ratingA, ratingB := 1500.0, 1500.0
		outcome := 0.0 // B wins

		newA, newB := UpdateELO(ratingA, ratingB, outcome)

		assert.Less(t, newA, ratingA, "Loser A's rating should decrease")
		assert.Greater(t, newB, ratingB, "Winner B's rating should increase")
	})

	t.Run("平局: 评分不变", func(t *testing.T) {
		ratingA, ratingB := 1500.0, 1500.0
		outcome := 0.5 // draw

		newA, newB := UpdateELO(ratingA, ratingB, outcome)

		// For equal ratings, expected = 0.5, outcome = 0.5, so no change
		assert.Equal(t, ratingA, newA, "Draw with equal ratings should not change A's rating")
		assert.Equal(t, ratingB, newB, "Draw with equal ratings should not change B's rating")
	})

	t.Run("不同初始评分的平局", func(t *testing.T) {
		ratingA, ratingB := 1600.0, 1400.0
		outcome := 0.5 // draw

		newA, newB := UpdateELO(ratingA, ratingB, outcome)

		// Higher rated player expected to win, so loses points in draw
		assert.Less(t, newA, ratingA, "Higher rated player should lose points in draw")
		assert.Greater(t, newB, ratingB, "Lower rated player should gain points in draw")
	})

	t.Run("rating不会低于100", func(t *testing.T) {
		ratingA, ratingB := 100.0, 100.0
		outcome := 0.0 // B wins

		newA, newB := UpdateELO(ratingA, ratingB, outcome)

		assert.GreaterOrEqual(t, newA, 100.0, "Rating should not go below 100")
		assert.GreaterOrEqual(t, newB, 100.0, "Rating should not go below 100")
	})

	t.Run("K factor determines magnitude of change", func(t *testing.T) {
		// K=32, so maximum change is 32 points
		ratingA, ratingB := 1500.0, 1500.0

		newA, _ := UpdateELO(ratingA, ratingB, 1.0)
		change := newA - ratingA

		assert.Less(t, change, 32.0, "Change should be less than K factor")
		assert.Greater(t, change, 0.0, "Winner should gain rating")
	})
}
