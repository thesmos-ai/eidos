// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package opt_test

import (
	"errors"
	"testing"

	"go.thesmos.sh/eidos/core/opt"
)

// holderOptions is the destination struct the Holder tests bind to.
// Two fields cover both the required-and-defaulted paths through
// [Options.Decode].
type holderOptions struct {
	Output string `eidos:"output_package,required"`
	Suffix string `eidos:"suffix,default=Repo"`
}

func TestBind(t *testing.T) {
	t.Parallel()

	t.Run("returns a holder whose schema reflects the target type", func(t *testing.T) {
		t.Parallel()
		var got holderOptions
		h := opt.Bind(&got)
		if !h.OptionsSchema().HasField("output_package") {
			t.Fatalf("schema should declare the output_package field")
		}
		if !h.OptionsSchema().HasField("suffix") {
			t.Fatalf("schema should declare the suffix field")
		}
	})

	t.Run("SetOptions writes decoded values into the bound target", func(t *testing.T) {
		t.Parallel()
		var got holderOptions
		h := opt.Bind(&got)
		err := h.SetOptions(opt.New(h.OptionsSchema(), map[string]string{
			"output_package": "gen",
		}))
		if err != nil {
			t.Fatalf("SetOptions: %v", err)
		}
		if got.Output != "gen" {
			t.Fatalf("Output = %q, want %q", got.Output, "gen")
		}
		if got.Suffix != "Repo" {
			t.Fatalf("Suffix should pick up its default; got %q", got.Suffix)
		}
	})

	t.Run("SetOptions surfaces Decode errors verbatim", func(t *testing.T) {
		t.Parallel()
		var got holderOptions
		h := opt.Bind(&got)
		err := h.SetOptions(opt.New(h.OptionsSchema(), map[string]string{}))
		if !errors.Is(err, opt.ErrMissingRequired) {
			t.Fatalf("missing required option should surface ErrMissingRequired; got %v", err)
		}
	})

	t.Run("OptionsSchema returns a stable schema across calls", func(t *testing.T) {
		t.Parallel()
		var got holderOptions
		h := opt.Bind(&got)
		first := h.OptionsSchema().Names()
		second := h.OptionsSchema().Names()
		if len(first) != len(second) {
			t.Fatalf("schema name slice length differs across calls (%d vs %d)", len(first), len(second))
		}
		for i, n := range first {
			if n != second[i] {
				t.Fatalf("schema names differ at %d: %q vs %q", i, n, second[i])
			}
		}
	})
}
