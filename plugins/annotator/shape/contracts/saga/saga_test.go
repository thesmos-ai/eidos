// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package saga_test

import (
	"testing"

	"go.thesmos.sh/eidos/core/diag"
	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/plugins/annotator/shape"
	"go.thesmos.sh/eidos/plugins/annotator/shape/contracts/internal/contracttest"
	"go.thesmos.sh/eidos/plugins/annotator/shape/contracts/saga"
)

func TestContract_Identity(t *testing.T) {
	t.Parallel()
	contracttest.AssertIdentity(t,
		saga.Contract(),
		saga.Name, saga.Roles)
}

// TestContract_Validate exercises the saga validator hook
// directly — accepts distinct compensations, flags shared
// compensations, and survives non-callable hosts via the
// stepLabel default branch.
func TestContract_Validate(t *testing.T) {
	t.Parallel()
	c := saga.Contract()

	t.Run("accepts distinct compensations", func(t *testing.T) {
		t.Parallel()
		members := map[string][]shape.ContractMember{
			"step": {
				{Host: &node.Function{Name: "Charge"}, Partners: map[string]string{"compensate": "x.Refund"}},
				{Host: &node.Function{Name: "Ship"}, Partners: map[string]string{"compensate": "x.Unship"}},
			},
			"compensate": {
				{Host: &node.Function{Name: "Refund"}},
				{Host: &node.Function{Name: "Unship"}},
			},
		}
		if got := c.Validate(members); len(got) != 0 {
			t.Fatalf("Validate(distinct) = %+v; want no violations", got)
		}
	})

	t.Run("handles non-callable host via stepLabel fallback", func(t *testing.T) {
		t.Parallel()
		// First member is a [*node.Struct] — neither Function
		// nor Method, so stepLabel returns the empty string for
		// the seen-map entry. The second step (a Function) then
		// tries to register the same compensate qname and gets
		// flagged.
		members := map[string][]shape.ContractMember{
			"step": {
				{Host: &node.Struct{Name: "S"}, Partners: map[string]string{"compensate": "x.A"}},
				{Host: &node.Function{Name: "Other"}, Partners: map[string]string{"compensate": "x.A"}},
			},
		}
		got := c.Validate(members)
		if len(got) != 1 {
			t.Fatalf("Validate(non-callable + dup) = %+v; want one violation on the second step", got)
		}
	})
}

// TestContract_PipelineRoundTrip exercises the happy path of one
// step + one compensate through umbrella → resolver → validator.
func TestContract_PipelineRoundTrip(t *testing.T) {
	t.Parallel()
	step := &node.Function{
		Name: "Charge", Package: "x",
		BaseNode: node.BaseNode{
			DirectiveList: []*directive.Directive{
				contracttest.HostDirective(saga.Name, "step", map[string]string{
					"compensate": "Refund",
				}),
			},
		},
	}
	refund := &node.Function{Name: "Refund", Package: "x"}
	pkg := &node.Package{
		Name: "x", Path: "x",
		Functions: []*node.Function{step, refund},
	}
	diags := contracttest.RunPipeline(t, saga.Contract(), pkg)
	contracttest.AssertNoErrorDiag(t, diags)

	contracttest.AssertRole(t, step.Meta(), saga.Name, "step")
	contracttest.AssertPartner(t, step.Meta(), saga.Name, "compensate", "x.Refund")
	contracttest.AssertRole(t, refund.Meta(), saga.Name, "compensate")
}

// TestContract_ValidatorFlagsSharedCompensate exercises the
// Validate hook through the actual [shape.Validator] annotator
// — two steps pointing at the same compensate produces a
// "already paired with step" diagnostic the validator must
// surface.
func TestContract_ValidatorFlagsSharedCompensate(t *testing.T) {
	t.Parallel()
	stepA := &node.Function{
		Name: "Charge", Package: "x",
		BaseNode: node.BaseNode{
			DirectiveList: []*directive.Directive{
				contracttest.HostDirective(saga.Name, "step", map[string]string{
					"compensate": "Refund",
				}),
			},
		},
	}
	stepB := &node.Function{
		Name: "Ship", Package: "x",
		BaseNode: node.BaseNode{
			DirectiveList: []*directive.Directive{
				contracttest.HostDirective(saga.Name, "step", map[string]string{
					"compensate": "Refund",
				}),
			},
		},
	}
	refund := &node.Function{Name: "Refund", Package: "x"}
	pkg := &node.Package{
		Name: "x", Path: "x",
		Functions: []*node.Function{stepA, stepB, refund},
	}
	diags := contracttest.RunPipeline(t, saga.Contract(), pkg)
	contracttest.AssertContainsDiag(t, diags, diag.Error, "already paired")
}
