// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package emit

import (
	"errors"
	"fmt"

	"go.thesmos.sh/eidos/core/directive"
)

// ErrSlotElementType is returned by [Slot.Append], [Slot.Prepend],
// and the insertion helpers when the supplied item does not match
// the slot's declared element kind.
var ErrSlotElementType = errors.New("emit: slot element kind mismatch")

// ErrProvenanceNotFound is returned by [Slot.InsertBefore] and
// [Slot.InsertAfter] when no item in the slot carries a matching
// [Provenance.ID]. Plugins that target other plugins' contributions
// must coordinate on the ID convention (typically via a shared
// const exported by the producing plugin).
var ErrProvenanceNotFound = errors.New("emit: provenance ID not found in slot")

// Slot is a named, kind-checked, provenance-tracked region attached
// to an emit [Node]. Generators append items with [Provenance]
// identifying the contributing plugin; cross-cutting generators
// inject contributions into other plugins' emit without owning the
// host node.
//
// Slot is the primary composition mechanism in emit: when a
// cross-cutter (validation, audit, debug) wants to add a statement
// to every Method's `prebody`, it calls method.Slot("prebody").Append(...)
// without needing to coordinate with the method's owning generator.
// Backends render slot contributions in append order alongside the
// host's typed content.
//
// Standard slot names by host kind (informational; the host's typed
// field model is the source of truth for direct content; slots are
// for cross-cutting injection):
//
//   - [File]:        "top", "bottom", "init"
//   - [Struct]:      "fields", "methods", "embeds"
//   - [Interface]:   "methods", "embeds"
//   - [Method]:      "prebody", "postbody", "params", "returns"
//   - [Function]:    "prebody", "postbody", "params", "returns"
//   - [Enum]:        "variants"
//   - [Field]:       "tags"
//
// Custom slot names are valid too — plugin-defined emit kinds may
// declare their own slot conventions.
//
// Slots are append-only by default. Insertion helpers
// ([Slot.Prepend], [Slot.InsertAt]) provide positioning when needed.
// Order matters for rendering: backends emit contributions in slot
// order, so plugin ordering (via the pipeline's capability topo)
// determines visible output order.
type Slot struct {
	BaseEmit

	// SlotName is the name this slot is registered under on its
	// owner.
	SlotName string `json:"slot_name"`

	// Owner is the host emit node that exposes this slot.
	//
	// Owner is excluded from JSON encoding to break the host →
	// slot cycle. Deserialized graphs re-wire Owner via
	// [RewireOwners].
	Owner Node `json:"-"`

	// ElemKind, when non-empty, constrains [Slot.Append] to accept
	// only items whose Kind() matches. An empty ElemKind accepts
	// any kind (use when the slot's content is intentionally
	// heterogeneous).
	ElemKind directive.Kind `json:"elem_kind,omitempty"`

	// Items holds the appended contributions in their final order.
	Items []Node `json:"-"`

	// Provenance is a parallel slice with Items: ProvenanceList[i]
	// records who appended Items[i] and where the contribution
	// originated.
	ProvenanceList []Provenance `json:"provenance,omitempty"`
}

// Kind returns [KindSlot].
func (*Slot) Kind() directive.Kind { return KindSlot }

// Len returns the number of items currently in the slot.
func (s *Slot) Len() int { return len(s.Items) }

// Append adds item to the end of the slot's Items, recording prov in
// the parallel ProvenanceList. Returns [ErrSlotElementType] wrapped
// with the offending kind when ElemKind is set and item.Kind() does
// not match.
func (s *Slot) Append(item Node, prov Provenance) error {
	if err := s.checkKind(item); err != nil {
		return err
	}
	s.Items = append(s.Items, item)
	s.ProvenanceList = append(s.ProvenanceList, prov)
	return nil
}

// Prepend inserts item at the beginning of the slot's Items.
// Subsequent items shift one position to the right.
func (s *Slot) Prepend(item Node, prov Provenance) error {
	return s.InsertAt(0, item, prov)
}

// InsertAt inserts item at the given index. Items at and after the
// index shift one position to the right. Out-of-range indexes return
// an error wrapping [ErrSlotElementType] (or a separate sentinel for
// the bound case below).
func (s *Slot) InsertAt(index int, item Node, prov Provenance) error {
	if index < 0 || index > len(s.Items) {
		return fmt.Errorf("emit: slot %q insert index %d out of range [0, %d]", s.SlotName, index, len(s.Items))
	}
	if err := s.checkKind(item); err != nil {
		return err
	}
	s.Items = append(s.Items[:index], append([]Node{item}, s.Items[index:]...)...)
	s.ProvenanceList = append(s.ProvenanceList[:index], append([]Provenance{prov}, s.ProvenanceList[index:]...)...)
	return nil
}

