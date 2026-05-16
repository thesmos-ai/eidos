// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package saga_test

import (
	"testing"

	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/plugins/annotator/shape/contracts/internal/contracttest"
	"go.thesmos.sh/eidos/plugins/annotator/shape/contracts/saga"
)

func TestContract_Identity(t *testing.T) {
	t.Parallel()
	contracttest.AssertIdentity(t,
		saga.Contract(),
		saga.Name, saga.Roles)
}

func TestContract_ValidateAcceptsPairedSteps(t *testing.T) {
	t.Parallel()
	c := saga.Contract()
	members := map[string][]node.Node{
		"step": {&node.Function{Name: "Charge"}, &node.Function{Name: "Ship"}},
		"compensate": {
			&node.Function{Name: "Refund"},
			&node.Function{Name: "Unship"},
		},
	}
	if got := c.Validate(members); len(got) != 0 {
		t.Fatalf("Validate(paired) = %+v; want no violations", got)
	}
}

func TestContract_ValidateFlagsCountMismatch(t *testing.T) {
	t.Parallel()
	c := saga.Contract()
	members := map[string][]node.Node{
		"step":       {&node.Function{Name: "Charge"}, &node.Function{Name: "Ship"}},
		"compensate": {&node.Function{Name: "Refund"}},
	}
	got := c.Validate(members)
	if len(got) != 1 {
		t.Fatalf("Validate(mismatch) = %+v; want one violation", got)
	}
}
