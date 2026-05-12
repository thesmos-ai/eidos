// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package pipeline_test

import (
	"errors"
	"testing"

	"go.thesmos.sh/eidos/core/diag"
	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/core/position"
	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/pipeline"
	"go.thesmos.sh/eidos/plugin"
	"go.thesmos.sh/eidos/sink"
)

// TestLayout_AlongsideSource exercises the framework default: a
// decl with a non-nil Origin and a plugin-declared filename suffix
// resolves to alongside-source — Target.Dir derived from the
// source file's directory and Target.Filename composed from the
// source basename + suffix.
func TestLayout_AlongsideSource(t *testing.T) {
	t.Parallel()

	t.Run("Dir + Filename derived from origin Pos.File + plugin suffix", func(t *testing.T) {
		t.Parallel()
		origin := &node.Struct{
			BaseNode: node.BaseNode{
				SourcePos: position.Pos{File: "internal/users/user.go", Line: 10},
			},
			Name: "User", Package: "example.com/users",
		}
		s := &emit.Struct{
			BaseEmit: emit.BaseEmit{OriginNode: origin},
			Name:     "UserRepo", Package: "users",
		}
		p, err := pipeline.New().
			WithFrontend(&stubFE{name: "fe"}).
			WithGenerator(&layoutGen{name: "rg", suffix: "_repo.go", pkg: &emit.Package{
				Name: "users", Path: "example.com/users",
				Structs: []*emit.Struct{s},
			}}).
			WithBackend(&stubBE{name: "be"}).
			WithSink(sink.NewMemory()).
			Build()
		assertNoError(t, err)
		assertNoError(t, p.Run(t.Context(), "x"))
		if got, want := s.Target.Dir, "internal/users"; got != want {
			t.Fatalf("Target.Dir = %q, want %q", got, want)
		}
		if got, want := s.Target.Filename, "user_repo.go"; got != want {
			t.Fatalf("Target.Filename = %q, want %q", got, want)
		}
	})

	t.Run("Package + ImportPath derived from origin node.Package lookup", func(t *testing.T) {
		t.Parallel()
		nodePkg := &node.Package{Name: "users", Path: "example.com/users"}
		origin := &node.Struct{
			BaseNode: node.BaseNode{SourcePos: position.Pos{File: "internal/users/user.go"}},
			Name:     "User", Package: "example.com/users",
		}
		s := &emit.Struct{
			BaseEmit: emit.BaseEmit{OriginNode: origin},
			Name:     "UserRepo", Package: "ignored",
		}
		p, err := pipeline.New().
			WithFrontend(&nodePackageFE{name: "fe", pkg: nodePkg}).
			WithGenerator(&layoutGen{name: "rg", suffix: "_repo.go", pkg: &emit.Package{
				Name: "users", Path: "example.com/users",
				Structs: []*emit.Struct{s},
			}}).
			WithBackend(&stubBE{name: "be"}).
			WithSink(sink.NewMemory()).
			Build()
		assertNoError(t, err)
		assertNoError(t, p.Run(t.Context(), "x"))
		if got, want := s.Target.Package, "users"; got != want {
			t.Fatalf("Target.Package = %q, want %q", got, want)
		}
		if got, want := s.Target.ImportPath, "example.com/users"; got != want {
			t.Fatalf("Target.ImportPath = %q, want %q", got, want)
		}
	})
}

// TestLayout_OutDirective verifies the +gen:out directive on an
// origin overrides the plugin-suffix-derived Filename — precedence
// layer 5.
func TestLayout_OutDirective(t *testing.T) {
	t.Parallel()

	t.Run("out directive on origin overrides Filename", func(t *testing.T) {
		t.Parallel()
		origin := &node.Interface{
			BaseNode: node.BaseNode{
				SourcePos: position.Pos{File: "internal/users/user.go"},
				DirectiveList: []*directive.Directive{
					{Name: pipeline.OutDirective, Args: []string{"user_mock_gen.go"}},
				},
			},
			Name: "UserRepo", Package: "example.com/users",
		}
		i := &emit.Interface{
			BaseEmit: emit.BaseEmit{OriginNode: origin},
			Name:     "UserRepoMock", Package: "users",
		}
		p, err := pipeline.New().
			WithFrontend(&stubFE{name: "fe"}).
			WithGenerator(&layoutGen{name: "mg", suffix: "_mock.go", pkg: &emit.Package{
				Name: "users", Path: "example.com/users",
				Interfaces: []*emit.Interface{i},
			}}).
			WithBackend(&stubBE{name: "be"}).
			WithSink(sink.NewMemory()).
			Build()
		assertNoError(t, err)
		assertNoError(t, p.Run(t.Context(), "x"))
		if got, want := i.Target.Filename, "user_mock_gen.go"; got != want {
			t.Fatalf("Target.Filename = %q, want %q (out directive)", got, want)
		}
		if got, want := i.Target.Dir, "internal/users"; got != want {
			t.Fatalf("Target.Dir = %q, want %q (alongside-source)", got, want)
		}
	})
}

// TestLayout_OutputFilenameOverride verifies the CLI -o override
// (precedence layer 6) wins over the +gen:out directive (layer 5)
// for Target.Filename.
func TestLayout_OutputFilenameOverride(t *testing.T) {
	t.Parallel()

	t.Run("WithOutputFilename pins Target.Filename, overriding +gen:out", func(t *testing.T) {
		t.Parallel()
		origin := &node.Struct{
			BaseNode: node.BaseNode{
				SourcePos: position.Pos{File: "internal/users/user.go"},
				DirectiveList: []*directive.Directive{
					{Name: pipeline.OutDirective, Args: []string{"directive.go"}},
				},
			},
			Name: "User", Package: "example.com/users",
		}
		s := &emit.Struct{
			BaseEmit: emit.BaseEmit{OriginNode: origin},
			Name:     "UserRepo", Package: "users",
		}
		p, err := pipeline.New().
			WithFrontend(&stubFE{name: "fe"}).
			WithGenerator(&layoutGen{name: "rg", suffix: "_repo.go", pkg: &emit.Package{
				Name: "users", Path: "example.com/users",
				Structs: []*emit.Struct{s},
			}}).
			WithBackend(&stubBE{name: "be"}).
			WithSink(sink.NewMemory()).
			WithOutputFilename("cli_pinned.go").
			Build()
		assertNoError(t, err)
		assertNoError(t, p.Run(t.Context(), "x"))
		if got, want := s.Target.Filename, "cli_pinned.go"; got != want {
			t.Fatalf("Target.Filename = %q, want %q (CLI -o wins)", got, want)
		}
	})
}

