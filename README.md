
# FSM: Finite State Machine Library for Go

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

A lightweight, idiomatic, and extensible Finite State Machine (FSM) library for Go. Designed for clarity, testability, and ease of integration into your projects.

## Features

- [**Simple API** — define states, triggers, and transitions with ease](./examples/simple_api_for_defining_states_triggers_and_transitions/simple_api_test.go)
- [**Side effects made easy** — run actions automatically during state transitions](./examples/simple_api_for_defining_states_triggers_and_transitions/simple_api_test.go)
- [**Fine-grained control** — guard transitions with custom conditions](./examples/simple_api_for_defining_states_triggers_and_transitions/simple_api_test.go)
- [**Flexible states** — add your own OnEntry and OnExit hooks](./examples/simple_api_for_defining_states_triggers_and_transitions/simple_api_test.go)
- [**Hierarchical states** — scale from simple to complex with nested state logic](./examples/hierarchical_states/hierarchical_test.go)
    - Supports up to 10 levels of nested sub-states
    - Automatic trigger bubbling up the state hierarchy
    - Initial substates for automatic state entry
    - Least Common Ancestor (LCA) optimization for state transitions
- [**Blazing fast** — transitions run with zero allocations](./benchmark_fire_test.go)
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
- **Thread-safe specifications** — build once, use safely across goroutines
- **Automatic documentation** — generate Mermaid.js state diagrams from your FSM specification
- **Sentinel errors** — well-defined errors for transition rejection and not-found scenarios
- **Query methods** — check if transitions can fire or if the FSM is in specific states
- **Reliable** — backed by comprehensive tests using Go's standard tools
- **MIT licensed** — free for both open-source and commercial use

## When to Use This Library

This FSM library is ideal for scenarios where you need to:

### ✅ Good Use Cases

- **Model business workflows** — order processing, approval flows, document lifecycles
- **Enforce state constraints** — ensure operations only happen in valid states
- **Complex state logic** — multiple states with conditional transitions based on business rules
- **Hierarchical state modeling** — nested states like "Order > Paid > PartiallyShipped"
- **Separate concerns** — decouple state management from business logic
- **Audit and visualization** — generate diagrams showing all possible state transitions
- **Testable state machines** — guards and actions are pure/isolated functions easy to test
- **Thread-safe state definitions** — share FSM specs safely across goroutines
- **Event-driven systems** — respond to triggers with well-defined state transitions

### Examples of Good Fits

```go
// Order fulfillment: Created -> Paid -> Shipped -> Delivered -> Completed
// Document approval: Draft -> Submitted -> UnderReview -> Approved/Rejected
// User onboarding: Registered -> EmailVerified -> ProfileCompleted -> Active
// Payment processing: Initiated -> Authorized -> Captured -> Settled
// Job execution: Queued -> Running -> Completed/Failed/Cancelled
```

## When NOT to Use This Library

### ❌ Consider Alternatives When

- **Simple boolean flags suffice** — if you only have 2-3 states with trivial transitions, a simple `if/else` or boolean flags may be clearer
- **No state persistence needed** — for purely transient, in-memory state that never needs to be saved or reconstructed
- **Extreme performance requirements** — if even 13ns per transition is too slow (though this is already extremely fast)
- **State is implicit** — when the state is naturally derived from other data rather than being explicit
- **No clear state boundaries** — if your domain doesn't have well-defined discrete states
- **Overkill for the problem** — adding FSM complexity when the problem is fundamentally simple

### When Simple Code Is Better

```go
// Instead of FSM for this simple case:
if user.EmailVerified && user.ProfileComplete {
    user.Active = true
}

// Or for simple status tracking:
switch order.Status {
case "pending":
    // handle pending
case "completed":
    // handle completed
}
```

**Rule of thumb:** If you find yourself thinking "this FSM definition is more complex than the problem it solves," it's probably overkill. Use FSMs when state management is a core concern of your domain, not when it's incidental.

## Installation

```sh
go get github.com/tobbstr/fsm
```

