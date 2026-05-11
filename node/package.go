// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package node

import "go.thesmos.sh/eidos/core/directive"

// Package is a top-level container for the declarations found in one
// source package. Frontends produce one Package per source package;
// the store indexes Packages and their members by qualified name.
//
// Declarations live on Package as flat slices regardless of which
// source [File] declared them — the package-as-flat-namespace view
// most generators expect. Per-file metadata (build tags, file-level
// doc comments, that file's import block) is reachable via
// [Package.Files], and any node's source file is accessible via its
// [position.Pos.File].
//
// Package-level doc comments (Go's `package foo` doc) live in
// [BaseNode.DocLines]; the package's directive comments live in
// [BaseNode.DirectiveList] alongside.
type Package struct {
	BaseNode

	// Name is the short package name (Go's `package name`).
	Name string `json:"name"`

	// Path is the import path (Go's "github.com/foo/bar/baz").
	Path string `json:"path,omitempty"`

	// Files are the source files contributing to this package, in
	// the order the frontend visited them.
	Files []*File `json:"files,omitempty"`

	// Imports is the deduplicated union of every File's imports.
	// Each Import here has Owner pointing back at the Package; the
	// per-file Import instances live on each [File.Imports].
	Imports []*Import `json:"imports,omitempty"`

	// Structs are the struct declarations in the package.
	Structs []*Struct `json:"structs,omitempty"`

	// Interfaces are the interface declarations in the package.
	Interfaces []*Interface `json:"interfaces,omitempty"`

	// Functions are the standalone (non-method) function
	// declarations.
	Functions []*Function `json:"functions,omitempty"`

	// Variables are the package-level var declarations.
	Variables []*Variable `json:"variables,omitempty"`

	// Constants are the package-level const declarations not
	// gathered into an Enum.
	Constants []*Constant `json:"constants,omitempty"`

	// Enums are the idiomatic enums detected by the frontend or
	// declared as first-class enums in languages that support them.
	Enums []*Enum `json:"enums,omitempty"`

	// Aliases are the type-alias and type-definition declarations.
	Aliases []*Alias `json:"aliases,omitempty"`
}

// Kind returns [KindPackage].
func (*Package) Kind() directive.Kind { return KindPackage }

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

// FunctionByName returns the function named name, or nil when absent.
func (p *Package) FunctionByName(name string) *Function {
	for _, f := range p.Functions {
		if f.Name == name {
			return f
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

// VariableByName returns the variable named name, or nil when absent.
func (p *Package) VariableByName(name string) *Variable {
	for _, v := range p.Variables {
		if v.Name == name {
			return v
		}
	}
	return nil
}

// ConstantByName returns the constant named name, or nil when absent.
// Constants gathered into an [Enum] do not appear here; query the
// owning Enum's variants instead.
func (p *Package) ConstantByName(name string) *Constant {
	for _, c := range p.Constants {
		if c.Name == name {
			return c
		}
	}
	return nil
}

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
