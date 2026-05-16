// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package shape_test

import (
	"reflect"
	"testing"

	"go.thesmos.sh/eidos/core/diag"
	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/core/meta"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/plugins/annotator/shape"
	"go.thesmos.sh/eidos/sdk"
	"go.thesmos.sh/eidos/store"
)

// atomicMixin is the canonical zero-param test mixin used by the
// presence-only stamping tests below.
func atomicMixin() shape.Mixin {
	return shape.Mixin{Name: "atomic"}
}

// rateLimitedMixin is the canonical multi-param test mixin used
// by the parameter-stamping tests below.
func rateLimitedMixin() shape.Mixin {
	return shape.Mixin{
		Name:   "rate-limited",
		Params: []string{"limit", "burst"},
	}
}

// TestMixin_DirectiveStamping covers the umbrella plugin's
// mixin stamping: each non-negated `+gen:mixin` directive on a
// callable appends to the mixins list and stamps its per-param
// keys without interfering with structural shape stamps.
func TestMixin_DirectiveStamping(t *testing.T) {
	t.Parallel()

	t.Run("stamps a parameter-less mixin", func(t *testing.T) {
		t.Parallel()
		fn := mixinFn(
			"Save",
			&directive.Directive{
				Name: shape.MixinDirectiveName,
				Args: []string{"atomic"},
			},
		)
		runAnnotate(t, shape.New().Mixins(atomicMixin()), pkgWithFunction(fn))

		assertMixins(t, fn.Meta(), []string{"atomic"})
	})

	t.Run("stamps parameter values under the mixin's namespace", func(t *testing.T) {
		t.Parallel()
		fn := mixinFn(
			"Charge",
			&directive.Directive{
				Name: shape.MixinDirectiveName,
				Args: []string{"rate-limited"},
				KV:   map[string]string{"limit": "100", "burst": "10"},
			},
		)
		runAnnotate(t, shape.New().Mixins(rateLimitedMixin()), pkgWithFunction(fn))

		assertMixins(t, fn.Meta(), []string{"rate-limited"})
		assertMeta(t, fn.Meta(), shape.MixinParamKey("rate-limited", "limit"), "100")
		assertMeta(t, fn.Meta(), shape.MixinParamKey("rate-limited", "burst"), "10")
	})

	t.Run("multiple mixins on one callable stack in declaration order", func(t *testing.T) {
		t.Parallel()
		fn := mixinFn(
			"Save",
			&directive.Directive{Name: shape.MixinDirectiveName, Args: []string{"atomic"}},
			&directive.Directive{
				Name: shape.MixinDirectiveName,
				Args: []string{"rate-limited"},
				KV:   map[string]string{"limit": "50"},
			},
		)
		runAnnotate(
			t,
			shape.New().Mixins(atomicMixin(), rateLimitedMixin()),
			pkgWithFunction(fn),
		)

		assertMixins(t, fn.Meta(), []string{"atomic", "rate-limited"})
		assertMeta(t, fn.Meta(), shape.MixinParamKey("rate-limited", "limit"), "50")
	})

	t.Run("mixin stamps alongside contract membership and structural shape", func(t *testing.T) {
		t.Parallel()
		// Reader-shaped callable, with both a contract membership
		// and a mixin attached. All three stamps must land.
		fn := readerFunc("Find")
		fn.DirectiveList = []*directive.Directive{
			{
				Name: shape.ContractDirectiveName,
				Args: []string{"tx"},
				KV:   map[string]string{"role": "begin"},
			},
			{Name: shape.MixinDirectiveName, Args: []string{"atomic"}},
		}
		runAnnotate(
			t,
			shape.New().
				Detectors(testReaderDetector()).
				Contracts(txContract()).
				Mixins(atomicMixin()),
			pkgWithFunction(fn),
		)

		assertShape(t, fn.Meta(), "reader")
		assertContracts(t, fn.Meta(), []string{"tx"})
		assertMixins(t, fn.Meta(), []string{"atomic"})
	})

	t.Run("unknown mixin name is silently skipped", func(t *testing.T) {
		t.Parallel()
		fn := mixinFn(
			"X",
			&directive.Directive{
				Name: shape.MixinDirectiveName,
				Args: []string{"never-registered"},
			},
		)
		runAnnotate(t, shape.New(), pkgWithFunction(fn))
		if got := shape.Mixins(fn.Meta()); len(got) != 0 {
			t.Fatalf("expected no mixin stamps; got %v", got)
		}
	})

	t.Run("negated directive is ignored", func(t *testing.T) {
		t.Parallel()
		fn := mixinFn(
			"Save",
			&directive.Directive{
				Name:    shape.MixinDirectiveName,
				Args:    []string{"atomic"},
				Negated: true,
			},
		)
		runAnnotate(t, shape.New().Mixins(atomicMixin()), pkgWithFunction(fn))
		if got := shape.Mixins(fn.Meta()); len(got) != 0 {
			t.Fatalf("expected negated directive to be ignored; got %v", got)
		}
	})

	t.Run("empty parameter values are skipped", func(t *testing.T) {
		t.Parallel()
		fn := mixinFn(
			"Charge",
			&directive.Directive{
				Name: shape.MixinDirectiveName,
				Args: []string{"rate-limited"},
				KV:   map[string]string{"limit": "100", "burst": ""},
			},
		)
		runAnnotate(t, shape.New().Mixins(rateLimitedMixin()), pkgWithFunction(fn))

		assertMeta(t, fn.Meta(), shape.MixinParamKey("rate-limited", "limit"), "100")
		if _, ok := shape.MixinParamKey("rate-limited", "burst").Get(fn.Meta()); ok {
			t.Fatalf("expected empty parameter value to be unstamped")
		}
	})

	t.Run("duplicate mixin directive does not duplicate the list entry", func(t *testing.T) {
		t.Parallel()
		fn := mixinFn(
			"Save",
			&directive.Directive{Name: shape.MixinDirectiveName, Args: []string{"atomic"}},
			&directive.Directive{Name: shape.MixinDirectiveName, Args: []string{"atomic"}},
		)
		runAnnotate(t, shape.New().Mixins(atomicMixin()), pkgWithFunction(fn))
		assertMixins(t, fn.Meta(), []string{"atomic"})
	})

	t.Run("method-bound mixins stamp the same as free functions", func(t *testing.T) {
		t.Parallel()
		m := &node.Method{
			Name: "Save",
			BaseNode: node.BaseNode{
				DirectiveList: []*directive.Directive{
					{Name: shape.MixinDirectiveName, Args: []string{"atomic"}},
				},
			},
		}
		s := &node.Struct{Name: "Repo", Package: "x", Methods: []*node.Method{m}}
		runAnnotate(t, shape.New().Mixins(atomicMixin()), pkgWithStruct(s))

		assertMixins(t, m.Meta(), []string{"atomic"})
	})

	t.Run("Mixins helper returns nil for an unstamped bag", func(t *testing.T) {
		t.Parallel()
		if got := shape.Mixins(nil); got != nil {
			t.Fatalf("Mixins(nil) = %v, want nil", got)
		}
		if got := shape.Mixins(meta.NewBag()); got != nil {
			t.Fatalf("Mixins(empty) = %v, want nil", got)
		}
	})
}

