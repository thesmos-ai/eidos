// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package pipeline_test

import (
	"testing"

	"go.thesmos.sh/eidos/pipeline"
)

func TestPhase_String(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		p    pipeline.Phase
		want string
	}{
		{"Frontend", pipeline.PhaseFrontend, "frontend"},
		{"Annotator", pipeline.PhaseAnnotator, "annotator"},
		{"Generator", pipeline.PhaseGenerator, "generator"},
		{"unknown stringifies with a marker", pipeline.Phase(99), "phase(?)"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := tc.p.String(); got != tc.want {
				t.Fatalf("%s String = %q, want %q", tc.name, got, tc.want)
			}
		})
	}
}
