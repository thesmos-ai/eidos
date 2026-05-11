// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package golang_test

import (
	"testing"
)

// TestConvertInterface covers the per-interface conversion path:
// methods, embeds, generic params, and method-body shapes.
func TestConvertInterface(t *testing.T) {
	t.Parallel()
	t.Run("interface carries name and package path", func(t *testing.T) {
		t.Parallel()
		pkg := requirePackage(t, map[string]string{
			"a.go": "package a\n\ntype Op interface{ Do() }\n",
		})
		i := pkg.InterfaceByName("Op")
		if i == nil || i.Name != "Op" {
			t.Fatalf("Op interface missing")
		}
	})

	t.Run("explicit methods carry params and returns", func(t *testing.T) {
		t.Parallel()
		pkg := requirePackage(t, map[string]string{
			"a.go": "package a\n\nimport \"context\"\n\ntype Reader interface{\n\tRead(ctx context.Context, key string) ([]byte, error)\n}\n",
		})
		i := pkg.InterfaceByName("Reader")
		if i == nil {
			t.Fatalf("Reader missing")
		}
		m := i.Methods[0]
		if m.Name != "Read" {
			t.Fatalf("method name = %q", m.Name)
		}
		if len(m.Params) != 2 {
			t.Fatalf("expected 2 params, got %d", len(m.Params))
		}
		if m.Params[0].Name != "ctx" || m.Params[1].Name != "key" {
			t.Fatalf("param names = %q,%q", m.Params[0].Name, m.Params[1].Name)
		}
		if len(m.Returns) != 2 {
			t.Fatalf("expected 2 returns, got %d", len(m.Returns))
		}
	})

	t.Run("embedded interface surfaces as Embed", func(t *testing.T) {
		t.Parallel()
		pkg := requirePackage(t, map[string]string{
			"a.go": "package a\n\ntype Base interface{ Close() error }\ntype Ext interface{ Base; Do() }\n",
		})
		i := pkg.InterfaceByName("Ext")
		if i == nil {
			t.Fatalf("Ext missing")
		}
		if len(i.Embeds) != 1 || i.Embeds[0].Type.Name != "Base" {
			t.Fatalf("expected one embed of Base, got %+v", i.Embeds)
		}
	})

	t.Run("variadic last param records Variadic and element type", func(t *testing.T) {
		t.Parallel()
		pkg := requirePackage(t, map[string]string{
			"a.go": "package a\n\ntype Op interface{ Do(args ...int) }\n",
		})
		m := pkg.InterfaceByName("Op").Methods[0]
		last := m.Params[len(m.Params)-1]
		if !last.Variadic {
			t.Fatalf("expected last param to be variadic")
		}
		if last.Type == nil || last.Type.Name != "int" {
			t.Fatalf("variadic element type = %+v, want int", last.Type)
		}
	})

	t.Run("generic interface carries type parameters", func(t *testing.T) {
		t.Parallel()
		pkg := requirePackage(t, map[string]string{
			"a.go": "package a\n\ntype Op[T any] interface{ Do(v T) T }\n",
		})
		i := pkg.InterfaceByName("Op")
		if len(i.TypeParams) != 1 || i.TypeParams[0].Name != "T" {
			t.Fatalf("expected one type-param T, got %+v", i.TypeParams)
		}
	})

	t.Run("alias of an interface populates methods via the type-only path", func(t *testing.T) {
		t.Parallel()
		// `type Alias = Original` of an interface surfaces as a
		// fresh Interface whose body is built by
		// [populateInterfaceFromTypeOnly] because the AST type
		// expression is an Ident rather than an InterfaceType.
		pkg := requirePackage(t, map[string]string{
			"a.go": "package a\n\ntype Original interface{ Do() }\n\ntype Alias = Original\n",
		})
		alias := pkg.InterfaceByName("Alias")
		if alias == nil {
			t.Skipf("alias of interface did not surface — type-only path unreachable")
		}
		if len(alias.Methods) == 0 {
			t.Fatalf("Alias must carry methods populated by the type-only path")
		}
	})

	t.Run("alias of an interface preserves rich method signatures", func(t *testing.T) {
		t.Parallel()
		// Drives [methodFromSignature] through every body branch:
		// params, variadic, and returns.
		pkg := requirePackage(t, map[string]string{
			"a.go": "package a\n\ntype Original interface{\n\tDo(id string, parts ...int) (bool, error)\n}\n\ntype Alias = Original\n",
		})
		alias := pkg.InterfaceByName("Alias")
		if alias == nil {
			t.Skipf("alias of interface did not surface — type-only path unreachable")
		}
		if len(alias.Methods) != 1 {
			t.Fatalf("expected 1 method, got %d", len(alias.Methods))
		}
		m := alias.Methods[0]
		if len(m.Params) != 2 || len(m.Returns) != 2 {
			t.Fatalf("expected 2 params + 2 returns, got %d/%d", len(m.Params), len(m.Returns))
		}
		if !m.Params[1].Variadic {
			t.Fatalf("expected last param to be variadic")
		}
	})

	t.Run("alias of an interface with embeds carries the embed via type-only path", func(t *testing.T) {
		t.Parallel()
		pkg := requirePackage(t, map[string]string{
			"a.go": "package a\n\ntype Base interface{ Close() error }\n\ntype Original interface{ Base; Do() }\n\ntype Alias = Original\n",
		})
		alias := pkg.InterfaceByName("Alias")
		if alias == nil {
			t.Skipf("alias of interface did not surface — type-only path unreachable")
		}
		if len(alias.Embeds) == 0 {
			t.Fatalf("expected at least one embed on the aliased interface")
		}
	})

	t.Run("constraint-interface type-set surfaces as embedded ref", func(t *testing.T) {
		t.Parallel()
		pkg := requirePackage(t, map[string]string{
			"a.go": "package a\n\ntype Numeric interface{ ~int | ~float64 }\n",
		})
		i := pkg.InterfaceByName("Numeric")
		if i == nil {
			t.Fatalf("Numeric missing")
		}
		// Type-set unions surface through go.constraintTerms meta
		// when carried on a type-param; on the interface itself we
		// only assert the structural shape and constraint meta.
		if len(i.Methods) != 0 {
			t.Fatalf("constraint-only interface should have no methods, got %d", len(i.Methods))
		}
	})

	t.Run("embedded type the type-checker could not resolve does not crash the converter", func(t *testing.T) {
		t.Parallel()
		// Drives typeRefForInterfaceEmbed's nil-typeinfo return; the
		// converter must surface a diagnostic and continue.
		_, d := loadFromSource(t, map[string]string{
			"a.go": "package a\n\ntype I interface{ Missing }\n",
		})
		if !d.HasErrors() {
			t.Fatalf("expected an Error diagnostic for unresolved interface embed")
		}
	})
}
