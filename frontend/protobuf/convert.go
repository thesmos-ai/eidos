// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package protobuf

import (
	"path/filepath"
	"sort"
	"strings"

	"google.golang.org/protobuf/reflect/protoreflect"

	"go.thesmos.sh/eidos/core/diag"
	"go.thesmos.sh/eidos/core/position"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/plugin"
)

// convertFiles walks the resolved descriptor set and produces one
// [node.Package] per distinct proto package qualifier. Multiple
// source files sharing one proto package merge into a single
// Package entry — the merge is deterministic: descriptor iteration
// sorts by file path before any merging or stamping, so the
// produced graph is byte-stable across runs regardless of
// protocompile's internal resolve order.
//
// Each produced Package carries the cross-frontend provenance
// marker (`frontend = "protobuf"`) plus a [node.File] entry per
// contributing proto source with that file's imports recorded;
// the Package's Imports slice is the deduplicated union across
// every contributing file.
func convertFiles(
	ctx *plugin.FrontendContext, ps *diag.PluginSink,
	descriptors []protoreflect.FileDescriptor,
) {
	sorted := sortDescriptors(descriptors)
	pkgs := map[string]*node.Package{}
	order := []string{}
	for _, fd := range sorted {
		qualifier := string(fd.Package())
		pkg, exists := pkgs[qualifier]
		if !exists {
			pkg = newPackage(qualifier)
			stampFrontendMarker(pkg)
			pkgs[qualifier] = pkg
			order = append(order, qualifier)
		}
		appendFile(pkg, fd)
		stampFileOptions(ps, pkg, fd)
	}
	for _, qualifier := range order {
		pkg := pkgs[qualifier]
		dedupeImports(pkg)
		if err := ctx.Store.Nodes().AddPackage(pkg); err != nil {
			ps.Errorf(
				position.Pos{File: firstFilePath(pkg)},
				"protobuf: add package %s: %v", qualifier, err,
			)
		}
	}
}

// sortDescriptors returns descriptors sorted by [protoreflect.FileDescriptor.Path]
// so the per-package merge order is deterministic across runs.
// protocompile resolves files in dependency order, which can vary
// between runs that touch transitively-shared imports — sorting
// here normalises that variation.
func sortDescriptors(descriptors []protoreflect.FileDescriptor) []protoreflect.FileDescriptor {
	out := append([]protoreflect.FileDescriptor(nil), descriptors...)
	sort.Slice(out, func(i, j int) bool { return out[i].Path() < out[j].Path() })
	return out
}

// newPackage allocates a new [node.Package] for the supplied proto
// package qualifier. Name is the last dotted segment of qualifier
// (`simple` for `eidos.protobuf.testdata.simple`); Path is the full
// qualifier verbatim. A qualifier without a dot uses the whole
// string as Name; the empty qualifier produces a package with empty
// Name and Path (callers reject such inputs upstream).
func newPackage(qualifier string) *node.Package {
	name := qualifier
	if dot := strings.LastIndex(qualifier, "."); dot >= 0 {
		name = qualifier[dot+1:]
	}
	return &node.Package{Name: name, Path: qualifier}
}

// appendFile records one [node.File] on pkg derived from fd. Name
// is the file's basename; Path is the protocompile-resolved path
// (relative to the configured import root, matching the source-form
// `import "..."` declarations downstream consumers cross-reference
// against). The file's import declarations land on
// [node.File.Imports] in source order; the package-level
// deduplicated union is computed post-pass by [dedupeImports].
func appendFile(pkg *node.Package, fd protoreflect.FileDescriptor) {
	path := fd.Path()
	pkg.Files = append(pkg.Files, &node.File{
		Name:    filepath.Base(path),
		Path:    path,
		Imports: collectFileImports(fd),
	})
}

// firstFilePath returns the path of the first file recorded on pkg,
// or the package's path qualifier when no files have landed yet.
// Used as a fallback diagnostic position when a store-side
// AddPackage call fails — the file path anchors the message even
// though the failure isn't tied to any single declaration.
func firstFilePath(pkg *node.Package) string {
	if len(pkg.Files) > 0 {
		return pkg.Files[0].Path
	}
	return pkg.Path
}
