# demoproject

A canonical Go testdata fixture consumed by demonstration plugins
under `plugins/`. The fixture is a self-contained nested module
(its own `go.mod`) so its sources never join the parent `eidos`
module's build graph.

## Layout

```
demoproject/
    go.mod          # module example.com/demoproject
    blog/           # primary fixture package
        article.go     # Article — repo + builder + register directives
        user.go        # User — repo + builder directives
        comment.go     # Comment — builder directive; generic field
        status.go      # Status enum (typed iota)
        writer.go      # LineWriter — io.Writer shape + embedded io.Closer
        searcher.go    # Searcher interface — mock directive
        score.go       # Score generic — type-set union constraint
        errors.go      # package-level error sentinels
    extras/         # secondary package referenced via external import
        uuid.go        # UUID alias [16]byte
```

## Directive coverage

| Source entity | Directives | Purpose |
|---|---|---|
| `blog.Article` | `+gen:repo`, `+gen:builder`, `+gen:register` | All three demonstration generators target this type; the multi-generator file-composition path is exercised against it. |
| `blog.User` | `+gen:repo`, `+gen:builder` | Repo + builder without registry — verifies the directive-driven opt-in is independent per generator. |
| `blog.Comment` | `+gen:builder` | Builder-only target; carries a generic field forcing the builder generator to render the type parameter intact. |
| `blog.LineWriter` | (none — heuristic detection) | The shape detector reaches LineWriter through signature matching alone; negative-override tests against this type verify the directive suppression path. |
| `blog.Searcher` | `+gen:mock` | User-authored interface flagged for mocking — verifies the mock generator targets interfaces beyond the ones repogen synthesises. |

## Rendering-surface coverage

The fixture exercises every renderable shape the backend learned
to handle:

- Pointer / slice / map / array / func composite types (`Article` fields, `Score` value type).
- Generics with `any` and type-set constraints (`Box[T any]`, `Score[T Numeric]` where `Numeric = ~int | ~float64`).
- Typed iota enums (`Status`).
- Named-return method (`Article.Validate() (err error)`).
- Anonymous-return method (`Article.String() string`).
- Embedded interface (`LineWriter` embeds `io.Closer`).
- Multi-method interface (`Searcher`).
- Cross-package external import (`blog.Article.ID` references `extras.UUID`).

## Module isolation

Because the fixture declares its own `go.mod`, `go build ./...` from
the eidos repo root never compiles the fixture as part of the
parent build. Tests that load the fixture do so through the
`eidostest/demopipe` harness, which points the Go frontend at
this directory through the frontend's `Dir` option.
