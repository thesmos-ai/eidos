// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package voidlifecycle

import (
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/plugins/annotator/shape"
)

// Name is the canonical shape name this detector stamps.
const Name = "voidlifecycle"

// Detector returns the [shape.Detector] this package contributes.
func Detector() shape.Detector {
	return shape.Detector{
		Name: Name,
		Detect: map[string]shape.DetectFunc{
			"golang": detectGolang,
		},
	}
}

// detectGolang accepts the bare `func ()` shape: no parameters,
// no return values.
func detectGolang(n node.Node) (shape.Match, bool) {
	params, returns := shape.GoCallable(n)
	if len(params) != 0 || len(returns) != 0 {
		return shape.Match{}, false
	}
	return shape.Match{}, true
}
