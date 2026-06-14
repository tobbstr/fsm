
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

### Basic Example with Dependency Injection

The FSM library uses closure-based dependency injection to keep `Fire()` call sites clean. Infrastructure dependencies (database, services, logger) are captured via closures when defining the FSM specification, while business data is passed to each `Fire()` call as part of the stimuli (trigger + input) that attempt to stimulate the FSM to transition.

```go
import (
    "context"
    "database/sql"
    "errors"
    "fmt"
    "log"
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

// OrderInput contains per-transition business data (passed to each Fire call)
type OrderInput struct {
    OrderID       int
    CustomerID    string
    CurrentStatus string
}

// Services contains infrastructure dependencies (captured via closure)
type Services struct {
    DB     *sql.DB
    Logger *log.Logger
}

// Define the FSM specification. It should be stored in a global variable to avoid having to recreate it each time.
// This is safe as the spec is read-only and thread-safe.
func fsmSpec(services Services) *fsm.Spec[orderState, orderTrigger, OrderInput] {
    builder := fsm.NewSpecBuilder[orderState, orderTrigger, OrderInput]()

    builder.Transition().From(statePaid).On(triggerShip).To(stateShipped).
        // Guards are pure functions that check business rules (no side effects!)
        WithGuard("status == paid", func(input OrderInput) error {
            if input.CurrentStatus != "paid" {
                return fmt.Errorf("order must be paid before shipping")
            }
            return nil
        }).
        // Actions perform side effects using captured services
        WithAction("update status and notify", func(ctx context.Context, input OrderInput) error {
            // Services are captured from outer scope via closure
            services.Logger.Printf("Shipping order %d", input.OrderID)
            
            if err := updateStatus(ctx, services.DB, input.OrderID, "shipped"); err != nil {
                return fmt.Errorf("updating status: %w", err)
            }
            if err := notifyShipped(ctx, services.DB, input.CustomerID); err != nil {
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
    
    // Create FSM spec once with services (typically done at app startup)
    services := Services{DB: s.db, Logger: s.logger}
    spec := fsmSpec(services)
    
    m := fsm.New(spec, stateFromString(order.Status))
    
    // Clean Fire() call - trigger and input together form the stimuli for the transition
    // Only business data, no infrastructure dependencies!
    if err := m.Fire(ctx, triggerShip, OrderInput{
        OrderID:       orderID,
        CustomerID:    order.CustomerID,
        CurrentStatus: order.Status,
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
func notifyShipped(ctx context.Context, db *sql.DB, customerID string) error {
    // ... send notification ...
    return nil
}
```

### Dependency Injection Pattern

**Key Principles:**

1. **Stimuli for state transitions**
   - The **trigger** and **input** together form the stimuli that attempt to stimulate the FSM to move into another state
   - The trigger identifies the type of event (e.g., "pay", "ship", "cancel")
   - The input provides the context and data needed for the transition (e.g., order details, customer information)
   - Together, they determine which transition to take and provide the necessary information for guards and actions

2. **Separate business data (input) from infrastructure dependencies (services)**
   - **Input:** Per-transition business data passed to each `Fire()` call as part of the stimuli
   - **Services:** Infrastructure dependencies (DB, logger, external APIs) captured via closures when defining the FSM spec

3. **Guards must be pure functions (no side effects)**
   - Guards should only validate data in the input
   - All side effects (DB calls, logging, etc.) belong in Actions, not Guards

```go
// ✅ Good: Clean separation of concerns
type OrderInput struct {
    OrderID      int
    CustomerID   string
    Amount       float64
    IsValidOrder bool  // Computed/validated before calling Fire
}

type Services struct {
    DB       *sql.DB
    Logger   *log.Logger
    EmailSvc EmailService
}

func SetupFSM(services Services) *fsm.Spec[State, Trigger, OrderInput] {
    builder := fsm.NewSpecBuilder[State, Trigger, OrderInput]()
    
    builder.Transition().From(Pending).On(Confirm).To(Confirmed).
        // Guards are pure - only check input data
        WithGuard("order is valid", func(input OrderInput) error {
            if !input.IsValidOrder || input.Amount <= 0 {
                return fmt.Errorf("invalid order")
            }
            return nil
        }).
        // Actions use services for side effects
        WithAction("process order", func(ctx context.Context, input OrderInput) error {
            // Services captured via closure - clean and type-safe!
            services.Logger.Printf("Processing order %d", input.OrderID)
            return services.DB.Exec(ctx, "UPDATE orders SET status = 'confirmed' WHERE id = ?", input.OrderID)
        })
    
    return builder.Build()
}

// Clean call site with only business data!
machine.Fire(ctx, Confirm, OrderInput{
    OrderID: 123, 
    CustomerID: "CUST-456", 
    Amount: 99.99,
    IsValidOrder: true,
})
```

### Handling Multiple Conditional Transitions

If your application service needs to perform several conditional operations in sequence, and you want
to return as soon as any operation succeeds, use the sentinel error `fsm.ErrTransitionRejected`.
This error lets you handle rejected transitions gracefully. Your application service controls the
order in which transitions are attempted, while the FSM checks business rules and executes side effects.

