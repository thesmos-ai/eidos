// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package reader

import (
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/plugins/annotator/shape"
)

// Name is the canonical shape name this detector stamps. Consumers
// switching on [shape.Get] compare against this constant rather
// than the literal string so renames surface as compile errors.
const Name = "reader"

// Detector returns the [shape.Detector] this package contributes
// to the umbrella shape plugin. Register one instance per
// pipeline:
//
//	pipe.Use(shape.New().Detectors(reader.Detector()))
func Detector() shape.Detector {
	return shape.Detector{
		Name: Name,
		Detect: map[string]shape.DetectFunc{
			"golang": detectGolang,
		},
	}
}

// detectGolang recognises the canonical Go reader signature:
// exactly one non-context input parameter, exactly one non-error
// return value, plus a trailing `error` return. The leading
// `context.Context` parameter is optional — both `(ctx, K)` and
// `(K)` forms detect.
func detectGolang(n node.Node) (shape.Match, bool) {
	params, returns := shape.GoCallable(n)
	if !shape.GoHasError(returns) {
		return shape.Match{}, false
	}
	keys := shape.GoStripContext(params)
	values := shape.GoStripError(returns)
	if len(keys) != 1 || len(values) != 1 {
		return shape.Match{}, false
	}
	return shape.Match{
		KeyType:   shape.QName(keys[0].Type),
		ValueType: shape.QName(values[0]),
	}, true
}
