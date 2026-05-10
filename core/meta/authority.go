// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package meta

import (
	"errors"
	"fmt"
	"strconv"
)

// Authority ranks the source of a metadata operation. When resolving
// the current value for a name, the entry with the highest authority
// wins. Within a single authority, tombstones beat values, and the
// last write to the (name, authority) slot beats earlier writes.
type Authority int

// The authority levels in ascending precedence.
const (
	// AuthorityPlugin is the default when a plugin's annotator or
	// generator calls Set or Tombstone on a Key. Multiple plugins
	// writing the same key at AuthorityPlugin contend with one
	// another: last write wins; tombstone wins over value.
	AuthorityPlugin Authority = iota
	// AuthorityDirective is reserved for the built-in directive-
	// override step that consumes +gen:meta / -gen:meta directives
	// (and registered sugar like +gen:shape). Directive entries beat
	// AuthorityPlugin entries regardless of write order.
	AuthorityDirective
	// AuthorityManual is reserved for test harnesses and programmatic
	// overrides. Manual entries beat both directive and plugin
	// entries; the contract guarantee is that no plugin and no
	// directive can shadow a Manual override.
	AuthorityManual
)

// ErrUnknownAuthority is returned by [ParseAuthority] for an input
// that does not name a defined level.
var ErrUnknownAuthority = errors.New("meta: unknown authority")

// String returns the lower-case textual form of a. Unknown values
// stringify as "authority(N)" so logs surface bad values without
// panicking.
func (a Authority) String() string {
	switch a {
	case AuthorityPlugin:
		return "plugin"
	case AuthorityDirective:
		return "directive"
	case AuthorityManual:
		return "manual"
	default:
		return "authority(" + strconv.Itoa(int(a)) + ")"
	}
}

// ParseAuthority returns the Authority matching the textual form
// produced by [Authority.String]. Unknown inputs return
// [ErrUnknownAuthority] wrapped with the offending string.
func ParseAuthority(s string) (Authority, error) {
	switch s {
	case "plugin":
		return AuthorityPlugin, nil
	case "directive":
		return AuthorityDirective, nil
	case "manual":
		return AuthorityManual, nil
	default:
		return 0, fmt.Errorf("%w: %q", ErrUnknownAuthority, s)
	}
}
