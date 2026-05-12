// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package pipeline_test

import (
	"errors"
	"testing"

	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/core/position"
	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/pipeline"
	"go.thesmos.sh/eidos/plugin"
	"go.thesmos.sh/eidos/sink"
)

// routerGen is a one-shot generator that adds a pre-built emit
// package to the store. Used by the router tests to seed entities
// with controlled Origin / Target / directive state and then watch
// the router phase resolve them.
type routerGen struct {
	name string
	pkg  *emit.Package
}

func (g *routerGen) Name() string { return g.name }

func (g *routerGen) Generate(ctx *plugin.GeneratorContext) error {
	return ctx.Store.Emit().AddPackage(g.pkg)
}

// TestRouter_AlongsideSource covers the default precedence rule:
// when Target.Dir is empty and Origin.Pos().File is set, the
// router stamps Target.Dir with the directory of the source file.
func TestRouter_AlongsideSource(t *testing.T) {
	t.Parallel()

	t.Run("empty Target.Dir + non-nil Origin → derived from source", func(t *testing.T) {
		t.Parallel()
		origin := &node.Struct{
			BaseNode: node.BaseNode{SourcePos: position.Pos{File: "internal/users/user.go", Line: 10}},
		}
		s := &emit.Struct{
			BaseEmit: emit.BaseEmit{OriginNode: origin},
			Name:     "User", Package: "users",
			Target: emit.Target{Filename: "user_gen.go", Package: "users"},
		}
		p, err := pipeline.New().
			WithFrontend(&stubFE{name: "fe"}).
			WithGenerator(&routerGen{name: "rg", pkg: &emit.Package{
				Name: "users", Path: "users",
				Structs: []*emit.Struct{s},
			}}).
			WithBackend(&stubBE{name: "be"}).
			WithSink(sink.NewMemory()).
			Build()
		assertNoError(t, err)
		assertNoError(t, p.Run(t.Context()))
		if s.Target.Dir != "internal/users" {
			t.Fatalf("Target.Dir = %q, want %q", s.Target.Dir, "internal/users")
		}
	})

	t.Run("plugin Target.Dir wins over source-derived default", func(t *testing.T) {
		t.Parallel()
		origin := &node.Function{
			BaseNode: node.BaseNode{SourcePos: position.Pos{File: "internal/users/user.go"}},
		}
		f := &emit.Function{
			BaseEmit: emit.BaseEmit{OriginNode: origin},
			Name:     "Save", Package: "users",
			Target: emit.Target{Dir: "gen/users", Filename: "save_gen.go", Package: "users"},
		}
		p, err := pipeline.New().
			WithFrontend(&stubFE{name: "fe"}).
			WithGenerator(&routerGen{name: "rg", pkg: &emit.Package{
				Name: "users", Path: "users",
				Functions: []*emit.Function{f},
			}}).
			WithBackend(&stubBE{name: "be"}).
			WithSink(sink.NewMemory()).
			Build()
		assertNoError(t, err)
		assertNoError(t, p.Run(t.Context()))
		if f.Target.Dir != "gen/users" {
			t.Fatalf("Target.Dir = %q, want %q (plugin explicit dir)", f.Target.Dir, "gen/users")
		}
	})
}

// TestRouter_OutDirective covers the +gen:out directive: when the
// originating node carries an `out` directive, the router stamps
// Target.Filename with the directive's positional argument,
// overriding the plugin's default filename.
func TestRouter_OutDirective(t *testing.T) {
	t.Parallel()

	t.Run("out directive on origin overrides Target.Filename", func(t *testing.T) {
		t.Parallel()
		origin := &node.Interface{
			BaseNode: node.BaseNode{
				SourcePos: position.Pos{File: "internal/users/user.go", Line: 30},
				DirectiveList: []*directive.Directive{
					{Name: pipeline.OutDirective, Args: []string{"user_mock_gen.go"}},
				},
			},
		}
		i := &emit.Interface{
			BaseEmit: emit.BaseEmit{OriginNode: origin},
			Name:     "UserRepo", Package: "users",
			Target: emit.Target{Filename: "default_filename.go", Package: "users"},
		}
		p, err := pipeline.New().
			WithFrontend(&stubFE{name: "fe"}).
			WithGenerator(&routerGen{name: "rg", pkg: &emit.Package{
				Name: "users", Path: "users",
				Interfaces: []*emit.Interface{i},
			}}).
			WithBackend(&stubBE{name: "be"}).
			WithSink(sink.NewMemory()).
			Build()
		assertNoError(t, err)
		assertNoError(t, p.Run(t.Context()))
		if i.Target.Filename != "user_mock_gen.go" {
			t.Fatalf("Target.Filename = %q, want %q (out directive override)", i.Target.Filename, "user_mock_gen.go")
		}
		if i.Target.Dir != "internal/users" {
			t.Fatalf("Target.Dir = %q, want %q (source-derived)", i.Target.Dir, "internal/users")
		}
	})
}

