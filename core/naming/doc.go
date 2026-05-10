// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package naming converts identifiers between common case conventions.
//
// The core primitive is Words, which splits any identifier into its
// component words by recognising camelCase / PascalCase boundaries,
// acronym runs (HTTPServer → ["HTTP", "Server"]), and the standard
// separator characters (_, -, ., space, slash, tab).
//
// Eight output styles are provided:
//
//   - Pascal          → PascalCase
//   - Camel           → camelCase
//   - Snake           → snake_case
//   - ScreamingSnake  → SCREAMING_SNAKE_CASE
//   - Kebab           → kebab-case
//   - ScreamingKebab  → SCREAMING-KEBAB-CASE
//   - Dot             → dot.case
//   - Title           → Title Case
//
// All converters delegate through a Caser, which holds a configurable
// set of recognised initialisms (URL, ID, HTTP, …). Initialism awareness
// is what lets snake → Pascal round-trip without losing acronym shape:
// "url_path" becomes "URLPath" rather than "UrlPath".
//
// The package exposes both Caser methods (for consumers that need a
// custom initialism set) and package-level shorthands that delegate to
// a default Caser pre-loaded with the common Go-style initialisms.
//
// Identifier is a separate concern: it sanitises an arbitrary string
// into a syntactically valid Go-style identifier without changing case.
package naming