// TestLayout_OutputPackageOverride verifies CLI -p
// ([Builder.WithOutputPackage]) pins Target.Package under
// alongside-source layout while leaving Target.Dir derived from
// origin — precedence layer 6 applied to Package only.
func TestLayout_OutputPackageOverride(t *testing.T) {
	t.Parallel()

	t.Run("WithOutputPackage pins Target.Package alongside-source", func(t *testing.T) {
		t.Parallel()
		nodePkg := &node.Package{Name: "users", Path: "example.com/users"}
		origin := &node.Struct{
			BaseNode: node.BaseNode{SourcePos: position.Pos{File: "internal/users/user.go"}},
			Name:     "User", Package: "example.com/users",
		}
		s := &emit.Struct{
			BaseEmit: emit.BaseEmit{OriginNode: origin},
			Name:     "UserRepo", Package: "ignored",
		}
		p, err := pipeline.New().
			WithFrontend(&nodePackageFE{name: "fe", pkg: nodePkg}).
			WithGenerator(&layoutGen{name: "rg", suffix: "_repo.go", pkg: &emit.Package{
				Name: "users", Path: "example.com/users",
				Structs: []*emit.Struct{s},
			}}).
			WithBackend(&stubBE{name: "be"}).
			WithSink(sink.NewMemory()).
			WithOutputPackage("generated").
			Build()
		assertNoError(t, err)
		assertNoError(t, p.Run(t.Context(), "x"))
		if got, want := s.Target.Package, "generated"; got != want {
			t.Fatalf("Target.Package = %q, want %q (-p override)", got, want)
		}
		if got, want := s.Target.Dir, "internal/users"; got != want {
			t.Fatalf("Target.Dir = %q, want %q (alongside-source preserved)", got, want)
		}
	})
}

// TestLayout_Centralised verifies [pipeline.LayoutCentralised]
// routes Dir + Package from the resolved policy rather than the
// origin's source location.
func TestLayout_Centralised(t *testing.T) {
	t.Parallel()

	t.Run(
		"centralised layout uses policy.Package for Dir + Package when Dir empty",
		func(t *testing.T) {
			t.Parallel()
			origin := &node.Struct{
				BaseNode: node.BaseNode{SourcePos: position.Pos{File: "internal/users/user.go"}},
				Name:     "User", Package: "example.com/users",
			}
			s := &emit.Struct{
				BaseEmit: emit.BaseEmit{OriginNode: origin},
				Name:     "UserRepo", Package: "ignored",
			}
			p, err := pipeline.New().
				WithFrontend(&stubFE{name: "fe"}).
				WithGenerator(&layoutGen{name: "rg", suffix: "_repo.go", pkg: &emit.Package{
					Name: "users", Path: "example.com/users",
					Structs: []*emit.Struct{s},
				}}).
				WithBackend(&stubBE{name: "be"}).
				WithSink(sink.NewMemory()).
				WithOutputLayout(pipeline.LayoutCentralised).
				WithOutputPackage("gen").
				Build()
			assertNoError(t, err)
			assertNoError(t, p.Run(t.Context(), "x"))
			if got, want := s.Target.Dir, "gen"; got != want {
				t.Fatalf(
					"Target.Dir = %q, want %q (centralised uses policy.Package as Dir)",
					got,
					want,
				)
			}
			if got, want := s.Target.Package, "gen"; got != want {
				t.Fatalf("Target.Package = %q, want %q", got, want)
			}
		},
	)

	t.Run("centralised layout with explicit WithOutputDir uses it for Dir", func(t *testing.T) {
		t.Parallel()
		origin := &node.Struct{
			BaseNode: node.BaseNode{SourcePos: position.Pos{File: "internal/users/user.go"}},
			Name:     "User", Package: "example.com/users",
		}
		s := &emit.Struct{
			BaseEmit: emit.BaseEmit{OriginNode: origin},
			Name:     "UserRepo", Package: "ignored",
		}
		p, err := pipeline.New().
			WithFrontend(&stubFE{name: "fe"}).
			WithGenerator(&layoutGen{name: "rg", suffix: "_repo.go", pkg: &emit.Package{
				Name: "users", Path: "example.com/users",
				Structs: []*emit.Struct{s},
			}}).
			WithBackend(&stubBE{name: "be"}).
			WithSink(sink.NewMemory()).
			WithOutputLayout(pipeline.LayoutCentralised).
			WithOutputPackage("gen").
			WithOutputDir("internal/gen").
			Build()
		assertNoError(t, err)
		assertNoError(t, p.Run(t.Context(), "x"))
		if got, want := s.Target.Dir, "internal/gen"; got != want {
			t.Fatalf("Target.Dir = %q, want %q (-output-dir override)", got, want)
		}
		if got, want := s.Target.Package, "gen"; got != want {
			t.Fatalf("Target.Package = %q, want %q", got, want)
		}
	})
}

// TestLayout_SyntheticDecl pins the routing error a decl with nil
// Origin produces. The Layout phase records an Error diagnostic
// and clears the decl's Target so the backend skips it; the run
// returns [pipeline.ErrRunHadErrors].
func TestLayout_SyntheticDecl(t *testing.T) {
	t.Parallel()

	t.Run("nil Origin surfaces a routing error and clears Target", func(t *testing.T) {
		t.Parallel()
		v := &emit.Variable{
			Name: "Sentinel", Package: "boot",
		}
		d := diag.New()
		p, err := pipeline.New().
			WithFrontend(&stubFE{name: "fe"}).
			WithGenerator(&layoutGen{name: "rg", suffix: "_gen.go", pkg: &emit.Package{
				Name: "boot", Path: "boot",
				Variables: []*emit.Variable{v},
			}}).
			WithBackend(&stubBE{name: "be"}).
			WithSink(sink.NewMemory()).
			WithDiag(d).
			Build()
		assertNoError(t, err)
		runErr := p.Run(t.Context(), "x")
		if !errors.Is(runErr, pipeline.ErrRunHadErrors) {
			t.Fatalf("Run = %v, want ErrRunHadErrors", runErr)
		}
		if v.Target != (emit.Target{}) {
			t.Fatalf("Target = %+v, want zero (synthetic decl)", v.Target)
		}
		if !hasDiagContaining(d, "synthetic variable") {
			t.Fatalf("expected synthetic-decl diagnostic; got %+v", d.Diagnostics())
		}
	})
}

