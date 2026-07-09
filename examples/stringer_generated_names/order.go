// Package stringergeneratednames shows how to get readable state/trigger names in errors, logs, and
// Mermaid diagrams without hand-writing a String() method for every enum.
//
// Instead of maintaining a switch statement that drifts out of sync with the const block, annotate the
// type with a //go:generate directive and let the stringer tool generate an efficient String() for you.
//
// To (re)generate the *_string.go files after adding or renaming a constant:
//
//	# one-time install of the generator (dev-time only; not needed to build or run this package)
//	go install golang.org/x/tools/cmd/stringer@latest
//
//	# regenerate from this package's directory (or `go generate ./...` from the repo root)
//	go generate ./...
//
// The generated files (orderstate_string.go, ordertrigger_string.go) are committed to the repo, so anyone
// building or importing this package never needs stringer installed — only contributors editing the enums do.
//
// stringer derives names directly from the Go identifiers, so states and triggers are named in plain
// past tense / imperative form (Created, Pay, ...) matching the naming convention the README recommends.
package stringergeneratednames

//go:generate stringer -type=orderState
type orderState uint

const (
	Created   orderState = iota // Order has been created
	Paid                        // Payment received
	Shipped                     // Order shipped to customer
	Delivered                   // Order delivered
	Completed                   // Order completed successfully
	Cancelled                   // Order cancelled
)

//go:generate stringer -type=orderTrigger
type orderTrigger uint

const (
	Pay      orderTrigger = iota // Customer pays
	Ship                         // Warehouse ships order
	Deliver                      // Carrier delivers order
	Complete                     // Customer confirms completion
	Cancel                       // Order cancelled
)
