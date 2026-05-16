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
// additionally checks that each step's compensate partner is
// unique — two steps cannot share one compensation, since the
// downstream codegen needs a one-to-one mapping for reversal.
func Contract() shape.Contract {
	return shape.Contract{
		Name:     Name,
		Roles:    Roles,
		Required: map[string][]string{"step": {"compensate"}},
		Validate: validate,
	}
}

// validate enforces saga's per-step pairing invariant. The
// Required check on `compensate` handles the missing-partner
// case; this hook adds the uniqueness check: each step must pair
// with a distinct compensate.
func validate(members map[string][]shape.ContractMember) []shape.ContractViolation {
	var out []shape.ContractViolation
	seen := make(map[string]string)
	for _, step := range members["step"] {
		comp := step.Partners["compensate"]
		if comp == "" {
			continue
		}
		if prev, exists := seen[comp]; exists {
			out = append(out, shape.ContractViolation{
				Host:    step.Host,
				Message: "saga: compensation " + comp + " is already paired with step " + prev,
			})
			continue
		}
		seen[comp] = stepLabel(step.Host)
	}
	return out
}

// stepLabel returns a human-readable identifier for a saga step
// host — function or method name — for inclusion in diagnostic
// messages. Returns the empty string for any other node kind.
func stepLabel(n node.Node) string {
	switch x := n.(type) {
	case *node.Function:
		return x.Name
	case *node.Method:
		return x.Name
	}
	return ""
}
