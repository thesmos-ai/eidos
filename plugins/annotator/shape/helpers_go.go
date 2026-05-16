// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package shape

import "go.thesmos.sh/eidos/node"

// The helpers in this file are the Go-flavoured query primitives
// a Go-language [DetectFunc] composes. They are intentionally lazy
// — each helper inspects only what its caller asked for, so a
// detector that cares about the leading context.Context does not
// pay the cost of scanning every result, and one that cares about
// `iter.Seq` does not pay for context inspection.
//
// Detectors typically begin by destructuring the callable into its
// raw param and return slices via [GoCallable], then compose the
// individual predicates as needed.
//
// A non-Go frontend that wants the same query primitives writes
// its own `helpers_<frontend>.go` next to this file. The shape
// library has no opinion on the spelling of those helpers; the
// only contract is what a positive [Match] looks like.

// GoCallable returns the parameter and return slices for the two
// callable node kinds. Returns `(nil, nil)` for any other node
// type so detectors can early-return on a falsy length check
// without a separate type-assertion ladder.
func GoCallable(n node.Node) (params []*node.Param, returns []*node.TypeRef) {
	switch x := n.(type) {
	case *node.Function:
		return x.Params, x.Returns
	case *node.Method:
		return x.Params, x.Returns
	}
	return nil, nil
}

// GoHasContext reports whether the first parameter is the
// Go-native `context.Context` type.
func GoHasContext(params []*node.Param) bool {
	return len(params) > 0 && isGoContextRef(params[0].Type)
}

// GoStripContext returns params with a leading `context.Context`
// parameter dropped. Returns params unchanged when no context
// parameter is present.
func GoStripContext(params []*node.Param) []*node.Param {
	if GoHasContext(params) {
		return params[1:]
	}
	return params
}

// GoTrailingVariadic returns the last parameter when it is
// variadic, or nil otherwise. Useful for detectors that want to
// peel a `...T` off the end before classifying the remaining
// fixed parameters.
func GoTrailingVariadic(params []*node.Param) *node.Param {
	n := len(params)
	if n == 0 || !params[n-1].Variadic {
		return nil
	}
	return params[n-1]
}

// GoStripVariadic returns params with a trailing variadic
// parameter dropped. Returns params unchanged when no variadic
// parameter is present.
func GoStripVariadic(params []*node.Param) []*node.Param {
	if GoTrailingVariadic(params) == nil {
		return params
	}
	return params[:len(params)-1]
}

// GoErrorIndex returns the index of the first bare builtin
// `error` return, or -1 when no error return is present.
func GoErrorIndex(returns []*node.TypeRef) int {
	for i, r := range returns {
		if isGoErrorRef(r) {
			return i
		}
	}
	return -1
}

// GoHasError reports whether any return is the bare builtin
// `error` type.
func GoHasError(returns []*node.TypeRef) bool {
	return GoErrorIndex(returns) >= 0
}

// GoStripError returns the return list with the first bare
// builtin `error` return dropped. Returns the slice unchanged
// when no error return is present.
func GoStripError(returns []*node.TypeRef) []*node.TypeRef {
	i := GoErrorIndex(returns)
	if i < 0 {
		return returns
	}
	out := make([]*node.TypeRef, 0, len(returns)-1)
	out = append(out, returns[:i]...)
	out = append(out, returns[i+1:]...)
	return out
}

// GoIterSeqElem returns the element type V when r is
// `iter.Seq[V]` and nil otherwise. Used by detectors that
// recognise the single-value iterator shape.
func GoIterSeqElem(r *node.TypeRef) *node.TypeRef {
	if !isIterSeqRef(r) || len(r.TypeArgs) == 0 {
		return nil
	}
	return r.TypeArgs[0]
}

// GoIterSeq2Args returns the (K, V) type args when r is
// `iter.Seq2[K, V]` and (nil, nil) otherwise. Used by detectors
// that recognise the key/value iterator shape.
func GoIterSeq2Args(r *node.TypeRef) (*node.TypeRef, *node.TypeRef) {
	if !isIterSeq2Ref(r) || len(r.TypeArgs) < 2 {
		return nil, nil
	}
	return r.TypeArgs[0], r.TypeArgs[1]
}

// QName returns the qualified type spelling for a type ref —
// `Package.Name` when Package is set, just `Name` otherwise.
// Detectors use this to populate [Match.KeyType] and
// [Match.ValueType]; consumers reading those meta keys get the
// same form back. Returns empty when r is nil so callers don't
// need a nil guard before calling.
func QName(r *node.TypeRef) string {
	if r == nil {
		return ""
	}
	if r.Package == "" {
		return r.Name
	}
	return r.Package + "." + r.Name
}

// isGoContextRef reports whether r is the Go-native
// `context.Context` type. The TypeRef carries the source package
// qualifier verbatim — see the Go frontend's emission contract.
func isGoContextRef(r *node.TypeRef) bool {
	if r == nil {
		return false
	}
	return r.Name == "Context" && r.Package == "context"
}

// isGoErrorRef reports whether r is the bare builtin `error`
// type. The Go frontend emits builtin error as a TypeRef with
// empty Package and Name "error".
func isGoErrorRef(r *node.TypeRef) bool {
	if r == nil {
		return false
	}
	return r.Name == "error" && r.Package == ""
}

// isIterSeqRef reports whether r is `iter.Seq[V]`.
func isIterSeqRef(r *node.TypeRef) bool {
	if r == nil {
		return false
	}
	return r.Name == "Seq" && r.Package == "iter"
}

// isIterSeq2Ref reports whether r is `iter.Seq2[K, V]`.
func isIterSeq2Ref(r *node.TypeRef) bool {
	if r == nil {
		return false
	}
	return r.Name == "Seq2" && r.Package == "iter"
}
