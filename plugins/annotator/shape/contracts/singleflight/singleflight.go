// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package singleflight

import "go.thesmos.sh/eidos/plugins/annotator/shape"

// Name is the canonical contract name this package stamps.
const Name = "singleflight"

// Roles enumerates the contract's role vocabulary — a single
// "fn" role since the contract is a per-callable marker.
//
//nolint:gochecknoglobals // intentionally exported as a per-contract constant set
var Roles = []string{"fn"}

// Contract returns the [shape.Contract] this package contributes.
func Contract() shape.Contract {
	return shape.Contract{Name: Name, Roles: Roles}
}
