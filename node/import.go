// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package node

import (
	"go.thesmos.sh/eidos/core/kind"
)

// Import is one import declaration — a dependency on another package.
// Frontends produce one Import per import line in a [File]; the
// owning [Package] also exposes a deduplicated union view over all
// files via [Package.Imports].
//
// Local aliases (Go's `import foo "github.com/bar/baz"`) populate
// Alias; the un-aliased form leaves Alias empty and downstream
// consumers derive the local name from the path's last segment.
type Import struct {
	BaseNode

	// Path is the imported package path.
	Path string `json:"path"`

	// Alias is the explicit local name supplied at the import site,
	// or empty when no alias was given.
	Alias string `json:"alias,omitempty"`

	// Owner is the [File] (per-file declaration) or [Package]
	// (deduplicated union) the import belongs to. Populated by the
	// constructing frontend.
	//
	// Owner is excluded from JSON encoding to break the host →
	// child cycle. Deserialized graphs re-wire Owner via
	// [RewireOwners].
	Owner Node `json:"-"`
}

// Kind reports [KindImport].
func (*Import) Kind() kind.Kind { return KindImport }

// LocalName returns the locally-visible name for the imported package
// — the Alias when supplied, otherwise the last `/`-delimited
// segment of [Import.Path]. Returns "" for malformed Paths.
func (i *Import) LocalName() string {
	if i.Alias != "" {
		return i.Alias
	}
	for j := len(i.Path) - 1; j >= 0; j-- {
		if i.Path[j] == '/' {
			return i.Path[j+1:]
		}
	}
	return i.Path
}
