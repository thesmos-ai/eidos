// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package storefixture

import (
	"fmt"

	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/store"
)

// defaultPackageName is the package name applied to a [Builder] that
// never calls [Builder.Package].
const defaultPackageName = "test"

// defaultPackagePath is the import path applied to a [Builder] that
// never calls [Builder.Package].
const defaultPackagePath = "example.com/test"

// Builder accumulates declarations into a single [node.Package] and
// turns the package into a populated [store.Store] on demand.
//
// All declarations the Builder accepts are added to one package; the
// fixture deliberately does not support multi-package construction —
// unit tests that need cross-package fixtures should build separate
// stores and merge in the test, or graduate to a [testpipe.Pipeline]
// driven by a synthetic frontend.
//
// Each declaration entry point ([Builder.Struct], [Builder.Interface],
// etc.) takes a configuration callback that receives the matching
// sub-builder. The callback runs synchronously; the Builder's state is
// updated before the entry point returns.
//
// A Builder is not safe for concurrent use. Tests typically construct
// a Builder in setup, configure it, call [Builder.Build], and let it
// fall out of scope.
type Builder struct {
	pkg *node.Package
}

// New returns a Builder seeded with an empty package whose Name is
// "test" and whose Path is "example.com/test". Call [Builder.Package]
// to override either value.
func New() *Builder {
	return &Builder{pkg: &node.Package{
		Name: defaultPackageName,
		Path: defaultPackagePath,
	}}
}

// Package overrides the package name and import path on the
// accumulating [node.Package]. Calling Package after declarations
// have already been added is allowed and rewrites the in-progress
// package's identity — existing decls' [node.Struct.Package],
// [node.Function.Package], and equivalents are rewritten so qualified
// names stay coherent with the new path.
func (b *Builder) Package(name, path string) *Builder {
	b.pkg.Name = name
	b.pkg.Path = path
	for _, s := range b.pkg.Structs {
		s.Package = path
	}
	for _, i := range b.pkg.Interfaces {
		i.Package = path
	}
	for _, f := range b.pkg.Functions {
		f.Package = path
	}
	for _, v := range b.pkg.Variables {
		v.Package = path
	}
	for _, c := range b.pkg.Constants {
		c.Package = path
	}
	for _, e := range b.pkg.Enums {
		e.Package = path
	}
	for _, a := range b.pkg.Aliases {
		a.Package = path
	}
	return b
}

// Import records an import path on the package's deduped import set.
// Imports are rarely meaningful in unit-test fixtures but the option
// is here so test cases that inspect the import view of a frontend's
// output have a way to seed entries.
func (b *Builder) Import(path string) *Builder {
	b.pkg.Imports = append(b.pkg.Imports, &node.Import{Path: path, Owner: b.pkg})
	return b
}

// Struct declares a struct in the accumulating package. When fn is
// non-nil it runs against a fresh [StructBuilder] before Struct
// returns, allowing the caller to populate fields, methods, embeds,
// directives, and docs.
//
// Duplicate struct names within the same package cause [Builder.Build]
// to fail with [store.ErrDuplicateQName]; the duplicate is not detected
// at Struct call time so callers may shadow earlier names intentionally
// in pathological tests.
func (b *Builder) Struct(name string, fn func(*StructBuilder)) *Builder {
	s := &node.Struct{Name: name, Package: b.pkg.Path}
	sb := &StructBuilder{s: s, pkgPath: b.pkg.Path}
	if fn != nil {
		fn(sb)
	}
	b.pkg.Structs = append(b.pkg.Structs, s)
	return b
}

// Interface declares an interface in the accumulating package.
func (b *Builder) Interface(name string, fn func(*InterfaceBuilder)) *Builder {
	i := &node.Interface{Name: name, Package: b.pkg.Path}
	ib := &InterfaceBuilder{i: i, pkgPath: b.pkg.Path}
	if fn != nil {
		fn(ib)
	}
	b.pkg.Interfaces = append(b.pkg.Interfaces, i)
	return b
}

// Function declares a standalone (non-method) function.
func (b *Builder) Function(name string, fn func(*FunctionBuilder)) *Builder {
	f := &node.Function{Name: name, Package: b.pkg.Path}
	fb := &FunctionBuilder{f: f}
	if fn != nil {
		fn(fb)
	}
	b.pkg.Functions = append(b.pkg.Functions, f)
	return b
}

// Variable declares a package-level variable.
func (b *Builder) Variable(name string, fn func(*VariableBuilder)) *Builder {
	v := &node.Variable{Name: name, Package: b.pkg.Path}
	vb := &VariableBuilder{v: v}
	if fn != nil {
		fn(vb)
	}
	b.pkg.Variables = append(b.pkg.Variables, v)
	return b
}

// Constant declares a package-level constant that is not part of an
// idiomatic enum group.
func (b *Builder) Constant(name string, fn func(*ConstantBuilder)) *Builder {
	c := &node.Constant{Name: name, Package: b.pkg.Path}
	cb := &ConstantBuilder{c: c}
	if fn != nil {
		fn(cb)
	}
	b.pkg.Constants = append(b.pkg.Constants, c)
	return b
}

// Enum declares an enum (a group of typed constants sharing an
// underlying type) in the accumulating package.
func (b *Builder) Enum(name string, fn func(*EnumBuilder)) *Builder {
	e := &node.Enum{Name: name, Package: b.pkg.Path}
	eb := &EnumBuilder{e: e}
	if fn != nil {
		fn(eb)
	}
	b.pkg.Enums = append(b.pkg.Enums, e)
	return b
}

// Alias declares a type alias or type definition. Use
// [AliasBuilder.True] to mark the declaration as an alias
// (`type X = Y`) rather than a definition (`type X Y`).
func (b *Builder) Alias(name string, fn func(*AliasBuilder)) *Builder {
	a := &node.Alias{Name: name, Package: b.pkg.Path}
	ab := &AliasBuilder{a: a}
	if fn != nil {
		fn(ab)
	}
	b.pkg.Aliases = append(b.pkg.Aliases, a)
	return b
}

// PackageNode returns the [node.Package] accumulated so far. The
// returned pointer aliases the Builder's internal storage — callers
// that mutate it will affect subsequent [Builder.Build] calls. Use
// this accessor to set typed metadata on individual nodes after
// construction or to assert against the raw shape.
func (b *Builder) PackageNode() *node.Package { return b.pkg }

// Build returns a fresh [store.Store] populated with the accumulated
// package. The builder is reusable: each call constructs an
// independent store, so consecutive calls return distinct stores
// containing the same configured package.
//
// Build panics on any state the underlying [store.NodeView.AddPackage]
// rejects — duplicate qualified names, nil entries. Such states reflect
// builder misuse rather than test data, and a panic at construction is
// easier to debug than a returned error swallowed silently. Tests
// catch the panic through the standard testing.T flow.
func (b *Builder) Build() *store.Store {
	s := store.New()
	if err := s.Nodes().AddPackage(b.pkg); err != nil {
		// Test-only fixture; misuse-on-construction surfaces as a panic.
		panic(fmt.Errorf("storefixture: build failed: %w", err)) //nolint:forbidigo
	}
	return s
}
