// Package fsm provides a lightweight, idiomatic, and extensible Finite State Machine (FSM) library for Go.
//
// Features:
//   - Simple API for defining states, triggers, and transitions.
//   - Side effects via transition actions and state entry/exit hooks.
//   - Fine-grained control with transition guards.
//   - Hierarchical states with support for nested state logic.
//   - Zero-allocation transitions for high performance.
//   - Type-safe by design, powered by Go generics.
//   - Thread-safe FSM specifications.
//
// Usage:
//
//	// Define your states and triggers as custom types.
//	type State uint
//	type Trigger uint
//
//	// Create a new FSM specification builder.
//	builder := fsm.NewSpecBuilder[State, Trigger, MyData](numStates, numTriggers)
//
//	// Define transitions and state hooks.
//	builder.Transition().From(...).On(...).To(...).WithAction(...).WithGuard(...)
//	builder.State(...).OnEntry(...).OnExit(...).Parent(...).Initial(...)
//
//	// Build the FSM specification (thread-safe, read-only).
//	spec := builder.Build()
//
//	// Create FSM instances as needed.
//	machine := fsm.New(spec, initialState)
//
//	// Fire triggers to perform transitions.
//	err := machine.Fire(ctx, trigger, data)
//
// See README.md and examples for more details.
package fsm

import (
	"context"
	"errors"
	"fmt"
	"slices"
)

const maxDepth = 10 // Needed constraint to allow zero-allocation fsm.Fire(...) runs.

var (
	ErrNotFound = fmt.Errorf("not found")
)

type (
	// Guard is a function that checks whether a transition is allowed to occur.
	Guard[D any] func(ctx context.Context, data D) error
	// Action is a function that performs an action when a transition occurs.
	Action[D any] func(ctx context.Context, data D) error
)

// Transition represents a state transition in the FSM.
type Transition[S ~uint, D any] struct {
	Valid  bool
	Next   S
	Guard  Guard[D]
	Action Action[D]
}

// StateHooks represents hooks that can be triggered on state entry and exit.
type StateHooks[D any] struct {
	OnEntry Action[D]
	OnExit  Action[D]
}

type specBuilder[S, T ~uint, D any] struct {
	stateCount    uint
	triggerCount  uint
	transitions   []Transition[S, D]
	stateHooks    []StateHooks[D]
	stateParents  []*S
	initialStates []*S

	/* ------------------------ Builder Chaining Helpers ------------------------ */
	// Enables tracking of completed transition definitions, so their .done() methods can be called when
	// building the FSM model.
	transitionToBuilders []*transitionToBuilder[S, T, D]
	// Enables panicking if not all transition definitions are completed, by comparing the number of started
	// transition definitions with the number of completed ones ( len(transitionToBuilders) ).
	numTransitionDefinitionsStarted int
	stateBuilders                   []*stateBuilder[S, T, D]
}

// NewSpecBuilder creates a new specBuilder used for building FSM specifications which define the states, triggers
// and transitions in the FSM.
func NewSpecBuilder[S, T ~uint, D any](numStates, numTriggers uint) *specBuilder[S, T, D] {
	if numStates == 0 || numTriggers == 0 {
		panic("number of states and triggers must be greater than zero")
	}

	return &specBuilder[S, T, D]{
		stateCount:    numStates,
		triggerCount:  numTriggers,
		transitions:   make([]Transition[S, D], numStates*numTriggers),
		stateHooks:    make([]StateHooks[D], numStates),
		stateParents:  make([]*S, numStates),
		initialStates: make([]*S, numStates),
	}
}

// Transition begins the definition of a new transition.
func (b *specBuilder[S, T, D]) Transition() *transitionBuilder[S, T, D] {
	b.numTransitionDefinitionsStarted++
	return &transitionBuilder[S, T, D]{
		b: b,
	}
}

// State begins the definition of a new state.
func (b *specBuilder[S, T, D]) State(state S) *stateBuilder[S, T, D] {
	sb := &stateBuilder[S, T, D]{
		b:     b,
		state: state,
	}
	b.stateBuilders = append(b.stateBuilders, sb)
	return sb
}

