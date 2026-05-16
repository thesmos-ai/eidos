// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package wrappedvia_test

import (
	"testing"

	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/plugins/annotator/shape"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/internal/mixintest"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/wrappedvia"
)

func TestMixin(t *testing.T) {
	t.Parallel()

	t.Run("identity", func(t *testing.T) {
		t.Parallel()
		mixintest.AssertIdentity(t, wrappedvia.Mixin(), wrappedvia.Name, wrappedvia.Params)
	})

	t.Run("resolver rewrites fn param to qualified name", func(t *testing.T) {
		t.Parallel()
		host := &node.Function{
			Name: "Wrapped", Package: "x",
			BaseNode: node.BaseNode{
				DirectiveList: []*directive.Directive{
					mixintest.HostDirective(wrappedvia.Name, map[string]string{
						"fn": "Delegate",
					}),
				},
			},
		}
		inner := &node.Function{Name: "Delegate", Package: "x"}
		mixintest.RunWithResolver(t, wrappedvia.Mixin(), &node.Package{
			Name: "x", Path: "x",
			Functions: []*node.Function{host, inner},
		})

		got, _ := shape.MixinParamKey(wrappedvia.Name, "fn").Get(host.Meta())
		if got != "x.Delegate" {
			t.Fatalf("fn param = %q, want %q", got, "x.Delegate")
		}
	})
}
