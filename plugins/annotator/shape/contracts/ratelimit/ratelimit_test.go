// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package ratelimit_test

import (
	"testing"

	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/plugins/annotator/shape"
	"go.thesmos.sh/eidos/plugins/annotator/shape/contracts/internal/contracttest"
	"go.thesmos.sh/eidos/plugins/annotator/shape/contracts/ratelimit"
)

func TestContract(t *testing.T) {
	t.Parallel()

	t.Run("identity", func(t *testing.T) {
		t.Parallel()
		contracttest.AssertIdentity(t, ratelimit.Contract(), ratelimit.Name, ratelimit.Roles)
	})

	t.Run("pipeline stamps rate and burst params", func(t *testing.T) {
		t.Parallel()
		fn := &node.Function{
			Name: "Call", Package: "x",
			BaseNode: node.BaseNode{
				DirectiveList: []*directive.Directive{
					contracttest.HostDirective(ratelimit.Name, "fn", map[string]string{
						"rate":  "100",
						"burst": "10",
					}),
				},
			},
		}
		pkg := &node.Package{Name: "x", Path: "x", Functions: []*node.Function{fn}}
		diags := contracttest.RunPipeline(t, ratelimit.Contract(), pkg)
		contracttest.AssertNoErrorDiag(t, diags)

		if got, _ := shape.ContractParamKey(ratelimit.Name, "rate").Get(fn.Meta()); got != "100" {
			t.Fatalf("rate = %q, want %q", got, "100")
		}
		if got, _ := shape.ContractParamKey(ratelimit.Name, "burst").Get(fn.Meta()); got != "10" {
			t.Fatalf("burst = %q, want %q", got, "10")
		}
	})
}