// TestLayout_MissingFilenameProvider pins the routing error a decl
// emitted by a plugin without [plugin.FilenameProvider] produces.
// The sentinel [pipeline.ErrMissingFilenameProvider] appears in the
// diagnostic message; the offending decl's Target is cleared.
func TestLayout_MissingFilenameProvider(t *testing.T) {
	t.Parallel()

	t.Run("plugin without FilenameProvider surfaces the typed sentinel", func(t *testing.T) {
		t.Parallel()
		origin := &node.Struct{
			BaseNode: node.BaseNode{SourcePos: position.Pos{File: "internal/users/user.go"}},
			Name:     "User", Package: "example.com/users",
		}
		s := &emit.Struct{
			BaseEmit: emit.BaseEmit{OriginNode: origin},
			Name:     "UserRepo", Package: "users",
		}
		d := diag.New()
		p, err := pipeline.New().
			WithFrontend(&stubFE{name: "fe"}).
			WithGenerator(&layoutGenNoSuffix{name: "rg", pkg: &emit.Package{
				Name: "users", Path: "example.com/users",
				Structs: []*emit.Struct{s},
			}}).
			WithBackend(&stubBE{name: "be"}).
			WithSink(sink.NewMemory()).
			WithDiag(d).
			Build()
		assertNoError(t, err)
		runErr := p.Run(t.Context(), "x")
		if !errors.Is(runErr, pipeline.ErrRunHadErrors) {
			t.Fatalf("Run = %v, want ErrRunHadErrors", runErr)
		}
		if s.Target != (emit.Target{}) {
			t.Fatalf("Target = %+v, want zero (missing FilenameProvider)", s.Target)
		}
		if !hasDiagContaining(d, pipeline.ErrMissingFilenameProvider.Error()) {
			t.Fatalf("expected ErrMissingFilenameProvider diagnostic; got %+v", d.Diagnostics())
		}
	})
}

// TestLayout_AllKinds pins per-kind dispatch — each routable emit
// kind is rerouted via its own dispatch arm. Synthesised origins
// per kind have their basename joined with the plugin's suffix.
func TestLayout_AllKinds(t *testing.T) {
	t.Parallel()

	t.Run(
		"Struct, Interface, Function, Variable, Constant, Enum route via Origin",
		func(t *testing.T) {
			t.Parallel()
			mkOrigin := func(file string) node.Node {
				return &node.Struct{
					BaseNode: node.BaseNode{SourcePos: position.Pos{File: file}},
					Package:  "example.com/x",
				}
			}
			st := &emit.Struct{
				BaseEmit: emit.BaseEmit{OriginNode: mkOrigin("internal/x/types.go")},
				Name:     "S",
				Package:  "x",
			}
			i := &emit.Interface{
				BaseEmit: emit.BaseEmit{OriginNode: mkOrigin("internal/x/types.go")},
				Name:     "I",
				Package:  "x",
			}
			fn := &emit.Function{
				BaseEmit: emit.BaseEmit{OriginNode: mkOrigin("internal/x/fns.go")},
				Name:     "F",
				Package:  "x",
			}
			vd := &emit.Variable{
				BaseEmit: emit.BaseEmit{OriginNode: mkOrigin("internal/x/vars.go")},
				Name:     "V",
				Package:  "x",
			}
			c := &emit.Constant{
				BaseEmit: emit.BaseEmit{OriginNode: mkOrigin("internal/x/consts.go")},
				Name:     "C",
				Package:  "x",
			}
			e := &emit.Enum{
				BaseEmit: emit.BaseEmit{OriginNode: mkOrigin("internal/x/enums.go")},
				Name:     "E",
				Package:  "x",
			}
			p, err := pipeline.New().
				WithFrontend(&stubFE{name: "fe"}).
				WithGenerator(&layoutGen{name: "rg", suffix: "_gen.go", pkg: &emit.Package{
					Name: "x", Path: "example.com/x",
					Structs: []*emit.Struct{st}, Interfaces: []*emit.Interface{i},
					Functions: []*emit.Function{fn}, Variables: []*emit.Variable{vd},
					Constants: []*emit.Constant{c}, Enums: []*emit.Enum{e},
				}}).
				WithBackend(&stubBE{name: "be"}).
				WithSink(sink.NewMemory()).
				Build()
			assertNoError(t, err)
			assertNoError(t, p.Run(t.Context(), "x"))
			cases := []struct {
				kind string
				got  emit.Target
				want string
			}{
				{"Struct", st.Target, "internal/x/types_gen.go"},
				{"Interface", i.Target, "internal/x/types_gen.go"},
				{"Function", fn.Target, "internal/x/fns_gen.go"},
				{"Variable", vd.Target, "internal/x/vars_gen.go"},
				{"Constant", c.Target, "internal/x/consts_gen.go"},
				{"Enum", e.Target, "internal/x/enums_gen.go"},
			}
			for _, tc := range cases {
				if got := tc.got.JoinPath(); got != tc.want {
					t.Errorf("%s routed to %q, want %q", tc.kind, got, tc.want)
				}
			}
		},
	)
}

// TestLayout_AliasFileField pins the alias-specific dispatch:
// [emit.Alias] stores its file Target in the File field, not the
// Target field (which holds the aliased type Ref).
func TestLayout_AliasFileField(t *testing.T) {
	t.Parallel()

	t.Run("Alias.File is resolved from Origin", func(t *testing.T) {
		t.Parallel()
		origin := &node.Alias{
			BaseNode: node.BaseNode{SourcePos: position.Pos{File: "internal/types/id.go"}},
			Name:     "UserID", Package: "example.com/types",
		}
		a := &emit.Alias{
			BaseEmit: emit.BaseEmit{OriginNode: origin},
			Name:     "UserID", Package: "types",
			Target: emit.Builtin("string"), IsAlias: true,
		}
		p, err := pipeline.New().
			WithFrontend(&stubFE{name: "fe"}).
			WithGenerator(&layoutGen{name: "rg", suffix: "_alias.go", pkg: &emit.Package{
				Name: "types", Path: "example.com/types",
				Aliases: []*emit.Alias{a},
			}}).
			WithBackend(&stubBE{name: "be"}).
			WithSink(sink.NewMemory()).
			Build()
		assertNoError(t, err)
		assertNoError(t, p.Run(t.Context(), "x"))
		if got, want := a.File.JoinPath(), "internal/types/id_alias.go"; got != want {
			t.Fatalf("Alias.File = %q, want %q", got, want)
		}
	})
}

