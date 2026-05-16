// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package pipeline_test

import (
	"errors"
	"io/fs"
	"strings"
	"sync"
	"testing"
	"text/template"

	"go.thesmos.sh/eidos/core/diag"
	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/core/opt"
	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/plugin"
	"go.thesmos.sh/eidos/priority"
	"go.thesmos.sh/eidos/store"
)

// assertNoError fails the test if err is non-nil.
func assertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// names extracts the plugin names from a plugin slice for compact
// equality assertions in resolver and plan tests.
func names[T interface{ Name() string }](plugins []T) []string {
	out := make([]string, len(plugins))
	for i, p := range plugins {
		out[i] = p.Name()
	}
	return out
}

// phaseLogs returns the message of every Info diagnostic the
// pipeline emitted under the "pipeline" attribution that starts
// with "phase=". Used by verbose-mode tests to verify the phase
// boundaries were announced in the expected order.
func phaseLogs(d *diag.Sink) []string {
	var out []string
	for _, e := range d.Diagnostics() {
		if e.Severity != diag.Info || e.Plugin != "pipeline" {
			continue
		}
		if !strings.HasPrefix(e.Message, "phase=") {
			continue
		}
		out = append(out, e.Message)
	}
	return out
}

// stubFE implements [plugin.Frontend].
type stubFE struct{ name string }

func (f *stubFE) Name() string                       { return f.name }
func (*stubFE) Load(_ *plugin.FrontendContext) error { return nil }

// stubAnn implements [plugin.Annotator].
type stubAnn struct{ name string }

func (a *stubAnn) Name() string                            { return a.name }
func (*stubAnn) Annotate(_ *plugin.AnnotatorContext) error { return nil }

// stubGen implements [plugin.Generator].
type stubGen struct{ name string }

func (g *stubGen) Name() string                            { return g.name }
func (*stubGen) Generate(_ *plugin.GeneratorContext) error { return nil }

// stubBE implements [plugin.Backend].
type stubBE struct{ name string }

func (b *stubBE) Name() string                        { return b.name }
func (*stubBE) Language() string                      { return "stub" }
func (*stubBE) Render(_ *plugin.BackendContext) error { return nil }

// stubFEWithOpts is a frontend that also implements
// [plugin.OptionsProvider]. The configured options struct exposes a
// single required string field; tests use it to drive the
// validation paths in Build.
type stubFEWithOpts struct {
	name string
	opts stubFEOptions
}

type stubFEOptions struct {
	Output string `eidos:"output,required"`
}

func (f *stubFEWithOpts) Name() string                       { return f.name }
func (*stubFEWithOpts) Load(_ *plugin.FrontendContext) error { return nil }
func (*stubFEWithOpts) OptionsSchema() opt.Schema            { return opt.Reflect(stubFEOptions{}) }

func (f *stubFEWithOpts) SetOptions(o opt.Options) error { return o.Decode(&f.opts) }

// stubAnnCap is an annotator that also implements
// [plugin.CapabilityProvider]. Used to drive bucket grouping and
// topo resolution in Build tests.
type stubAnnCap struct {
	name     string
	priority priority.Priority
	provides []string
	requires []string
}

func (a *stubAnnCap) Name() string                            { return a.name }
func (*stubAnnCap) Annotate(_ *plugin.AnnotatorContext) error { return nil }
func (a *stubAnnCap) Priority() priority.Priority             { return a.priority }
func (a *stubAnnCap) Provides() []string                      { return a.provides }
func (a *stubAnnCap) Requires() []string                      { return a.requires }

// stubGenCap is a generator that also implements
// [plugin.CapabilityProvider]. Mirrors stubAnnCap for the
// generator phase.
type stubGenCap struct {
	name     string
	priority priority.Priority
	provides []string
	requires []string
}

func (g *stubGenCap) Name() string                            { return g.name }
func (*stubGenCap) Generate(_ *plugin.GeneratorContext) error { return nil }
func (g *stubGenCap) Priority() priority.Priority             { return g.priority }
func (g *stubGenCap) Provides() []string                      { return g.provides }
func (g *stubGenCap) Requires() []string                      { return g.requires }

