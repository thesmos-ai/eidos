# Templates with generator plugins

When a generator emits a plugin-defined emit kind — a custom
type outside the `emit.*` namespace, like `Registration` from
`registrygen` or `Saga` from a hypothetical workflow plugin —
the backend needs to know how to render it. The
`sdk.TemplateProvider` capability is the bridge: a plugin ships
templates alongside its code, the backend picks them up, and
the rendered output flows through the same finalisation passes
(import resolution, `gofmt`, header stamping) as the core emit
kinds.

This guide walks through the full pattern end-to-end using
[`reference/registrygen`](../../reference/registrygen) as the
canonical example.

## The capability

```go
type TemplateProvider interface {
    Templates(lang string) (fs.FS, bool)
    TemplateFuncs(lang string) template.FuncMap
    TemplateOverrides(lang string) template.FuncMap
}
```

All three methods are language-scoped via the `lang` argument
(`"golang"`, `"rust"`, `"ts"`, …). A plugin that contributes
templates only to the Go backend returns `(nil, false)` from
`Templates` for every other language.

## Step 1: define the emit kind

A plugin-defined emit kind is a Go struct embedding
`emit.BaseEmit` plus a `Kind()` method returning a namespaced
`kind.Kind` constant. From registrygen:

```go
package registrygen

import (
    "go.thesmos.sh/eidos/core/kind"
    "go.thesmos.sh/eidos/emit"
)

// Kind keeps the registration kind outside the emit.* namespace
// reserved for core emit types.
const Kind kind.Kind = "registrygen.registration"

type Registration struct {
    emit.BaseEmit

    Name         string
    NameLit      *emit.Expr  // pre-built string literal
    Init         *emit.Expr  // value passed to register call
    RegisterFunc *emit.Expr  // the register call's callee
}

func (*Registration) Kind() kind.Kind { return Kind }

// Compile-time check that the type satisfies emit.Node.
var _ emit.Node = (*Registration)(nil)
```

**Naming convention**: the kind string is `<plugin>.<entity>`.
The dotted spelling matches the template-naming convention
below and keeps the kind discoverable in diagnostics.

## Step 2: write the template

Templates live in a per-language subdirectory the plugin owns,
typically `templates/<lang>/<entity>.tmpl`. Use `//go:embed`
to ship them as part of the plugin's binary.

From `reference/registrygen/templates/golang/registration.tmpl`:

```
{{- define "registrygen.registration" -}}
{{ renderExpr .RegisterFunc }}({{ renderExpr .NameLit }}, {{ renderExpr .Init }})
{{- end -}}
```

That's the entire template. It defines a Go `text/template`
named `registrygen.registration` — **matching the
`Kind.String()` verbatim** — that emits a single line:

```go
log.Print("Article", Article{})
```

**Naming contract**: every emit kind needs a template whose
name equals its `Kind()` value. The backend's main render
function (`render` in the funcmap) dispatches by
`Node.Kind()`, so template selection is a string match.

**Reserved prefix**: template names beginning with `fragment.`
are reserved for future shared partials; using one fails Build
with `ErrReservedTemplatePrefix`.

## Step 3: ship the template via `Templates`

```go
//go:embed templates/golang/*.tmpl
var templatesFS embed.FS

func (*Plugin) Templates(lang string) (fs.FS, bool) {
    if lang != "golang" {
        return nil, false
    }
    sub, _ := fs.Sub(templatesFS, "templates/golang")
    return sub, true
}
```

The backend's template-collection step walks every plugin's
returned `fs.FS` looking for `*.tmpl` files, parses each, and
adds the defined names to the rendering tree. A plugin can ship
multiple `.tmpl` files; each may define multiple templates.

## Step 4: emit the entity in `Generate`

A generator constructing a plugin-defined emit kind looks the
same as one constructing a core kind. From registrygen's
`Generate`:

```go
func (p *Plugin) Generate(ctx *sdk.GeneratorContext) error {
    c := builder.For(Name, emit.Target{})
    for _, s := range ctx.Reader.Structs().Slice() {
        if !s.HasPositiveDirective(DirectiveName) {
            continue
        }
        reg := &Registration{
            BaseEmit: emit.BaseEmit{
                OriginNode: s,
                SetByName:  c.SetBy(),
                SourcePos:  s.Pos(),
            },
            Name:         s.Name,
            NameLit:      emit.NewLiteralString(s.Name),
            Init:         emit.NewComposite(emit.External(s.Package, s.Name), nil),
            RegisterFunc: emit.NewExternal(p.opts.RegisterPackage, p.opts.RegisterFunc),
        }
        // Append into the file-level init slot — the layout phase
        // routes the slot's host file based on the source struct's
        // origin.
        if err := ctx.Store.Emit().AppendOriginSlot(
            s, "init", reg, c.Provenance("registry."+s.Name),
        ); err != nil {
            return err
        }
    }
    return nil
}
```

`emit.NewExternal` is the key: it produces an `Expr` referencing
an identifier in a specific package, and the backend
automatically registers that package as an import on the
rendered file. The plugin never touches the file's import set
directly.

## Step 5: render

At render time, the backend:

1. Groups every emit entity by its `emit.Target`
2. For each target, calls the canonical template for each entity
   — `render <entity>` dispatches by `Node.Kind()` to the
   appropriately-named template
