// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package naming_test

import (
	"testing"

	"go.thesmos.sh/eidos/core/naming"
)

func TestCaser_Words(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   string
		want []string
	}{
		{"empty input returns nil", "", nil},
		{"separator-only input returns nil", "___---...", nil},
		{"single lowercase word", "hello", []string{"hello"}},
		{"camelCase splits at lower-to-upper", "helloWorld", []string{"hello", "World"}},
		{"PascalCase splits into multiple words", "HelloWorld", []string{"Hello", "World"}},
		{"acronym preceding word splits at boundary", "HTTPServer", []string{"HTTP", "Server"}},
		{"trailing acronym stays whole", "userID", []string{"user", "ID"}},
		{"all-uppercase identifier stays whole", "HTTP", []string{"HTTP"}},
		{"snake_case splits at underscore", "hello_world", []string{"hello", "world"}},
		{"kebab-case splits at hyphen", "hello-world", []string{"hello", "world"}},
		{"dot.case splits at dot", "hello.world", []string{"hello", "world"}},
		{"space splits at space", "hello world", []string{"hello", "world"}},
		{"tab splits at tab", "hello\tworld", []string{"hello", "world"}},
		{"forward slash splits", "hello/world", []string{"hello", "world"}},
		{"repeated mixed separators collapse", "__hello-_-world__", []string{"hello", "world"}},
		{"digits attach to surrounding word", "Version2", []string{"Version2"}},
		{"complex mixed input", "URLPath_v2 helloWorld", []string{"URL", "Path", "v2", "hello", "World"}},
	}

	c := naming.Default()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assertEqualSlices(t, c.Words(tc.in), tc.want)
		})
	}
}

func TestWords_packageLevel(t *testing.T) {
	t.Parallel()
	assertEqualSlices(t, naming.Words("HelloWorld"), []string{"Hello", "World"})
}
