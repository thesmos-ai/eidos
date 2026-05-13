// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package extras

// Tag mirrors the cross-package proto message
// `eidos.test.buildfixture.extras.Tag`. The bridge translates
// references to it through the parent fixture's `go_package`
// option; the Go-side stub gives the rendered references an
// existing target.
type Tag struct {
	Name string
}
