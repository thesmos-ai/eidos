// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package builder

import (
	"errors"

	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/node"
)

// PackageBuilder accumulates a single [emit.Package]. Spawned by
// [Context.Package]; consumed by [PackageBuilder.Build], which
// returns the populated *emit.Package alongside the joined error
// from any structural rule violations detected during construction
// (e.g. methods on a true alias).
//
// Every decl-spawning method appends to the underlying package and
// returns the same builder so calls chain. Each appended decl
// inherits the Context's [emit.Target]; a plugin emitting into
// multiple files chains through several Contexts via
// [Context.WithTarget].
//
// # Output tagging
//
// A multi-output plugin obtains a sub-context for a specific
// output via [PackageBuilder.File] — decls built through that
// sub-context stamp [emit.BaseEmit.OutputTag] with the supplied
// tag so the routing layer can dispatch each decl to its
// declared [go.thesmos.sh/eidos/plugin.Output]. Sub-contexts
// share the same underlying [emit.Package], anchor default
// origin, and error sink as the root builder.
//
// Errors are accumulated rather than returned per call so the
// fluent chain stays unbroken; [PackageBuilder.Build] surfaces them
// in one [errors.Join] result. Callers that want to abort the chain
// early can inspect [PackageBuilder.Err] between calls.
type PackageBuilder struct {
	ctx           *Context
	pkg           *emit.Package
	defaultOrigin node.Node
	errs          []error

	// outputTag is the tag every decl built through this
	// PackageBuilder receives via [emit.BaseEmit.OutputTag]. The
	// root builder carries the empty string; sub-contexts
	// returned by [PackageBuilder.File] carry a non-empty tag.
	outputTag string

	// files maps tag → sub-context PackageBuilder. Populated on
	// the root; nil on sub-contexts. Memoisation: repeated
	// [PackageBuilder.File] calls with the same tag return the
	// same sub-context instance.
	files map[string]*PackageBuilder

	// root points to the package's root builder for sub-contexts;
	// nil on the root itself. Sub-contexts forward error
	// recording and Build to the root so accumulated state stays
	// centralised regardless of which sub-context the violation
	// originated from.
	root *PackageBuilder
}

// Package returns a fresh [PackageBuilder] seeded with an
// [emit.Package] whose Name and Path are set to the supplied values.
// Subsequent decl appends inherit the Context's target.
func (c *Context) Package(name, path string) *PackageBuilder {
	return &PackageBuilder{
		ctx: c,
		pkg: &emit.Package{
			BaseEmit: emit.BaseEmit{SetByName: c.SetBy()},
			Name:     name,
			Path:     path,
		},
	}
}

// Node returns the underlying [emit.Package]. Use this accessor to
// stamp typed metadata on the package directly, or to pass the
// pointer into a store-aware helper before [PackageBuilder.Build].
func (b *PackageBuilder) Node() *emit.Package { return b.pkg }

// Err returns the joined error from every structural violation
// recorded so far, or nil when none. Callers that want to abort the
// fluent chain on first error can probe Err between calls; callers
// that prefer to drive the chain to completion read Err once via
// [PackageBuilder.Build]. Sub-context callers see the same joined
// error as the root — every recordErr forwards to the root.
func (b *PackageBuilder) Err() error {
	if b.root != nil {
		return b.root.Err()
	}
	return errors.Join(b.errs...)
}

// File returns a PackageBuilder sub-context whose decl-spawning
// methods stamp [emit.BaseEmit.OutputTag] with tag. The
// sub-context shares the underlying [emit.Package], anchor
// default origin, and error sink with the root — decls land in
// the same package, and recorded errors surface through the
// root's [PackageBuilder.Build].
//
// Memoisation: repeated calls with the same tag return the
// same sub-context instance, so a plugin building N decls under
// `pkg.File("test")` in a loop reuses one builder rather than
// allocating one per call.
//
// `File("")` is the identity form — returns the receiver
// unchanged so plugin code that programmatically computes tags
// can write `pkg.File(maybeEmpty).<Decl>(...)` without
// special-casing the default-output case.
//
// Nested chains overwrite rather than compose:
// `pkg.File("a").File("b")` returns the sub-context tagged
// `"b"`, equivalent to `pkg.File("b")`. Nesting is not a
// supported authoring pattern — express each logical output as
// a single `pkg.File(<tag>)` call directly off the root.
func (b *PackageBuilder) File(tag string) *PackageBuilder {
	if tag == "" {
		return b
	}
	root := b.root
	if root == nil {
		root = b
	}
	if root.files == nil {
		root.files = map[string]*PackageBuilder{}
	}
	if sub, ok := root.files[tag]; ok {
		return sub
	}
	sub := &PackageBuilder{
		ctx:           root.ctx,
		pkg:           root.pkg,
		defaultOrigin: root.defaultOrigin,
		outputTag:     tag,
		root:          root,
	}
	root.files[tag] = sub
	return sub
}

// Build returns the populated [emit.Package] and the joined error
// from every structural violation recorded during construction. The
// builder remains reusable; further decl appends mutate the same
// package, which is the typical pattern for plugins that emit a
// partial package early, receive a host reference back, then append
// further decls. Calling Build on a sub-context returned by
// [PackageBuilder.File] is equivalent to calling Build on the root.
//
// The *emit.Package is returned even when error is non-nil — callers
// that want to render best-effort output on a partially valid graph
// (typical for surfacing diagnostics with attached partial output)
// can pass it through; callers that want to fail closed compare the
// error and skip the AddPackage call.
func (b *PackageBuilder) Build() (*emit.Package, error) {
	if b.root != nil {
		return b.root.Build()
	}
	return b.pkg, b.Err()
}

// Docs appends doc-comment lines preserved verbatim onto the
// package. The lines are the raw textual content without `//` or
// `/* */` markers, matching what a frontend would record.
func (b *PackageBuilder) Docs(lines ...string) *PackageBuilder {
	b.pkg.DocLines = append(b.pkg.DocLines, lines...)
	return b
}

// recordErr appends err to the package builder's accumulating error
// list. Used by sub-builders that detect structural rule violations
// during their callback (e.g. a method declaration on a true alias).
// Sub-context calls forward to the root so accumulated state stays
// centralised regardless of which sub-context the violation
// originated from.
func (b *PackageBuilder) recordErr(err error) {
	if b.root != nil {
		b.root.recordErr(err)
		return
	}
	b.errs = append(b.errs, err)
}
