// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package diag

import (
	"fmt"

	"go.thesmos.sh/eidos/core/position"
)

// PluginSink is a [Sink] view that stamps a fixed plugin name on every
// diagnostic it emits. Plugin authors obtain one via [Sink.For] and
// then call its methods throughout the plugin's execution; the name is
// preserved verbatim and never re-derived from the call stack.
//
// PluginSink is safe to share across goroutines for the duration of a
// single plugin invocation; the underlying [Sink] handles concurrency.
type PluginSink struct {
	sink   *Sink
	plugin string
}

// Plugin returns the plugin name this PluginSink stamps on every
// diagnostic it records.
func (p *PluginSink) Plugin() string { return p.plugin }

// Errorf records an Error diagnostic attributed to the plugin.
func (p *PluginSink) Errorf(pos position.Pos, format string, args ...any) {
	p.sink.Append(Diag{
		Severity: Error,
		Plugin:   p.plugin,
		Pos:      pos,
		Message:  fmt.Sprintf(format, args...),
	})
}

// Warnf records a Warn diagnostic attributed to the plugin.
func (p *PluginSink) Warnf(pos position.Pos, format string, args ...any) {
	p.sink.Append(Diag{
		Severity: Warn,
		Plugin:   p.plugin,
		Pos:      pos,
		Message:  fmt.Sprintf(format, args...),
	})
}

// Infof records an Info diagnostic attributed to the plugin. Info is
// hidden by default and surfaced via the --verbose flag.
func (p *PluginSink) Infof(pos position.Pos, format string, args ...any) {
	p.sink.Append(Diag{
		Severity: Info,
		Plugin:   p.plugin,
		Pos:      pos,
		Message:  fmt.Sprintf(format, args...),
	})
}

// AppendDetail records a diagnostic attributed to the plugin with both
// a primary message and multi-line detail context. Used when the
// failure context (expected vs actual values, stack traces) does not
// fit into a single line.
func (p *PluginSink) AppendDetail(sev Severity, pos position.Pos, message, detail string) {
	p.sink.Append(Diag{
		Severity: sev,
		Plugin:   p.plugin,
		Pos:      pos,
		Message:  message,
		Detail:   detail,
	})
}
