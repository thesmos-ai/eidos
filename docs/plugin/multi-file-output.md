# Multi-file plugin output — emitting more than one file per source

A single plugin can declare an **ordered set of outputs**
(optionally tagged) and tag each emit decl with which output it
belongs to. The framework routes decls to the matching file by
suffix; the rest of the routing pipeline (Anchor, the `_test.go`
package shift, `+gen:out` overrides, project / CLI policy, the
cross-package qualifier on `emit.Internal` refs) flows on top
per output. This is how a single plugin emits, for example,
both `<src>_enum.go` (production code) and `<src>_enum_test.go`
(tests) from one source enum.

This guide covers the plugin contract, the per-decl tagging
surface, the precedence pipeline's per-output scoping, the
project-config schema, validation rules, and the conformance
contract. Read [routing.md][1] first — the precedence layers
described there compose with the per-output dispatch described
here.

[1]: routing.md

## TL;DR

| Concept | Surface | Use |
|--------|----------|-----|
| Plugin declares outputs | `Outputs(lang) []plugin.Output` | One entry per rendered file the plugin produces |
| Decl belongs to output | `BaseEmit.OutputTag` field | Set via `pkg.File(tag).<Decl>(...)`; empty = first output |
| Layout looks up suffix | `composeTarget` reads `OutputTag` | Matches against the plugin's declared `Output.Tag` |
| Per-output override | `+gen:out tag=<tag> <path>` / `-o <plugin>:<tag>=<path>` | Scopes routing overrides to one output |
| Per-output config | `output.<plugin>.tags.<tag>.<field>` | Project-level routing per output |

Single-file plugins declare one output, set no tags, and behave
identically to the pre-multi-output framework. Multi-file plugins
declare multiple outputs and tag their decls accordingly.

## The contract — `plugin.Output` + `Outputs(lang)`

The `FilenameProvider` capability returns a slice of outputs
keyed by tag. Each `Output` is the plugin's declaration of one
rendered file:

```go
type Output struct {
    // Tag is the stable per-plugin identifier for this output —
    // surfaces in `+gen:out tag=<tag>`, CLI -o flags, project
    // config, and the manifest. Empty for the plugin's primary
    // output (single-file plugins, or the default file in a
    // multi-output plugin).
    Tag string

    // Suffix is the per-source-basename filename suffix the
    // Layout phase appends. Required, non-empty. Composed as
    // <source-basename><Suffix> for alongside-source routing.
    Suffix string
}

type FilenameProvider interface {
    Plugin
    // Outputs returns the set of rendered files this plugin
    // produces in the given backend language. The pipeline
    // calls Outputs once per (plugin, language) after
    // WithPluginOptions applies and caches the result on the
    // resolved plan — static after Build, not static across
    // runs. Options changing between runs produces a different
    // Outputs slice and a different plugin cache identity.
    //
    // Returning nil or an empty slice signals the plugin has no
    // routable output in the requested language — the same
    // meaning as a non-implementer of FilenameProvider. A plugin
    // that ships templates for one language and stays silent for
    // others returns its slice for the supported language and
    // nil for the rest.
    Outputs(lang string) []Output
}
```

The slice is ordered, deterministic, and part of the plugin's
contract. External tools (CLI, config, manifest) reference
outputs by tag. The resolved Outputs slice participates in the
plugin's cache identity — option-driven variation produces
distinct cache entries.

## Per-decl tagging — `BaseEmit.OutputTag` + `pkg.File(tag)`

Every emit decl carries an `OutputTag string` field on `BaseEmit`.
Empty tag means "the plugin's primary output" (the Output at
index 0 in the slice — see the validation rules below). Non-empty
tag must match one of the plugin's declared `Output.Tag` values.

The recommended way to set the tag is the `PackageBuilder.File(tag)`
sub-context:

```go
// Default output — decls land in the plugin's primary file
// (single-file plugin behaviour, OutputTag stays empty).
pkg.Struct("Status", func(sb *builder.StructBuilder) { ... })

// Secondary output — decls built through the sub-context get
// OutputTag = "test" stamped automatically.
pkg.File("test").Function("TestStatusString_RoundTrip", func(fb *builder.FunctionBuilder) { ... })
```

