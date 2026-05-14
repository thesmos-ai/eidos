// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package cli

// yamlError translates a goccy/go-yaml decode error into a
// [*ConfigError] carrying the file path and the underlying
// message. goccy/go-yaml embeds line/column information in the
// error string, so wrapping it verbatim preserves position
// context for the diagnostic sink.
func yamlError(path string, err error) error {
	return &ConfigError{Path: path, Reason: err.Error()}
}
