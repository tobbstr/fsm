package fsm

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	locked state = iota
	unlocked
	root
	child
	grandchild
)

const (
	unlock trigger = iota
	lock
)

type state uint

func (s state) String() string {
	switch s {
	case locked:
		return "locked"
	case unlocked:
		return "unlocked"
	case root:
		return "root"
	case child:
		return "child"
	case grandchild:
		return "grandchild"
	default:
		return fmt.Sprintf("state(%d)", s)
	}
}

type trigger uint

func (t trigger) String() string {
	switch t {
	case unlock:
		return "unlock"
	case lock:
		return "lock"
	default:
		return fmt.Sprintf("trigger(%d)", t)
	}
}

type data struct{}

func TestNewSpecBuilder(t *testing.T) {
	// Test Types
	type (
		args struct {
			numStates   uint
			numTriggers uint
		}
		given struct {
			args args
		}
		want struct {
			specBuilder *specBuilder[state, trigger, data]
			panic       bool
		}
	)

	// Test Cases
	tests := []struct {
		name  string
		given given
		want  want
	}{
		{
			name: "panics when numStates is 0",
			given: given{
				args: args{
					numStates:   0,
					numTriggers: 1,
				},
			},
			want: want{
				specBuilder: nil,
				panic:       true,
			},
		},
		{
			name: "panics when numTriggers is 0",
			given: given{
				args: args{
					numStates:   1,
					numTriggers: 0,
				},
			},
			want: want{
				specBuilder: nil,
				panic:       true,
			},
		},
		{
			name: "returns valid specBuilder when inputs are valid",
			given: given{
				args: args{
					numStates:   4,
					numTriggers: 3,
				},
			},
			want: want{
				specBuilder: &specBuilder[state, trigger, data]{
					stateCount:    4,
					triggerCount:  3,
					transitions:   make([]Transition[state, data], 12),
					stateHooks:    make([]StateHooks[data], 4),
					stateParents:  make([]*state, 4),
					initialStates: make([]*state, 4),
				},
				panic: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			/* ---------------------------------- When ---------------------------------- */
			require := require.New(t)
			res := tryUnarySupplier(func() *specBuilder[state, trigger, data] {
				return NewSpecBuilder[state, trigger, data](tt.given.args.numStates, tt.given.args.numTriggers)
			})

			/* ---------------------------------- Then ---------------------------------- */
			if tt.want.panic {
				require.True(res.panicked, "Expected panic but did not get one")
				require.False(res.optional.valid, "Expected optional value to be unset")
				return
			}
			require.False(res.panicked, "Expected no panic but got one")
			require.True(res.optional.valid, "Expected optional value to be set")
			require.Equal(tt.want.specBuilder, res.optional.value)
		})
	}
}
func TestSpecBuilder_Transition(t *testing.T) {
	// Test Types
	type (
		given struct {
			numTransitionCalls int
		}
		want struct {
			numTransitionDefinitionsStarted int
		}
	)

	// Test Cases
	tests := []struct {
		name  string
		given given
		want  want
	}{
		{
			name: "every call to Transition increments the numTransitionDefinitionsStarted counter",
			given: given{
				numTransitionCalls: 3,
			},
			want: want{
				numTransitionDefinitionsStarted: 3,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			/* ---------------------------------- Given --------------------------------- */
			require := require.New(t)
			builder := NewSpecBuilder[state, trigger, data](5, 4)
			var got *transitionBuilder[state, trigger, data]

			/* ---------------------------------- When ---------------------------------- */
			for range tt.given.numTransitionCalls {
				got = builder.Transition()
			}

			/* ---------------------------------- Then ---------------------------------- */
			require.Equal(tt.want.numTransitionDefinitionsStarted, builder.numTransitionDefinitionsStarted, "Unexpected number of transition definitions started")
			require.Equal(builder, got.b, "Transition builder does not reference the correct spec builder")
		})
	}
}

