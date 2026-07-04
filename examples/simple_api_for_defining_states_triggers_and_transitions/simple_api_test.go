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

// OrderInput represents the business data passed per Fire() call.
// This contains only the per-transition data, not infrastructure dependencies.
type OrderInput struct {
	OrderID    string
	CustomerID string
	Amount     float64
}

// Services represents infrastructure dependencies that are injected once
// at FSM specification creation time via closures.
type Services struct {
	// In a real application, this would include things like:
	// DB       *sql.DB
	// Logger   *log.Logger
	// EmailSvc EmailService
}

var handleError = func(err error) {
	// .. handle error ..
}

func TestSimpleAPI(t *testing.T) {
	// Infrastructure dependencies are defined once and captured via closure.
	// In a real application, you would initialize these with actual implementations.
	services := Services{
		// DB:       db,
		// Logger:   logger,
		// EmailSvc: emailSvc,
	}

	// Constructs a new FSM specification builder.
	// Note: Only 3 type parameters (state, trigger, OrderInput); the number of states and triggers is derived
	// automatically from the definitions below.
	builder := fsm.NewBuilder[state, trigger, OrderInput]()

	// Define transition from red to yellow.
	builder.
		From(red).                                                                   // The state from which this transition is valid.
		On(next).                                                                    // The trigger that causes the transition.
		To(yellow).                                                                  // The state to transition to.
		Do("transitionToYellow", func(ctx context.Context, input OrderInput) error { // The action to perform during the transition.
			// Services are captured from outer scope via Go closures.
			// This keeps the Fire() call site clean!
			_ = services // In a real app, you'd use: services.Logger.Printf(...), services.DB.Exec(...)
			fmt.Printf("Transitioning order %s from red to yellow\n", input.OrderID)
			return nil
		}).
		When("no-op", func(input OrderInput) bool { // The condition to check before allowing transition.
			// Services can also be accessed in conditions via closure.
			_ = services
			fmt.Printf("Protecting the transition for order %s from red to yellow\n", input.OrderID)
			return true
		})

	// Define transition from yellow to green.
	builder.
		From(yellow).
		On(next).
		To(green).
		Do("transitionToGreen", func(ctx context.Context, input OrderInput) error {
			_ = services
			fmt.Printf("Transitioning order %s from yellow to green\n", input.OrderID)
			return nil
		}).
		When("no-op", func(input OrderInput) bool {
			_ = services
			fmt.Printf("Protecting the transition for order %s from yellow to green\n", input.OrderID)
			return true
		})

	// Define state hooks for red.
	builder.
		From(red). // Configures the `red` state.
		WithHooks(fsm.StateHooks[OrderInput]{
			OnEntry: func(ctx context.Context, input OrderInput) error { // Called whenever transitioning into the `red` state.
				_ = services
				fmt.Printf("Order %s entering red state\n", input.OrderID)
				return nil
			},
			OnExit: func(ctx context.Context, input OrderInput) error { // Called whenever transitioning out of the `red` state.
				_ = services
				fmt.Printf("Order %s exiting red state\n", input.OrderID)
				return nil
			},
		})

	// Define state hooks for yellow.
	builder.
		From(yellow). // Configures the `yellow` state.
		WithHooks(fsm.StateHooks[OrderInput]{
			OnEntry: func(ctx context.Context, input OrderInput) error { // Called whenever transitioning into the `yellow` state.
				_ = services
				fmt.Printf("Order %s entering yellow state\n", input.OrderID)
				return nil
			},
			OnExit: func(ctx context.Context, input OrderInput) error { // Called whenever transitioning out of the `yellow` state.
				_ = services
				fmt.Printf("Order %s exiting yellow state\n", input.OrderID)
				return nil
			},
		})

	// Define state hooks for green.
	builder.
		From(green). // Configures the `green` state.
		WithHooks(fsm.StateHooks[OrderInput]{
			OnEntry: func(ctx context.Context, input OrderInput) error { // Called whenever transitioning into the `green` state.
				_ = services
				fmt.Printf("Order %s entering green state\n", input.OrderID)
				return nil
			},
			OnExit: func(ctx context.Context, input OrderInput) error { // Called whenever transitioning out of the `green` state.
				_ = services
				fmt.Printf("Order %s exiting green state\n", input.OrderID)
				return nil
			},
		})

	// Only initialize this once as it is read-only, meaning thread-safe.
	// Store it in a global variable. It is meant to called at startup of your application and may panic if the
	// specification is incomplete. For example if a transition definition has been started without completing it.
	spec := builder.Build()

	// Creates instance of the FSM with the initial state `red`.
	// This constructor should be called every time the FSM is needed. For example, in a request handler.
	m := fsm.New(spec, red)

	// Trigger events with clean Fire() calls!
	// Notice: We only pass business data (OrderInput), not services/infrastructure.
	currentState := m.State() // returns red
	fmt.Printf("Current state: %s\n", currentState)

	// Clean Fire() call - only business input, no DB/Logger/Services!
	err := m.Fire(context.Background(), next, OrderInput{
		OrderID:    "ORD-123",
		CustomerID: "CUST-456",
		Amount:     99.99,
	})
	handleError(err)

	currentState = m.State() // returns yellow
	fmt.Printf("Current state: %s\n", currentState)

	// Another clean Fire() call
	err = m.Fire(context.Background(), next, OrderInput{
		OrderID:    "ORD-123",
		CustomerID: "CUST-456",
		Amount:     99.99,
	})
	handleError(err)

	currentState = m.State() // returns green
	fmt.Printf("Current state: %s\n", currentState)

	// Returns a fsm.ErrNotFound error as there is not a defined transition from green to another state for the trigger (next).
	err = m.Fire(context.Background(), next, OrderInput{
		OrderID:    "ORD-123",
		CustomerID: "CUST-456",
		Amount:     99.99,
	})
	require.ErrorIs(t, err, fsm.ErrNotFound, "Expected ErrNotFound")
	handleError(err)
}
