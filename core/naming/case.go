// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package naming

import "strings"

// Pascal converts s to PascalCase.
//
// Each word's first rune is upper-cased and the rest lower-cased,
// except that words whose upper-cased form is a recognised initialism
// (see [CommonInitialisms]) are upper-cased in full and already-all-upper
// inputs are preserved. An empty or separator-only input returns "".
func (c *Caser) Pascal(s string) string {
	words := c.Words(s)
	if len(words) == 0 {
		return ""
	}
	var b strings.Builder
	for _, w := range words {
		b.WriteString(c.titleWord(w))
	}
	return b.String()
}

// Camel converts s to camelCase.
//
// The first word is fully lower-cased; subsequent words are title-cased
// using the same rules as [Caser.Pascal] (initialism preservation
// included). The first word's lower-casing is unconditional, so
// "URLPath" → "urlPath", "HTTPServer" → "httpServer".
func (c *Caser) Camel(s string) string {
	words := c.Words(s)
	if len(words) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(strings.ToLower(words[0]))
	for _, w := range words[1:] {
		b.WriteString(c.titleWord(w))
	}
	return b.String()
}

// Snake converts s to snake_case (lower-case words joined by '_').
func (c *Caser) Snake(s string) string {
	return joinWith(c.Words(s), "_", strings.ToLower)
}

// ScreamingSnake converts s to SCREAMING_SNAKE_CASE (upper-case words
// joined by '_').
func (c *Caser) ScreamingSnake(s string) string {
	return joinWith(c.Words(s), "_", strings.ToUpper)
}

// Kebab converts s to kebab-case (lower-case words joined by '-').
func (c *Caser) Kebab(s string) string {
	return joinWith(c.Words(s), "-", strings.ToLower)
}

// ScreamingKebab converts s to SCREAMING-KEBAB-CASE (upper-case words
// joined by '-').
func (c *Caser) ScreamingKebab(s string) string {
	return joinWith(c.Words(s), "-", strings.ToUpper)
}

// Dot converts s to dot.case (lower-case words joined by '.').
func (c *Caser) Dot(s string) string {
	return joinWith(c.Words(s), ".", strings.ToLower)
}

// Title converts s to Title Case (each word title-cased per the rules
// of [Caser.Pascal], joined by single spaces).
func (c *Caser) Title(s string) string {
	words := c.Words(s)
	if len(words) == 0 {
		return ""
	}
	parts := make([]string, len(words))
	for i, w := range words {
		parts[i] = c.titleWord(w)
	}
	return strings.Join(parts, " ")
}

// joinWith applies transform to each word and joins the result with sep.
// Returns "" for an empty input slice. The transform is invoked exactly
// once per word.
func joinWith(words []string, sep string, transform func(string) string) string {
	if len(words) == 0 {
		return ""
	}
	parts := make([]string, len(words))
	for i, w := range words {
		parts[i] = transform(w)
	}
	return strings.Join(parts, sep)
}

// Words returns the component words of s using the default Caser. See
// [Caser.Words] for the splitting rules.
func Words(s string) []string { return Default().Words(s) }

// Pascal converts s to PascalCase using the default Caser.
func Pascal(s string) string { return Default().Pascal(s) }

// Camel converts s to camelCase using the default Caser.
func Camel(s string) string { return Default().Camel(s) }

// Snake converts s to snake_case using the default Caser.
func Snake(s string) string { return Default().Snake(s) }

// ScreamingSnake converts s to SCREAMING_SNAKE_CASE using the default Caser.
func ScreamingSnake(s string) string { return Default().ScreamingSnake(s) }

// Kebab converts s to kebab-case using the default Caser.
func Kebab(s string) string { return Default().Kebab(s) }

// ScreamingKebab converts s to SCREAMING-KEBAB-CASE using the default Caser.
func ScreamingKebab(s string) string { return Default().ScreamingKebab(s) }

// Dot converts s to dot.case using the default Caser.
func Dot(s string) string { return Default().Dot(s) }

// Title converts s to Title Case using the default Caser.
func Title(s string) string { return Default().Title(s) }
