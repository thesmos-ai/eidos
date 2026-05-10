// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package naming_test

import (
	"testing"

	"go.thesmos.sh/eidos/core/naming"
)

// styleCase is the shared shape for case-converter tables: a Caser
// (so individual rows can vary the initialism set), an input, and an
// expected output.
type styleCase struct {
	name  string
	caser *naming.Caser
	in    string
	want  string
}

// runStyleCases drives a single converter (selected by fn) over a
// slice of styleCase rows, each as its own subtest.
func runStyleCases(t *testing.T, fn func(*naming.Caser, string) string, cases []styleCase) {
	t.Helper()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			caser := tc.caser
			if caser == nil {
				caser = naming.Default()
			}
			assertEqualString(t, fn(caser, tc.in), tc.want)
		})
	}
}

func TestCaser_Pascal(t *testing.T) {
	t.Parallel()

	runStyleCases(
		t,
		func(c *naming.Caser, s string) string { return c.Pascal(s) },
		[]styleCase{
			{name: "empty input returns empty", in: "", want: ""},
			{name: "snake_case becomes PascalCase", in: "hello_world", want: "HelloWorld"},
			{name: "camelCase becomes PascalCase", in: "helloWorld", want: "HelloWorld"},
			{name: "already PascalCase stays unchanged", in: "HelloWorld", want: "HelloWorld"},
			{name: "known initialism is upper-cased", in: "url_path", want: "URLPath"},
			{
				name:  "unknown acronym is title-cased without initialisms",
				caser: naming.New(),
				in:    "url_path",
				want:  "UrlPath",
			},
			{name: "preserves all-upper acronym in mixed input", in: "HTTPServer", want: "HTTPServer"},
			{name: "trailing initialism preserved", in: "user_id", want: "UserID"},
			{name: "all-upper non-initialism word is preserved", caser: naming.New(), in: "FOO", want: "FOO"},
		},
	)
}

func TestCaser_Camel(t *testing.T) {
	t.Parallel()

	runStyleCases(
		t,
		func(c *naming.Caser, s string) string { return c.Camel(s) },
		[]styleCase{
			{name: "empty input returns empty", in: "", want: ""},
			{name: "snake_case becomes camelCase", in: "hello_world", want: "helloWorld"},
			{name: "PascalCase becomes camelCase", in: "HelloWorld", want: "helloWorld"},
			{name: "first word fully lower-cased even when initialism", in: "URL_path", want: "urlPath"},
			{name: "trailing initialism preserved", in: "user_id", want: "userID"},
		},
	)
}

func TestCaser_Snake(t *testing.T) {
	t.Parallel()

	runStyleCases(
		t,
		func(c *naming.Caser, s string) string { return c.Snake(s) },
		[]styleCase{
			{name: "empty input returns empty", in: "", want: ""},
			{name: "PascalCase becomes snake_case", in: "HelloWorld", want: "hello_world"},
			{name: "acronym run is lower-cased", in: "HTTPServer", want: "http_server"},
			{name: "idempotent on snake_case input", in: "hello_world", want: "hello_world"},
		},
	)
}

func TestCaser_ScreamingSnake(t *testing.T) {
	t.Parallel()

	runStyleCases(
		t,
		func(c *naming.Caser, s string) string { return c.ScreamingSnake(s) },
		[]styleCase{
			{name: "empty input returns empty", in: "", want: ""},
			{name: "PascalCase becomes SCREAMING_SNAKE", in: "HelloWorld", want: "HELLO_WORLD"},
			{name: "acronym run preserved upper", in: "HTTPServer", want: "HTTP_SERVER"},
		},
	)
}

func TestCaser_Kebab(t *testing.T) {
	t.Parallel()

	runStyleCases(
		t,
		func(c *naming.Caser, s string) string { return c.Kebab(s) },
		[]styleCase{
			{name: "empty input returns empty", in: "", want: ""},
			{name: "PascalCase becomes kebab-case", in: "HelloWorld", want: "hello-world"},
			{name: "snake_case becomes kebab-case", in: "hello_world", want: "hello-world"},
		},
	)
}

func TestCaser_ScreamingKebab(t *testing.T) {
	t.Parallel()

	runStyleCases(
		t,
		func(c *naming.Caser, s string) string { return c.ScreamingKebab(s) },
		[]styleCase{
			{name: "empty input returns empty", in: "", want: ""},
			{name: "PascalCase becomes SCREAMING-KEBAB", in: "HelloWorld", want: "HELLO-WORLD"},
		},
	)
}

func TestCaser_Dot(t *testing.T) {
	t.Parallel()

	runStyleCases(
		t,
		func(c *naming.Caser, s string) string { return c.Dot(s) },
		[]styleCase{
			{name: "empty input returns empty", in: "", want: ""},
			{name: "PascalCase becomes dot.case", in: "HelloWorld", want: "hello.world"},
		},
	)
}

func TestCaser_Title(t *testing.T) {
	t.Parallel()

	runStyleCases(
		t,
		func(c *naming.Caser, s string) string { return c.Title(s) },
		[]styleCase{
			{name: "empty input returns empty", in: "", want: ""},
			{
				name: "snake_case becomes Title Case with initialism preserved",
				in:   "user_id_fetcher",
				want: "User ID Fetcher",
			},
			{name: "camelCase becomes Title Case", in: "helloWorld", want: "Hello World"},
		},
	)
}

// TestPackageLevelShorthands covers the package-level façade: each
// shorthand routes to the same converter on the default Caser. This is
// not table-driven because each row tests a different function rather
// than the same function with different inputs.
func TestPackageLevelShorthands(t *testing.T) {
	t.Parallel()

	t.Run("Pascal", func(t *testing.T) {
		t.Parallel()
		assertEqualString(t, naming.Pascal("hello_world"), "HelloWorld")
	})
	t.Run("Camel", func(t *testing.T) {
		t.Parallel()
		assertEqualString(t, naming.Camel("hello_world"), "helloWorld")
	})
	t.Run("Snake", func(t *testing.T) {
		t.Parallel()
		assertEqualString(t, naming.Snake("HelloWorld"), "hello_world")
	})
	t.Run("ScreamingSnake", func(t *testing.T) {
		t.Parallel()
		assertEqualString(t, naming.ScreamingSnake("HelloWorld"), "HELLO_WORLD")
	})
	t.Run("Kebab", func(t *testing.T) {
		t.Parallel()
		assertEqualString(t, naming.Kebab("HelloWorld"), "hello-world")
	})
	t.Run("ScreamingKebab", func(t *testing.T) {
		t.Parallel()
		assertEqualString(t, naming.ScreamingKebab("HelloWorld"), "HELLO-WORLD")
	})
	t.Run("Dot", func(t *testing.T) {
		t.Parallel()
		assertEqualString(t, naming.Dot("HelloWorld"), "hello.world")
	})
	t.Run("Title", func(t *testing.T) {
		t.Parallel()
		assertEqualString(t, naming.Title("hello_world"), "Hello World")
	})
}
