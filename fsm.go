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
//   - Closure-based dependency injection for clean separation of concerns.
//
// Basic Usage:
//
//	// Define your states and triggers as custom types.
//	type State uint
//	type Trigger uint
//
//	// Define your payload type (per-call business data).
//	type OrderPayload struct {
//		OrderID    string
//		CustomerID string
//	}
//
//	// Create a new FSM specification builder.
//	builder := fsm.NewSpecBuilder[State, Trigger, OrderPayload](numStates, numTriggers)
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
//	// Fire triggers to perform transitions with business data.
//	err := machine.Fire(ctx, trigger, OrderPayload{OrderID: "123", CustomerID: "456"})
//
// Dependency Injection Pattern:
//
// Infrastructure dependencies (database, services, logger) should be injected via closures
// when defining the FSM specification, keeping Fire() calls clean and focused on business data:
//
//	// Define services/dependencies.
//	type Services struct {
//		DB     *sql.DB
//		Logger *log.Logger
//	}
//
//	// Create spec with services captured via closure.
//	func SetupOrderFSM(services Services) *fsm.Spec[State, Trigger, OrderPayload] {
//		builder := fsm.NewSpecBuilder[State, Trigger, OrderPayload](numStates, numTriggers)
//
//		// Services are captured from outer scope.
//		builder.Transition().
//			From(Pending).On(Confirm).To(Confirmed).
//			WithAction("save order", func(ctx context.Context, payload OrderPayload) error {
//				// Access services via closure - clean and type-safe.
//				services.Logger.Printf("Confirming order %s", payload.OrderID)
//				return saveOrder(services.DB, payload.OrderID)
//			})
//
//		return builder.Build()
//	}
//
//	// Usage: Clean call sites with only business data.
//	spec := SetupOrderFSM(services)
//	machine := fsm.New(spec, Pending)
//	machine.Fire(ctx, Confirm, OrderPayload{OrderID: "123"})
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
	ErrNotFound           = fmt.Errorf("not found")
	ErrTransitionRejected = fmt.Errorf("transition rejected")
)

type (
	// Guard is a function that checks whether a transition is allowed to occur.
	Guard[Payload any] func(payload Payload) error
	// Action is a function that performs an action when a transition occurs.
	Action[Payload any] func(ctx context.Context, payload Payload) error
)

// Transition represents a state transition in the FSM.
type Transition[S ~uint, Payload any] struct {
	Valid             bool
	Next              S
	Guard             Guard[Payload]
	GuardDescription  string
	Action            Action[Payload]
	ActionDescription string
}

// StateHooks represents hooks that can be triggered on state entry and exit.
type StateHooks[Payload any] struct {
	OnEntry Action[Payload]
	OnExit  Action[Payload]
}

type specBuilder[S, T ~uint, Payload any] struct {
	stateCount    uint
	triggerCount  uint
	transitions   []Transition[S, Payload]
	stateHooks    []StateHooks[Payload]
	stateParents  []*S
	initialStates []*S

	/* ------------------------ Builder Chaining Helpers ------------------------ */
	// Enables tracking of completed transition definitions, so their .done() methods can be called when
	// building the FSM model.
	transitionToBuilders []*transitionToBuilder[S, T, Payload]
	// Enables panicking if not all transition definitions are completed, by comparing the number of started
	// transition definitions with the number of completed ones ( len(transitionToBuilders) ).
	numTransitionDefinitionsStarted int
	stateBuilders                   []*stateBuilder[S, T, Payload]
}

// NewSpecBuilder creates a new specBuilder used for building FSM specifications which define the states, triggers
// and transitions in the FSM.
func NewSpecBuilder[S, T ~uint, Payload any](numStates, numTriggers uint) *specBuilder[S, T, Payload] {
	if numStates == 0 || numTriggers == 0 {
		panic("number of states and triggers must be greater than zero")
	}

	return &specBuilder[S, T, Payload]{
		stateCount:    numStates,
		triggerCount:  numTriggers,
		transitions:   make([]Transition[S, Payload], numStates*numTriggers),
		stateHooks:    make([]StateHooks[Payload], numStates),
		stateParents:  make([]*S, numStates),
		initialStates: make([]*S, numStates),
	}
}

