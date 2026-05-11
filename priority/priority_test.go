// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package priority_test

import (
	"testing"

	"go.thesmos.sh/eidos/priority"
)

func TestPriority_AnnotatorBucketsAreOrdered(t *testing.T) {
	t.Parallel()

	t.Run("AnnotatorShape < AnnotatorRefinement < AnnotatorValidation", func(t *testing.T) {
		t.Parallel()
		if priority.AnnotatorShape >= priority.AnnotatorRefinement ||
			priority.AnnotatorRefinement >= priority.AnnotatorValidation {

			t.Fatalf("annotator buckets out of order: shape=%d refine=%d validate=%d",
				priority.AnnotatorShape, priority.AnnotatorRefinement, priority.AnnotatorValidation)
		}
	})
}

func TestPriority_GeneratorBucketsAreOrdered(t *testing.T) {
	t.Parallel()

	t.Run("Foundation < Composition < CrossCutting < Finalize", func(t *testing.T) {
		t.Parallel()
		buckets := []priority.Priority{
			priority.GeneratorFoundation,
			priority.GeneratorComposition,
			priority.GeneratorCrossCutting,
			priority.GeneratorFinalize,
		}
		for i := 1; i < len(buckets); i++ {
			if buckets[i-1] >= buckets[i] {
				t.Fatalf("generator buckets out of order at index %d: %v", i, buckets)
			}
		}
	})
}

func TestPriority_DefaultIsCrossCuttingBucket(t *testing.T) {
	t.Parallel()

	t.Run("Default lands in the cross-cutting/generator-default bucket", func(t *testing.T) {
		t.Parallel()
		if priority.Default != priority.GeneratorCrossCutting {
			t.Fatalf("Default = %d, want %d", priority.Default, priority.GeneratorCrossCutting)
		}
	})
}
