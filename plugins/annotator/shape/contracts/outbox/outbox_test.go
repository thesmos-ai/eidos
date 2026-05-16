// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package outbox_test

import (
	"testing"

	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/plugins/annotator/shape/contracts/internal/contracttest"
	"go.thesmos.sh/eidos/plugins/annotator/shape/contracts/outbox"
)

func TestContract_Identity(t *testing.T) {
	t.Parallel()
	contracttest.AssertIdentity(t,
		outbox.Contract(),
		outbox.Name, outbox.Roles)
}

func TestContract_PipelineRoundTrip(t *testing.T) {
	t.Parallel()
	app := &node.Function{
		Name: "Append", Package: "x",
		BaseNode: node.BaseNode{
			DirectiveList: []*directive.Directive{
				contracttest.HostDirective(outbox.Name, "append", map[string]string{
					"subscribe": "Subscribe",
				}),
			},
		},
	}
	sub := &node.Function{Name: "Subscribe", Package: "x"}
	pkg := &node.Package{
		Name: "x", Path: "x",
		Functions: []*node.Function{app, sub},
	}
	diags := contracttest.RunPipeline(t, outbox.Contract(), pkg)
	contracttest.AssertNoErrorDiag(t, diags)

	contracttest.AssertRole(t, app.Meta(), outbox.Name, "append")
	contracttest.AssertPartner(t, app.Meta(), outbox.Name, "subscribe", "x.Subscribe")
	contracttest.AssertRole(t, sub.Meta(), outbox.Name, "subscribe")
	contracttest.AssertPartner(t, sub.Meta(), outbox.Name, "append", "x.Append")
}
