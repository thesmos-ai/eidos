// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package builder is the production fluent-builder generator. For
// every source-side struct annotated with `+gen:builder`, it emits
// a `<Type><Suffix>` struct carrying one field per exported source
// field, one `<SetterPrefix><Field>(value) *<Type>Builder` setter
// per field, and a `Build() *<Type>` finalizer that returns a
// pointer to the populated source value.
//
// # Generated shape
//
// Given:
//
//	//+gen:builder
//	type User struct {
//	    Name  string
//	    Email string
//	}
//
// The plugin emits:
//
//	type UserBuilder struct {
//	    Name  string
//	    Email string
//	}
//
//	func (b *UserBuilder) WithName(value string) *UserBuilder {
//	    b.Name = value
//	    return b
//	}
//
//	func (b *UserBuilder) WithEmail(value string) *UserBuilder {
//	    b.Email = value
//	    return b
//	}
//
//	func (b *UserBuilder) Build() *User {
//	    return &User{Name: b.Name, Email: b.Email}
//	}
//
// # Routing
//
// The plugin makes no placement decisions. Every emit decl
// anchors on the source struct via [builder.Context.Anchor]; the
// pipeline's Layout phase composes [emit.Target] from the
// resolved precedence pipeline (framework default → project
// config → per-directive `out=` / `pkg=` → CLI flags) — see
// [docs/plugin/routing.md] for the full surface.
//
// # Extension surface
//
// The emitted struct + methods expose the standard slot surface
// every cross-cutting concern attaches to:
//
//   - `Struct.FieldsSlot()` — append extra fields (e.g. a
//     validation-tracking flag from a validator cross-cutter).
//   - `Struct.MethodsSlot()` — append helper methods (e.g.
//     `Reset()`, `Clone()`).
//   - `Method.Prebody()` / `Method.Postbody()` — statements that
//     run before / after each setter or the Build finalizer.
//
// # Position in the pipeline
//
// The plugin runs in [sdk.GeneratorFoundation]: it emits onto
// the source-side struct only, requiring no prior generator
// output. Composition-bucket generators that decorate the
// builder register higher up (e.g. [sdk.GeneratorComposition])
// and see the emitted builder when they walk
// [store.Reader.EmitStructs].
package builder
