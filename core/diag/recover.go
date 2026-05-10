// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package diag

import (
	"fmt"
	"runtime/debug"

	"go.thesmos.sh/eidos/core/position"
)

// RecoverAs is intended to be deferred at the top of any plugin
// invocation. If the surrounding goroutine panics, the panic value is
// converted to a diagnostic emitted by the supplied PluginSink:
//
//   - panics in user plugins are recorded at Error severity (user-side
//     bug; pipeline still completes the remaining plugins);
//   - panics tagged Internal severity via [RecoverInternal] are
//     reserved for core packages and contribute to a distinct exit
//     code.
//
// The recovered stack trace is captured via [runtime/debug.Stack] and
// stored verbatim in the diagnostic's Detail field. The panic itself
// is suppressed; callers that need to short-circuit further work in
// the same goroutine should observe [Sink.HasErrors] after the
// deferred call has run.
//
// Usage:
//
//	func runAnnotator(s *diag.Sink, name string, fn func()) {
//	    defer diag.RecoverAs(s.For(name), position.Pos{})
//	    fn()
//	}
//
// pos is the best-known position at the moment of invocation (often
// the zero value when the panic site is unknown). When the panic value
// is nil — i.e. the goroutine did not actually panic — RecoverAs is a
// no-op.
func RecoverAs(p *PluginSink, pos position.Pos) {
	r := recover()
	if r == nil {
		return
	}
	p.AppendDetail(
		Error,
		pos,
		fmt.Sprintf("plugin %q panicked: %v", p.Plugin(), r),
		string(debug.Stack()),
	)
}

// RecoverInternal is the same as [RecoverAs] but records the resulting
// diagnostic at Internal severity. Reserved for guards inside core
// packages where a panic indicates a framework bug rather than a
// plugin's fault.
func RecoverInternal(s *Sink, pos position.Pos, location string) {
	r := recover()
	if r == nil {
		return
	}
	s.Append(Diag{
		Severity: Internal,
		Pos:      pos,
		Message:  fmt.Sprintf("internal panic in %s: %v", location, r),
		Detail:   string(debug.Stack()),
	})
}
