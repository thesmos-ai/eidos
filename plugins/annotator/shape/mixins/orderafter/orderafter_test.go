// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package orderafter_test

import (
	"testing"

	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/plugins/annotator/shape"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/internal/mixintest"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/orderafter"
)

func TestMixin(t *testing.T) {
	t.Parallel()

	t.Run("identity", func(t *testing.T) {
		t.Parallel()
		mixintest.AssertIdentity(t, orderafter.Mixin(), orderafter.Name, orderafter.Params)
	})

	t.Run("resolver rewrites fn param to qualified name", func(t *testing.T) {
		t.Parallel()
		host := &node.Function{
			Name: "DoWork", Package: "x",
			BaseNode: node.BaseNode{
				DirectiveList: []*directive.Directive{
					mixintest.HostDirective(orderafter.Name, map[string]string{
						"fn": "Initialise",
					}),
				},
			},
		}
		initFn := &node.Function{Name: "Initialise", Package: "x"}
		mixintest.RunWithResolver(t, orderafter.Mixin(), &node.Package{
			Name: "x", Path: "x",
			Functions: []*node.Function{host, initFn},
		})

		got, _ := shape.MixinParamKey(orderafter.Name, "fn").Get(host.Meta())
		if got != "x.Initialise" {
			t.Fatalf("fn param = %q, want %q", got, "x.Initialise")
		}
	})
}
