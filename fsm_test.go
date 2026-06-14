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

type input struct{}

func TestNewSpecBuilder(t *testing.T) {
	require := require.New(t)

	/* ---------------------------------- When ---------------------------------- */
	builder := NewSpecBuilder[state, trigger, input]()

	/* ---------------------------------- Then ---------------------------------- */
	// The builder starts empty; dimensions are derived later, at Build() time.
	require.NotNil(builder, "Expected a non-nil builder")
	require.Empty(builder.transitionToBuilders, "Expected no transition definitions yet")
	require.Empty(builder.stateBuilders, "Expected no state definitions yet")
	require.Zero(builder.stateCount, "Expected stateCount to be unset before Build()")
	require.Zero(builder.triggerCount, "Expected triggerCount to be unset before Build()")
}

// TestBuild_DerivesDimensions verifies that Build() sizes the specification to fit the highest state and trigger
// index referenced by the definitions, removing the need to declare the counts up front.
func TestBuild_DerivesDimensions(t *testing.T) {
	// Test Cases
	tests := []struct {
		name             string
		configure        func(*specBuilder[state, trigger, input])
		wantStateCount   uint
		wantTriggerCount uint
	}{
		{
			name:             "empty builder yields a minimal 1x1 specification",
			configure:        func(b *specBuilder[state, trigger, input]) {},
			wantStateCount:   1,
			wantTriggerCount: 1,
		},
		{
			name: "dimensions derived from transition states and triggers",
			configure: func(b *specBuilder[state, trigger, input]) {
				// Highest state referenced is locked(0)/unlocked(1) => stateCount 2.
				// Highest trigger referenced is lock(1) => triggerCount 2.
				b.Transition().From(unlocked).On(lock).To(locked)
			},
			wantStateCount:   2,
			wantTriggerCount: 2,
		},
		{
			name: "dimensions account for states referenced only in the hierarchy",
			configure: func(b *specBuilder[state, trigger, input]) {
				// grandchild(4) is referenced only via the hierarchy, so it must still be in range.
				b.Transition().From(unlocked).On(lock).To(locked)
				b.State(grandchild).Parent(child)
				b.State(child).Parent(root)
			},
			wantStateCount:   5, // grandchild(4) + 1
			wantTriggerCount: 2, // lock(1) + 1
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require := require.New(t)

			/* ---------------------------------- Given --------------------------------- */
			builder := NewSpecBuilder[state, trigger, input]()
			tt.configure(builder)

			/* ---------------------------------- When ---------------------------------- */
			spec := builder.Build()

			/* ---------------------------------- Then ---------------------------------- */
			require.Equal(tt.wantStateCount, spec.stateCount, "Unexpected derived state count")
			require.Equal(tt.wantTriggerCount, spec.triggerCount, "Unexpected derived trigger count")
			require.Len(spec.transitions, int(tt.wantStateCount*tt.wantTriggerCount), "Unexpected transitions slice size")
			require.Len(spec.stateHooks, int(tt.wantStateCount), "Unexpected stateHooks slice size")
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
			builder := NewSpecBuilder[state, trigger, input]()
			var got *transitionBuilder[state, trigger, input]

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
			builder := NewSpecBuilder[state, trigger, input]()

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
			configure  func(*specBuilder[state, trigger, input], outputs)
			fsmTrigger trigger
		}
		want struct {
			panic         bool
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
				configure: func(b *specBuilder[state, trigger, input], o outputs) {
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
				configure: func(b *specBuilder[state, trigger, input], o outputs) {
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
				configure: func(b *specBuilder[state, trigger, input], o outputs) {
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
				configure: func(b *specBuilder[state, trigger, input], o outputs) {
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
				configure: func(b *specBuilder[state, trigger, input], o outputs) {
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
				configure: func(b *specBuilder[state, trigger, input], o outputs) {
					b.Transition().From(unlocked).On(lock).To(locked)
				},
				fsmTrigger: lock,
			},
			want: want{
				panic: false,
			},
		},
		{
			name: "transition action and guards called",
			given: given{
				configure: func(b *specBuilder[state, trigger, input], o outputs) {
					b.Transition().From(unlocked).On(lock).To(locked).
						WithAction("actionCalled is set", func(ctx context.Context, p input) error {
							*o.actionCalled = true
							return nil
						}).
						WithGuard("guardCalled is set", func(p input) error {
							*o.guardCalled = true
							return nil
						})
					b.Transition().From(locked).On(unlock).To(unlocked)
				},
				fsmTrigger: lock,
			},
			want: want{
				panic:        false,
				guardCalled:  true,
				actionCalled: true,
			},
		},
		{
			name: "state hooks called",
			given: given{
				configure: func(b *specBuilder[state, trigger, input], o outputs) {
					b.Transition().From(unlocked).On(lock).To(locked)
					b.Transition().From(locked).On(unlock).To(unlocked)
					b.State(unlocked).OnExit(func(ctx context.Context, p input) error {
						*o.onExitCalled = true
						return nil
					})
					b.State(locked).OnEntry(func(ctx context.Context, p input) error {
						*o.onEntryCalled = true
						return nil
					})
				},
				fsmTrigger: lock,
			},
			want: want{
				panic:         false,
				onEntryCalled: true,
				onExitCalled:  true,
			},
		},
		{
			name: "initial state with correct parent defined",
			given: given{
				configure: func(b *specBuilder[state, trigger, input], o outputs) {
					b.State(root).Initial(child)
					b.State(child).Parent(root)
					b.Transition().From(unlocked).On(lock).To(locked)
				},
				fsmTrigger: lock,
			},
			want: want{
				panic: false,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			/* ---------------------------------- Given --------------------------------- */
			require := require.New(t)
			builder := NewSpecBuilder[state, trigger, input]()

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
			res := tryUnarySupplier(func() *Spec[state, trigger, input] {
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

			// Build the FSM from the specification and fire the given trigger.
			fsm := New(got, unlocked)
			err := fsm.Fire(t.Context(), tt.given.fsmTrigger, input{})
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
	specBuilder := NewSpecBuilder[state, trigger, input]()
	specBuilder.Transition().From(unlocked).On(lock).To(locked)
	specBuilder.Transition().From(locked).On(unlock).To(unlocked)
	spec := specBuilder.Build()

	// Create a new FSM with an initial state.
	fsm := New(spec, unlocked)

	// Assert the initial state of the FSM.
	require.Equal(unlocked, fsm.State(), "Expected initial state to be unlocked")

	/* --------------------------------- When 1 --------------------------------- */
	// Fire the lock trigger and assert the state is now locked.
	err := fsm.Fire(t.Context(), lock, input{})

	/* --------------------------------- Then 1 --------------------------------- */
	require.NoError(err, "Unexpected error when firing trigger")
	require.Equal(locked, fsm.State(), "Expected state to be locked")

	/* --------------------------------- When 2 --------------------------------- */
	// Fire the unlock trigger and assert the state is now unlocked.
	err = fsm.Fire(t.Context(), unlock, input{})

	/* --------------------------------- Then 2 --------------------------------- */
	require.NoError(err, "Unexpected error when firing trigger")
	require.Equal(unlocked, fsm.State(), "Expected state to be unlocked")
}

func TestMachine_Fire_ReturnsErrors(t *testing.T) {
	/* ---------------------------------- Given --------------------------------- */
	require := require.New(t)

	// Create a new FSM specification.
	specBuilder := NewSpecBuilder[state, trigger, input]()
	specBuilder.Transition().From(unlocked).On(lock).To(locked).
		WithGuard("always errors", func(p input) error {
			return fmt.Errorf("guard error")
		})
	spec := specBuilder.Build()

	// Create a new FSM with an initial state.
	fsm := New(spec, unlocked)

	/* --------------------------------- When 1 --------------------------------- */
	// Fire the lock trigger and assert the error is ErrTransitionRejected and the state is still unlocked.
	err := fsm.Fire(t.Context(), lock, input{})

	/* --------------------------------- Then 1 --------------------------------- */
	require.Error(err, "Expected error when firing trigger")
	require.ErrorIs(err, ErrTransitionRejected, "Expected ErrTransitionRejected when guard fails")
	require.Equal(unlocked, fsm.State(), "Expected state to be unlocked")

	/* --------------------------------- When 2 --------------------------------- */
	// Fire the unlock trigger and assert the error is ErrNotFound and the state is still unlocked.
	err = fsm.Fire(t.Context(), unlock, input{})

	/* --------------------------------- Then 2 --------------------------------- */
	require.Error(err, "Expected error when firing trigger")
	require.ErrorIs(err, ErrNotFound, "Expected ErrNotFound when transition is not found")
	require.Equal(unlocked, fsm.State(), "Expected state to be unlocked")
}

func TestMachine_CanFire(t *testing.T) {
	/* ---------------------------------- Given --------------------------------- */
	require := require.New(t)

	// Create a new FSM specification.
	spec := NewSpecBuilder[state, trigger, input]()
	spec.Transition().From(unlocked).On(lock).To(locked)

	// Create a new FSM with an initial state.
	fsm := New(spec.Build(), unlocked)

	/* ---------------------------------- When ---------------------------------- */
	got := fsm.CanFire(t.Context(), unlock, input{})

	/* ---------------------------------- Then ---------------------------------- */
	require.False(got, "Expected CanFire to return false for undefined transition")

	/* --------------------------------- When 2 --------------------------------- */
	got = fsm.CanFire(t.Context(), lock, input{})

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
	spec := NewSpecBuilder[state, trigger, input]()
	spec.Transition().From(root).On(lock).To(grandchild).WithAction("rootActionCalled is set", func(ctx context.Context, p input) error {
		rootActionCalled = true
		return nil
	})
	spec.Transition().From(grandchild).On(unlock).To(child).WithAction("grandchildActionCalled is set", func(ctx context.Context, p input) error {
		grandchildActionCalled = true
		return nil
	})

	// Configure state hierarchy.
	spec.State(grandchild).Parent(child)
	spec.State(child).Parent(root)

	// Create a new FSM with an initial state.
	fsm := New(spec.Build(), grandchild)

	/* ---------------------------------- When ---------------------------------- */
	err := fsm.Fire(t.Context(), lock, input{})

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
	spec := NewSpecBuilder[state, trigger, input]()
	spec.Transition().From(root).On(unlock).To(grandchild).WithAction("rootActionCalled is set", func(ctx context.Context, p input) error {
		rootActionCalled = true
		return nil
	})
	spec.Transition().From(grandchild).On(unlock).To(child).WithAction("grandchildActionCalled is set", func(ctx context.Context, p input) error {
		grandchildActionCalled = true
		return nil
	})

	// Configure state hierarchy.
	spec.State(grandchild).Parent(child)
	spec.State(child).Parent(root)

	// Create a new FSM with an initial state.
	fsm := New(spec.Build(), grandchild)

	/* ---------------------------------- When ---------------------------------- */
	err := fsm.Fire(t.Context(), lock, input{})

	/* ---------------------------------- Then ---------------------------------- */
	require.Error(err, "Unexpected error when firing trigger")
	require.False(rootActionCalled, "Expected root action to be called")
	require.False(grandchildActionCalled, "Expected grandchild action to NOT be called")
}

func TestMachine_ActiveHierarchy(t *testing.T) {
	/* ---------------------------------- Given --------------------------------- */
	require := require.New(t)

	// Create a new FSM specification.
	spec := NewSpecBuilder[state, trigger, input]()

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
	specBuilder := NewSpecBuilder[state, trigger, input]()
	specBuilder.Transition().From(grandchild).On(lock).To(locked).WithAction("actionCalled is set", func(ctx context.Context, p input) error {
		actionCalled = true
		callOrder = append(callOrder, "action")
		return nil
	})

	// Configure state hierarchy.
	specBuilder.State(root).
		OnEntry(func(ctx context.Context, p input) error {
			rootOnEntryCalled = true
			callOrder = append(callOrder, "rootOnEntry")
			return nil
		}).
		OnExit(func(ctx context.Context, p input) error {
			rootOnExitCalled = true
			callOrder = append(callOrder, "rootOnExit")
			return nil
		})
	specBuilder.State(grandchild).
		Parent(child).
		OnEntry(func(ctx context.Context, p input) error {
			grandchildOnEntryCalled = true
			callOrder = append(callOrder, "grandchildOnEntry")
			return nil
		}).
		OnExit(func(ctx context.Context, p input) error {
			grandchildOnExitCalled = true
			callOrder = append(callOrder, "grandchildOnExit")
			return nil
		})
	specBuilder.State(child).
		Parent(root).
		OnEntry(func(ctx context.Context, p input) error {
			lcaOnEntryCalled = true
			callOrder = append(callOrder, "lcaOnEntry")
			return nil
		}).
		OnExit(func(ctx context.Context, p input) error {
			lcaOnExitCalled = true
			callOrder = append(callOrder, "lcaOnExit")
			return nil
		})
	specBuilder.State(locked).
		Parent(child).
		OnEntry(func(ctx context.Context, p input) error {
			lockedOnEntryCalled = true
			callOrder = append(callOrder, "lockedOnEntry")
			return nil
		}).
		OnExit(func(ctx context.Context, p input) error {
			lockedOnExitCalled = true
			callOrder = append(callOrder, "lockedOnExit")
			return nil
		})

	// Create a new FSM with an initial state.
	spec := specBuilder.Build()
	fsm := New(spec, grandchild)

	/* ---------------------------------- When ---------------------------------- */
	err := fsm.Fire(t.Context(), lock, input{})

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
	specBuilder := NewSpecBuilder[state, trigger, input]()
	specBuilder.Transition().From(unlocked).On(lock).To(locked).
		WithGuard("true", func(p input) error { return nil }).
		WithAction("runOp()", func(ctx context.Context, p input) error { return nil })
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
	specBuilder := NewSpecBuilder[state, trigger, input]()
	specBuilder.Transition().From(grandchild).On(lock).To(root).WithAction("actionCalled is set", func(ctx context.Context, p input) error {
		actionCalled = true
		callOrder = append(callOrder, "action")
		return nil
	})

	// Configure state hierarchy.
	specBuilder.State(root).
		Initial(locked).
		OnEntry(func(ctx context.Context, p input) error {
			rootOnEntryCalled = true
			callOrder = append(callOrder, "rootOnEntry")
			return nil
		}).
		OnExit(func(ctx context.Context, p input) error {
			rootOnExitCalled = true
			callOrder = append(callOrder, "rootOnExit")
			return nil
		})
	specBuilder.State(grandchild).
		Parent(child).
		OnEntry(func(ctx context.Context, p input) error {
			grandchildOnEntryCalled = true
			callOrder = append(callOrder, "grandchildOnEntry")
			return nil
		}).
		OnExit(func(ctx context.Context, p input) error {
			grandchildOnExitCalled = true
			callOrder = append(callOrder, "grandchildOnExit")
			return nil
		})

	specBuilder.State(child).
		Parent(root).
		OnEntry(func(ctx context.Context, p input) error {
			childOnEntryCalled = true
			callOrder = append(callOrder, "childOnEntry")
			return nil
		}).
		OnExit(func(ctx context.Context, p input) error {
			childOnExitCalled = true
			callOrder = append(callOrder, "childOnExit")
			return nil
		})
	specBuilder.State(locked).
		Parent(root).
		OnEntry(func(ctx context.Context, p input) error {
			lockedOnEntryCalled = true
			callOrder = append(callOrder, "lockedOnEntry")
			return nil
		}).
		OnExit(func(ctx context.Context, p input) error {
			lockedOnExitCalled = true
			callOrder = append(callOrder, "lockedOnExit")
			return nil
		})

	// Build the FSM specification.
	spec := specBuilder.Build()

	// Create a new FSM with an initial state.
	fsm := New(spec, grandchild)

	/* ---------------------------------- When ---------------------------------- */
	err := fsm.Fire(t.Context(), lock, input{})

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
