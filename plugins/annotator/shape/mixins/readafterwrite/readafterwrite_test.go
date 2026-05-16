// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package readafterwrite_test

import (
	"reflect"
	"testing"

	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/readafterwrite"
)

func TestMixin_Identity(t *testing.T) {
	t.Parallel()
	m := readafterwrite.Mixin()
	if m.Name != readafterwrite.Name {
		t.Fatalf("Mixin().Name = %q, want %q", m.Name, readafterwrite.Name)
	}
	if !reflect.DeepEqual(m.Params, readafterwrite.Params) {
		t.Fatalf("Mixin().Params = %v, want %v", m.Params, readafterwrite.Params)
	}
}