// InsertBefore inserts item immediately before the first existing
// item whose [Provenance.ID] equals id. Returns
// [ErrProvenanceNotFound] when no item carries that ID (including
// the empty-string case — an empty id never matches).
//
// Cross-cutting plugins use InsertBefore to position their
// contributions relative to a specific prior contribution rather
// than at a numeric index, which would shift as other plugins
// append.
func (s *Slot) InsertBefore(id string, item Node, prov Provenance) error {
	if id == "" {
		return fmt.Errorf("%w: empty id", ErrProvenanceNotFound)
	}
	for i, p := range s.ProvenanceList {
		if p.ID == id {
			return s.InsertAt(i, item, prov)
		}
	}
	return fmt.Errorf("%w: %q in slot %q", ErrProvenanceNotFound, id, s.SlotName)
}

// InsertAfter inserts item immediately after the first existing
// item whose [Provenance.ID] equals id. Returns
// [ErrProvenanceNotFound] when no item carries that ID.
func (s *Slot) InsertAfter(id string, item Node, prov Provenance) error {
	if id == "" {
		return fmt.Errorf("%w: empty id", ErrProvenanceNotFound)
	}
	for i, p := range s.ProvenanceList {
		if p.ID == id {
			return s.InsertAt(i+1, item, prov)
		}
	}
	return fmt.Errorf("%w: %q in slot %q", ErrProvenanceNotFound, id, s.SlotName)
}

// At returns the item at the given index, or nil when index is out
// of range.
func (s *Slot) At(i int) Node {
	if i < 0 || i >= len(s.Items) {
		return nil
	}
	return s.Items[i]
}

// ProvenanceAt returns the Provenance recorded for the item at the
// given index, or the zero Provenance when index is out of range.
func (s *Slot) ProvenanceAt(i int) Provenance {
	if i < 0 || i >= len(s.ProvenanceList) {
		return Provenance{}
	}
	return s.ProvenanceList[i]
}

// checkKind validates item against ElemKind. An empty ElemKind
// accepts any item; otherwise the item's Kind must match exactly.
func (s *Slot) checkKind(item Node) error {
	if s.ElemKind == "" {
		return nil
	}
	if item == nil {
		return fmt.Errorf("%w: slot %q: expected %s, got nil", ErrSlotElementType, s.SlotName, s.ElemKind)
	}
	if got := item.Kind(); got != s.ElemKind {
		return fmt.Errorf("%w: slot %q: expected %s, got %s", ErrSlotElementType, s.SlotName, s.ElemKind, got)
	}
	return nil
}

// SlotHost is the interface every emit value that owns slots
// satisfies — [Struct], [Interface], [Function], [Method], [Enum],
// [Alias], [File], and [Package] all expose `Slot(name)` for
// generic, name-based slot access. Template helpers and tooling
// that operate on "the slot named X on this host" without caring
// about the concrete host type accept SlotHost.
type SlotHost interface {
	Node

	// Slot returns the named slot on the host, creating it lazily
	// without an element-kind constraint.
	Slot(name string) *Slot
}

// slotMap is a small reusable type carrying a lazily-allocated map
// of slots; embedded by every host kind that exposes slots. The
// map's key is the slot name.
type slotMap struct {
	slots map[string]*Slot
}

// slot returns the named slot, creating it lazily with the supplied
// owner and element-kind constraint. Repeat calls return the same
// instance.
func (m *slotMap) slot(owner Node, name string, elemKind directive.Kind) *Slot {
	if m.slots == nil {
		m.slots = map[string]*Slot{}
	}
	if existing, ok := m.slots[name]; ok {
		return existing
	}
	sl := &Slot{SlotName: name, Owner: owner, ElemKind: elemKind}
	m.slots[name] = sl
	return sl
}

// SlotsByName returns every slot registered on this host, indexed
// by name. The returned map aliases the host's internal state;
// callers must not mutate it.
func (m *slotMap) slotsByName() map[string]*Slot {
	if m.slots == nil {
		m.slots = map[string]*Slot{}
	}
	return m.slots
}
