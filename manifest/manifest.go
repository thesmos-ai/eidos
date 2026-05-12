// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package manifest

import "go.thesmos.sh/eidos/emit"

// Version is the current manifest schema version. Bump whenever the
// on-disk format changes incompatibly so older readers detect the
// mismatch and refuse to interpret the file rather than silently
// mis-parsing it.
const Version = 1

// Output is one entry in a [Manifest] — the [emit.Target] the
// pipeline routed the file to, the plugins that contributed to it,
// the SHA-256 hex digest of the final file content, and the
// observability block describing how routing resolved.
type Output struct {
	// Target identifies the destination the pipeline routed this
	// file to.
	Target emit.Target `json:"target"`

	// Plugins is the list of plugin names that contributed to the
	// rendered file. Multiple plugins appear when free-floating
	// declarations from different generators share a Target or
	// when slot-based composition mixes contributions.
	Plugins []string `json:"plugins"`

	// Hash is the SHA-256 hex digest of the rendered file content
	// prefixed with "sha256:". A change in any plugin's
	// contribution flips the hash and so flips this Output's
	// fingerprint for the next run's prune diff.
	Hash string `json:"hash"`

	// ResolvedLayout records the routing decision that produced
	// this output — the resolved layout, package, directory, and
	// filename plus a per-field breakdown of which precedence
	// layer supplied each value. The block is observability only
	// and is not consumed by drift detection; it answers "why did
	// this file land here?" for the `explain` command and
	// post-mortem diagnosis. Optional — older manifests written
	// before the routing layer landed omit the field.
	ResolvedLayout *ResolvedLayout `json:"resolved_layout,omitempty"`
}

// Layer names the precedence layer that supplied a composed
// field's final value in a [ResolvedLayout.ResolvedFrom] entry.
// The set is stable: consumers compare against the constants
// rather than the underlying strings so a typo at the producer
// side surfaces at compile time.
type Layer string

// Precedence layers, ordered from lowest to highest priority.
// The Layout phase composes Target fields by walking the layers
// and stamping the field's [Layer] each time a layer takes
// effect; the final stamp is the value recorded in the manifest.
const (
	// LayerFramework is the framework default — alongside-source
	// layout, Package + ImportPath derived from the origin's
	// source package, Filename derived from the source basename
	// when no plugin suffix is declared.
	LayerFramework Layer = "framework"

	// LayerPluginSuffix is a generator's declared filename suffix
	// supplied through the FilenameProvider capability.
	LayerPluginSuffix Layer = "plugin-suffix"

	// LayerProject is the project-level `output.*` block in
	// `.eidos.yaml`. Reserved for the config consumer; the
	// Layout phase does not stamp this layer until the consumer
	// is wired.
	LayerProject Layer = "project"

	// LayerPerPlugin is a per-plugin `plugins[*].output.*`
	// override block. Reserved for the config consumer like
	// [LayerProject].
	LayerPerPlugin Layer = "per-plugin"

	// LayerDirective is a `+gen:out` directive on the
	// originating source node. Overrides Filename only.
	LayerDirective Layer = "directive"

	// LayerCLI is a command-line override (`-o` / `-p` /
	// `-layout` / `-output-dir` / `-target`). Pinned at the
	// highest priority among the field-composing layers.
	LayerCLI Layer = "cli"
)

// ResolvedLayout summarises the routing decision for one
// rendered output. Layout, Package, Dir, and Filename mirror
// the values stamped into [Output.Target]; ResolvedFrom names
// the highest-priority precedence layer that supplied each
// field.
type ResolvedLayout struct {
	// Layout is the resolved layout selector — either
	// `alongside-source` or `centralised`.
	Layout string `json:"layout"`

	// Package is the resolved Go package name (matches
	// [emit.Target.Package]).
	Package string `json:"package,omitempty"`

	// Dir is the resolved output directory (matches
	// [emit.Target.Dir]).
	Dir string `json:"dir,omitempty"`

	// Filename is the resolved rendered filename (matches
	// [emit.Target.Filename]).
	Filename string `json:"filename"`

	// ResolvedFrom maps each composed field (`layout`, `package`,
	// `dir`, `filename`) to the precedence [Layer] that supplied
	// its final value.
	ResolvedFrom map[string]Layer `json:"resolved_from"`
}

// Manifest is the per-run record of every [Output] the pipeline
// produced. The Version field disambiguates schema generations;
// Brand identifies the tool that wrote the manifest (the consumer
// binary's hardcoded identity); RunID identifies the run for
// diagnostics; Outputs is the full set of files the pipeline
// claimed in the run.
type Manifest struct {
	// Version is the manifest schema version.
	Version int `json:"version"`

	// Brand identifies the tool that wrote the manifest. Matches
	// the brand stamped into the rendered files' header marker
	// (`// Code generated by <Brand>. DO NOT EDIT.`) and footer
	// (`// <Brand>:provenance <hash>`). `eidos prune` reads this
	// field to compose the marker-line guard before deleting any
	// claimed file.
	Brand string `json:"brand,omitempty"`

	// RunID identifies the run that produced the manifest. The
	// pipeline typically uses an RFC 3339 UTC timestamp; the field
	// is opaque to the manifest package — clients write whatever
	// they need to attribute outputs back to a run.
	RunID string `json:"run_id"`

	// Outputs is the list of every file the run produced.
	Outputs []Output `json:"outputs"`
}

// New returns a Manifest stamped with the current [Version] and the
// supplied RunID. Outputs starts empty; clients append entries as
// the run produces files. Brand stays empty — callers stamp it
// after construction via direct field assignment.
func New(runID string) *Manifest {
	return &Manifest{Version: Version, RunID: runID, Outputs: []Output{}}
}

// Add appends an output entry to the manifest.
func (m *Manifest) Add(out Output) {
	m.Outputs = append(m.Outputs, out)
}

// Targets returns the set of [emit.Target] values claimed by the
// manifest's outputs. Used by [Prune] and by callers that want to
// verify "did the previous run produce a file at this Target?".
func (m *Manifest) Targets() []emit.Target {
	out := make([]emit.Target, 0, len(m.Outputs))
	for _, o := range m.Outputs {
		out = append(out, o.Target)
	}
	return out
}

// HasTarget reports whether any output in the manifest claims the
// supplied target.
func (m *Manifest) HasTarget(t emit.Target) bool {
	for _, o := range m.Outputs {
		if o.Target == t {
			return true
		}
	}
	return false
}