func TestSpecBuilder_State(t *testing.T) {
	// Test Types
	type (
		given struct {
			states []state
		}
	)

	// Test Cases
	tests := []struct {
		name  string
		given given
	}{
		{
			name: "adds a stateBuilder for each state defined",
			given: given{
				states: []state{locked, unlocked},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			/* ---------------------------------- Given --------------------------------- */
			require := require.New(t)
			builder := NewSpecBuilder[state, trigger, data](5, 4)

			/* ---------------------------------- When ---------------------------------- */
			for _, state := range tt.given.states {
				builder.State(state)
			}

			/* ---------------------------------- Then ---------------------------------- */
			require.Len(tt.given.states, len(builder.stateBuilders), "Unexpected number of state builders created")
			for _, sb := range builder.stateBuilders {
				require.Equal(builder, sb.b, "State builder does not reference the correct spec builder")
			}
		})
	}
}

func TestSpecBuilder_Build(t *testing.T) {
	// Test Types
	type (
		outputs struct {
			// Transitions
			guardCalled  *bool
			actionCalled *bool
			// States
			onEntryCalled *bool
			onExitCalled  *bool
		}
		given struct {
			numStates   uint
			numTriggers uint
			configure   func(*specBuilder[state, trigger, data], outputs)
			fsmTrigger  trigger
		}
		want struct {
			panic         bool
			numStates     uint
			numTriggers   uint
			guardCalled   bool
			actionCalled  bool
			onEntryCalled bool
			onExitCalled  bool
		}
	)

	// Test Cases
	tests := []struct {
		name  string
		given given
		want  want
	}{
		{
			name: "panics when incomplete transition is defined 1",
			given: given{
				numStates:   5,
				numTriggers: 4,
				configure: func(b *specBuilder[state, trigger, data], o outputs) {
					b.Transition()
				},
			},
			want: want{
				panic: true,
			},
		},
		{
			name: "panics when incomplete transition is defined 2",
			given: given{
				numStates:   5,
				numTriggers: 4,
				configure: func(b *specBuilder[state, trigger, data], o outputs) {
					b.Transition().From(unlocked)
				},
			},
			want: want{
				panic: true,
			},
		},
		{
			name: "panics when incomplete transition is defined 3",
			given: given{
				numStates:   5,
				numTriggers: 4,
				configure: func(b *specBuilder[state, trigger, data], o outputs) {
					b.Transition().From(unlocked).On(lock)
				},
			},
			want: want{
				panic: true,
			},
		},
		{
			name: "panics when initial state without parent defined",
			given: given{
				numStates:   5,
				numTriggers: 4,
				configure: func(b *specBuilder[state, trigger, data], o outputs) {
					b.State(root).Initial(child)
				},
			},
			want: want{
				panic: true,
			},
		},
		{
			name: "panics when initial state with wrong parent defined",
			given: given{
				numStates:   5,
				numTriggers: 4,
				configure: func(b *specBuilder[state, trigger, data], o outputs) {
					b.State(root).Initial(child)
					b.State(child).Parent(unlocked)
				},
			},
			want: want{
				panic: true,
			},
		},
		{
			name: "single transition defined",
			given: given{
				numStates:   5,
				numTriggers: 4,
				configure: func(b *specBuilder[state, trigger, data], o outputs) {
					b.Transition().From(unlocked).On(lock).To(locked)
				},
				fsmTrigger: lock,
			},
			want: want{
				panic:       false,
				numStates:   5,
				numTriggers: 4,
			},
		},
		{
			name: "transition action and guards called",
			given: given{
				numStates:   5,
				numTriggers: 4,
				configure: func(b *specBuilder[state, trigger, data], o outputs) {
					b.Transition().From(unlocked).On(lock).To(locked).
						WithAction("actionCalled is set", func(ctx context.Context, opts data) error {
							*o.actionCalled = true
							return nil
						}).
						WithGuard("guardCalled is set", func(opts data) error {
							*o.guardCalled = true
							return nil
						})
					b.Transition().From(locked).On(unlock).To(unlocked)
				},
				fsmTrigger: lock,
			},
			want: want{
				panic:        false,
				numStates:    5,
				numTriggers:  4,
				guardCalled:  true,
				actionCalled: true,
			},
		},
		{
			name: "state hooks called",
			given: given{
				numStates:   5,
				numTriggers: 4,
				configure: func(b *specBuilder[state, trigger, data], o outputs) {
					b.Transition().From(unlocked).On(lock).To(locked)
					b.Transition().From(locked).On(unlock).To(unlocked)
					b.State(unlocked).OnExit(func(ctx context.Context, opts data) error {
						*o.onExitCalled = true
						return nil
					})
					b.State(locked).OnEntry(func(ctx context.Context, opts data) error {
						*o.onEntryCalled = true
						return nil
					})
				},
				fsmTrigger: lock,
			},
			want: want{
				panic:         false,
				numStates:     5,
				numTriggers:   4,
				onEntryCalled: true,
				onExitCalled:  true,
			},
		},
		{
			name: "initial state with correct parent defined",
			given: given{
				numStates:   5,
				numTriggers: 4,
				configure: func(b *specBuilder[state, trigger, data], o outputs) {
					b.State(root).Initial(child)
					b.State(child).Parent(root)
					b.Transition().From(unlocked).On(lock).To(locked)
				},
				fsmTrigger: lock,
			},
			want: want{
				panic:       false,
				numStates:   5,
				numTriggers: 4,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			/* ---------------------------------- Given --------------------------------- */
			require := require.New(t)
			builder := NewSpecBuilder[state, trigger, data](tt.given.numStates, tt.given.numTriggers)

			// Outputs to be checked against in the Then section of the test.
			var (
				guardCalled   bool
				actionCalled  bool
				onEntryCalled bool
				onExitCalled  bool
			)
			tt.given.configure(builder, outputs{
				guardCalled:   &guardCalled,
				actionCalled:  &actionCalled,
				onEntryCalled: &onEntryCalled,
				onExitCalled:  &onExitCalled,
			})

			/* ---------------------------------- When ---------------------------------- */
			res := tryUnarySupplier(func() *Spec[state, trigger, data] {
				return builder.Build()
			})

			/* ---------------------------------- Then ---------------------------------- */
			// Assert the function panics as expected.
			if tt.want.panic {
				require.True(res.panicked, "Expected panic but did not get one")
				require.False(res.optional.valid, "Expected optional value to be unset")
				return
			}
			require.True(res.optional.valid, "Expected optional value to be set")
			got := res.optional.value

			// Assert the number of states and triggers in the built spec.
			require.Equal(tt.want.numStates, got.stateCount, "Unexpected number of states in built spec")
			require.Equal(tt.want.numTriggers, got.triggerCount, "Unexpected number of triggers in built spec")

			// Build the FSM from the specification and fire the given trigger.
			fsm := New(got, unlocked)
			err := fsm.Fire(t.Context(), tt.given.fsmTrigger, data{})
			require.NoError(err, "Unexpected error when firing trigger")

			// Assert transition guard and action calls.
			require.Equal(tt.want.guardCalled, guardCalled, "Unexpected value for guardCalled")
			require.Equal(tt.want.actionCalled, actionCalled, "Unexpected value for actionCalled")

			// Assert state entry and exit calls.
			require.Equal(tt.want.onEntryCalled, onEntryCalled, "Unexpected value for onEntryCalled")
			require.Equal(tt.want.onExitCalled, onExitCalled, "Unexpected value for onExitCalled")
		})
	}
}

