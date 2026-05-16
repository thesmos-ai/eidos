# Plugin authoring guide

eidos plugins are the unit of extension. Every code-generation
behaviour — Go-source parsing, repository emission, mock
generation, debug-trace weaving — runs as a plugin against the
framework's shared store and pipeline.

This guide collects the documents a plugin author reads in
order:

1. **[quickstart.md](quickstart.md)** — Write your first plugin
   from scratch in ten minutes. Targets an annotator (the
   simplest role) and walks through every layer the framework
   exposes.

2. **[recipes.md](recipes.md)** — Pattern catalog. For each
   common plugin shape (one struct → one emit, cross-cutting
   slot contribution, plugin-defined emit kind, …) points at a
   working reference plugin and summarises its structure.

3. **[conformance.md](conformance.md)** — Testing your plugin
   against the framework conformance suite. What `plugintest`
   provides, which suite applies to which role, how to write
   fixtures.

4. **[templates.md](templates.md)** — Shipping templates from a
   generator plugin via `sdk.TemplateProvider`. Plugin-defined
   emit kinds, the template naming contract, the funcmap, and
   funcmap extensions / overrides — walked through end-to-end
   with `registrygen` as the canonical example.

## Reference plugins

Every reference plugin in `reference/` is a complete, tested,
production-grade example:

| Plugin                         | Role             | Pattern                            |
|--------------------------------|------------------|------------------------------------|
| [shapewriter](../../reference/shapewriter)   | Annotator        | Infer structural shape; stamp meta |
| [repogen](../../reference/repogen)           | Generator        | Per-source-struct emit (CRUD repo) |
| [buildergen](../../reference/buildergen)     | Generator        | Per-source-struct emit (builder)   |
| [mockgen](../../reference/mockgen)           | Generator        | Per-source-interface emit (mock)   |
| [auditweaver](../../reference/auditweaver)   | Cross-cutting    | Method prebody-slot contribution   |
| [debugweaver](../../reference/debugweaver)   | Cross-cutting    | Method prebody-slot contribution   |
| [registrygen](../../reference/registrygen)   | Cross-cutting    | Plugin-defined emit kind + init() registration |

Read the reference plugin matching your intended pattern before
writing your own — every framework idiom appears in at least one
of them.

## The SDK façade

The [`sdk` package](../../sdk) re-exports the plugin contract
surface (roles, capabilities, hooks, priority buckets, directive
schema builders) under one import. A typical plugin's imports
shrink from eight to four:

```go
import (
    "go.thesmos.sh/eidos/sdk"          // role + capability contracts
    "go.thesmos.sh/eidos/core/opt"     // typed options (when applicable)
    "go.thesmos.sh/eidos/node"         // source-side store (read)
    "go.thesmos.sh/eidos/emit"         // emit-side store (generators / backends)
    "go.thesmos.sh/eidos/emit/builder" // emit construction helpers
)
```

The high-volume packages (`node`, `emit`, `emit/builder`,
`core/opt`) stay as separate imports — flattening them into
`sdk` would clash on common names like `Schema`, `Field`,
`Walk`, and `New`.
