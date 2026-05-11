// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package plugin

import "go.thesmos.sh/eidos/core/opt"

// OptionsProvider is the optional capability for plugins that
// declare typed configuration. The pipeline calls
// [OptionsProvider.OptionsSchema] at Build time to validate the
// configuration source (config file or programmatic
// Builder.WithPluginOptions call) and then calls
// [OptionsProvider.SetOptions] with the populated values.
//
// Plugins typically declare their options as a struct with `eidos:`
// tags and produce the schema via [opt.Reflect]:
//
//	type Options struct {
//	    Output  string `eidos:"output_package,required"`
//	    Naming  string `eidos:"naming,one_of=Pascal|Camel,default=Pascal"`
//	}
//	func (p *Plugin) OptionsSchema() opt.Schema { return opt.Reflect(Options{}) }
//	func (p *Plugin) SetOptions(o opt.Options)  { _ = o.Decode(&p.opts) }
type OptionsProvider interface {
	// OptionsSchema returns the schema describing the plugin's
	// configuration shape and validation rules. The pipeline uses
	// the schema to validate the configuration source before
	// calling [OptionsProvider.SetOptions].
	OptionsSchema() opt.Schema

	// SetOptions hands the validated configuration to the plugin.
	// The plugin typically decodes opts into a struct field for
	// later access from its role methods.
	SetOptions(opts opt.Options)
}