// Transition begins the definition of a new transition.
func (b *specBuilder[S, T, Payload]) Transition() *transitionBuilder[S, T, Payload] {
	b.numTransitionDefinitionsStarted++
	return &transitionBuilder[S, T, Payload]{
		b: b,
	}
}

// State begins the definition of a new state.
func (b *specBuilder[S, T, Payload]) State(state S) *stateBuilder[S, T, Payload] {
	sb := &stateBuilder[S, T, Payload]{
		b:     b,
		state: state,
	}
	b.stateBuilders = append(b.stateBuilders, sb)
	return sb
}

// Build finalizes the FSM specification and returns a new Spec instance.
func (b *specBuilder[S, T, Payload]) Build() *Spec[S, T, Payload] {
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

	return &Spec[S, T, Payload]{
		stateCount:    b.stateCount,
		triggerCount:  b.triggerCount,
		transitions:   b.transitions,
		stateHooks:    b.stateHooks,
		stateParents:  b.stateParents,
		initialStates: b.initialStates,
	}
}

type transitionBuilder[S, T ~uint, Payload any] struct {
	b *specBuilder[S, T, Payload]
}

// From sets the source state for the transition.
func (tb *transitionBuilder[S, T, Payload]) From(state S) *transitionFromBuilder[S, T, Payload] {
	return &transitionFromBuilder[S, T, Payload]{
		b:    tb.b,
		from: state,
	}
}

type transitionFromBuilder[S, T ~uint, Payload any] struct {
	b    *specBuilder[S, T, Payload]
	from S
}

// On sets the trigger for the transition.
func (fb *transitionFromBuilder[S, T, Payload]) On(trigger T) *transitionOnBuilder[S, T, Payload] {
	return &transitionOnBuilder[S, T, Payload]{
		from:    fb.from,
		trigger: trigger,
		b:       fb.b,
	}
}

type transitionOnBuilder[S, T ~uint, Payload any] struct {
	from    S
	trigger T
	b       *specBuilder[S, T, Payload]
}

// To sets the target state for the transition.
func (tb *transitionOnBuilder[S, T, Payload]) To(state S) *transitionToBuilder[S, T, Payload] {
	toBuilder := &transitionToBuilder[S, T, Payload]{
		b:       tb.b,
		from:    tb.from,
		trigger: tb.trigger,
		to:      state,
	}
	tb.b.transitionToBuilders = append(tb.b.transitionToBuilders, toBuilder)
	return toBuilder
}

type transitionToBuilder[S, T ~uint, Payload any] struct {
	b                 *specBuilder[S, T, Payload]
	from              S
	trigger           T
	to                S
	guard             Guard[Payload]
	guardDescription  string
	action            Action[Payload]
	actionDescription string
}

// WithGuard sets a guard function and its description for the transition.
//
// The guard is a predicate that determines whether the transition is allowed to occur.
// If the guard returns an error, the transition is blocked and the error is propagated.
//
// The desc parameter should provide a concise, human-readable explanation of the guard's purpose or logic.
// This description is used for documentation and visualization purposes, such as generating Mermaid.js diagrams,
// to make the FSM's specification easily understandable.
//
// Example desc values:
//
//	"balance >= amount"
//	"isUserAuthenticated"
//	"canWithdraw"
func (tb *transitionToBuilder[S, T, Payload]) WithGuard(desc string, guard Guard[Payload]) *transitionToBuilder[S, T, Payload] {
	tb.guard = guard
	tb.guardDescription = desc
	return tb
}

// WithAction sets an action function and its description for the transition.
//
// The action is executed when the transition occurs, allowing you to perform side effects such as updating state,
// calling external services, or emitting events. If the action returns an error, the transition is aborted and the
// error is propagated.
//
// The desc parameter should be a concise, human-readable description of what the action does. This is useful for
// documentation and visualization purposes, such as generating Mermaid.js diagrams, to make the FSM's specification
// easily understandable.
//
// Example desc values:
//
//	"deduct balance"
//	"send notification"
//	"logTransition()"
func (tb *transitionToBuilder[S, T, Payload]) WithAction(desc string, action Action[Payload]) *transitionToBuilder[S, T, Payload] {
	tb.action = action
	tb.actionDescription = desc
	return tb
}

