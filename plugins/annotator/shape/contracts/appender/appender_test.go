// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package appender_test

import (
	"testing"

	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/plugins/annotator/shape/contracts/appender"
	"go.thesmos.sh/eidos/plugins/annotator/shape/contracts/internal/contracttest"
)

func TestContract_Identity(t *testing.T) {
	t.Parallel()
	contracttest.AssertIdentity(t,
		appender.Contract(),
		appender.Name, appender.Roles)
}

func TestContract_PipelineRoundTrip(t *testing.T) {
	t.Parallel()
	fn := &node.Function{
		Name: "Append", Package: "x",
		BaseNode: node.BaseNode{
			DirectiveList: []*directive.Directive{
				contracttest.HostDirective(appender.Name, "fn", nil),
			},
		},
	}
	pkg := &node.Package{Name: "x", Path: "x", Functions: []*node.Function{fn}}
	diags := contracttest.RunPipeline(t, appender.Contract(), pkg)

	contracttest.AssertRole(t, fn.Meta(), appender.Name, "fn")
	contracttest.AssertNoErrorDiag(t, diags)
}
