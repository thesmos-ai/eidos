// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package bounded

import "go.thesmos.sh/eidos/plugins/annotator/shape"

// Name is the canonical mixin name this package stamps.
const Name = "bounded"

// Params enumerates the KV parameter names this mixin accepts.
//
//nolint:gochecknoglobals // intentionally exported as a per-mixin constant set
var Params = []string{"limit"}

// Mixin returns the [shape.Mixin] this package contributes.
func Mixin() shape.Mixin {
	return shape.Mixin{Name: Name, Params: Params}
}
