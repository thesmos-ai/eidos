// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package shape_test

import (
	"reflect"
	"testing"

	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/core/meta"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/plugins/annotator/shape"
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
