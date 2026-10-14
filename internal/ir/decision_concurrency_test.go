package ir_test

import (
	"sync"
	"testing"

	"github.com/getsyntegrity/feelc/internal/ir"
)

// lookupAllConcurrently resolves every decision of cm from many goroutines at once, starting
// together, so the first lookups race each other. Before Seal existed, the lazy index made
// this abort the process with "fatal error: concurrent map writes".
func lookupAllConcurrently(t *testing.T, cm *ir.CompiledModel) {
	t.Helper()
	const goroutines = 32
	start := make(chan struct{})
	var wg sync.WaitGroup
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for i := range cm.Decisions {
				name := cm.Decisions[i].Name
				d, ok := cm.Decision(name)
				if !ok || d.Name != name {
					t.Errorf("Decision(%q) = %v, %v", name, d, ok)
					return
				}
			}
		}()
	}
	close(start)
	wg.Wait()
}

func TestDecisionLookupIsSafeOnColdDecodedModel(t *testing.T) {
	blob, err := ir.Encode(compileRich(t))
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	// A fresh model per round: the race only exists on first use.
	for round := 0; round < 50; round++ {
		cm, err := ir.Decode(blob)
		if err != nil {
			t.Fatalf("decode: %v", err)
		}
		lookupAllConcurrently(t, cm)
	}
}

func TestDecisionLookupIsSafeOnUnsealedModel(t *testing.T) {
	src := compileRich(t)
	// A literal model never passes through a constructor, so it is not sealed.
	cm := &ir.CompiledModel{Name: src.Name, Decisions: append([]ir.Decision(nil), src.Decisions...)}
	lookupAllConcurrently(t, cm)
}

func TestDecisionLookupSeesDecisionsAddedAfterSeal(t *testing.T) {
	cm := compileRich(t)
	added := cm.Decisions[0]
	added.Name = "added_after_seal"
	cm.Decisions = append(cm.Decisions, added)
	d, ok := cm.Decision("added_after_seal")
	if !ok || d.Name != "added_after_seal" {
		t.Fatalf("Decision(added_after_seal) = %v, %v", d, ok)
	}
}

func TestDecisionLookupUnknownName(t *testing.T) {
	if d, ok := compileRich(t).Decision("does_not_exist"); ok || d != nil {
		t.Fatalf("Decision(does_not_exist) = %v, %v; want nil, false", d, ok)
	}
}
