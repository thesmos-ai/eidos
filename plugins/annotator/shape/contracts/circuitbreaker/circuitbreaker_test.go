// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package circuitbreaker_test

import (
	"testing"

	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/plugins/annotator/shape/contracts/circuitbreaker"
	"go.thesmos.sh/eidos/plugins/annotator/shape/contracts/internal/contracttest"
)

func TestContract(t *testing.T) {
	t.Parallel()

	t.Run("identity", func(t *testing.T) {
		t.Parallel()
		contracttest.AssertIdentity(t, circuitbreaker.Contract(), circuitbreaker.Name, circuitbreaker.Roles)
	})

	t.Run("pipeline stamps role", func(t *testing.T) {
		t.Parallel()
		fn := &node.Function{
			Name: "Call", Package: "x",
			BaseNode: node.BaseNode{
				DirectiveList: []*directive.Directive{
					contracttest.HostDirective(circuitbreaker.Name, "fn", nil),
				},
			},
		}
		pkg := &node.Package{Name: "x", Path: "x", Functions: []*node.Function{fn}}
		diags := contracttest.RunPipeline(t, circuitbreaker.Contract(), pkg)
		contracttest.AssertNoErrorDiag(t, diags)
		contracttest.AssertRole(t, fn.Meta(), circuitbreaker.Name, "fn")
	})
}