// TestLayout_PendingOriginSlots pins the slot-materialisation pass:
// an origin-anchored slot contribution lands in the resolved File's
// named slot. The File is created via [store.EmitView.FileFor] when
// absent, so the slot lookup after Run finds the materialised
// contribution.
func TestLayout_PendingOriginSlots(t *testing.T) {
	t.Parallel()

	t.Run("AppendOriginSlot tuple lands in resolved File's named slot", func(t *testing.T) {
		t.Parallel()
		origin := &node.Struct{
			BaseNode: node.BaseNode{SourcePos: position.Pos{File: "internal/users/user.go"}},
			Name:     "User", Package: "example.com/users",
		}
		item := &emit.Constant{Name: "Init", Package: "users"}
		gen := &slotContributingGen{
			name:   "rg",
			suffix: "_meta.go",
			contribute: func(ctx *plugin.GeneratorContext) error {
				return ctx.Store.Emit().
					AppendOriginSlot(origin, "init", item, emit.Provenance{SetBy: "rg"})
			},
		}
		p, err := pipeline.New().
			WithFrontend(&stubFE{name: "fe"}).
			WithGenerator(gen).
			WithBackend(&stubBE{name: "be"}).
			WithSink(sink.NewMemory()).
			Build()
		assertNoError(t, err)
		assertNoError(t, p.Run(t.Context(), "x"))
		f, ok := p.Store().Emit().Files().ByQName("internal/users/user_meta.go")
		if !ok {
			t.Fatalf("expected File at internal/users/user_meta.go; have %v",
				p.Store().Emit().Files().Items())
		}
		slot := f.Slot("init")
		if got := len(slot.Items); got != 1 {
			t.Fatalf("init slot items = %d, want 1", got)
		}
	})
}

// TestLayout_OneFileOnePackage pins the invariant enforcer: two
// decls routing to the same (Dir, Filename) with conflicting
// Package values surface an error and both decls have their
// Target cleared so the backend skips them.
func TestLayout_OneFileOnePackage(t *testing.T) {
	t.Parallel()

	t.Run(
		"conflicting Package values clear both Targets and surface a diagnostic",
		func(t *testing.T) {
			t.Parallel()
			// Two distinct origins in two distinct source packages.
			// Both carry the same +gen:out directive so the Layout
			// phase composes the same Filename for each; the source
			// directory is the same too (the directive's directory
			// is implicit from origin Pos), so (Dir, Filename) align.
			// Their owning source packages differ — pkgX vs pkgY —
			// so the Layout phase composes divergent Target.Package
			// values under alongside-source mode and the
			// one-file-one-package invariant fires.
			other := &node.Struct{
				BaseNode: node.BaseNode{
					SourcePos: position.Pos{File: "internal/x/x.go"},
					DirectiveList: []*directive.Directive{
						{Name: pipeline.OutDirective, Args: []string{"shared.go"}},
					},
				},
				Name: "Other", Package: "example.com/other",
			}
			altOrigin := &node.Struct{
				BaseNode: node.BaseNode{
					SourcePos: position.Pos{File: "internal/x/x.go"},
					DirectiveList: []*directive.Directive{
						{Name: pipeline.OutDirective, Args: []string{"shared.go"}},
					},
				},
				Name: "Alt", Package: "example.com/x",
			}
			// Two node.Packages so the alongside-source Package field
			// resolves to different values for the two origins.
			pkgX := &node.Package{Name: "x", Path: "example.com/x"}
			pkgY := &node.Package{Name: "y", Path: "example.com/y"}
			altOrigin.Package = "example.com/y"
			s1 := &emit.Struct{BaseEmit: emit.BaseEmit{OriginNode: other}, Name: "S1", Package: "x"}
			s2 := &emit.Struct{
				BaseEmit: emit.BaseEmit{OriginNode: altOrigin},
				Name:     "S2",
				Package:  "y",
			}
			d := diag.New()
			p, err := pipeline.New().
				WithFrontend(&multiNodePackageFE{name: "fe", pkgs: []*node.Package{pkgX, pkgY}}).
				WithGenerator(&layoutGen{name: "rg", suffix: "_gen.go", pkg: &emit.Package{
					Name: "x", Path: "example.com/x",
					Structs: []*emit.Struct{s1, s2},
				}}).
				WithBackend(&stubBE{name: "be"}).
				WithSink(sink.NewMemory()).
				WithDiag(d).
				Build()
			assertNoError(t, err)
			runErr := p.Run(t.Context(), "x")
			if !errors.Is(runErr, pipeline.ErrRunHadErrors) {
				t.Fatalf("Run = %v, want ErrRunHadErrors", runErr)
			}
			if s1.Target != (emit.Target{}) || s2.Target != (emit.Target{}) {
				t.Fatalf("Targets not cleared: s1=%+v s2=%+v", s1.Target, s2.Target)
			}
			if !hasDiagContaining(d, "one-file-one-package") {
				t.Fatalf("expected one-file-one-package diagnostic; got %+v", d.Diagnostics())
			}
		},
	)
}

