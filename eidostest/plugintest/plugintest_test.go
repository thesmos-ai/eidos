// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package plugintest_test

import (
	"testing"

	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/eidostest/plugintest"
	"go.thesmos.sh/eidos/plugin"
)

// TestRunSuite_PassesForWellFormedPlugin pins the happy path of
// the conformance suite: a plugin satisfying every documented
// contract clears every check. The fixture covers every optional
// role interface the suite probes so a single struct exercises
// the full surface.
func TestRunSuite_PassesForWellFormedPlugin(t *testing.T) {
	t.Parallel()

	t.Run("plugin satisfying every contract clears every check", func(t *testing.T) {
		t.Parallel()
		plugintest.RunSuite(t, &fixturePlugin{
			name:     "fixture",
			provides: []string{"cap.one"},
			requires: []string{"cap.zero"},
			schemas: []directive.Schema{
				directive.NewSchema("foo").On("Struct").Build(),
				directive.NewSchema("bar").On("Interface").Build(),
			},
			version: "v1.0.0",
		})
	})
}

// fixturePlugin is the fixture every positive-path subtest
// instantiates. The fields cover every optional role interface
// the suite probes so a single struct satisfies the full
// contract surface.
type fixturePlugin struct {
	name     string
	provides []string
	requires []string
	schemas  []directive.Schema
	version  string
}

// Name returns the configured name. Stable across calls because
// it's a field read.
func (p *fixturePlugin) Name() string { return p.name }

// Generate satisfies [plugin.Generator] so the role probe sees
// at least one role.
func (*fixturePlugin) Generate(_ *plugin.GeneratorContext) error { return nil }

// Provides satisfies [plugin.CapabilityProvider].
func (p *fixturePlugin) Provides() []string { return p.provides }

// Requires satisfies [plugin.CapabilityProvider].
func (p *fixturePlugin) Requires() []string { return p.requires }

// Directives satisfies [plugin.DirectiveProvider].
func (p *fixturePlugin) Directives() []directive.Schema { return p.schemas }

// Version satisfies [plugin.Versioned].
func (p *fixturePlugin) Version() string { return p.version }
