// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package emit

import (
	"errors"

	"go.thesmos.sh/eidos/core/directive"
)

// Return is one return-value slot on a [Function] or [Method].
// Each slot carries an optional Name plus a required Type. A
// Function or Method whose Returns slice has any non-empty Name
// produces a named-returns signature; an all-empty-name slice
// produces an anonymous-returns signature. Mixing the two in a
// single slice is forbidden by Go's grammar — backends surface
// it as [ErrMixedNamedReturns].
//
// Return embeds [BaseEmit] so it satisfies [Node] — the form
// cross-cutting plugins use when appending to
// [Function.ReturnsSlot] / [Method.ReturnsSlot]. Most fields on
// BaseEmit are unused by the rendered output (a Return has no
// docs or directive surface in Go's grammar) but they're available
// for plugins that thread provenance through their contributions.
type Return struct {
	BaseEmit

	// Name is the return identifier when the function declares a
	// named return ("func F() (n int, err error)"). Empty for the
	// anonymous-returns form.
	Name string `json:"name,omitempty"`

	// Type is the return slot's declared type.
	Type Ref `json:"-"`
}

// Kind returns [KindReturn].
func (*Return) Kind() directive.Kind { return KindReturn }

// ErrMixedNamedReturns is returned by render helpers when a
// function or method's Returns slice mixes named and unnamed
// entries. Go's grammar requires all return slots to be either
// named or all anonymous within a single signature; the wrapped
// message names the offending entity so the diagnostic locates
// the bug without a stack trace.
var ErrMixedNamedReturns = errors.New("emit: returns mix named and unnamed entries")

// AnonReturns wraps a sequence of types as anonymous [*Return]
// slots — the convenience shape for callers that don't need
// named returns. Named-return callers construct slots directly
// via struct literals.
func AnonReturns(types ...Ref) []*Return {
	out := make([]*Return, 0, len(types))
	for _, t := range types {
		out = append(out, &Return{Type: t})
	}
	return out
}
