// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package golang

import (
	"bytes"
	"fmt"
	"text/template"

	"go.thesmos.sh/eidos/core/position"
	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/plugin"
)

// Name is the stable plugin identifier the pipeline uses for
// registration, diagnostic attribution, and cache-key derivation.
// Plugin authors reference it when supplying backend-specific
// options via [pipeline.Builder.WithPluginOptions].
const Name = "backend.golang"

// Language is the target-language identifier shared between the
// backend, plugin-supplied template providers, and downstream
// tooling. Plugin authors target this exact string when
// implementing [plugin.TemplateProvider.Templates].
const Language = "golang"

// Backend renders emit graphs to Go source. The zero value is
// unusable; construct via [New]. A Backend instance is safe for
// concurrent use — [Backend.Render] holds no state across calls
// and dispatches per-target work through the supplied
// [plugin.BackendContext].
type Backend struct {
	tmpl *template.Template
}

// New returns a Backend ready for registration on a pipeline via
// [pipeline.Builder.WithBackend].
func New() *Backend {
	return &Backend{tmpl: loadTemplates()}
}

// Name reports the stable plugin identifier — returns [Name].
func (*Backend) Name() string { return Name }

// Language reports the target-language identifier — returns
// [Language].
func (*Backend) Language() string { return Language }

// Render groups emit entities by their [emit.Target] and writes one
// gofmt-clean file per non-empty Target through ctx.Sink. The
// per-Target pipeline executes the merged template set against the
// entities sharing that Target, runs the result through
// [go/format.Source], and writes the finalised bytes.
//
// Zero-valued Targets are filtered upstream by [store.EmitView]'s
// by-target index and never reach the loop. Empty Targets whose
// entity list resolves to nothing renderable are skipped before
// the template executes so no zero-content file ever reaches the
// sink.
//
// Template execution failures surface as Error diagnostics on
// ctx.Diag for the offending Target and the loop continues with
// the next Target. Format failures surface as Warn diagnostics on
// the same Target and the unformatted body is still written. Sink
// write failures propagate as a wrapped error from Render — they
// indicate an I/O or backend-side fault rather than a content
// defect.
func (b *Backend) Render(ctx *plugin.BackendContext) error {
	ps := ctx.Diag.For(Name)
	for _, target := range ctx.Store.Emit().ByTarget().Keys() {
		entities := ctx.Store.Emit().ByTarget().Get(target)
		body, err := renderFile(b.tmpl, target, entities)
		if err != nil {
			ps.Errorf(position.Pos{}, "%s: %v", target.JoinPath(), err)
			continue
		}
		body = formatBody(body, target, ps)
		if err := ctx.Sink.Write(target, body); err != nil {
			return fmt.Errorf("%s: sink write %s: %w", Name, target.JoinPath(), err)
		}
	}
	return nil
}

// fileData is the value the `emit.file` template executes against.
// PackageName supplies the `package <name>` clause; Decls carries
// the entities to render at file scope in deterministic order.
type fileData struct {
	PackageName string
	Decls       []emit.Node
}

// renderFile executes the `emit.file` template against the entities
// routed to target. The returned bytes are the raw rendered body
// awaiting the [formatBody] pass.
func renderFile(tmpl *template.Template, target emit.Target, entities []emit.Node) ([]byte, error) {
	data := fileData{PackageName: target.Package, Decls: entities}
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "emit.file", data); err != nil {
		return nil, fmt.Errorf("backend/golang: execute emit.file: %w", err)
	}
	return buf.Bytes(), nil
}
