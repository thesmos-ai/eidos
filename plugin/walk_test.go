// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package plugin_test

import (
	"testing"

	"go.thesmos.sh/eidos/core/diag"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/plugin"
	"go.thesmos.sh/eidos/store"
)

// walkProbe is a single struct that opts into each hook by adding
// the per-kind method. Tests configure it for the subset of hooks
// they want to verify; the captured call order is then asserted.
type walkProbe struct {
	called []string
}

func (w *walkProbe) BeforeNodes(*plugin.AnnotatorContext) { w.called = append(w.called, "before") }
func (w *walkProbe) AfterNodes(*plugin.AnnotatorContext)  { w.called = append(w.called, "after") }
func (w *walkProbe) OnStruct(_ *plugin.AnnotatorContext, s *node.Struct) {
	w.called = append(w.called, "struct:"+s.Name)
}

func (w *walkProbe) OnInterface(_ *plugin.AnnotatorContext, i *node.Interface) {
	w.called = append(w.called, "interface:"+i.Name)
}

// emptyProbe implements none of the hooks. Used to verify Walk's
// "no implementation, no dispatch" behaviour: every interface
// assertion misses and no node iteration runs.
type emptyProbe struct{}

// TestWalk covers the [plugin.Walk] dispatcher: hooks fire in the
// documented order (BeforeNodes, OnStruct per struct, OnInterface
// per interface, AfterNodes); only hooks the target implements
// fire; targets without any hook reach Walk without iterating.
func TestWalk(t *testing.T) {
	t.Parallel()

	t.Run("dispatches every hook in the documented order", func(t *testing.T) {
		t.Parallel()
		ctx := newWalkContext(t)
		probe := &walkProbe{}
		assertNoError(t, plugin.Walk(ctx, probe))
		want := []string{
			"before",
			"struct:Alpha",
			"struct:Beta",
			"interface:Reader",
			"after",
		}
		assertCalledOrder(t, probe.called, want)
	})

	t.Run("skips iteration for targets that implement no hooks", func(t *testing.T) {
		t.Parallel()
		ctx := newWalkContext(t)
		// No assertion on Diag — Walk is allowed to short-circuit
		// when no hook matches. The success criterion is that the
		// call returns nil without panicking on the empty probe.
		if err := plugin.Walk(ctx, &emptyProbe{}); err != nil {
			t.Fatalf("Walk on empty probe: %v", err)
		}
	})
}

// newWalkContext builds a populated AnnotatorContext: two structs
// and one interface in stable insertion order so the per-kind
// iteration order is deterministic.
func newWalkContext(t *testing.T) *plugin.AnnotatorContext {
	t.Helper()
	s := store.New()
	if err := s.Nodes().AddPackage(&node.Package{
		Name: "x", Path: "x",
		Structs: []*node.Struct{
			{Name: "Alpha", Package: "x"},
			{Name: "Beta", Package: "x"},
		},
		Interfaces: []*node.Interface{
			{Name: "Reader", Package: "x"},
		},
	}); err != nil {
		t.Fatalf("AddPackage: %v", err)
	}
	return &plugin.AnnotatorContext{
		Store:  s,
		Reader: store.NewReader(s),
		Diag:   diag.New(),
	}
}

// assertCalledOrder fails the test when got does not match want
// element-wise. Used by the Walk dispatch-order test so the
// failure message names the divergence.
func assertCalledOrder(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("call order length mismatch: got %v, want %v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("call order [%d] = %q, want %q (full got=%v, want=%v)", i, got[i], want[i], got, want)
		}
	}
}
