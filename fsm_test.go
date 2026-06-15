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

func TestNewBuilder(t *testing.T) {
	require := require.New(t)

	/* ---------------------------------- When ---------------------------------- */
	builder := NewBuilder[state, trigger, input]()

	/* ---------------------------------- Then ---------------------------------- */
	require.NotNil(builder, "Expected a non-nil builder")
	require.Empty(builder.branchDefs, "Expected no branch definitions yet")
	require.Empty(builder.stateBuilders, "Expected no state definitions yet")
}

// TestBuild_DerivesDimensions verifies that Build() sizes the specification to fit the highest state and trigger
// index referenced by the definitions, removing the need to declare the counts up front.
func TestBuild_DerivesDimensions(t *testing.T) {
	// Test Cases
	tests := []struct {
		name             string
		configure        func(*Builder[state, trigger, input])
		wantStateCount   uint
		wantTriggerCount uint
	}{
		{
			name:             "empty builder yields a minimal 1x1 specification",
			configure:        func(b *Builder[state, trigger, input]) {},
			wantStateCount:   1,
			wantTriggerCount: 1,
		},
		{
			name: "dimensions derived from transition states and triggers",
			configure: func(b *Builder[state, trigger, input]) {
				// Highest state referenced is locked(0)/unlocked(1) => stateCount 2.
				// Highest trigger referenced is lock(1) => triggerCount 2.
				b.From(unlocked).On(lock).To(locked)
			},
			wantStateCount:   2,
			wantTriggerCount: 2,
		},
		{
			name: "dimensions account for states referenced only in the hierarchy",
			configure: func(b *Builder[state, trigger, input]) {
				// grandchild(4) is referenced only via the hierarchy, so it must still be in range.
				b.From(unlocked).On(lock).To(locked)
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
			builder := NewBuilder[state, trigger, input]()
			tt.configure(builder)

			/* ---------------------------------- When ---------------------------------- */
			spec := builder.Build()

			/* ---------------------------------- Then ---------------------------------- */
			require.Equal(tt.wantStateCount, spec.stateCount, "Unexpected derived state count")
			require.Equal(tt.wantTriggerCount, spec.triggerCount, "Unexpected derived trigger count")
			require.Len(spec.slots, int(tt.wantStateCount*tt.wantTriggerCount), "Unexpected slots slice size")
			require.Len(spec.stateHooks, int(tt.wantStateCount), "Unexpected stateHooks slice size")
		})
	}
}

// TestBuild_ValidatesHierarchyDepth verifies that Build() accepts a hierarchy at the maximum supported depth and
// panics on one that exceeds it.
func TestBuild_ValidatesHierarchyDepth(t *testing.T) {
	buildChain := func(levels int) func() {
		return func() {
			b := NewBuilder[state, trigger, input]()
			for i := 1; i < levels; i++ {
				b.State(state(i)).Parent(state(i - 1))
			}
			b.Build()
		}
	}

	t.Run("allows a hierarchy at the maximum supported depth", func(t *testing.T) {
		require.NotPanics(t, buildChain(maxDepth))
	})

	t.Run("panics when a hierarchy exceeds the maximum supported depth", func(t *testing.T) {
		require.Panics(t, buildChain(maxDepth+1))
	})
}

func TestBuilder_State(t *testing.T) {
	// Test Cases
	tests := []struct {
		name  string
		given []state
	}{
		{
			name:  "adds a stateBuilder for each state defined",
			given: []state{locked, unlocked},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require := require.New(t)
			builder := NewBuilder[state, trigger, input]()

			for _, s := range tt.given {
				builder.State(s)
			}

			require.Len(tt.given, len(builder.stateBuilders), "Unexpected number of state builders created")
			for _, sb := range builder.stateBuilders {
				require.Equal(builder, sb.b, "State builder does not reference the correct spec builder")
			}
		})
	}
}

