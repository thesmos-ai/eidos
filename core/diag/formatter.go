// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package diag

import "io"

// Formatter renders a slice of diagnostics for a target audience.
//
// Implementations decide layout and content, but each should be
// deterministic — the same input slice must produce byte-identical
// output across runs — and should not retain references to the input
// after Format returns.
//
// The returned error is the first IO error encountered while writing
// to w; partial output up to that point may already have been emitted.
type Formatter interface {
	Format(w io.Writer, diags []Diag) error
}
