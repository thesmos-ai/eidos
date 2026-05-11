// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package plugin defines the contracts every eidos plugin satisfies.
// A plugin is anything that implements [Plugin] (a single Name()
// accessor) plus one or more role interfaces:
//
//   - [Frontend]: parses input into the source-side store
//   - [Annotator]: stamps metadata on existing nodes
//   - [Generator]: produces emit entities into the output-side store
//   - [Backend]: renders emit to a target language and writes to a
//     [sink.Sink]
//
// Plugins may also opt into capability interfaces:
//
//   - [CapabilityProvider]: priority + Provides/Requires for
//     pipeline ordering
//   - [TemplateProvider]: ships templates / func-map extensions for
//     a backend
//   - [OptionsProvider]: declares typed configuration the pipeline
//     populates at Build()
//   - [DirectiveProvider]: auto-registers directive schemas at
//     Build()
//
// One plugin may implement any combination of role and capability
// interfaces; the pipeline detects each via interface assertion. The
// roles and capabilities are deliberately small so plugin authors
// can compose them without ceremony.
package plugin
