package simpleapifordefiningstatestriggersandtransitions

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tobbstr/fsm"
)

const (
	red state = iota
	yellow
	green
)

const (
	next trigger = iota
)

type state uint

func (s state) String() string {
	switch s {
	case red:
		return "red"
	case yellow:
		return "yellow"
	case green:
		return "green"
	default:
		return fmt.Sprintf("state(%d)", s)
	}
}

type trigger uint

func (t trigger) String() string {
	switch t {
	case next:
		return "next"
	default:
		return fmt.Sprintf("trigger(%d)", t)
	}
}

type anyTypeExample struct {
	whatever string
}

// data is a data structure that holds the arguments you want to pass to the state hooks and transition functions such
// as the Guard and Action. It can have as many fields as needed, and they can be of any type, e.g., pgxpool.Pool, your
// application services, database repositories etc.
type data struct {
	payload anyTypeExample // This can be any type, import
}

var handleError = func(err error) {
	// .. handle error ..
}

func TestSimpleAPI(t *testing.T) {
	// Constructs a new FSM specification builder.
	builder := fsm.NewSpecBuilder[state, trigger, data](3, 1) // 3 states, 1 trigger

	// Define transition from red to yellow.
	builder.Transition().
		From(red).                                                                    // The state from which this transition is valid.
		On(next).                                                                     // The trigger that causes the transition.
		To(yellow).                                                                   // The state to transition to.
		WithAction("transitionToYellow", func(ctx context.Context, data data) error { // The action to perform during the transition.
			fmt.Println("Transitioning from red to yellow")
			return nil
		}).
		WithGuard("no-op", func(data data) error { // The guard to check before allowing transition.
			fmt.Println("Protecting the transition from red to yellow")
			return nil // Returning an error will disallow the transition.
		})

	// Define transition from yellow to green.
	builder.Transition().
		From(yellow).
		On(next).
		To(green).
		WithAction("transitionToGreen", func(ctx context.Context, data data) error {
			fmt.Println("Transitioning from yellow to green")
			return nil
		}).
		WithGuard("no-op", func(data data) error {
			fmt.Println("Protecting the transition from yellow to green")
			return nil
		})

	// Define state hooks for red.
	builder.
		State(red).                                          // Configures the `red` state.
		OnEntry(func(ctx context.Context, data data) error { // Called whenever transitioning into the `red` state.
			fmt.Println("Entering red state")
			return nil
		}).
		OnExit(func(ctx context.Context, data data) error { // Called whenever transitioning out of the `red` state.
			fmt.Println("Exiting red state")
			return nil
		})

	// Define state hooks for yellow.
	builder.
		State(yellow).                                       // Configures the `yellow` state.
		OnEntry(func(ctx context.Context, data data) error { // Called whenever transitioning into the `yellow` state.
			fmt.Println("Entering yellow state")
			return nil
		}).
		OnExit(func(ctx context.Context, data data) error { // Called whenever transitioning out of the `yellow` state.
			fmt.Println("Exiting yellow state")
			return nil
		})

	// Define state hooks for green.
	builder.
		State(green).                                        // Configures the `green` state.
		OnEntry(func(ctx context.Context, data data) error { // Called whenever transitioning into the `green` state.
			fmt.Println("Entering green state")
			return nil
		}).
		OnExit(func(ctx context.Context, data data) error { // Called whenever transitioning out of the `green` state.
			fmt.Println("Exiting green state")
			return nil
		})

	// Only initialize this once as it is read-only, meaning thread-safe.
	// Store it in a global variable. It is meant to called at startup of your application and may panic if the
	// specification is incomplete. For example if a transition definition has been started without completing it.
	spec := builder.Build()

	// Creates instance of the FSM with the initial state `red`.
	// This constructor should be called every time the FSM is needed. For example, in a request handler.
	m := fsm.New(spec, red)

	// Trigger events
	state := m.State() // returns red
	fmt.Printf("Current state: %s\n", state)
	err := m.Fire(context.Background(), next, data{payload: anyTypeExample{whatever: "example"}})
	handleError(err)

	state = m.State() // returns yellow
	fmt.Printf("Current state: %s\n", state)
	err = m.Fire(context.Background(), next, data{payload: anyTypeExample{whatever: "example"}})
	handleError(err)

	state = m.State() // returns green
	fmt.Printf("Current state: %s\n", state)

	// Returns a fsm.ErrNotFound error as there is not a defined transition from green to another state for the trigger (next).
	err = m.Fire(context.Background(), next, data{payload: anyTypeExample{whatever: "example"}})
	require.ErrorIs(t, err, fsm.ErrNotFound, "Expected ErrNotFound")
	handleError(err)
}
