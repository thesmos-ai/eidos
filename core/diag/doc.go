// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package diag aggregates positioned diagnostics emitted by every layer
// of the pipeline into one consistent collector.
//
// Diagnostic anatomy: a [Severity] (Info, Warn, Error, Internal), an
// optional [position.Pos], the producing plugin name, a primary message,
// and optional detail. The producing plugin attaches its name once via
// [Sink.For]; the returned [PluginSink] then stamps the name on every
// diagnostic the plugin emits.
//
// A [Sink] is concurrent-safe and append-only; the zero value is unusable
// (use [New]). Diagnostics survive in insertion order so renderings are
// deterministic across runs.
//
// Plugin code never panics user-visibly: [RecoverAs] is the canonical
// deferred guard inside any plugin invocation. A recovered panic is
// converted into an Error diagnostic carrying the panic value plus the
// captured stack trace; subsequent plugins continue to run. Internal
// panics (in core packages) emit Internal severity instead, which the
// CLI maps to a distinct exit code.
//
// Output is decoupled from collection: a [Formatter] takes a []Diag and
// renders it for a target audience. [TextFormatter] is the human-facing
// terminal renderer; [JSONFormatter] emits one diagnostic per line plus
// a final aggregate object for CI / IDE consumption.
package diag
