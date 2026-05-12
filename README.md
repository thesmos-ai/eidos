# eidos

[![CI](https://github.com/thesmos-ai/eidos/actions/workflows/ci.yml/badge.svg)](https://github.com/thesmos-ai/eidos/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/go.thesmos.sh/eidos.svg)](https://pkg.go.dev/go.thesmos.sh/eidos)
[![Go Report Card](https://goreportcard.com/badge/go.thesmos.sh/eidos)](https://goreportcard.com/report/go.thesmos.sh/eidos)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/thesmos-ai/eidos)](go.mod)

A plugin-driven code-generation library built around typed metadata, a
queryable intermediate representation, and composable injection slots.
Output is byte-deterministic, `gofmt`-clean, and cache-friendly.

```
Source files → Frontend → Annotators → Generators → Backend → Sink
                          (stamp meta) (emit fragments) (render)
```

## What it is

`eidos` is a Go library for assembling code generators. Hosts embed it
and compose a pipeline from plugins:

- a **Frontend** parses source into a language-agnostic node graph
  (the source-side IR);
- **Annotators** detect patterns and stamp typed metadata onto nodes;
- a directive-override step lets users override any of that from
  source comments (`+gen:`, `-gen:`);
- **Generators** read the model and emit code fragments on a parallel
  output-side IR — including fragments that target named *slots* on
  other generators' output;
- a **Backend** runs the fragments through templates, resolves
  imports, formats, and writes through a configurable sink.

The supported target today is Go → Go. The pipeline, plugin contracts,
and slot model are language-agnostic; additional frontends and backends
slot in through the same plugin interfaces.

## Quality contracts

These are guarantees the library provides, tested in CI:

- **Determinism.** The same inputs produce byte-identical output
  across runs and machines. Topological ties break alphabetically;
  every output carries a SHA-256 provenance hash over its body bytes;
  the header and footer carry no run-dependent fields.
- **Composability.** Cross-cutting concerns stack into typed slots on
  generated code in deterministic order — capability-topological
  across plugins, append-sequence within a plugin. Generators don't
  become god-objects.
- **Provenance.** Every metadata entry, slot contribution, and emit
  entity carries `(setBy, authority, sourcePos)` provenance. The
  authority ladder (plugin < directive < manual) resolves conflicting
  writes deterministically.
- **Parallel safety.** Annotators with disjoint `Provides` may run
  concurrently; backend per-file rendering is concurrency-safe; the
  Store and emit graph are race-detector clean and enforce mutability
  windows per phase.
- **Caching.** Per-plugin, per-input cache keys; bumping a plugin's
  version (via the optional `Versioned` capability) invalidates only
  that plugin's outputs.
- **Panic isolation.** A plugin that panics produces an `Error`
  diagnostic with a stack trace; subsequent plugins still run; the
  pipeline returns a structured error rather than a raw panic.
- **Layering.** `depguard` rules in `.golangci.yml` enforce the
  package layering: `node/` and `emit/` are language-agnostic and
  forbidden from importing language-specific stdlib (`go/ast`,
  `go/format`, `text/template`); `core/*` cannot import any specific
  frontend or backend; frontends and backends cannot import each
  other.

## Installation

```bash
go get go.thesmos.sh/eidos
```

Requires Go 1.26 or later.

## Minimal example

A working pipeline rendering one greeting struct per source struct,
alongside each source file:

```go
package main

import (
    "context"
    "fmt"

    bgolang "go.thesmos.sh/eidos/backend/golang"
    "go.thesmos.sh/eidos/core/position"
    "go.thesmos.sh/eidos/eidostest/testpipe"
    "go.thesmos.sh/eidos/emit"
    "go.thesmos.sh/eidos/emit/builder"
    "go.thesmos.sh/eidos/node"
    "go.thesmos.sh/eidos/pipeline"
    "go.thesmos.sh/eidos/plugin"
    "go.thesmos.sh/eidos/sink"
)

// helloGenerator emits one `<Source>Greeting` struct per source
// struct. It implements plugin.FilenameProvider so the routing
// layer composes `<src-basename>_hello.go` for each rendered file,
// and sets Origin on every emitted decl so the Layout phase can
// resolve Dir / Package / ImportPath from the source.
type helloGenerator struct{}

func (helloGenerator) Name() string { return "hellogen" }

// FilenameSuffix returns the per-source suffix the routing layer
// appends to the source basename. The plugin ships Go output today;
// other backends receive the empty signal until matching templates
// land. The suffix is the only output-naming hook plugins surface —
// directory, package, and import-path are framework concerns.
func (helloGenerator) FilenameSuffix(lang string) string {
    if lang == "golang" {
        return "_hello.go"
    }
    return ""
}

func (g helloGenerator) Generate(ctx *plugin.GeneratorContext) error {
    c := builder.For(g.Name(), emit.Target{})
    for _, src := range ctx.Reader.Structs().Slice() {
        pkg, err := c.Package(src.Package, src.Package).
            Struct(src.Name+"Greeting", func(s *builder.StructBuilder) {
                s.Origin(src)
                s.Field("Message", emit.Builtin("string"), nil)
            }).
            Build()
        if err != nil {
            return err
        }
        if err := ctx.Store.Emit().AddPackage(pkg); err != nil {
            return err
        }
    }
    return nil
}

func main() {
    src := &node.Package{Name: "x", Path: "x"}
    src.Structs = []*node.Struct{{
        Name:     "User",
        Package:  src.Path,
        BaseNode: node.BaseNode{SourcePos: position.Pos{File: "user.go", Line: 1}},
    }}
    p, err := pipeline.New().
        WithFrontend(testpipe.FromNodes(src)).
        WithGenerator(helloGenerator{}).
        WithBackend(bgolang.New()).
        WithSink(sink.NewDisk("./out")).
        Build()
    if err != nil {
        fmt.Println(err)
        return
    }
    if err := p.Run(context.Background()); err != nil {
        fmt.Println(err)
    }
}
```

The rendered output lands at `./out/user_hello.go`. Layout
composes the filename from the source basename (`user`) and the
plugin's declared suffix (`_hello.go`); package and directory come
from the source struct's package.

For real Go-source input, swap `testpipe.FromNodes(...)` for
`frontend/golang.New()` and pass package patterns to `p.Run`:

```go
p.Run(ctx, "./...")
```

## Public API surface

### Composing a pipeline

```go
pipeline.New().
    WithFrontend(p plugin.Frontend).
    WithAnnotator(p plugin.Annotator).
    WithGenerator(p plugin.Generator).
    WithBackend(p plugin.Backend).
    WithSink(s sink.Sink).
    WithCache(c cache.Cache).
    WithDiag(s *diag.Sink).
    WithDirective(schemas ...directive.Schema).
    WithDirectivePrefix(prefix string).
    WithParallel(phases ...pipeline.Phase).
    WithPluginOptions(name string, kv map[string]string).
    WithManifestPath(path string).
    WithVerbose(v bool).
    WithOutputLayout(layout string).         // alongside-source | centralised
    WithOutputPackage(name string).          // pins Target.Package for every decl in scope
    WithOutputDir(dir string).               // centralised-layout output directory
    WithOutputFilename(filename string).     // pins Target.Filename for every decl in scope
    WithTargetSymbol(name string).           // scope filter; matches Name or QName suffix .Name
    Build()           // (*Pipeline, error)
```

`Build` returns sentinel errors callers compare with `errors.Is`:
`ErrNoFrontend`, `ErrNoBackend`, `ErrMultipleBackends`, `ErrNoSink`,
`ErrDuplicatePlugin`, `ErrDuplicateProvider`, `ErrCycle`,
`ErrInvalidOptions`, `ErrDuplicateDirective`, `ErrIncompatibleEmitVersion`,
`ErrInvalidDirectivePrefix`, `ErrTemplateFuncCollision`. The full list
lives in `pipeline/errors.go`.

`Pipeline.Run(ctx, patterns...)` runs the configured pipeline and
returns `ErrRunHadErrors` when any plugin emitted an Error diagnostic.
`Pipeline.DryRun(ctx)` returns the resolved `*Plan` without executing.

### Plugin role interfaces

Every plugin implements `plugin.Plugin` (`Name() string`) plus one or
more role interfaces:

```go
type Frontend interface {
    Plugin
    Load(*FrontendContext) error
}

type Annotator interface {
    Plugin
    Annotate(*AnnotatorContext) error
}

type Generator interface {
    Plugin
    Generate(*GeneratorContext) error
}

type Backend interface {
    Plugin
    Language() string
    Render(*BackendContext) error
}
```

Optional capabilities a plugin may also implement:

- `CapabilityProvider` — declares `Provides` / `Requires` capability
  names for capability-topological ordering within the role bucket.
- `OptionsProvider` — declares a typed `OptionsSchema` (`required`,
  `default`, `one_of`, custom validators); plugin options surface as
  positioned diagnostics on misconfiguration rather than silent
  no-ops.
- `Versioned` — declares the plugin's emit-contract version so the
  backend can detect incompatible-version pairings at `Build` time
  and so cache keys invalidate when the plugin's contract changes.
- `TemplateProvider` (Backend-side) — ships a `fs.FS` of templates and
  a funcmap merged into the backend's funcmap with conflict resolution
  by capability topology.
- `DirectiveProvider` — declares directive schemas (`AppliesTo`,
  `RequiredKeys`, `AllowedKeys`, `MutuallyExclusiveWith`,
  `PositionalArgs`).
- `FilenameProvider` — declares the per-source filename suffix the
  routing layer appends to each origin's source basename. **Required**
  for any generator that emits routable decls or file-level slot
  contributions; pure cross-cutting plugins that only attach to other
  plugins' methods do not implement it.

### Building emit graphs

Plugin Generators and Annotators assemble their output through
`emit/builder` rather than hand-wiring `emit.Package` /
`emit.Struct` / ... struct literals. The builder threads
`Target`, `Owner` back-pointers, and slot `Provenance.SetBy`
automatically so plugin code stays focused on intent.

```go
// Bind plugin identity for Provenance.SetBy stamping. The Target
// argument is reserved for builder-internal threading; the routing
// layer composes the final Target from Origin and the resolved
// per-plugin Layout policy, so plugin code passes the zero value
// and sets Origin on each emitted decl instead.
c := builder.For("user-repo-gen", emit.Target{})

pkg, err := c.Package("users", "example.com/users").
    Struct("Repo", func(s *builder.StructBuilder) {
        s.Origin(src) // src is the *node.Struct this decl derives from
        s.Field("db", emit.External("database/sql", "DB"), nil)
        s.Method("Get", func(m *builder.MethodBuilder) {
            m.Receiver("r", emit.Ptr(emit.Internal(s.Node())))
            m.Param("ctx", emit.External("context", "Context"))
            m.Param("id", emit.Builtin("string"))
            m.Return(emit.Ptr(emit.External("example.com/users", "User")))
            m.Return(emit.Builtin("error"))
        })
    }).
    Build()
```

Structural rule violations (e.g. a method on a true alias)
accumulate on the builder and surface from `Build`; the partial
graph is still returned so callers can render best-effort output
alongside diagnostics.

### Cross-cutting slot contributions

Cross-cutting plugins use the same `Context` to append into named
slots on emit values built by other plugins. The available slots
cover the common composition points: per-method `Prebody` /
`Postbody`, per-struct `Field` / `Method`, per-file `Top` /
`Bottom` / `Init` / `Imports`, and a few more. Each `Append*`
call stamps `Provenance.SetBy` automatically and accepts an
optional anchor id later contributions can position themselves
relative to.

A "debug tracer" generator that injects a `log.Printf` at the
top of every emitted method's body:

```go
func (p *debugTracer) Generate(ctx *plugin.GeneratorContext) error {
    c := builder.For(p.Name(), emit.Target{})
    for _, m := range ctx.Reader.EmitMethods().Slice() {
        stmt := emit.NewExprStmt(emit.NewCall(
            emit.NewField(emit.NewIdent("log"), "Printf"),
            emit.NewLiteralString("debug: "+m.Name+" entered"),
        ))
        if err := c.AppendPrebody(m, stmt, "trace.entry"); err != nil {
            return err
        }
    }
    return nil
}
```

A "registry" generator that lands one `registry.Register(...)`
call per `+gen:register` struct into the resolved file's
`func init() { ... }` block, anchored to each source struct's
Origin so the routing layer composes the destination file:

```go
func (p *registryGen) Generate(ctx *plugin.GeneratorContext) error {
    c := builder.For(p.Name(), emit.Target{})
    for _, s := range ctx.Reader.Structs().Where(store.WithDirective[*node.Struct]("register")).Slice() {
        stmt := emit.NewExprStmt(emit.NewCall(
            emit.NewField(emit.NewIdent("registry"), "Register"),
            emit.NewLiteralString(s.Name),
            emit.NewComposite(emit.External(s.Package, s.Name), nil),
        ))
        if err := ctx.Store.Emit().AppendOriginSlot(
            s, "init", stmt, c.Provenance("registry."+s.Name),
        ); err != nil {
            return err
        }
    }
    return nil
}
```

The Layout phase resolves each contribution's Origin to a rendered
file using the same precedence model that routes standalone decls.
Multiple contributions resolving to the same file compose into one
`init` block.

Ordering across plugins is capability-topological with append
order as the tiebreaker. A later plugin that wants to position
its statement relative to one of the calls above uses the
context's positional inserts:

```go
c.InsertPrebody(method, stmt, builder.After("trace.entry"))
```

`Before` / `After` target a `Provenance.ID` anchor; `Prepend`
and `At(index)` are the absolute alternatives.

The framework also provides two helpers that satisfy common plugin
contracts without per-plugin boilerplate:

- `opt.Bind(&p.opts)` returns an `*opt.Holder[Options]` that
  plugins embed to pick up `OptionsSchema` / `SetOptions` via
  method promotion.
- `directive.HasPositive` / `directive.HasNegated` (plus
  `HasPositiveDirective` / `HasNegatedDirective` methods on
  `node.BaseNode` and `emit.BaseEmit`) express
  opt-in / opt-out gating without per-plugin directive walks.

### Sinks and caches

```go
sink.NewDisk(root string) *Disk           // writes files under root/
sink.NewMemory() *Memory                  // in-memory map; for tests
sink.NewMulti(sinks ...Sink) *Multi       // fan-out
sink.NewStdout(w io.Writer) *Stdout       // single stream
```

```go
cache.NewDisk(root string) *Disk          // persistent
cache.NewNone() *None                     // disabled
```

### Frontend / backend implementations

- `frontend/golang.New()` — Go AST → node graph; populates `go.*`
  metadata keys (`go.iterValueType`, `go.elementType`, …) consumed
  by downstream annotators.
- `backend/golang.New()` — renders the emit graph to gofmt-clean Go
  source through a template-driven pipeline. Contract documented in
  [`docs/backend/golang.md`](docs/backend/golang.md).

## Determinism and provenance

Every file the Go backend writes ends in a two-line footer:

```go
// <brand>: end of generated content.
// <brand>:provenance <sha256-of-body-bytes>
```

The hash is over the body bytes alone (header and footer excluded), so
the same emit graph produces an identical hash across runs regardless
of `Command` or `Plugins` header text. The header itself carries no
timestamp — two runs over the same input produce byte-identical files,
header and footer included. `Brand` defaults to `eidos`; library
embedders set `BackendContext.Brand` to re-brand their output.

The provenance trail is queryable in-process: every `meta.Entry`
carries `(setBy, authority, sourcePos)`; every slot contribution
carries the contributing plugin's name; every emit entity threads its
`OriginNode` back to the source-side IR. See
[`docs/backend/golang.md`](docs/backend/golang.md) for the full
envelope contract and the `imp` / `slot` / `provenance` template
funcmap entries.

## Test harness

`eidostest/` ships in-tree helpers for unit and integration tests.

`eidostest/storefixture` — fluent builders for hand-crafting a
source-side node graph without parsing real source:

```go
import "go.thesmos.sh/eidos/eidostest/storefixture"

pkg := storefixture.New().
    Package("users", "example.com/users").
    Struct("User", func(b *storefixture.StructBuilder) {
        b.Field("ID", storefixture.Named("string"), nil)
        b.Field("Email", storefixture.Named("string"), nil)
    }).
    PackageNode()
```

`eidostest/testpipe` — full pipeline harness over an in-memory sink
with golden-file diffing:

```go
import "go.thesmos.sh/eidos/eidostest/testpipe"

p := testpipe.New(t).
    WithFrontend(testpipe.FromNodes(pkg)).
    WithGenerator(myGen).
    WithBackend(backend_golang.New()).
    Build()
p.Run("./...")
p.AssertFile("user.go").
    Contains("type User struct").
    MatchesGolden("testdata/user.go.golden")
```

The package registers a `-update-golden` flag; run the test binary
with `-update-golden` to rewrite golden fixtures atomically.

`core/diag.Capture()` / `core/diag.Discard()` produce diagnostic
sinks for tests that respectively assert on or ignore emitted
diagnostics.

## Project layout

```
node/                 source-side IR (language-agnostic)
emit/                 output-side IR (language-agnostic)
emit/builder/         fluent decl + slot-contribution API plugins use
store/                two-view (Source / Emit) store with mutability windows
writer/               file-builder primitives: ImportSet, Header, Footer
sink/                 Sink interface + Disk / Memory / Multi / Stdout
cache/                Cache interface + Disk / None
manifest/             written-output manifest for prune support
priority/             capability-topo sort helpers
plugin/               role interfaces (Frontend, Annotator, Generator, Backend)
                      plus optional capabilities (Versioned, OptionsProvider,
                      CapabilityProvider, TemplateProvider, …)
pipeline/             Builder + Pipeline orchestration

core/                 language-agnostic foundation primitives:
  core/diag/            diagnostics (Info / Warn / Error) with positions
  core/directive/       +gen: / -gen: parsing, schemas, registry
  core/meta/            typed metadata keys, authority levels, provenance
  core/naming/          case conversion (Pascal / Camel / Snake / Screaming / Title)
  core/opt/             typed plugin-options primitives
  core/position/        Pos / Range

frontend/golang/      Go AST → node graph + go.* metadata
backend/golang/       Go renderer: templates, funcmap, ImportSet, gofmt

eidostest/
  eidostest/storefixture/   typed source-graph builders
  eidostest/testpipe/       pipeline harness, golden-file diffing
  eidostest/pluginfixture/  plugin-defined emit-kind test fixture

docs/
  docs/backend/golang.md    Go-backend contract reference (template set,
                            funcmap, envelope, sentinels)
  docs/frontend/golang.md   Go-frontend contract reference
```

Layering is enforced by `depguard` in `.golangci.yml`.

## Design decisions

A few choices worth being explicit about — each one trades something
away, deliberately.

**Plugins are static Go imports, not dynamically loaded.** A
different pipeline is a different binary. The cost is that swapping
plugins requires a rebuild; the wins are compile-time type-checking
on plugin contracts, deterministic ordering, single-binary deployment,
and no `plugin.Open` complexity.

**Metadata is the universal extension mechanism.** Plugins do not
subclass each other or call into each other directly. They
communicate through typed, namespaced metadata keys with explicit
authority levels (plugin / directive / manual) and full provenance.
Source can override anything with `+gen:meta KEY=value` or delete it
with `-gen:meta KEY` — no plugin code change required.

**Composition through slots, not inheritance.** A generator emits
emit-entities with named slots typed by content kind. Cross-cutting
plugins append into the relevant slot; ordering is capability-topo
across plugins with alphabetical tie-break.

**Templates are owned by each backend and each plugin, not the
framework.** Adding a target language is mostly authoring templates
plus a small format/imports pass. The Go backend's core templates
live in `backend/golang/templates/`; plugins ship their own templates
that merge into the same funcmap, with override resolution by
capability topology.

**Single backend per pipeline run.** Generating the same project to
multiple languages = multiple pipeline runs. Keeps import resolution,
formatting, and target conventions monolingual and predictable per
invocation.

## Contributing

See [`CONTRIBUTING.md`](CONTRIBUTING.md) for development workflow,
linting, and test-coverage expectations. Security issues:
[`SECURITY.md`](SECURITY.md).

## License

MIT. Copyright Thesmos B.V. See [`LICENSE`](LICENSE).
