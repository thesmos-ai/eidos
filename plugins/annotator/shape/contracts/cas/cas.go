// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package cas

import "go.thesmos.sh/eidos/plugins/annotator/shape"

// Name is the canonical contract name this package stamps.
const Name = "cas"

// Roles enumerates the contract's role vocabulary.
//
//nolint:gochecknoglobals // intentionally exported as a per-contract constant set
var Roles = []string{"writer"}

// Params enumerates the directive's opaque KV keys.
//
//nolint:gochecknoglobals // intentionally exported as a per-contract constant set
var Params = []string{"version"}

// Contract returns the [shape.Contract] this package contributes.
// The `version` KV is an opaque field-name reference — the
// resolver never tries to look it up as a sibling callable.
func Contract() shape.Contract {
	return shape.Contract{Name: Name, Roles: Roles, Params: Params}
}
