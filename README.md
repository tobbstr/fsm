
# FSM: Finite State Machine Library for Go

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

A lightweight, idiomatic, and extensible Finite State Machine (FSM) library for Go. Designed for clarity, testability, and ease of integration into your projects.

## Features

- [**Simple API** — define states, triggers, and transitions with ease](./examples/simple_api_for_defining_states_triggers_and_transitions/simple_api_test.go)
- [**Side effects made easy** — run actions automatically during state transitions](./examples/simple_api_for_defining_states_triggers_and_transitions/simple_api_test.go)
- [**Fine-grained control** — guard transitions with custom conditions](./examples/simple_api_for_defining_states_triggers_and_transitions/simple_api_test.go)
- [**Flexible states** — add your own OnEntry and OnExit hooks](./examples/simple_api_for_defining_states_triggers_and_transitions/simple_api_test.go)
- [**Hierarchical states** — scale from simple to complex with nested state logic](./examples/hierarchical_states/hierarchical_test.go)
    - Supports up to 9 levels of nested sub-states
- [**Blazing fast** — transitions run with zero allocations](./benchmark_fire_test.go).
    ```
    Example with:
      - Number of states: 26
      - Number of triggers: 26

    goos: darwin
    goarch: amd64
    pkg: github.com/tobbstr/fsm
    cpu: Intel(R) Core(TM) i7-9750H CPU @ 2.60GHz
    BenchmarkFire
    BenchmarkFire-12        77544386                13.75 ns/op            0 B/op          0 allocs/op
    PASS
    ok      github.com/tobbstr/fsm  1.424s
    ```
- **Type-safe by design** — powered by Go generics for maximum flexibility
- [**Automatic documentation** — generate Mermaid.js diagrams with ease](./fsm_test.go#715)
- **Reliable** — backed by comprehensive tests using Go’s standard tools
- **MIT licensed** — free for both open-source and commercial use

## Installation

```sh
go get github.com/tobbstr/fsm
```

## Usage

```go
import (
    "context"
    "database/sql"
    "errors"
    "github.com/tobbstr/fsm"
)

type orderState uint
const (
    stateCreated orderState = iota
    statePaid
    stateShipped
)

type orderTrigger uint
const (
    triggerPay orderTrigger = iota
    triggerShip
)

// All data needed for guards and actions is loaded outside the FSM and passed in here
type orderData struct {
    orderID int
    db *sql.DB
    currentStatus string // loaded from DB before calling Fire
}

// Define the FSM specification. It should be stored in a global variable to avoid having to recreate it each time.
// This is safe as the spec is read-only.
func fsmSpec() *fsm.Spec[orderState, orderTrigger, orderData] {
    builder := fsm.NewSpecBuilder[orderState, orderTrigger, orderData](3, 2)

    builder.Transition().From(statePaid).On(triggerShip).To(stateShipped).
        // Guards are pure functions and the implementations of business rules.
        WithGuard(func(d orderData) bool {
            return d.currentStatus == "paid"
        }).
        // Actions perform side effects.
        WithAction(func(ctx context.Context, d orderData) error {
            if err := updateStatus(ctx, d.db, d.orderID, "shipped"); err != nil {
                return err
            }
            return notifyShipped(ctx, d.db, d.orderID)
        })

    // ...other transitions...
    return builder.Build()
}

// Usage in your application service method:
func (s *OrderService) ShipOrder(ctx context.Context, orderID int) error {
    order, err := s.loadOrder(ctx, orderID)
    if err != nil {
        return err
    }
    m := fsm.New(fsmSpec(), stateFromString(order.Status))
    // Pass all required data to the FSM
    return m.Fire(ctx, triggerShip, orderData{
        orderID: orderID,
        db: s.db,
        currentStatus: order.Status,
    })
}

func stateFromString(status string) orderState {
    switch status {
    case "created":
        return stateCreated
    case "paid":
        return statePaid
    case "shipped":
        return stateShipped
    default:
        panic("unknown state")
    }
}

// Example side-effect functions
func updateStatus(ctx context.Context, db *sql.DB, orderID int, status string) error {
    // ... update order in DB ...
    return nil
}
func notifyShipped(ctx context.Context, db *sql.DB, orderID int) error {
    // ... send notification ...
    return nil
}
```

If your application service needs to perform several conditional operations in sequence, and you want
to return as soon as any operation succeeds, use the sentinel error `fsm.ErrTransitionRejected`.
This error lets you handle rejected transitions gracefully. Your application service controls the
order in which transitions are attempted, while the FSM checks business rules and executes side effects.

```go
func (s *Service) RunOperationWithMultipleConditionalSteps(ctx context.Context) error {
    // Gather necessary data from in-memory, the database or external services.
    // ...
    m := fsm.New(fsmSpec(), stateA)
    err := m.Fire(ctx, triggerX, data{...})
    if err == nil { // Return early if the transition was successful.
        return nil
    }
    if !errors.Is(err, fsm.ErrTransitionRejected) {
        return fmt.Errorf("add more context: %w", err) // an actual error occurred
    }
    // Otherwise, the transition was rejected due to the guard function not passing, i.e., the business rules were not met.

    // That means we can try another transition.
    err = m.Fire(ctx, triggerY, data{...})
    if err == nil { // Return early if the transition was successful.
        return nil
    }
    if !errors.Is(err, fsm.ErrTransitionRejected) {
        return fmt.Errorf("add more context: %w", err) // an actual error occurred
    }
    // The transition was rejected due to the guard function not passing, i.e., the business rules were not met.

    // If we have reached the final transition to attempt, then...
    if err = m.Fire(ctx, triggerZ, data{...}); err != nil {
        return fmt.Errorf("add more context: %w", err) // either the transition was rejected or an actual error occurred
    }
    return nil // the transition was successful
}

```

## API Reference

See [fsm.go](fsm.go) for full API documentation and comments.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

## Contributing

Contributions, issues, and feature requests are welcome! Feel free to open an issue or submit a pull request.

## Author

- [Tobias Strandberg](https://github.com/tobbstr)
