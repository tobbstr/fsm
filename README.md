
# FSM: Finite State Machine Library for Go

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

A lightweight, idiomatic, and extensible Finite State Machine (FSM) library for Go. Designed for clarity, testability, and ease of integration into your projects.

## Features

- [**Simple API** — define states, triggers, and transitions with ease](./examples/simple_api_for_defining_states_triggers_and_transitions/simple_api_test.go)
- [**Side effects made easy** — run actions automatically during state transitions](./examples/simple_api_for_defining_states_triggers_and_transitions/simple_api_test.go)
- [**Fine-grained control** — guard transitions with boolean conditions](./examples/simple_api_for_defining_states_triggers_and_transitions/simple_api_test.go)
- **Multiple guarded branches** — define several candidate transitions per `(state, trigger)` with first-match-wins semantics and an optional unconditional `Otherwise` fallback
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
    goarch: arm64
    pkg: github.com/tobbstr/fsm
    cpu: Apple M5 Pro
    BenchmarkFire-18                   239354520    4.818 ns/op    0 B/op    0 allocs/op
    BenchmarkFire_SingleConditional-18 226444806    5.224 ns/op    0 B/op    0 allocs/op
    BenchmarkFire_Branching-18         207989106    5.749 ns/op    0 B/op    0 allocs/op
    PASS
    ```
- **Introspection** — `Explain()` returns a full decision trace showing which branch matched and why, including hierarchy bubble-up
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
- **Testable state machines** — conditions and actions are pure/isolated functions easy to test
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
- **Extreme performance requirements** — if even ~5ns per transition is too slow (though this is already extremely fast)
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

// Define the FSM specification. Store in a global variable — it is read-only and thread-safe.
func fsmSpec(services Services) *fsm.Spec[orderState, orderTrigger, OrderInput] {
    builder := fsm.NewBuilder[orderState, orderTrigger, OrderInput]()

    builder.From(statePaid).On(triggerShip).To(stateShipped).
        // Conditions are pure boolean functions that check business rules (no side effects!)
        When("status == paid", func(input OrderInput) bool {
            return input.CurrentStatus == "paid"
        }).
        // Actions perform side effects using captured services
        Do("update status and notify", func(ctx context.Context, input OrderInput) error {
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

    services := Services{DB: s.db, Logger: s.logger}
    spec := fsmSpec(services)

    m := fsm.New(spec, stateFromString(order.Status))

    // Clean Fire() call — trigger and input together form the stimuli for the transition.
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
```

### Dependency Injection Pattern

**Key Principles:**

1. **Stimuli for state transitions**
   - The **trigger** and **input** together form the stimuli that attempt to stimulate the FSM to move into another state
   - The trigger identifies the type of event (e.g., "pay", "ship", "cancel")
   - The input provides the context and data needed for the transition (e.g., order details, customer information)

2. **Separate business data (input) from infrastructure dependencies (services)**
   - **Input:** Per-transition business data passed to each `Fire()` call as part of the stimuli
   - **Services:** Infrastructure dependencies (DB, logger, external APIs) captured via closures when defining the FSM spec

3. **Conditions must be pure functions (no side effects)**
   - Conditions should only inspect data in the input and return `true` or `false`
   - All side effects (DB calls, logging, etc.) belong in actions, not conditions

```go
// ✅ Good: Clean separation of concerns
type OrderInput struct {
    OrderID      int
    CustomerID   string
    Amount       float64
    IsValidOrder bool  // Computed/validated before calling Fire
}

func SetupFSM(services Services) *fsm.Spec[State, Trigger, OrderInput] {
    builder := fsm.NewBuilder[State, Trigger, OrderInput]()

    builder.From(Pending).On(Confirm).To(Confirmed).
        // Conditions are pure — only inspect input data
        When("order is valid", func(input OrderInput) bool {
            return input.IsValidOrder && input.Amount > 0
        }).
        // Actions use services for side effects
        Do("process order", func(ctx context.Context, input OrderInput) error {
            // Services captured via closure — clean and type-safe!
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

## Core Concepts

### FSM Specification (Spec)

The FSM specification is a **read-only, thread-safe** structure that defines all states, triggers, and transitions.
Build it once at application startup and reuse it across goroutines.

```go
// Build once, typically in a package-level variable or during initialization
var orderFSMSpec = buildOrderFSMSpec(services)

