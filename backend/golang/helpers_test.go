// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package golang_test

import (
	"errors"
	"path/filepath"
	"runtime"
	"testing"

	"go.thesmos.sh/eidos/core/diag"
	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/plugin"
	"go.thesmos.sh/eidos/sink"
	"go.thesmos.sh/eidos/store"
)

// newBackendContext returns a fresh [plugin.BackendContext] backed
// by an empty in-memory [store.Store], an in-memory sink, and a
// fresh diagnostic sink. Tests populate the store via
// [store.EmitView.AddPackage] before invoking Render.
func newBackendContext(t *testing.T) (*plugin.BackendContext, *sink.Memory, *diag.Sink) {
	t.Helper()
	s := store.New()
	mem := sink.NewMemory()
	d := diag.New()
	r := store.NewReader(s)
	ctx := &plugin.BackendContext{
		Store:  s,
		Reader: r,
		Diag:   d,
		Sink:   mem,
		Lang:   "golang",
	}
	return ctx, mem, d
}

// addEmitPackage seeds ctx.Store.Emit() with pkg, failing the test
// on any duplication error.
func addEmitPackage(t *testing.T, ctx *plugin.BackendContext, pkg *emit.Package) {
	t.Helper()
	if err := ctx.Store.Emit().AddPackage(pkg); err != nil {
		t.Fatalf("AddPackage: %v", err)
	}
}

// emitStructWithFields builds a minimal [*emit.Struct] declaring
// name in pkgPath with the supplied builtin-typed fields and
// targeted at target. Field order in the resulting struct matches
// the order of the supplied (name, type) pairs.
func emitStructWithFields(pkgPath, name string, target emit.Target, fields ...fieldSpec) *emit.Struct {
	out := &emit.Struct{
		Name:    name,
		Package: pkgPath,
		Target:  target,
	}
	for _, f := range fields {
		out.Fields = append(out.Fields, &emit.Field{
			Name: f.name,
			Type: emit.Builtin(f.builtin),
		})
	}
	return out
}

// fieldSpec captures a single field's name + builtin-type name for
// concise fixture construction. The two-field shape keeps tests
// readable when a struct has many fields.
type fieldSpec struct {
	name    string
	builtin string
}

// emitPackage builds an [*emit.Package] holding the supplied
// structs under pkgPath.
func emitPackage(pkgPath string, structs ...*emit.Struct) *emit.Package {
	return &emit.Package{
		Name:    pkgPath,
		Path:    pkgPath,
		Structs: structs,
	}
}

// errSinkBoom is the sentinel a [failingSink] returns to exercise
// the backend's sink-error propagation path.
var errSinkBoom = errors.New("backend/golang test: simulated sink failure")

// failingSink satisfies [sink.Sink] by always returning
// [errSinkBoom] from Write.
type failingSink struct{}

func (*failingSink) Write(emit.Target, []byte) error { return errSinkBoom }

// goldenPath returns the absolute path of the supplied
// testdata/golden/<name>.go.golden fixture, resolved against the
// directory of the calling test source file rather than the
// working directory. Sibling tests in this package pivot
// [os.Chdir] under [chdirMu]-style serialisation; a relative path
// would race those pivots.
func goldenPath(t *testing.T, name string) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("runtime.Caller(0) failed; cannot resolve golden path")
	}
	return filepath.Join(filepath.Dir(file), "testdata", "golden", name)
}
