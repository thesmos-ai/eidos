// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package pipeline

import (
	"fmt"
	"slices"

	"go.thesmos.sh/eidos/plugin"
	"go.thesmos.sh/eidos/priority"
)

// resolvePhase orders the supplied plugins into execution order.
// Plugins are grouped by [priority.Priority] (plugins that don't
// implement [plugin.CapabilityProvider] land in [priority.Default]),
// buckets are visited in ascending priority order, and within each
// bucket plugins are topo-sorted by Provides / Requires with
// alphabetical tie-break.
//
// Returns [ErrCycle] when a bucket's Requires graph cannot be
// linearised; [ErrDuplicateProvider] when two plugins in the same
// bucket Provide the same capability name.
//
// Cross-bucket Requires are intentionally not resolved (per the
// spec): a Requires that names a capability not Provided in the
// same bucket is silently ignored at this layer (verbose-mode
// diagnostics for unresolved requires are emitted by the caller).
func resolvePhase[T plugin.Plugin](plugins []T) ([]T, error) {
	if len(plugins) == 0 {
		return nil, nil
	}

	// Group by priority bucket.
	buckets := map[priority.Priority][]T{}
	for _, p := range plugins {
		buckets[pluginPriority(p)] = append(buckets[pluginPriority(p)], p)
	}

	// Sort buckets ascending by priority value.
	keys := make([]priority.Priority, 0, len(buckets))
	for k := range buckets {
		keys = append(keys, k)
	}
	slices.Sort(keys)

	// Topo sort each bucket and concatenate.
	out := make([]T, 0, len(plugins))
	for _, k := range keys {
		ordered, err := topoSortBucket(buckets[k])
		if err != nil {
			return nil, err
		}
		out = append(out, ordered...)
	}
	return out, nil
}

// pluginPriority returns the priority bucket p declares via
// [plugin.CapabilityProvider], or [priority.Default] when p does
// not implement the capability.
func pluginPriority(p plugin.Plugin) priority.Priority {
	if cp, ok := any(p).(plugin.CapabilityProvider); ok {
		return cp.Priority()
	}
	return priority.Default
}

// pluginCapability returns the [plugin.CapabilityProvider] facet of
// p along with true when p implements it; otherwise (nil, false).
func pluginCapability(p plugin.Plugin) (plugin.CapabilityProvider, bool) {
	cp, ok := any(p).(plugin.CapabilityProvider)
	return cp, ok
}

// topoSortBucket runs Kahn's algorithm over the plugins in a single
// priority bucket. Edges go from the plugin that Provides a
// capability to every plugin in the same bucket that Requires it.
// The "ready" frontier is kept sorted alphabetically by plugin
// name so the produced order is deterministic across runs.
//
// Returns [ErrCycle] when iteration completes with one or more
// plugins still carrying a non-zero in-degree. Returns
// [ErrDuplicateProvider] when two plugins claim the same Provides
// name in this bucket.
func topoSortBucket[T plugin.Plugin](plugins []T) ([]T, error) {
	provides, err := buildProvidesIndex(plugins)
	if err != nil {
		return nil, err
	}

	byName := make(map[string]T, len(plugins))
	for _, p := range plugins {
		byName[p.Name()] = p
	}

	inDegree := make(map[string]int, len(plugins))
	dependents := make(map[string][]string, len(plugins))
	for _, p := range plugins {
		inDegree[p.Name()] = 0
	}
	for _, p := range plugins {
		cp, ok := pluginCapability(p)
		if !ok {
			continue
		}
		for _, req := range cp.Requires() {
			provider, found := provides[req]
			if !found {
				// Cross-bucket or simply absent — silently ignored
				// at this layer.
				continue
			}
			if provider.Name() == p.Name() {
				// Self-Requires; ignore so the plugin is still
				// schedulable.
				continue
			}
			dependents[provider.Name()] = append(dependents[provider.Name()], p.Name())
			inDegree[p.Name()]++
		}
	}

	ready := make([]string, 0, len(plugins))
	for name, deg := range inDegree {
		if deg == 0 {
			ready = append(ready, name)
		}
	}
	slices.Sort(ready)

	out := make([]T, 0, len(plugins))
	for len(ready) > 0 {
		name := ready[0]
		ready = ready[1:]
		out = append(out, byName[name])
		for _, dep := range dependents[name] {
			inDegree[dep]--
			if inDegree[dep] == 0 {
				ready = insertSorted(ready, dep)
			}
		}
	}

	if len(out) < len(plugins) {
		cyclic := make([]string, 0, len(plugins)-len(out))
		for name, deg := range inDegree {
			if deg > 0 {
				cyclic = append(cyclic, name)
			}
		}
		slices.Sort(cyclic)
		return nil, fmt.Errorf("%w: %v", ErrCycle, cyclic)
	}
	return out, nil
}

// buildProvidesIndex maps each capability name declared by any
// plugin in the bucket to the producing plugin. Returns
// [ErrDuplicateProvider] when two plugins claim the same name.
func buildProvidesIndex[T plugin.Plugin](plugins []T) (map[string]T, error) {
	provides := map[string]T{}
	for _, p := range plugins {
		cp, ok := pluginCapability(p)
		if !ok {
			continue
		}
		for _, name := range cp.Provides() {
			if existing, dup := provides[name]; dup {
				return nil, fmt.Errorf("%w: %q claimed by %s and %s",
					ErrDuplicateProvider, name, existing.Name(), p.Name())
			}
			provides[name] = p
		}
	}
	return provides, nil
}

// insertSorted inserts v into the sorted slice s and returns the
// resulting slice. Equivalent to "append + sort" but linear instead
// of O(n log n) per insert; the topo frontier rarely exceeds a few
// elements so the constant factor matters in tight loops.
func insertSorted(s []string, v string) []string {
	i, _ := slices.BinarySearch(s, v)
	s = append(s, "")
	copy(s[i+1:], s[i:])
	s[i] = v
	return s
}
