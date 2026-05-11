// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package writer

import (
	"fmt"
	"strings"
	"sync"
)

// AliasFunc derives the default local alias for an import path
// when the caller has not registered an explicit one. The default
// implementation [DefaultAlias] returns the last "/"-delimited
// segment of the path (Go convention); other languages may
// substitute their own derivation.
type AliasFunc func(path string) string

// DefaultAlias returns the last "/"-delimited segment of path —
// "github.com/foo/bar" → "bar", "context" → "context". This is the
// Go-conventional default; pass it (or a custom function) to
// [NewImportSet] to control aliasing.
func DefaultAlias(path string) string {
	if i := strings.LastIndexByte(path, '/'); i >= 0 {
		return path[i+1:]
	}
	return path
}

// Import is one entry in the [ImportSet]: the canonical package
// path and the local alias the writer assigned. Alias is always
// non-empty after [ImportSet.Imp] has been called.
type Import struct {
	Path  string
	Alias string
}

// ImportSet manages the per-file import block. Each path the
// backend asks about via [ImportSet.Imp] is recorded once;
// subsequent calls for the same path return the same alias.
// Collisions on the derived alias are resolved deterministically
// with a numeric suffix (e.g. two paths both derived to "context"
// produce aliases "context" and "context2").
//
// ImportSet is safe for concurrent use; the backend's parallel
// per-file rendering can dispatch through one ImportSet per file
// without coordination.
//
// The zero value is unusable; construct via [NewImportSet].
type ImportSet struct {
	mu       sync.Mutex
	derive   AliasFunc
	order    []string          // paths in insertion order
	aliases  map[string]string // path -> assigned alias
	explicit map[string]string // path -> caller-supplied override
	used     map[string]string // alias -> path it resolved to
}

// NewImportSet returns an empty ImportSet. Pass nil for derive to
// use [DefaultAlias] (the Go-conventional last-segment derivation).
func NewImportSet(derive AliasFunc) *ImportSet {
	if derive == nil {
		derive = DefaultAlias
	}
	return &ImportSet{
		derive:   derive,
		aliases:  map[string]string{},
		explicit: map[string]string{},
		used:     map[string]string{},
	}
}

// Imp records path and returns the local alias to use in rendered
// output. Repeat calls for the same path return the same alias.
// Returns [ErrEmptyPath] when path is empty.
//
// Collision handling: when the derived alias is already taken by a
// different path, Imp appends a numeric suffix ("alias", "alias2",
// "alias3", …) until a free name is found. The suffix is
// deterministic for the same registration order.
func (i *ImportSet) Imp(path string) (string, error) {
	if path == "" {
		return "", ErrEmptyPath
	}
	i.mu.Lock()
	defer i.mu.Unlock()

	if existing, ok := i.aliases[path]; ok {
		return existing, nil
	}

	desired := i.explicit[path]
	if desired == "" {
		desired = i.derive(path)
	}

	alias := desired
	for n := 2; ; n++ {
		owner, taken := i.used[alias]
		if !taken || owner == path {
			break
		}
		alias = fmt.Sprintf("%s%d", desired, n)
	}

	i.order = append(i.order, path)
	i.aliases[path] = alias
	i.used[alias] = path
	return alias, nil
}

// Alias registers an explicit local alias for path, overriding the
// derived default. The override must be registered before path is
// first imported via [ImportSet.Imp]; otherwise Alias returns
// [ErrAliasAfterImp].
//
// Empty path returns [ErrEmptyPath].
func (i *ImportSet) Alias(path, alias string) error {
	if path == "" {
		return ErrEmptyPath
	}
	i.mu.Lock()
	defer i.mu.Unlock()
	if _, alreadyImped := i.aliases[path]; alreadyImped {
		return fmt.Errorf("%w: %q", ErrAliasAfterImp, path)
	}
	i.explicit[path] = alias
	return nil
}

// AliasOf returns the assigned alias for path along with true; when
// the path has not been imported yet, returns "" and false.
func (i *ImportSet) AliasOf(path string) (string, bool) {
	i.mu.Lock()
	defer i.mu.Unlock()
	a, ok := i.aliases[path]
	return a, ok
}

// Imports returns every recorded import in insertion order. The
// returned slice is safe for the caller to mutate; subsequent
// changes to the ImportSet are not reflected.
func (i *ImportSet) Imports() []Import {
	i.mu.Lock()
	defer i.mu.Unlock()
	out := make([]Import, len(i.order))
	for k, p := range i.order {
		out[k] = Import{Path: p, Alias: i.aliases[p]}
	}
	return out
}

// Len returns the number of recorded imports.
func (i *ImportSet) Len() int {
	i.mu.Lock()
	defer i.mu.Unlock()
	return len(i.order)
}
