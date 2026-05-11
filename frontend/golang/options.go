// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package golang

import (
	"go.thesmos.sh/eidos/core/opt"
)

// Options carries the Go frontend's user-tunable settings. The
// fields are populated via the [plugin.OptionsProvider] surface at
// pipeline Build time — either programmatically through
// [pipeline.Builder.WithPluginOptions] or from a config-file entry
// scoped to the frontend's [plugin.Plugin.Name].
//
// All fields have sensible defaults; the typical pipeline registers
// the frontend with no options at all.
type Options struct {
	// IncludeTests controls whether `*_test.go` files contribute
	// declarations to the loaded packages. Defaults to false so
	// generators only see production-facing types.
	IncludeTests bool `json:"include_tests" eidos:"include_tests,default=false"`

	// BuildTags is a space-separated list of build tags to apply
	// when loading packages. Empty by default — Go's standard build
	// tag rules apply.
	BuildTags string `json:"build_tags" eidos:"build_tags,default="`

	// SkipCgoFiles drops cgo-synthesized declarations from the
	// loaded packages. Cgo wrappers rarely participate in
	// generator-driven workflows; defaulting to true keeps the
	// store free of noise. Set to false when a generator legitimately
	// needs visibility into cgo types.
	SkipCgoFiles bool `json:"skip_cgo_files" eidos:"skip_cgo_files,default=true"`
}

// optionsSchema is the reflected option schema cached at package
// init time. Each [Frontend] instance reuses it across pipeline
// invocations.
var optionsSchema = opt.Reflect(
	Options{},
) //nolint:gochecknoglobals // schema is stateless and reflection result
