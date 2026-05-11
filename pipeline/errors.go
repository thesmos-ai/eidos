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

// ErrCycle is returned by [Builder.Build] when a priority bucket
// contains a cycle in its [plugin.CapabilityProvider.Requires]
// graph. The wrapped message lists the plugins involved in the
// cycle in alphabetical order.
var ErrCycle = errors.New("pipeline: cycle in plugin requires")

// ErrDuplicateProvider is returned by [Builder.Build] when two
// plugins in the same priority bucket both declare the same
// capability name in [plugin.CapabilityProvider.Provides]. Each
// capability must have a single provider per bucket so topo
// resolution stays deterministic.
var ErrDuplicateProvider = errors.New("pipeline: duplicate capability provider")

// ErrTemplateFuncCollision is returned by [Builder.Build] when two
// plugins both register a [text/template] func of the same name in
// [plugin.TemplateProvider.TemplateFuncs] for the backend's
// language. Intentional overrides go through
// [plugin.TemplateProvider.TemplateOverrides] and bypass this
// check.
var ErrTemplateFuncCollision = errors.New("pipeline: template func collision")

// ErrNoSink is returned by [Pipeline.Run] when no [sink.Sink] was
// configured at Build time. The backend has nowhere to write so the
// run cannot complete.
var ErrNoSink = errors.New("pipeline: no sink configured")

// ErrRunHadErrors is returned by [Pipeline.Run] when one or more
// plugins emitted a [diag.Error] diagnostic. The pipeline runs to
// completion regardless of intermediate errors; ErrRunHadErrors is
// the post-hoc signal that the user's code or a plugin reported
// problems. Inspect the [diag.Sink] (via [Pipeline.Diag]) for
// per-error details.
var ErrRunHadErrors = errors.New("pipeline: run completed with error diagnostics")

// ErrDuplicateDirective is returned by [Builder.Build] when two
// directive schemas register under the same name. Schemas are
// shared contracts (multiple plugins may consume one), so the
// pipeline rejects duplicate registrations rather than guessing
// which definition wins.
var ErrDuplicateDirective = errors.New("pipeline: duplicate directive schema")

// ErrIncompatibleEmitVersion is returned by [Builder.Build] when a
// plugin implementing [plugin.EmitVersioned] declares a list of
// supported emit majors that does not include the in-tree
// [emit.Major]. The wrapping message names the plugin and lists
// both the declared and the in-tree majors so the upgrade path is
// obvious.
var ErrIncompatibleEmitVersion = errors.New("pipeline: incompatible emit version")
