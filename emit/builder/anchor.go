// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package builder

import (
	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/node"
)

// Anchor returns a [PackageBuilder] whose emit.Package.Path is
// derived from anchor's owning source package, and whose default
// [emit.BaseEmit.OriginNode] is anchor itself. Every decl built
// through the returned PackageBuilder inherits anchor as its
// origin unless the per-decl builder overrides via its own Origin
// setter.
//
// This is the principled entry point for plugins emitting against a
// single source declaration. Plugins call:
//
//	pkg := builder.For(p.Name()).Anchor(iface)
//	pkg.Struct(...)
//	out, _ := pkg.Build()
//
// and never name a package path or a target — the pipeline's Layout
// phase composes those from the source anchor plus any
// `+gen:out` / config / CLI overrides.
//
// Anchor accepts any [node.Node] whose owner chain reaches a packaged
// ancestor (Struct, Interface, Function, Variable, Constant, Enum,
// Alias, Package itself; Method, Field, EnumVariant walk to their
// host). Nodes that cannot resolve to a packaged ancestor (Param,
// TypeParam, …) anchor to an empty path; the pipeline surfaces this
// as a routing error if the resulting decls are routable.
func (c *Context) Anchor(anchor node.Node) *PackageBuilder {
	b := c.Package("", packagePath(anchor))
	b.defaultOrigin = anchor
	return b
}

// packagePath extracts the source-side import path from n. Walks the
// owner chain for kinds that don't carry a Package field directly
// (Method, Field, EnumVariant). Returns the empty string for kinds
// whose routing semantics are undefined (Param, TypeParam, Import).
func packagePath(n node.Node) string {
	cur := n
	for cur != nil {
		switch v := cur.(type) {
		case *node.Package:
			return v.Path
		case *node.Struct:
			return v.Package
		case *node.Interface:
			return v.Package
		case *node.Function:
			return v.Package
		case *node.Variable:
			return v.Package
		case *node.Constant:
			return v.Package
		case *node.Enum:
			return v.Package
		case *node.Alias:
			return v.Package
		case *node.File:
			if v.Owner != nil {
				return v.Owner.Path
			}
			return ""
		case *node.Method:
			cur = v.Owner
			continue
		case *node.Field:
			cur = v.Owner
			continue
		case *node.EnumVariant:
			if v.Owner == nil {
				return ""
			}
			return v.Owner.Package
		}
		return ""
	}
	return ""
}

// applyBuilderDefaults stamps b's defaults onto dst — the Anchor's
// default Origin (when the per-decl callback didn't supply one) and
// the sub-context's OutputTag. Decl constructors call this on the
// freshly-built emit value so both defaults flow through uniformly
// without each constructor restating them.
func applyBuilderDefaults(b *PackageBuilder, dst *emit.BaseEmit) {
	if b == nil {
		return
	}
	if dst.OriginNode == nil && b.defaultOrigin != nil {
		dst.OriginNode = b.defaultOrigin
	}
	if dst.OutputTagName == "" {
		dst.OutputTagName = b.outputTag
	}
}
