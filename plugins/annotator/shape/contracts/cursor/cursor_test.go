// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package cursor_test

import (
	"testing"

	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/plugins/annotator/shape/contracts/cursor"
	"go.thesmos.sh/eidos/plugins/annotator/shape/contracts/internal/contracttest"
)

func TestContract_Identity(t *testing.T) {
	t.Parallel()
	contracttest.AssertIdentity(t,
		cursor.Contract(),
		cursor.Name, cursor.Roles)
}

func TestContract_PipelineRoundTrip(t *testing.T) {
	t.Parallel()
	next := &node.Function{
		Name: "Next", Package: "x",
		BaseNode: node.BaseNode{
			DirectiveList: []*directive.Directive{
				contracttest.HostDirective(cursor.Name, "next", map[string]string{
					"close": "Close",
				}),
			},
		},
	}
	closeFn := &node.Function{Name: "Close", Package: "x"}
	pkg := &node.Package{
		Name: "x", Path: "x",
		Functions: []*node.Function{next, closeFn},
	}
	diags := contracttest.RunPipeline(t, cursor.Contract(), pkg)
	contracttest.AssertNoErrorDiag(t, diags)

	contracttest.AssertRole(t, next.Meta(), cursor.Name, "next")
	contracttest.AssertPartner(t, next.Meta(), cursor.Name, "close", "x.Close")
	contracttest.AssertRole(t, closeFn.Meta(), cursor.Name, "close")
	contracttest.AssertPartner(t, closeFn.Meta(), cursor.Name, "next", "x.Next")
}
