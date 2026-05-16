// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package saga_test

import (
	"testing"

	"go.thesmos.sh/eidos/core/diag"
	"go.thesmos.sh/eidos/core/directive"
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

// TestContract_ValidatorFlagsCountMismatch exercises the Validate
// hook through the actual [shape.Validator] annotator — two
// steps both pointing at the same compensate produces a 2-vs-1
// count mismatch the validator must surface.
func TestContract_ValidatorFlagsCountMismatch(t *testing.T) {
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
	contracttest.AssertContainsDiag(t, diags, diag.Error, "count mismatch")
}
