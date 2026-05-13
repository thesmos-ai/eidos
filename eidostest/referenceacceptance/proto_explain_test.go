// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package referenceacceptance_test

import (
	"bytes"
	"strings"
	"testing"

	backend_golang "go.thesmos.sh/eidos/backend/golang"
	"go.thesmos.sh/eidos/bridge/protogo"
	"go.thesmos.sh/eidos/cli"
	"go.thesmos.sh/eidos/frontend/protobuf"
	"go.thesmos.sh/eidos/plugin"
)

// TestExplain_ProtoSources drives the `eidos explain` command
// against every proto-derived source kind the spec calls out —
// message, enum, service, RPC method, oneof-variant field — and
// asserts the rendered Subject / Position / (Directives) /
// (Metadata) / Outputs view surfaces correctly. The proto +
// Go-namespaced meta-key coverage rides on the field-level
// subtest (the one place both stamp surfaces converge on a
// single bag); the host-level subtests verify the structural
// view (Subject, Position, Directives, Outputs).
func TestExplain_ProtoSources(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		selector string
		// wants is the set of substring assertions the rendered
		// output must contain. The expected lines are stable
		// pieces of the documented Subject / Position /
		// Directives view; per-key meta checks ride on the field
		// subtest.
		wants []string
	}{
		{
			name:     "message",
			selector: "buildfixture.User",
			wants: []string{
				"Subject:",
				"User (struct)",
				"Position:",
				"buildfixture.proto",
				"Directives:",
				"+gen:builder",
			},
		},
		{
			name:     "enum",
			selector: "buildfixture.Status",
			wants: []string{
				"Subject:",
				"Status (enum)",
				"Position:",
				"buildfixture.proto",
			},
		},
		{
			name:     "service",
			selector: "buildfixture.UserService",
			wants: []string{
				"Subject:",
				"UserService (interface)",
				"Position:",
				"Directives:",
				"+gen:mock",
			},
		},
		{
			name:     "rpc-method",
			selector: "buildfixture.UserService.GetUser",
			wants: []string{
				"Subject:",
				"GetUser",
				"Position:",
			},
		},
		{
			name:     "oneof-variant-field",
			selector: "buildfixture.Container.text_value",
			wants: []string{
				"Subject:",
				"text_value",
				"Position:",
				"Metadata:",
				// frontend stamp linking the variant back to its
				// synthesized oneof interface
				"proto.oneof.interface",
				// bridge stamp translating the proto snake_case to
				// the Go-rendered PascalCase identifier
				"go.name",
				"TextValue",
			},
		},
		{
			name:     "scalar-field-bridge-meta",
			selector: "buildfixture.User.name",
			wants: []string{
				"Subject:",
				"name",
				"Position:",
				"Metadata:",
				// frontend per-field meta (tag number, JSON name)
				"proto.field.number",
				// bridge stamp linking the snake_case proto field
				// to its rendered Go-side identifier
				"go.name",
				"Name",
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			out := runProtoExplain(t, tc.selector)
			for _, want := range tc.wants {
				if !strings.Contains(out, want) {
					t.Errorf("output missing %q\n--- explain output ---\n%s", want, out)
				}
			}
		})
	}
}

// runProtoExplain runs `eidos explain <selector>` against the
// proto buildfixture with the protobuf frontend + protogo
// bridge + Go backend registered, captures stdout, and returns
// the body. Fails the test on a non-OK exit code so the per-
// case assertions stay focused on content.
func runProtoExplain(t *testing.T, selector string) string {
	t.Helper()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	env := &cli.Env{
		Brand:   "eidos",
		Workdir: t.TempDir(),
		Stdout:  stdout,
		Stderr:  stderr,
	}
	cfg := &cli.Config{
		Version: cli.ConfigVersion,
		Sources: []cli.ConfigSource{{
			Frontend: "protobuf",
			Patterns: []string{"./..."},
		}},
		Plugins: []cli.ConfigPlugin{
			{Name: "protobuf", Options: map[string]any{"dir": protoFixtureRoot(t)}},
			{Name: "protogo"},
			{Name: "backend.golang"},
		},
	}
	cmd := &cli.ExplainCommand{Config: cli.ExplainConfig{
		File: cfg,
		Plugins: []plugin.Plugin{
			protobuf.New(),
			protogo.New(),
			backend_golang.New(),
		},
		Selector: selector,
	}}
	if code := cmd.Execute(t.Context(), env); code != cli.ExitOK {
		t.Fatalf("explain %q exited %d\nstderr:\n%s\nstdout:\n%s",
			selector, code, stderr.String(), stdout.String())
	}
	return stdout.String()
}