// stubBEWithTemplates is a backend that also implements
// [plugin.TemplateProvider]. Used by template-collision tests to
// drive the validateTemplateFuncs path indirectly through Build.
type stubBEWithTemplates struct {
	name      string
	lang      string
	funcs     template.FuncMap
	overrides template.FuncMap
}

func (b *stubBEWithTemplates) Name() string                                { return b.name }
func (b *stubBEWithTemplates) Language() string                            { return b.lang }
func (*stubBEWithTemplates) Render(_ *plugin.BackendContext) error         { return nil }
func (*stubBEWithTemplates) Templates(_ string) (fs.FS, bool)              { return nil, false }
func (b *stubBEWithTemplates) TemplateFuncs(_ string) template.FuncMap     { return b.funcs }
func (b *stubBEWithTemplates) TemplateOverrides(_ string) template.FuncMap { return b.overrides }

// stubGenWithTemplates is a generator that ships templates for the
// pipeline's backend language. Used to drive cross-plugin template
// func collision tests.
type stubGenWithTemplates struct {
	name      string
	funcs     template.FuncMap
	overrides template.FuncMap
}

func (g *stubGenWithTemplates) Name() string                                { return g.name }
func (*stubGenWithTemplates) Generate(_ *plugin.GeneratorContext) error     { return nil }
func (*stubGenWithTemplates) Templates(_ string) (fs.FS, bool)              { return nil, false }
func (g *stubGenWithTemplates) TemplateFuncs(_ string) template.FuncMap     { return g.funcs }
func (g *stubGenWithTemplates) TemplateOverrides(_ string) template.FuncMap { return g.overrides }

// recFE is a recording frontend used by Run tests. It records every
// pattern it received via Load and can be configured to return an
// error from Load so diagnostic capture can be exercised. The
// optional loadFn hook lets a test populate the supplied store on
// each Load call.
type recFE struct {
	mu     sync.Mutex
	name   string
	loaded []string
	err    error
	loadFn func(s *store.Store)
}

func (f *recFE) Name() string { return f.name }
func (f *recFE) Load(ctx *plugin.FrontendContext) error {
	f.mu.Lock()
	f.loaded = append(f.loaded, ctx.Pattern)
	f.mu.Unlock()
	if f.loadFn != nil {
		f.loadFn(ctx.Store)
	}
	return f.err
}

// recAnn is a recording annotator. It records its call count and
// can return an error from Annotate; the optional annotate hook
// runs for each call so a test can stamp metadata.
type recAnn struct {
	mu       sync.Mutex
	name     string
	calls    int
	err      error
	annotate func(ctx *plugin.AnnotatorContext)
}

func (a *recAnn) Name() string { return a.name }
func (a *recAnn) Annotate(ctx *plugin.AnnotatorContext) error {
	a.mu.Lock()
	a.calls++
	a.mu.Unlock()
	if a.annotate != nil {
		a.annotate(ctx)
	}
	return a.err
}

// recGen is a recording generator. Mirrors recAnn for the generator
// phase and exposes a generate hook for tests that want to populate
// emit before the backend runs. The generator advertises a default
// filename suffix so the Layout phase can route every emit decl it
// produces; tests that need a specific suffix set the [recGen.suffix]
// field directly.
type recGen struct {
	mu       sync.Mutex
	name     string
	calls    int
	err      error
	suffix   string
	generate func(ctx *plugin.GeneratorContext)
}

func (g *recGen) Name() string { return g.name }
func (g *recGen) Outputs(_ string) []plugin.Output {
	suffix := g.suffix
	if suffix == "" {
		suffix = "_gen.go"
	}
	return []plugin.Output{{Suffix: suffix}}
}

func (g *recGen) Generate(ctx *plugin.GeneratorContext) error {
	g.mu.Lock()
	g.calls++
	g.mu.Unlock()
	if g.generate != nil {
		g.generate(ctx)
	}
	return g.err
}

// recBE is a recording backend. It records its call count and can
// return an error from Render; the render hook lets tests exercise
// the sink writes a real backend would perform.
type recBE struct {
	mu     sync.Mutex
	name   string
	lang   string
	calls  int
	err    error
	render func(ctx *plugin.BackendContext)
}

