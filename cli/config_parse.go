// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package cli

import (
	"errors"

	"go.yaml.in/yaml/v4"
)

// parseConfig decodes raw YAML into a *Config seeded with defaults.
// YAML parse errors surface as [*ConfigError] with file:line:col
// when the underlying yaml.TypeError carries position information.
func parseConfig(path string, raw []byte) (*Config, error) {
	c := DefaultConfig()
	if err := yaml.Unmarshal(raw, c); err != nil {
		return nil, yamlError(path, err)
	}
	return c, nil
}

// yamlError translates a yaml.v4 error into a [*ConfigError]
// carrying the file path and the underlying message. yaml.v4
// surfaces structured load errors as [*yaml.LoadErrors]; lexer /
// syntax faults arrive as bare errors with embedded position
// information in the message.
func yamlError(path string, err error) error {
	if le, ok := errors.AsType[*yaml.LoadErrors](err); ok {
		return &ConfigError{Path: path, Reason: le.Error()}
	}
	return &ConfigError{Path: path, Reason: err.Error()}
}
