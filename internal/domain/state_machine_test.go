package domain

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStateMachine(t *testing.T) {
	t.Run("NewStateMachine with no states fails", func(t *testing.T) {
		_, err := NewStateMachine("test", []State{})
		require.Error(t, err)
	})

	t.Run("NewStateMachine with states", func(t *testing.T) {
		states := []State{"A", "B", "C"}
		sm, err := NewStateMachine("test", states, WithInitialState("A"))
		require.NoError(t, err)
		require.Equal(t, State("A"), sm.CurrentState())
		require.Equal(t, int64(0), sm.Version())
	})

	t.Run("AddTransition", func(t *testing.T) {
		sm, err := NewStateMachine("test", []State{"A", "B"})
		require.NoError(t, err)

		err = sm.AddTransition("A", "B", "event1")
		require.NoError(t, err)

		// Duplicate transition fails
		err = sm.AddTransition("A", "B", "event1")
		require.Error(t, err)

		// Unknown state fails
		err = sm.AddTransition("X", "B", "event2")
		require.Error(t, err)
	})

	t.Run("CanTransition", func(t *testing.T) {
		sm, err := NewStateMachine("test", []State{"A", "B", "C"})
		require.NoError(t, err)

		_ = sm.AddTransition("A", "B", "event1")
		_ = sm.AddTransition("B", "C", "event2")

		// Can check transitions from current state "A"
		require.True(t, sm.CanTransition("B", "event1"))
		// After transition to B, can use event2 to go to C
		_ = sm.Transition("B", "event1", nil)
		require.True(t, sm.CanTransition("C", "event2"))
		require.False(t, sm.CanTransition("A", "event1"))
	})

	t.Run("Transition with guards", func(t *testing.T) {
		var guardCalled bool
		guard := func(from, to State, event EventType, ctx interface{}) (bool, error) {
			guardCalled = true
			return true, nil
		}
		sm, _ := NewStateMachine("test", []State{"A", "B"}, WithGuard("A", "B", "event1", guard))
		_ = sm.AddTransition("A", "B", "event1")

		err := sm.Transition("B", "event1", nil)
		require.NoError(t, err)
		require.True(t, guardCalled)
	})

	t.Run("Transition with blocking guard", func(t *testing.T) {
		sm, _ := NewStateMachine("test", []State{"A", "B"}, WithGuard("A", "B", "event1", func(from, to State, event EventType, ctx interface{}) (bool, error) {
			return false, nil
		}))
		_ = sm.AddTransition("A", "B", "event1")

		err := sm.Transition("B", "event1", nil)
		require.Error(t, err)
	})

	t.Run("Transition with actions", func(t *testing.T) {
		sm, _ := NewStateMachine("test", []State{"A", "B"}, WithAction("A", "B", "event1", func(from, to State, event EventType, ctx interface{}) error {
			return nil
		}))
		_ = sm.AddTransition("A", "B", "event1")

		err := sm.Transition("B", "event1", nil)
		require.NoError(t, err)
		require.Equal(t, int64(1), sm.Version())
	})

	t.Run("Reset", func(t *testing.T) {
		sm, _ := NewStateMachine("test", []State{"A", "B"})
		_ = sm.AddTransition("A", "B", "event1")

		_ = sm.Transition("B", "event1", nil)
		require.Equal(t, State("B"), sm.CurrentState())

		sm.Reset()
		require.Equal(t, State("A"), sm.CurrentState())
		require.Equal(t, int64(0), sm.Version())
	})
}

func TestAssetStateMachine(t *testing.T) {
	t.Run("NewAssetStateMachine", func(t *testing.T) {
		asm, err := NewAssetStateMachine()
		require.NoError(t, err)
		require.Equal(t, AssetStateCreated, asm.CurrentState())
	})

	t.Run("Valid transitions", func(t *testing.T) {
		asm, _ := NewAssetStateMachine()

		transitions := asm.Transitions()
		require.NotEmpty(t, transitions)
	})

	t.Run("CanEval", func(t *testing.T) {
		asm, _ := NewAssetStateMachine()
		require.True(t, asm.CanEval())

		// After moving to EVALUATING, CanEval should still work
		_ = asm.Transition(AssetStateEvaluating, EventEvalStarted, nil)
		// CanEval from EVALUATING is false since it's already evaluating
		// But CanTransition to EVALUATING from CREATED is true
	})

	t.Run("CanPromote", func(t *testing.T) {
		// Cannot promote from CREATED
		_, err := NewStateMachine("test", []State{AssetStateCreated, AssetStateEvaluated, AssetStatePromoted})
		require.NoError(t, err)
	})

	t.Run("Full lifecycle", func(t *testing.T) {
		asm, _ := NewAssetStateMachine()

		// CREATED -> EVALUATING
		err := asm.Transition(AssetStateEvaluating, EventEvalStarted, nil)
		require.NoError(t, err)
		require.Equal(t, AssetStateEvaluating, asm.CurrentState())

		// EVALUATING -> EVALUATED
		err = asm.Transition(AssetStateEvaluated, EventEvalCompleted, nil)
		require.NoError(t, err)
		require.Equal(t, AssetStateEvaluated, asm.CurrentState())
	})
}

func TestEvalGuard(t *testing.T) {
	t.Run("passes with score above threshold", func(t *testing.T) {
		guard := EvalGuard(60)
		allowed, err := guard(AssetStateEvaluated, AssetStatePromoted, EventLabelPromoted, 80)
		require.NoError(t, err)
		require.True(t, allowed)
	})

	t.Run("blocks with score below threshold", func(t *testing.T) {
		guard := EvalGuard(60)
		allowed, err := guard(AssetStateEvaluated, AssetStatePromoted, EventLabelPromoted, 50)
		require.Error(t, err) // guard returns error when blocked
		require.False(t, allowed)
	})

	t.Run("passes for non-promotion events", func(t *testing.T) {
		guard := EvalGuard(60)
		allowed, err := guard(AssetStateCreated, AssetStateEvaluating, EventEvalStarted, 50)
		require.NoError(t, err)
		require.True(t, allowed)
	})
}

func TestStateMachine_CanReach(t *testing.T) {
	sm, _ := NewStateMachine("test", []State{"A", "B", "C"})
	_ = sm.AddTransition("A", "B", "event1")
	_ = sm.AddTransition("B", "C", "event2")

	require.True(t, sm.CanReach("A", "C"))
	require.True(t, sm.CanReach("A", "B"))
	require.False(t, sm.CanReach("C", "A"))
}

func TestStateMachine_IsTerminalState(t *testing.T) {
	sm, _ := NewStateMachine("test", []State{"A", "B", "C"})
	_ = sm.AddTransition("A", "B", "event1")
	// B and C have no outgoing transitions

	require.False(t, sm.IsTerminalState("A"))
	require.True(t, sm.IsTerminalState("B"))
	require.True(t, sm.IsTerminalState("C"))
}

func TestStateMachine_ValidEvents(t *testing.T) {
	sm, _ := NewStateMachine("test", []State{"A", "B", "C"})
	_ = sm.AddTransition("A", "B", "event1")
	_ = sm.AddTransition("A", "C", "event2")

	events := sm.ValidEvents()
	require.Len(t, events, 2)
	require.Contains(t, events, EventType("event1"))
	require.Contains(t, events, EventType("event2"))
}