3. Composes the rendered fragments into one file body, with
   imports resolved and the standard header / footer stamped

For a source struct named `Article`, the registration template
renders as:

```go
log.Print("Article", Article{})
```

The Go backend then groups all registrations for the same file
inside one `func init() { ... }` block (via the file-level
`init` slot), runs `gofmt` + `goimports`, and writes the result
through `ctx.Sink`.

## The funcmap

Templates have access to a funcmap of helper functions. The
core funcmap, exposed by the Go backend, includes:

- **Dispatch helpers** — `render`, `renderType`, `renderStmt`,
  `renderExpr` — route to the appropriate sub-template based on
  the value's kind. Most plugin templates use `renderExpr` to
  render `*emit.Expr` values and `renderType` for type
  references.
- **Slot-composition helpers** — `render<Host><Slot>` (e.g.
  `renderMethodPrebody`, `renderStructFields`) — render the
  contents of a slot on a host entity, including contributions
  from cross-cutting weavers.
- **Render helpers** — `renderParams`, `renderReturns`,
  `renderReceiver` — render canonical param lists and receiver
  forms.
- **Collision helpers** — `imp` (import path → alias), `slot`
  (named slot accessor on a host).
- **Metadata** — `provenance` (the rendered file's content
  hash).

These are **reserved**. A plugin cannot override them; doing so
fails Build with `ErrReservedFuncName`.

Overrideable leaf utilities (case conversion, string operations,
date formatting) sit alongside and are extended / overridden
through the other two methods on `TemplateProvider`:

## Funcmap extensions: `TemplateFuncs`

Returns funcmap entries the plugin contributes. The backend
merges every plugin's returned map at Build time; cross-plugin
name collisions or collisions with the core canonical entries
fail Build with `ErrTemplateFuncCollision`.

```go
func (*Plugin) TemplateFuncs(lang string) template.FuncMap {
    if lang != "golang" {
        return nil
    }
    return template.FuncMap{
        "myco_camelCase": camelCase,
        "myco_snakeCase": snakeCase,
    }
}
```

**Convention**: prefix extension names with your plugin
identifier (`myco_camelCase`, not `camelCase`) to keep
cross-plugin collisions rare. Reserved names (the dispatch
helpers above) are off-limits regardless.

## Funcmap overrides: `TemplateOverrides`

Returns funcmap entries that **intentionally replace**
previously-registered names. The backend records each override
as a diagnostic, naming the winning plugin and the previous
owner, so users can see which plugin's definition won.

```go
func (*Plugin) TemplateOverrides(lang string) template.FuncMap {
    if lang != "golang" {
        return nil
    }
    return template.FuncMap{
        // Replace the project-wide camelCase with our locale-aware
        // variant.
        "camelCase": ourLocaleAwareCamelCase,
    }
}
```

The override pass runs after the extension pass in capability
topological order, so a downstream plugin can replace an
upstream plugin's funcmap entry. Reserved-name overrides still
fail with `ErrReservedFuncName`.

## Anti-patterns

- **Template name doesn't match `Kind()`.** The backend
  dispatches by `Node.Kind()` value as a string; a mismatched
  template name produces an `ErrUnknownKind` at render time.

- **Using `fragment.*` template names.** Reserved for future
  shared partials. Plugin-defined templates using the prefix are
  rejected at parse time.

- **Manually composing imports inside the template.** The
  backend resolves imports from `emit.NewExternal` /
  `emit.NewBuiltin` references on emit entities; templates
  should `renderExpr` against the entity, never embed raw
  package paths.

- **Bare funcmap names without a plugin prefix.** Cross-plugin
  collisions happen the moment two plugins ship the same name.
  Prefixing keeps the collision surface small and makes the
  intent ("I am extending the funcmap with these specific
  utilities") explicit.

- **Re-implementing rendering logic the canonical funcmap
  already provides.** `renderExpr`, `renderType`, `renderStmt`
  cover the common cases. A template that hand-builds Go syntax
  has likely missed a helper.

## Quick reference: registrygen end-to-end

The full picture, in one diff against a fresh project:

1. **Source file**: `reg/article.go`

   ```go
   package reg

   // +gen:register
   type Article struct { ID string }
   ```

2. **Plugin runs**: registrygen sees the `+gen:register`
   directive, appends a `Registration{Name: "Article", ...}` to
   the file-level `init` slot.

3. **Backend renders**: template `registrygen.registration`
   emits `log.Print("Article", Article{})`. The Go backend wraps
   it in a file-level `func init()` block.

4. **Output file**: `reg/article_registry.go`

   ```go
   // Code generated by eidos. DO NOT EDIT.

   package reg

   import "log"

   func init() {
       log.Print("Article", Article{})
   }
   ```

The plugin author wrote ~50 lines of Go + a 3-line template;
the framework handled routing, import resolution, slot
composition, header stamping, and `gofmt` finalisation.

## Conformance

A plugin shipping templates still satisfies the framework
conformance suite via the standard role + options suites
(`RunSuite`, `RunGeneratorSuite`, `RunOptionsSuite`). The
template content is exercised end-to-end through
`pipelinetest` or `backendtest` once your plugin lands in a
project that consumes it; the unit-level conformance suite does
not currently load templates.
