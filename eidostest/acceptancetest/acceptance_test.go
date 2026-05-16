// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package acceptancetest_test

import (
	"strings"
	"testing"

	"go.thesmos.sh/eidos/eidostest/acceptancetest"
)

// TestVersionSubcommand pins the binary's basic startup +
// version-listing path. A clean exit + the documented plugin
// names in the rendered output proves the binary wires every
// in-tree plugin and the version command's listing path works
// end-to-end.
func TestVersionSubcommand(t *testing.T) {
	t.Parallel()
	res := acceptancetest.RunCmd(t, "", "version")
	if res.ExitCode != 0 {
		t.Fatalf("version exit code = %d, want 0\nstderr:\n%s", res.ExitCode, res.Stderr)
	}
	if !strings.Contains(res.Stdout, "eidos-reference") {
		t.Errorf("version output missing brand identifier; got:\n%s", res.Stdout)
	}
	// Every in-tree plugin appears in the listing. Spot-check the
	// canonical names; a regression that drops one would surface
	// here before the user sees it.
	for _, want := range []string{
		"golang",       // backend
		"protobuf",     // proto frontend
		"protogo",      // bridge
		"shape-writer", // annotator
		"repogen",      // foundation generator
		"buildergen",   // foundation generator
		"mockgen",      // composition generator
		"audit-weaver", // cross-cutter
		"debug-weaver", // cross-cutter
		"registry-gen", // cross-cutter
	} {
		if !strings.Contains(res.Stdout, want) {
			t.Errorf("version output missing plugin %q; got:\n%s", want, res.Stdout)
		}
	}
}

// TestHelpSubcommand pins the top-level help banner — when the
// binary is invoked with no subcommand (or `help` / `-h` /
// `--help`), it prints the usage banner and exits cleanly.
func TestHelpSubcommand(t *testing.T) {
	t.Parallel()

	t.Run("no arguments prints usage", func(t *testing.T) {
		t.Parallel()
		res := acceptancetest.RunCmd(t, "")
		if res.ExitCode != 0 {
			t.Fatalf("bare invocation exit code = %d, want 0", res.ExitCode)
		}
		if !strings.Contains(res.Stdout, "Usage:") {
			t.Errorf("bare invocation should print usage banner; got:\n%s", res.Stdout)
		}
	})

	t.Run("--help prints usage", func(t *testing.T) {
		t.Parallel()
		res := acceptancetest.RunCmd(t, "", "--help")
		if res.ExitCode != 0 {
			t.Fatalf("--help exit code = %d, want 0", res.ExitCode)
		}
		if !strings.Contains(res.Stdout, "Usage:") {
			t.Errorf("--help should print usage banner; got:\n%s", res.Stdout)
		}
	})

	t.Run("help keyword prints usage", func(t *testing.T) {
		t.Parallel()
		res := acceptancetest.RunCmd(t, "", "help")
		if res.ExitCode != 0 {
			t.Fatalf("help keyword exit code = %d, want 0", res.ExitCode)
		}
		if !strings.Contains(res.Stdout, "Usage:") {
			t.Errorf("help keyword should print usage banner; got:\n%s", res.Stdout)
		}
	})
}

// TestUnknownSubcommand pins the user-error path: an invalid
// subcommand exits with [cli.ExitUserError] (1) and writes the
// usage banner to stderr.
func TestUnknownSubcommand(t *testing.T) {
	t.Parallel()
	res := acceptancetest.RunCmd(t, "", "no-such-command")
	if res.ExitCode != 1 {
		t.Fatalf("unknown subcommand exit code = %d, want 1 (ExitUserError)", res.ExitCode)
	}
	if !strings.Contains(res.Stderr, "unknown command") {
		t.Errorf("unknown subcommand stderr should mention 'unknown command'; got:\n%s", res.Stderr)
	}
}

// TestPlanSubcommand pins the `plan` subcommand: an empty
// project (no source patterns, no config) prints a resolved
// plan and exits cleanly. The plan output lists the plugins in
// their canonical capability-topo order so test pins on
// specific ordering survive plugin additions that don't change
// the topo.
func TestPlanSubcommand(t *testing.T) {
	t.Parallel()
	res := acceptancetest.RunCmd(t, t.TempDir(), "plan")
	if res.ExitCode != 0 {
		t.Fatalf("plan exit code = %d, want 0\nstderr:\n%s", res.ExitCode, res.Stderr)
	}
	// Output shape is documented by [cli.PlanCommand]; a
	// regression that breaks plan-rendering surfaces as missing
	// header or empty stdout.
	if res.Stdout == "" {
		t.Errorf("plan should print a non-empty plan; got empty stdout")
	}
}
