// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package emit

import (
	"go.thesmos.sh/eidos/core/kind"
)

// Package is the top-level container for emit declarations targeting
// one output language-level package. Generators produce Packages
// during the generator phase; backends consume Packages and render
// one or more output files per [Target] grouping.
//
// Like [node.Package], declarations live on Package as flat slices —
// the package-as-flat-namespace view backends expect — regardless of
// which output [File] they route to. The [Target] field on each
// declaration determines the destination file; [Package.Files]
// indexes those files for slot-based per-file cross-cutting content.
//
// Package mirrors the shape of [node.Package] (the source-side
// container) so generators that read a node.Package and produce an
// emit.Package have a one-to-one mapping for most fields. The
// divergences (Refs, Slots, Origin) are documented in the package
// overview.
type Package struct {
	BaseEmit

	// Name is the package's short name (Go's `package name`).
	Name string `json:"name"`

	// Path is the import path the package declares
	// ("github.com/foo/bar/baz"). Empty for languages without
	// import-path semantics.
	Path string `json:"path,omitempty"`

	// Dir is the output directory relative to the project root.
	// Backends combine this with each declaration's [Target.Filename]
	// to compute the final file path.
	Dir string `json:"dir,omitempty"`

	// Files are the per-file metadata entries — one File per unique
	// [Target] used by the declarations in this Package. Files carry
	// per-file imports and "top"/"bottom"/"init" slots.
	Files []*File `json:"files,omitempty"`

	// Imports is the package-wide deduplicated import view.
	// Per-file import blocks live on each [File.Imports].
	Imports []*Import `json:"imports,omitempty"`

	// Structs are the struct declarations.
	Structs []*Struct `json:"structs,omitempty"`

	// Interfaces are the interface declarations.
	Interfaces []*Interface `json:"interfaces,omitempty"`

	// Functions are the standalone (non-method) function
	// declarations.
	Functions []*Function `json:"functions,omitempty"`

	// Variables are the package-level var declarations.
	Variables []*Variable `json:"variables,omitempty"`

	// Constants are the package-level const declarations not
	// gathered into an [Enum].
	Constants []*Constant `json:"constants,omitempty"`

	// Enums are the enum declarations.
	Enums []*Enum `json:"enums,omitempty"`

	// Aliases are the type-alias and type-definition declarations.
	Aliases []*Alias `json:"aliases,omitempty"`

	slotMap
}

// Kind returns [KindPackage].
func (*Package) Kind() kind.Kind { return KindPackage }

// Slot returns the named slot, creating it lazily without an
// element-kind constraint. Package-level slots support cross-cutting
// contributions that span the whole package (registration tables,
// generated init blocks).
func (p *Package) Slot(name string) *Slot { return p.slot(p, name, "") }

// FileByName returns the file with the given basename, or nil when
// absent. Empty input returns nil.
func (p *Package) FileByName(name string) *File {
	if name == "" {
		return nil
	}
	for _, f := range p.Files {
		if f.Name == name {
			return f
		}
	}
	return nil
}

// FileByTarget returns the file matching the given Target, or nil
// when no such file exists. Useful for backends grouping
// declarations by Target.
func (p *Package) FileByTarget(t Target) *File {
	for _, f := range p.Files {
		if f.Target() == t {
			return f
		}
	}
	return nil
}

// ImportByPath returns the import with the given path from the
// package-level deduplicated view, or nil when absent. Empty input
// returns nil.
func (p *Package) ImportByPath(path string) *Import {
	if path == "" {
		return nil
	}
	for _, imp := range p.Imports {
		if imp.Path == path {
			return imp
		}
	}
	return nil
}

// StructByName returns the struct named name, or nil when absent.
func (p *Package) StructByName(name string) *Struct {
	for _, s := range p.Structs {
		if s.Name == name {
			return s
		}
	}
	return nil
}

// InterfaceByName returns the interface named name, or nil when
// absent.
func (p *Package) InterfaceByName(name string) *Interface {
	for _, i := range p.Interfaces {
		if i.Name == name {
			return i
		}
	}
	return nil
}

// FunctionByName returns the function named name, or nil when
// absent.
func (p *Package) FunctionByName(name string) *Function {
	for _, f := range p.Functions {
		if f.Name == name {
			return f
		}
	}
	return nil
}

// VariableByName returns the variable named name, or nil when absent.
func (p *Package) VariableByName(name string) *Variable {
	for _, v := range p.Variables {
		if v.Name == name {
			return v
		}
	}
	return nil
}

// ConstantByName returns the constant named name, or nil when
// absent. Constants gathered into an [Enum] do not appear here;
// query the owning Enum's variants instead.
func (p *Package) ConstantByName(name string) *Constant {
	for _, c := range p.Constants {
		if c.Name == name {
			return c
		}
	}
	return nil
}

// EnumByName returns the enum named name, or nil when absent.
func (p *Package) EnumByName(name string) *Enum {
	for _, e := range p.Enums {
		if e.Name == name {
			return e
		}
	}
	return nil
}

// AliasByName returns the alias named name, or nil when absent.
func (p *Package) AliasByName(name string) *Alias {
	for _, a := range p.Aliases {
		if a.Name == name {
			return a
		}
	}
	return nil
}

// FilesByTarget returns the files matching pred in declaration order.
func (p *Package) FilesByTarget(pred func(*File) bool) []*File {
	out := make([]*File, 0, len(p.Files))
	for _, f := range p.Files {
		if pred(f) {
			out = append(out, f)
		}
	}
	return out
}
