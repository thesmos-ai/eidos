// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package golang

import (
	"errors"
	"go/ast"

	"go.thesmos.sh/eidos/core/directive"
)

// docsAndDirectives extracts the doc-comment block and directive
// list for a declaration. Doc lines come from the spec's own
// comment group when present, falling back to the surrounding
// block-level group. Directives use the pipeline-supplied
// [directive.Parser] from [plugin.FrontendContext.Parser] — the
// frontend does no comment parsing of its own.
func (c *converter) docsAndDirectives(specDoc, blockDoc *ast.CommentGroup) ([]string, []*directive.Directive) {
	docSrc := preferred(specDoc, blockDoc)
	docs := docLinesFromCommentGroup(docSrc)

	var dirs []*directive.Directive
	// Directives can appear in either the spec doc or the block
	// doc; walk both rather than the preferred-only fallback used
	// for doc lines, because a single block decl may carry block-
	// level directives plus spec-level ones.
	for _, g := range []*ast.CommentGroup{blockDoc, specDoc} {
		if g == nil {
			continue
		}
		dirs = append(dirs, c.parseDirectives(g)...)
	}
	return docs, dirs
}

// parseDirectives iterates the comments in g and returns every
// [directive.Directive] the pipeline parser successfully parses.
// Non-directive comments are silently ignored (the parser's
// ParseComment returns (nil, nil) for them); parse errors are
// emitted as positioned diagnostics on the run's diag sink and the
// offending comment is skipped.
//
// A nil g returns nil without consulting the parser — callers
// thread [ast.Field.Doc], which is nil whenever the field carries
// no doc comment, and dispatching to a no-op keeps the per-field
// hot path branch-free.
func (c *converter) parseDirectives(g *ast.CommentGroup) []*directive.Directive {
	if g == nil {
		return nil
	}
	ps := c.ctx.Diag.For(FrontendName)
	parser := c.ctx.Parser
	var out []*directive.Directive
	for _, comment := range g.List {
		pos := posOf(c.fset, comment.Pos())
		d, err := parser.ParseComment(comment.Text, pos)
		if err != nil {
			if errors.Is(err, directive.ErrMalformedDirective) {
				ps.Errorf(pos, "directive parse: %v", err)
			}
			continue
		}
		if d != nil {
			out = append(out, d)
		}
	}
	return out
}

// preferred returns the first non-nil, non-empty comment group from
// the supplied alternatives. Used by callers that want the spec's
// own doc preferred over the surrounding block's doc.
func preferred(groups ...*ast.CommentGroup) *ast.CommentGroup {
	for _, g := range groups {
		if g != nil && len(g.List) > 0 {
			return g
		}
	}
	return nil
}
