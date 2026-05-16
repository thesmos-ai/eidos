// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package tx

import "go.thesmos.sh/eidos/plugins/annotator/shape"

// Name is the canonical contract name this package stamps.
const Name = "tx"

// Roles enumerates the contract's role vocabulary.
//
//nolint:gochecknoglobals // intentionally exported as a per-contract constant set
var Roles = []string{"begin", "commit", "rollback"}

// Contract returns the [shape.Contract] this package contributes.
// Begin requires both Commit and Rollback partners — the
// validator flags any Begin declaration missing either side.
func Contract() shape.Contract {
	return shape.Contract{
		Name:     Name,
		Roles:    Roles,
		Required: map[string][]string{"begin": {"commit", "rollback"}},
	}
}
