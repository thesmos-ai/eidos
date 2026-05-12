// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package protobuf

import (
	"errors"
	"strings"

	"google.golang.org/protobuf/reflect/protoreflect"

	"go.thesmos.sh/eidos/core/diag"
	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/core/meta"
	"go.thesmos.sh/eidos/core/position"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/plugin"
)

// docLinesAndTrailing returns the docs-attachable lines and the
// trailing same-line comment for the supplied source location.
//
// Per the documented comment-attachment rule, leading comments
// land in [node.BaseNode.DocLines] with the line-comment marker
// stripped; trailing same-line comments do NOT enter DocLines and
// instead route to a per-host `proto.<host>.trailing_doc` meta
// key. Detached comment blocks immediately preceding the
// declaration also attach to DocLines — protocompile materialises
// them on the SourceLocation as LeadingDetachedComments alongside
// the LeadingComments string.
func docLinesAndTrailing(sl protoreflect.SourceLocation) ([]string, string) {
	docs := commentBlockLines(sl.LeadingComments)
	for _, detached := range sl.LeadingDetachedComments {
		docs = append(docs, commentBlockLines(detached)...)
	}
	trailing := strings.TrimSpace(sl.TrailingComments)
	return docs, trailing
}

// commentBlockLines splits raw — a comment block protocompile
// surfaces with the `//` or `/* */` markers stripped — into one
// entry per source line. The leading-space convention proto
// tooling preserves (a single space after `//` on each line) is
// trimmed so the produced DocLines are immediately usable as
// rendered documentation.
//
// Empty or whitespace-only input returns nil so callers don't
// have to special-case empty blocks before appending.
func commentBlockLines(raw string) []string {
	trimmed := strings.TrimRight(raw, "\n")
	if strings.TrimSpace(trimmed) == "" {
		return nil
	}
	lines := strings.Split(trimmed, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		out = append(out, strings.TrimPrefix(line, " "))
	}
	return out
}

// parseDirectiveBlock walks the lines of raw and asks the
// pipeline-supplied parser to extract directives from each line.
// Lines that don't carry the `+gen:` / `-gen:` prefixes parse to
// (nil, nil) and are silently ignored; the returned slice is the
// per-line concatenation in source order.
//
// Malformed-directive errors surface as positioned diagnostics
// attributed to the frontend's plugin name. The per-line offset
// rides on the diagnostic position so a typo five lines into a
// comment block anchors near the offending line rather than at
// the block's start. protocompile does not surface per-line
// positions for comment blocks, so the offset is approximate
// (block start + line index) — close enough for users to find
// the line in a text editor.
func parseDirectiveBlock(
	parser *directive.Parser, ps *diag.PluginSink,
	raw string, blockPos position.Pos,
) []*directive.Directive {
	if raw == "" || parser == nil {
		return nil
	}
	var out []*directive.Directive
	lineIdx := 0
	for line := range strings.SplitSeq(raw, "\n") {
		linePos := blockPos
		linePos.Line += lineIdx
		lineIdx++
		ds, err := parser.ParseComment(line, linePos)
		if err != nil {
			if errors.Is(err, directive.ErrMalformedDirective) && ps != nil {
				ps.Errorf(linePos, "directive parse: %v", err)
			}
			continue
		}
		out = append(out, ds...)
	}
	return out
}

// directivesFor parses every directive declared on sl's leading
// comments — both the immediate LeadingComments string and any
// LeadingDetachedComments blocks — using ctx's pipeline parser.
// Returns nil when ctx is nil (test paths that bypass the
// pipeline) or when no directives are present.
func directivesFor(
	ctx *plugin.FrontendContext, sl protoreflect.SourceLocation, pos position.Pos,
) []*directive.Directive {
	if ctx == nil || ctx.Parser == nil {
		return nil
	}
	ps := ctx.Diag.For(FrontendName)
	out := parseDirectiveBlock(ctx.Parser, ps, sl.LeadingComments, pos)
	for _, detached := range sl.LeadingDetachedComments {
		out = append(out, parseDirectiveBlock(ctx.Parser, ps, detached, pos)...)
	}
	return out
}

