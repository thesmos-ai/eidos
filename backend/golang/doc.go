// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package golang is eidos' Go-target backend. It implements
// [plugin.Backend] and renders emit graphs to gofmt-clean Go source
// through a template-driven pipeline.
//
// Consumers register a backend instance with [pipeline.Builder.WithBackend].
// Plugins that contribute templates or extend the funcmap implement
// [plugin.TemplateProvider]; the backend collects every provider
// from [plugin.BackendContext.Plugins] (extensions, with cross-plugin
// name-collision rejected) and [plugin.BackendContext.Ordered]
// (overrides, applied in capability topo order so later registrants
// win deterministically).
//
// Rendering groups emit entities by their [emit.Target], composes one
// file per group through the merged template set, runs the result
// through [go/format.Source] for canonical Go formatting, and writes
// the finalised bytes through the supplied [sink.Sink]. The package's
// public surface is the [Backend] type plus the constants identifying
// it to the pipeline ([Name], [Language]).
package golang
