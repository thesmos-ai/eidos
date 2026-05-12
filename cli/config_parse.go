// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package cli

import (
	"errors"

	"go.yaml.in/yaml/v4"
)

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