func buildOrderFSMSpec(services Services) *fsm.Spec[orderState, orderTrigger, OrderInput] {
    builder := fsm.NewBuilder[orderState, orderTrigger, OrderInput]()
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

### Conditions

Conditions are **pure boolean functions** that implement business rules. They determine whether a branch is taken based solely on the input data.

**Important Principles:**
- ✅ Conditions should be **side-effect free** (no database calls, no API calls, no logging, no mutations)
- ✅ Conditions should only inspect data in the input
- ✅ Conditions return `bool`: `true` to allow the branch, `false` to skip it
- ❌ Conditions should **NOT** access services or perform I/O operations

```go
// ✅ Good: Pure condition checking only input data
builder.From(statePaid).On(triggerShip).To(stateShipped).
    When("inventory available", func(input OrderInput) bool {
        return input.InventoryCount >= input.OrderQuantity
    })

// ❌ Bad: Condition with side effects (database call)
builder.From(statePaid).On(triggerShip).To(stateShipped).
    When("inventory check", func(input OrderInput) bool {
        // DON'T DO THIS! Conditions should be pure functions
        count, _ := services.DB.QueryInventory(input.ProductID)
        return count >= input.Quantity
    })
```

**Why Pure Conditions?**
- Predictable and testable
- No hidden side effects
- Can be safely called multiple times (e.g., in `CanFire()` and `Explain()`)
- Clear separation: conditions check rules, actions perform effects

### Actions

Actions perform side effects during transitions, such as database updates or external API calls.

```go
// Assuming services is captured from outer scope
builder.From(statePaid).On(triggerShip).To(stateShipped).
    Do("ship order", func(ctx context.Context, input OrderInput) error {
        // services.ShippingService is captured via closure
        return services.ShippingService.CreateShipment(ctx, input.OrderID)
    })
```

Actions are executed **after** the winning branch is selected and **before** the state changes.

### Returning Data from Actions

Action functions return only `error`, so to surface data created inside an action (e.g. a newly created resource ID), write it back via a pointer field in the input struct:

```go
type OrderInput struct {
    OrderID    int
    CustomerID string
    Result     *ShipResult // populated by the action; nil if not needed
}

type ShipResult struct {
    ShipmentID string
}

// In the FSM spec:
builder.From(statePaid).On(triggerShip).To(stateShipped).
    Do("create shipment", func(ctx context.Context, input OrderInput) error {
        id, err := services.ShippingService.CreateShipment(ctx, input.OrderID)
        if err != nil {
            return err
        }
        input.Result.ShipmentID = id // write back via pointer
        return nil
    })

// Call site:
result := &ShipResult{}
err := m.Fire(ctx, triggerShip, OrderInput{
    OrderID:    123,
    CustomerID: "CUST-456",
    Result:     result,
})
// result.ShipmentID is now populated
```

This keeps the FSM spec a package-level variable (no rebuild per call) while letting action output flow back to the caller cleanly.

## Branching: Multiple Guarded Transitions

A single `(from, trigger)` pair can have multiple candidate branches, evaluated in definition order with **first-match-wins** semantics. This replaces the need to try multiple triggers in sequence just to express conditional routing.

### Syntax

Chain `.To(target).When(desc, cond)` for each guarded branch. Use `.Otherwise(target)` as the final unconditional fallback.

```go
builder.From(statePending).On(triggerWithdraw).
    To(stateCompleted).When("balance >= amount", func(in Input) bool {
        return in.Balance >= in.Amount
    }).
    To(stateOverdraft).When("overdraftAllowed", func(in Input) bool {
        return in.OverdraftAllowed
    }).
    Otherwise(stateRejected) // unconditional fallback — must be last
```

### Semantics

- Branches are evaluated **in definition order**; the first branch whose condition returns `true` wins.
- An `Otherwise` branch (or any branch with no `When`) is unconditional and always matches — it must be the **last** branch. Placing an unconditional branch before a later one panics at `Build()` time.
- If no branch matches and there is no `Otherwise`, `Fire` returns `ErrTransitionRejected`. The error message lists all the tried condition descriptions so the cause is immediately clear.
- A transition with no `When` at all is equivalent to a single unconditional branch (the original single-branch behavior, preserved for backwards compatibility and the zero-alloc hot path).

### Mermaid Diagram

Each branch is emitted as a separate edge, so the diagram faithfully reflects all conditional routes:

```
stateDiagram-v2
    Pending --> Completed : Withdraw [balance >= amount] / deduct balance
    Pending --> Overdraft : Withdraw [overdraftAllowed]
    Pending --> Rejected : Withdraw
```

### Rejection Error Message

When no branch matches, the error message includes every condition description that was tried, grouped by hierarchy level:

```
transition rejected for trigger (Withdraw) from state (Pending): no branch matched
  [Pending: "balance >= amount", "overdraftAllowed"; Account: "flagged"]
```

## Introspection with Explain

`Explain` reports a full **multi-level decision trace** for what `Fire` would do — without actually firing. It is the recommended tool for debugging, logging, and building diagnostic UIs.

```go
decision := machine.Explain(triggerWithdraw, input)

fmt.Println(decision.Found)        // true if any level had a rule for this trigger
fmt.Println(decision.Matched)      // true if a branch was selected
fmt.Println(decision.Target)       // the state that would be entered (valid if Matched)
fmt.Println(decision.ResolvedFrom) // the hierarchy level whose branch won

for _, level := range decision.Levels {
    fmt.Printf("Level %v (matched=%v):\n", level.State, level.Matched)
    for _, branch := range level.Branches {
        fmt.Printf("  -> %v [%s]: %v\n", branch.Target, branch.Condition, branch.Outcome)
    }
}
```

### Decision Types

```go
type Outcome uint8
const (
    NotMatched Outcome = iota // condition returned false
    Matched                   // this was the winning branch
    Skipped                   // a later branch that was never evaluated (first-match-wins)
)

type BranchVerdict[S ~uint] struct {
    Target    S       // the branch's target state
    Condition string  // the When description; "" for unconditional/Otherwise
    Outcome   Outcome
}

type LevelVerdict[S ~uint] struct {
    State    S
    Matched  bool
    Branches []BranchVerdict[S]
}

type Decision[S ~uint] struct {
    Found        bool              // any level had a rule for (state, trigger)?
    Matched      bool              // did a branch match?
    Target       S                 // state that would be entered (valid iff Matched)
    ResolvedFrom S                 // level whose branch won
    Levels       []LevelVerdict[S] // deepest-first: current state, then ancestors
}
```

### Worked Example

Current state `Pending` (child of `Account`), `Explain(Withdraw)`, account *is* flagged but balance is short and no overdraft allowed:

```
Found: true, Matched: true, Target: Frozen, ResolvedFrom: Account
Levels: [
  { State: Pending, Matched: false,
    Branches: [ {Completed, "balance >= amount", NotMatched},
                {Overdraft, "overdraftAllowed",  NotMatched} ] },
  { State: Account, Matched: true,
    Branches: [ {Frozen, "flagged", Matched} ] },
]
```

The bubble-up to `Account` is visible in `Levels` — you can see exactly why `Pending`'s branches were skipped.

> `Explain` allocates. It is never called by `Fire` or `CanFire`, so the zero-alloc hot path is unaffected.

## State Hooks

States can have `OnEntry` and `OnExit` hooks that run when entering or leaving a state.

```go
// Assuming services is captured from outer scope
builder.From(stateShipped).
    WithHooks(fsm.StateHooks[OrderInput]{
        OnEntry: func(ctx context.Context, input OrderInput) error {
            // Called when entering the "shipped" state
            return services.Analytics.TrackEvent(ctx, "order_shipped", input.OrderID)
        },
        OnExit: func(ctx context.Context, input OrderInput) error {
            // Called when leaving the "shipped" state
            return services.Cache.Invalidate(ctx, input.OrderID)
        },
    })
```

**Execution Order During Transition:**
1. Find transition (with automatic trigger bubbling up the hierarchy if needed)
2. Select the first matching branch — rejected if no branch matches
3. Exit current state and ancestors up to LCA (OnExit hooks)
4. Execute transition action (if defined)
5. Enter target state and ancestors from LCA down (OnEntry hooks)
6. If target state has an initial substate, enter it (OnEntry hook)

## Hierarchical States

Hierarchical states allow you to model complex state machines with parent-child relationships.

### Defining Hierarchies

```go
builder := fsm.NewBuilder[state, trigger, input]()

// Define parent-child relationships
builder.From(stateChild).WithParent(stateParent)
builder.From(stateGrandchild).WithParent(stateChild)
```

### Trigger Bubbling

When a trigger is fired from a child state, the FSM automatically searches up the hierarchy for a matching branch:

```go
// Define transition only on parent
builder.From(stateParent).On(triggerX).To(stateOther)

// Create FSM starting in grandchild
machine := fsm.New(spec, stateGrandchild)

// Trigger bubbles up: grandchild -> child -> parent (branch found!)
machine.Fire(ctx, triggerX, input) // Successfully transitions to stateOther
```

If a child state *has* a slot for the trigger but no branch matches, the FSM still bubbles up to the parent — it only stops at a level that actually selects a branch.

### Initial Substates

When transitioning to a state with an initial substate, the FSM automatically enters that substate:

```go
builder.From(stateParent).WithInitial(stateDefaultChild)
builder.From(stateDefaultChild).WithParent(stateParent)

builder.From(stateA).On(triggerX).To(stateParent)

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

Checks if a transition can be made without actually performing it. Allocation-free — does not call `Explain` internally.

```go
if machine.CanFire(ctx, triggerShip, input) {
    fmt.Println("Can ship the order")
} else {
    fmt.Println("Cannot ship the order yet")
}
```

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

| Error | When returned |
|---|---|
| `ErrTransitionRejected` | A slot exists for `(state, trigger)` but no branch's condition matched |
| `ErrNotFound` | No slot is defined for `(state, trigger)` at any hierarchy level |

```go
err := machine.Fire(ctx, trigger, input)
switch {
case err == nil:
    // transition succeeded
case errors.Is(err, fsm.ErrTransitionRejected):
    // a rule existed but no branch matched — conditions not met
    fmt.Println(err) // lists all tried condition descriptions
case errors.Is(err, fsm.ErrNotFound):
    // no transition defined for this state+trigger combination
}
```

## Mermaid Diagram Generation

Generate Mermaid.js state diagrams from your FSM specification for documentation. Each branch is emitted as a separate edge:

```go
spec := builder.Build()
diagram := spec.MermaidJSDiagram()
fmt.Println(diagram)
```

**Output:**
```
stateDiagram-v2
    Pending --> Completed : Withdraw [balance >= amount] / deduct balance
    Pending --> Overdraft : Withdraw [overdraftAllowed]
    Pending --> Rejected : Withdraw
    Paid --> Shipped : Ship / ship order
```

The diagram includes:
- All branches with their triggers
- Condition descriptions (in square brackets)
- Action descriptions (after forward slash)

You can use this in your documentation, wikis, or any tool that supports Mermaid.js.

## API Reference

See [fsm.go](fsm.go) for full API documentation and comments.

### Main Types

- `Spec[S, T, Input]` - Thread-safe FSM specification
- `Machine[S, T, Input]` - FSM instance with current state
- `Condition[Input]` - Function type for branch conditions: `func(input Input) bool`
- `Action[Input]` - Function type for transition actions and state hooks: `func(ctx context.Context, input Input) error`
- `Decision[S]` / `LevelVerdict[S]` / `BranchVerdict[S]` / `Outcome` — returned by `Explain`

### Builder API

- `NewBuilder[S, T, Input]()` - Create a new spec builder (dimensions derived automatically at `Build()` time)
- `.From(S).On(T).To(S)` - Open the first branch of a transition group
- `.When(desc string, cond func(Input) bool)` - Add a boolean condition to the current branch
- `.Do(desc string, action Action[Input])` - Add an action to the current branch
- `.To(S)` *(on branchStep)* - Close the current branch and open the next in the same group
- `.Otherwise(S)` - Open the final unconditional fallback branch (must be last)
- `.From(S).WithHooks(StateHooks[Input])` - Set entry/exit hooks for a state
- `.From(S).WithParent(S)` - Set parent state for hierarchical FSMs
- `.From(S).WithInitial(S)` - Set initial substate for hierarchical FSMs
- `.Build()` - Build the FSM specification

### Machine API

- `New[S, T, Input](spec *Spec, initialState S)` - Create a new FSM instance
- `.Fire(ctx, trigger, input)` - Attempt a state transition
- `.CanFire(ctx, trigger, input)` - Check if a branch would match (allocation-free)
- `.Explain(trigger, input)` - Return a full decision trace (allocates)
- `.State()` - Get current state
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