// TestLayout_OneFileOnePackage_AllKinds extends the conflict path
// across every routable emit kind. One decl per kind from one
// source package collides with another decl-per-kind from a
// second source package on the same (Dir, Filename); each kind's
// arm in [clearConflictedTargets] runs and zeroes its Target /
// File field.
func TestLayout_OneFileOnePackage_AllKinds(t *testing.T) {
	t.Parallel()

	t.Run("every routable kind's Target is cleared when packages conflict", func(t *testing.T) {
		t.Parallel()
		mkOrigin := func(name, pkgPath string) node.Node {
			return &node.Struct{
				BaseNode: node.BaseNode{
					SourcePos: position.Pos{File: "x/x.go"},
					DirectiveList: []*directive.Directive{
						{Name: pipeline.OutDirective, Args: []string{"shared.go"}},
					},
				},
				Name: name, Package: pkgPath,
			}
		}
		oA := mkOrigin("A", "example.com/a")
		oB := mkOrigin("B", "example.com/b")
		pkgA := &node.Package{Name: "a", Path: "example.com/a"}
		pkgB := &node.Package{Name: "b", Path: "example.com/b"}
		// Two decls per kind: one rooted in pkgA, one in pkgB. Their
		// composed (Dir, Filename) is (x, shared.go); their composed
		// Package values are "a" vs "b" — the spec-violating
		// conflict the invariant fires on.
		mk := func() (*emit.Struct, *emit.Struct) {
			return &emit.Struct{
					BaseEmit: emit.BaseEmit{OriginNode: oA},
					Name:     "S_A", Package: "a",
				},
				&emit.Struct{
					BaseEmit: emit.BaseEmit{OriginNode: oB},
					Name:     "S_B", Package: "b",
				}
		}
		_ = mk
		structA := &emit.Struct{BaseEmit: emit.BaseEmit{OriginNode: oA}, Name: "S_A", Package: "a"}
		structB := &emit.Struct{BaseEmit: emit.BaseEmit{OriginNode: oB}, Name: "S_B", Package: "b"}
		ifA := &emit.Interface{BaseEmit: emit.BaseEmit{OriginNode: oA}, Name: "I_A", Package: "a"}
		ifB := &emit.Interface{BaseEmit: emit.BaseEmit{OriginNode: oB}, Name: "I_B", Package: "b"}
		fnA := &emit.Function{BaseEmit: emit.BaseEmit{OriginNode: oA}, Name: "F_A", Package: "a"}
		fnB := &emit.Function{BaseEmit: emit.BaseEmit{OriginNode: oB}, Name: "F_B", Package: "b"}
		varA := &emit.Variable{BaseEmit: emit.BaseEmit{OriginNode: oA}, Name: "V_A", Package: "a"}
		varB := &emit.Variable{BaseEmit: emit.BaseEmit{OriginNode: oB}, Name: "V_B", Package: "b"}
		cA := &emit.Constant{BaseEmit: emit.BaseEmit{OriginNode: oA}, Name: "C_A", Package: "a"}
		cB := &emit.Constant{BaseEmit: emit.BaseEmit{OriginNode: oB}, Name: "C_B", Package: "b"}
		eA := &emit.Enum{BaseEmit: emit.BaseEmit{OriginNode: oA}, Name: "E_A", Package: "a"}
		eB := &emit.Enum{BaseEmit: emit.BaseEmit{OriginNode: oB}, Name: "E_B", Package: "b"}
		aA := &emit.Alias{
			BaseEmit: emit.BaseEmit{OriginNode: oA},
			Name:     "AL_A", Package: "a",
			Target: emit.Builtin("string"), IsAlias: true,
		}
		aB := &emit.Alias{
			BaseEmit: emit.BaseEmit{OriginNode: oB},
			Name:     "AL_B", Package: "b",
			Target: emit.Builtin("string"), IsAlias: true,
		}
		d := diag.New()
		p, err := pipeline.New().
			WithFrontend(&multiNodePackageFE{name: "fe", pkgs: []*node.Package{pkgA, pkgB}}).
			WithGenerator(&layoutGen{name: "rg", suffix: "_gen.go", pkg: &emit.Package{
				Name: "x", Path: "example.com/x",
				Structs:    []*emit.Struct{structA, structB},
				Interfaces: []*emit.Interface{ifA, ifB},
				Functions:  []*emit.Function{fnA, fnB},
				Variables:  []*emit.Variable{varA, varB},
				Constants:  []*emit.Constant{cA, cB},
				Enums:      []*emit.Enum{eA, eB},
				Aliases:    []*emit.Alias{aA, aB},
			}}).
			WithBackend(&stubBE{name: "be"}).
			WithSink(sink.NewMemory()).
			WithDiag(d).
			Build()
		assertNoError(t, err)
		runErr := p.Run(t.Context(), "x")
		if !errors.Is(runErr, pipeline.ErrRunHadErrors) {
			t.Fatalf("Run = %v, want ErrRunHadErrors", runErr)
		}
		// Every kind's pair must be cleared by the invariant enforcer.
		cases := []struct {
			name    string
			cleared emit.Target
		}{
			{"struct_A", structA.Target},
			{"struct_B", structB.Target},
			{"interface_A", ifA.Target},
			{"interface_B", ifB.Target},
			{"function_A", fnA.Target},
			{"function_B", fnB.Target},
			{"variable_A", varA.Target},
			{"variable_B", varB.Target},
			{"constant_A", cA.Target},
			{"constant_B", cB.Target},
			{"enum_A", eA.Target},
			{"enum_B", eB.Target},
			{"alias_A", aA.File},
			{"alias_B", aB.File},
		}
		for _, tc := range cases {
			if tc.cleared != (emit.Target{}) {
				t.Errorf("%s Target not cleared: %+v", tc.name, tc.cleared)
			}
		}
	})
}

// TestLayout_OriginKinds covers every owner-walked node kind
// [originPackagePath] handles: Method and Field walk Owner chains
// up to a packaged ancestor; EnumVariant terminates at its
// *Enum owner; File terminates at its *Package owner; Package
// itself is its own answer. The test exercises each through the
// origin-anchored slot path so a varied Origin reaches
// [composeTarget] for every kind.
func TestLayout_OriginKinds(t *testing.T) {
	t.Parallel()

	mkPkg := func() *node.Package {
		pkg := &node.Package{Name: "x", Path: "example.com/x"}
		owner := &node.Struct{
			BaseNode: node.BaseNode{SourcePos: position.Pos{File: "x/x.go"}},
			Name:     "Host", Package: "example.com/x",
		}
		owner.Methods = []*node.Method{
			{
				Name:     "M",
				Owner:    owner,
				BaseNode: node.BaseNode{SourcePos: position.Pos{File: "x/x.go"}},
			},
		}
		owner.Fields = []*node.Field{
			{
				Name:     "F",
				Owner:    owner,
				BaseNode: node.BaseNode{SourcePos: position.Pos{File: "x/x.go"}},
			},
		}
		enum := &node.Enum{
			BaseNode: node.BaseNode{SourcePos: position.Pos{File: "x/x.go"}},
			Name:     "E", Package: "example.com/x",
		}
		enum.Variants = []*node.EnumVariant{
			{
				Name:     "V",
				Owner:    enum,
				BaseNode: node.BaseNode{SourcePos: position.Pos{File: "x/x.go"}},
			},
		}
		file := &node.File{
			BaseNode: node.BaseNode{SourcePos: position.Pos{File: "x/x.go"}},
			Name:     "x.go", Path: "x/x.go", Owner: pkg,
		}
		pkg.Structs = []*node.Struct{owner}
		pkg.Enums = []*node.Enum{enum}
		pkg.Files = []*node.File{file}
		return pkg
	}

	t.Run("Method origin resolves package via Owner chain", func(t *testing.T) {
		t.Parallel()
		pkg := mkPkg()
		method := pkg.Structs[0].Methods[0]
		runOriginKind(t, pkg, method, "method")
	})
	t.Run("Field origin resolves package via Owner chain", func(t *testing.T) {
		t.Parallel()
		pkg := mkPkg()
		field := pkg.Structs[0].Fields[0]
		runOriginKind(t, pkg, field, "field")
	})
	t.Run("EnumVariant origin resolves package via owning Enum", func(t *testing.T) {
		t.Parallel()
		pkg := mkPkg()
		variant := pkg.Enums[0].Variants[0]
		runOriginKind(t, pkg, variant, "variant")
	})
	t.Run("File origin resolves package via its Owner Package", func(t *testing.T) {
		t.Parallel()
		pkg := mkPkg()
		file := pkg.Files[0]
		runOriginKind(t, pkg, file, "file")
	})

	// Top-level kinds expose Package directly — exercising each
	// as an Origin covers the [originPackagePath] arm per kind.
	mkPos := func() position.Pos { return position.Pos{File: "x/x.go"} }
	t.Run("Interface origin reads Package field directly", func(t *testing.T) {
		t.Parallel()
		pkg := &node.Package{Name: "x", Path: "example.com/x"}
		iface := &node.Interface{
			BaseNode: node.BaseNode{SourcePos: mkPos()},
			Name:     "I", Package: "example.com/x",
		}
		pkg.Interfaces = []*node.Interface{iface}
		runOriginKind(t, pkg, iface, "interface")
	})
	t.Run("Function origin reads Package field directly", func(t *testing.T) {
		t.Parallel()
		pkg := &node.Package{Name: "x", Path: "example.com/x"}
		fn := &node.Function{
			BaseNode: node.BaseNode{SourcePos: mkPos()},
			Name:     "F", Package: "example.com/x",
		}
		pkg.Functions = []*node.Function{fn}
		runOriginKind(t, pkg, fn, "function")
	})
	t.Run("Variable origin reads Package field directly", func(t *testing.T) {
		t.Parallel()
		pkg := &node.Package{Name: "x", Path: "example.com/x"}
		v := &node.Variable{
			BaseNode: node.BaseNode{SourcePos: mkPos()},
			Name:     "V", Package: "example.com/x",
		}
		pkg.Variables = []*node.Variable{v}
		runOriginKind(t, pkg, v, "variable")
	})
	t.Run("Constant origin reads Package field directly", func(t *testing.T) {
		t.Parallel()
		pkg := &node.Package{Name: "x", Path: "example.com/x"}
		c := &node.Constant{
			BaseNode: node.BaseNode{SourcePos: mkPos()},
			Name:     "C", Package: "example.com/x",
		}
		pkg.Constants = []*node.Constant{c}
		runOriginKind(t, pkg, c, "constant")
	})
	t.Run("Enum origin reads Package field directly", func(t *testing.T) {
		t.Parallel()
		pkg := &node.Package{Name: "x", Path: "example.com/x"}
		e := &node.Enum{
			BaseNode: node.BaseNode{SourcePos: mkPos()},
			Name:     "E", Package: "example.com/x",
		}
		pkg.Enums = []*node.Enum{e}
		runOriginKind(t, pkg, e, "enum")
	})
	t.Run("Alias origin reads Package field directly", func(t *testing.T) {
		t.Parallel()
		pkg := &node.Package{Name: "x", Path: "example.com/x"}
		a := &node.Alias{
			BaseNode: node.BaseNode{SourcePos: mkPos()},
			Name:     "A", Package: "example.com/x",
		}
		pkg.Aliases = []*node.Alias{a}
		runOriginKind(t, pkg, a, "alias")
	})
}

