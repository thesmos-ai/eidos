// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package multireader

import (
	"go.thesmos.sh/eidos/core/meta"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/plugins/annotator/shape"
)

// Name is the canonical shape name this detector stamps.
const Name = "multireader"

// ValueTypes carries the full list of non-error return types
// stamped on a positive match. The primary value also lands on
// the universal [shape.MetaValueType] for cross-shape uniformity.
//
//nolint:gochecknoglobals // registry-singleton key
var ValueTypes = meta.NewKey("shape.multireader.value_types", meta.StringListParser)

// Detector returns the [shape.Detector] this package contributes.
func Detector() shape.Detector {
	return shape.Detector{
		Name: Name,
		Detect: map[string]shape.DetectFunc{
			"golang": detectGolang,
		},
	}
}

// detectGolang accepts a callable with exactly one non-context
// parameter and two or more non-error returns followed by a
// trailing error.
func detectGolang(n node.Node) (shape.Match, bool) {
	params, returns := shape.GoCallable(n)
	keys := shape.GoStripContext(params)
	if len(keys) != 1 || !shape.GoHasError(returns) {
		return shape.Match{}, false
	}
	values := shape.GoStripError(returns)
	if len(values) < 2 {
		return shape.Match{}, false
	}
	return shape.Match{
		KeyType:   shape.QName(keys[0].Type),
		ValueType: shape.QName(values[0]),
	}, true
}
