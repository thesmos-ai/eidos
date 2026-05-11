// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package pipeline assembles plugins into a runnable [Pipeline] via
// the fluent [Builder]. Build-time validation catches the
// configuration errors that would otherwise surface only at run
// time: duplicate plugin names, missing frontends, missing or
// duplicate backends, plugin-options validation failures.
//
// A Build that returns successfully has stashed the participating
// plugins, sink, cache, and diagnostic sink on the [Pipeline]; the
// Run / DryRun execution surface lands as later milestones layer
// plan resolution and phase orchestration on top.
//
// The pipeline writes Build-time diagnostics to its [diag.Sink] in
// addition to returning an aggregated error. Callers that want
// positioned details inspect the sink; callers that just want
// "did Build succeed?" can rely on the returned error alone.
package pipeline
