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
// -trimprefix trims the shared prefix so stateCreated prints as "Created" and triggerPay as "Pay",
// matching the naming convention the README recommends.
package stringergeneratednames

//go:generate stringer -type=orderState -trimprefix=state
type orderState uint

const (
	stateCreated orderState = iota // Order has been created
	statePaid                      // Payment received
	stateShipped                   // Order shipped to customer
	stateDelivered                 // Order delivered
	stateCompleted                 // Order completed successfully
	stateCancelled                 // Order cancelled
)

//go:generate stringer -type=orderTrigger -trimprefix=trigger
type orderTrigger uint

const (
	triggerPay      orderTrigger = iota // Customer pays
	triggerShip                         // Warehouse ships order
	triggerDeliver                      // Carrier delivers order
	triggerComplete                     // Customer confirms completion
	triggerCancel                       // Order cancelled
)