func TestMachine_Fire(t *testing.T) {
	/* ---------------------------------- Given --------------------------------- */
	require := require.New(t)

	// Create a new FSM specification.
	specBuilder := NewSpecBuilder[state, trigger, data](2, 2)
	specBuilder.Transition().From(unlocked).On(lock).To(locked)
	specBuilder.Transition().From(locked).On(unlock).To(unlocked)
	spec := specBuilder.Build()

	// Create a new FSM with an initial state.
	fsm := New(spec, unlocked)

	// Assert the initial state of the FSM.
	require.Equal(unlocked, fsm.State(), "Expected initial state to be unlocked")

	/* --------------------------------- When 1 --------------------------------- */
	// Fire the lock trigger and assert the state is now locked.
	err := fsm.Fire(t.Context(), lock, data{})

	/* --------------------------------- Then 1 --------------------------------- */
	require.NoError(err, "Unexpected error when firing trigger")
	require.Equal(locked, fsm.State(), "Expected state to be locked")

	/* --------------------------------- When 2 --------------------------------- */
	// Fire the unlock trigger and assert the state is now unlocked.
	err = fsm.Fire(t.Context(), unlock, data{})

	/* --------------------------------- Then 2 --------------------------------- */
	require.NoError(err, "Unexpected error when firing trigger")
	require.Equal(unlocked, fsm.State(), "Expected state to be unlocked")
}