func (tb *transitionToBuilder[S, T, Payload]) done() *specBuilder[S, T, Payload] {
	idx := transitionIndex(tb.from, tb.trigger, tb.b.triggerCount)
	tb.b.transitions[idx] = Transition[S, Payload]{
		Valid:             true,
		Next:              tb.to,
		Guard:             tb.guard,
		GuardDescription:  tb.guardDescription,
		Action:            tb.action,
		ActionDescription: tb.actionDescription,
	}

	return tb.b
}

func transitionIndex[S, T ~uint](from S, trigger T, numTrigger uint) int {
	return int(uint(from)*numTrigger + uint(trigger))
}

type stateBuilder[S, T ~uint, Payload any] struct {
	b                 *specBuilder[S, T, Payload]
	state             S
	hooks             StateHooks[Payload]
	parent            S
	isParentSet       bool
	initialState      S
	isInitialStateSet bool
}

// OnEntry sets the OnEntry hook for the state. It is called when the state is entered.
func (sb *stateBuilder[S, T, Payload]) OnEntry(action Action[Payload]) *stateBuilder[S, T, Payload] {
	sb.hooks.OnEntry = action
	return sb
}

// OnExit sets the OnExit hook for the state. It is called when the state is exited.
func (sb *stateBuilder[S, T, Payload]) OnExit(action Action[Payload]) *stateBuilder[S, T, Payload] {
	sb.hooks.OnExit = action
	return sb
}

// Parent sets the parent state for hierarchical state machines.
func (sb *stateBuilder[S, T, Payload]) Parent(state S) *stateBuilder[S, T, Payload] {
	sb.parent = state
	sb.isParentSet = true
	return sb
}

// Initial sets the initial sub-state for hierarchical state machines.
//
// NOTE: If set, the initial state MUST have the same parent as the state it is being defined on. Otherwise,
// the call to build the FSM specification will panic.
func (sb *stateBuilder[S, T, Payload]) Initial(state S) *stateBuilder[S, T, Payload] {
	sb.initialState = state
	sb.isInitialStateSet = true
	return sb
}