```go
func (s *Service) RunOperationWithMultipleConditionalSteps(ctx context.Context, orderID int) error {
    // Gather necessary business data
    input := OrderInput{OrderID: orderID, /* ... */}
    
    m := fsm.New(fsmSpec(), stateA)
    
    err := m.Fire(ctx, triggerX, input)
    if err == nil {
        return nil // The transition was successful
    }
    if !errors.Is(err, fsm.ErrTransitionRejected) {
        return fmt.Errorf("attempting first transition: %w", err) // An actual error occurred
    }
    // Otherwise, the transition was rejected due to the guard function not passing

    // Try another transition
    err = m.Fire(ctx, triggerY, input)
    if err == nil {
        return nil // The transition was successful
    }
    if !errors.Is(err, fsm.ErrTransitionRejected) {
        return fmt.Errorf("attempting second transition: %w", err) // An actual error occurred
    }
    
    // Final attempt
    if err = m.Fire(ctx, triggerZ, input); err != nil {
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
var orderFSMSpec = buildOrderFSMSpec(services)

func buildOrderFSMSpec(services Services) *fsm.Spec[orderState, orderTrigger, OrderInput] {
    builder := fsm.NewSpecBuilder[orderState, orderTrigger, OrderInput]()
    // ... define transitions and states with services captured via closure ...
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

Guards are **pure functions** that implement business rules. They determine whether a transition is allowed based solely on the input data.

**Important Principles:**
- ✅ Guards should be **side-effect free** (no database calls, no API calls, no logging, no mutations)
- ✅ Guards should only validate data in the input
- ✅ Guards return `error`: return `nil` to allow the transition, or any error to reject it
- ❌ Guards should **NOT** access services or perform I/O operations

```go
// ✅ Good: Pure guard checking only input data
builder.Transition().From(statePaid).On(triggerShip).To(stateShipped).
    WithGuard("inventory available", func(input OrderInput) error {
        if input.InventoryCount < input.OrderQuantity {
            return fmt.Errorf("insufficient inventory")
        }
        return nil // Allow transition
    })

// ❌ Bad: Guard with side effects (database call)
builder.Transition().From(statePaid).On(triggerShip).To(stateShipped).
    WithGuard("inventory check", func(input OrderInput) error {
        // DON'T DO THIS! Guards should be pure functions
        count, err := services.DB.QueryInventory(input.ProductID)
        if err != nil || count < input.Quantity {
            return fmt.Errorf("insufficient inventory")
        }
        return nil
    })
```

**Why Pure Guards?**
- Predictable and testable
- No hidden side effects
- Can be safely called multiple times (e.g., in `CanFire()`)
- Clear separation: guards check rules, actions perform effects

### Actions

Actions perform side effects during transitions, such as database updates or external API calls.

```go
// Assuming services is captured from outer scope
builder.Transition().From(statePaid).On(triggerShip).To(stateShipped).
    WithAction("ship order", func(ctx context.Context, input OrderInput) error {
        // services.ShippingService is captured via closure
        return services.ShippingService.CreateShipment(ctx, input.OrderID)
    })
```

Actions are executed **after** guards pass and **before** the state changes.

## State Hooks

States can have `OnEntry` and `OnExit` hooks that run when entering or leaving a state.

```go
// Assuming services is captured from outer scope
builder.State(stateShipped).
    OnEntry(func(ctx context.Context, input OrderInput) error {
        // Called when entering the "shipped" state
        // services.Analytics is captured via closure
        return services.Analytics.TrackEvent(ctx, "order_shipped", input.OrderID)
    }).
    OnExit(func(ctx context.Context, input OrderInput) error {
        // Called when leaving the "shipped" state
        // services.Cache is captured via closure
        return services.Cache.Invalidate(ctx, input.OrderID)
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
builder := fsm.NewSpecBuilder[state, trigger, input]()

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
machine.Fire(ctx, triggerX, input) // Successfully transitions from parent to stateOther
```

### Initial Substates

When transitioning to a state with an initial substate, the FSM automatically enters that substate:

```go
builder.State(stateParent).Initial(stateDefaultChild)
builder.State(stateDefaultChild).Parent(stateParent)

builder.Transition().From(stateA).On(triggerX).To(stateParent)

machine := fsm.New(spec, stateA)
machine.Fire(ctx, triggerX, input)
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
if machine.CanFire(ctx, triggerShip, input) {
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
err := machine.Fire(ctx, trigger, input)
if errors.Is(err, fsm.ErrTransitionRejected) {
    // Guard rejected the transition - business rules not met
    fmt.Println("Transition not allowed at this time")
}
```

### ErrNotFound

Returned when no valid transition exists for the current state and trigger:

```go
err := machine.Fire(ctx, trigger, input)
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

- `Spec[S, T, Input]` - Thread-safe FSM specification
- `Machine[S, T, Input]` - FSM instance with current state
- `Guard[Input]` - Function type for transition guards: `func(input Input) error`
- `Action[Input]` - Function type for transition actions and state hooks: `func(ctx context.Context, input Input) error`

### Builder API

- `NewSpecBuilder[S, T, Input]()` - Create a new spec builder (the number of states and triggers is derived automatically at `Build()` time)
- `.Transition().From(S).On(T).To(S)` - Define a transition
- `.WithGuard(desc string, guard Guard[Input])` - Add a guard to a transition
- `.WithAction(desc string, action Action[Input])` - Add an action to a transition
- `.State(S)` - Configure a state
- `.OnEntry(action Action[Input])` - Add an entry hook to a state
- `.OnExit(action Action[Input])` - Add an exit hook to a state
- `.Parent(S)` - Set parent state for hierarchical FSMs
- `.Initial(S)` - Set initial substate for hierarchical FSMs
- `.Build()` - Build the FSM specification

### Machine API

- `New[S, T, Input](spec *Spec, initialState S)` - Create a new FSM instance
- `.Fire(ctx, trigger, input)` - Attempt a state transition. The trigger and input together form the stimuli that attempt to stimulate the FSM to move into another state.
- `.State()` - Get current state
- `.CanFire(ctx, trigger, input)` - Check if transition is possible. The trigger and input together form the stimuli that would attempt to stimulate the FSM.
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
