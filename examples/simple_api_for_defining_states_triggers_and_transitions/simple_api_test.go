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

// OrderPayload represents the business data passed per Fire() call.
// This contains only the per-transition data, not infrastructure dependencies.
type OrderPayload struct {
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
	// Note: Only 3 type parameters now (state, trigger, OrderPayload)
	builder := fsm.NewSpecBuilder[state, trigger, OrderPayload](3, 1) // 3 states, 1 trigger

	// Define transition from red to yellow.
	builder.Transition().
		From(red).     // The state from which this transition is valid.
		On(next).      // The trigger that causes the transition.
		To(yellow).    // The state to transition to.
		WithAction("transitionToYellow", func(ctx context.Context, payload OrderPayload) error { // The action to perform during the transition.
			// Services are captured from outer scope via Go closures.
			// This keeps the Fire() call site clean!
			_ = services // In a real app, you'd use: services.Logger.Printf(...), services.DB.Exec(...)
			fmt.Printf("Transitioning order %s from red to yellow\n", payload.OrderID)
			return nil
		}).
		WithGuard("no-op", func(payload OrderPayload) error { // The guard to check before allowing transition.
			// Services can also be accessed in guards via closure.
			_ = services
			fmt.Printf("Protecting the transition for order %s from red to yellow\n", payload.OrderID)
			return nil // Returning an error will disallow the transition.
		})

	// Define transition from yellow to green.
	builder.Transition().
		From(yellow).
		On(next).
		To(green).
		WithAction("transitionToGreen", func(ctx context.Context, payload OrderPayload) error {
			_ = services
			fmt.Printf("Transitioning order %s from yellow to green\n", payload.OrderID)
			return nil
		}).
		WithGuard("no-op", func(payload OrderPayload) error {
			_ = services
			fmt.Printf("Protecting the transition for order %s from yellow to green\n", payload.OrderID)
			return nil
		})

	// Define state hooks for red.
	builder.
		State(red). // Configures the `red` state.
		OnEntry(func(ctx context.Context, payload OrderPayload) error { // Called whenever transitioning into the `red` state.
			_ = services
			fmt.Printf("Order %s entering red state\n", payload.OrderID)
			return nil
		}).
		OnExit(func(ctx context.Context, payload OrderPayload) error { // Called whenever transitioning out of the `red` state.
			_ = services
			fmt.Printf("Order %s exiting red state\n", payload.OrderID)
			return nil
		})

	// Define state hooks for yellow.
	builder.
		State(yellow). // Configures the `yellow` state.
		OnEntry(func(ctx context.Context, payload OrderPayload) error { // Called whenever transitioning into the `yellow` state.
			_ = services
			fmt.Printf("Order %s entering yellow state\n", payload.OrderID)
			return nil
		}).
		OnExit(func(ctx context.Context, payload OrderPayload) error { // Called whenever transitioning out of the `yellow` state.
			_ = services
			fmt.Printf("Order %s exiting yellow state\n", payload.OrderID)
			return nil
		})

	// Define state hooks for green.
	builder.
		State(green). // Configures the `green` state.
		OnEntry(func(ctx context.Context, payload OrderPayload) error { // Called whenever transitioning into the `green` state.
			_ = services
			fmt.Printf("Order %s entering green state\n", payload.OrderID)
			return nil
		}).
		OnExit(func(ctx context.Context, payload OrderPayload) error { // Called whenever transitioning out of the `green` state.
			_ = services
			fmt.Printf("Order %s exiting green state\n", payload.OrderID)
			return nil
		})

	// Only initialize this once as it is read-only, meaning thread-safe.
	// Store it in a global variable. It is meant to called at startup of your application and may panic if the
	// specification is incomplete. For example if a transition definition has been started without completing it.
	spec := builder.Build()

	// Creates instance of the FSM with the initial state `red`.
	// This constructor should be called every time the FSM is needed. For example, in a request handler.
	m := fsm.New(spec, red)

	// Trigger events with clean Fire() calls!
	// Notice: We only pass business data (OrderPayload), not services/infrastructure.
	currentState := m.State() // returns red
	fmt.Printf("Current state: %s\n", currentState)
	
	// Clean Fire() call - only business payload, no DB/Logger/Services!
	err := m.Fire(context.Background(), next, OrderPayload{
		OrderID:    "ORD-123",
		CustomerID: "CUST-456",
		Amount:     99.99,
	})
	handleError(err)

	currentState = m.State() // returns yellow
	fmt.Printf("Current state: %s\n", currentState)
	
	// Another clean Fire() call
	err = m.Fire(context.Background(), next, OrderPayload{
		OrderID:    "ORD-123",
		CustomerID: "CUST-456",
		Amount:     99.99,
	})
	handleError(err)

	currentState = m.State() // returns green
	fmt.Printf("Current state: %s\n", currentState)

	// Returns a fsm.ErrNotFound error as there is not a defined transition from green to another state for the trigger (next).
	err = m.Fire(context.Background(), next, OrderPayload{
		OrderID:    "ORD-123",
		CustomerID: "CUST-456",
		Amount:     99.99,
	})
	require.ErrorIs(t, err, fsm.ErrNotFound, "Expected ErrNotFound")
	handleError(err)
}
