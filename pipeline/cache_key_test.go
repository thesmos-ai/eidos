// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package pipeline_test

import (
	"sync"
	"testing"

	"go.thesmos.sh/eidos/pipeline"
	"go.thesmos.sh/eidos/sink"
)

// Test list for the routing-aware cache-key composition:
//
//  1. Same routing inputs → identical per-plugin cache keys across
//     two runs.
//  2. -target flip → every per-plugin key differs (run-wide scope).
//  3. -o flip → every per-plugin key differs (run-wide scope).
//  4. -p flip → every per-plugin key differs because the resolved
//     LayoutPolicy.Package changes for every plugin (CLI is a
//     run-wide layer).
//  5. -layout flip → every per-plugin key differs.
//  6. -output-dir flip → every per-plugin key differs.
//  7. Project-level config flip → every per-plugin key differs.
//  8. Per-plugin output override flip → only the affected plugin's
//     key differs; other plugins' keys stay stable.

// recordingCache captures every Put key the pipeline writes. Used
// by the cache-key composition tests to assert keys differ across
// runs that flip routing inputs.
type recordingCache struct {
	mu   sync.Mutex
	keys []string
}

func (*recordingCache) Get(string) ([]byte, bool) { return nil, false }

func (c *recordingCache) Put(key string, _ []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.keys = append(c.keys, key)
	return nil
}

// keyFor returns the recorded cache key whose prefix matches
// `plugin:<name>:`. Used by assertions that target one plugin's
// key out of the full Put log.
func (c *recordingCache) keyFor(t *testing.T, pluginName string) string {
	t.Helper()
	c.mu.Lock()
	defer c.mu.Unlock()
	prefix := "plugin:" + pluginName + ":"
	for _, k := range c.keys {
		if hasPrefix(k, prefix) {
			return k
		}
	}
	t.Fatalf("no recorded cache key for plugin %q; recorded keys: %v", pluginName, c.keys)
	return ""
}

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

// runForCacheKeys constructs a one-plugin pipeline using the
// supplied routing configurator, runs it once, and returns the
// recorded cache key for the named generator. The pipeline uses
// a stub frontend / backend so the per-plugin attribution is the
// only routing surface under test.
func runForCacheKeys(t *testing.T, configure func(b *pipeline.Builder)) (string, string, string) {
	t.Helper()
	c := &recordingCache{}
	b := pipeline.New().
		WithFrontend(&stubFE{name: "fe"}).
		WithAnnotator(&stubAnn{name: "ann"}).
		WithGenerator(&stubGen{name: "gen"}).
		WithBackend(&stubBE{name: "be"}).
		WithSink(sink.NewMemory()).
		WithCache(c)
	if configure != nil {
		configure(b)
	}
	p, err := b.Build()
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if err := p.Run(t.Context(), "x"); err != nil {
		t.Fatalf("Run: %v", err)
	}
	return c.keyFor(t, "ann"), c.keyFor(t, "gen"), c.keyFor(t, "be")
}

// TestCacheKey_Identical_AcrossRuns pins the baseline: two runs
// against identical routing inputs record identical per-plugin
// cache keys. Without this property cache invalidation tests are
// meaningless — every flip would trivially produce different keys.
func TestCacheKey_Identical_AcrossRuns(t *testing.T) {
	t.Parallel()

	t.Run("identical inputs produce identical keys", func(t *testing.T) {
		t.Parallel()
		annA, genA, beA := runForCacheKeys(t, nil)
		annB, genB, beB := runForCacheKeys(t, nil)
		if annA != annB || genA != genB || beA != beB {
			t.Fatalf("keys differ across identical runs:\n run A: ann=%q gen=%q be=%q\n run B: ann=%q gen=%q be=%q",
				annA, genA, beA, annB, genB, beB)
		}
	})
}

// TestCacheKey_RoutingFlips pins the invalidation contract: every
// routing input flip changes every affected per-plugin cache key.
// Run-wide inputs (-target, -o) flip every plugin's key.
// Per-plugin-merge inputs (-p, -layout, -output-dir, project
// layer) also flip every key today since they enter the resolved
// LayoutPolicy uniformly across all plugins (the per-plugin
// layer's narrowing-to-one-plugin behaviour is exercised by its
// dedicated subtest below).
func TestCacheKey_RoutingFlips(t *testing.T) {
	t.Parallel()

	baseAnn, baseGen, baseBE := runForCacheKeys(t, nil)

	flips := []struct {
		name      string
		configure func(b *pipeline.Builder)
	}{
		{"target", func(b *pipeline.Builder) { b.WithTargetSymbol("Foo") }},
		{"output filename", func(b *pipeline.Builder) {
			b.WithTargetSymbol("Foo").WithOutputFilename("x.go")
		}},
		{"output package", func(b *pipeline.Builder) { b.WithOutputPackage("gen") }},
		{"layout", func(b *pipeline.Builder) {
			b.WithOutputLayout(pipeline.LayoutCentralised).WithOutputPackage("gen")
		}},
		{"output dir", func(b *pipeline.Builder) {
			b.WithOutputLayout(pipeline.LayoutCentralised).WithOutputPackage("gen").WithOutputDir("x")
		}},
		{"project layer package", func(b *pipeline.Builder) {
			b.WithProjectOutput("", "gen", "")
		}},
	}

	for _, tc := range flips {
		t.Run(tc.name+" flip invalidates every plugin's cache key", func(t *testing.T) {
			t.Parallel()
			ann, gen, be := runForCacheKeys(t, tc.configure)
			if ann == baseAnn {
				t.Errorf("ann key unchanged after %s flip: %q", tc.name, ann)
			}
			if gen == baseGen {
				t.Errorf("gen key unchanged after %s flip: %q", tc.name, gen)
			}
			if be == baseBE {
				t.Errorf("be key unchanged after %s flip: %q", tc.name, be)
			}
		})
	}
}

// TestCacheKey_PerPluginOverride pins the narrowing semantic: a
// per-plugin output override changes only the affected plugin's
// cache key. Other plugins resolve the same project + CLI merge
// and so produce identical keys to the baseline.
func TestCacheKey_PerPluginOverride(t *testing.T) {
	t.Parallel()

	t.Run("per-plugin override flips only the targeted plugin's key", func(t *testing.T) {
		t.Parallel()
		baseAnn, baseGen, baseBE := runForCacheKeys(t, nil)
		ann, gen, be := runForCacheKeys(t, func(b *pipeline.Builder) {
			b.WithPluginOutput("gen", "", "mocks", "")
		})
		if gen == baseGen {
			t.Errorf("gen key unchanged after per-plugin override: %q", gen)
		}
		if ann != baseAnn {
			t.Errorf("ann key changed after gen-only override: %q vs %q", ann, baseAnn)
		}
		if be != baseBE {
			t.Errorf("be key changed after gen-only override: %q vs %q", be, baseBE)
		}
	})
}
