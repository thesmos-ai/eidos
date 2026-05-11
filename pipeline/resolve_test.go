// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package pipeline_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"go.thesmos.sh/eidos/pipeline"
	"go.thesmos.sh/eidos/priority"
)

func TestResolvePlan_AnnotatorBucketOrder(t *testing.T) {
	t.Parallel()

	t.Run("buckets are visited in ascending priority", func(t *testing.T) {
		t.Parallel()
		// Two annotators, one in a higher bucket than the other.
		// The higher-priority bucket comes second in registration
		// order so we know the resolver isn't just preserving input.
		highBucket := &stubAnnCap{name: "high", priority: priority.AnnotatorValidation}
		lowBucket := &stubAnnCap{name: "low", priority: priority.AnnotatorShape}
		p, err := pipeline.New().
			WithFrontend(&stubFE{name: "fe"}).
			WithAnnotator(highBucket).
			WithAnnotator(lowBucket).
			WithBackend(&stubBE{name: "be"}).
			Build()
		assertNoError(t, err)
		got := names(p.Plan().Annotators)
		if !slices.Equal(got, []string{"low", "high"}) {
			t.Fatalf("buckets should resolve ascending; got %v", got)
		}
	})

	t.Run("plain annotators land in priority.Default", func(t *testing.T) {
		t.Parallel()
		// "plain" doesn't implement CapabilityProvider — it goes into
		// priority.Default (300) so it sorts after the priority-200
		// shape bucket and before priority-400 validation.
		shape := &stubAnnCap{name: "shape", priority: priority.AnnotatorShape}
		val := &stubAnnCap{name: "val", priority: priority.AnnotatorValidation}
		plain := &stubAnn{name: "plain"}
		p, err := pipeline.New().
			WithFrontend(&stubFE{name: "fe"}).
			WithAnnotator(val).
			WithAnnotator(plain).
			WithAnnotator(shape).
			WithBackend(&stubBE{name: "be"}).
			Build()
		assertNoError(t, err)
		got := names(p.Plan().Annotators)
		if !slices.Equal(got, []string{"shape", "plain", "val"}) {
			t.Fatalf("plain plugin should land in priority.Default; got %v", got)
		}
	})
}

func TestResolvePlan_TopoWithinBucket(t *testing.T) {
	t.Parallel()

	t.Run("Requires drives ordering within a single bucket", func(t *testing.T) {
		t.Parallel()
		// "consumer" requires "producer.cap" which "producer" provides.
		// Topo must place producer before consumer regardless of
		// registration order.
		producer := &stubAnnCap{
			name: "producer", priority: priority.AnnotatorShape,
			provides: []string{"producer.cap"},
		}
		consumer := &stubAnnCap{
			name: "consumer", priority: priority.AnnotatorShape,
			requires: []string{"producer.cap"},
		}
		p, err := pipeline.New().
			WithFrontend(&stubFE{name: "fe"}).
			WithAnnotator(consumer).
			WithAnnotator(producer).
			WithBackend(&stubBE{name: "be"}).
			Build()
		assertNoError(t, err)
		got := names(p.Plan().Annotators)
		if !slices.Equal(got, []string{"producer", "consumer"}) {
			t.Fatalf("topo should place producer first; got %v", got)
		}
	})

	t.Run("alphabetical tie-break orders independent plugins", func(t *testing.T) {
		t.Parallel()
		// Three plugins in the same bucket with no dependencies —
		// the resolver should output them alphabetically.
		c := &stubAnnCap{name: "c", priority: priority.AnnotatorShape}
		a := &stubAnnCap{name: "a", priority: priority.AnnotatorShape}
		b := &stubAnnCap{name: "b", priority: priority.AnnotatorShape}
		p, err := pipeline.New().
			WithFrontend(&stubFE{name: "fe"}).
			WithAnnotator(c).
			WithAnnotator(a).
			WithAnnotator(b).
			WithBackend(&stubBE{name: "be"}).
			Build()
		assertNoError(t, err)
		got := names(p.Plan().Annotators)
		if !slices.Equal(got, []string{"a", "b", "c"}) {
			t.Fatalf("independent plugins should sort alphabetically; got %v", got)
		}
	})

	t.Run("self-Requires is silently ignored", func(t *testing.T) {
		t.Parallel()
		// A plugin that Requires its own Provides should still be
		// schedulable (the self-edge would otherwise create a cycle).
		self := &stubAnnCap{
			name: "self", priority: priority.AnnotatorShape,
			provides: []string{"x"},
			requires: []string{"x"},
		}
		_, err := pipeline.New().
			WithFrontend(&stubFE{name: "fe"}).
			WithAnnotator(self).
			WithBackend(&stubBE{name: "be"}).
			Build()
		assertNoError(t, err)
	})
}

