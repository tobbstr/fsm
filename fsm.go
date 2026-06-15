// Package fsm provides a lightweight, idiomatic, and extensible Finite State Machine (FSM) library for Go.
//
// Features:
//   - Simple API for defining states, triggers, and transitions.
//   - Stimuli-based transitions: the trigger and input together form the stimuli that attempt to stimulate
//     the FSM to move into another state.
//   - Side effects via transition actions and state entry/exit hooks.
//   - Fine-grained control with guarded branches using boolean conditions.
//   - Multiple guarded transitions per (from, trigger) with first-match-wins semantics.
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
//	// Define your input type (per-call business data).
//	type OrderInput struct {
//		OrderID    string
//		CustomerID string
//	}
//
//	// Create a new FSM specification builder.
//	builder := fsm.NewBuilder[State, Trigger, OrderInput]()
//
//	// Define transitions and state hooks.
//	builder.From(...).On(...).To(...).Do(...).When(...)
//	builder.State(...).OnEntry(...).OnExit(...).Parent(...).Initial(...)
//
//	// Build the FSM specification (thread-safe, read-only).
//	spec := builder.Build()
//
//	// Create FSM instances as needed.
//	machine := fsm.New(spec, initialState)
//
//	// Fire triggers to perform transitions with business data.
//	err := machine.Fire(ctx, trigger, OrderInput{OrderID: "123", CustomerID: "456"})
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
//	func SetupOrderFSM(services Services) *fsm.Spec[State, Trigger, OrderInput] {
//		builder := fsm.NewBuilder[State, Trigger, OrderInput]()
//
//		// Services are captured from outer scope.
//		builder.
//			From(Pending).On(Confirm).To(Confirmed).
//			Do("save order", func(ctx context.Context, input OrderInput) error {
//				// Access services via closure - clean and type-safe.
//				services.Logger.Printf("Confirming order %s", input.OrderID)
//				return saveOrder(services.DB, input.OrderID)
//			})
//
//		return builder.Build()
//	}
//
//	// Usage: Clean call sites with only business data.
//	spec := SetupOrderFSM(services)
//	machine := fsm.New(spec, Pending)
//	machine.Fire(ctx, Confirm, OrderInput{OrderID: "123"})
//
// See README.md and examples for more details.
package fsm

import (
	"context"
	"fmt"
	"slices"
	"strings"
)

const maxDepth = 10 // Needed constraint to allow zero-allocation fsm.Fire(...) runs.

var (
	ErrNotFound           = fmt.Errorf("not found")
	ErrTransitionRejected = fmt.Errorf("transition rejected")
)

type (
	// Condition is a predicate that determines whether a branch is taken.
	Condition[Input any] func(input Input) bool
	// Action is a function that performs an action when a transition occurs.
	Action[Input any] func(ctx context.Context, input Input) error
)

// branch is one candidate transition within a (from, trigger) group.
type branch[S ~uint, Input any] struct {
	next       S
	cond       Condition[Input] // nil = unconditional (always matches)
	condDesc   string
	action     Action[Input]
	actionDesc string
}

// slot holds all branches for one (from, trigger), in definition order.
// Inline first keeps the overwhelmingly common single-branch case allocation-free;
// more is nil unless the group actually has multiple branches.
type slot[S ~uint, Input any] struct {
	valid bool
	first branch[S, Input]
	more  []branch[S, Input] // nil for single-branch groups
}

// match returns the first branch whose condition is nil or returns true, else nil. No allocation.
func (s *slot[S, Input]) match(in Input) *branch[S, Input] {
	if s.first.cond == nil || s.first.cond(in) {
		return &s.first
	}
	for i := range s.more {
		if s.more[i].cond == nil || s.more[i].cond(in) {
			return &s.more[i]
		}
	}
	return nil
}

// StateHooks represents hooks that can be triggered on state entry and exit.
type StateHooks[Input any] struct {
	OnEntry Action[Input]
	OnExit  Action[Input]
}

// Builder builds FSM specifications. Create one with NewBuilder.
type Builder[S, T ~uint, Input any] struct {
	branchDefs    []*branchDef[S, T, Input]
	onSteps       []*onStep[S, T, Input]
	stateBuilders []*stateBuilder[S, T, Input]
}