// TestLayout_SlotMissingFilenameProvider pins
// [materialiseOriginSlots]'s sentinel-firing path: a slot
// contribution carrying a [emit.Provenance.SetBy] that doesn't
// match any registered [plugin.FilenameProvider] surfaces
// [pipeline.ErrMissingFilenameProvider] and drops the
// contribution.
func TestLayout_SlotMissingFilenameProvider(t *testing.T) {
	t.Parallel()

	t.Run("slot tuple from unknown plugin surfaces the typed sentinel", func(t *testing.T) {
		t.Parallel()
		origin := &node.Struct{
			BaseNode: node.BaseNode{SourcePos: position.Pos{File: "x/x.go"}},
			Name:     "X", Package: "example.com/x",
		}
		gen := &slotContributingGen{
			name:   "rg",
			suffix: "_gen.go",
			contribute: func(ctx *plugin.GeneratorContext) error {
				// Attribute the contribution to a plugin name that
				// is not registered on the pipeline so the suffix
				// lookup misses.
				return ctx.Store.Emit().AppendOriginSlot(
					origin, "init",
					&emit.Constant{Name: "X", Package: "x"},
					emit.Provenance{SetBy: "ghost"},
				)
			},
		}
		d := diag.New()
		p, err := pipeline.New().
			WithFrontend(&stubFE{name: "fe"}).
			WithGenerator(gen).
			WithBackend(&stubBE{name: "be"}).
			WithSink(sink.NewMemory()).
			WithDiag(d).
			Build()
		assertNoError(t, err)
		runErr := p.Run(t.Context(), "x")
		if !errors.Is(runErr, pipeline.ErrRunHadErrors) {
			t.Fatalf("Run = %v, want ErrRunHadErrors", runErr)
		}
		if !hasDiagContaining(d, pipeline.ErrMissingFilenameProvider.Error()) {
			t.Fatalf("expected ErrMissingFilenameProvider diagnostic; got %+v", d.Diagnostics())
		}
	})
}

// TestLayout_EmptyOriginPackage pins the fall-through path:
// [originSourcePackage] returns nil when origin's owner chain
// never reaches a packaged ancestor (a Method with no Owner is
// the canonical case). Layout still produces a routable Target,
// but with empty Package / ImportPath fields — downstream
// renderers either emit a generic header or surface the missing
// attribution as needed.
func TestLayout_EmptyOriginPackage(t *testing.T) {
	t.Parallel()

	t.Run("Method with nil Owner leaves Target.Package empty", func(t *testing.T) {
		t.Parallel()
		// Method with no Owner — the chain walks to nil and the
		// helper returns the empty package path.
		method := &node.Method{
			BaseNode: node.BaseNode{SourcePos: position.Pos{File: "x/x.go"}},
			Name:     "M",
		}
		item := &emit.Constant{Name: "K", Package: "x"}
		gen := &slotContributingGen{
			name:   "rg",
			suffix: "_meta.go",
			contribute: func(ctx *plugin.GeneratorContext) error {
				return ctx.Store.Emit().AppendOriginSlot(
					method, "init", item, emit.Provenance{SetBy: "rg"},
				)
			},
		}
		p, err := pipeline.New().
			WithFrontend(&stubFE{name: "fe"}).
			WithGenerator(gen).
			WithBackend(&stubBE{name: "be"}).
			WithSink(sink.NewMemory()).
			Build()
		assertNoError(t, err)
		assertNoError(t, p.Run(t.Context(), "x"))
		// The materialised File exists; its Package value is empty
		// since the lookup found no node.Package.
		f, ok := p.Store().Emit().Files().ByQName("x/x_meta.go")
		if !ok {
			t.Fatalf("expected materialised File; got none")
		}
		if f.Package != "" {
			t.Fatalf("File.Package = %q, want empty (no source package resolution)", f.Package)
		}
	})
}

