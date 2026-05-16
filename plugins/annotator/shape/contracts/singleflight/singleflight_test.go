// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package singleflight_test

import (
	"testing"

	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/plugins/annotator/shape/contracts/internal/contracttest"
	"go.thesmos.sh/eidos/plugins/annotator/shape/contracts/singleflight"
)

func TestContract_Identity(t *testing.T) {
	t.Parallel()
	contracttest.AssertIdentity(t,
		singleflight.Contract(),
		singleflight.Name, singleflight.Roles)
}

func TestContract_PipelineRoundTrip(t *testing.T) {
	t.Parallel()
	fn := &node.Function{
		Name: "Fetch", Package: "x",
		BaseNode: node.BaseNode{
			DirectiveList: []*directive.Directive{
				contracttest.HostDirective(singleflight.Name, "fn", nil),
			},
		},
	}
	pkg := &node.Package{Name: "x", Path: "x", Functions: []*node.Function{fn}}
	diags := contracttest.RunPipeline(t, singleflight.Contract(), pkg)

	contracttest.AssertRole(t, fn.Meta(), singleflight.Name, "fn")
	contracttest.AssertNoErrorDiag(t, diags)
}
