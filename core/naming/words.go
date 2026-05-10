// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package naming

import "unicode"

// Words splits s into its component words.
//
// Boundaries are inserted at:
//
//   - Separator runes (_, -, ., space, slash, tab) — consumed, never emitted.
//   - Lower-to-upper transitions (helloWorld → ["hello", "World"]).
//   - Acronym-then-word boundaries: an upper-case run followed by an
//     upper-case rune that is itself followed by a lower-case rune —
//     "HTTPServer" → ["HTTP", "Server"], "URLPath" → ["URL", "Path"].
//
// Digits never trigger a boundary; they belong to the surrounding word.
// "Version2" stays one word; consumers that want "Version_2" can
// pre-separate the digit.
//
// An empty or separator-only input returns nil.
//
// Words splits by structural rules only; the receiver's initialism set
// is used by the case converters that build on top of Words, not by
// the splitter itself. The method is defined on Caser purely for API
// symmetry with the converters.
func (*Caser) Words(s string) []string {
	if s == "" {
		return nil
	}
	var words []string
	var current []rune
	flush := func() {
		if len(current) > 0 {
			words = append(words, string(current))
			current = nil
		}
	}
	runes := []rune(s)
	for i, r := range runes {
		if isSeparator(r) {
			flush()
			continue
		}
		if i > 0 && shouldBreakBefore(runes, i) {
			flush()
		}
		current = append(current, r)
	}
	flush()
	return words
}

// shouldBreakBefore returns true when a word boundary should be
// inserted before runes[i]. The caller has already established that
// i > 0 and runes[i] is not a separator.
func shouldBreakBefore(runes []rune, i int) bool {
	cur := runes[i]
	prev := runes[i-1]
	if unicode.IsLower(prev) && unicode.IsUpper(cur) {
		return true
	}
	if unicode.IsUpper(prev) && unicode.IsUpper(cur) && i+1 < len(runes) && unicode.IsLower(runes[i+1]) {
		return true
	}
	return false
}

// isSeparator reports whether r is one of the recognised word-separator
// runes. Separators are consumed by the splitter — they appear in
// neither the words slice nor any subsequent rendering.
func isSeparator(r rune) bool {
	switch r {
	case '_', '-', '.', ' ', '\t', '/':
		return true
	default:
		return false
	}
}

// upperRune is a thin wrapper over unicode.ToUpper, named to make the
// case-conversion intent obvious at call sites in this package.
func upperRune(r rune) rune { return unicode.ToUpper(r) }

// lowerRune is the symmetrical lower-case wrapper.
func lowerRune(r rune) rune { return unicode.ToLower(r) }
