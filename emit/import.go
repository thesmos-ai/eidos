// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package emit

import "go.thesmos.sh/eidos/core/directive"

// Import is one import declaration in an emit [File]. Each Import
// names a third-party or stdlib package the emit file references.
//
// Backends commonly derive Imports from [ExternalRef] usage at
// render time via the `imp` template func; explicit Import nodes
// let generators force-include packages for side effects (Go's
// `import _ "..."`) or to declare an alias up-front.
type Import struct {
	BaseEmit

	// Path is the imported package path.
	Path string

	// Alias is the explicit local name for the import, or empty
	// when the import uses the path-derived default. A leading
	// underscore alias means "import for side effects only".
	Alias string

	// Owner is the [File] this import belongs to.
	//
	// Owner is excluded from JSON encoding to break the host →
	// child cycle. Deserialized graphs re-wire Owner via
	// [RewireOwners].
	Owner *File `json:"-"`
}

// Kind returns [KindImport].
func (*Import) Kind() directive.Kind { return KindImport }

// LocalName returns the locally-visible name for the imported
// package — the Alias when supplied, otherwise the last
// `/`-delimited segment of [Import.Path]. Returns "" for malformed
// Paths.
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

// IsBlank reports whether the import is a "side-effect-only" import
// (Go's `import _ "..."` form).
func (i *Import) IsBlank() bool { return i.Alias == "_" }
