// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package builder

import (
	"errors"

	"go.thesmos.sh/eidos/emit"
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
// Errors are accumulated rather than returned per call so the
// fluent chain stays unbroken; [PackageBuilder.Build] surfaces them
// in one [errors.Join] result. Callers that want to abort the chain
// early can inspect [PackageBuilder.Err] between calls.
type PackageBuilder struct {
	ctx  *Context
	pkg  *emit.Package
	errs []error
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
// [PackageBuilder.Build].
func (b *PackageBuilder) Err() error { return errors.Join(b.errs...) }

// Build returns the populated [emit.Package] and the joined error
// from every structural violation recorded during construction. The
// builder remains reusable; further decl appends mutate the same
// package, which is the typical pattern for plugins that emit a
// partial package early, receive a host reference back, then append
// further decls.
//
// The *emit.Package is returned even when error is non-nil — callers
// that want to render best-effort output on a partially valid graph
// (typical for surfacing diagnostics with attached partial output)
// can pass it through; callers that want to fail closed compare the
// error and skip the AddPackage call.
func (b *PackageBuilder) Build() (*emit.Package, error) {
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
func (b *PackageBuilder) recordErr(err error) { b.errs = append(b.errs, err) }