`pkg.File(tag)` is memoised per tag on the parent PackageBuilder.
Repeated calls with the same tag return the same sub-context;
a plugin building N decls under `pkg.File("test")` in a loop
reuses one sub-context, not N. `pkg.File("")` is the identity
form — it returns the parent PackageBuilder unchanged, so plugin
code that programmatically computes tags from options can write
`pkg.File(maybeEmpty).<Decl>(...)` without special-casing the
default-output case.

`File` returns a `*PackageBuilder` decorated with the tag — the
full PackageBuilder API surface (Struct, Interface, Function,
Method, AppendOriginSlot, …) is available on the sub-context.
Nested `pkg.File("a").File("b")` overwrites: the second call
returns a sub-context tagged `"b"`, not a composed `"a.b"`.
Nesting is not a supported pattern — express each logical
sub-file as a single `pkg.File(<tag>)` call directly off the
root `pkg`.

The sub-context shares the parent PackageBuilder's underlying
`emit.Package`, Anchor default-origin, and routing state; only
`OutputTag` differs.

Each decl belongs to exactly one output. The `OutputTag` field
is a single string. A plugin that needs the same logical content
in two files (a helper function appearing in both production and
test outputs, say) emits the decl twice — once tagged for each
output. The framework never deduplicates decls across outputs.

## Default-tag semantics

The framework's default-tag rule keeps single-file plugin code
unchanged:

- A plugin declaring one output with an empty `Tag` produces
  decls with empty `OutputTag` — visually and structurally
  identical to today's single-file plugins.
- A plugin declaring multiple outputs must declare the
  primary one with an empty `Tag` at index 0. Decls without
  explicit `pkg.File(...)` use stamping land in that primary
  output.

A plugin can also declare every output with a non-empty tag and
require explicit tagging on every decl. In that mode, a decl
reaching Layout with empty `OutputTag` is a hard error — the
framework refuses to silently route it to `outputs[0]` because
the plugin's "no default output exists" intent was explicit.
See the validation rules table below for the diagnostic.

## Routing precedence — per-output scoping

The precedence layers from [routing.md][1] apply per output. A
decl with `OutputTag = "test"` flows through the same pipeline
(framework default → project layout → directive → CLI) as any
other decl; the layers consult per-output keys where they exist.
The `_test.go → <pkg>_test` package shift runs at the
framework-default layer per resolved Target — each tagged output
computes its own Target, and the shift fires independently when
that output's resolved filename ends in `_test.go` (or is
skipped when a higher layer already set the package on that
output).

### `+gen:out` with `tag=` scope

A `+gen:out` directive on a source applies to a specific output
when scoped with `tag=`:

```go
//+gen:out tag=test testkit/
//+gen:enum
type Status int
```

- Production-output decls land in `store/searcher_enum.go` (the
  default).
- Test-output decls (tag `test`) land in `store/testkit/searcher_enum_test.go`.

The override resolves only when the `enum` plugin's `Outputs`
slice declares a `test` tag; an unknown tag is a Layout-time
error per the unknown-tag rule in the validation table.

When two plugins emitting against the same origin each declare a
`test` tag in their own Outputs, an unscoped `+gen:out tag=test
<path>` applies to every such plugin — `tag=` without `plugin=`
expresses a cross-cutting intent ("route every plugin's test
output here"), mirroring how `out=` / `pkg=` already propagate
across companions. Reach for `plugin=<name> tag=test` to scope
strictly to one plugin's test output (form covered in the
intersection paragraph below).

`tag=` accepts a `pkg=` companion to pin the rendered package
clause on the targeted output independently:

```go
//+gen:out tag=test pkg=storetest testkit/
//+gen:enum
type Status int
```

- Production-output decls keep the source package (`store`).
- Test-output decls render under `package storetest` in
  `store/testkit/searcher_enum_test.go`.

`tag=`, `pkg=`, and `plugin=` are keyword arguments — order is
irrelevant and the positional path may appear anywhere among
them. `plugin=` and `tag=` compose as an intersection: an
override scoped `plugin=mock tag=test` applies only to the
`test` output of the `mock` plugin — not to any other plugin's
`test` output and not to the `mock` plugin's primary output.

Unscoped `+gen:out` on a multi-output plugin is **rejected** at
Layout time with a teaching diagnostic — uniform application
would silently collapse `_test.go` and `_main.go` into one file,
which Go's per-file test-classification rule then misreads.

Unscoped `+gen:out` on a single-output plugin continues to work
as today (no ambiguity — the plugin has one output, which IS the
default).