// TestLayout_EmptyOriginPos pins [originSourceDirBasename]'s
// empty-Pos.File branch: a non-nil origin without a source
// position is treated as having no derivable directory or
// basename. Layout still composes the suffix-only filename
// (e.g. "_meta.go") and leaves the directory empty; the
// downstream sink rejects the empty directory at write time, so
// the failure surfaces a second time at IO if the run still
// attempts the write.
func TestLayout_EmptyOriginPos(t *testing.T) {
	t.Parallel()

	t.Run("Method origin with empty Pos.File yields suffix-only Filename", func(t *testing.T) {
		t.Parallel()
		method := &node.Method{Name: "M"} // no SourcePos
		item := &emit.Constant{Name: "K", Package: "x"}
		gen := &slotContributingGen{
			name:   "rg",
			suffix: "_meta.go",
			contribute: func(ctx *plugin.GeneratorContext) error {
				return ctx.Store.Emit().AppendOriginSlot(
					method, "init", item, emit.Provenance{SetBy: "rg"},
				)
			},
		}
		p, err := pipeline.New().
			WithFrontend(&stubFE{name: "fe"}).
			WithGenerator(gen).
			WithBackend(&stubBE{name: "be"}).
			WithSink(sink.NewMemory()).
			Build()
		assertNoError(t, err)
		assertNoError(t, p.Run(t.Context(), "x"))
		// Filename is suffix-only; Dir is empty.
		var found *emit.File
		p.Store().Emit().Files().Range(func(f *emit.File) bool {
			if f.Name == "_meta.go" && f.Dir == "" {
				found = f
			}
			return true
		})
		if found == nil {
			t.Fatalf("expected suffix-only file %q with empty Dir", "_meta.go")
		}
	})
}

// runOriginKind drives one slot contribution anchored at origin
// through the pipeline and asserts the resulting File lands under
// the expected (Dir, Filename) — `x/x_meta.go` for every origin
// kind in the fixture above.
func runOriginKind(t *testing.T, pkg *node.Package, origin node.Node, label string) {
	t.Helper()
	item := &emit.Constant{Name: "Init_" + label, Package: "x"}
	gen := &slotContributingGen{
		name:   "rg",
		suffix: "_meta.go",
		contribute: func(ctx *plugin.GeneratorContext) error {
			return ctx.Store.Emit().AppendOriginSlot(
				origin, "init", item, emit.Provenance{SetBy: "rg"},
			)
		},
	}
	p, err := pipeline.New().
		WithFrontend(&nodePackageFE{name: "fe", pkg: pkg}).
		WithGenerator(gen).
		WithBackend(&stubBE{name: "be"}).
		WithSink(sink.NewMemory()).
		Build()
	assertNoError(t, err)
	assertNoError(t, p.Run(t.Context(), "x"))
	f, ok := p.Store().Emit().Files().ByQName("x/x_meta.go")
	if !ok {
		t.Fatalf("%s origin: file at x/x_meta.go missing", label)
	}
	if got := len(f.Slot("init").Items); got != 1 {
		t.Fatalf("%s origin: init slot items = %d, want 1", label, got)
	}
}

// TestLayout_SlotAppendError pins the error path materialiseOriginSlots
// surfaces when [emit.Slot.Append] rejects a kind-mismatched item.
// The slot under test pins its element kind through the typed
// per-host accessor (Method's body slot accepts only Stmt nodes);
// passing a Struct as the slot item triggers the kind check.
func TestLayout_SlotAppendError(t *testing.T) {
	t.Parallel()

	t.Run("kind-mismatched item surfaces a diagnostic", func(t *testing.T) {
		t.Parallel()
		// Layout composes the slot's owning Target from origin —
		// Dir from source dir, Filename from basename+suffix,
		// Package + ImportPath from the node-store lookup. To
		// match the Target the generator pre-pins below, the test
		// loads a node.Package so the lookup resolves to its short
		// name "x" rather than empty.
		nodePkg := &node.Package{Name: "x", Path: "example.com/x"}
		origin := &node.Struct{
			BaseNode: node.BaseNode{SourcePos: position.Pos{File: "x/x.go"}},
			Name:     "X", Package: "example.com/x",
		}
		nodePkg.Structs = []*node.Struct{origin}
		gen := &slotContributingGen{
			name:   "rg",
			suffix: "_meta.go",
			contribute: func(ctx *plugin.GeneratorContext) error {
				// Pre-create the File at the Target Layout will
				// compose and pin its imports slot's element kind
				// via the typed accessor — the generic File.Slot
				// re-uses the same Slot so the kind check fires on
				// subsequent appends through the by-name path.
				target := emit.Target{
					Dir: "x", Filename: "x_meta.go",
					Package: "x", ImportPath: "example.com/x",
				}
				f, _ := ctx.Store.Emit().FileFor(target)
				_ = f.ImportsSlot()
				// Queue a Constant for the same slot via origin
				// anchoring. Layout composes the same Target and
				// tries to append the Constant to the import-kinded
				// slot — Append rejects on kind mismatch.
				return ctx.Store.Emit().AppendOriginSlot(
					origin, "imports",
					&emit.Constant{Name: "WrongKind", Package: "x"},
					emit.Provenance{SetBy: "rg"},
				)
			},
		}
		d := diag.New()
		p, err := pipeline.New().
			WithFrontend(&nodePackageFE{name: "fe", pkg: nodePkg}).
			WithGenerator(gen).
			WithBackend(&stubBE{name: "be"}).
			WithSink(sink.NewMemory()).
			WithDiag(d).
			Build()
		assertNoError(t, err)
		runErr := p.Run(t.Context(), "x")
		if !errors.Is(runErr, pipeline.ErrRunHadErrors) {
			t.Fatalf("Run = %v, want ErrRunHadErrors (kind mismatch)", runErr)
		}
		if !hasDiagContaining(d, `slot "imports"`) {
			t.Fatalf("expected slot-append diagnostic; got %+v", d.Diagnostics())
		}
	})
}

// TestLayout_OutDirective_NoArgs pins the defensive arm of
// [outDirectiveFilename]: an `out` directive without a positional
// argument is treated as absent, so the Filename composition
// falls through to the source-basename + suffix default.
func TestLayout_OutDirective_NoArgs(t *testing.T) {
	t.Parallel()

	t.Run("out directive without args leaves Filename composed from suffix", func(t *testing.T) {
		t.Parallel()
		origin := &node.Struct{
			BaseNode: node.BaseNode{
				SourcePos: position.Pos{File: "x/host.go"},
				DirectiveList: []*directive.Directive{
					{Name: pipeline.OutDirective}, // no Args
				},
			},
			Name: "Host", Package: "example.com/x",
		}
		s := &emit.Struct{
			BaseEmit: emit.BaseEmit{OriginNode: origin},
			Name:     "HostX", Package: "x",
		}
		p, err := pipeline.New().
			WithFrontend(&stubFE{name: "fe"}).
			WithGenerator(&layoutGen{name: "rg", suffix: "_x.go", pkg: &emit.Package{
				Name: "x", Path: "example.com/x",
				Structs: []*emit.Struct{s},
			}}).
			WithBackend(&stubBE{name: "be"}).
			WithSink(sink.NewMemory()).
			Build()
		assertNoError(t, err)
		assertNoError(t, p.Run(t.Context(), "x"))
		if got, want := s.Target.Filename, "host_x.go"; got != want {
			t.Fatalf("Filename = %q, want %q (no-args directive falls through)", got, want)
		}
	})
}

