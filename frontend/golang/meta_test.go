// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package golang_test

import (
	"testing"

	"go.thesmos.sh/eidos/frontend/golang"
)

// TestMetaKeyNames verifies every documented `go.*` key carries the
// canonical name. The constants are programmer-facing identifiers
// the rest of the codebase pivots on (cache keys, --explain output,
// directive routing); a typo here would silently shadow the
// expected name without a build error.
//
// Table-driven because every row is the same (Key → expected name)
// uniform shape.
func TestMetaKeyNames(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		key  string
		want string
	}{
		{"MetaIsChannel", golang.MetaIsChannel.Name(), "go.isChannel"},
		{"MetaChanDir", golang.MetaChanDir.Name(), "go.chanDir"},
		{"MetaChanElem", golang.MetaChanElem.Name(), "go.chanElem"},
		{"MetaIsContext", golang.MetaIsContext.Name(), "go.isContext"},
		{"MetaIsError", golang.MetaIsError.Name(), "go.isError"},
		{"MetaIsStringer", golang.MetaIsStringer.Name(), "go.isStringer"},
		{"MetaIsComparable", golang.MetaIsComparable.Name(), "go.isComparable"},
		{"MetaEmbedsInterface", golang.MetaEmbedsInterface.Name(), "go.embedsInterface"},
		{"MetaIsEmptyInterface", golang.MetaIsEmptyInterface.Name(), "go.isEmptyInterface"},
		{"MetaIsConstraintInterface", golang.MetaIsConstraintInterface.Name(), "go.isConstraintInterface"},
		{"MetaUnderlyingKind", golang.MetaUnderlyingKind.Name(), "go.underlyingKind"},
		{"MetaIsIterSeq", golang.MetaIsIterSeq.Name(), "go.isIterSeq"},
		{"MetaIsIterSeq2", golang.MetaIsIterSeq2.Name(), "go.isIterSeq2"},
		{"MetaIterKeyType", golang.MetaIterKeyType.Name(), "go.iterKeyType"},
		{"MetaIterValueType", golang.MetaIterValueType.Name(), "go.iterValueType"},
		{"MetaIotaValue", golang.MetaIotaValue.Name(), "go.iotaValue"},
		{"MetaReceiverIsPointer", golang.MetaReceiverIsPointer.Name(), "go.receiverIsPointer"},
		{"MetaConstraintTerms", golang.MetaConstraintTerms.Name(), "go.constraintTerms"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if tc.key != tc.want {
				t.Fatalf("%s name = %q, want %q", tc.name, tc.key, tc.want)
			}
		})
	}
}

// TestMetaFrontend covers the cross-frontend provenance-marker
// convention: every produced [node.Package] carries the bare
// `frontend` meta key whose value is the producing frontend's
// plugin name. The marker is the scope mechanism downstream bridge
// annotators and the cross-namespace audit step pivot on.
func TestMetaFrontend(t *testing.T) {
	t.Parallel()

	t.Run("key carries the documented bare 'frontend' name", func(t *testing.T) {
		t.Parallel()
		if got := golang.MetaFrontend.Name(); got != "frontend" {
			t.Fatalf("MetaFrontend name = %q, want %q", got, "frontend")
		}
	})

	t.Run("converter stamps the marker on every produced node.Package", func(t *testing.T) {
		t.Parallel()
		pkg := requirePackage(t, map[string]string{
			"a.go": "package a\n",
		})
		got, ok := golang.MetaFrontend.Get(pkg.Meta())
		if !ok {
			t.Fatalf("MetaFrontend missing on go-frontend package %+v", pkg.Meta())
		}
		if got != golang.FrontendName {
			t.Fatalf("MetaFrontend = %q, want %q", got, golang.FrontendName)
		}
	})
}

// TestMetaTagPrefix verifies the documented namespace under which
// struct-tag entries are stamped.
func TestMetaTagPrefix(t *testing.T) {
	t.Parallel()
	t.Run("uses the documented go.tag. namespace", func(t *testing.T) {
		t.Parallel()
		if golang.MetaTagPrefix != "go.tag." {
			t.Fatalf("MetaTagPrefix = %q, want %q", golang.MetaTagPrefix, "go.tag.")
		}
	})
}

// TestStampChanMeta covers channel-direction metadata stamped onto
// every chan-typed reference produced by the converter.
func TestStampChanMeta(t *testing.T) {
	t.Parallel()
	t.Run("bidirectional channel records 'both'", func(t *testing.T) {
		t.Parallel()
		pkg := requirePackage(t, map[string]string{
			"a.go": "package a\n\ntype Holder struct { Ch chan int }\n",
		})
		s := pkg.StructByName("Holder")
		if s == nil {
			t.Fatalf("Holder missing")
		}
		f := s.FieldByName("Ch")
		if f == nil || f.Type == nil {
			t.Fatalf("Ch field type missing")
		}
		got, ok := golang.MetaIsChannel.Get(f.Type.Meta())
		if !ok || !got {
			t.Fatalf("expected MetaIsChannel=true, got (%v, %v)", got, ok)
		}
		dir, ok := golang.MetaChanDir.Get(f.Type.Meta())
		if !ok || dir != "both" {
			t.Fatalf("expected MetaChanDir=both, got (%q, %v)", dir, ok)
		}
		elem, ok := golang.MetaChanElem.Get(f.Type.Meta())
		if !ok || elem != "int" {
			t.Fatalf("expected MetaChanElem=int, got (%q, %v)", elem, ok)
		}
	})

	t.Run("send-only channel records 'send'", func(t *testing.T) {
		t.Parallel()
		pkg := requirePackage(t, map[string]string{
			"a.go": "package a\n\ntype Holder struct { Out chan<- int }\n",
		})
		f := pkg.StructByName("Holder").FieldByName("Out")
		dir, _ := golang.MetaChanDir.Get(f.Type.Meta())
		if dir != "send" {
			t.Fatalf("expected MetaChanDir=send, got %q", dir)
		}
	})

	t.Run("recv-only channel records 'recv'", func(t *testing.T) {
		t.Parallel()
		pkg := requirePackage(t, map[string]string{
			"a.go": "package a\n\ntype Holder struct { In <-chan int }\n",
		})
		f := pkg.StructByName("Holder").FieldByName("In")
		dir, _ := golang.MetaChanDir.Get(f.Type.Meta())
		if dir != "recv" {
			t.Fatalf("expected MetaChanDir=recv, got %q", dir)
		}
	})
}