### Per-directive `tag=` on emitter-owned directives

The companion-aware form 3 from [routing.md][1] — `out=` and
`pkg=` keys on an emitter's own directive — recognises `tag=`
for per-output scoping:

```go
//+gen:mock tag=test out=tests/ pkg=mocktest
type Store interface { ... }
```

The override applies only to the `mock` plugin's `test` output;
the primary output keeps the framework default. Unlike `out=`
and `pkg=` (which propagate to every companion plugin emitting
against the same origin), `tag=` scopes strictly within the
emitter's own output namespace — tag values are plugin-scoped,
and propagating one would route a sibling plugin to a tag it
doesn't declare. Companion plugins continue to share the `out=`
/ `pkg=` envelope but each routes its own output set
independently.

Form 3 without `tag=` follows the same "no implicit
multi-output collapse" rule as form 2: an unscoped `out=` that
would force two outputs to share a filename is rejected at
Layout time with the same diagnostic. Directory-only overrides
stay safe — the per-output suffixes keep filenames distinct
within the shared directory.

### CLI `-o <plugin>=<path>` / `-o <plugin>:<tag>=<path>`

CLI flag syntax mirrors the directive's scope:

- `-o mock=mocks/handlers.go` — overrides the `mock` plugin's
  primary (default) output. Backward-compatible with the existing
  CLI form.
- `-o mock:test=tests/handlers.go` — overrides the `mock` plugin's
  `test` output specifically.

One `=` separator between key and value; `:` lives inside the key
to disambiguate plugin+tag from path-with-colon.

## Project-config schema

The `output` block accepts per-plugin and per-tag overrides:

```yaml
output:
  # Single-output plugin (no tags): plugin-level routing block.
  buildergen:
    layout: alongside-source
    dir: internal/builders

  # Multi-output plugin: per-tag routing via `tags:`.
  mock:
    layout: alongside-source        # applies to primary output
    tags:
      test:
        layout: alongside-source    # applies to `test` output
        dir: testkit
```

The `tags:` sub-namespace avoids collisions between tag names and
field names — a plugin shipping a `format` tag and a `format`
field coexist cleanly.

## Validation rules

The framework rejects malformed `Outputs` slices at Build time:

| Rule | Diagnostic |
|------|-----------|
| Output with empty `Suffix` | `output #<i>: Suffix is required` |
| Duplicate `Tag` values | `outputs declare tag %q twice` |
| More than one Output with empty Tag | `at most one output may declare an empty Tag (the plugin's primary output)` |
| Output with empty Tag exists but is not at index 0 | `output with empty Tag must be declared at index 0` |
| Decl with `OutputTag = "xyz"` not in plugin's declared outputs | Layout-time error: `decl tags unknown output %q on plugin %q (declared: %v)` |
| Decl with empty `OutputTag` on plugin declaring no empty-Tag output | Layout-time error: `decl carries empty OutputTag; plugin %q declares no default output, every decl must use pkg.File(<tag>)` |

Build-time validation fires regardless of whether the plugin
runs `plugintest.RunSuite` — the framework enforces every rule
in this table on its own. The conformance suite is the earlier
signal for authors who run it during development: it checks the
static shape of Outputs at registration time so violations
surface before a pipeline run.

## Manifest reporting

The per-target manifest entry's plugin attribution gains an
optional `output_tag` field. Single-output runs and primary-output
files omit it (the field is `omitempty`), keeping byte-stable
parity with manifests produced before multi-output support
landed. Secondary outputs surface the tag. The same enum source
producing both files records side-by-side entries — the primary
matches a pre-multi-output manifest byte-for-byte, and the
secondary carries the tag explicitly:

```json
[
  {
    "target": { "dir": "store", "filename": "searcher_enum.go" },
    "plugins": [{"name": "enum", "version": "1.0.0"}]
  },
  {
    "target": { "dir": "store", "filename": "searcher_enum_test.go" },
    "plugins": [{"name": "enum", "version": "1.0.0", "output_tag": "test"}]
  }
]
```

### Cross-tool naming convention

