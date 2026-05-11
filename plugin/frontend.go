// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package plugin

import (
	"go.thesmos.sh/eidos/core/diag"
	"go.thesmos.sh/eidos/store"
)

// Frontend is the role for plugins that parse input into the source
// side of the [store.Store]. The pipeline runs every Frontend in
// the frontend phase; multiple frontends may coexist (e.g. a Go
// frontend alongside an OpenAPI schema frontend), and their loaded
// packages share the same store.
//
// Load is called once per pipeline run with the user-supplied
// pattern (typically a Go-style import path or filesystem glob).
// Errors that prevent the frontend from making any progress are
// returned directly; per-input issues are emitted as positioned
// diagnostics on diag and execution continues.
type Frontend interface {
	Plugin

	// Load parses the input identified by pattern and records the
	// resulting nodes in store.Nodes(). Per-input issues attach to
	// diag; fatal failures return a non-nil error.
	Load(pattern string, store *store.Store, diag *diag.Sink) error
}
