package hierarchicalstates

import (
	"context"
	"fmt"
	"testing"

	"github.com/tobbstr/fsm"
)

// Define hierarchical states
const (
	Root state = iota
	Parent
	ChildA
	GrandchildA
	ChildB
	GrandchildB
)

const (
	ChangeFocus trigger = iota
)

type state uint

func (s state) String() string {
	switch s {
	case Root:
		return "Root"
	case Parent:
		return "Parent"
	case ChildA:
		return "ChildA"
	case GrandchildA:
		return "GrandchildA"
	case ChildB:
		return "ChildB"
	case GrandchildB:
		return "GrandchildB"
	default:
		return fmt.Sprintf("state(%d)", s)
	}
}

type trigger uint

func (t trigger) String() string {
	switch t {
	case ChangeFocus:
		return "ChangeFocus"
	default:
		return fmt.Sprintf("trigger(%d)", t)
	}
}

type data struct{}

var handleError = func(err error) {
	// .. handle error ..
}

func TestHierarchicalStates(t *testing.T) {
	// States
	//
	//         Root
	//          │
	//       Parent
	//       /    \
	//   Child A   Child B
	//      │         │
	// Grandchild A  Grandchild B

	// Transitions
	//  - Grandchild A to Grandchild B

	// When making the transition, the FSM will first move up the hierarchy to the nearest common ancestor (Parent)
	// and then down to the target state (Grandchild B). On the way up the states' OnExit hooks will be called and on
	// the way down the OnEntry hooks will be called.

	builder := fsm.NewSpecBuilder[state, trigger, data](6, 1) // 6 states, 1 trigger

	// Hierarchy setup
	builder.State(Parent).Parent(Root)
	builder.State(ChildA).Parent(Parent)
	builder.State(GrandchildA).Parent(ChildA)
	builder.State(ChildB).Parent(Parent)
	builder.State(GrandchildB).Parent(ChildB)

	// Sole transition: GrandchildA -> GrandchildB
	builder.Transition().
		From(GrandchildA).
		On(ChangeFocus).
		To(GrandchildB).
		WithAction("printTransition()", func(ctx context.Context, data data) error {
			fmt.Println("Transitioning from GrandchildA to GrandchildB")
			return nil
		}).
		WithGuard("no-op", func(data data) error {
			fmt.Println("Guarding transition from GrandchildA to GrandchildB")
			return nil
		})

	// State hooks for all states
	for _, s := range []state{Root, Parent, ChildA, GrandchildA, ChildB, GrandchildB} {
		builder.State(s).
			OnEntry(func(ctx context.Context, data data) error {
				fmt.Printf("Entering %v state\n", s)
				return nil
			}).
			OnExit(func(ctx context.Context, data data) error {
				fmt.Printf("Exiting %v state\n", s)
				return nil
			})
	}

	spec := builder.Build()
	m := fsm.New(spec, GrandchildA) // Initial state is GrandchildA

	fmt.Printf("Current state: %v\n", m.State())
	err := m.Fire(context.Background(), ChangeFocus, data{})
	handleError(err)
	fmt.Printf("Current state: %v\n", m.State())
}