Tags are a **plugin-scoped namespace**, not a global one — two
plugins may each declare a `test` tag without collision. Tooling
that surfaces tags must preserve scope when the surface is
human-readable; the CLI `-o <plugin>:<tag>=<path>` form
establishes `<plugin>:<tag>` as the canonical rendering, and
explain commands, diagnostics, log lines, and any other free-text
surface should follow the same shape (`enum:test`, `mock:test`,
…) so a reader of a multi-plugin pipeline can tell whose `test`
output a message refers to. The manifest JSON itself is exempt:
the structural pairing of `"name"` and `"output_tag"` under the
same plugin entry already scopes the tag unambiguously, and
structured consumers should read both fields together rather
than reconstructing the `<plugin>:<tag>` string.

## Migration from `FilenameSuffix`

`FilenameSuffix(lang) string` is removed in favour of
`Outputs(lang) []Output`. The single-file migration is one entry
returning the existing suffix:

```go
// Before
func (*Plugin) FilenameSuffix(lang string) string {
    if lang == "golang" {
        return "_mock.go"
    }
    return ""
}

// After
func (*Plugin) Outputs(lang string) []plugin.Output {
    if lang != "golang" {
        return nil
    }
    return []plugin.Output{{Suffix: "_mock.go"}}
}
```

Reference plugins (~7 sites) migrate identically. Test fixtures
do the same. No behaviour change for any existing single-output
plugin.

## Multi-output example — the enum stringer pattern

A single enum plugin emitting both production code and tests:

```go
func (*Plugin) Outputs(lang string) []plugin.Output {
    if lang != "golang" {
        return nil
    }
    return []plugin.Output{
        {Suffix: "_enum.go"},          // primary, empty tag
        {Tag: "test", Suffix: "_enum_test.go"},
    }
}

func (p *Plugin) Generate(ctx *sdk.GeneratorContext) error {
    for _, e := range ctx.Reader.Enums().Slice() {
        if !e.HasPositiveDirective(DirectiveName) {
            continue
        }
        pkg := builder.For(Name).Anchor(e)
        p.emitProduction(pkg, e)
        p.emitTests(pkg.File("test"), e)
        out, err := pkg.Build()
        if err != nil {
            return err
        }
        if err := ctx.Store.Emit().AddPackage(out); err != nil {
            return err
        }
    }
    return nil
}
```

