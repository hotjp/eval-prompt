package domain

import (
	"fmt"
	"slices"
	"time"
)

// State represents a state in the state machine.
type State string

// Transition represents a state transition.
type Transition struct {
	From  State
	To    State
	Event EventType
}

// Guard is a function that can prevent a state transition.
type Guard func(from, to State, event EventType, ctx interface{}) (bool, error)

// Action is a function that executes during a state transition.
type Action func(from, to State, event EventType, ctx interface{}) error

// StateMachine manages state transitions with guards and actions.
type StateMachine struct {
	name        string
	states      []State
	transitions []Transition
	guards      map[Transition][]Guard
	actions     map[Transition][]Action
	initial     State
	current     State
	version     int64
}

// StateMachineOption configures the state machine.
type StateMachineOption func(*StateMachine)

// WithInitialState sets the initial state.
func WithInitialState(state State) StateMachineOption {
	return func(sm *StateMachine) {
		sm.initial = state
		sm.current = state
	}
}

// WithGuard adds a guard for a specific transition.
func WithGuard(from, to State, event EventType, guard Guard) StateMachineOption {
	return func(sm *StateMachine) {
		trans := Transition{From: from, To: to, Event: event}
		sm.guards[trans] = append(sm.guards[trans], guard)
	}
}

// WithAction adds an action for a specific transition.
func WithAction(from, to State, event EventType, action Action) StateMachineOption {
	return func(sm *StateMachine) {
		trans := Transition{From: from, To: to, Event: event}
		sm.actions[trans] = append(sm.actions[trans], action)
	}
}

// NewStateMachine creates a new state machine with the given name and states.
func NewStateMachine(name string, states []State, opts ...StateMachineOption) (*StateMachine, error) {
	if len(states) == 0 {
		return nil, fmt.Errorf("state machine requires at least one state")
	}

	sm := &StateMachine{
		name:        name,
		states:      states,
		transitions: []Transition{},
		guards:      make(map[Transition][]Guard),
		actions:     make(map[Transition][]Action),
		initial:     states[0],
		current:     states[0],
		version:     0,
	}

	for _, opt := range opts {
		opt(sm)
	}

	return sm, nil
}

// AddTransition registers a valid state transition.
func (sm *StateMachine) AddTransition(from, to State, event EventType) error {
	if !sm.hasState(from) {
		return fmt.Errorf("unknown state: %s", from)
	}
	if !sm.hasState(to) {
		return fmt.Errorf("unknown state: %s", to)
	}

	trans := Transition{From: from, To: to, Event: event}
	for _, t := range sm.transitions {
		if t == trans {
			return fmt.Errorf("transition already exists: %s -> %s on %s", from, to, event)
		}
	}

	sm.transitions = append(sm.transitions, trans)
	return nil
}

// CanTransition checks if a transition is valid without executing it.
func (sm *StateMachine) CanTransition(to State, event EventType) bool {
	trans := Transition{From: sm.current, To: to, Event: event}
	for _, t := range sm.transitions {
		if t == trans {
			return true
		}
	}
	return false
}

// CurrentState returns the current state.
func (sm *StateMachine) CurrentState() State {
	return sm.current
}

// Version returns the current version.
func (sm *StateMachine) Version() int64 {
	return sm.version
}

// Transition executes a state transition with guards and actions.
func (sm *StateMachine) Transition(to State, event EventType, ctx interface{}) error {
	return sm.transitionTo(sm.current, to, event, ctx)
}

// transitionTo executes a state transition from a specific state.
func (sm *StateMachine) transitionTo(from, to State, event EventType, ctx interface{}) error {
	trans := Transition{From: from, To: to, Event: event}

	// Find the transition
	var found bool
	for _, t := range sm.transitions {
		if t == trans {
			found = true
			break
		}
	}
	if !found {
		return ErrStateTransition(sm.name, string(from), string(to))
	}

	// Execute guards
	for _, guard := range sm.guards[trans] {
		allowed, err := guard(from, to, event, ctx)
		if err != nil {
			return err
		}
		if !allowed {
			return ErrStateTransition(sm.name, string(from), string(to))
		}
	}

	// Execute actions before transition
	for _, action := range sm.actions[trans] {
		if err := action(from, to, event, ctx); err != nil {
			return err
		}
	}

	// Update state and version
	sm.current = to
	sm.version++

	return nil
}

// hasState checks if the state machine has a given state.
func (sm *StateMachine) hasState(state State) bool {
	for _, s := range sm.states {
		if s == state {
			return true
		}
	}
	return false
}

// Transitions returns all registered transitions.
func (sm *StateMachine) Transitions() []Transition {
	return sm.transitions
}

// States returns all states.
func (sm *StateMachine) States() []State {
	return sm.states
}

// Reset resets the state machine to its initial state.
func (sm *StateMachine) Reset() {
	sm.current = sm.initial
	sm.version = 0
}

// AssetStateMachine is a specialized state machine for assets.
type AssetStateMachine struct {
	*StateMachine
}

// Asset states as defined in DESIGN.md.
const (
	AssetStateCreated    State = "CREATED"
	AssetStateEvaluating State = "EVALUATING"
	AssetStateEvaluated  State = "EVALUATED"
	AssetStatePromoted   State = "PROMOTED"
	AssetStateArchived   State = "ARCHIVED"
)