// NewBuilder creates a new Builder used for building FSM specifications which define the states, triggers
// and transitions in the FSM.
//
// The number of states and triggers is derived automatically from the definitions added to the builder, so there is
// no need to declare them up front. Build() sizes the specification to fit the highest state and trigger index
// referenced by any transition or state definition.
func NewBuilder[S, T ~uint, Input any]() *Builder[S, T, Input] {
	return &Builder[S, T, Input]{}
}

// branchDef accumulates the fields for one branch in definition order.
type branchDef[S, T ~uint, Input any] struct {
	from       S
	trigger    T
	to         S
	cond       Condition[Input]
	condDesc   string
	action     Action[Input]
	actionDesc string
	isDefault  bool // set by Otherwise
}

// onStep tracks that an On() call was made and whether a To() completed it.
type onStep[S, T ~uint, Input any] struct {
	b        *Builder[S, T, Input]
	from     S
	trigger  T
	consumed bool
}

// fromStep is returned by Builder.From.
type fromStep[S, T ~uint, Input any] struct {
	b    *Builder[S, T, Input]
	from S
}

// branchStep is returned after To() and allows chaining When/Do/To/Otherwise.
type branchStep[S, T ~uint, Input any] struct {
	b       *Builder[S, T, Input]
	cur     *branchDef[S, T, Input]
	from    S
	trigger T
}

// From begins the definition of a new transition group.
func (b *Builder[S, T, Input]) From(state S) *fromStep[S, T, Input] {
	return &fromStep[S, T, Input]{b: b, from: state}
}

// On sets the trigger for the transition group.
func (fs *fromStep[S, T, Input]) On(trigger T) *onStep[S, T, Input] {
	os := &onStep[S, T, Input]{b: fs.b, from: fs.from, trigger: trigger}
	fs.b.onSteps = append(fs.b.onSteps, os)
	return os
}

// To opens the first branch of the group with the given target state.
func (os *onStep[S, T, Input]) To(state S) *branchStep[S, T, Input] {
	os.consumed = true
	def := &branchDef[S, T, Input]{from: os.from, trigger: os.trigger, to: state}
	os.b.branchDefs = append(os.b.branchDefs, def)
	return &branchStep[S, T, Input]{b: os.b, cur: def, from: os.from, trigger: os.trigger}
}

// When sets a boolean condition and its description on the current branch.
func (bs *branchStep[S, T, Input]) When(desc string, cond func(Input) bool) *branchStep[S, T, Input] {
	bs.cur.cond = cond
	bs.cur.condDesc = desc
	return bs
}

// Do sets an action and its description on the current branch.
func (bs *branchStep[S, T, Input]) Do(desc string, action func(ctx context.Context, in Input) error) *branchStep[S, T, Input] {
	bs.cur.action = action
	bs.cur.actionDesc = desc
	return bs
}

// To closes the current branch and opens the next branch in the same group.
func (bs *branchStep[S, T, Input]) To(state S) *branchStep[S, T, Input] {
	def := &branchDef[S, T, Input]{from: bs.from, trigger: bs.trigger, to: state}
	bs.b.branchDefs = append(bs.b.branchDefs, def)
	bs.cur = def
	return bs
}

// Otherwise opens the final unconditional fallback branch.
func (bs *branchStep[S, T, Input]) Otherwise(state S) *branchStep[S, T, Input] {
	def := &branchDef[S, T, Input]{from: bs.from, trigger: bs.trigger, to: state, isDefault: true}
	bs.b.branchDefs = append(bs.b.branchDefs, def)
	bs.cur = def
	return bs
}

// State begins the definition of a new state.
func (b *Builder[S, T, Input]) State(state S) *stateBuilder[S, T, Input] {
	sb := &stateBuilder[S, T, Input]{
		b:     b,
		state: state,
	}
	b.stateBuilders = append(b.stateBuilders, sb)
	return sb
}

