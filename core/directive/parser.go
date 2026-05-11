// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package directive

import (
	"errors"
	"fmt"
	"strings"
	"unicode"

	"go.thesmos.sh/eidos/core/position"
)

// ErrMalformedDirective is returned by [Parser.Parse] when the input
// is not a well-formed directive. The wrapping error message describes
// the specific failure (missing prefix, missing name, unterminated
// quoted string, unknown escape, and so on).
var ErrMalformedDirective = errors.New("directive: malformed")

// ErrInvalidPrefix is returned by [NewParser] when the supplied prefix
// is empty or contains characters reserved by the directive grammar
// (whitespace, `:`, `+`, `-`). Prefixes must be valid identifier-like
// strings — typically a project tag such as "gen" or "testkit".
var ErrInvalidPrefix = errors.New("directive: invalid parser prefix")

// Parser turns directive-bearing comment lines into [Directive]
// values. The Prefix selects the family the parser recognises;
// for a prefix of "gen", the accepted line forms are:
//
//	gen:NAME ARGS…    -- neutral set form
//	+gen:NAME ARGS…   -- explicit set form
//	-gen:NAME ARGS…   -- negated form (Negated = true)
//
// Each form is otherwise identical: the directive name and argument
// list parse the same way. The leading sigil only influences the
// resulting [Directive.Negated] flag.
//
// Parsers are immutable after construction. Share a single Parser
// across a pipeline rather than constructing one per call.
type Parser struct {
	prefix    string
	neutral   string
	setForm   string
	negateFor string
}

// NewParser returns a Parser configured for the given prefix.
// The prefix must be non-empty and may not contain whitespace or
// any of `:`, `+`, `-`. Returns [ErrInvalidPrefix] wrapped with the
// offending value otherwise.
func NewParser(prefix string) (*Parser, error) {
	if !isValidPrefix(prefix) {
		return nil, fmt.Errorf("%w: %q", ErrInvalidPrefix, prefix)
	}
	return newParser(prefix), nil
}

// newParser constructs a Parser for prefix without revalidating it.
// Internal callers with a known-valid prefix (the [DefaultPrefix]
// constant) use this directly; external callers route through
// [NewParser], which validates first.
func newParser(prefix string) *Parser {
	return &Parser{
		prefix:    prefix,
		neutral:   prefix + ":",
		setForm:   "+" + prefix + ":",
		negateFor: "-" + prefix + ":",
	}
}

// Prefix returns the configured directive prefix (without sigils or
// colon). Useful for diagnostics.
func (p *Parser) Prefix() string { return p.prefix }

// Parse parses a single directive body. The input is the comment
// content with leading comment markers (`//`, `/* */`) already
// stripped — typically what [Parser.ParseComment] passes in. pos
// identifies the start of the directive in source for downstream
// diagnostics.
//
// Returns [ErrMalformedDirective] wrapped with the specific failure
// when the body is not a directive or contains syntactic errors.
func (p *Parser) Parse(text string, pos position.Pos) (*Directive, error) {
	trimmed := strings.TrimLeft(text, " \t")
	negated, body, ok := p.stripPrefix(trimmed)
	if !ok {
		return nil, fmt.Errorf(
			"%w: missing %q, %q, or %q prefix",
			ErrMalformedDirective, p.neutral, p.setForm, p.negateFor,
		)
	}
	raw := body

	l := &lexer{src: body}
	name := l.readBareTo(" \t=")
	if name == "" {
		return nil, fmt.Errorf("%w: empty directive name", ErrMalformedDirective)
	}
	if !isValidName(name) {
		return nil, fmt.Errorf("%w: invalid directive name %q", ErrMalformedDirective, name)
	}

	d := &Directive{
		Name:    Name(name),
		Negated: negated,
		Raw:     raw,
		Pos:     pos,
		KV:      map[string]string{},
	}

	for {
		l.skipWS()
		if l.eof() {
			break
		}
		key, value, isKV, err := l.readArg()
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrMalformedDirective, err)
		}
		if isKV {
			d.KV[key] = value
		} else {
			d.Args = append(d.Args, key)
		}
	}
	return d, nil
}

// ParseComment is a convenience over [Parser.Parse] that strips
// leading `//` and surrounding `/* */` Go-style comment markers and
// returns (nil, nil) for comments that don't carry a directive of
// this Parser's prefix.
//
// Whitespace inside the comment is trimmed before the prefix check,
// so "// gen:foo", "//gen:foo", and "/* +gen:foo */" all parse when
// the Parser is configured for prefix "gen".
func (p *Parser) ParseComment(comment string, pos position.Pos) (*Directive, error) {
	body := strings.TrimSpace(comment)
	body = strings.TrimPrefix(body, "//")
	body = strings.TrimPrefix(body, "/*")
	body = strings.TrimSuffix(body, "*/")
	body = strings.TrimSpace(body)
	if body == "" {
		return nil, nil //nolint:nilnil // empty comment is intentionally a no-directive
	}
	if _, _, ok := p.stripPrefix(body); !ok {
		return nil, nil //nolint:nilnil // non-directive comments are not errors
	}
	return p.Parse(body, pos)
}

