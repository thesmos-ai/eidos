// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package cli_test

import (
	"flag"
	"reflect"
	"testing"

	"go.thesmos.sh/eidos/cli"
	"go.thesmos.sh/eidos/pipeline"
)

// assertRoutingFlagsParse drives the standard routing-flag input
// set through the supplied FlagSet and verifies the parsed values
// land on target. Used by every routing-consuming command's
// RegisterFlags test so the per-command surface stays consistent
// with the [cli.RoutingFlags] contract.
func assertRoutingFlagsParse(t *testing.T, fs *flag.FlagSet, target *cli.RoutingFlags) {
	t.Helper()
	args := []string{
		"-" + cli.FlagTarget, "Article",
		"-" + cli.FlagOutput, "article_gen.go",
		"-" + cli.FlagPackage, "generated",
		"-" + cli.FlagLayout, pipeline.LayoutCentralised,
		"-" + cli.FlagOutputDir, "internal/gen",
	}
	if err := fs.Parse(args); err != nil {
		t.Fatalf("Parse: %v", err)
	}
	want := cli.RoutingFlags{
		Target:    "Article",
		Output:    []string{"article_gen.go"},
		Package:   "generated",
		Layout:    pipeline.LayoutCentralised,
		OutputDir: "internal/gen",
	}
	if !reflect.DeepEqual(*target, want) {
		t.Fatalf("RoutingFlags = %+v, want %+v", *target, want)
	}
}
