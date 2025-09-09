
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
package main

import (
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tobbstr/fsm"
)

var stateNames = []string{"red", "yellow", "green"}
var triggerNames = []string{"next"}

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
	return stateNames[s]
}

type trigger uint

func (t trigger) String() string {
	return triggerNames[t]
}

type data struct{
    pool *pgxpool.Pool
}

func main() {
    // Initialize database connection pool.
    myPgxPool, err := pgxpool.New(context.Background(), "your-database-url")
    // .. handle error ..
    defer myPgxPool.Close()

    // Constructs a new FSM specification builder.
    builder := fsm.NewSpecBuilder[state, trigger, data](3, 1)

    // Define transitions
    builder.Transition().From(red).On(next).To(yellow)
    builder.Transition().From(yellow).On(next).To(green)

    // Only initialize this once as it is read-only, meaning thread-safe.
    // Store it in a global variable.
    spec := builder.Build()

    // Creates instance of the FSM with the initial state `red`.
    // This should be instantiated every time the FSM is needed. For example, in a request handler.
    m := fsm.New(spec, red)

    // Trigger events
    state := m.State() // returns red
    err := m.Fire(context.Background(), next, data{pool: myPgxPool})
    // .. handle error ..

    state := m.State() // returns yellow
    err := m.Fire(context.Background(), next, data{pool: myPgxPool})
    // .. handle error ..

    state := m.State() // returns green

    err := m.Fire(context.Background(), next, data{pool: myPgxPool}) // Returns a fsm.ErrNotFound error as there is not defined transition from green for the trigger (next).
}
```

For more examples, see [examples](./examples).

## API Reference

See [fsm.go](fsm.go) for full API documentation and comments.

## Testing

Run unit tests with:

```sh
go test ./...
```

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

## Contributing

Contributions, issues, and feature requests are welcome! Feel free to open an issue or submit a pull request.

## Author

- [Tobias Strandberg](https://github.com/tobbstr)
