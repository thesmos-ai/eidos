// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package shape_test

import (
	"testing"

	"go.thesmos.sh/eidos/core/diag"
	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/plugin"
	"go.thesmos.sh/eidos/plugins/annotator/shape"
	"go.thesmos.sh/eidos/priority"
	"go.thesmos.sh/eidos/sdk"
	"go.thesmos.sh/eidos/store"
)

// TestValidator_Contract pins the role / capability surface the
// framework asserts at registration / build time.
func TestValidator_Contract(t *testing.T) {
	t.Parallel()

	t.Run("name is stable", func(t *testing.T) {
		t.Parallel()
		v := shape.New().Validator()
		first := v.Name()
		second := v.Name()
		if first == "" || first != second {
			t.Fatalf("Name() unstable or empty: first=%q second=%q", first, second)
		}
	})

	t.Run("priority is AnnotatorValidation", func(t *testing.T) {
		t.Parallel()
		if got, want := shape.New().Validator().Priority(), priority.AnnotatorValidation; got != want {
			t.Fatalf("Priority() = %v, want %v", got, want)
		}
	})

	t.Run("satisfies plugin.Annotator", func(t *testing.T) {
		t.Parallel()
		var _ plugin.Annotator = shape.New().Validator()
	})

	t.Run("Annotate on empty store does not panic", func(t *testing.T) {
		t.Parallel()
		ctx := newAnnotatorContext(t, store.New())
		if err := shape.New().Validator().Annotate(ctx); err != nil {
			t.Fatalf("Annotate(empty): %v", err)
		}
	})
}

// TestValidator_RequiredPartners covers the per-role
// required-partner check: the validator emits a positioned
// diagnostic when a role declared in [Contract.Required] is
// missing a partner stamp after the resolver has run.
func TestValidator_RequiredPartners(t *testing.T) {
	t.Parallel()

	t.Run("missing required partner surfaces a diagnostic", func(t *testing.T) {
		t.Parallel()
		spec := shape.Contract{
			Name:     "tx",
			Roles:    []string{"begin", "commit", "rollback"},
			Required: map[string][]string{"begin": {"commit", "rollback"}},
		}
		// Directive declares only `commit=`; the validator must
		// flag the missing `rollback=` partner.
		fn := contractFn("Begin",
			&directive.Directive{
				Name: shape.ContractDirectiveName,
				Args: []string{"tx"},
				KV:   map[string]string{"role": "begin", "commit": "Commit"},
			},
			// A standalone commit function so the resolver succeeds
			// for the one partner that is provided.
		)
		commit := &node.Function{Name: "Commit", Package: "x"}
		pkg := &node.Package{
			Name: "x", Path: "x",
			Functions: []*node.Function{fn, commit},
		}
		diags := runFullPipeline(t, spec, pkg)
		assertContainsDiag(t, diags, diag.Error, "rollback")
	})

	t.Run("all required partners present emits no diagnostic", func(t *testing.T) {
		t.Parallel()
		spec := shape.Contract{
			Name:     "tx",
			Roles:    []string{"begin", "commit"},
			Required: map[string][]string{"begin": {"commit"}},
		}
		fn := contractFn("Begin",
			&directive.Directive{
				Name: shape.ContractDirectiveName,
				Args: []string{"tx"},
				KV:   map[string]string{"role": "begin", "commit": "Commit"},
			},
		)
		commit := &node.Function{Name: "Commit", Package: "x"}
		pkg := &node.Package{
			Name: "x", Path: "x",
			Functions: []*node.Function{fn, commit},
		}
		for _, d := range runFullPipeline(t, spec, pkg) {
			if d.Severity >= diag.Error {
				t.Fatalf("unexpected error diagnostic: %+v", d)
			}
		}
	})
}

