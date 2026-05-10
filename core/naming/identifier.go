// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package naming

import (
	"strings"
	"unicode"
)

// Identifier sanitises s into a syntactically valid Go-style identifier:
// letters and digits survive, underscores survive, every other rune is
// replaced with an underscore, and a leading digit is prefixed with an
// underscore. An empty input returns the single underscore "_".
//
// The function does not change case; combine with [Pascal], [Camel],
// or any other style converter when a specific shape is required.
//
// Reserved words (Go keywords, language built-ins) are not handled here —
// callers that need that behaviour layer it on top, typically inside a
// language-specific frontend or backend helper.
func Identifier(s string) string {
	if s == "" {
		return "_"
	}
	var b strings.Builder
	b.Grow(len(s) + 1)
	for i, r := range s {
		switch {
		case r == '_' || unicode.IsLetter(r):
			b.WriteRune(r)
		case unicode.IsDigit(r):
			if i == 0 {
				b.WriteByte('_')
			}
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	return b.String()
}