func TestResolvePlan_Cycles(t *testing.T) {
	t.Parallel()

	t.Run("returns ErrCycle when a bucket has a Requires loop", func(t *testing.T) {
		t.Parallel()
		// a Provides "A" and Requires "B"; b Provides "B" and Requires "A".
		// Same bucket → cycle.
		a := &stubAnnCap{
			name: "a", priority: priority.AnnotatorShape,
			provides: []string{"A"}, requires: []string{"B"},
		}
		b := &stubAnnCap{
			name: "b", priority: priority.AnnotatorShape,
			provides: []string{"B"}, requires: []string{"A"},
		}
		_, err := pipeline.New().
			WithFrontend(&stubFE{name: "fe"}).
			WithAnnotator(a).
			WithAnnotator(b).
			WithBackend(&stubBE{name: "be"}).
			Build()
		if !errors.Is(err, pipeline.ErrCycle) {
			t.Fatalf("Build should return ErrCycle; got %v", err)
		}
		if !strings.Contains(err.Error(), "a") || !strings.Contains(err.Error(), "b") {
			t.Fatalf("cycle error should list participants; got %q", err.Error())
		}
	})

	t.Run("cross-bucket Requires do not cycle", func(t *testing.T) {
		t.Parallel()
		// a in bucket 200 Requires "later"; b in bucket 400 Provides "later"
		// and Requires "earlier"; a Provides "earlier".
		// Within-bucket alone: bucket 200 has only a (no resolution
		// needed since "later" is cross-bucket); bucket 400 has only
		// b (cross-bucket Requires ignored). No cycle.
		a := &stubAnnCap{
			name: "a", priority: priority.AnnotatorShape,
			provides: []string{"earlier"}, requires: []string{"later"},
		}
		b := &stubAnnCap{
			name: "b", priority: priority.AnnotatorValidation,
			provides: []string{"later"}, requires: []string{"earlier"},
		}
		_, err := pipeline.New().
			WithFrontend(&stubFE{name: "fe"}).
			WithAnnotator(a).
			WithAnnotator(b).
			WithBackend(&stubBE{name: "be"}).
			Build()
		assertNoError(t, err)
	})

	t.Run("returns ErrCycle for a generator-bucket loop", func(t *testing.T) {
		t.Parallel()
		a := &stubGenCap{
			name: "a", priority: priority.GeneratorComposition,
			provides: []string{"A"}, requires: []string{"B"},
		}
		b := &stubGenCap{
			name: "b", priority: priority.GeneratorComposition,
			provides: []string{"B"}, requires: []string{"A"},
		}
		_, err := pipeline.New().
			WithFrontend(&stubFE{name: "fe"}).
			WithGenerator(a).
			WithGenerator(b).
			WithBackend(&stubBE{name: "be"}).
			Build()
		if !errors.Is(err, pipeline.ErrCycle) {
			t.Fatalf("Build should return ErrCycle for a generator cycle; got %v", err)
		}
	})
}