func TestMachine_Fire_ReturnsErrors(t *testing.T) {
	/* ---------------------------------- Given --------------------------------- */
	require := require.New(t)

	// Create a new FSM specification.
	specBuilder := NewSpecBuilder[state, trigger, data](2, 2)
	specBuilder.Transition().From(unlocked).On(lock).To(locked).
		WithGuard("always errors", func(data data) error {
			return fmt.Errorf("guard error")
		})
	spec := specBuilder.Build()

	// Create a new FSM with an initial state.
	fsm := New(spec, unlocked)

	/* --------------------------------- When 1 --------------------------------- */
	// Fire the lock trigger and assert the error is ErrTransitionRejected and the state is still unlocked.
	err := fsm.Fire(t.Context(), lock, data{})

	/* --------------------------------- Then 1 --------------------------------- */
	require.Error(err, "Expected error when firing trigger")
	require.ErrorIs(err, ErrTransitionRejected, "Expected ErrTransitionRejected when guard fails")
	require.Equal(unlocked, fsm.State(), "Expected state to be unlocked")

	/* --------------------------------- When 2 --------------------------------- */
	// Fire the unlock trigger and assert the error is ErrNotFound and the state is still unlocked.
	err = fsm.Fire(t.Context(), unlock, data{})

	/* --------------------------------- Then 2 --------------------------------- */
	require.Error(err, "Expected error when firing trigger")
	require.ErrorIs(err, ErrNotFound, "Expected ErrNotFound when transition is not found")
	require.Equal(unlocked, fsm.State(), "Expected state to be unlocked")
}

func TestMachine_CanFire(t *testing.T) {
	/* ---------------------------------- Given --------------------------------- */
	require := require.New(t)

	// Create a new FSM specification.
	spec := NewSpecBuilder[state, trigger, data](2, 2)
	spec.Transition().From(unlocked).On(lock).To(locked)

	// Create a new FSM with an initial state.
	fsm := New(spec.Build(), unlocked)

	/* ---------------------------------- When ---------------------------------- */
	got := fsm.CanFire(t.Context(), unlock, data{})

	/* ---------------------------------- Then ---------------------------------- */
	require.False(got, "Expected CanFire to return false for undefined transition")

	/* --------------------------------- When 2 --------------------------------- */
	got = fsm.CanFire(t.Context(), lock, data{})

	/* --------------------------------- Then 2 --------------------------------- */
	require.True(got, "Expected CanFire to return true for defined transition")
}

// TestMachine_Fire_HierarchicalStates_TriggerBubbling tests that trigger events bubble up the state hierarchy and
// that the first found transition is made.
func TestMachine_Fire_HierarchicalStates_TriggerBubbling(t *testing.T) {
	/* ---------------------------------- Given --------------------------------- */
	require := require.New(t)

	// Outputs
	rootActionCalled := false
	grandchildActionCalled := false

	// Create a new FSM specification.
	spec := NewSpecBuilder[state, trigger, data](5, 2)
	spec.Transition().From(root).On(lock).To(grandchild).WithAction("rootActionCalled is set", func(ctx context.Context, opts data) error {
		rootActionCalled = true
		return nil
	})
	spec.Transition().From(grandchild).On(unlock).To(child).WithAction("grandchildActionCalled is set", func(ctx context.Context, opts data) error {
		grandchildActionCalled = true
		return nil
	})

	// Configure state hierarchy.
	spec.State(grandchild).Parent(child)
	spec.State(child).Parent(root)

	// Create a new FSM with an initial state.
	fsm := New(spec.Build(), grandchild)

	/* ---------------------------------- When ---------------------------------- */
	err := fsm.Fire(t.Context(), lock, data{})

	/* ---------------------------------- Then ---------------------------------- */
	require.NoError(err, "Unexpected error when firing trigger")
	require.True(rootActionCalled, "Expected root action to be called")
	require.False(grandchildActionCalled, "Expected grandchild action to NOT be called")
}

