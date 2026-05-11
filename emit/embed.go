// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package emit

import "go.thesmos.sh/eidos/core/directive"

// Embed is one embedded type in a [Struct] or [Interface]. Go's
// struct embedding and interface composition both surface as Embed
// nodes; downstream consumers use Owner to discriminate.
type Embed struct {
	BaseEmit

	// Type is the embedded type reference.
	Type Ref

	// Owner is the struct or interface doing the embedding.
	//
	// Owner is excluded from JSON encoding to break the host →
	// child cycle. Deserialized graphs re-wire Owner via
	// [RewireOwners].
	Owner Node `json:"-"`
}

// Kind returns [KindEmbed].
func (*Embed) Kind() directive.Kind { return KindEmbed }
