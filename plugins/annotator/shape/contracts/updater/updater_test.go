// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package updater_test

import (
	"testing"

	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/plugins/annotator/shape/contracts/internal/contracttest"
	"go.thesmos.sh/eidos/plugins/annotator/shape/contracts/updater"
)

func TestContract_Identity(t *testing.T) {
	t.Parallel()
	contracttest.AssertIdentity(t,
		updater.Contract(),
		updater.Name, updater.Roles)
}

// TestContract_PipelineRoundTrip exercises both the writer-side
// stamping and the resolver's back-stamp on the reader sibling.
func TestContract_PipelineRoundTrip(t *testing.T) {
	t.Parallel()
	save := &node.Function{
		Name: "Save", Package: "x",
		BaseNode: node.BaseNode{
			DirectiveList: []*directive.Directive{
				contracttest.HostDirective(updater.Name, "writer", map[string]string{
					"reader": "GetByID",
				}),
			},
		},
	}
	get := &node.Function{Name: "GetByID", Package: "x"}
	pkg := &node.Package{
		Name: "x", Path: "x",
		Functions: []*node.Function{save, get},
	}
	diags := contracttest.RunPipeline(t, updater.Contract(), pkg)
	contracttest.AssertNoErrorDiag(t, diags)

	contracttest.AssertRole(t, save.Meta(), updater.Name, "writer")
	contracttest.AssertPartner(t, save.Meta(), updater.Name, "reader", "x.GetByID")
	contracttest.AssertRole(t, get.Meta(), updater.Name, "reader")
	contracttest.AssertPartner(t, get.Meta(), updater.Name, "writer", "x.Save")
}
