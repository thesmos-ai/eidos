// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package protobuf_test

import (
	"testing"

	"go.thesmos.sh/eidos/core/directive"
)

// TestConvert_DirectiveIntegration covers the comment → directive
// translation: a proto declaration carrying a `+gen:<name>` line
// in its leading comments has the parsed [directive.Directive]
// land on the produced node's [node.BaseNode.DirectiveList]. The
// pipeline's gate-fire step reads DirectiveList to decide whether
// a directive-gated plugin (buildergen, repogen, …) runs against
// the node; this test pins the per-node wiring without bringing
// the gated plugins themselves into the picture.
func TestConvert_DirectiveIntegration(t *testing.T) {
	t.Parallel()

	env := loadFixture(t, "messages", "./...")
	if env.diag.HasErrors() {
		t.Fatalf("expected no error diagnostics; got %+v", env.diag.Diagnostics())
	}
	pkg := requireSinglePackage(t, env)

	t.Run("+gen:builder on a proto message lands on the Struct's DirectiveList", func(t *testing.T) {
		t.Parallel()
		user := findStruct(pkg, "User")
		if user == nil {
			t.Fatalf("Struct %q missing", "User")
		}
		var found bool
		for _, d := range user.DirectiveList {
			if string(d.Name) == "builder" && !d.Negated {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf(
				"User.DirectiveList missing a positive `builder` directive; got %+v",
				directiveNames(user.DirectiveList),
			)
		}
	})
}

// directiveNames returns one entry per directive — `+name` for
// positive, `-name` for negated. Used by the directive integration
// test's failure output so the human-readable list of parsed
// directives is visible at-a-glance.
func directiveNames(dl []*directive.Directive) []string {
	out := make([]string, 0, len(dl))
	for _, d := range dl {
		if d.Negated {
			out = append(out, "-"+string(d.Name))
			continue
		}
		out = append(out, "+"+string(d.Name))
	}
	return out
}
