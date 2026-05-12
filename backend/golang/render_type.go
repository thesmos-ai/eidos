// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package golang

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"go.thesmos.sh/eidos/emit"
)

// ErrUnsupportedRef is returned by [renderState.renderType] when
// called with an [emit.Ref] kind the current funcmap can't render,
// or by [internalTargetName] when a [emit.TypeRef] points at a
// target kind whose name can't yet be extracted. The wrapped
// message names the concrete Go type so diagnostics attribute the
// gap precisely.
var ErrUnsupportedRef = errors.New("backend/golang: unsupported Ref")

// renderType produces the Go source spelling for r. Supported kinds:
//
//   - [emit.BuiltinRef] — rendered as the builtin's Name.
//   - [emit.ExternalRef] — rendered as "<alias>.<Name>", with
//     <alias> obtained by registering the package path via the
//     state's [writer.ImportSet].
//   - [emit.TypeRef] — rendered as the unqualified name of the
//     target node (TypeRef is same-package by contract).
//   - [emit.CompositeRef] — dispatched to [renderState.renderComposite]
//     for the per-shape rendering.
//
// Other ref kinds return [ErrUnsupportedRef] wrapped with the
// concrete Go type.
//
// `renderType` is one of the reserved canonical-render funcmap
// entries — plugin overrides are rejected at Build time.
func (s *renderState) renderType(r emit.Ref) (string, error) {
	switch typed := r.(type) {
	case *emit.BuiltinRef:
		return typed.Name, nil
	case *emit.ExternalRef:
		alias, err := s.imports.Imp(typed.Package)
		if err != nil {
			return "", fmt.Errorf("backend/golang: renderType: %w", err)
		}
		return alias + "." + typed.Name, nil
	case *emit.TypeRef:
		return internalTargetName(typed.Target)
	case *emit.CompositeRef:
		return s.renderComposite(typed)
	default:
		return "", fmt.Errorf("%w: %T", ErrUnsupportedRef, r)
	}
}

// renderComposite dispatches on the [emit.CompositeRef.Shape] and
// returns the Go source spelling for the composite. All documented
// shapes are wired: Pointer (`*T`), Slice (`[]T`), Array (`[N]T`),
// Map (`map[K]V`), Func (`func(P) R`), and Union (`A | ~B | C`).
// Unknown shape values surface as [ErrUnsupportedRef] wrapped with
// the offending shape — a future variant added to the discriminator
// would land here.
func (s *renderState) renderComposite(r *emit.CompositeRef) (string, error) {
	switch r.Shape {
	case emit.ShapePointer:
		elem, err := s.renderType(r.Elem)
		if err != nil {
			return "", err
		}
		return "*" + elem, nil
	case emit.ShapeSlice:
		elem, err := s.renderType(r.Elem)
		if err != nil {
			return "", err
		}
		return "[]" + elem, nil
	case emit.ShapeArray:
		elem, err := s.renderType(r.Elem)
		if err != nil {
			return "", err
		}
		return "[" + strconv.Itoa(r.ArrayLen) + "]" + elem, nil
	case emit.ShapeMap:
		key, err := s.renderType(r.MapKey)
		if err != nil {
			return "", err
		}
		val, err := s.renderType(r.MapValue)
		if err != nil {
			return "", err
		}
		return "map[" + key + "]" + val, nil
	case emit.ShapeFunc:
		return s.renderFuncShape(r.FuncParams, r.FuncReturns)
	case emit.ShapeUnion:
		return s.renderUnion(r.UnionTerms)
	default:
		return "", fmt.Errorf("%w: composite shape %s", ErrUnsupportedRef, r.Shape)
	}
}

// renderFuncShape returns the Go source spelling of a function
// type: `func(P1, P2) R`, `func()`, `func(P) (R1, R2)`. Parameter
// and return types render through [renderState.renderType]; both
// lists are unnamed (the type-only form Go allows in field /
// variable / parameter declarations of function type).
func (s *renderState) renderFuncShape(params, returns []emit.Ref) (string, error) {
	paramParts := make([]string, 0, len(params))
	for _, p := range params {
		r, err := s.renderType(p)
		if err != nil {
			return "", err
		}
		paramParts = append(paramParts, r)
	}
	retText, err := s.renderReturns(emit.AnonReturns(returns...))
	if err != nil {
		return "", err
	}
	out := "func(" + strings.Join(paramParts, ", ") + ")"
	if retText != "" {
		out += " " + retText
	}
	return out, nil
}

// renderUnion produces the Go union-constraint spelling for a
// `T1 | T2 | ~T3` sequence: terms joined by " | ", with the
// approximation marker `~` prefixing terms whose Approx flag is
// set. Empty term slices yield the empty string — the caller
// (typically [renderState.renderTypeParams]) is responsible for
// treating an empty constraint as a programming error if relevant.
func (s *renderState) renderUnion(terms []emit.UnionTerm) (string, error) {
	parts := make([]string, 0, len(terms))
	for _, t := range terms {
		rendered, err := s.renderType(t.Type)
		if err != nil {
			return "", err
		}
		if t.Approx {
			rendered = "~" + rendered
		}
		parts = append(parts, rendered)
	}
	return strings.Join(parts, " | "), nil
}

// internalTargetName returns the unqualified declaration name of a
// [emit.TypeRef] target node — the spelling used when the target
// lives in the same Go package as the referring entity. Returns
// [ErrUnsupportedRef] wrapped with the concrete Go type when the
// target is a kind whose name can't be extracted.
func internalTargetName(n emit.Node) (string, error) {
	switch t := n.(type) {
	case *emit.Struct:
		return t.Name, nil
	case *emit.Interface:
		return t.Name, nil
	case *emit.Alias:
		return t.Name, nil
	case *emit.Enum:
		return t.Name, nil
	case *emit.Function:
		return t.Name, nil
	default:
		return "", fmt.Errorf("%w: TypeRef target %T", ErrUnsupportedRef, n)
	}
}