## Defining States and Triggers

States and triggers should be defined as custom types with underlying type `uint`. This provides type safety and enables the FSM to use array-based lookups for optimal performance.

### Basic Pattern

```go
// Define custom types
type orderState uint
type orderTrigger uint

// Define states using iota for auto-incrementing values
const (
    stateCreated orderState = iota
    statePaid
    stateShipped
    stateDelivered
    stateCompleted
    stateCancelled
)

// Define triggers using iota
const (
    triggerPay orderTrigger = iota
    triggerShip
    triggerDeliver
    triggerComplete
    triggerCancel
)
```

### Making States and Triggers Print Nicely

Implement the `String()` method to make your states and triggers readable in logs, error messages, and Mermaid diagrams:

```go
func (s orderState) String() string {
    switch s {
    case stateCreated:
        return "Created"
    case statePaid:
        return "Paid"
    case stateShipped:
        return "Shipped"
    case stateDelivered:
        return "Delivered"
    case stateCompleted:
        return "Completed"
    case stateCancelled:
        return "Cancelled"
    default:
        return fmt.Sprintf("orderState(%d)", s)
    }
}

func (t orderTrigger) String() string {
    switch t {
    case triggerPay:
        return "Pay"
    case triggerShip:
        return "Ship"
    case triggerDeliver:
        return "Deliver"
    case triggerComplete:
        return "Complete"
    case triggerCancel:
        return "Cancel"
    default:
        return fmt.Sprintf("orderTrigger(%d)", t)
    }
}
```

### Benefits of String() Methods

With `String()` methods defined:

**Error messages are readable:**
```go
// Without String(): finding transition for trigger (2) and current state (1): not found
// With String():    finding transition for trigger (Ship) and current state (Paid): not found
```

**Logs are meaningful:**
```go
log.Printf("Order transitioned to: %v", machine.State()) // Prints: "Order transitioned to: Shipped"
```

**Mermaid diagrams are beautiful:**
```
stateDiagram-v2
    Created --> Paid : Pay
    Paid --> Shipped : Ship
    
// Instead of:
stateDiagram-v2
    0 --> 1 : 0
    1 --> 2 : 1
```

### Best Practices

1. **Use descriptive names** — `stateOrderPlaced` is better than `state1`
2. **Prefix consistently** — use `state` prefix for states, `trigger` for triggers
3. **Name states in past tense** — states represent outcomes of events: `stateCompleted`, `stateStarted`, `stateReadied`, `statePaid`, `stateShipped` (not `stateComplete`, `stateStart`, `statePay`)
4. **Name triggers as commands** — triggers are imperative actions: `triggerStart`, `triggerComplete`, `triggerPay`, `triggerShip` (not `triggerStarting`, `triggerCompletion`)
5. **Always implement String()** — even for internal states, it helps debugging
6. **Include default case** — handle unexpected values gracefully in String()
7. **Document transitions** — add comments showing valid state → trigger → state paths

**Naming Rationale:**
- States are facts about what *has happened* (past tense)
- Triggers are commands to *make something happen* (imperative verbs)

### Complete Example

```go
package orders

import "fmt"

// orderState represents the lifecycle state of an order
type orderState uint

const (
    stateCreated   orderState = iota // Order has been created
    statePaid                         // Payment received
    stateShipped                      // Order shipped to customer
    stateDelivered                    // Order delivered
    stateCompleted                    // Order completed successfully
    stateCancelled                    // Order cancelled
)

func (s orderState) String() string {
    switch s {
    case stateCreated:
        return "Created"
    case statePaid:
        return "Paid"
    case stateShipped:
        return "Shipped"
    case stateDelivered:
        return "Delivered"
    case stateCompleted:
        return "Completed"
    case stateCancelled:
        return "Cancelled"
    default:
        return fmt.Sprintf("orderState(%d)", s)
    }
}

// orderTrigger represents events that cause state transitions
type orderTrigger uint

const (
    triggerPay      orderTrigger = iota // Customer pays
    triggerShip                          // Warehouse ships order
    triggerDeliver                       // Carrier delivers order
    triggerComplete                      // Customer confirms completion
    triggerCancel                        // Order cancelled
)

func (t orderTrigger) String() string {
    switch t {
    case triggerPay:
        return "Pay"
    case triggerShip:
        return "Ship"
    case triggerDeliver:
        return "Deliver"
    case triggerComplete:
        return "Complete"
    case triggerCancel:
        return "Cancel"
    default:
        return fmt.Sprintf("orderTrigger(%d)", t)
    }
}
```

