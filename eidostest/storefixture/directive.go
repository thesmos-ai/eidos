// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package storefixture

import (
	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/core/position"
)

// Directive constructs a parsed [directive.Directive] suitable for
// attaching to a fixture node via [StructBuilder.Directive],
// [MethodBuilder.Directive], and equivalents.
//
// The result mirrors what a frontend's directive parser produces:
// Name, Negated, positional Args, and keyword KV are all populated
// from the supplied options. Raw is left empty — tests rarely care
// about it and a real frontend would set it to the source text.
func Directive(name directive.Name, opts ...DirectiveOption) *directive.Directive {
	d := &directive.Directive{Name: name, KV: map[string]string{}}
	for _, opt := range opts {
		opt(d)
	}
	return d
}

// DirectiveOption mutates a [directive.Directive] under construction.
// Options are applied left-to-right; later options overwrite earlier
// ones when they target the same field.
type DirectiveOption func(*directive.Directive)

// Negated marks the directive as a `-gen:` form. The default — without
// applying Negated — is the `+gen:` form.
func Negated() DirectiveOption {
	return func(d *directive.Directive) { d.Negated = true }
}

// Arg appends a positional argument.
func Arg(arg string) DirectiveOption {
	return func(d *directive.Directive) { d.Args = append(d.Args, arg) }
}

// KV records a key=value argument.
func KV(key, value string) DirectiveOption {
	return func(d *directive.Directive) { d.KV[key] = value }
}

// At records the directive's source position.
func At(p position.Pos) DirectiveOption {
	return func(d *directive.Directive) { d.Pos = p }
}