// TestRouter_UnroutableEntityErrors covers the failure mode: when
// Target.Dir is empty and Origin is nil (purely synthetic emit
// with no plugin Dir override), the router records an Error
// diagnostic and the run returns ErrRunHadErrors.
func TestRouter_UnroutableEntityErrors(t *testing.T) {
	t.Parallel()

	t.Run("synthetic entity with no Target.Dir and no Origin errors out", func(t *testing.T) {
		t.Parallel()
		v := &emit.Variable{
			Name: "Sentinel", Package: "boot",
			Target: emit.Target{Filename: "sentinel_gen.go", Package: "boot"},
			// No Origin, no Target.Dir → unroutable.
		}
		p, err := pipeline.New().
			WithFrontend(&stubFE{name: "fe"}).
			WithGenerator(&routerGen{name: "rg", pkg: &emit.Package{
				Name: "boot", Path: "boot",
				Variables: []*emit.Variable{v},
			}}).
			WithBackend(&stubBE{name: "be"}).
			WithSink(sink.NewMemory()).
			Build()
		assertNoError(t, err)
		runErr := p.Run(t.Context())
		if !errors.Is(runErr, pipeline.ErrRunHadErrors) {
			t.Fatalf("Run should return ErrRunHadErrors; got %v", runErr)
		}
	})
}

// TestRouter_AliasFileField covers the alias-specific path: aliases
// store their file routing in [emit.Alias.File] rather than the
// Target field (which is the aliased Ref). The router uses the
// same resolution rule.
func TestRouter_AliasFileField(t *testing.T) {
	t.Parallel()

	t.Run("Alias.File is resolved from Origin like other decls", func(t *testing.T) {
		t.Parallel()
		origin := &node.Alias{
			BaseNode: node.BaseNode{SourcePos: position.Pos{File: "internal/types/id.go"}},
		}
		a := &emit.Alias{
			BaseEmit: emit.BaseEmit{OriginNode: origin},
			Name:     "UserID", Package: "types",
			Target:  emit.Builtin("string"),
			IsAlias: true,
			File:    emit.Target{Filename: "user_id_gen.go", Package: "types"},
		}
		p, err := pipeline.New().
			WithFrontend(&stubFE{name: "fe"}).
			WithGenerator(&routerGen{name: "rg", pkg: &emit.Package{
				Name: "types", Path: "types",
				Aliases: []*emit.Alias{a},
			}}).
			WithBackend(&stubBE{name: "be"}).
			WithSink(sink.NewMemory()).
			Build()
		assertNoError(t, err)
		assertNoError(t, p.Run(t.Context()))
		if a.File.Dir != "internal/types" {
			t.Fatalf("Alias.File.Dir = %q, want %q (source-derived)", a.File.Dir, "internal/types")
		}
	})
}