// NewAssetStateMachine creates a new asset state machine.
func NewAssetStateMachine() (*AssetStateMachine, error) {
	states := []State{
		AssetStateCreated,
		AssetStateEvaluating,
		AssetStateEvaluated,
		AssetStatePromoted,
		AssetStateArchived,
	}

	sm, err := NewStateMachine("Asset", states, WithInitialState(AssetStateCreated))
	if err != nil {
		return nil, err
	}

	asm := &AssetStateMachine{StateMachine: sm}

	// Add valid transitions as per DESIGN.md
	// CREATED --[Eval Pass]--> EVALUATED
	_ = asm.AddTransition(AssetStateCreated, AssetStateEvaluating, EventEvalStarted)
	_ = asm.AddTransition(AssetStateEvaluating, AssetStateEvaluated, EventEvalCompleted)
	_ = asm.AddTransition(AssetStateEvaluating, AssetStateCreated, EventEvalFailed)

	// EVALUATED --[Label Set Prod]--> PROMOTED
	_ = asm.AddTransition(AssetStateEvaluated, AssetStatePromoted, EventLabelPromoted)

	// Content Changed can revert to CREATED
	_ = asm.AddTransition(AssetStateEvaluated, AssetStateCreated, EventPromptAssetUpdated)
	_ = asm.AddTransition(AssetStatePromoted, AssetStateCreated, EventPromptAssetUpdated)

	// Archive transition
	_ = asm.AddTransition(AssetStateCreated, AssetStateArchived, EventPromptAssetArchived)
	_ = asm.AddTransition(AssetStateEvaluated, AssetStateArchived, EventPromptAssetArchived)
	_ = asm.AddTransition(AssetStatePromoted, AssetStateArchived, EventPromptAssetArchived)

	return asm, nil
}

// CanEval returns true if the asset can be evaluated from its current state.
func (asm *AssetStateMachine) CanEval() bool {
	return asm.CanTransition(AssetStateEvaluating, EventEvalStarted) ||
		asm.CanTransition(AssetStateCreated, EventEvalStarted)
}

// CanPromote returns true if the asset can be promoted from its current state.
func (asm *AssetStateMachine) CanPromote() bool {
	return asm.CanTransition(AssetStatePromoted, EventLabelPromoted) ||
		asm.CanTransition(AssetStateEvaluated, EventLabelPromoted)
}

// CanArchive returns true if the asset can be archived from its current state.
func (asm *AssetStateMachine) CanArchive() bool {
	return asm.CanTransition(AssetStateArchived, EventPromptAssetArchived)
}

// EvalGuard is a guard that checks if evaluation is allowed.
func EvalGuard(evalThreshold int) Guard {
	return func(from, to State, event EventType, ctx interface{}) (bool, error) {
		if event == EventLabelPromoted {
			// Check if ctx contains eval score
			if score, ok := ctx.(int); ok {
				if score < evalThreshold {
					return false, ErrDomainViolation("eval_threshold",
						fmt.Sprintf("eval score %d below threshold %d", score, evalThreshold))
				}
			}
		}
		return true, nil
	}
}

// RecordAction is an action that records the transition in the aggregate.
func RecordAction(record func(from, to State, event EventType)) Action {
	return func(from, to State, event EventType, ctx interface{}) error {
		record(from, to, event)
		return nil
	}
}

// AggregateStateMachine is a state machine that tracks transitions in an aggregate.
type AggregateStateMachine struct {
	*StateMachine
	onTransition func(from, to State, event EventType)
}

// NewAggregateStateMachine creates a new aggregate state machine with a callback.
func NewAggregateStateMachine(name string, states []State, onTransition func(from, to State, event EventType), opts ...StateMachineOption) (*AggregateStateMachine, error) {
	opts = append(opts, func(sm *StateMachine) {
		if onTransition != nil {
			sm.AddTransition(states[0], states[0], "") // Ensure states exist
		}
	})

	sm, err := NewStateMachine(name, states, opts...)
	if err != nil {
		return nil, err
	}

	return &AggregateStateMachine{
		StateMachine: sm,
		onTransition: onTransition,
	}, nil
}

// Transition executes a transition and calls the callback.
func (asm *AggregateStateMachine) Transition(to State, event EventType, ctx interface{}) error {
	from := asm.CurrentState()
	if err := asm.StateMachine.Transition(to, event, ctx); err != nil {
		return err
	}
	if asm.onTransition != nil {
		asm.onTransition(from, to, event)
	}
	return nil
}

// TransitionHistory tracks the history of state transitions.
type TransitionHistory struct {
	transitions []TransitionRecord
}

// TransitionRecord records a single state transition.
type TransitionRecord struct {
	From    State
	To      State
	Event   EventType
	At      time.Time
	Version int64
}

func (th *TransitionHistory) Record(from, to State, event EventType, version int64) {
	th.transitions = append(th.transitions, TransitionRecord{
		From:    from,
		To:      to,
		Event:   event,
		At:      time.Now(),
		Version: version,
	})
}

// Transitions returns all recorded transitions.
func (th *TransitionHistory) Transitions() []TransitionRecord {
	return th.transitions
}

// CanReach checks if one state can be reached from another through valid transitions.
func (sm *StateMachine) CanReach(from, to State) bool {
	visited := make(map[State]bool)
	queue := []State{from}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if current == to {
			return true
		}

		if visited[current] {
			continue
		}
		visited[current] = true

		for _, t := range sm.transitions {
			if t.From == current && !visited[t.To] {
				queue = append(queue, t.To)
			}
		}
	}

	return false
}

// IsTerminalState checks if a state has no outgoing transitions.
func (sm *StateMachine) IsTerminalState(state State) bool {
	for _, t := range sm.transitions {
		if t.From == state {
			return false
		}
	}
	return true
}

// ValidEvents returns the valid events that can be triggered from the current state.
func (sm *StateMachine) ValidEvents() []EventType {
	var events []EventType
	for _, t := range sm.transitions {
		if t.From == sm.current {
			if !slices.Contains(events, t.Event) {
				events = append(events, t.Event)
			}
		}
	}
	return events
}