// Build finalizes the FSM specification and returns a new Spec instance.
func (b *Builder[S, T, Input]) Build() *Spec[S, T, Input] {
	// Completion check: every On() must have a following To().
	for _, os := range b.onSteps {
		if !os.consumed {
			panic(fmt.Sprintf("incomplete transition: From(%v).On(%v) has no To(...)", os.from, os.trigger))
		}
	}

	// Derive the FSM's dimensions from the definitions provided.
	var maxState, maxTrigger uint
	noteState := func(s S) {
		if uint(s) > maxState {
			maxState = uint(s)
		}
	}
	for _, def := range b.branchDefs {
		noteState(def.from)
		noteState(def.to)
		if uint(def.trigger) > maxTrigger {
			maxTrigger = uint(def.trigger)
		}
	}
	for _, sb := range b.stateBuilders {
		noteState(sb.state)
		if sb.isParentSet {
			noteState(sb.parent)
		}
		if sb.isInitialStateSet {
			noteState(sb.initialState)
		}
	}

	// Always allocate at least 1x1 to avoid empty-slice edge cases.
	stateCount := maxState + 1
	triggerCount := maxTrigger + 1

	slots := make([]slot[S, Input], stateCount*triggerCount)
	stateHooks := make([]StateHooks[Input], stateCount)
	stateParents := make([]*S, stateCount)
	initialStates := make([]*S, stateCount)

	// Group branchDefs into slots in definition order.
	for _, def := range b.branchDefs {
		idx := transitionIndex(def.from, def.trigger, triggerCount)
		br := branch[S, Input]{
			next:       def.to,
			cond:       def.cond,
			condDesc:   def.condDesc,
			action:     def.action,
			actionDesc: def.actionDesc,
		}
		if !slots[idx].valid {
			slots[idx].valid = true
			slots[idx].first = br
		} else {
			slots[idx].more = append(slots[idx].more, br)
		}
	}

	// Per-group ordering validation: unconditional branch must be last.
	for from := uint(0); from < stateCount; from++ {
		for trigger := uint(0); trigger < triggerCount; trigger++ {
			idx := transitionIndex(S(from), T(trigger), triggerCount)
			s := &slots[idx]
			if !s.valid {
				continue
			}
			// Build a flat view to check ordering.
			branches := make([]branch[S, Input], 0, 1+len(s.more))
			branches = append(branches, s.first)
			branches = append(branches, s.more...)
			for i, br := range branches {
				if br.cond == nil && i < len(branches)-1 {
					panic(fmt.Sprintf(
						"unconditional branch from state (%v) on trigger (%v) to (%v) shadows later branches; an unconditional/Otherwise branch must be last",
						S(from), T(trigger), br.next,
					))
				}
			}
		}
	}

	// Finalize state builders.
	for _, sb := range b.stateBuilders {
		stateHooks[sb.state] = sb.hooks
		if sb.isParentSet {
			parent := sb.parent
			stateParents[sb.state] = &parent
		}
		if sb.isInitialStateSet {
			initial := sb.initialState
			initialStates[sb.state] = &initial
		}
	}

	// Ensure all initial states have the correct parent states defined.
	for stateWithInitialState, initialState := range initialStates {
		if initialState == nil {
			continue
		}
		parent := stateParents[*initialState]
		if parent == nil {
			panic(fmt.Sprintf("initial state (%v) must have a parent state defined", *initialState))
		}
		if *parent != S(stateWithInitialState) {
			panic(fmt.Sprintf("initial state (%v) must be same as parent state (%v)", *initialState, *parent))
		}
	}

	// Ensure no state hierarchy is deeper than maxDepth.
	for s := uint(0); s < stateCount; s++ {
		depth := 1
		for parent := stateParents[s]; parent != nil; parent = stateParents[*parent] {
			depth++
			if depth > maxDepth {
				panic(fmt.Sprintf(
					"state hierarchy starting at state (%v) exceeds the maximum supported depth of %d (check for an overly deep hierarchy or a cycle in the parent definitions)",
					S(s), maxDepth,
				))
			}
		}
	}

	return &Spec[S, T, Input]{
		stateCount:    stateCount,
		triggerCount:  triggerCount,
		slots:         slots,
		stateHooks:    stateHooks,
		stateParents:  stateParents,
		initialStates: initialStates,
	}
}

func transitionIndex[S, T ~uint](from S, trigger T, numTrigger uint) int {
	return int(uint(from)*numTrigger + uint(trigger))
}

type stateBuilder[S, T ~uint, Input any] struct {
	b                 *Builder[S, T, Input]
	state             S
	hooks             StateHooks[Input]
	parent            S
	isParentSet       bool
	initialState      S
	isInitialStateSet bool
}

// OnEntry sets the OnEntry hook for the state. It is called when the state is entered.
func (sb *stateBuilder[S, T, Input]) OnEntry(action Action[Input]) *stateBuilder[S, T, Input] {
	sb.hooks.OnEntry = action
	return sb
}

