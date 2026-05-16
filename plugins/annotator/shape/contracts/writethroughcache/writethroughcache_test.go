// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package writethroughcache_test

import (
	"testing"

	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/plugins/annotator/shape/contracts/internal/contracttest"
	"go.thesmos.sh/eidos/plugins/annotator/shape/contracts/writethroughcache"
)

func TestContract(t *testing.T) {
	t.Parallel()

	t.Run("identity", func(t *testing.T) {
		t.Parallel()
		contracttest.AssertIdentity(t, writethroughcache.Contract(),
			writethroughcache.Name, writethroughcache.Roles)
	})

	t.Run("pipeline round-trip back-stamps backing partner", func(t *testing.T) {
		t.Parallel()
		cache := &node.Function{
			Name: "Get", Package: "x",
			BaseNode: node.BaseNode{
				DirectiveList: []*directive.Directive{
					contracttest.HostDirective(writethroughcache.Name, "cache", map[string]string{
						"backing": "GetFromDB",
					}),
				},
			},
		}
		backing := &node.Function{Name: "GetFromDB", Package: "x"}
		pkg := &node.Package{
			Name: "x", Path: "x",
			Functions: []*node.Function{cache, backing},
		}
		diags := contracttest.RunPipeline(t, writethroughcache.Contract(), pkg)
		contracttest.AssertNoErrorDiag(t, diags)

		contracttest.AssertRole(t, cache.Meta(), writethroughcache.Name, "cache")
		contracttest.AssertPartner(t, cache.Meta(), writethroughcache.Name, "backing", "x.GetFromDB")
		contracttest.AssertRole(t, backing.Meta(), writethroughcache.Name, "backing")
	})
}
