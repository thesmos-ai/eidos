// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package pipeline

import (
	"strings"

	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/store"
)

// scopeBySymbol returns a [store.ScopePredicate] implementing the
// `-target` matching rule: a node is in scope when its unqualified
// Name equals symbol, or when its qualified name ends with
// `.<symbol>` (or `/<symbol>` to honour QNames whose package path
// segment uses slash separators). The qualified form is the
// disambiguator users reach for when the same Name appears in
// multiple source packages (`pkg.Foo` selects Foo from package
// pkg, leaving same-named decls in other packages out of scope).
//
// The constructor is only invoked from [Builder.resolveScope]
// for a non-empty symbol; the empty case short-circuits there
// to a nil predicate. Range queries on the resulting
// [store.Reader] pre-filter to in-scope nodes; direct lookups
// bypass the predicate per the [store.ScopePredicate] contract —
// a caller asking for a specific node by name should get it
// whether or not it's in scope.
//
// Method, field, enum-variant, package, file, and import kinds
// are matched on their unqualified Name only; qualified-form
// targeting is documented for top-level decls (struct, interface,
// function, variable, constant, enum, alias) since the spec's
// open-question resolution scopes `-target` to those.
func scopeBySymbol(symbol string) store.ScopePredicate {
	dotSuffix := "." + symbol
	slashSuffix := "/" + symbol
	return func(n node.Node) bool {
		name, qname := identityOfNode(n)
		if name == symbol {
			return true
		}
		if qname == "" {
			return false
		}
		// Qualified names are formatted "<import-path>.<Name>", so
		// a path-prefixed segment like "pkg/sub.Name" matches when
		// the QName ends with "/pkg/sub.Name"; a bare-name-with-
		// short-package form like "pkg.Name" matches when the
		// QName ends with "/pkg.Name". An already-fully-qualified
		// symbol matches by equality.
		return qname == symbol ||
			strings.HasSuffix(qname, dotSuffix) ||
			strings.HasSuffix(qname, slashSuffix)
	}
}

// identityOfNode returns n's unqualified Name and qualified Name
// in one pass. Top-level kinds expose both; nested and structural
// kinds return their Name with an empty qualified form because
// qualified-form targeting is documented for top-level decls only.
// Returns empty strings for kinds without a meaningful Name
// (imports without an alias). Param and TypeParam are not
// surfaced through [store.Reader]'s range queries, so their
// arms are intentionally omitted — a future Reader extension
// that exposes them will add the arms here at the same time.
func identityOfNode(n node.Node) (name, qname string) {
	switch v := n.(type) {
	case *node.Struct:
		return v.Name, v.QName()
	case *node.Interface:
		return v.Name, v.QName()
	case *node.Function:
		return v.Name, v.QName()
	case *node.Variable:
		return v.Name, v.QName()
	case *node.Constant:
		return v.Name, v.QName()
	case *node.Enum:
		return v.Name, v.QName()
	case *node.Alias:
		return v.Name, v.QName()
	case *node.Method:
		return v.Name, ""
	case *node.Field:
		return v.Name, ""
	case *node.EnumVariant:
		return v.Name, ""
	case *node.Package:
		return v.Name, ""
	case *node.File:
		return v.Name, ""
	case *node.Import:
		return v.Alias, ""
	}
	return "", ""
}
