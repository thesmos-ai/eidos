// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package validates_test

import (
	"testing"

	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/plugins/annotator/shape"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/internal/mixintest"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/validates"
)

func TestMixin(t *testing.T) {
	t.Parallel()

	t.Run("identity", func(t *testing.T) {
		t.Parallel()
		mixintest.AssertIdentity(t, validates.Mixin(), validates.Name, validates.Params)
	})

	t.Run("resolver rewrites fn param to qualified name", func(t *testing.T) {
		t.Parallel()
		host := &node.Function{
			Name: "Save", Package: "x",
			BaseNode: node.BaseNode{
				DirectiveList: []*directive.Directive{
					mixintest.HostDirective(validates.Name, map[string]string{
						"fn": "ValidateInput",
					}),
				},
			},
		}
		validator := &node.Function{Name: "ValidateInput", Package: "x"}
		mixintest.RunWithResolver(t, validates.Mixin(), &node.Package{
			Name: "x", Path: "x",
			Functions: []*node.Function{host, validator},
		})

		got, _ := shape.MixinParamKey(validates.Name, "fn").Get(host.Meta())
		if got != "x.ValidateInput" {
			t.Fatalf("fn param = %q, want %q", got, "x.ValidateInput")
		}
	})
}