// stripPrefix tests body against the three recognised prefix forms.
// On match, returns (negated, body-with-prefix-removed, true).
// On no match, returns (false, body, false).
func (p *Parser) stripPrefix(body string) (negated bool, remainder string, ok bool) {
	switch {
	case strings.HasPrefix(body, p.negateFor):
		return true, body[len(p.negateFor):], true
	case strings.HasPrefix(body, p.setForm):
		return false, body[len(p.setForm):], true
	case strings.HasPrefix(body, p.neutral):
		return false, body[len(p.neutral):], true
	}
	return false, body, false
}

// isValidPrefix reports whether s is a syntactically valid Parser
// prefix: non-empty, no whitespace, no `:`, no `+`, no `-`.
func isValidPrefix(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r == ':' || r == '+' || r == '-' || unicode.IsSpace(r) {
			return false
		}
	}
	return true
}

// isValidName reports whether s is a syntactically valid directive
// name: starts with a letter, contains only letters / digits /
// underscores / hyphens. Dots are not permitted in names — they are
// reserved for KV keys (so `+gen:meta shape.writer=true` works).
// The caller guarantees s is non-empty (Parse returns early on the
// empty case before calling here).
func isValidName(s string) bool {
	if !isNameStart(rune(s[0])) {
		return false
	}
	for i := 1; i < len(s); i++ {
		if !isNameRune(rune(s[i])) {
			return false
		}
	}
	return true
}

// isNameStart reports whether r may begin a directive name.
func isNameStart(r rune) bool { return unicode.IsLetter(r) }

// isNameRune reports whether r may continue a directive name.
func isNameRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-'
}

// lexer is the small string scanner used by [Parser.Parse] for
// directive argument tokenisation.
type lexer struct {
	src string
	pos int
}

// eof reports whether the lexer has consumed all input.
func (l *lexer) eof() bool { return l.pos >= len(l.src) }

// skipWS advances past any whitespace.
func (l *lexer) skipWS() {
	for !l.eof() && (l.src[l.pos] == ' ' || l.src[l.pos] == '\t') {
		l.pos++
	}
}

// readBareTo reads a bare (unquoted) token up to but not including
// any rune in stopAt or whitespace. The stop rune is not consumed.
func (l *lexer) readBareTo(stopAt string) string {
	start := l.pos
	for !l.eof() {
		c := l.src[l.pos]
		if c == ' ' || c == '\t' {
			break
		}
		if strings.IndexByte(stopAt, c) >= 0 {
			break
		}
		l.pos++
	}
	return l.src[start:l.pos]
}

// readQuoted reads a double-quoted string starting at the current
// position. The caller guarantees the current byte is the opening
// `"`. Supports \", \\, \n, and \t escapes; rejects others. The
// closing `"` is consumed.
func (l *lexer) readQuoted() (string, error) {
	l.pos++ // opening quote
	var b strings.Builder
	for !l.eof() {
		c := l.src[l.pos]
		switch c {
		case '\\':
			l.pos++
			if l.eof() {
				return "", errors.New("unterminated escape in quoted string")
			}
			esc := l.src[l.pos]
			switch esc {
			case '"', '\\':
				b.WriteByte(esc)
			case 'n':
				b.WriteByte('\n')
			case 't':
				b.WriteByte('\t')
			default:
				return "", fmt.Errorf("unknown escape \\%c", esc)
			}
			l.pos++
		case '"':
			l.pos++ // closing quote
			return b.String(), nil
		default:
			b.WriteByte(c)
			l.pos++
		}
	}
	return "", errors.New("unterminated quoted string")
}

// readArg reads one argument from the current position. Returns the
// key (or positional value), the KV value (only meaningful when
// isKV=true), an isKV flag, and any lexical error.
func (l *lexer) readArg() (key, value string, isKV bool, err error) {
	first := l.readBareTo("=")
	if !l.eof() && l.src[l.pos] == '=' {
		l.pos++ // consume '='
		val, valueErr := l.readValue()
		if valueErr != nil {
			return "", "", false, valueErr
		}
		return first, val, true, nil
	}
	return first, "", false, nil
}

// readValue reads a KV value at the current position: either a
// double-quoted string (with escapes) or a bare token up to the next
// whitespace.
func (l *lexer) readValue() (string, error) {
	if !l.eof() && l.src[l.pos] == '"' {
		return l.readQuoted()
	}
	return l.readBareTo(""), nil
}
