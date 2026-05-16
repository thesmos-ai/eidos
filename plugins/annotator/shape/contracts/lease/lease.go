// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package lease

import "go.thesmos.sh/eidos/plugins/annotator/shape"

// Name is the canonical contract name this package stamps.
const Name = "lease"

// Roles enumerates the contract's role vocabulary.
//
//nolint:gochecknoglobals // intentionally exported as a per-contract constant set
var Roles = []string{"acquire", "release"}

// Contract returns the [shape.Contract] this package contributes.
// The acquire side requires a release partner.
func Contract() shape.Contract {
	return shape.Contract{
		Name:     Name,
		Roles:    Roles,
		Required: map[string][]string{"acquire": {"release"}},
	}
}
