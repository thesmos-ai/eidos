// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package opt is the typed plugin-options system.
//
// Plugin authors declare their options as a Go struct with `eidos:`
// field tags:
//
//	type Options struct {
//	    OutputPackage string `eidos:"output_package,required"`
//	    InterfaceName string `eidos:"interface_name,default=Repository"`
//	    Naming        string `eidos:"naming,one_of=Pascal|Camel,default=Pascal"`
//	}
//
// [Reflect] derives a [Schema] from the example struct's tags; the
// pipeline collects every plugin's schema at Build() time. Raw
// per-plugin values arrive as `map[string]string` from whichever
// config loader is in use (YAML, TOML, JSON, …) — the [Options]
// constructor binds those values to the schema, and [Options.Decode]
// validates and populates the plugin's typed struct via reflection.
//
// Validation surfaces as Go errors wrapping the relevant sentinel —
// the pipeline converts them to positioned diagnostics at the
// offending key in the config file. Errors come in two families:
//
//   - Tag errors ([ErrInvalidTag], [ErrUnsupportedFieldType]) signal
//     programmer mistakes in the plugin's options struct. [Reflect]
//     panics on these; [ReflectChecked] returns them.
//   - Decode errors ([ErrMissingRequired], [ErrInvalidValue],
//     [ErrUnknownField], [ErrInvalidDecodeTarget]) signal user-side
//     config problems and surface during pipeline build.
//
// All values flow through opt as strings. Config-format loaders are
// responsible for marshalling their native types into strings before
// constructing an [Options]; the opt package handles parsing back to
// typed Go values at decode time.
package opt
