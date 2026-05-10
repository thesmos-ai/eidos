// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package node_test

import (
	"testing"

	"go.thesmos.sh/eidos/node"
)

func TestField_HasTag(t *testing.T) {
	t.Parallel()

	t.Run("returns true when Tag is set", func(t *testing.T) {
		t.Parallel()
		f := node.Field{Tag: `json:"name"`}
		if !f.HasTag() {
			t.Fatalf("Tag set should report HasTag true")
		}
	})

	t.Run("returns false when Tag is empty", func(t *testing.T) {
		t.Parallel()
		var f node.Field
		if f.HasTag() {
			t.Fatalf("empty Tag should report HasTag false")
		}
	})
}
