// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package writethroughcache

import "go.thesmos.sh/eidos/plugins/annotator/shape"

// Name is the canonical contract name this package stamps.
const Name = "cache"

// Roles enumerates the contract's role vocabulary.
//
//nolint:gochecknoglobals // intentionally exported as a per-contract constant set
var Roles = []string{"cache", "backing"}

// Contract returns the [shape.Contract] this package contributes.
// The cache role requires a backing partner so cache misses can
// fall through.
func Contract() shape.Contract {
	return shape.Contract{
		Name:     Name,
		Roles:    Roles,
		Required: map[string][]string{"cache": {"backing"}},
	}
}