`emitProduction` builds decls through `pkg` directly — they land
in the primary output (`<src>_enum.go`). `emitTests` builds
through `pkg.File("test")` — those decls carry `OutputTag = "test"`
and land in `<src>_enum_test.go`. The framework's `_test.go →
<pkg>_test` shift applies to the test file's resolved package
clause automatically.

The two files share the same source anchor, the same directory,
and (until any directive override) the same project layout
policy. Per-output routing overrides target individual tags
without affecting the other.

## When to reach for siblings instead

Multi-output and sibling plugins both produce more than one file
from the same source — the distinction is **shared lifecycle**.
Use multi-output when the files are tightly coupled and must
move together:

- They share the same source anchor and the same project routing
  policy.
- A user enabling the plugin enables every output; disabling
  the plugin disables every output.
- Per-output overrides are the rare case, not the common one.
- The files' contents conceptually represent one feature
  (production code + the tests that pin its contract; a builder and its compile-time interface assertion).

Reach for sibling plugins when the files have independent
lifecycles or compose into different consumer pipelines:

- A user may want one without the other (mock generation without
  recording overlays; a builder without the JSON-schema export
  it pairs with).
- They run in different priority buckets, plug into different
  generator phases, or react to different directives.
- Their output volumes / cadences differ enough that bundling
  them would confuse users about which plugin produced which
  file.

The mock plugin family is the canonical sibling pattern — `mock`,
`mocktest`, and `mockrecord` each own a distinct concern and a
distinct user opt-in even though all three share the same source
interface anchor. The enum plugin's production code + tests pair
is the canonical multi-output pattern — splitting them into two
plugins would force users to opt into both for the tests to be
useful, with no scenario where one without the other makes sense.

## Composition with weaver plugins

Weaver-style plugins — pure slot-contributing cross-cutters that
decorate decls owned by other plugins (auditweaver, debugweaver,
recording / fault-injection / tracing) — compose with multi-output
hosts without API change:

- **Slot contributions on a host decl** (Prebody, Postbody,
  FieldsSlot, MethodsSlot, …) inherit the host's `OutputTag`. The
  weaver appends a `Stmt` / `Field` / `Method` to the host's slot;
  the framework's render path emits the contribution where the
  host decl renders. The weaver never sees `OutputTag` and is not
  required to implement `Outputs`. Pure-weaver plugins keep the
  "no `FilenameProvider`" signal that already exists today.

  A weaver iterating decls across a multi-output host therefore
  contributes to every output the host emits — its contributions
  flow with each decl's routing. A debug-tracer decorating the
  enum plugin's `String` method (production output) and its
  `TestStringRoundTrip` function (test output) lands one prebody
  in `<src>_enum.go` and another in `<src>_enum_test.go` from a
  single iteration, with no per-output coordination. Weavers that
  want to scope to one output filter on `OutputTag` (or on a
  per-host meta key) in the iteration body — the field is exposed
  via `BaseEmit` like every other routing-relevant attribute.
- **Origin-anchored slot contributions** (file-level slots
  appended via `AppendOriginSlot` against a source node) compose
  through the weaver's *own* `Outputs` slice — the framework
  routes the contribution to the weaver's primary output by
  default. A weaver targeting a specific own-output uses
  `pkg.File(tag).AppendOriginSlot(...)` the same way emit decls
  do.
- **Weavers that emit routable decls** (rare; the framework
  recommends splitting into a generator + a weaver instead)
  implement `Outputs(lang)` like any other generator.

Cross-plugin output reach — a weaver targeting another plugin's
output namespace (e.g. adding a helper function to the `enum`
plugin's `test` output from outside `enum`) is intentionally not
supported by this spec. `OutputTag` values are plugin-scoped;
a contribution lives in either the host decl's file (via slot
inheritance) or the contributing plugin's own file (via its own
`Outputs`). Cross-plugin file-sharing patterns — when they
emerge — need explicit cross-plugin coordination (published meta
keys, shared anchors) outside the routing surface.

## Composition with templates

Multi-file output is **orthogonal** to the [templates surface][tmpl]:
templates control *how* an emit decl renders into text, outputs
control *where* that text lands. A plugin shipping templates for
custom emit kinds (e.g. an `enum.Stringer` kind) gains nothing or
loses nothing from declaring multiple outputs — the template
renders the kind's text; the framework's per-output routing
deposits the rendered text into the right file based on
`OutputTag`.

Two patterns plugin authors use to interleave templates with
multi-output emission:

[tmpl]: templates.md

### Single template, branches on `OutputTag`

The decl carries `OutputTag` through to the render context. A
single `.tmpl` file can branch on it for kinds whose rendered
shape varies per output:

```gotemplate
{{- define "enum.stringer" -}}
{{- if eq .OutputTag "test" -}}
{{- /* test-side rendering: helpers, table-driven cases */ -}}
{{- else -}}
{{- /* production rendering: name const, index var, String method */ -}}
{{- end -}}
{{- end -}}
```

### Distinct emit kinds per output

Plugin authors who prefer separate render shapes give the
production-side and test-side decls **different emit kinds** and
ship one template per kind. The backend's existing
`Kind()`-based dispatch routes each decl to its own template
without any per-tag knowledge — `OutputTag` decides which file
the rendered text lands in; `Kind()` decides what the rendered
text looks like.

```go
type stringer struct{ emit.BaseEmit; ... }
func (*stringer) Kind() kind.Kind { return "enum.stringer" }

type stringerTest struct{ emit.BaseEmit; ... }
func (*stringerTest) Kind() kind.Kind { return "enum.stringer.test" }
```

The plugin ships `enum.stringer.tmpl` and `enum.stringer.test.tmpl`
via `sdk.TemplateProvider`; the backend resolves each through the
funcmap's `render` entry as it would any plugin-defined kind. The
two outputs read cleanly as two kinds, with no `OutputTag`
branches in either template.

The framework imposes no contract on which pattern a plugin
picks. Branch-on-`OutputTag` keeps one template and one kind;
distinct kinds keep templates focused at the cost of two kind
registrations. Both compose cleanly with the per-decl tagging
surface — neither requires backend changes.

## Conformance contract

`plugintest.RunSuite` adds a static check on `Outputs(lang)`:

- The slice satisfies the validation rules above.
- The slice is deterministic — multiple calls with the same
  options return equal slices (the pipeline relies on the
  caching of the static set after `WithPluginOptions` applies).
- For backends the plugin claims to support (`Outputs` returns
  non-empty), every declared Suffix renders successfully through
  the backend's per-target render path.

Plugin authors run the suite against every backend language they
contribute to; the suite catches output-shape regressions before
the plugin reaches a real pipeline.
