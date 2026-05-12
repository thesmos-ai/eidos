# multipkg fixture

A multi-package testdata fixture for the import-handling and
generics-rendering paths. Its own `go.mod` keeps the fixture out of
the parent `eidos` module's build graph.

## Layout

```
multipkg/
    go.mod                   module example.com/multipkg
    domain/                  core entities + generics (Result, Page, Filter, Box, Numeric)
        types.go               User, Order, OrderItem, Product, ID
        generics.go            Box[T], Result[T], Page[T, C], Map[K, V], Filter[T], Sum[T Numeric]
    events/                  current events surface
        events.go              Event[T], Handler[T], Dispatcher[T]
    legacy/events/           older events surface — same short name `events`
        events.go              LegacyEvent, LegacyDispatcher
    storage/                 generic Repository[T] surface; mocked into storage_test
        repository.go          Repository[T], Query[T]
    api/                     headline cross-package consumer; imports BOTH `events` packages
        handler.go             Handler, Service
    internal/codec/          generic codec primitives, internal-only
        codec.go               Codec[T], PairCodec[K, V]
```

## What this fixture proves

### Same-package import elision

Files generated alongside a source struct (e.g. `domain/user_repo.go`)
land in the same Go package as the source. The renderer must NOT
emit a self-import for references back to types declared in that
package — verified by every `*_repo.go` / `*_builder.go` /
`registry.go` produced under `domain/`.

### Short-name collisions

`example.com/multipkg/events` and `example.com/multipkg/legacy/events`
both declare `package events`. The `api/handler.go` file imports
both, forcing the writer's ImportSet to detect the alias collision
and resolve the second one as `events2` (or whichever suffix the
collision discipline picks). Generated `Handler` mocks must thread
both aliases through the rendered method signatures.

### Cross-package generic instantiations

`storage.Repository[domain.User]`, `events.Event[*domain.User]`, and
`events.Dispatcher[*domain.Order]` reach the renderer with
non-trivial type arguments. `renderType` for `ExternalRef` must
include the bracketed type-arg list with the correct nested
qualifiers, and any same-package elision must apply per-element.

### `+gen:out` filename override

`domain.Product` carries `+gen:out=product_codegen.go`. The router
honours the directive and the builder file lands at that name
rather than the conventional `product_builder.go`.

### Generated test-package output

`storage.Repository[T]` and `events.Handler[T]` are mocked with
mockgen configured as `test: true`. The rendered mock files end in
`_test.go` and declare `package storage_test` / `package
events_test`. References from the test file back to the regular
package's types qualify properly: `storage.Repository[T]` is *not*
elided even though the file lives in the same directory as the
regular `storage` package, because the test package's effective
import identity differs from the regular package's.

## How the acceptance test runs

`go.thesmos.sh/eidos/eidostest/multipkgacceptance` builds the
`cmd/eidos` binary, runs it against this fixture, and asserts:

- every generated file's import block resolves cleanly
- `go build ./...` inside the fixture exits 0 after generation
- the second run is byte-identical to the first (idempotency)
- the `*_test.go` outputs declare the `_test` package and import
  the regular package
- `events2` (or the equivalent suffixed alias) appears in
  `api/handler_mock.go` for the legacy events import
