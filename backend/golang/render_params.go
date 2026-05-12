// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package golang

import (
	"errors"
	"fmt"
	"strings"

	"go.thesmos.sh/eidos/emit"
)

// ErrMixedNamedParams is returned by [renderState.renderParams]
// when called with a parameter list that mixes named and unnamed
// entries — forbidden by Go's grammar ("Within a list of
// parameters or results, the names must either all be present or
// all be absent"). The wrapped message names the offending entity
// so generators can locate and fix the inconsistency.
var ErrMixedNamedParams = errors.New("backend/golang: param list mixes named and unnamed entries")

// renderParams produces the parenthesised parameter list of a Go
// function or method signature. Each entry is `Name renderType(Type)`
// when names are present and `renderType(Type)` when names are
// absent. Variadic parameters receive the `...` prefix on their
// type. An empty parameter list renders as `()`.
//
// Mixed-named parameters (a list where some entries have names and
// others don't) violate Go's grammar; this case fails with
// [ErrMixedNamedParams] wrapped with the parameter-name context.
//
// `renderParams` is one of the reserved canonical-render funcmap
// entries — plugin overrides are rejected at Build time.
func (s *renderState) renderParams(params []*emit.Param) (string, error) {
	if len(params) == 0 {
		return "()", nil
	}
	var anyNamed, anyUnnamed bool
	for _, p := range params {
		if p.Name == "" {
			anyUnnamed = true
		} else {
			anyNamed = true
		}
	}
	if anyNamed && anyUnnamed {
		return "", fmt.Errorf("%w: %s", ErrMixedNamedParams, paramListSummary(params))
	}
	parts := make([]string, 0, len(params))
	for _, p := range params {
		entry, err := s.renderParamEntry(p)
		if err != nil {
			return "", err
		}
		parts = append(parts, entry)
	}
	return "(" + strings.Join(parts, ", ") + ")", nil
}

// renderParamEntry renders a single parameter as it appears inside
// the parenthesised list — name + type (or just type for anonymous
// entries), with `...` prefixing the type for variadic params.
func (s *renderState) renderParamEntry(p *emit.Param) (string, error) {
	t, err := s.renderType(p.Type)
	if err != nil {
		return "", err
	}
	if p.Variadic {
		t = "..." + t
	}
	if p.Name == "" {
		return t, nil
	}
	return p.Name + " " + t, nil
}

// paramListSummary returns a short, comma-separated list of the
// parameter names (using `_` for anonymous entries) suitable for
// inclusion in diagnostic messages.
func paramListSummary(params []*emit.Param) string {
	names := make([]string, 0, len(params))
	for _, p := range params {
		if p.Name == "" {
			names = append(names, "_")
		} else {
			names = append(names, p.Name)
		}
	}
	return strings.Join(names, ", ")
}

// renderReceiver produces the receiver clause of a Go method
// signature — `(name Type)` when [emit.Method.ReceiverName] is set,
// `(Type)` for the anonymous receiver form, or the empty string
// when the method carries no [emit.Method.Receiver]. The empty
// case applies to interface methods, which are rendered nested
// inside their owning interface's template; standalone-rendered
// methods always carry a Receiver.
//
// `renderReceiver` is one of the reserved canonical-render funcmap
// entries — plugin overrides are rejected at Build time.
func (s *renderState) renderReceiver(m *emit.Method) (string, error) {
	if m == nil || m.Receiver == nil {
		return "", nil
	}
	t, err := s.renderType(m.Receiver)
	if err != nil {
		return "", err
	}
	if m.ReceiverName == "" {
		return "(" + t + ")", nil
	}
	return "(" + m.ReceiverName + " " + t + ")", nil
}

// renderReturns produces the return-clause text of a Go function
// or method signature, following the three-case truth table:
//
//   - Zero returns → empty string (no clause).
//   - Exactly one unnamed return → bare `renderType(Type)` (no
//     parentheses).
//   - Multiple returns → parenthesised, comma-separated list.
//
// `renderReturns` is one of the reserved canonical-render funcmap
// entries — plugin overrides are rejected at Build time.
func (s *renderState) renderReturns(returns []emit.Ref) (string, error) {
	switch len(returns) {
	case 0:
		return "", nil
	case 1:
		return s.renderType(returns[0])
	}
	parts := make([]string, 0, len(returns))
	for _, r := range returns {
		t, err := s.renderType(r)
		if err != nil {
			return "", err
		}
		parts = append(parts, t)
	}
	return "(" + strings.Join(parts, ", ") + ")", nil
}
