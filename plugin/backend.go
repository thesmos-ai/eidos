// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package plugin

import (
	"go.thesmos.sh/eidos/core/diag"
	"go.thesmos.sh/eidos/sink"
	"go.thesmos.sh/eidos/store"
)

// Backend is the role for plugins that render emit entities to a
// target language and write the result through a [sink.Sink].
// Exactly one Backend runs per pipeline (the builder rejects zero
// or multiple Backends).
//
// Render groups emit entities by their [emit.Target], renders one
// file per group through the language-specific templates, finalises
// the output (import resolution, formatting), and writes via
// ctx.Sink. Per-file issues attach to ctx.Diag; fatal failures
// return a non-nil error.
type Backend interface {
	Plugin

	// Language returns the target-language identifier — the same
	// string the [TemplateProvider] interface uses to scope
	// templates and func-map extensions ("go", "rust", "ts", …).
	Language() string

	// Render renders the emit entities in ctx.Store.Emit() and
	// writes finalised file content through ctx.Sink.
	Render(ctx *BackendContext) error
}

// BackendContext is the per-run context handed to the [Backend]. It
// supplies the shared store, the diagnostic sink, the destination
// sink, the language identifier (so plugin-supplied template /
// func-map extensions can target it), and the ordered list of
// plugins that participated in the run (so the backend can collect
// template providers and apply overrides in capability topo order).
type BackendContext struct {
	// Store is the shared in-memory database. The backend reads
	// from Store.Emit().
	Store *store.Store

	// Reader is the per-plugin read-tracking handle.
	Reader *store.Reader

	// Diag is the diagnostic sink shared with every plugin in the
	// run.
	Diag *diag.Sink

	// Sink is the destination the backend writes finalised file
	// content through.
	Sink sink.Sink

	// Lang is the target-language identifier the backend renders
	// for. Equal to the value [Backend.Language] returns; mirrored
	// here so plugin-supplied template providers can target it
	// without consulting the Backend implementation directly.
	Lang string

	// Plugins is the full list of plugins that participated in the
	// run, in the order the user registered them. The backend uses
	// this to find every [TemplateProvider] for template merging.
	Plugins []Plugin

	// Ordered is the same plugins as [Plugins], reordered by the
	// pipeline's resolved capability topo. The backend uses this to
	// apply [TemplateProvider.TemplateOverrides] in deterministic
	// order so collisions resolve last-write-wins predictably.
	Ordered []Plugin
}