// OnExit sets the OnExit hook for the state. It is called when the state is exited.
func (sb *stateBuilder[S, T, Input]) OnExit(action Action[Input]) *stateBuilder[S, T, Input] {
	sb.hooks.OnExit = action
	return sb
}

// Parent sets the parent state for hierarchical state machines.
func (sb *stateBuilder[S, T, Input]) Parent(state S) *stateBuilder[S, T, Input] {
	sb.parent = state
	sb.isParentSet = true
	return sb
}

// Initial sets the initial sub-state for hierarchical state machines.
//
// NOTE: If set, the initial state MUST have the same parent as the state it is being defined on. Otherwise,
// the call to build the FSM specification will panic.
func (sb *stateBuilder[S, T, Input]) Initial(state S) *stateBuilder[S, T, Input] {
	sb.initialState = state
	sb.isInitialStateSet = true
	return sb
}

// Spec represents the specification of the FSM, including its states, triggers, and transitions. It is safe to make
// shallow copies of the Spec as it is read-only, making it thread-safe.
type Spec[S, T ~uint, Input any] struct {
	stateCount    uint
	triggerCount  uint
	slots         []slot[S, Input]
	stateHooks    []StateHooks[Input]
	stateParents  []*S
	initialStates []*S
}

// MermaidJSDiagram returns a state diagram in Mermaid.js syntax for the FSM Spec.
func (spec *Spec[S, T, Input]) MermaidJSDiagram() string {
	diagram := "stateDiagram-v2\n"
	for from := uint(0); from < spec.stateCount; from++ {
		for trigger := uint(0); trigger < spec.triggerCount; trigger++ {
			idx := transitionIndex(S(from), T(trigger), spec.triggerCount)
			s := &spec.slots[idx]
			if !s.valid {
				continue
			}
			fromStr := fmt.Sprintf("%v", S(from))
			triggerStr := fmt.Sprintf("%v", T(trigger))
			branches := make([]branch[S, Input], 0, 1+len(s.more))
			branches = append(branches, s.first)
			branches = append(branches, s.more...)
			for _, br := range branches {
				toStr := fmt.Sprintf("%v", br.next)
				guardDesc := ""
				if br.condDesc != "" {
					guardDesc = " [" + br.condDesc + "]"
				}
				actionDesc := ""
				if br.actionDesc != "" {
					actionDesc = " / " + br.actionDesc
				}
				diagram += fromStr + " --> " + toStr + " : " + triggerStr + guardDesc + actionDesc + "\n"
			}
		}
	}
	return diagram
}

// Machine is a finite state machine (FSM) instance. It keeps track of its current state and uses the FSM specification
// to determine valid state transitions and is the executor of defined transition actions and state hooks.
type Machine[S, T ~uint, Input any] struct {
	state S
	spec  Spec[S, T, Input]
}

// New creates a new FSM instance with the given specification and initial state.
func New[S, T ~uint, Input any](spec *Spec[S, T, Input], initialState S) *Machine[S, T, Input] {
	return &Machine[S, T, Input]{
		spec:  *spec,
		state: initialState,
	}
}

// State returns the current state of the FSM.
func (m *Machine[S, T, Input]) State() S {
	return m.state
}

// ActiveHierarchy returns the active hierarchy of states in the FSM.
func (m *Machine[S, T, Input]) ActiveHierarchy() []S {
	var hierarchy [maxDepth]S
	i := m.readHierarchy(m.state, &hierarchy)
	out := hierarchy[:i]
	return out
}

// IsIn checks if the FSM is currently in the specified state.
func (m *Machine[S, T, Input]) IsIn(state S) bool {
	var hierarchy [maxDepth]S
	i := m.readHierarchy(m.state, &hierarchy)
	return slices.Contains(hierarchy[:i], state)
}

