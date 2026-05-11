// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package diag

// Capture returns a fresh [Sink] that records every emitted
// diagnostic for later inspection. It is a thin alias for [New]
// chosen for read-grep clarity in test code:
//
//	d := diag.Capture()
//	p.Run(t.Context())
//	if d.HasErrors() { … }
//
// Use Capture when the test asserts on emitted diagnostics. Use
// [Discard] when the test exercises plugin code through the diag
// surface but does not care about the recorded output.
func Capture() *Sink { return New() }

// Discard returns a [Sink] that drops every emitted diagnostic
// without recording it. Append, Errorf, Warnf, Infof, and the
// equivalents on [PluginSink] become no-ops; HasErrors always
// reports false; Diagnostics always returns nil; Count always
// returns 0.
//
// Use Discard in unit tests that drive plugin code through the
// diag surface but make no assertions about what was emitted.
func Discard() *Sink {
	s := New()
	s.discard = true
	return s
}