// Build finalizes the FSM specification and returns a new Spec instance.
func (b *specBuilder[S, T, D]) Build() *Spec[S, T, D] {
	if b.numTransitionDefinitionsStarted != len(b.transitionToBuilders) {
		panic("not all transition definitions were completed")
	}
	// Ensure all transition definitions are finalized and added to the FSM model by calling done() on
	// each transition builder.
	for _, tb := range b.transitionToBuilders {
		tb.done()
	}
	b.transitionToBuilders = nil // Remove circular references.

	// Ensure all state definitions are finalized and added to the FSM model by calling done() on each state builder.
	for _, sb := range b.stateBuilders {
		sb.done()
	}
	b.stateBuilders = nil // Remove circular references.

	// Ensure all initial states have the correct parent states defined.
	for stateWithInitialState, initialState := range b.initialStates {
		if initialState == nil {
			continue
		}
		parent := b.stateParents[*initialState]
		if parent == nil {
			panic(fmt.Sprintf("initial state (%v) must have a parent state defined", *initialState))
		}
		if *parent != S(stateWithInitialState) {
			panic(fmt.Sprintf("initial state (%v) must be same as parent state (%v)", *initialState, *parent))
		}
	}

	return &Spec[S, T, D]{
		stateCount:    b.stateCount,
		triggerCount:  b.triggerCount,
		transitions:   b.transitions,
		stateHooks:    b.stateHooks,
		stateParents:  b.stateParents,
		initialStates: b.initialStates,
	}
}

type transitionBuilder[S, T ~uint, D any] struct {
	b *specBuilder[S, T, D]
}

// From sets the source state for the transition.
func (tb *transitionBuilder[S, T, D]) From(state S) *transitionFromBuilder[S, T, D] {
	return &transitionFromBuilder[S, T, D]{
		b:    tb.b,
		from: state,
	}
}

type transitionFromBuilder[S, T ~uint, D any] struct {
	b    *specBuilder[S, T, D]
	from S
}

// On sets the trigger for the transition.
func (fb *transitionFromBuilder[S, T, D]) On(trigger T) *transitionOnBuilder[S, T, D] {
	return &transitionOnBuilder[S, T, D]{
		from:    fb.from,
		trigger: trigger,
		b:       fb.b,
	}
}

type transitionOnBuilder[S, T ~uint, D any] struct {
	from    S
	trigger T
	b       *specBuilder[S, T, D]
}

// To sets the target state for the transition.
func (tb *transitionOnBuilder[S, T, D]) To(state S) *transitionToBuilder[S, T, D] {
	toBuilder := &transitionToBuilder[S, T, D]{
		b:       tb.b,
		from:    tb.from,
		trigger: tb.trigger,
		to:      state,
	}
	tb.b.transitionToBuilders = append(tb.b.transitionToBuilders, toBuilder)
	return toBuilder
}

type transitionToBuilder[S, T ~uint, D any] struct {
	b       *specBuilder[S, T, D]
	from    S
	trigger T
	to      S
	guard   Guard[D]
	action  Action[D]
}

// WithGuard sets the guard for the transition.
func (tb *transitionToBuilder[S, T, D]) WithGuard(guard Guard[D]) *transitionToBuilder[S, T, D] {
	tb.guard = guard
	return tb
}

// WithAction sets the action for the transition. It is called when a transition is taking place from one state
// to another.
func (tb *transitionToBuilder[S, T, D]) WithAction(action Action[D]) *transitionToBuilder[S, T, D] {
	tb.action = action
	return tb
}

func (tb *transitionToBuilder[S, T, D]) done() *specBuilder[S, T, D] {
	idx := transitionIndex(tb.from, tb.trigger, tb.b.triggerCount)
	tb.b.transitions[idx] = Transition[S, D]{
		Valid:  true,
		Next:   tb.to,
		Guard:  tb.guard,
		Action: tb.action,
	}

	return tb.b
}

func transitionIndex[S, T ~uint](from S, trigger T, numTrigger uint) int {
	return int(uint(from)*numTrigger + uint(trigger))
}

type stateBuilder[S, T ~uint, D any] struct {
	b                 *specBuilder[S, T, D]
	state             S
	hooks             StateHooks[D]
	parent            S
	isParentSet       bool
	initialState      S
	isInitialStateSet bool
}

// OnEntry sets the OnEntry hook for the state. It is called when the state is entered.
func (sb *stateBuilder[S, T, D]) OnEntry(action Action[D]) *stateBuilder[S, T, D] {
	sb.hooks.OnEntry = action
	return sb
}

// OnExit sets the OnExit hook for the state. It is called when the state is exited.
func (sb *stateBuilder[S, T, D]) OnExit(action Action[D]) *stateBuilder[S, T, D] {
	sb.hooks.OnExit = action
	return sb
}