func TestResolvePlan_DuplicateProvides(t *testing.T) {
	t.Parallel()

	t.Run("returns ErrDuplicateProvider when same-bucket plugins share a Provides name", func(t *testing.T) {
		t.Parallel()
		a := &stubAnnCap{
			name: "a", priority: priority.AnnotatorShape,
			provides: []string{"shared"},
		}
		b := &stubAnnCap{
			name: "b", priority: priority.AnnotatorShape,
			provides: []string{"shared"},
		}
		_, err := pipeline.New().
			WithFrontend(&stubFE{name: "fe"}).
			WithAnnotator(a).
			WithAnnotator(b).
			WithBackend(&stubBE{name: "be"}).
			Build()
		if !errors.Is(err, pipeline.ErrDuplicateProvider) {
			t.Fatalf("Build should return ErrDuplicateProvider; got %v", err)
		}
	})

	t.Run("two plugins in different buckets may claim the same Provides", func(t *testing.T) {
		t.Parallel()
		// Same name in different buckets is fine — Provides namespaces
		// are per-bucket.
		a := &stubAnnCap{
			name: "a", priority: priority.AnnotatorShape,
			provides: []string{"shared"},
		}
		b := &stubAnnCap{
			name: "b", priority: priority.AnnotatorValidation,
			provides: []string{"shared"},
		}
		_, err := pipeline.New().
			WithFrontend(&stubFE{name: "fe"}).
			WithAnnotator(a).
			WithAnnotator(b).
			WithBackend(&stubBE{name: "be"}).
			Build()
		assertNoError(t, err)
	})
}

func TestResolvePlan_Generators(t *testing.T) {
	t.Parallel()

	t.Run("generator buckets resolve the same way as annotators", func(t *testing.T) {
		t.Parallel()
		// Foundation < Composition < CrossCutting; register out of
		// order to confirm bucket sorting.
		cross := &stubGenCap{name: "cross", priority: priority.GeneratorCrossCutting}
		comp := &stubGenCap{name: "comp", priority: priority.GeneratorComposition}
		found := &stubGenCap{name: "found", priority: priority.GeneratorFoundation}
		p, err := pipeline.New().
			WithFrontend(&stubFE{name: "fe"}).
			WithGenerator(cross).
			WithGenerator(comp).
			WithGenerator(found).
			WithBackend(&stubBE{name: "be"}).
			Build()
		assertNoError(t, err)
		got := names(p.Plan().Generators)
		if !slices.Equal(got, []string{"found", "comp", "cross"}) {
			t.Fatalf("generator bucket order mismatch; got %v", got)
		}
	})
}

func TestResolvePlan_FrontendsPreserveRegistrationOrder(t *testing.T) {
	t.Parallel()

	t.Run("Frontends in the plan match registration order verbatim", func(t *testing.T) {
		t.Parallel()
		p, err := pipeline.New().
			WithFrontend(&stubFE{name: "second-registered"}).
			WithFrontend(&stubFE{name: "first-registered-but-named-second"}).
			WithBackend(&stubBE{name: "be"}).
			Build()
		assertNoError(t, err)
		got := []string{p.Plan().Frontends[0].Name(), p.Plan().Frontends[1].Name()}
		if got[0] != "second-registered" || got[1] != "first-registered-but-named-second" {
			t.Fatalf("frontends should preserve registration order; got %v", got)
		}
	})
}

func TestResolvePlan_BackendOnPlan(t *testing.T) {
	t.Parallel()

	t.Run("plan exposes the registered backend directly", func(t *testing.T) {
		t.Parallel()
		be := &stubBE{name: "be"}
		p, err := pipeline.New().
			WithFrontend(&stubFE{name: "fe"}).
			WithBackend(be).
			Build()
		assertNoError(t, err)
		if p.Plan().Backend != be {
			t.Fatalf("Plan.Backend should be the registered instance")
		}
	})
}