func TestMachine_Fire_HierarchicalStates_ReturnsErrorWhenNoTransitionIsFound(t *testing.T) {
	/* ---------------------------------- Given --------------------------------- */
	require := require.New(t)

	// Outputs
	rootActionCalled := false
	grandchildActionCalled := false

	// Create a new FSM specification.
	spec := NewSpecBuilder[state, trigger, data](5, 2)
	spec.Transition().From(root).On(unlock).To(grandchild).WithAction("rootActionCalled is set", func(ctx context.Context, opts data) error {
		rootActionCalled = true
		return nil
	})
	spec.Transition().From(grandchild).On(unlock).To(child).WithAction("grandchildActionCalled is set", func(ctx context.Context, opts data) error {
		grandchildActionCalled = true
		return nil
	})

	// Configure state hierarchy.
	spec.State(grandchild).Parent(child)
	spec.State(child).Parent(root)

	// Create a new FSM with an initial state.
	fsm := New(spec.Build(), grandchild)

	/* ---------------------------------- When ---------------------------------- */
	err := fsm.Fire(t.Context(), lock, data{})

	/* ---------------------------------- Then ---------------------------------- */
	require.Error(err, "Unexpected error when firing trigger")
	require.False(rootActionCalled, "Expected root action to be called")
	require.False(grandchildActionCalled, "Expected grandchild action to NOT be called")
}

func TestMachine_ActiveHierarchy(t *testing.T) {
	/* ---------------------------------- Given --------------------------------- */
	require := require.New(t)

	// Create a new FSM specification.
	spec := NewSpecBuilder[state, trigger, data](5, 2)

	// Configure state hierarchy.
	spec.State(grandchild).Parent(child)
	spec.State(child).Parent(root)

	// Create a new FSM with an initial state.
	fsm := New(spec.Build(), grandchild)

	/* ---------------------------------- When ---------------------------------- */
	const maxDepth = 10
	var hierarchy [maxDepth]state
	depth := fsm.readHierarchy(grandchild, &hierarchy)

	/* ---------------------------------- Then ---------------------------------- */
	require.Equal(depth, 3, "Expected state hierarchy to have 3 levels")
	require.Equal(hierarchy[0], grandchild, "Expected grandchild to be the first level")
	require.Equal(hierarchy[1], child, "Expected child to be the second level")
	require.Equal(hierarchy[2], root, "Expected root to be the third level")
}

