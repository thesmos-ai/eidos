// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package publisher_test

import (
	"testing"

	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/plugins/annotator/shape/contracts/internal/contracttest"
	"go.thesmos.sh/eidos/plugins/annotator/shape/contracts/publisher"
)

func TestContract_Identity(t *testing.T) {
	t.Parallel()
	contracttest.AssertIdentity(t,
		publisher.Contract(),
		publisher.Name, publisher.Roles)
}

func TestContract_PipelineRoundTrip(t *testing.T) {
	t.Parallel()
	pub := &node.Function{
		Name: "Publish", Package: "x",
		BaseNode: node.BaseNode{
			DirectiveList: []*directive.Directive{
				contracttest.HostDirective(publisher.Name, "publish", map[string]string{
					"subscribe": "Subscribe",
				}),
			},
		},
	}
	sub := &node.Function{Name: "Subscribe", Package: "x"}
	pkg := &node.Package{
		Name: "x", Path: "x",
		Functions: []*node.Function{pub, sub},
	}
	diags := contracttest.RunPipeline(t, publisher.Contract(), pkg)
	contracttest.AssertNoErrorDiag(t, diags)

	contracttest.AssertRole(t, pub.Meta(), publisher.Name, "publish")
	contracttest.AssertPartner(t, pub.Meta(), publisher.Name, "subscribe", "x.Subscribe")
	contracttest.AssertRole(t, sub.Meta(), publisher.Name, "subscribe")
	contracttest.AssertPartner(t, sub.Meta(), publisher.Name, "publish", "x.Publish")
}
