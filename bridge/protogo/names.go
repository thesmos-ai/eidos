// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package protogo

import (
	"strings"
	"unicode"
)

// commonInitialisms is the canonical set of acronyms the
// rendered Go form preserves in upper-case (matching the
// protobuf-Go canonical generator). Names containing these
// substrings keep them uppercase rather than running through
// the title-case rule that would produce `Id` / `Url` / etc.
//
//nolint:gochecknoglobals // immutable lookup table.
var commonInitialisms = map[string]bool{
	"ACL":   true,
	"API":   true,
	"ASCII": true,
	"CPU":   true,
	"CSS":   true,
	"DNS":   true,
	"EOF":   true,
	"GUID":  true,
	"HTML":  true,
	"HTTP":  true,
	"HTTPS": true,
	"ID":    true,
	"IP":    true,
	"JSON":  true,
	"LHS":   true,
	"QPS":   true,
	"RAM":   true,
	"RHS":   true,
	"RPC":   true,
	"SLA":   true,
	"SMTP":  true,
	"SQL":   true,
	"SSH":   true,
	"TCP":   true,
	"TLS":   true,
	"TTL":   true,
	"UDP":   true,
	"UI":    true,
	"UID":   true,
	"UUID":  true,
	"URI":   true,
	"URL":   true,
	"UTF8":  true,
	"VM":    true,
	"XML":   true,
	"XMPP":  true,
	"XSRF":  true,
	"XSS":   true,
}

// GoFieldName returns the Go-idiomatic PascalCase identifier for
// a proto-style snake_case name. The rule splits on underscores,
// title-cases each segment, then promotes any segment that is a
// recognised initialism to its uppercase form (`id` → `ID`,
// `url` → `URL`). Empty input produces empty output; segments
// of length 0 between consecutive underscores are skipped.
//
// Examples:
//
//	"user_id"     → "UserID"
//	"created_at"  → "CreatedAt"
//	"http_status" → "HTTPStatus"
//	"id"          → "ID"
func GoFieldName(name string) string {
	if name == "" {
		return ""
	}
	var b strings.Builder
	for segment := range strings.SplitSeq(name, "_") {
		if segment == "" {
			continue
		}
		upper := strings.ToUpper(segment)
		if commonInitialisms[upper] {
			b.WriteString(upper)
			continue
		}
		runes := []rune(segment)
		runes[0] = unicode.ToUpper(runes[0])
		b.WriteString(string(runes))
	}
	return b.String()
}

// GoPackageName returns the Go package clause derived from a
// proto package qualifier or from a `go_package` option value.
// The input forms it handles:
//
//   - `pkg/path;name`     → "name" (explicit semicolon-suffix)
//   - `pkg/path`          → last `/`-separated segment
//   - `dotted.qualifier`  → last dotted segment
//   - bare identifier     → unchanged
//
// The result is the identifier the Go backend emits in the
// `package <X>` clause. Empty input produces empty output;
// callers fall back to the proto package qualifier when
// GoPackageName returns empty.
func GoPackageName(value string) string {
	if value == "" {
		return ""
	}
	if _, after, ok := strings.Cut(value, ";"); ok {
		return after
	}
	if i := strings.LastIndexByte(value, '/'); i >= 0 {
		return value[i+1:]
	}
	if i := strings.LastIndexByte(value, '.'); i >= 0 {
		return value[i+1:]
	}
	return value
}

// GoImportPath returns the Go import path from a
// `go_package` option value. The semicolon-suffix form
// (`pkg/path;name`) trims the trailing identifier; the bare
// path form returns unchanged. Empty input produces empty output.
func GoImportPath(value string) string {
	if value == "" {
		return ""
	}
	if before, _, ok := strings.Cut(value, ";"); ok {
		return before
	}
	return value
}
