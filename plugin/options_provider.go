// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package plugin

import "go.thesmos.sh/eidos/core/opt"

// OptionsProvider is the optional capability for plugins that
// declare typed configuration. The pipeline calls
// [OptionsProvider.OptionsSchema] at Build time to discover the
// schema, then calls [OptionsProvider.SetOptions] with the
// configured values; validation errors returned from SetOptions
// surface as Build-time diagnostics.
//
// Plugins typically declare their options as a struct with `eidos:`
// tags and produce the schema via [opt.Reflect]:
//
//	type Options struct {
//	    Output  string `eidos:"output_package,required"`
//	    Naming  string `eidos:"naming,one_of=Pascal|Camel,default=Pascal"`
//	}
//	func (p *Plugin) OptionsSchema() opt.Schema      { return opt.Reflect(Options{}) }
//	func (p *Plugin) SetOptions(o opt.Options) error { return o.Decode(&p.opts) }
type OptionsProvider interface {
	// OptionsSchema returns the schema describing the plugin's
	// configuration shape and validation rules. The pipeline calls
	// [OptionsProvider.SetOptions] with the populated values; the
	// plugin's implementation typically delegates to
	// [opt.Options.Decode] which performs validation as part of
	// decoding into the plugin's options struct.
	OptionsSchema() opt.Schema

	// SetOptions hands the configuration to the plugin and returns
	// any validation error so the pipeline can surface it as a
	// diagnostic at Build time.
	SetOptions(opts opt.Options) error
}