func TestMachine_Fire_HierarchicalStates_CallsStateHooksAndTransActionInCorrectOrder(t *testing.T) {
	/* ---------------------------------- Given --------------------------------- */
	require := require.New(t)

	// Outputs
	actionCalled := false
	grandchildOnEntryCalled := false
	grandchildOnExitCalled := false
	lcaOnEntryCalled := false
	lcaOnExitCalled := false
	rootOnEntryCalled := false
	rootOnExitCalled := false
	lockedOnEntryCalled := false
	lockedOnExitCalled := false
	callOrder := make([]string, 0, 10)

	// Create a new FSM specification.
	specBuilder := NewSpecBuilder[state, trigger, data](5, 2)
	specBuilder.Transition().From(grandchild).On(lock).To(locked).WithAction("actionCalled is set", func(ctx context.Context, opts data) error {
		actionCalled = true
		callOrder = append(callOrder, "action")
		return nil
	})

	// Configure state hierarchy.
	specBuilder.State(root).
		OnEntry(func(ctx context.Context, opts data) error {
			rootOnEntryCalled = true
			callOrder = append(callOrder, "rootOnEntry")
			return nil
		}).
		OnExit(func(ctx context.Context, opts data) error {
			rootOnExitCalled = true
			callOrder = append(callOrder, "rootOnExit")
			return nil
		})
	specBuilder.State(grandchild).
		Parent(child).
		OnEntry(func(ctx context.Context, opts data) error {
			grandchildOnEntryCalled = true
			callOrder = append(callOrder, "grandchildOnEntry")
			return nil
		}).
		OnExit(func(ctx context.Context, opts data) error {
			grandchildOnExitCalled = true
			callOrder = append(callOrder, "grandchildOnExit")
			return nil
		})
	specBuilder.State(child).
		Parent(root).
		OnEntry(func(ctx context.Context, opts data) error {
			lcaOnEntryCalled = true
			callOrder = append(callOrder, "lcaOnEntry")
			return nil
		}).
		OnExit(func(ctx context.Context, opts data) error {
			lcaOnExitCalled = true
			callOrder = append(callOrder, "lcaOnExit")
			return nil
		})
	specBuilder.State(locked).
		Parent(child).
		OnEntry(func(ctx context.Context, opts data) error {
			lockedOnEntryCalled = true
			callOrder = append(callOrder, "lockedOnEntry")
			return nil
		}).
		OnExit(func(ctx context.Context, opts data) error {
			lockedOnExitCalled = true
			callOrder = append(callOrder, "lockedOnExit")
			return nil
		})

	// Create a new FSM with an initial state.
	spec := specBuilder.Build()
	fsm := New(spec, grandchild)

	/* ---------------------------------- When ---------------------------------- */
	err := fsm.Fire(t.Context(), lock, data{})

	/* ---------------------------------- Then ---------------------------------- */
	require.NoError(err, "Unexpected error when firing trigger")

	// Assert the state hook calls when moving up the hierarchy.
	require.True(grandchildOnExitCalled, "Expected grandchild onExit to be called")
	require.False(grandchildOnEntryCalled, "Expected grandchild onEntry to NOT be called")
	require.False(lcaOnEntryCalled, "Expected LCA onEntry to NOT be called")
	require.False(lcaOnExitCalled, "Expected LCA onExit to NOT be called")
	require.False(rootOnEntryCalled, "Expected root onEntry to NOT be called")
	require.False(rootOnExitCalled, "Expected root onExit to NOT be called")

	// Assert the transition action was called.
	require.True(actionCalled, "Expected root action to be called")

	// Assert the state hook calls when moving down the hierarchy.
	require.True(lockedOnEntryCalled, "Expected locked onEntry to be called")
	require.False(lockedOnExitCalled, "Expected locked onExit to NOT be called")

	// Assert the call order.
	require.Equal([]string{"grandchildOnExit", "action", "lockedOnEntry"}, callOrder, "Unexpected call order")

	// Assert the FSM state.
	require.Equal(locked, fsm.State(), "Expected FSM to be in locked state")
}

func TestSpec_MermaidDiagram(t *testing.T) {
	/* ---------------------------------- Given --------------------------------- */
	require := require.New(t)

	// Create a new FSM specification with multiple states and transitions
	specBuilder := NewSpecBuilder[state, trigger, data](5, 2)
	specBuilder.Transition().From(unlocked).On(lock).To(locked).
		WithGuard("true", func(data data) error { return nil }).
		WithAction("runOp()", func(ctx context.Context, data data) error { return nil })
	specBuilder.Transition().From(locked).On(unlock).To(unlocked)
	specBuilder.Transition().From(root).On(lock).To(child)
	specBuilder.Transition().From(child).On(unlock).To(grandchild)
	specBuilder.Transition().From(grandchild).On(lock).To(root)

	// Configure state hierarchy
	specBuilder.State(grandchild).Parent(child)
	specBuilder.State(child).Parent(root)

	spec := specBuilder.Build()

	// Expected lines in the diagram
	expectedLines := []string{
		"stateDiagram-v2",
		"unlocked --> locked : lock [true] / runOp()",
		"locked --> unlocked : unlock",
		"root --> child : lock",
		"child --> grandchild : unlock",
		"grandchild --> root : lock",
	}

	/* ---------------------------------- When ---------------------------------- */
	diagram := spec.MermaidJSDiagram()

	/* ---------------------------------- Then ---------------------------------- */
	for _, line := range expectedLines {
		require.Contains(diagram, line, "Diagram missing expected line: %q", line)
	}
}

