// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package transaction_test

import (
	"testing"

	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/plugins/annotator/shape/contracts/internal/contracttest"
	"go.thesmos.sh/eidos/plugins/annotator/shape/contracts/transaction"
)

func TestContract_Identity(t *testing.T) {
	t.Parallel()
	contracttest.AssertIdentity(t,
		transaction.Contract(),
		transaction.Name, transaction.Roles)
}

// TestContract_PipelineRoundTrip exercises the umbrella →
// resolver → validator sequence for a real `+gen:contract
// transaction` directive — proving the package's [Contract]
// value plugs into the framework correctly.
func TestContract_PipelineRoundTrip(t *testing.T) {
	t.Parallel()
	fn := &node.Function{
		Name: "Charge", Package: "x",
		BaseNode: node.BaseNode{
			DirectiveList: []*directive.Directive{
				contracttest.HostDirective(transaction.Name, "fn", nil),
			},
		},
	}
	pkg := &node.Package{Name: "x", Path: "x", Functions: []*node.Function{fn}}
	diags := contracttest.RunPipeline(t, transaction.Contract(), pkg)

	contracttest.AssertRole(t, fn.Meta(), transaction.Name, "fn")
	contracttest.AssertNoErrorDiag(t, diags)
}
