// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package scope_test

import (
	"testing"

	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/internal/mixintest"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/scope"
)

func TestMixin(t *testing.T) {
	t.Parallel()

	t.Run("identity", func(t *testing.T) {
		t.Parallel()
		mixintest.AssertIdentity(t, scope.Mixin(), scope.Name, scope.Params)
	})

	t.Run("pipeline stamps name param", func(t *testing.T) {
		t.Parallel()
		fn := &node.Function{
			Name: "Begin", Package: "x",
			BaseNode: node.BaseNode{
				DirectiveList: []*directive.Directive{
					mixintest.HostDirective(scope.Name, map[string]string{
						"name": "request",
					}),
				},
			},
		}
		bag := mixintest.RunPipeline(t, scope.Mixin(), fn)
		mixintest.AssertAttached(t, bag, scope.Name)
		mixintest.AssertParam(t, bag, scope.Name, "name", "request")
	})
}