func (b *recBE) Name() string     { return b.name }
func (b *recBE) Language() string { return b.lang }
func (b *recBE) Render(ctx *plugin.BackendContext) error {
	b.mu.Lock()
	b.calls++
	b.mu.Unlock()
	if b.render != nil {
		b.render(ctx)
	}
	return b.err
}

// panickyFE is a frontend that panics from Load. Used to verify the
// pipeline's panic-recovery wrapper.
type panickyFE struct {
	name string
	msg  string
}

func (f *panickyFE) Name() string                         { return f.name }
func (f *panickyFE) Load(_ *plugin.FrontendContext) error { panic(f.msg) }

// panickyAnn is an annotator that panics. Mirrors panickyFE for the
// annotator phase.
type panickyAnn struct {
	name string
	msg  string
}

func (a *panickyAnn) Name() string                              { return a.name }
func (a *panickyAnn) Annotate(_ *plugin.AnnotatorContext) error { panic(a.msg) }

// panickyGen is a generator that panics. Mirrors panickyFE for the
// generator phase.
type panickyGen struct {
	name string
	msg  string
}

func (g *panickyGen) Name() string                              { return g.name }
func (g *panickyGen) Generate(_ *plugin.GeneratorContext) error { panic(g.msg) }

// panickyBE is a backend that panics. Mirrors panickyFE for the
// backend phase.
type panickyBE struct {
	name string
	lang string
	msg  string
}

func (b *panickyBE) Name() string                          { return b.name }
func (b *panickyBE) Language() string                      { return b.lang }
func (b *panickyBE) Render(_ *plugin.BackendContext) error { panic(b.msg) }

// frozenAddAnn is an annotator that attempts to call AddPackage
// after the node view is frozen. Used to verify the pipeline
// converts store.ErrFrozen into an Internal diagnostic.
type frozenAddAnn struct {
	name string
}

func (a *frozenAddAnn) Name() string { return a.name }
func (*frozenAddAnn) Annotate(ctx *plugin.AnnotatorContext) error {
	return ctx.Store.Nodes().AddPackage(&node.Package{Name: "x", Path: "x"})
}

// frozenAddGen is a generator that attempts to call AddPackage on
// the emit view after it has been frozen — which the pipeline does
// only after the generator phase, so callers from later phases
// would hit this. We use it here to drive the post-generator path
// through the backend.
type frozenAddBE struct {
	name string
	lang string
}

func (b *frozenAddBE) Name() string     { return b.name }
func (b *frozenAddBE) Language() string { return b.lang }
func (*frozenAddBE) Render(ctx *plugin.BackendContext) error {
	return ctx.Store.Emit().AddPackage(&emit.Package{Name: "x", Path: "x", Dir: "x"})
}

// emitVersionedFE is a frontend that declares an EmitVersions list.
// Used to drive the Build-time emit-version compatibility check.
type emitVersionedFE struct {
	name     string
	versions []string
}

func (f *emitVersionedFE) Name() string                       { return f.name }
func (*emitVersionedFE) Load(_ *plugin.FrontendContext) error { return nil }
func (f *emitVersionedFE) EmitVersions() []string             { return f.versions }

// outputsGen is a generator that returns a caller-supplied Outputs
// slice for any language. Used to drive the Build-time
// Outputs-shape validation check against deliberately malformed
// configurations.
type outputsGen struct {
	name    string
	outputs []plugin.Output
}

func (g *outputsGen) Name() string                            { return g.name }
func (g *outputsGen) Outputs(_ string) []plugin.Output        { return g.outputs }
func (*outputsGen) Generate(_ *plugin.GeneratorContext) error { return nil }

// hasPanicMessage returns true if any diagnostic in d carries msg
// in its detail (where the panic message and stack are stored) or
// in its top-level message. Used by panic-recovery tests to verify
// diag.RecoverAs captured the panic.
func hasPanicMessage(d *diag.Sink, msg string) bool {
	for _, e := range d.Diagnostics() {
		if strings.Contains(e.Detail, msg) || strings.Contains(e.Message, msg) {
			return true
		}
	}
	return false
}

// internalDiagsFor returns every Internal-severity diagnostic in d.
// Used by mutability tests to verify ErrFrozen violations surface
// at Internal severity.
func internalDiagsFor(d *diag.Sink) []diag.Diag {
	var out []diag.Diag
	for _, e := range d.Diagnostics() {
		if e.Severity == diag.Internal {
			out = append(out, e)
		}
	}
	return out
}

