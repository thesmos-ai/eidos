// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package upserter_test

import (
	"testing"

	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/plugins/annotator/shape/contracts/internal/contracttest"
	"go.thesmos.sh/eidos/plugins/annotator/shape/contracts/upserter"
)

func TestContract_Identity(t *testing.T) {
	t.Parallel()
	contracttest.AssertIdentity(t, upserter.Contract(), upserter.Name, upserter.Roles)
}

func TestContract_PipelineRoundTrip(t *testing.T) {
	t.Parallel()
	put := &node.Function{
		Name: "Put", Package: "x",
		BaseNode: node.BaseNode{
			DirectiveList: []*directive.Directive{
				contracttest.HostDirective(upserter.Name, "writer", map[string]string{
					"reader": "GetByID",
				}),
			},
		},
	}
	get := &node.Function{Name: "GetByID", Package: "x"}
	pkg := &node.Package{Name: "x", Path: "x", Functions: []*node.Function{put, get}}
	diags := contracttest.RunPipeline(t, upserter.Contract(), pkg)
	contracttest.AssertNoErrorDiag(t, diags)

	contracttest.AssertRole(t, put.Meta(), upserter.Name, "writer")
	contracttest.AssertPartner(t, put.Meta(), upserter.Name, "reader", "x.GetByID")
	contracttest.AssertRole(t, get.Meta(), upserter.Name, "reader")
}