func TestBuild(t *testing.T) {
	// Test Types
	type (
		outputs struct {
			condCalled    *bool
			actionCalled  *bool
			onEntryCalled *bool
			onExitCalled  *bool
		}
		given struct {
			configure  func(*Builder[state, trigger, input], outputs)
			fsmTrigger trigger
		}
		want struct {
			panic         bool
			condCalled    bool
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
			name: "panics when incomplete transition is defined (On without To)",
			given: given{
				configure: func(b *Builder[state, trigger, input], o outputs) {
					b.From(unlocked).On(lock) // no To()
				},
			},
			want: want{panic: true},
		},
		{
			name: "panics when unconditional branch shadows later branches",
			given: given{
				configure: func(b *Builder[state, trigger, input], o outputs) {
					// unconditional To first, then another branch — must panic
					b.From(unlocked).On(lock).To(locked).To(unlocked).When("cond", func(input) bool { return true })
				},
			},
			want: want{panic: true},
		},
		{
			name: "panics when initial state without parent defined",
			given: given{
				configure: func(b *Builder[state, trigger, input], o outputs) {
					b.State(root).Initial(child)
				},
			},
			want: want{panic: true},
		},
		{
			name: "panics when initial state with wrong parent defined",
			given: given{
				configure: func(b *Builder[state, trigger, input], o outputs) {
					b.State(root).Initial(child)
					b.State(child).Parent(unlocked)
				},
			},
			want: want{panic: true},
		},
		{
			name: "single transition defined",
			given: given{
				configure: func(b *Builder[state, trigger, input], o outputs) {
					b.From(unlocked).On(lock).To(locked)
				},
				fsmTrigger: lock,
			},
			want: want{panic: false},
		},
		{
			name: "transition action and condition called",
			given: given{
				configure: func(b *Builder[state, trigger, input], o outputs) {
					b.From(unlocked).On(lock).To(locked).
						Do("actionCalled is set", func(ctx context.Context, p input) error {
							*o.actionCalled = true
							return nil
						}).
						When("condCalled is set", func(p input) bool {
							*o.condCalled = true
							return true
						})
					b.From(locked).On(unlock).To(unlocked)
				},
				fsmTrigger: lock,
			},
			want: want{
				panic:        false,
				condCalled:   true,
				actionCalled: true,
			},
		},
		{
			name: "state hooks called",
			given: given{
				configure: func(b *Builder[state, trigger, input], o outputs) {
					b.From(unlocked).On(lock).To(locked)
					b.From(locked).On(unlock).To(unlocked)
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
				configure: func(b *Builder[state, trigger, input], o outputs) {
					b.State(root).Initial(child)
					b.State(child).Parent(root)
					b.From(unlocked).On(lock).To(locked)
				},
				fsmTrigger: lock,
			},
			want: want{panic: false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require := require.New(t)
			builder := NewBuilder[state, trigger, input]()

			var (
				condCalled    bool
				actionCalled  bool
				onEntryCalled bool
				onExitCalled  bool
			)
			tt.given.configure(builder, outputs{
				condCalled:    &condCalled,
				actionCalled:  &actionCalled,
				onEntryCalled: &onEntryCalled,
				onExitCalled:  &onExitCalled,
			})

			res := tryUnarySupplier(func() *Spec[state, trigger, input] {
				return builder.Build()
			})

			if tt.want.panic {
				require.True(res.panicked, "Expected panic but did not get one")
				require.False(res.optional.valid, "Expected optional value to be unset")
				return
			}
			require.True(res.optional.valid, "Expected optional value to be set")
			got := res.optional.value

			fsm := New(got, unlocked)
			err := fsm.Fire(t.Context(), tt.given.fsmTrigger, input{})
			require.NoError(err, "Unexpected error when firing trigger")

			require.Equal(tt.want.condCalled, condCalled, "Unexpected value for condCalled")
			require.Equal(tt.want.actionCalled, actionCalled, "Unexpected value for actionCalled")
			require.Equal(tt.want.onEntryCalled, onEntryCalled, "Unexpected value for onEntryCalled")
			require.Equal(tt.want.onExitCalled, onExitCalled, "Unexpected value for onExitCalled")
		})
	}
}

func TestMachine_Fire(t *testing.T) {
	/* ---------------------------------- Given --------------------------------- */
	require := require.New(t)

	builder := NewBuilder[state, trigger, input]()
	builder.From(unlocked).On(lock).To(locked)
	builder.From(locked).On(unlock).To(unlocked)
	spec := builder.Build()

	fsm := New(spec, unlocked)
	require.Equal(unlocked, fsm.State(), "Expected initial state to be unlocked")

	/* --------------------------------- When 1 --------------------------------- */
	err := fsm.Fire(t.Context(), lock, input{})

	/* --------------------------------- Then 1 --------------------------------- */
	require.NoError(err, "Unexpected error when firing trigger")
	require.Equal(locked, fsm.State(), "Expected state to be locked")

	/* --------------------------------- When 2 --------------------------------- */
	err = fsm.Fire(t.Context(), unlock, input{})

	/* --------------------------------- Then 2 --------------------------------- */
	require.NoError(err, "Unexpected error when firing trigger")
	require.Equal(unlocked, fsm.State(), "Expected state to be unlocked")
}

func TestMachine_Fire_ReturnsErrors(t *testing.T) {
	/* ---------------------------------- Given --------------------------------- */
	require := require.New(t)

	builder := NewBuilder[state, trigger, input]()
	builder.From(unlocked).On(lock).To(locked).
		When("always false", func(p input) bool {
			return false
		})
	spec := builder.Build()

	fsm := New(spec, unlocked)

	/* --------------------------------- When 1 --------------------------------- */
	err := fsm.Fire(t.Context(), lock, input{})

	/* --------------------------------- Then 1 --------------------------------- */
	require.Error(err, "Expected error when firing trigger")
	require.ErrorIs(err, ErrTransitionRejected, "Expected ErrTransitionRejected when condition fails")
	require.Equal(unlocked, fsm.State(), "Expected state to be unlocked")

	/* --------------------------------- When 2 --------------------------------- */
	err = fsm.Fire(t.Context(), unlock, input{})

	/* --------------------------------- Then 2 --------------------------------- */
	require.Error(err, "Expected error when firing trigger")
	require.ErrorIs(err, ErrNotFound, "Expected ErrNotFound when transition is not found")
	require.Equal(unlocked, fsm.State(), "Expected state to be unlocked")
}

func TestMachine_CanFire(t *testing.T) {
	/* ---------------------------------- Given --------------------------------- */
	require := require.New(t)

	builder := NewBuilder[state, trigger, input]()
	builder.From(unlocked).On(lock).To(locked)

	fsm := New(builder.Build(), unlocked)

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

	rootActionCalled := false
	grandchildActionCalled := false

	builder := NewBuilder[state, trigger, input]()
	builder.From(root).On(lock).To(grandchild).Do("rootActionCalled is set", func(ctx context.Context, p input) error {
		rootActionCalled = true
		return nil
	})
	builder.From(grandchild).On(unlock).To(child).Do("grandchildActionCalled is set", func(ctx context.Context, p input) error {
		grandchildActionCalled = true
		return nil
	})

	builder.State(grandchild).Parent(child)
	builder.State(child).Parent(root)

	fsm := New(builder.Build(), grandchild)

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

	rootActionCalled := false
	grandchildActionCalled := false

	builder := NewBuilder[state, trigger, input]()
	builder.From(root).On(unlock).To(grandchild).Do("rootActionCalled is set", func(ctx context.Context, p input) error {
		rootActionCalled = true
		return nil
	})
	builder.From(grandchild).On(unlock).To(child).Do("grandchildActionCalled is set", func(ctx context.Context, p input) error {
		grandchildActionCalled = true
		return nil
	})

	builder.State(grandchild).Parent(child)
	builder.State(child).Parent(root)

	fsm := New(builder.Build(), grandchild)

	/* ---------------------------------- When ---------------------------------- */
	err := fsm.Fire(t.Context(), lock, input{})

	/* ---------------------------------- Then ---------------------------------- */
	require.Error(err, "Unexpected error when firing trigger")
	require.False(rootActionCalled, "Expected root action to NOT be called")
	require.False(grandchildActionCalled, "Expected grandchild action to NOT be called")
}

func TestMachine_ActiveHierarchy(t *testing.T) {
	/* ---------------------------------- Given --------------------------------- */
	require := require.New(t)

	builder := NewBuilder[state, trigger, input]()
	builder.State(grandchild).Parent(child)
	builder.State(child).Parent(root)

	fsm := New(builder.Build(), grandchild)

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

	builder := NewBuilder[state, trigger, input]()
	builder.From(grandchild).On(lock).To(locked).Do("actionCalled is set", func(ctx context.Context, p input) error {
		actionCalled = true
		callOrder = append(callOrder, "action")
		return nil
	})

	builder.State(root).
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
	builder.State(grandchild).
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
	builder.State(child).
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
	builder.State(locked).
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

	spec := builder.Build()
	fsm := New(spec, grandchild)

	/* ---------------------------------- When ---------------------------------- */
	err := fsm.Fire(t.Context(), lock, input{})

	/* ---------------------------------- Then ---------------------------------- */
	require.NoError(err, "Unexpected error when firing trigger")

	require.True(grandchildOnExitCalled, "Expected grandchild onExit to be called")
	require.False(grandchildOnEntryCalled, "Expected grandchild onEntry to NOT be called")
	require.False(lcaOnEntryCalled, "Expected LCA onEntry to NOT be called")
	require.False(lcaOnExitCalled, "Expected LCA onExit to NOT be called")
	require.False(rootOnEntryCalled, "Expected root onEntry to NOT be called")
	require.False(rootOnExitCalled, "Expected root onExit to NOT be called")

	require.True(actionCalled, "Expected root action to be called")

	require.True(lockedOnEntryCalled, "Expected locked onEntry to be called")
	require.False(lockedOnExitCalled, "Expected locked onExit to NOT be called")

	require.Equal([]string{"grandchildOnExit", "action", "lockedOnEntry"}, callOrder, "Unexpected call order")
	require.Equal(locked, fsm.State(), "Expected FSM to be in locked state")
}

func TestSpec_MermaidDiagram(t *testing.T) {
	/* ---------------------------------- Given --------------------------------- */
	require := require.New(t)

	builder := NewBuilder[state, trigger, input]()
	builder.From(unlocked).On(lock).To(locked).
		When("true", func(p input) bool { return true }).
		Do("runOp()", func(ctx context.Context, p input) error { return nil })
	builder.From(locked).On(unlock).To(unlocked)
	builder.From(root).On(lock).To(child)
	builder.From(child).On(unlock).To(grandchild)
	builder.From(grandchild).On(lock).To(root)

	builder.State(grandchild).Parent(child)
	builder.State(child).Parent(root)

	spec := builder.Build()

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

	builder := NewBuilder[state, trigger, input]()
	builder.From(grandchild).On(lock).To(root).Do("actionCalled is set", func(ctx context.Context, p input) error {
		actionCalled = true
		callOrder = append(callOrder, "action")
		return nil
	})

	builder.State(root).
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
	builder.State(grandchild).
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
	builder.State(child).
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
	builder.State(locked).
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

	spec := builder.Build()
	fsm := New(spec, grandchild)

	/* ---------------------------------- When ---------------------------------- */
	err := fsm.Fire(t.Context(), lock, input{})

	/* ---------------------------------- Then ---------------------------------- */
	require.NoError(err, "Unexpected error when firing trigger")

	require.True(grandchildOnExitCalled, "Expected grandchild onExit to be called")
	require.False(grandchildOnEntryCalled, "Expected grandchild onEntry to NOT be called")
	require.False(childOnEntryCalled, "Expected child onEntry to NOT be called")
	require.True(childOnExitCalled, "Expected child onExit to be called")
	require.False(rootOnEntryCalled, "Expected root onEntry NOT to be called")
	require.False(rootOnExitCalled, "Expected root onExit to NOT be called")

	require.True(actionCalled, "Expected root action to be called")

	require.True(lockedOnEntryCalled, "Expected locked onEntry to be called")
	require.False(lockedOnExitCalled, "Expected locked onExit to NOT be called")

	require.Equal([]string{"grandchildOnExit", "childOnExit", "action", "lockedOnEntry"}, callOrder, "Unexpected call order")
	require.Equal(locked, fsm.State(), "Expected FSM to be in locked state")
}

// TestMachine_Fire_MultipleBranches tests first-match-wins semantics with multiple guarded branches.
func TestMachine_Fire_MultipleBranches(t *testing.T) {
	type inp struct{ val int }

	const (
		sA state = iota
		sB
		sC
		sD
	)
	const tX trigger = 0

	t.Run("first matching branch wins", func(t *testing.T) {
		require := require.New(t)

		builder := NewBuilder[state, trigger, inp]()
		builder.From(sA).On(tX).
			To(sB).When("val==1", func(i inp) bool { return i.val == 1 }).
			To(sC).When("val==2", func(i inp) bool { return i.val == 2 }).
			Otherwise(sD)

		spec := builder.Build()

		fsm := New(spec, sA)
		require.NoError(fsm.Fire(t.Context(), tX, inp{val: 1}))
		require.Equal(sB, fsm.State())

		fsm = New(spec, sA)
		require.NoError(fsm.Fire(t.Context(), tX, inp{val: 2}))
		require.Equal(sC, fsm.State())

		fsm = New(spec, sA)
		require.NoError(fsm.Fire(t.Context(), tX, inp{val: 99}))
		require.Equal(sD, fsm.State())
	})

	t.Run("no branch matches returns ErrTransitionRejected", func(t *testing.T) {
		require := require.New(t)

		builder := NewBuilder[state, trigger, inp]()
		builder.From(sA).On(tX).
			To(sB).When("val==1", func(i inp) bool { return i.val == 1 }).
			To(sC).When("val==2", func(i inp) bool { return i.val == 2 })

		fsm := New(builder.Build(), sA)
		err := fsm.Fire(t.Context(), tX, inp{val: 99})
		require.ErrorIs(err, ErrTransitionRejected)
		require.Equal(sA, fsm.State(), "state must not change on rejection")
	})

	t.Run("bubbling: child branches all reject, parent branch matches", func(t *testing.T) {
		require := require.New(t)

		const (
			parentState state = root
			childState  state = child
			targetState state = locked
		)
		const trig trigger = 0

		builder := NewBuilder[state, trigger, inp]()
		builder.From(childState).On(trig).
			To(sB).When("never", func(i inp) bool { return false })
		builder.From(parentState).On(trig).
			To(targetState)
		builder.State(childState).Parent(parentState)

		fsm := New(builder.Build(), childState)
		err := fsm.Fire(t.Context(), trig, inp{})
		require.NoError(err)
		require.Equal(targetState, fsm.State())
	})

	t.Run("no slot at all returns ErrNotFound", func(t *testing.T) {
		require := require.New(t)

		builder := NewBuilder[state, trigger, inp]()
		builder.From(sA).On(tX).To(sB)

		fsm := New(builder.Build(), sB) // sB has no transition on tX
		err := fsm.Fire(t.Context(), tX, inp{})
		require.ErrorIs(err, ErrNotFound)
	})
}

// TestMachine_Explain tests the Explain introspection method.
func TestMachine_Explain(t *testing.T) {
	type inp struct{ val int }

	const (
		sA state = iota
		sB
		sC
		sD
		sParent
	)
	const tX trigger = 0

	t.Run("matched at current level", func(t *testing.T) {
		require := require.New(t)

		builder := NewBuilder[state, trigger, inp]()
		builder.From(sA).On(tX).
			To(sB).When("val==1", func(i inp) bool { return i.val == 1 }).
			Otherwise(sC)

		fsm := New(builder.Build(), sA)
		d := fsm.Explain(tX, inp{val: 1})

		require.True(d.Found)
		require.True(d.Matched)
		require.Equal(sB, d.Target)
		require.Equal(sA, d.ResolvedFrom)
		require.Len(d.Levels, 1)
		require.True(d.Levels[0].Matched)
		require.Equal(Matched, d.Levels[0].Branches[0].Outcome)
		require.Equal(Skipped, d.Levels[0].Branches[1].Outcome)
	})

	t.Run("matched at ancestor level (bubble-up)", func(t *testing.T) {
		require := require.New(t)

		builder := NewBuilder[state, trigger, inp]()
		// child: never matches
		builder.From(sA).On(tX).
			To(sB).When("never", func(i inp) bool { return false })
		// parent: always matches
		builder.From(sParent).On(tX).To(sC)
		builder.State(sA).Parent(sParent)

		fsm := New(builder.Build(), sA)
		d := fsm.Explain(tX, inp{})

		require.True(d.Found)
		require.True(d.Matched)
		require.Equal(sC, d.Target)
		require.Equal(sParent, d.ResolvedFrom)
		require.Len(d.Levels, 2)
		require.False(d.Levels[0].Matched)
		require.Equal(sA, d.Levels[0].State)
		require.True(d.Levels[1].Matched)
		require.Equal(sParent, d.Levels[1].State)
	})

	t.Run("no match anywhere — Found true but Matched false", func(t *testing.T) {
		require := require.New(t)

		builder := NewBuilder[state, trigger, inp]()
		builder.From(sA).On(tX).
			To(sB).When("never", func(i inp) bool { return false })

		fsm := New(builder.Build(), sA)
		d := fsm.Explain(tX, inp{})

		require.True(d.Found)
		require.False(d.Matched)
		require.Equal(sA, d.ResolvedFrom)
		require.Len(d.Levels, 1)
		require.False(d.Levels[0].Matched)
	})

	t.Run("no rule anywhere — Found false", func(t *testing.T) {
		require := require.New(t)

		builder := NewBuilder[state, trigger, inp]()
		builder.From(sB).On(tX).To(sC) // rule for sB, not sA

		fsm := New(builder.Build(), sA)
		d := fsm.Explain(tX, inp{})

		require.False(d.Found)
		require.Nil(d.Levels)
	})

	t.Run("non-hierarchical machine has len(Levels)==1", func(t *testing.T) {
		require := require.New(t)

		builder := NewBuilder[state, trigger, inp]()
		builder.From(sA).On(tX).To(sB)

		fsm := New(builder.Build(), sA)
		d := fsm.Explain(tX, inp{})

		require.True(d.Matched)
		require.Len(d.Levels, 1)
	})

	t.Run("Explain agrees with Fire", func(t *testing.T) {
		require := require.New(t)

		builder := NewBuilder[state, trigger, inp]()
		builder.From(sA).On(tX).
			To(sB).When("val==1", func(i inp) bool { return i.val == 1 }).
			Otherwise(sC)

		spec := builder.Build()

		for _, v := range []int{1, 99} {
			in := inp{val: v}
			m := New(spec, sA)
			d := m.Explain(tX, in)
			err := m.Fire(t.Context(), tX, in)
			if d.Matched {
				require.NoError(err, "Explain says matched, Fire should succeed")
				require.Equal(d.Target, m.State(), "Fire target must equal Explain.Target")
			} else {
				require.Error(err, "Explain says no match, Fire should fail")
			}
		}
	})
}

// TestMachine_CanFire_MatchesExplain verifies CanFire agrees with Explain.Matched.
func TestMachine_CanFire_MatchesExplain(t *testing.T) {
	type inp struct{ val int }
	const (
		sA state = iota
		sB
		sC
	)
	const tX trigger = 0

	builder := NewBuilder[state, trigger, inp]()
	builder.From(sA).On(tX).
		To(sB).When("val==1", func(i inp) bool { return i.val == 1 }).
		To(sC).When("val==2", func(i inp) bool { return i.val == 2 })

	spec := builder.Build()

	for _, v := range []int{1, 2, 99} {
		in := inp{val: v}
		m := New(spec, sA)
		d := m.Explain(tX, in)
		can := m.CanFire(t.Context(), tX, in)
		require.Equal(t, d.Matched, can, "CanFire and Explain.Matched must agree for val=%d", v)
	}
}

// TestSpec_MermaidDiagram_MultipleBranches verifies one edge per branch is emitted.
func TestSpec_MermaidDiagram_MultipleBranches(t *testing.T) {
	require := require.New(t)

	// Use the package-level state/trigger types which have String() methods.
	// locked=0, unlocked=1, root=2 — use these as sA, sB, sC.
	builder := NewBuilder[state, trigger, input]()
	builder.From(locked).On(lock).
		To(unlocked).When("cond1", func(input) bool { return true }).
		To(root).When("cond2", func(input) bool { return false })

	spec := builder.Build()
	diagram := spec.MermaidJSDiagram()

	require.Contains(diagram, "locked --> unlocked : lock [cond1]")
	require.Contains(diagram, "locked --> root : lock [cond2]")
}