## Usage

### Basic Example

```go
import (
    "context"
    "database/sql"
    "errors"
    "fmt"
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
    orderID       int
    db            *sql.DB
    currentStatus string // loaded from DB before calling Fire
}

// Define the FSM specification. It should be stored in a global variable to avoid having to recreate it each time.
// This is safe as the spec is read-only and thread-safe.
func fsmSpec() *fsm.Spec[orderState, orderTrigger, orderData] {
    builder := fsm.NewSpecBuilder[orderState, orderTrigger, orderData](3, 2)

    builder.Transition().From(statePaid).On(triggerShip).To(stateShipped).
        // Guards are pure functions that implement business rules.
        // They return an error if the transition should be blocked.
        WithGuard("status == paid", func(d orderData) error {
            if d.currentStatus != "paid" {
                return fmt.Errorf("order must be paid before shipping")
            }
            return nil
        }).
        // Actions perform side effects.
        // The first parameter is a human-readable description used for documentation.
        WithAction("update status and notify", func(ctx context.Context, d orderData) error {
            if err := updateStatus(ctx, d.db, d.orderID, "shipped"); err != nil {
                return fmt.Errorf("updating status: %w", err)
            }
            if err := notifyShipped(ctx, d.db, d.orderID); err != nil {
                return fmt.Errorf("notifying customer: %w", err)
            }
            return nil
        })

    // ...other transitions...
    return builder.Build()
}

// Usage in your application service method:
func (s *OrderService) ShipOrder(ctx context.Context, orderID int) error {
    order, err := s.loadOrder(ctx, orderID)
    if err != nil {
        return fmt.Errorf("loading order: %w", err)
    }
    
    m := fsm.New(fsmSpec(), stateFromString(order.Status))
    
    // Pass all required data to the FSM
    if err := m.Fire(ctx, triggerShip, orderData{
        orderID:       orderID,
        db:            s.db,
        currentStatus: order.Status,
    }); err != nil {
        if errors.Is(err, fsm.ErrTransitionRejected) {
            return fmt.Errorf("cannot ship order in current state: %w", err)
        }
        return fmt.Errorf("shipping order: %w", err)
    }
    return nil
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

### Handling Multiple Conditional Transitions

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
    if err == nil {
        return nil // The transition was successful
    }
    if !errors.Is(err, fsm.ErrTransitionRejected) {
        return fmt.Errorf("attempting first transition: %w", err) // An actual error occurred
    }
    // Otherwise, the transition was rejected due to the guard function not passing

    // Try another transition
    err = m.Fire(ctx, triggerY, data{...})
    if err == nil {
        return nil // The transition was successful
    }
    if !errors.Is(err, fsm.ErrTransitionRejected) {
        return fmt.Errorf("attempting second transition: %w", err) // An actual error occurred
    }
    
    // Final attempt
    if err = m.Fire(ctx, triggerZ, data{...}); err != nil {
        return fmt.Errorf("attempting final transition: %w", err)
    }
    return nil
}
```

## Core Concepts

### FSM Specification (Spec)

The FSM specification is a **read-only, thread-safe** structure that defines all states, triggers, and transitions. 
Build it once at application startup and reuse it across goroutines.

```go
// Build once, typically in a package-level variable or during initialization
var orderFSMSpec = buildOrderFSMSpec()

func buildOrderFSMSpec() *fsm.Spec[orderState, orderTrigger, orderData] {
    builder := fsm.NewSpecBuilder[orderState, orderTrigger, orderData](3, 2)
    // ... define transitions and states ...
    return builder.Build()
}
```