// Parent sets the parent state for hierarchical state machines.
func (sb *stateBuilder[S, T, D]) Parent(state S) *stateBuilder[S, T, D] {
	sb.parent = state
	sb.isParentSet = true
	return sb
}

// Initial sets the initial sub-state for hierarchical state machines.
//
// NOTE: If set, the initial state MUST have the same parent as the state it is being defined on. Otherwise,
// the call to build the FSM specification will panic.
func (sb *stateBuilder[S, T, D]) Initial(state S) *stateBuilder[S, T, D] {
	sb.initialState = state
	sb.isInitialStateSet = true
	return sb
}

func (sb *stateBuilder[S, T, D]) done() *specBuilder[S, T, D] {
	sb.b.stateHooks[sb.state] = sb.hooks
	if sb.isParentSet {
		sb.b.stateParents[sb.state] = &sb.parent
	}
	if sb.isInitialStateSet {
		sb.b.initialStates[sb.state] = &sb.initialState
	}
	return sb.b
}

// Spec represents the specification of the FSM, including its states, triggers, and transitions. It is safe to make
// shallow copies of the Spec as it is read-only, making it thread-safe.
type Spec[S, T ~uint, D any] struct {
	stateCount    uint
	triggerCount  uint
	transitions   []Transition[S, D]
	stateHooks    []StateHooks[D]
	stateParents  []*S
	initialStates []*S
}

// MermaidJSDiagram returns a state diagram in Mermaid.js syntax for the FSM Spec.
func (spec *Spec[S, T, D]) MermaidJSDiagram() string {
	diagram := "stateDiagram-v2\n"
	for from := uint(0); from < spec.stateCount; from++ {
		for trigger := uint(0); trigger < spec.triggerCount; trigger++ {
			idx := transitionIndex(S(from), T(trigger), spec.triggerCount)
			trans := spec.transitions[idx]
			if trans.Valid {
				fromStr := fmt.Sprintf("%v", S(from))
				toStr := fmt.Sprintf("%v", trans.Next)
				triggerStr := fmt.Sprintf("%v", T(trigger))
				diagram += fromStr + " --> " + toStr + " : " + triggerStr + "\n"
			}
		}
	}
	return diagram
}

// Machine is a finite state machine (FSM) instance. It keeps track of its current state and uses the FSM specification
// to determine valid state transitions and is the executor of defined transition actions and state hooks.
type Machine[S, T ~uint, D any] struct {
	state S
	spec  Spec[S, T, D]
}

// New creates a new FSM instance with the given specification and initial state.
func New[S, T ~uint, D any](spec *Spec[S, T, D], initialState S) *Machine[S, T, D] {
	return &Machine[S, T, D]{
		spec:  *spec,
		state: initialState,
	}
}

// State returns the current state of the FSM.
func (f *Machine[S, T, D]) State() S {
	return f.state
}

// ActiveHierarchy returns the active hierarchy of states in the FSM.
func (f *Machine[S, T, D]) ActiveHierarchy() []S {
	var hierarchy [maxDepth]S
	i := f.readHierarchy(f.state, &hierarchy)
	out := hierarchy[:i]
	return out
}

// IsIn checks if the FSM is currently in the specified state.
func (f *Machine[S, T, D]) IsIn(state S) bool {
	var hierarchy [maxDepth]S
	i := f.readHierarchy(f.state, &hierarchy)
	return slices.Contains(hierarchy[:i], state)
}

// Fire attempts to perform a state transition based on the provided trigger, options and current state.
// If a defined transition cannot be found for the current state, it will search up the state hierarchy for
// a valid transition until one is found. If none is found, it will return an ErrNotFound error.
func (f *Machine[S, T, D]) Fire(ctx context.Context, trigger T, opts D) error {
	state := f.state
	var transition Transition[S, D]
	for {
		trans, err := f.findTransition(trigger, state)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				parent := f.spec.stateParents[state]
				if parent != nil {
					state = *parent // Move up the hierarchy and try to find a transition from the parent state.
					continue
				}
			}
			return fmt.Errorf("finding transition for trigger (%v) and current state (%v): %w", trigger, f.state, err)
		}

		// Return an error if the guard rejects the transition.
		if guard := trans.Guard; guard != nil {
			if err := guard(ctx, opts); err != nil {
				return fmt.Errorf("guard rejected transition: %w", err)
			}
		}

		transition = trans
		break
	}

	// Try to find least common ancestor (LCA).
	// 	        Root (LCA)
	//        /    \
	//    Child A   Child B
	//      |          |
	// Grandchild A  Grandchild B
	var lca *S
	var lcaTargetStatesIdx *int
	var sourceStatesArr [maxDepth]S // Using a constant array to avoid allocations.
	var targetStatesArr [maxDepth]S
	i := f.readHierarchy(f.state, &sourceStatesArr)
	sourceStates := sourceStatesArr[:i]
	i = f.readHierarchy(transition.Next, &targetStatesArr)
	targetStates := targetStatesArr[:i]