func (sb *stateBuilder[S, T, Payload]) done() *specBuilder[S, T, Payload] {
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
type Spec[S, T ~uint, Payload any] struct {
	stateCount    uint
	triggerCount  uint
	transitions   []Transition[S, Payload]
	stateHooks    []StateHooks[Payload]
	stateParents  []*S
	initialStates []*S
}

// MermaidJSDiagram returns a state diagram in Mermaid.js syntax for the FSM Spec.
func (spec *Spec[S, T, Payload]) MermaidJSDiagram() string {
	diagram := "stateDiagram-v2\n"
	for from := uint(0); from < spec.stateCount; from++ {
		for trigger := uint(0); trigger < spec.triggerCount; trigger++ {
			idx := transitionIndex(S(from), T(trigger), spec.triggerCount)
			trans := spec.transitions[idx]
			if trans.Valid {
				fromStr := fmt.Sprintf("%v", S(from))
				toStr := fmt.Sprintf("%v", trans.Next)
				triggerStr := fmt.Sprintf("%v", T(trigger))
				guardDesc := ""
				if trans.GuardDescription != "" {
					guardDesc = " [" + trans.GuardDescription + "]"
				}
				actionDesc := ""
				if trans.ActionDescription != "" {
					actionDesc = " / " + trans.ActionDescription
				}
				diagram += fromStr + " --> " + toStr + " : " + triggerStr + guardDesc + actionDesc + "\n"
			}
		}
	}
	return diagram
}

// Machine is a finite state machine (FSM) instance. It keeps track of its current state and uses the FSM specification
// to determine valid state transitions and is the executor of defined transition actions and state hooks.
type Machine[S, T ~uint, Payload any] struct {
	state S
	spec  Spec[S, T, Payload]
}

// New creates a new FSM instance with the given specification and initial state.
func New[S, T ~uint, Payload any](spec *Spec[S, T, Payload], initialState S) *Machine[S, T, Payload] {
	return &Machine[S, T, Payload]{
		spec:  *spec,
		state: initialState,
	}
}

// State returns the current state of the FSM.
func (m *Machine[S, T, Payload]) State() S {
	return m.state
}

// ActiveHierarchy returns the active hierarchy of states in the FSM.
func (m *Machine[S, T, Payload]) ActiveHierarchy() []S {
	var hierarchy [maxDepth]S
	i := m.readHierarchy(m.state, &hierarchy)
	out := hierarchy[:i]
	return out
}

// IsIn checks if the FSM is currently in the specified state.
func (m *Machine[S, T, Payload]) IsIn(state S) bool {
	var hierarchy [maxDepth]S
	i := m.readHierarchy(m.state, &hierarchy)
	return slices.Contains(hierarchy[:i], state)
}

// Fire attempts to perform a state transition based on the provided trigger, payload and current state.
//
// If a defined transition cannot be found for the current state, it will search up the state hierarchy for
// a valid transition until one is found. If none is found, it will return an ErrNotFound error.
//
// If a transition is found but has a guard that rejects the transition, it will return an ErrTransitionRejected error.
func (m *Machine[S, T, Payload]) Fire(ctx context.Context, trigger T, payload Payload) error {
	state := m.state
	var transition Transition[S, Payload]
	for {
		trans, err := m.findTransition(trigger, state)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				parent := m.spec.stateParents[state]
				if parent != nil {
					state = *parent // Move up the hierarchy and try to find a transition from the parent state.
					continue
				}
			}
			return fmt.Errorf("finding transition for trigger (%v) and current state (%v): %w", trigger, m.state, err)
		}

		// Return an error if the guard rejects the transition.
		if guard := trans.Guard; guard != nil {
			if err := guard(payload); err != nil {
				err = fmt.Errorf("rejecting transition from state (%v) to (%v) for trigger (%v): %w", state, trans.Next, trigger, err)
				return errors.Join(ErrTransitionRejected, err)
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
	i := m.readHierarchy(m.state, &sourceStatesArr)
	sourceStates := sourceStatesArr[:i]
	i = m.readHierarchy(transition.Next, &targetStatesArr)
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
		if onExit := m.spec.stateHooks[state].OnExit; onExit != nil {
			if err := onExit(ctx, payload); err != nil {
				return fmt.Errorf("invoking OnExit state hook for state %v: %w", state, err)
			}
		}
	}

	// Return an error if the transition's action fails.
	if action := transition.Action; action != nil {
		if err := action(ctx, payload); err != nil {
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
		if onEntry := m.spec.stateHooks[state].OnEntry; onEntry != nil {
			if err := onEntry(ctx, payload); err != nil {
				return fmt.Errorf("invoking OnEntry state hook for state (%v): %w", state, err)
			}
		}
	}

	// If configured, set the FSM's current state to the defined initial sub-state and run its OnEntry hook.
	intialSubstate := m.spec.initialStates[transition.Next]
	if intialSubstate != nil {
		// Invoke the state's OnEntry hook if it exists.
		if onEntry := m.spec.stateHooks[*intialSubstate].OnEntry; onEntry != nil {
			if err := onEntry(ctx, payload); err != nil {
				return fmt.Errorf("invoking OnEntry state hook for state (%v): %w", *intialSubstate, err)
			}
		}
		m.state = *intialSubstate
		return nil
	}

	// Update the current state.
	m.state = transition.Next
	return nil
}

// CanFire checks if a state transition can be made given the trigger, current state and the guard defined
// for the transition. It returns true if the transition can be made, otherwise false.
//
// It will search up the state hierarchy for a valid transition until one is found or the root is reached.
func (m *Machine[S, T, Payload]) CanFire(ctx context.Context, trigger T, payload Payload) bool {
	state := m.state
	for {
		trans, err := m.findTransition(trigger, state)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				parent := m.spec.stateParents[state]
				if parent != nil {
					state = *parent // Move up the hierarchy and try to find a transition from the parent state.
					continue
				}
			}
			return false
		}

		// Return an error if the guard rejects the transition.
		if guard := trans.Guard; guard != nil {
			if err := guard(payload); err != nil {
				return false
			}
		}
		break
	}
	return true
}

func (m *Machine[S, T, Payload]) findTransition(trigger T, state S) (Transition[S, Payload], error) {
	transIdx := transitionIndex(state, trigger, m.spec.triggerCount)
	trans := m.spec.transitions[transIdx]
	if !trans.Valid {
		return Transition[S, Payload]{}, ErrNotFound
	}
	return trans, nil
}

func (m *Machine[S, T, Payload]) readHierarchy(fromState S, hierarchy *[maxDepth]S) int {
	var next *S = &fromState
	i := 0
	for ; next != nil && i < maxDepth; i++ {
		(*hierarchy)[i] = *next
		next = m.spec.stateParents[*next]
	}
	return i
}
