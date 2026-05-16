// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package manifest

import (
	"encoding/json"
	"fmt"
	"strconv"

	"go.thesmos.sh/eidos/emit"
)

// Version is the current manifest schema version. Bump whenever the
// on-disk format changes incompatibly so older readers detect the
// mismatch and refuse to interpret the file rather than silently
// mis-parsing it.
const Version = 1

// PluginAttribution names one plugin's contribution to a
// rendered [Output]. Name is the plugin's stable identifier
// (matches [plugin.Plugin.Name]); OutputTag identifies the
// multi-output emission the contribution lives in (empty for the
// plugin's primary output, non-empty for a tagged secondary
// output).
//
// Serialisation is polymorphic so primary-output manifests stay
// byte-identical to pre-multi-output manifests: an attribution
// whose OutputTag is empty marshals as a bare JSON string; an
// attribution carrying a non-empty OutputTag marshals as a JSON
// object with `name` and `output_tag` fields. Unmarshal accepts
// either form on the wire.
type PluginAttribution struct {
	// Name is the plugin's stable identifier.
	Name string `json:"name"`

	// OutputTag identifies the multi-output emission this
	// contribution lives in — matches the [plugin.Output.Tag]
	// declared by the plugin for its routable outputs. Empty
	// means the plugin's primary output; non-empty values pair
	// structurally with Name so consumers can render
	// `<plugin>:<tag>` for human-readable surfaces.
	OutputTag string `json:"output_tag,omitempty"`
}

// String renders the attribution in the canonical
// `<plugin>:<tag>` cross-tool form — the bare plugin name when
// OutputTag is empty, otherwise plugin name and tag joined by a
// colon. Diagnostics, log lines, and explain-style human-readable
// surfaces use this representation so tag references stay
// scope-aware across a multi-plugin pipeline.
func (a PluginAttribution) String() string {
	if a.OutputTag == "" {
		return a.Name
	}
	return a.Name + ":" + a.OutputTag
}

// MarshalJSON emits a bare JSON string when no OutputTag is set,
// or a JSON object otherwise. The bare-string form keeps
// primary-output manifest entries byte-identical to the format
// pre-dating multi-output support. Both branches encode the
// fields via [strconv.AppendQuote] — total over every Go string,
// so the method never returns a non-nil error and the signature's
// `error` return exists only to satisfy the [json.Marshaler]
// contract.
func (a PluginAttribution) MarshalJSON() ([]byte, error) {
	if a.OutputTag == "" {
		return strconv.AppendQuote(nil, a.Name), nil
	}
	buf := make([]byte, 0, len(a.Name)+len(a.OutputTag)+pluginAttributionObjectOverhead)
	buf = append(buf, `{"name":`...)
	buf = strconv.AppendQuote(buf, a.Name)
	buf = append(buf, `,"output_tag":`...)
	buf = strconv.AppendQuote(buf, a.OutputTag)
	buf = append(buf, '}')
	return buf, nil
}

// pluginAttributionObjectOverhead is the byte count of the
// non-string scaffolding [PluginAttribution.MarshalJSON] emits
// when rendering the object form: `{"name":""`+`,"output_tag":""}`.
// Used to pre-size the buffer; the value avoids a recurring
// magic-number explanation at the call site.
const pluginAttributionObjectOverhead = len(`{"name":""`) + len(`,"output_tag":""}`)

// UnmarshalJSON accepts both the bare-string form (legacy /
// untagged attribution) and the object form (tagged attribution).
// Producers may interleave both shapes within one manifest's
// plugins array — the framework emits whichever form keeps the
// entry minimal, and consumers handle both transparently.
func (a *PluginAttribution) UnmarshalJSON(data []byte) error {
	if len(data) > 0 && data[0] == '"' {
		if err := json.Unmarshal(data, &a.Name); err != nil {
			return fmt.Errorf("manifest: unmarshal plugin name: %w", err)
		}
		return nil
	}
	type attributionObject PluginAttribution
	if err := json.Unmarshal(data, (*attributionObject)(a)); err != nil {
		return fmt.Errorf("manifest: unmarshal plugin attribution: %w", err)
	}
	return nil
}

// Output is one entry in a [Manifest] — the [emit.Target] the
// pipeline routed the file to, the plugins that contributed to it,
// the SHA-256 hex digest of the final file content, and the
// observability block describing how routing resolved.
type Output struct {
	// Target identifies the destination the pipeline routed this
	// file to.
	Target emit.Target `json:"target"`

	// Plugins is the list of plugin attributions that contributed
	// to the rendered file. Each entry pairs a plugin name with
	// an optional [plugin.Output.Tag] identifying the
	// multi-output emission the contribution lives in. Multiple
	// plugins appear when free-floating declarations from
	// different generators share a Target or when slot-based
	// composition mixes contributions.
	//
	// Serialisation is polymorphic for byte-stable parity with
	// pre-multi-output manifests: an untagged attribution renders
	// as a bare JSON string ("enum"); a tagged attribution renders
	// as a JSON object ({"name":"enum","output_tag":"test"}). The
	// custom [PluginAttribution.MarshalJSON] /
	// [PluginAttribution.UnmarshalJSON] pair handles both forms on
	// both sides of the wire.
	Plugins []PluginAttribution `json:"plugins"`

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