outerLoop:
	for i := 1; i < len(sourceStates); i++ {
		for j := 0; j < len(targetStates); j++ {
			if sourceStates[i] == targetStates[j] {
				lca = &sourceStates[i]
				lcaTargetStatesIdx = &j
				break outerLoop
			}
		}
	}

	// Move up the hierarchy and invoke OnExit hooks for all ancestor states, except for the LCA if it exists.
	// Example: grandchild (current state in FSM) => child => root
	for _, state := range sourceStates {
		if lca != nil && state == *lca { // Do not run OnExit for LCA as we're not leaving that state.
			break
		}
		// Invoke the state's OnExit hook if it exists.
		if onExit := f.spec.stateHooks[state].OnExit; onExit != nil {
			if err := onExit(ctx, opts); err != nil {
				return fmt.Errorf("invoking OnExit state hook for state %v: %w", state, err)
			}
		}
	}

	// Return an error if the transition's action fails.
	if action := transition.Action; action != nil {
		if err := action(ctx, opts); err != nil {
			return fmt.Errorf("invoking transition action from states (%v) to (%v): %w", state, transition.Next, err)
		}
	}

	// Move down the hierarchy and invoke OnEntry hooks for all descendant states, starting at the LCA if it exists,
	// otherwise from the root of the hierarchy.
	// Example (No LCA): root => child => grandchild (target state) :: Run OnEntry on root, child and then grandchild
	// Example (With LCA): root => child (LCA) => grandchild (target state) :: Run OnEntry on grandchild
	startIdx := 0
	if lcaTargetStatesIdx != nil {
		startIdx = *lcaTargetStatesIdx
	} else {
		startIdx = len(targetStates) - 1
	}
	for i := startIdx; i >= 0; i-- {
		state := targetStates[i]
		if lca != nil && state == *lca { // Do not run OnEntry for LCA as we're not entering that state.
			continue
		}
		// Invoke the state's OnEntry hook if it exists.
		if onEntry := f.spec.stateHooks[state].OnEntry; onEntry != nil {
			if err := onEntry(ctx, opts); err != nil {
				return fmt.Errorf("invoking OnEntry state hook for state (%v): %w", state, err)
			}
		}
	}

	// If configured, set the FSM's current state to the defined initial sub-state and run its OnEntry hook.
	intialSubstate := f.spec.initialStates[transition.Next]
	if intialSubstate != nil {
		// Invoke the state's OnEntry hook if it exists.
		if onEntry := f.spec.stateHooks[*intialSubstate].OnEntry; onEntry != nil {
			if err := onEntry(ctx, opts); err != nil {
				return fmt.Errorf("invoking OnEntry state hook for state (%v): %w", *intialSubstate, err)
			}
		}
		f.state = *intialSubstate
		return nil
	}

	// Update the current state.
	f.state = transition.Next
	return nil
}

// CanFire checks if a state transition can be made given the trigger, current state and the guard defined
// for the transition.
func (f *Machine[S, T, D]) CanFire(ctx context.Context, trigger T, opts D) bool {
	trans, err := f.findTransition(trigger, f.state)
	if err != nil {
		return false
	}

	// Return false if the transition is not defined.
	if !trans.Valid {
		return false
	}

	// Return false if the guard rejects the transition.
	if guard := trans.Guard; guard != nil {
		if err := guard(ctx, opts); err != nil {
			return false
		}
	}

	return true
}

func (f *Machine[S, T, D]) findTransition(trigger T, state S) (Transition[S, D], error) {
	transIdx := transitionIndex(state, trigger, f.spec.triggerCount)
	trans := f.spec.transitions[transIdx]
	if !trans.Valid {
		return Transition[S, D]{}, ErrNotFound
	}
	return trans, nil
}

func (f *Machine[S, T, D]) readHierarchy(fromState S, hierarchy *[maxDepth]S) int {
	var next *S = &fromState
	i := 0
	for ; next != nil && i < maxDepth; i++ {
		(*hierarchy)[i] = *next
		next = f.spec.stateParents[*next]
	}
	return i
}
