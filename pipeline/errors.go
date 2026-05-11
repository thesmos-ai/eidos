// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package pipeline

import "errors"

// ErrDuplicatePlugin is returned by [Builder.Build] when two plugins
// register under the same [plugin.Plugin.Name]. Names must be unique
// across every role; collisions are programmer errors.
var ErrDuplicatePlugin = errors.New("pipeline: duplicate plugin name")

// ErrNoFrontend is returned by [Builder.Build] when no frontend was
// registered. A pipeline without a frontend has no source to load
// and so cannot run.
var ErrNoFrontend = errors.New("pipeline: no frontend registered")

// ErrNoBackend is returned by [Builder.Build] when no backend was
// registered. A pipeline without a backend cannot render emit
// values to output and so cannot run.
var ErrNoBackend = errors.New("pipeline: no backend registered")

// ErrMultipleBackends is returned by [Builder.Build] when more than
// one backend is registered. Exactly one backend is supported per
// pipeline; multi-language output is achieved by running multiple
// pipelines.
var ErrMultipleBackends = errors.New("pipeline: multiple backends registered")

// ErrInvalidOptions is returned by [Builder.Build] (wrapped with
// the offending plugin name and the underlying validation error)
// when an [plugin.OptionsProvider]'s SetOptions returns a
// validation failure.
var ErrInvalidOptions = errors.New("pipeline: invalid plugin options")