func TestMachine_Fire_HierarchicalStates_InitialSubstate(t *testing.T) {
	/* ---------------------------------- Given --------------------------------- */
	require := require.New(t)

	// Outputs
	actionCalled := false
	grandchildOnEntryCalled := false
	grandchildOnExitCalled := false
	childOnEntryCalled := false
	childOnExitCalled := false
	rootOnEntryCalled := false
	rootOnExitCalled := false
	lockedOnEntryCalled := false
	lockedOnExitCalled := false
	callOrder := make([]string, 0, 10)

	// Create a new FSM specification.
	specBuilder := NewSpecBuilder[state, trigger, data](5, 2)
	specBuilder.Transition().From(grandchild).On(lock).To(root).WithAction("actionCalled is set", func(ctx context.Context, opts data) error {
		actionCalled = true
		callOrder = append(callOrder, "action")
		return nil
	})

	// Configure state hierarchy.
	specBuilder.State(root).
		Initial(locked).
		OnEntry(func(ctx context.Context, opts data) error {
			rootOnEntryCalled = true
			callOrder = append(callOrder, "rootOnEntry")
			return nil
		}).
		OnExit(func(ctx context.Context, opts data) error {
			rootOnExitCalled = true
			callOrder = append(callOrder, "rootOnExit")
			return nil
		})
	specBuilder.State(grandchild).
		Parent(child).
		OnEntry(func(ctx context.Context, opts data) error {
			grandchildOnEntryCalled = true
			callOrder = append(callOrder, "grandchildOnEntry")
			return nil
		}).
		OnExit(func(ctx context.Context, opts data) error {
			grandchildOnExitCalled = true
			callOrder = append(callOrder, "grandchildOnExit")
			return nil
		})

	specBuilder.State(child).
		Parent(root).
		OnEntry(func(ctx context.Context, opts data) error {
			childOnEntryCalled = true
			callOrder = append(callOrder, "childOnEntry")
			return nil
		}).
		OnExit(func(ctx context.Context, opts data) error {
			childOnExitCalled = true
			callOrder = append(callOrder, "childOnExit")
			return nil
		})
	specBuilder.State(locked).
		Parent(root).
		OnEntry(func(ctx context.Context, opts data) error {
			lockedOnEntryCalled = true
			callOrder = append(callOrder, "lockedOnEntry")
			return nil
		}).
		OnExit(func(ctx context.Context, opts data) error {
			lockedOnExitCalled = true
			callOrder = append(callOrder, "lockedOnExit")
			return nil
		})

	// Build the FSM specification.
	spec := specBuilder.Build()

	// Create a new FSM with an initial state.
	fsm := New(spec, grandchild)

	/* ---------------------------------- When ---------------------------------- */
	err := fsm.Fire(t.Context(), lock, data{})

	/* ---------------------------------- Then ---------------------------------- */
	require.NoError(err, "Unexpected error when firing trigger")

	// Assert the state hook calls when moving up the hierarchy.
	require.True(grandchildOnExitCalled, "Expected grandchild onExit to be called")
	require.False(grandchildOnEntryCalled, "Expected grandchild onEntry to NOT be called")
	require.False(childOnEntryCalled, "Expected child onEntry to NOT be called")
	require.True(childOnExitCalled, "Expected child onExit to be called")
	require.False(rootOnEntryCalled, "Expected root onEntry NOT to be called")
	require.False(rootOnExitCalled, "Expected root onExit to NOT be called")

	// Assert the transition action was called.
	require.True(actionCalled, "Expected root action to be called")

	// Assert the state hook calls when moving down the hierarchy.
	require.True(lockedOnEntryCalled, "Expected locked onEntry to be called")
	require.False(lockedOnExitCalled, "Expected locked onExit to NOT be called")

	// Assert the call order.
	require.Equal([]string{"grandchildOnExit", "childOnExit", "action", "lockedOnEntry"}, callOrder, "Unexpected call order")

	// Assert the FSM state.
	require.Equal(locked, fsm.State(), "Expected FSM to be in locked state")
}
