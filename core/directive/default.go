// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package directive

import "sync"

// DefaultPrefix is the project-wide directive prefix every eidos
// frontend, generator, and tool uses. Comments of the form
// `+gen:NAME …` and `-gen:NAME …` carry directives; the prefix is
// fixed by convention rather than configurable so cross-plugin code
// can rely on a single canonical syntax.
const DefaultPrefix = "gen"

// defaultParser is the shared [Parser] instance for [DefaultPrefix],
// constructed lazily on first call to [DefaultParser]. A parser is
// safe for concurrent use, so a single instance suffices for the
// entire process.
var (
	defaultParser     *Parser   //nolint:gochecknoglobals // shared singleton
	defaultParserOnce sync.Once //nolint:gochecknoglobals // shared singleton
)

// DefaultParser returns the singleton [Parser] configured for
// [DefaultPrefix]. Pipelines and frontends call this instead of
// constructing their own parsers so every directive across the
// process parses identically.
//
// The returned parser is safe for concurrent use.
func DefaultParser() *Parser {
	defaultParserOnce.Do(func() {
		defaultParser = newParser(DefaultPrefix)
	})
	return defaultParser
}
