// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package routing supplies the canonical [emit.Target] constructors
// reference plugins use when laying generated decls into a project
// tree. Two layouts are supported today: alongside-source (the
// generated file lives next to the source it derives from) and
// centralised (every plugin's output lands in one shared output
// package).
//
// The helpers exist because plugin code shouldn't be re-implementing
// "what's my output filename / package / import path" inline. Each
// plugin used to inline a Target struct literal at 3+ emission
// sites; this package collapses every site to one call. The Router
// phase still owns directory resolution and `+gen:out` directive
// honoring — these helpers produce the plugin-default Target the
// Router then refines.
package routing

import (
	"strings"

	"go.thesmos.sh/eidos/core/position"
	"go.thesmos.sh/eidos/core/srcfile"
	"go.thesmos.sh/eidos/emit"
)

// AlongsideSource returns the canonical [emit.Target] for a decl
// emitted alongside the source it derives from.
//
// The returned Target leaves [emit.Target.Dir] empty so the
// pipeline's Router phase fills it from srcPos at routing time.
// [emit.Target.Filename] combines the source file's basename with
// suffix via [srcfile.WithSuffix] — `+gen:out` on the originating
// source node overrides this at the Router phase.
// [emit.Target.Package] and [emit.Target.ImportPath] inherit from
// pkgName / pkgPath so the backend's same-package import-elision
// treats refs into the source package as bare names rather than
// qualified imports.
//
// Plugins call AlongsideSource for every emission whose output
// belongs next to the source — the alongside-source layout is the
// default for generator plugins that don't configure an
// OutputPackage.
func AlongsideSource(srcPos position.Pos, srcName, pkgName, pkgPath, suffix string) emit.Target {
	return emit.Target{
		Filename:   srcfile.WithSuffix(srcPos, srcName, suffix),
		Package:    pkgName,
		ImportPath: pkgPath,
	}
}

// Centralised returns the canonical [emit.Target] for a decl
// emitted into a shared output package — every plugin's output
// landing in one directory keyed by outputPackage.
//
// [emit.Target.Filename] is srcName lower-cased plus suffix so
// multiple per-entity decls compose into one file when their
// source names align. [emit.Target.Dir] and [emit.Target.Package]
// both equal outputPackage so the rendered package clause matches
// the directory. [emit.Target.ImportPath] is left empty —
// centralised layout assumes the package has no canonical import
// path the renderer needs to elide against.
func Centralised(srcName, outputPackage, suffix string) emit.Target {
	return emit.Target{
		Dir:      outputPackage,
		Filename: strings.ToLower(srcName) + suffix,
		Package:  outputPackage,
	}
}
