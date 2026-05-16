// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package saga

import (
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/plugins/annotator/shape"
)

// Name is the canonical contract name this package stamps.
const Name = "saga"

// Roles enumerates the contract's role vocabulary.
//
//nolint:gochecknoglobals // intentionally exported as a per-contract constant set
var Roles = []string{"step", "compensate"}

// Contract returns the [shape.Contract] this package contributes.
// Every step requires a compensate partner; the [Validate] hook
// additionally checks that every step has a paired compensation
// after the resolver has run.
func Contract() shape.Contract {
	return shape.Contract{
		Name:     Name,
		Roles:    Roles,
		Required: map[string][]string{"step": {"compensate"}},
		Validate: validate,
	}
}

// validate enforces the saga structural invariant: the number of
// resolved compensate members must match the number of step
// members so every step has a paired compensation.
func validate(members map[string][]node.Node) []shape.ContractViolation {
	steps := len(members["step"])
	comps := len(members["compensate"])
	if steps == comps {
		return nil
	}
	if steps == 0 {
		return nil
	}
	host := members["step"][0]
	return []shape.ContractViolation{
		{
			Host:    host,
			Message: "saga: step / compensate count mismatch",
		},
	}
}