// TestValidator_ContractValidate covers the per-contract
// invariant hook — [Contract.Validate] receives the resolved
// member set and may emit violations the validator surfaces as
// positioned diagnostics.
func TestValidator_ContractValidate(t *testing.T) {
	t.Parallel()

	t.Run("validator hook receives the member set keyed by role", func(t *testing.T) {
		t.Parallel()
		var captured map[string][]node.Node
		spec := shape.Contract{
			Name:  "tx",
			Roles: []string{"begin", "commit"},
			Validate: func(members map[string][]node.Node) []shape.ContractViolation {
				captured = members
				return nil
			},
		}
		begin := contractFn("Begin",
			&directive.Directive{
				Name: shape.ContractDirectiveName,
				Args: []string{"tx"},
				KV:   map[string]string{"role": "begin", "commit": "Commit"},
			},
		)
		commit := &node.Function{Name: "Commit", Package: "x"}
		pkg := &node.Package{
			Name: "x", Path: "x",
			Functions: []*node.Function{begin, commit},
		}
		_ = runFullPipeline(t, spec, pkg)

		if got := len(captured["begin"]); got != 1 {
			t.Fatalf("members[begin] = %d nodes, want 1", got)
		}
		if got := len(captured["commit"]); got != 1 {
			t.Fatalf("members[commit] = %d nodes, want 1", got)
		}
	})

	t.Run("violations surface as positioned diagnostics", func(t *testing.T) {
		t.Parallel()
		spec := shape.Contract{
			Name:  "tx",
			Roles: []string{"begin", "commit"},
			Validate: func(members map[string][]node.Node) []shape.ContractViolation {
				return []shape.ContractViolation{
					{
						Host:    members["begin"][0],
						Message: "tx: synthetic invariant breach",
					},
				}
			},
		}
		begin := contractFn("Begin",
			&directive.Directive{
				Name: shape.ContractDirectiveName,
				Args: []string{"tx"},
				KV:   map[string]string{"role": "begin", "commit": "Commit"},
			},
		)
		commit := &node.Function{Name: "Commit", Package: "x"}
		pkg := &node.Package{
			Name: "x", Path: "x",
			Functions: []*node.Function{begin, commit},
		}
		diags := runFullPipeline(t, spec, pkg)
		assertContainsDiag(t, diags, diag.Error, "synthetic invariant breach")
	})

	t.Run("nil Validate hook is a permissive no-op", func(t *testing.T) {
		t.Parallel()
		spec := shape.Contract{
			Name:  "tx",
			Roles: []string{"begin", "commit"},
		}
		fn := contractFn("Begin",
			&directive.Directive{
				Name: shape.ContractDirectiveName,
				Args: []string{"tx"},
				KV:   map[string]string{"role": "begin", "commit": "Commit"},
			},
		)
		commit := &node.Function{Name: "Commit", Package: "x"}
		pkg := &node.Package{
			Name: "x", Path: "x",
			Functions: []*node.Function{fn, commit},
		}
		for _, d := range runFullPipeline(t, spec, pkg) {
			if d.Severity >= diag.Error {
				t.Fatalf("nil Validate must not produce errors; got %+v", d)
			}
		}
	})
}

// runFullPipeline wires pkg into a fresh store and runs the
// umbrella → resolver → validator sequence with the supplied
// contract registered on all three. Returns the accumulated
// diagnostic snapshot.
func runFullPipeline(t *testing.T, c shape.Contract, pkg *node.Package) []diag.Diag {
	t.Helper()
	s := store.New()
	if err := s.Nodes().AddPackage(pkg); err != nil {
		t.Fatalf("AddPackage: %v", err)
	}
	frontendMarker.Set(pkg.Meta(), "golang", "test")

	umbrella := shape.New().Contracts(c)
	sink := diag.New()
	ctx := &sdk.AnnotatorContext{
		Store:  s,
		Reader: store.NewReader(s),
		Diag:   sink,
	}
	if err := umbrella.Annotate(ctx); err != nil {
		t.Fatalf("umbrella.Annotate: %v", err)
	}
	if err := umbrella.Resolver().Annotate(ctx); err != nil {
		t.Fatalf("resolver.Annotate: %v", err)
	}
	if err := umbrella.Validator().Annotate(ctx); err != nil {
		t.Fatalf("validator.Annotate: %v", err)
	}
	return sink.Diagnostics()
}