// TestLayout_EmptySuffixIsFiltered pins the collection-boundary
// contract: a plugin that implements [plugin.FilenameProvider]
// but returns the empty string is filtered out of the suffix
// lookup and surfaces [pipeline.ErrMissingFilenameProvider] when
// it emits a routable decl — the same code path as a plugin
// that doesn't implement the capability at all.
func TestLayout_EmptySuffixIsFiltered(t *testing.T) {
	t.Parallel()

	t.Run("empty FilenameSuffix() is treated as no declaration", func(t *testing.T) {
		t.Parallel()
		origin := &node.Struct{
			BaseNode: node.BaseNode{SourcePos: position.Pos{File: "x/x.go"}},
			Name:     "X", Package: "example.com/x",
		}
		s := &emit.Struct{
			BaseEmit: emit.BaseEmit{OriginNode: origin},
			Name:     "X", Package: "x",
		}
		// layoutGen with suffix="" implements FilenameProvider
		// but returns the empty string — must surface the typed
		// sentinel just like a plugin without the capability.
		d := diag.New()
		p, err := pipeline.New().
			WithFrontend(&stubFE{name: "fe"}).
			WithGenerator(&layoutGen{name: "rg", suffix: "", pkg: &emit.Package{
				Name: "x", Path: "example.com/x",
				Structs: []*emit.Struct{s},
			}}).
			WithBackend(&stubBE{name: "be"}).
			WithSink(sink.NewMemory()).
			WithDiag(d).
			Build()
		assertNoError(t, err)
		runErr := p.Run(t.Context(), "x")
		if !errors.Is(runErr, pipeline.ErrRunHadErrors) {
			t.Fatalf("Run = %v, want ErrRunHadErrors", runErr)
		}
		if !hasDiagContaining(d, pipeline.ErrMissingFilenameProvider.Error()) {
			t.Fatalf("expected ErrMissingFilenameProvider diagnostic; got %+v", d.Diagnostics())
		}
	})
}

// TestLayout_OutDirective_MixedDirectives pins the directive-skip
// loop in [outDirectiveFilename]: a directive list mixing
// non-`out` directives with the `out` directive must skip the
// non-matching ones and find the `out` entry.
func TestLayout_OutDirective_MixedDirectives(t *testing.T) {
	t.Parallel()

	t.Run("non-out directives are skipped before reaching the out directive", func(t *testing.T) {
		t.Parallel()
		origin := &node.Struct{
			BaseNode: node.BaseNode{
				SourcePos: position.Pos{File: "x/x.go"},
				DirectiveList: []*directive.Directive{
					{Name: "other"},
					{Name: "another"},
					{Name: pipeline.OutDirective, Args: []string{"pinned.go"}},
				},
			},
			Name: "X", Package: "example.com/x",
		}
		s := &emit.Struct{
			BaseEmit: emit.BaseEmit{OriginNode: origin},
			Name:     "X", Package: "x",
		}
		p, err := pipeline.New().
			WithFrontend(&stubFE{name: "fe"}).
			WithGenerator(&layoutGen{name: "rg", suffix: "_gen.go", pkg: &emit.Package{
				Name: "x", Path: "example.com/x",
				Structs: []*emit.Struct{s},
			}}).
			WithBackend(&stubBE{name: "be"}).
			WithSink(sink.NewMemory()).
			Build()
		assertNoError(t, err)
		assertNoError(t, p.Run(t.Context(), "x"))
		if got, want := s.Target.Filename, "pinned.go"; got != want {
			t.Fatalf("Target.Filename = %q, want %q (mixed-directive find)", got, want)
		}
	})
}

// TestLayout_OriginSourceDirBasename_NoDir pins the
// current-directory normalisation in [originSourceDirBasename]:
// a Pos.File without a directory component ("x.go") yields
// filepath.Dir == ".", which the helper normalises to the empty
// string so downstream consumers don't accumulate stray dots in
// rendered paths.
func TestLayout_OriginSourceDirBasename_NoDir(t *testing.T) {
	t.Parallel()

	t.Run("Pos.File without a directory yields empty Target.Dir", func(t *testing.T) {
		t.Parallel()
		origin := &node.Struct{
			BaseNode: node.BaseNode{SourcePos: position.Pos{File: "x.go"}},
			Name:     "X", Package: "example.com/x",
		}
		s := &emit.Struct{
			BaseEmit: emit.BaseEmit{OriginNode: origin},
			Name:     "X", Package: "x",
		}
		p, err := pipeline.New().
			WithFrontend(&stubFE{name: "fe"}).
			WithGenerator(&layoutGen{name: "rg", suffix: "_gen.go", pkg: &emit.Package{
				Name: "x", Path: "example.com/x",
				Structs: []*emit.Struct{s},
			}}).
			WithBackend(&stubBE{name: "be"}).
			WithSink(sink.NewMemory()).
			Build()
		assertNoError(t, err)
		assertNoError(t, p.Run(t.Context(), "x"))
		if s.Target.Dir != "" {
			t.Fatalf("Target.Dir = %q, want empty (`.` normalisation)", s.Target.Dir)
		}
		if got, want := s.Target.Filename, "x_gen.go"; got != want {
			t.Fatalf("Target.Filename = %q, want %q", got, want)
		}
	})
}

// TestLayout_OutDirectiveRegistered confirms the `out` directive
// is registered in the framework's core directive set at Build
// time, regardless of whether the caller also supplies directives
// via [Builder.WithDirective].
func TestLayout_OutDirectiveRegistered(t *testing.T) {
	t.Parallel()

	t.Run("out directive registered with no caller-supplied schemas", func(t *testing.T) {
		t.Parallel()
		p, err := pipeline.New().
			WithFrontend(&stubFE{name: "fe"}).
			WithBackend(&stubBE{name: "be"}).
			WithSink(sink.NewMemory()).
			Build()
		assertNoError(t, err)
		if _, ok := p.DirectiveRegistry().Lookup(pipeline.OutDirective); !ok {
			t.Fatalf("OutDirective should be registered in the core set")
		}
	})
}
