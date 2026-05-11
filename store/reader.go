// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package store

import (
	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/node"
)

// Reader is a per-plugin read-tracking handle onto a [Store]. The
// pipeline gives each plugin its own Reader; queries through the
// Reader update the underlying [ReadSet] which the cache layer
// hashes for the plugin's cache key.
//
// Reader exposes typed [Query] constructors per node and emit kind
// — every terminal call ([Query.Each], [Query.Slice], [Query.First],
// [Query.Count]) records the query's source tag in the ReadSet so
// cache invalidation can detect "this plugin no longer reads X".
//
// For tests and tooling that don't need read-tracking, accessing
// the underlying [Bucket] / [MultiIndex] directly through
// [Store.Nodes] / [Store.Emit] remains fine — those bypass the
// Reader entirely.
type Reader struct {
	store *Store
	reads *ReadSet
}

// NewReader returns a Reader that wraps s with a fresh empty
// [ReadSet]. The ReadSet is private to the returned Reader; create
// one Reader per plugin per pipeline run.
func NewReader(s *Store) *Reader {
	return &Reader{store: s, reads: NewReadSet()}
}

// Store returns the underlying [Store] the reader wraps.
func (r *Reader) Store() *Store { return r.store }

// ReadSet returns the [ReadSet] populated by the reader's terminal
// calls. The ReadSet is the same instance across the reader's
// lifetime; cache keys are derived from it after the plugin's
// phase completes.
func (r *Reader) ReadSet() *ReadSet { return r.reads }

// Packages returns a [Query] over every recorded [node.Package].
func (r *Reader) Packages() *Query[*node.Package] {
	return newQuery(r.store.Nodes().Packages().Items(), r.reads, "node:packages")
}

// Files returns a [Query] over every recorded [node.File].
func (r *Reader) Files() *Query[*node.File] {
	return newQuery(r.store.Nodes().Files().Items(), r.reads, "node:files")
}

// Imports returns a [Query] over every recorded [node.Import].
func (r *Reader) Imports() *Query[*node.Import] {
	return newQuery(r.store.Nodes().Imports().Items(), r.reads, "node:imports")
}

// Structs returns a [Query] over every recorded [node.Struct].
func (r *Reader) Structs() *Query[*node.Struct] {
	return newQuery(r.store.Nodes().Structs().Items(), r.reads, "node:structs")
}

// Interfaces returns a [Query] over every recorded [node.Interface].
func (r *Reader) Interfaces() *Query[*node.Interface] {
	return newQuery(r.store.Nodes().Interfaces().Items(), r.reads, "node:interfaces")
}

// Methods returns a [Query] over every recorded [node.Method].
func (r *Reader) Methods() *Query[*node.Method] {
	return newQuery(r.store.Nodes().Methods().Items(), r.reads, "node:methods")
}

// Fields returns a [Query] over every recorded [node.Field].
func (r *Reader) Fields() *Query[*node.Field] {
	return newQuery(r.store.Nodes().Fields().Items(), r.reads, "node:fields")
}

// Functions returns a [Query] over every recorded [node.Function].
func (r *Reader) Functions() *Query[*node.Function] {
	return newQuery(r.store.Nodes().Functions().Items(), r.reads, "node:functions")
}

// Variables returns a [Query] over every recorded [node.Variable].
func (r *Reader) Variables() *Query[*node.Variable] {
	return newQuery(r.store.Nodes().Variables().Items(), r.reads, "node:variables")
}

// Constants returns a [Query] over every recorded [node.Constant].
func (r *Reader) Constants() *Query[*node.Constant] {
	return newQuery(r.store.Nodes().Constants().Items(), r.reads, "node:constants")
}

// Enums returns a [Query] over every recorded [node.Enum].
func (r *Reader) Enums() *Query[*node.Enum] {
	return newQuery(r.store.Nodes().Enums().Items(), r.reads, "node:enums")
}

// EnumVariants returns a [Query] over every recorded
// [node.EnumVariant].
func (r *Reader) EnumVariants() *Query[*node.EnumVariant] {
	return newQuery(r.store.Nodes().EnumVariants().Items(), r.reads, "node:enum_variants")
}

// Aliases returns a [Query] over every recorded [node.Alias].
func (r *Reader) Aliases() *Query[*node.Alias] {
	return newQuery(r.store.Nodes().Aliases().Items(), r.reads, "node:aliases")
}

// EmitPackages returns a [Query] over every recorded [emit.Package].
func (r *Reader) EmitPackages() *Query[*emit.Package] {
	return newQuery(r.store.Emit().Packages().Items(), r.reads, "emit:packages")
}

// EmitFiles returns a [Query] over every recorded [emit.File].
func (r *Reader) EmitFiles() *Query[*emit.File] {
	return newQuery(r.store.Emit().Files().Items(), r.reads, "emit:files")
}

// EmitImports returns a [Query] over every recorded [emit.Import].
func (r *Reader) EmitImports() *Query[*emit.Import] {
	return newQuery(r.store.Emit().Imports().Items(), r.reads, "emit:imports")
}

// EmitStructs returns a [Query] over every recorded [emit.Struct].
func (r *Reader) EmitStructs() *Query[*emit.Struct] {
	return newQuery(r.store.Emit().Structs().Items(), r.reads, "emit:structs")
}

// EmitInterfaces returns a [Query] over every recorded [emit.Interface].
func (r *Reader) EmitInterfaces() *Query[*emit.Interface] {
	return newQuery(r.store.Emit().Interfaces().Items(), r.reads, "emit:interfaces")
}

// EmitMethods returns a [Query] over every recorded [emit.Method].
func (r *Reader) EmitMethods() *Query[*emit.Method] {
	return newQuery(r.store.Emit().Methods().Items(), r.reads, "emit:methods")
}

// EmitFields returns a [Query] over every recorded [emit.Field].
func (r *Reader) EmitFields() *Query[*emit.Field] {
	return newQuery(r.store.Emit().Fields().Items(), r.reads, "emit:fields")
}

// EmitFunctions returns a [Query] over every recorded [emit.Function].
func (r *Reader) EmitFunctions() *Query[*emit.Function] {
	return newQuery(r.store.Emit().Functions().Items(), r.reads, "emit:functions")
}

// EmitVariables returns a [Query] over every recorded [emit.Variable].
func (r *Reader) EmitVariables() *Query[*emit.Variable] {
	return newQuery(r.store.Emit().Variables().Items(), r.reads, "emit:variables")
}

// EmitConstants returns a [Query] over every recorded [emit.Constant].
func (r *Reader) EmitConstants() *Query[*emit.Constant] {
	return newQuery(r.store.Emit().Constants().Items(), r.reads, "emit:constants")
}

// EmitEnums returns a [Query] over every recorded [emit.Enum].
func (r *Reader) EmitEnums() *Query[*emit.Enum] {
	return newQuery(r.store.Emit().Enums().Items(), r.reads, "emit:enums")
}

// EmitEnumVariants returns a [Query] over every recorded
// [emit.EnumVariant].
func (r *Reader) EmitEnumVariants() *Query[*emit.EnumVariant] {
	return newQuery(r.store.Emit().EnumVariants().Items(), r.reads, "emit:enum_variants")
}

// EmitAliases returns a [Query] over every recorded [emit.Alias].
func (r *Reader) EmitAliases() *Query[*emit.Alias] {
	return newQuery(r.store.Emit().Aliases().Items(), r.reads, "emit:aliases")
}