// TestRouter_AllKinds covers the per-kind range loops in
// [runRouter] for the kinds the dedicated single-purpose tests
// above don't already exercise — Constant and Enum — plus a sanity
// check that the rerouted Targets land where the source's
// directory says they should.
func TestRouter_AllKinds(t *testing.T) {
	t.Parallel()

	t.Run("Constant and Enum entities reroute from Origin like other decls", func(t *testing.T) {
		t.Parallel()
		constOrigin := &node.Constant{
			BaseNode: node.BaseNode{SourcePos: position.Pos{File: "internal/x/consts.go"}},
		}
		enumOrigin := &node.Enum{
			BaseNode: node.BaseNode{SourcePos: position.Pos{File: "internal/x/enums.go"}},
		}
		c := &emit.Constant{
			BaseEmit: emit.BaseEmit{OriginNode: constOrigin},
			Name:     "Pi", Package: "x",
			Target: emit.Target{Filename: "consts_gen.go", Package: "x"},
		}
		e := &emit.Enum{
			BaseEmit: emit.BaseEmit{OriginNode: enumOrigin},
			Name:     "Status", Package: "x",
			Target: emit.Target{Filename: "status_gen.go", Package: "x"},
		}
		p, err := pipeline.New().
			WithFrontend(&stubFE{name: "fe"}).
			WithGenerator(&routerGen{name: "rg", pkg: &emit.Package{
				Name: "x", Path: "x",
				Constants: []*emit.Constant{c},
				Enums:     []*emit.Enum{e},
			}}).
			WithBackend(&stubBE{name: "be"}).
			WithSink(sink.NewMemory()).
			Build()
		assertNoError(t, err)
		assertNoError(t, p.Run(t.Context()))
		if c.Target.Dir != "internal/x" {
			t.Fatalf("Constant.Target.Dir = %q, want %q", c.Target.Dir, "internal/x")
		}
		if e.Target.Dir != "internal/x" {
			t.Fatalf("Enum.Target.Dir = %q, want %q", e.Target.Dir, "internal/x")
		}
	})

	t.Run("unroutable Interface emits a router error", func(t *testing.T) {
		t.Parallel()
		i := &emit.Interface{
			Name: "Lost", Package: "boot",
			Target: emit.Target{Filename: "lost.go", Package: "boot"},
		}
		p, err := pipeline.New().
			WithFrontend(&stubFE{name: "fe"}).
			WithGenerator(&routerGen{name: "rg", pkg: &emit.Package{
				Name: "boot", Path: "boot",
				Interfaces: []*emit.Interface{i},
			}}).
			WithBackend(&stubBE{name: "be"}).
			WithSink(sink.NewMemory()).
			Build()
		assertNoError(t, err)
		runErr := p.Run(t.Context())
		if !errors.Is(runErr, pipeline.ErrRunHadErrors) {
			t.Fatalf("Run should return ErrRunHadErrors; got %v", runErr)
		}
	})

	t.Run("unroutable Function / Struct / Constant / Enum / Alias all error", func(t *testing.T) {
		t.Parallel()
		// Bundle every routable kind without an Origin and without an
		// explicit Target.Dir so the router emits an error per kind.
		fn := &emit.Function{Name: "F", Package: "x", Target: emit.Target{Filename: "f.go", Package: "x"}}
		st := &emit.Struct{Name: "S", Package: "x", Target: emit.Target{Filename: "s.go", Package: "x"}}
		c := &emit.Constant{Name: "C", Package: "x", Target: emit.Target{Filename: "c.go", Package: "x"}}
		e := &emit.Enum{Name: "E", Package: "x", Target: emit.Target{Filename: "e.go", Package: "x"}}
		a := &emit.Alias{Name: "A", Package: "x", Target: emit.Builtin("int"), File: emit.Target{Filename: "a.go", Package: "x"}}
		p, err := pipeline.New().
			WithFrontend(&stubFE{name: "fe"}).
			WithGenerator(&routerGen{name: "rg", pkg: &emit.Package{
				Name: "x", Path: "x",
				Functions: []*emit.Function{fn},
				Structs:   []*emit.Struct{st},
				Constants: []*emit.Constant{c},
				Enums:     []*emit.Enum{e},
				Aliases:   []*emit.Alias{a},
			}}).
			WithBackend(&stubBE{name: "be"}).
			WithSink(sink.NewMemory()).
			Build()
		assertNoError(t, err)
		runErr := p.Run(t.Context())
		if !errors.Is(runErr, pipeline.ErrRunHadErrors) {
			t.Fatalf("Run should return ErrRunHadErrors; got %v", runErr)
		}
	})
}

// TestOutDirective_RegisteredInCoreSet covers the core-directive
// registration: every pipeline.Build() ends up with "out" in the
// directive registry, regardless of whether the consumer also
// supplied directives via WithDirective.
func TestOutDirective_RegisteredInCoreSet(t *testing.T) {
	t.Parallel()

	t.Run("out directive registered even with no WithDirective calls", func(t *testing.T) {
		t.Parallel()
		p, err := pipeline.New().
			WithFrontend(&stubFE{name: "fe"}).
			WithBackend(&stubBE{name: "be"}).
			WithSink(sink.NewMemory()).
			Build()
		assertNoError(t, err)
		if _, ok := p.DirectiveRegistry().Lookup(pipeline.OutDirective); !ok {
			t.Fatalf("out directive should be registered in the core set")
		}
	})
}
