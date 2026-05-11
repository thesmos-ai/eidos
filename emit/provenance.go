// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package emit

import (
	"fmt"
	"strings"

	"go.thesmos.sh/eidos/core/position"
)

// Provenance records who appended an item to a [Slot] and where the
// contribution originated. The pipeline stamps Provenance.SetBy with
// the contributing plugin's name; cross-cutting generators carry the
// originating directive's [position.Pos] through to Pos for
// `eidos explain` trails.
type Provenance struct {
	// SetBy is the plugin name (or "manual"/"directive") that
	// appended the contribution.
	SetBy string `json:"set_by,omitempty"`

	// Pos is the source position the contribution originates from
	// when relevant (the directive's position for directive-driven
	// contributions, the source-node position for nodes-derived
	// contributions). Zero for plugins that don't track a position.
	Pos position.Pos `json:"pos,omitzero"`

	// ID is an optional unique identifier the appending plugin
	// stamps so later plugins can target this contribution via
	// [Slot.InsertBefore] / [Slot.InsertAfter]. Empty when the
	// contribution does not opt into positioned-insert targeting;
	// duplicates are not enforced (the insert helpers pick the
	// first match).
	ID string `json:"id,omitempty"`
}

// String renders p in the form "set by <SetBy> at <Pos>" — the
// trailing position is omitted when zero.
func (p Provenance) String() string {
	var b strings.Builder
	if p.SetBy != "" {
		fmt.Fprintf(&b, "set by %s", p.SetBy)
	} else {
		b.WriteString("set")
	}
	if !p.Pos.IsZero() {
		fmt.Fprintf(&b, " at %s", p.Pos)
	}
	return b.String()
}
