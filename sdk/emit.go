// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package sdk

import (
	"go.thesmos.sh/eidos/emit"
	emitbuilder "go.thesmos.sh/eidos/emit/builder"
)

// BaseEmit is the canonical [emit.BaseEmit] every plugin
// embeds in its plugin-defined emit values. Re-exported here
// so plugin authors only need a single `sdk` import for the
// common emit-value plumbing.
type BaseEmit = emit.BaseEmit

// EmitNode is the [emit.Node] interface plugin-defined emit
// values satisfy. The conventional `var _ EmitNode = (*MyValue)(nil)`
// compile-time assertion uses this alias instead of reaching
// into the emit package directly.
type EmitNode = emit.Node

// EmitTarget is the [emit.Target] descriptor the routing
// layer composes for each emit value. Plugins typically
// construct a zero value (`sdk.EmitTarget{}`) and let the
// Layout phase fill in [emit.Target.Dir] / [emit.Target.Filename]
// / [emit.Target.Package] / [emit.Target.ImportPath] from
// the contribution's origin + project / per-plugin / CLI
// overrides.
type EmitTarget = emit.Target

// Ref is the [emit.Ref] interface every type-side reference
// satisfies — [emit.BuiltinRef], [emit.ExternalRef], the
// composite shapes. Used as the parameter / return type for
// helpers that hand templates type-ref expressions to render
// through `renderType`.
type Ref = emit.Ref

// Expr is the [emit.Expr] type every value-side / callable
// reference is wrapped in. Used as the parameter / return
// type for helpers that hand templates callable expressions
// to render through `renderExpr`.
type Expr = emit.Expr

// NewExternal re-exports [emit.NewExternal] — the factory
// for fully-qualified package + name expressions the Go
// backend's `renderExpr` registers the import for
// automatically.
//
//nolint:gochecknoglobals // alias re-export of a stable factory.
var NewExternal = emit.NewExternal

// Provenance is the per-contribution provenance context
// returned by [NewProvenance]. Plugins call its `SetBy()` /
// `Provenance(id…)` methods to thread `set-by` /
// per-contribution-ID metadata onto each appended slot
// contribution.
type Provenance = emitbuilder.Context

// NewProvenance returns a fresh per-plugin provenance
// builder bound to setBy (the plugin's stable identifier)
// and the supplied default [EmitTarget]. Plugins typically
// pass the zero target — the Layout phase composes the real
// target from the contribution's origin.
//
//nolint:gochecknoglobals // alias re-export of a stable factory.
var NewProvenance = emitbuilder.For
