package stringergeneratednames

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tobbstr/fsm"
)

// payload carries per-Fire business data. Empty here — the point of this example is the generated names.
type payload struct{}

func buildSpec() *fsm.Spec[orderState, orderTrigger, payload] {
	b := fsm.NewBuilder[orderState, orderTrigger, payload]()
	b.From(stateCreated).On(triggerPay).To(statePaid)
	b.From(statePaid).On(triggerShip).To(stateShipped)
	b.From(stateShipped).On(triggerDeliver).To(stateDelivered)
	b.From(stateDelivered).On(triggerComplete).To(stateCompleted)
	return b.Build()
}

// TestGeneratedNamesAreReadable shows that the stringer-generated String() methods make State(),
// error messages, and Mermaid diagrams human-readable — with no hand-written switch statements.
func TestGeneratedNamesAreReadable(t *testing.T) {
	spec := buildSpec()
	m := fsm.New(spec, stateCreated)

	// State() prints the generated name, not a bare integer.
	require.Equal(t, "Created", m.State().String())

	require.NoError(t, m.Fire(context.Background(), triggerPay, payload{}))
	require.Equal(t, "Paid", m.State().String())

	// An undefined transition surfaces the readable names in the error message.
	err := m.Fire(context.Background(), triggerComplete, payload{})
	require.ErrorIs(t, err, fsm.ErrNotFound)
	require.Contains(t, err.Error(), "trigger (Complete)")
	require.Contains(t, err.Error(), "state (Paid)")

	// The Mermaid diagram uses the generated names for both states and triggers.
	diagram := spec.MermaidJSDiagram()
	require.Contains(t, diagram, "Created --> Paid : Pay")
	require.Contains(t, diagram, "Paid --> Shipped : Ship")
	require.False(t, strings.Contains(diagram, "0 --> 1"), "diagram should not contain raw integer states")
}
