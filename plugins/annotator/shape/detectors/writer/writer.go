// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package writer

import (
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/plugins/annotator/shape"
)

// Name is the canonical shape name this detector stamps. Consumers
// switching on [shape.Get] compare against this constant rather
// than the literal string so renames surface as compile errors.
const Name = "writer"

// Detector returns the [shape.Detector] this package contributes
// to the umbrella shape plugin. Register one instance per
// pipeline:
//
//	pipe.Use(shape.New().Detectors(writer.Detector()))
func Detector() shape.Detector {
	return shape.Detector{
		Name:     Name,
		Priority: 500,
		Detect: map[string]shape.DetectFunc{
			"golang": detectGolang,
		},
	}
}

// detectGolang recognises the canonical Go writer signature:
// exactly one non-context input parameter, a trailing `error`
// return, and either zero or one additional non-error return
// (the with-result variant). The leading `context.Context`
// parameter is optional.
func detectGolang(n node.Node) (shape.Match, bool) {
	params, returns := shape.GoCallable(n)
	if !shape.GoHasError(returns) {
		return shape.Match{}, false
	}
	values := shape.GoStripContext(params)
	results := shape.GoStripError(returns)
	if len(values) != 1 || len(results) > 1 {
		return shape.Match{}, false
	}
	return shape.Match{
		ValueType: shape.QName(values[0].Type),
	}, true
}
