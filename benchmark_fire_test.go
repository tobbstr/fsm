// Benchmark for fsm.Fire performance (CPU and memory)
package fsm

import (
	"context"
	"testing"
)

// Dummy states, triggers, and payload types for benchmarking
const (
	stateA = iota
	stateB
	stateC
	stateD
	stateE
	stateF
	stateG
	stateH
	stateI
	stateJ
	stateK
	stateL
	stateM
	stateN
	stateO
	stateP
	stateQ
	stateR
	stateS
	stateT
	stateU
	stateV
	stateW
	stateX
	stateY
	stateZ
)

const (
	triggerA = iota
	triggerB
	triggerC
	triggerD
	triggerE
	triggerF
	triggerG
	triggerH
	triggerI
	triggerJ
	triggerK
	triggerL
	triggerM
	triggerN
	triggerO
	triggerP
	triggerQ
	triggerR
	triggerS
	triggerT
	triggerU
	triggerV
	triggerW
	triggerX
	triggerY
	triggerZ
)

type dummyPayload struct{}

func setupBenchmarkFSM() *Machine[uint, uint, dummyPayload] {
	builder := NewBuilder[uint, uint, dummyPayload]()
	builder.From(stateA).On(triggerA).To(stateB)
	builder.From(stateB).On(triggerA).To(stateC)
	builder.From(stateC).On(triggerA).To(stateD)
	builder.From(stateD).On(triggerA).To(stateE)
	builder.From(stateE).On(triggerA).To(stateF)
	builder.From(stateF).On(triggerA).To(stateG)
	builder.From(stateG).On(triggerA).To(stateH)
	builder.From(stateH).On(triggerA).To(stateI)
	builder.From(stateI).On(triggerA).To(stateJ)
	builder.From(stateJ).On(triggerA).To(stateK)
	builder.From(stateK).On(triggerA).To(stateL)
	builder.From(stateL).On(triggerA).To(stateM)
	builder.From(stateM).On(triggerA).To(stateN)
	builder.From(stateN).On(triggerA).To(stateO)
	builder.From(stateO).On(triggerA).To(stateP)
	builder.From(stateP).On(triggerA).To(stateQ)
	builder.From(stateQ).On(triggerA).To(stateR)
	builder.From(stateR).On(triggerA).To(stateS)
	builder.From(stateS).On(triggerA).To(stateT)
	builder.From(stateT).On(triggerA).To(stateU)
	builder.From(stateU).On(triggerA).To(stateV)
	builder.From(stateV).On(triggerA).To(stateW)
	builder.From(stateW).On(triggerA).To(stateX)
	builder.From(stateX).On(triggerA).To(stateY)
	builder.From(stateY).On(triggerA).To(stateZ)
	builder.From(stateZ).On(triggerA).To(stateA) // Loop back to A
	spec := builder.Build()
	return New(spec, stateA)
}

func BenchmarkFire(b *testing.B) {
	ctx := context.Background()
	fsm := setupBenchmarkFSM()
	payload := dummyPayload{}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = fsm.Fire(ctx, triggerA, payload)
	}
}

func BenchmarkFire_SingleConditional(b *testing.B) {
	ctx := context.Background()
	builder := NewBuilder[uint, uint, dummyPayload]()
	builder.From(stateA).On(triggerA).To(stateB).When("always true", func(dummyPayload) bool { return true })
	builder.From(stateB).On(triggerA).To(stateA)
	fsm := New(builder.Build(), stateA)
	payload := dummyPayload{}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = fsm.Fire(ctx, triggerA, payload)
	}
}

func BenchmarkFire_Branching(b *testing.B) {
	ctx := context.Background()
	builder := NewBuilder[uint, uint, dummyPayload]()
	// 3 branches: first two have conditions that return false, last one is Otherwise (unconditional).
	builder.From(stateA).On(triggerA).
		To(stateB).When("false1", func(dummyPayload) bool { return false }).
		To(stateC).When("false2", func(dummyPayload) bool { return false }).
		Otherwise(stateD)
	builder.From(stateB).On(triggerA).To(stateA)
	builder.From(stateC).On(triggerA).To(stateA)
	builder.From(stateD).On(triggerA).To(stateA)
	fsm := New(builder.Build(), stateA)
	payload := dummyPayload{}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = fsm.Fire(ctx, triggerA, payload)
	}
}
