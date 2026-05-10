// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package emit_test

import (
	"strings"
	"testing"

	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/emit"
)

func TestKindConstants_AreUniqueAndPrefixed(t *testing.T) {
	t.Parallel()

	t.Run("every Kind is non-empty, prefixed emit., and unique", func(t *testing.T) {
		t.Parallel()
		all := []directive.Kind{
			emit.KindPackage, emit.KindFile, emit.KindImport,
			emit.KindStruct, emit.KindInterface, emit.KindMethod,
			emit.KindField, emit.KindFunction, emit.KindVariable,
			emit.KindConstant, emit.KindEnum, emit.KindEnumVariant,
			emit.KindAlias, emit.KindEmbed, emit.KindParam, emit.KindTypeParam,
			emit.KindTypeRef, emit.KindExternalRef, emit.KindBuiltinRef,
			emit.KindCompositeRef, emit.KindStmt, emit.KindExpr, emit.KindSlot,
		}
		seen := make(map[directive.Kind]struct{}, len(all))
		for _, k := range all {
			if k == "" {
				t.Fatalf("found an empty Kind constant")
			}
			if !strings.HasPrefix(string(k), "emit.") {
				t.Fatalf("Kind %q lacks emit. prefix", k)
			}
			if _, dup := seen[k]; dup {
				t.Fatalf("duplicate Kind constant %q", k)
			}
			seen[k] = struct{}{}
		}
	})
}