// Fire attempts to perform a state transition based on the provided trigger, input and current state.
// The trigger and input together form the stimuli that attempt to stimulate the FSM to move into another state.
//
// If a defined transition cannot be found for the current state, it will search up the state hierarchy for
// a valid transition until one is found. If none is found, it will return an ErrNotFound error.
//
// If transitions exist for (state, trigger) but no branch's condition matches, it returns ErrTransitionRejected.
// The error message lists all tried condition descriptions from every rule-bearing level considered.
func (m *Machine[S, T, Input]) Fire(ctx context.Context, trigger T, input Input) error {
	state := m.state
	var selected *branch[S, Input]
	sawSlot := false

	// Accumulate rejected condition descriptions per level for the error message.
	// Only allocated on the rejection path — never on success.
	type levelRejection struct {
		state     S
		condDescs []string
	}
	var rejectedLevels []levelRejection

	for {
		if s := m.slotAt(trigger, state); s != nil && s.valid {
			sawSlot = true
			if b := s.match(input); b != nil {
				selected = b
				break
			}
			// Collect condition descriptions for error reporting (only on miss path).
			descs := make([]string, 0, 1+len(s.more))
			if s.first.condDesc != "" {
				descs = append(descs, fmt.Sprintf("%q", s.first.condDesc))
			}
			for _, br := range s.more {
				if br.condDesc != "" {
					descs = append(descs, fmt.Sprintf("%q", br.condDesc))
				}
			}
			rejectedLevels = append(rejectedLevels, levelRejection{state: state, condDescs: descs})
		}
		parent := m.spec.stateParents[state]
		if parent == nil {
			if sawSlot {
				// Build rejection error with all tried conditions.
				var sb strings.Builder
				fmt.Fprintf(&sb, "transition rejected for trigger (%v) from state (%v): no branch matched", trigger, m.state)
				if len(rejectedLevels) > 0 {
					sb.WriteString("\n  [")
					for i, lvl := range rejectedLevels {
						if i > 0 {
							sb.WriteString("; ")
						}
						fmt.Fprintf(&sb, "%v: %s", lvl.state, strings.Join(lvl.condDescs, ", "))
					}
					sb.WriteString("]")
				}
				return fmt.Errorf("%w\n%s", ErrTransitionRejected, sb.String())
			}
			return fmt.Errorf("finding transition for trigger (%v) and current state (%v): %w", trigger, m.state, ErrNotFound)
		}
		state = *parent
	}

	// LCA exit/action/entry machinery (unchanged logic, reads selected.next/selected.action).
	var lca *S
	var lcaTargetStatesIdx *int
	var sourceStatesArr [maxDepth]S
	var targetStatesArr [maxDepth]S
	i := m.readHierarchy(m.state, &sourceStatesArr)
	sourceStates := sourceStatesArr[:i]
	i = m.readHierarchy(selected.next, &targetStatesArr)
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

	for _, st := range sourceStates {
		if lca != nil && st == *lca {
			break
		}
		if onExit := m.spec.stateHooks[st].OnExit; onExit != nil {
			if err := onExit(ctx, input); err != nil {
				return fmt.Errorf("invoking OnExit state hook for state %v: %w", st, err)
			}
		}
	}

	if action := selected.action; action != nil {
		if err := action(ctx, input); err != nil {
			return fmt.Errorf("invoking transition action from states (%v) to (%v): %w", state, selected.next, err)
		}
	}

	startIdx := 0
	if lcaTargetStatesIdx != nil {
		startIdx = *lcaTargetStatesIdx
	} else {
		startIdx = len(targetStates) - 1
	}
	for i := startIdx; i >= 0; i-- {
		st := targetStates[i]
		if lca != nil && st == *lca {
			continue
		}
		if onEntry := m.spec.stateHooks[st].OnEntry; onEntry != nil {
			if err := onEntry(ctx, input); err != nil {
				return fmt.Errorf("invoking OnEntry state hook for state (%v): %w", st, err)
			}
		}
	}

	initialSubstate := m.spec.initialStates[selected.next]
	if initialSubstate != nil {
		if onEntry := m.spec.stateHooks[*initialSubstate].OnEntry; onEntry != nil {
			if err := onEntry(ctx, input); err != nil {
				return fmt.Errorf("invoking OnEntry state hook for state (%v): %w", *initialSubstate, err)
			}
		}
		m.state = *initialSubstate
		return nil
	}

	m.state = selected.next
	return nil
}

// CanFire checks if a state transition can be made given the trigger, input, current state and the conditions defined
// for the transition branches. The trigger and input together form the stimuli that would attempt to stimulate the FSM.
// It returns true if a branch matches, otherwise false.
//
// It will search up the state hierarchy for a valid transition until one is found or the root is reached.
// Implemented on the shared alloc-free walk — never calls Explain.
func (m *Machine[S, T, Input]) CanFire(ctx context.Context, trigger T, input Input) bool {
	state := m.state
	for {
		if s := m.slotAt(trigger, state); s != nil && s.valid {
			if b := s.match(input); b != nil {
				return true
			}
			// Slot exists but no branch matched — keep bubbling (same as Fire).
		}
		parent := m.spec.stateParents[state]
		if parent == nil {
			return false
		}
		state = *parent
	}
}