// attachStructDocs walks fd's source-location index for md and
// stamps any leading comments onto s.DocLines plus any trailing
// same-line comment under [MetaMessageTrailingDoc]. Leading-
// comment directives parsed from the same block route into
// s.DirectiveList so the pipeline's gate-fire step picks them up.
func attachStructDocs(
	ctx *plugin.FrontendContext, s *node.Struct,
	fd protoreflect.FileDescriptor, md protoreflect.MessageDescriptor,
) {
	sl := fd.SourceLocations().ByDescriptor(md)
	docs, trailing := docLinesAndTrailing(sl)
	if len(docs) > 0 {
		s.DocLines = append(s.DocLines, docs...)
	}
	pos := position.Pos{File: fd.Path(), Line: sl.StartLine + 1, Column: sl.StartColumn + 1}
	if dirs := directivesFor(ctx, sl, pos); len(dirs) > 0 {
		s.DirectiveList = append(s.DirectiveList, dirs...)
	}
	if trailing != "" {
		MetaMessageTrailingDoc.SetAt(
			s.Meta(), trailing, meta.AuthorityPlugin, FrontendName, pos,
		)
	}
}

// attachFieldDocs walks the source-location index for desc and
// stamps any leading comments onto f.DocLines plus any trailing
// same-line comment under [MetaFieldTrailingDoc]. Field-level
// directives parsed from the leading comments populate
// f.DirectiveList.
func attachFieldDocs(
	ctx *plugin.FrontendContext, f *node.Field,
	fd protoreflect.FileDescriptor, desc protoreflect.FieldDescriptor,
) {
	sl := fd.SourceLocations().ByDescriptor(desc)
	docs, trailing := docLinesAndTrailing(sl)
	if len(docs) > 0 {
		f.DocLines = append(f.DocLines, docs...)
	}
	pos := position.Pos{File: fd.Path(), Line: sl.StartLine + 1, Column: sl.StartColumn + 1}
	if dirs := directivesFor(ctx, sl, pos); len(dirs) > 0 {
		f.DirectiveList = append(f.DirectiveList, dirs...)
	}
	if trailing != "" {
		MetaFieldTrailingDoc.SetAt(
			f.Meta(), trailing, meta.AuthorityPlugin, FrontendName, pos,
		)
	}
}

// attachEnumDocs walks the source-location index for ed and
// stamps any leading comments onto e.DocLines plus any trailing
// same-line comment under [MetaEnumTrailingDoc]. Enum-level
// directives parsed from the leading comments populate
// e.DirectiveList.
func attachEnumDocs(
	ctx *plugin.FrontendContext, e *node.Enum,
	fd protoreflect.FileDescriptor, ed protoreflect.EnumDescriptor,
) {
	sl := fd.SourceLocations().ByDescriptor(ed)
	docs, trailing := docLinesAndTrailing(sl)
	if len(docs) > 0 {
		e.DocLines = append(e.DocLines, docs...)
	}
	pos := position.Pos{File: fd.Path(), Line: sl.StartLine + 1, Column: sl.StartColumn + 1}
	if dirs := directivesFor(ctx, sl, pos); len(dirs) > 0 {
		e.DirectiveList = append(e.DirectiveList, dirs...)
	}
	if trailing != "" {
		MetaEnumTrailingDoc.SetAt(
			e.Meta(), trailing, meta.AuthorityPlugin, FrontendName, pos,
		)
	}
}

// attachVariantDocs walks the source-location index for vd and
// stamps any leading comments onto v.DocLines plus any trailing
// same-line comment under [MetaEnumVariantTrailingDoc]. Variant-
// level directives parsed from the leading comments populate
// v.DirectiveList.
func attachVariantDocs(
	ctx *plugin.FrontendContext, v *node.EnumVariant,
	fd protoreflect.FileDescriptor, vd protoreflect.EnumValueDescriptor,
) {
	sl := fd.SourceLocations().ByDescriptor(vd)
	docs, trailing := docLinesAndTrailing(sl)
	if len(docs) > 0 {
		v.DocLines = append(v.DocLines, docs...)
	}
	pos := position.Pos{File: fd.Path(), Line: sl.StartLine + 1, Column: sl.StartColumn + 1}
	if dirs := directivesFor(ctx, sl, pos); len(dirs) > 0 {
		v.DirectiveList = append(v.DirectiveList, dirs...)
	}
	if trailing != "" {
		MetaEnumVariantTrailingDoc.SetAt(
			v.Meta(), trailing, meta.AuthorityPlugin, FrontendName, pos,
		)
	}
}
