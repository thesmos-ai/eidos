// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package builder

import (
	"errors"
	"fmt"

	"go.thesmos.sh/eidos/emit"
)

// ErrUnknownInsertPos surfaces only if a future contributor adds a
// new [posKind] variant without updating [applyInsert]'s switch.
// All [InsertPos] values produced by [Prepend] / [At] / [Before] /
// [After] (or the zero-value Append default) match an existing
// case, so this sentinel is unreachable at runtime under the
// current code. Callers compare with [errors.Is].
var ErrUnknownInsertPos = errors.New("builder: unknown insert position")

// posKind discriminates the [InsertPos] variants. The variants
// mirror the operations exposed on [emit.Slot]:
//
//   - kindAppend: insert at the end (the zero-value default)
//   - kindPrepend: insert at index 0
//   - kindAt: insert at a numeric index
//   - kindBefore: insert immediately before the item carrying the
//     supplied [emit.Provenance.ID]
//   - kindAfter: insert immediately after the item carrying the
//     supplied [emit.Provenance.ID]
type posKind int

const (
	kindAppend posKind = iota
	kindPrepend
	kindAt
	kindBefore
	kindAfter
)

// InsertPos captures where a cross-cutting contribution lands in
// its target slot. Construct via [Prepend], [At], [Before], or
// [After]; the zero value places the contribution at the end (the
// equivalent of the dedicated [Context.AppendField] / etc.).
//
// `Before` / `After` target the item whose [emit.Provenance.ID]
// equals the supplied anchor. The empty-anchor case returns
// [emit.ErrProvenanceNotFound] from the underlying slot op so a
// missed positional intent surfaces as a hard error rather than
// silent append-at-end behaviour.
type InsertPos struct {
	kind   posKind
	index  int
	anchor string
}

// Prepend returns an [InsertPos] placing the contribution at the
// start of its target slot. Subsequent items shift one position to
// the right.
func Prepend() InsertPos { return InsertPos{kind: kindPrepend} }

// At returns an [InsertPos] placing the contribution at the given
// numeric index. Use sparingly — indices shift as other plugins
// append, so positional intents that depend on a specific index
// are fragile. Prefer [Before] / [After] when the intent is
// "relative to plugin X's contribution".
func At(index int) InsertPos { return InsertPos{kind: kindAt, index: index} }

// Before returns an [InsertPos] placing the contribution
// immediately before the existing item whose [emit.Provenance.ID]
// equals anchor. The cross-cutting positional primitive for
// "insert validator checks before audit logging" patterns.
func Before(anchor string) InsertPos { return InsertPos{kind: kindBefore, anchor: anchor} }

// After returns an [InsertPos] placing the contribution
// immediately after the existing item whose [emit.Provenance.ID]
// equals anchor. Mirror of [Before].
func After(anchor string) InsertPos { return InsertPos{kind: kindAfter, anchor: anchor} }

// applyInsert dispatches op against slot for the position p,
// stamping prov. Centralised so every per-family Insert* helper
// shares the same dispatch shape. The default arm guards against
// a future contributor adding a new [posKind] variant without
// updating this switch — see [ErrUnknownInsertPos].
func applyInsert(slot *emit.Slot, item emit.Node, prov emit.Provenance, p InsertPos) error {
	switch p.kind {
	case kindAppend:
		return slot.Append(item, prov)
	case kindPrepend:
		return slot.Prepend(item, prov)
	case kindAt:
		return slot.InsertAt(p.index, item, prov)
	case kindBefore:
		return slot.InsertBefore(p.anchor, item, prov)
	case kindAfter:
		return slot.InsertAfter(p.anchor, item, prov)
	default:
		return fmt.Errorf("%w: %d", ErrUnknownInsertPos, p.kind)
	}
}