### FSM Machine

Each FSM instance maintains its own current state. Create a new machine for each stateful entity:

```go
// Create a new FSM instance for each order
machine := fsm.New(orderFSMSpec, order.CurrentState)
```

### Guards

Guards are pure functions that implement business rules. They determine whether a transition is allowed.

```go
builder.Transition().From(statePaid).On(triggerShip).To(stateShipped).
    WithGuard("inventory available", func(d orderData) error {
        if d.inventoryCount < d.orderQuantity {
            return fmt.Errorf("insufficient inventory")
        }
        return nil // Allow transition
    })
```

**Important:** Guards return `error`. Return `nil` to allow the transition, or any error to reject it.

### Actions

Actions perform side effects during transitions, such as database updates or external API calls.

```go
builder.Transition().From(statePaid).On(triggerShip).To(stateShipped).
    WithAction("ship order", func(ctx context.Context, d orderData) error {
        return d.shippingService.CreateShipment(ctx, d.orderID)
    })
```

Actions are executed **after** guards pass and **before** the state changes.

## State Hooks

States can have `OnEntry` and `OnExit` hooks that run when entering or leaving a state.

```go
builder.State(stateShipped).
    OnEntry(func(ctx context.Context, d orderData) error {
        // Called when entering the "shipped" state
        return d.analytics.TrackEvent(ctx, "order_shipped", d.orderID)
    }).
    OnExit(func(ctx context.Context, d orderData) error {
        // Called when leaving the "shipped" state
        return d.cache.Invalidate(ctx, d.orderID)
    })
```

**Execution Order During Transition:**
1. Find transition (with automatic trigger bubbling up the hierarchy if needed)
2. Execute transition guard (if defined) - transition is rejected if guard returns error
3. Exit current state and ancestors up to LCA (OnExit hooks)
4. Execute transition action (if defined)
5. Enter target state and ancestors from LCA down (OnEntry hooks)
6. If target state has an initial substate, enter it (OnEntry hook)

## Hierarchical States

Hierarchical states allow you to model complex state machines with parent-child relationships.

### Defining Hierarchies

```go
builder := fsm.NewSpecBuilder[state, trigger, data](6, 2)

// Define parent-child relationships
builder.State(stateChild).Parent(stateParent)
builder.State(stateGrandchild).Parent(stateChild)
```

### Trigger Bubbling

When a trigger is fired from a child state, the FSM automatically searches up the hierarchy for a valid transition:

```go
// Define transition only on parent
builder.Transition().From(stateParent).On(triggerX).To(stateOther)

// Create FSM starting in grandchild
machine := fsm.New(spec, stateGrandchild)

// Trigger bubbles up: grandchild -> child -> parent (transition found!)
machine.Fire(ctx, triggerX, data) // Successfully transitions from parent to stateOther
```

### Initial Substates

When transitioning to a state with an initial substate, the FSM automatically enters that substate:

```go
builder.State(stateParent).Initial(stateDefaultChild)
builder.State(stateDefaultChild).Parent(stateParent)

builder.Transition().From(stateA).On(triggerX).To(stateParent)

machine := fsm.New(spec, stateA)
machine.Fire(ctx, triggerX, data)
// Machine is now in stateDefaultChild (not stateParent)
```

### Least Common Ancestor (LCA) Optimization

When transitioning between states in different branches of the hierarchy, the FSM intelligently:
1. Exits states up to the common ancestor
2. Enters states down to the target
3. Skips OnExit/OnEntry for the common ancestor

```
        Root (LCA - hooks NOT called)
       /    \
    Child A   Child B
      |          |
 Grandchild A  Grandchild B

Transition: Grandchild A -> Grandchild B
- Exit: Grandchild A, Child A
- Action: Transition action
- Enter: Child B, Grandchild B
```

## Query Methods

### State()

Returns the current state of the FSM:

```go
currentState := machine.State()
fmt.Printf("Current state: %v\n", currentState)
```

### CanFire()

Checks if a transition can be made without actually performing it:

```go
if machine.CanFire(ctx, triggerShip, data) {
    fmt.Println("Can ship the order")
} else {
    fmt.Println("Cannot ship the order yet")
}
```

This is useful for UI state management or conditional logic.

### IsIn()

Checks if the FSM is currently in a specific state (including hierarchical checks):

```go
if machine.IsIn(stateParent) {
    // Returns true if current state is stateParent OR any of its descendants
    fmt.Println("FSM is in parent state or one of its children")
}
```

### ActiveHierarchy()

Returns the complete hierarchy from the current state to the root:

```go
hierarchy := machine.ActiveHierarchy()
// Returns: [stateGrandchild, stateChild, stateParent, stateRoot]
for _, state := range hierarchy {
    fmt.Printf("Active state: %v\n", state)
}
```

## Sentinel Errors

The library provides two sentinel errors for common scenarios:

### ErrTransitionRejected

Returned when a guard function rejects a transition:

```go
err := machine.Fire(ctx, trigger, data)
if errors.Is(err, fsm.ErrTransitionRejected) {
    // Guard rejected the transition - business rules not met
    fmt.Println("Transition not allowed at this time")
}
```

### ErrNotFound

Returned when no valid transition exists for the current state and trigger:

```go
err := machine.Fire(ctx, trigger, data)
if errors.Is(err, fsm.ErrNotFound) {
    // No transition defined for this state-trigger combination
    fmt.Println("Invalid operation for current state")
}
```

## Mermaid Diagram Generation

Generate Mermaid.js state diagrams from your FSM specification for documentation:

```go
spec := builder.Build()
diagram := spec.MermaidJSDiagram()
fmt.Println(diagram)
```

**Output:**
```
stateDiagram-v2
    stateCreated --> statePaid : triggerPay [payment valid] / process payment
    statePaid --> stateShipped : triggerShip [inventory available] / ship order
```

The diagram includes:
- All transitions with their triggers
- Guard descriptions (in square brackets)
- Action descriptions (after forward slash)

You can use this in your documentation, wikis, or any tool that supports Mermaid.js.

## API Reference

See [fsm.go](fsm.go) for full API documentation and comments.

### Main Types

- `Spec[S, T, D]` - Thread-safe FSM specification
- `Machine[S, T, D]` - FSM instance with current state
- `Guard[D]` - Function type for transition guards: `func(data D) error`
- `Action[D]` - Function type for transition actions and state hooks: `func(ctx context.Context, data D) error`

### Builder API

- `NewSpecBuilder[S, T, D](numStates, numTriggers uint)` - Create a new spec builder
- `.Transition().From(S).On(T).To(S)` - Define a transition
- `.WithGuard(desc string, guard Guard[D])` - Add a guard to a transition
- `.WithAction(desc string, action Action[D])` - Add an action to a transition
- `.State(S)` - Configure a state
- `.OnEntry(action Action[D])` - Add an entry hook to a state
- `.OnExit(action Action[D])` - Add an exit hook to a state
- `.Parent(S)` - Set parent state for hierarchical FSMs
- `.Initial(S)` - Set initial substate for hierarchical FSMs
- `.Build()` - Build the FSM specification

### Machine API

- `New[S, T, D](spec *Spec, initialState S)` - Create a new FSM instance
- `.Fire(ctx, trigger, data)` - Attempt a state transition
- `.State()` - Get current state
- `.CanFire(ctx, trigger, data)` - Check if transition is possible
- `.IsIn(state)` - Check if FSM is in state (including hierarchy)
- `.ActiveHierarchy()` - Get active state hierarchy

### Spec API

- `.MermaidJSDiagram()` - Generate Mermaid.js diagram

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

## Contributing

Contributions, issues, and feature requests are welcome! Feel free to open an issue or submit a pull request.

## Author

- [Tobias Strandberg](https://github.com/tobbstr)
