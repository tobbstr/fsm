// Benchmark for fsm.Fire performance (CPU and memory)
package fsm

import (
	"context"
	"testing"
)

// Dummy states, triggers, and data types for benchmarking
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

type dummyOpts struct{}

func setupBenchmarkFSM() *Machine[uint, uint, dummyOpts] {
	builder := NewSpecBuilder[uint, uint, dummyOpts](26, 26)
	builder.Transition().From(stateA).On(triggerA).To(stateB)
	builder.Transition().From(stateB).On(triggerA).To(stateC)
	builder.Transition().From(stateC).On(triggerA).To(stateD)
	builder.Transition().From(stateD).On(triggerA).To(stateE)
	builder.Transition().From(stateE).On(triggerA).To(stateF)
	builder.Transition().From(stateF).On(triggerA).To(stateG)
	builder.Transition().From(stateG).On(triggerA).To(stateH)
	builder.Transition().From(stateH).On(triggerA).To(stateI)
	builder.Transition().From(stateI).On(triggerA).To(stateJ)
	builder.Transition().From(stateJ).On(triggerA).To(stateK)
	builder.Transition().From(stateK).On(triggerA).To(stateL)
	builder.Transition().From(stateL).On(triggerA).To(stateM)
	builder.Transition().From(stateM).On(triggerA).To(stateN)
	builder.Transition().From(stateN).On(triggerA).To(stateO)
	builder.Transition().From(stateO).On(triggerA).To(stateP)
	builder.Transition().From(stateP).On(triggerA).To(stateQ)
	builder.Transition().From(stateQ).On(triggerA).To(stateR)
	builder.Transition().From(stateR).On(triggerA).To(stateS)
	builder.Transition().From(stateS).On(triggerA).To(stateT)
	builder.Transition().From(stateT).On(triggerA).To(stateU)
	builder.Transition().From(stateU).On(triggerA).To(stateV)
	builder.Transition().From(stateV).On(triggerA).To(stateW)
	builder.Transition().From(stateW).On(triggerA).To(stateX)
	builder.Transition().From(stateX).On(triggerA).To(stateY)
	builder.Transition().From(stateY).On(triggerA).To(stateZ)
	builder.Transition().From(stateZ).On(triggerA).To(stateA) // Loop back to A
	spec := builder.Build()
	return New(spec, stateA)
}

func BenchmarkFire(b *testing.B) {
	ctx := context.Background()
	fsm := setupBenchmarkFSM()
	opts := dummyOpts{}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = fsm.Fire(ctx, triggerA, opts)
	}
}