// mixinFn returns a free-function node carrying the supplied
// directives — used by every test that exercises directive-driven
// mixin stamping.
func mixinFn(name string, dirs ...*directive.Directive) *node.Function {
	return &node.Function{
		Name: name, Package: "x",
		BaseNode: node.BaseNode{DirectiveList: dirs},
	}
}

// assertMixins fails the test when the mixin list stamped on bag
// does not deep-equal want.
func assertMixins(t *testing.T, bag *meta.Bag, want []string) {
	t.Helper()
	got := shape.Mixins(bag)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Mixins = %v, want %v", got, want)
	}
}

// TestMixin_SiblingResolution covers the resolver rewriting
// declared [shape.Mixin.SiblingParams] values from raw names to
// qualified names — exercising the per-mixin sibling-resolution
// pass added alongside the contract resolver.
func TestMixin_SiblingResolution(t *testing.T) {
	t.Parallel()
	rafw := shape.Mixin{
		Name:          "readafterwrite",
		Params:        []string{"write"},
		SiblingParams: []string{"write"},
	}
	find := mixinFn("Find", &directive.Directive{
		Name: shape.MixinDirectiveName,
		Args: []string{"readafterwrite"},
		KV:   map[string]string{"write": "Save"},
	})
	save := &node.Function{Name: "Save", Package: "x"}
	pkg := &node.Package{
		Name: "x", Path: "x",
		Functions: []*node.Function{find, save},
	}

	umbrella := shape.New().Mixins(rafw)
	ctx := contracttestCtxForMixin(t, pkg)
	if err := umbrella.Annotate(ctx); err != nil {
		t.Fatalf("umbrella.Annotate: %v", err)
	}
	if err := umbrella.Resolver().Annotate(ctx); err != nil {
		t.Fatalf("resolver.Annotate: %v", err)
	}

	got, _ := shape.MixinParamKey("readafterwrite", "write").Get(find.Meta())
	if got != "x.Save" {
		t.Fatalf("mixin sibling param = %q, want %q", got, "x.Save")
	}
}

// TestMixin_Validate covers the validator invoking the
// [shape.Mixin.Validate] hook after sibling resolution. The
// flagging mixin emits one violation per attachment; the
// validator surfaces it as a positioned diagnostic.
func TestMixin_Validate(t *testing.T) {
	t.Parallel()
	flagging := shape.Mixin{
		Name: "flagging",
		Validate: func(attachments []shape.MixinAttachment) []shape.MixinViolation {
			out := make([]shape.MixinViolation, 0, len(attachments))
			for _, a := range attachments {
				out = append(out, shape.MixinViolation{
					Host: a.Host, Message: "synthetic flag",
				})
			}
			return out
		},
	}
	fn := mixinFn("X", &directive.Directive{
		Name: shape.MixinDirectiveName,
		Args: []string{"flagging"},
	})
	pkg := &node.Package{Name: "x", Path: "x", Functions: []*node.Function{fn}}
	ctx := contracttestCtxForMixin(t, pkg)
	umbrella := shape.New().Mixins(flagging)
	if err := umbrella.Annotate(ctx); err != nil {
		t.Fatalf("umbrella.Annotate: %v", err)
	}
	if err := umbrella.Resolver().Annotate(ctx); err != nil {
		t.Fatalf("resolver.Annotate: %v", err)
	}
	if err := umbrella.Validator().Annotate(ctx); err != nil {
		t.Fatalf("validator.Annotate: %v", err)
	}
	assertContainsDiag(t, ctx.Diag.Diagnostics(), diag.Error, "synthetic flag")
}

// contracttestCtxForMixin builds an annotator context backed by a
// fresh store seeded with pkg and stamped with the "golang"
// frontend marker. Used by the mixin pipeline tests above.
func contracttestCtxForMixin(t *testing.T, pkg *node.Package) *sdk.AnnotatorContext {
	t.Helper()
	s := store.New()
	if err := s.Nodes().AddPackage(pkg); err != nil {
		t.Fatalf("AddPackage: %v", err)
	}
	frontendMarker.Set(pkg.Meta(), "golang", "test")
	return &sdk.AnnotatorContext{
		Store:  s,
		Reader: store.NewReader(s),
		Diag:   diag.New(),
	}
}