// Outcome describes the verdict of a branch evaluation during Explain.
type Outcome uint8

const (
	NotMatched Outcome = iota // condition returned false
	Matched                   // this was the winning branch
	Skipped                   // a later branch that was never evaluated (first-match-wins)
)

// BranchVerdict is the evaluation result for one branch in an Explain call.
type BranchVerdict[S ~uint] struct {
	Target    S
	Condition string // condDesc; "" for unconditional/Otherwise
	Outcome   Outcome
}

// LevelVerdict is the result for one hierarchy level in an Explain call.
type LevelVerdict[S ~uint] struct {
	State    S
	Matched  bool
	Branches []BranchVerdict[S]
}

// Decision is the full introspection result returned by Explain.
type Decision[S ~uint] struct {
	Found        bool              // did any level have a rule for (state, trigger)?
	Matched      bool              // did a branch match anywhere?
	Target       S                 // state that would be entered (valid iff Matched)
	ResolvedFrom S                 // level whose branch won; if none matched, the deepest level considered
	Levels       []LevelVerdict[S] // deepest-first: current state, then ancestors with rules up to the resolver
}

// Explain reports a multi-level decision trace for what Fire would do with the given trigger and input.
// It allocates; never called by Fire or CanFire.
func (m *Machine[S, T, Input]) Explain(trigger T, in Input) Decision[S] {
	state := m.state
	var levels []LevelVerdict[S]

	for {
		s := m.slotAt(trigger, state)
		if s == nil || !s.valid {
			parent := m.spec.stateParents[state]
			if parent == nil {
				break
			}
			state = *parent
			continue
		}

		// Evaluate branches, stopping at first match.
		branches := make([]branch[S, Input], 0, 1+len(s.more))
		branches = append(branches, s.first)
		branches = append(branches, s.more...)

		var verdicts []BranchVerdict[S]
		matchIdx := -1
		for i, br := range branches {
			if br.cond == nil || br.cond(in) {
				matchIdx = i
				break
			}
		}

		for i, br := range branches {
			var outcome Outcome
			switch {
			case i == matchIdx:
				outcome = Matched
			case matchIdx >= 0 && i > matchIdx:
				outcome = Skipped
			default:
				outcome = NotMatched
			}
			verdicts = append(verdicts, BranchVerdict[S]{
				Target:    br.next,
				Condition: br.condDesc,
				Outcome:   outcome,
			})
		}

		lv := LevelVerdict[S]{
			State:    state,
			Matched:  matchIdx >= 0,
			Branches: verdicts,
		}
		levels = append(levels, lv)

		if matchIdx >= 0 {
			// This level matched — stop bubbling.
			return Decision[S]{
				Found:        true,
				Matched:      true,
				Target:       branches[matchIdx].next,
				ResolvedFrom: state,
				Levels:       levels,
			}
		}

		parent := m.spec.stateParents[state]
		if parent == nil {
			break
		}
		state = *parent
	}

	if len(levels) == 0 {
		return Decision[S]{Found: false}
	}

	// Had rules but no branch matched.
	return Decision[S]{
		Found:        true,
		Matched:      false,
		ResolvedFrom: levels[0].State,
		Levels:       levels,
	}
}

// slotAt returns the slot for (state, trigger) with bounds checking, or nil if out of range.
func (m *Machine[S, T, Input]) slotAt(trigger T, state S) *slot[S, Input] {
	if uint(state) >= m.spec.stateCount || uint(trigger) >= m.spec.triggerCount {
		return nil
	}
	return &m.spec.slots[transitionIndex(state, trigger, m.spec.triggerCount)]
}

func (m *Machine[S, T, Input]) readHierarchy(fromState S, hierarchy *[maxDepth]S) int {
	state := fromState
	i := 0
	for i < maxDepth {
		(*hierarchy)[i] = state
		i++
		if uint(state) >= m.spec.stateCount {
			break
		}
		parent := m.spec.stateParents[state]
		if parent == nil {
			break
		}
		state = *parent
	}
	return i
}