// stubAnnCapRec is an annotator that implements CapabilityProvider
// AND records its calls. Used by parallel-execution tests to verify
// concurrent invocations.
type stubAnnCapRec struct {
	mu       sync.Mutex
	name     string
	priority priority.Priority
	provides []string
	requires []string
	hook     func(ctx *plugin.AnnotatorContext)
	calls    int
}

func (a *stubAnnCapRec) Name() string                { return a.name }
func (a *stubAnnCapRec) Priority() priority.Priority { return a.priority }
func (a *stubAnnCapRec) Provides() []string          { return a.provides }
func (a *stubAnnCapRec) Requires() []string          { return a.requires }
func (a *stubAnnCapRec) Annotate(ctx *plugin.AnnotatorContext) error {
	a.mu.Lock()
	a.calls++
	a.mu.Unlock()
	if a.hook != nil {
		a.hook(ctx)
	}
	return nil
}

// stubGenNodesOnly is a generator that implements both
// CapabilityProvider and NodesOnly. Used by generator-parallelism
// tests where every generator must opt in for the bucket to
// parallelise.
type stubGenNodesOnly struct {
	mu        sync.Mutex
	name      string
	priority  priority.Priority
	provides  []string
	requires  []string
	nodesOnly bool
	hook      func(ctx *plugin.GeneratorContext)
	calls     int
}

func (g *stubGenNodesOnly) Name() string                { return g.name }
func (g *stubGenNodesOnly) Priority() priority.Priority { return g.priority }
func (g *stubGenNodesOnly) Provides() []string          { return g.provides }
func (g *stubGenNodesOnly) Requires() []string          { return g.requires }
func (g *stubGenNodesOnly) NodesOnly() bool             { return g.nodesOnly }
func (g *stubGenNodesOnly) Generate(ctx *plugin.GeneratorContext) error {
	g.mu.Lock()
	g.calls++
	g.mu.Unlock()
	if g.hook != nil {
		g.hook(ctx)
	}
	return nil
}

// emptyReadSetHash returns the SHA-256 hex digest a fresh
// [store.ReadSet] produces. Used by cache-key tests where the
// plugin under test makes no Reader queries, so its ReadSet is
// expected to be empty.
func emptyReadSetHash() string {
	return store.NewReadSet().Hash()
}

// failingSink returns the configured error from every Write. Used
// by manifest tests that need to exercise the inner-sink-failure
// path of recordingSink.
type failingSink struct{ err error }

func (f *failingSink) Write(emit.Target, []byte) error { return f.err }

// errFailingSink is the sentinel returned by [failingSink].
var errFailingSink = errors.New("pipeline: failing sink (test)")

// layoutGen is a one-shot generator used by Layout-phase tests. It
// implements [plugin.FilenameProvider] so the Layout phase has a
// suffix to consult, and adds a pre-built [emit.Package] to the
// store on Generate. Before AddPackage runs the helper stamps
// SetByName on every routable decl so the Layout phase resolves
// each decl's SetBy → suffix lookup without ambiguity.
//
// Tests construct the supplied package with controlled
// Origin / Target / Package state on each decl so the Layout
// algorithm can be exercised independently of plugin builder
// machinery.
type layoutGen struct {
	name    string
	suffix  string
	outputs []plugin.Output // takes precedence over suffix when non-empty
	pkg     *emit.Package
}

func (g *layoutGen) Name() string { return g.name }

func (g *layoutGen) Outputs(_ string) []plugin.Output {
	if len(g.outputs) > 0 {
		return g.outputs
	}
	if g.suffix == "" {
		return nil
	}
	return []plugin.Output{{Suffix: g.suffix}}
}

func (g *layoutGen) Generate(ctx *plugin.GeneratorContext) error {
	stampSetByOnEmitPackage(g.pkg, g.name)
	return ctx.Store.Emit().AddPackage(g.pkg)
}

// layoutGenNoSuffix is the [layoutGen] counterpart used to exercise
// the Layout phase's [pipeline.ErrMissingFilenameProvider] path: it
// emits routable decls without implementing
// [plugin.FilenameProvider], so the Layout phase has no suffix to
// resolve and surfaces the typed error per decl.
type layoutGenNoSuffix struct {
	name string
	pkg  *emit.Package
}

