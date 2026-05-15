// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package node

import (
	"go.thesmos.sh/eidos/core/kind"
)

// Embed is one embedded type in a [Struct] or [Interface]. Go's
// struct embedding and interface composition both surface as Embed
// nodes; downstream consumers use Owner to discriminate.
type Embed struct {
	BaseNode

	// Type is the embedded type reference.
	Type *TypeRef `json:"type,omitempty"`

	// Owner is the struct or interface doing the embedding.
	// Populated by the constructing frontend.
	//
	// Owner is excluded from JSON encoding to break the host →
	// child cycle. Deserialized graphs re-wire Owner via
	// [RewireOwners].
	Owner Node `json:"-"`
}

// Kind returns [KindEmbed].
func (*Embed) Kind() kind.Kind { return KindEmbed }
