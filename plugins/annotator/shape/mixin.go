// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package shape

import (
	"slices"

	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/core/meta"
)

// Mixin is one orthogonal invariant assertion that decorates a
// callable on top of its structural shape. The mental model is
// layered: the callable's structural [shape] picks the base laws
// the downstream test suite asserts (e.g. Writer →
// "write-then-observe"); each mixin adds one more invariant
// (atomic, idempotent, monotonic, …).
//
// Unlike [Contract], a mixin is a per-callable stamp — it does
// not bind multiple callables together. Mixin directives may
// still carry KV parameters; the values are opaque to the
// umbrella plugin (use a custom resolver if a param value names
// a sibling callable that needs qname rewriting).
//
// Mixin sub-packages export one of these from their public API;
// the consumer composes them into the umbrella plugin via
// [Plugin.Mixins].
type Mixin struct {
	// Name is the mixin's stable identifier (e.g. `"atomic"`,
	// `"idempotent"`, `"monotonic"`). Used as the directive-name
	// argument to `+gen:mixin` and as a path component in every
	// per-mixin param meta key.
	Name string

	// Params enumerates the KV parameter names the mixin accepts.
	// Exported for documentation and future validation; the
	// umbrella plugin's stamping is permissive and accepts any
	// KV (unknown keys still stamp).
	Params []string
}

// MixinDirectiveName is the `+gen:` directive consumers write to
// attach a mixin to a callable.
//
// Syntax:
//
//	//+gen:mixin <name> [<param>=<value>]…
//
// Examples:
//
//	//+gen:mixin atomic
//	//+gen:mixin idempotent
//	//+gen:mixin readafterwrite write=Save
//	//+gen:mixin rate-limited limit=100 burst=10
//
// Multiple `+gen:mixin` directives on one callable stack: each
// one appends to [MetaMixins] and stamps its parameters under
// its own per-mixin namespace.
const MixinDirectiveName = directive.Name("mixin")

// MetaMixins is the per-callable list of mixins decorating the
// callable. Populated by [Plugin.applyMixins] each time a
// non-negated `+gen:mixin` directive is recognised; consumers
// iterate this list to discover every mixin attached to the
// callable, then read the per-mixin param keys.
//
//nolint:gochecknoglobals // registry-singleton key
var MetaMixins = meta.EnsureKey("shape.mixins", meta.StringListParser)

// MixinParamKey returns the typed meta key carrying a mixin's
// KV parameter value — stamped at `shape.mixin.<name>.<param>`.
// Constructed on demand via [meta.EnsureKey] so multiple
// per-mixin sub-packages referencing the same name resolve to
// one canonical key.
func MixinParamKey(name, param string) meta.Key[string] {
	return meta.EnsureKey(
		"shape.mixin."+name+"."+param,
		meta.StringParser,
	)
}

// mixinStampedBy is the setBy attribution used for every
// mixin-related stamp this plugin writes. Distinct from
// [PluginName] so meta provenance distinguishes structural-shape
// stamps from mixin-attachment stamps.
const mixinStampedBy = PluginName + ".mixin"

// applyMixins stamps every non-negated `+gen:mixin` directive on
// bag. Unknown mixin names are silently skipped (the resolver
// pass surfaces them as positioned diagnostics). Unknown
// parameter keys are still stamped verbatim so a resolver has
// the raw data needed to diagnose them.
//
// The function is permissive: the framework's directive
// validator handles schema-level enforcement (positional
// missing, malformed KV) at Build time; this pass concerns
// itself only with meta stamping for directives that already
// passed parse-time validation.
func (p *Plugin) applyMixins(bag *meta.Bag, dirs []*directive.Directive) {
	for _, d := range dirs {
		if d == nil || d.Name != MixinDirectiveName || d.Negated {
			continue
		}
		name := mixinNameFromDirective(d)
		if name == "" {
			continue
		}
		if _, registered := p.mixins[name]; !registered {
			continue
		}
		for k, v := range d.KV {
			if v == "" {
				continue
			}
			MixinParamKey(name, k).Set(bag, v, mixinStampedBy)
		}
		appendMixin(bag, name)
	}
}

// mixinNameFromDirective returns the mixin name declared by d —
// the first positional argument. Returns empty when no
// positional was supplied (the framework validator rejects this
// at parse time; the runtime guard is defence-in-depth).
func mixinNameFromDirective(d *directive.Directive) string {
	if len(d.Args) > 0 {
		return d.Args[0]
	}
	return ""
}

// appendMixin adds name to the [MetaMixins] list on bag,
// preserving insertion order and skipping duplicates. Idempotent:
// repeated calls with the same name leave the list unchanged.
func appendMixin(bag *meta.Bag, name string) {
	current, _ := MetaMixins.Get(bag)
	if slices.Contains(current, name) {
		return
	}
	MetaMixins.Set(bag, append(current, name), mixinStampedBy)
}

// Mixins returns the mixins attached to the callable, in
// insertion order. Returns empty when the callable carries no
// mixin attachments.
//
// Consumers wanting per-mixin parameter data combine the list
// with [MixinParamKey]:
//
//	for _, name := range shape.Mixins(m.Meta()) {
//	    limit, _ := shape.MixinParamKey(name, "limit").Get(m.Meta())
//	    // …
//	}
func Mixins(bag *meta.Bag) []string {
	if bag == nil {
		return nil
	}
	out, _ := MetaMixins.Get(bag)
	return out
}