func (g *layoutGenNoSuffix) Name() string { return g.name }
func (g *layoutGenNoSuffix) Generate(ctx *plugin.GeneratorContext) error {
	stampSetByOnEmitPackage(g.pkg, g.name)
	return ctx.Store.Emit().AddPackage(g.pkg)
}

// stampSetByOnEmitPackage walks pkg and stamps SetByName = name on
// every routable decl in the package. Tests that hand-build emit
// packages (bypassing the builder constructors that normally stamp
// SetByName) call this so the Layout phase's plugin attribution
// lookup succeeds.
func stampSetByOnEmitPackage(pkg *emit.Package, name string) {
	for _, s := range pkg.Structs {
		s.SetByName = name
	}
	for _, i := range pkg.Interfaces {
		i.SetByName = name
	}
	for _, f := range pkg.Functions {
		f.SetByName = name
	}
	for _, vd := range pkg.Variables {
		vd.SetByName = name
	}
	for _, c := range pkg.Constants {
		c.SetByName = name
	}
	for _, e := range pkg.Enums {
		e.SetByName = name
	}
	for _, a := range pkg.Aliases {
		a.SetByName = name
	}
}

// nodePackageFE is a frontend that adds a single [*node.Package] to
// the store on Load. Layout-phase tests use it to seed the
// alongside-source package lookup so Target.Package and
// Target.ImportPath resolve to the configured values.
type nodePackageFE struct {
	name string
	pkg  *node.Package
}

func (f *nodePackageFE) Name() string { return f.name }
func (f *nodePackageFE) Load(ctx *plugin.FrontendContext) error {
	return ctx.Store.Nodes().AddPackage(f.pkg)
}

// multiNodePackageFE is a frontend that adds multiple
// [*node.Package] values to the store on Load. Used by Layout-phase
// tests whose alongside-source resolution needs to distinguish two
// origins by their source-package paths.
type multiNodePackageFE struct {
	name string
	pkgs []*node.Package
}

func (f *multiNodePackageFE) Name() string { return f.name }
func (f *multiNodePackageFE) Load(ctx *plugin.FrontendContext) error {
	for _, p := range f.pkgs {
		if err := ctx.Store.Nodes().AddPackage(p); err != nil {
			return err
		}
	}
	return nil
}

// slotContributingGen is a generator that calls a caller-supplied
// hook from Generate. Used by Layout-phase tests that exercise the
// origin-anchored slot-attachment path: the hook calls
// [store.EmitView.AppendOriginSlot] to queue a contribution and the
// Layout phase materialises it into the resolved File's named
// slot. The plugin implements [plugin.FilenameProvider] so the
// Layout phase has a suffix to compose with.
type slotContributingGen struct {
	name       string
	suffix     string
	contribute func(ctx *plugin.GeneratorContext) error
}

func (g *slotContributingGen) Name() string { return g.name }
func (g *slotContributingGen) Outputs(_ string) []plugin.Output {
	if g.suffix == "" {
		return nil
	}
	return []plugin.Output{{Suffix: g.suffix}}
}

func (g *slotContributingGen) Generate(ctx *plugin.GeneratorContext) error {
	if g.contribute == nil {
		return nil
	}
	return g.contribute(ctx)
}

// hasDiagContaining reports whether any diagnostic in d carries
// substr in its message or detail. Used by Layout-phase tests to
// verify the typed error surfaces in the expected diagnostic
// without coupling to its precise formatting.
func hasDiagContaining(d *diag.Sink, substr string) bool {
	for _, e := range d.Diagnostics() {
		if strings.Contains(e.Message, substr) || strings.Contains(e.Detail, substr) {
			return true
		}
	}
	return false
}

// layoutGenWithDirective extends [layoutGen] with the
// [plugin.DirectiveProvider] surface so the Layout-phase
// per-directive routing tests can drive the auto-allow path for
// `out=`/`pkg=` keys on the plugin's owning directive.
type layoutGenWithDirective struct {
	layoutGen
	schema directive.Schema
}

func (g *layoutGenWithDirective) Directives() []directive.Schema {
	return []directive.Schema{g.schema}
}
