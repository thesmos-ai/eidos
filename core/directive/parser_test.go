// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package directive_test

import (
	"errors"
	"testing"

	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/core/position"
)

func TestNewParser(t *testing.T) {
	t.Parallel()

	t.Run("returns a parser exposing the configured prefix", func(t *testing.T) {
		t.Parallel()
		p, err := directive.NewParser("gen")
		assertNoError(t, err, "NewParser")
		assertEqualString(t, p.Prefix(), "gen")
	})

	t.Run("rejects an empty prefix with ErrInvalidPrefix", func(t *testing.T) {
		t.Parallel()
		_, err := directive.NewParser("")
		if !errors.Is(err, directive.ErrInvalidPrefix) {
			t.Fatalf("err = %v, want ErrInvalidPrefix", err)
		}
	})

	t.Run("rejects prefixes containing reserved characters", func(t *testing.T) {
		t.Parallel()
		cases := []string{":", "gen:", "+gen", "-gen", "ge n", "gen\t"}
		for _, in := range cases {
			t.Run(in, func(t *testing.T) {
				t.Parallel()
				_, err := directive.NewParser(in)
				if !errors.Is(err, directive.ErrInvalidPrefix) {
					t.Fatalf("NewParser(%q) err = %v, want ErrInvalidPrefix", in, err)
				}
			})
		}
	})
}

func TestParser_Parse(t *testing.T) {
	t.Parallel()

	parser, err := directive.NewParser("gen")
	assertNoError(t, err, "NewParser")
	pos := position.At("a.go", 1, 1)

	t.Run("parses the neutral form", func(t *testing.T) {
		t.Parallel()
		d, err := parser.Parse("gen:mock", pos)
		assertNoError(t, err, "Parse")
		if d.Name != "mock" {
			t.Fatalf("Name = %q, want mock", d.Name)
		}
		if d.Negated {
			t.Fatalf("Negated should be false for the neutral form")
		}
	})

	t.Run("parses the explicit set form", func(t *testing.T) {
		t.Parallel()
		d, err := parser.Parse("+gen:mock", pos)
		assertNoError(t, err, "Parse")
		if d.Negated {
			t.Fatalf("Negated should be false for the +gen: form")
		}
	})

	t.Run("parses the negated form", func(t *testing.T) {
		t.Parallel()
		d, err := parser.Parse("-gen:mock", pos)
		assertNoError(t, err, "Parse")
		if !d.Negated {
			t.Fatalf("Negated should be true for the -gen: form")
		}
	})

	t.Run("populates positional args", func(t *testing.T) {
		t.Parallel()
		d, err := parser.Parse("gen:shape writer paginator", pos)
		assertNoError(t, err, "Parse")
		if len(d.Args) != 2 || d.Args[0] != "writer" || d.Args[1] != "paginator" {
			t.Fatalf("Args = %v", d.Args)
		}
	})

	t.Run("populates KV args from key=value pairs", func(t *testing.T) {
		t.Parallel()
		d, err := parser.Parse("gen:mock target=Repository out=mocks/", pos)
		assertNoError(t, err, "Parse")
		if d.Value("target") != "Repository" {
			t.Fatalf("target = %q", d.Value("target"))
		}
		if d.Value("out") != "mocks/" {
			t.Fatalf("out = %q", d.Value("out"))
		}
	})

	t.Run("mixes positional and KV in any order", func(t *testing.T) {
		t.Parallel()
		d, err := parser.Parse("gen:shape writer method=Write buffer=buf", pos)
		assertNoError(t, err, "Parse")
		if d.Arg(0) != "writer" || d.Value("method") != "Write" || d.Value("buffer") != "buf" {
			t.Fatalf("parse mismatch: args=%v kv=%v", d.Args, d.KV)
		}
	})

	t.Run("supports quoted string values with embedded spaces", func(t *testing.T) {
		t.Parallel()
		d, err := parser.Parse(`gen:foo desc="hello world"`, pos)
		assertNoError(t, err, "Parse")
		assertEqualString(t, d.Value("desc"), "hello world")
	})

	t.Run("supports escape sequences in quoted values", func(t *testing.T) {
		t.Parallel()
		d, err := parser.Parse(`gen:foo desc="say \"hi\" then \\ and a \n line"`, pos)
		assertNoError(t, err, "Parse")
		assertEqualString(t, d.Value("desc"), "say \"hi\" then \\ and a \n line")
	})

	t.Run("preserves tab escape in quoted values", func(t *testing.T) {
		t.Parallel()
		d, err := parser.Parse(`gen:foo desc="col1\tcol2"`, pos)
		assertNoError(t, err, "Parse")
		assertEqualString(t, d.Value("desc"), "col1\tcol2")
	})

	t.Run("records the directive position", func(t *testing.T) {
		t.Parallel()
		d, err := parser.Parse("gen:mock", pos)
		assertNoError(t, err, "Parse")
		if d.Pos != pos {
			t.Fatalf("Pos = %+v, want %+v", d.Pos, pos)
		}
	})

	t.Run("records the raw body after stripping the prefix", func(t *testing.T) {
		t.Parallel()
		d, err := parser.Parse("gen:mock target=Repo", pos)
		assertNoError(t, err, "Parse")
		assertEqualString(t, d.Raw, "mock target=Repo")
	})

	t.Run("rejects input without a recognised prefix", func(t *testing.T) {
		t.Parallel()
		_, err := parser.Parse("nope:mock", pos)
		if !errors.Is(err, directive.ErrMalformedDirective) {
			t.Fatalf("err = %v, want ErrMalformedDirective", err)
		}
	})

	t.Run("rejects an empty directive name", func(t *testing.T) {
		t.Parallel()
		_, err := parser.Parse("gen:", pos)
		if !errors.Is(err, directive.ErrMalformedDirective) {
			t.Fatalf("err = %v, want ErrMalformedDirective", err)
		}
	})

	t.Run("rejects a name that starts with a non-letter", func(t *testing.T) {
		t.Parallel()
		_, err := parser.Parse("gen:9mock", pos)
		if !errors.Is(err, directive.ErrMalformedDirective) {
			t.Fatalf("err = %v, want ErrMalformedDirective", err)
		}
	})

	t.Run("rejects a name that contains a non-name rune after the first byte", func(t *testing.T) {
		t.Parallel()
		_, err := parser.Parse("gen:mock.bad", pos)
		if !errors.Is(err, directive.ErrMalformedDirective) {
			t.Fatalf("err = %v, want ErrMalformedDirective", err)
		}
	})

	t.Run("rejects an unterminated quoted value", func(t *testing.T) {
		t.Parallel()
		_, err := parser.Parse(`gen:foo desc="hello`, pos)
		if !errors.Is(err, directive.ErrMalformedDirective) {
			t.Fatalf("err = %v, want ErrMalformedDirective", err)
		}
	})

	t.Run("rejects an unknown escape in a quoted value", func(t *testing.T) {
		t.Parallel()
		_, err := parser.Parse(`gen:foo desc="oops \x"`, pos)
		if !errors.Is(err, directive.ErrMalformedDirective) {
			t.Fatalf("err = %v, want ErrMalformedDirective", err)
		}
	})

	t.Run("rejects an unterminated escape inside quotes", func(t *testing.T) {
		t.Parallel()
		_, err := parser.Parse(`gen:foo desc="oops \`, pos)
		if !errors.Is(err, directive.ErrMalformedDirective) {
			t.Fatalf("err = %v, want ErrMalformedDirective", err)
		}
	})

	t.Run("tolerates leading whitespace before the prefix", func(t *testing.T) {
		t.Parallel()
		d, err := parser.Parse("   gen:mock", pos)
		assertNoError(t, err, "Parse")
		if d.Name != "mock" {
			t.Fatalf("Name = %q, want mock", d.Name)
		}
	})
}

func TestParser_Parse_CustomPrefix(t *testing.T) {
	t.Parallel()

	t.Run("recognises a non-default prefix", func(t *testing.T) {
		t.Parallel()
		parser, err := directive.NewParser("testkit")
		assertNoError(t, err, "NewParser")
		d, err := parser.Parse("testkit:fixture target=DB", position.At("a.go", 1, 1))
		assertNoError(t, err, "Parse")
		if d.Name != "fixture" || d.Value("target") != "DB" {
			t.Fatalf("custom-prefix parse mismatch: %+v", d)
		}
	})

	t.Run("rejects the default prefix when configured for another", func(t *testing.T) {
		t.Parallel()
		parser, err := directive.NewParser("testkit")
		assertNoError(t, err, "NewParser")
		_, err = parser.Parse("gen:mock", position.At("a.go", 1, 1))
		if !errors.Is(err, directive.ErrMalformedDirective) {
			t.Fatalf("err = %v, want ErrMalformedDirective", err)
		}
	})
}

func TestParser_ParseComment(t *testing.T) {
	t.Parallel()

	parser, err := directive.NewParser("gen")
	assertNoError(t, err, "NewParser")
	pos := position.At("a.go", 1, 1)

	t.Run("strips the // line-comment marker", func(t *testing.T) {
		t.Parallel()
		d, err := parser.ParseComment("//gen:mock", pos)
		assertNoError(t, err, "ParseComment")
		if d == nil || d.Name != "mock" {
			t.Fatalf("ParseComment returned %+v", d)
		}
	})

	t.Run("tolerates whitespace between marker and prefix", func(t *testing.T) {
		t.Parallel()
		d, err := parser.ParseComment("// gen:mock", pos)
		assertNoError(t, err, "ParseComment")
		if d == nil || d.Name != "mock" {
			t.Fatalf("ParseComment returned %+v", d)
		}
	})

	t.Run("strips block-comment markers", func(t *testing.T) {
		t.Parallel()
		d, err := parser.ParseComment("/* +gen:mock target=Repo */", pos)
		assertNoError(t, err, "ParseComment")
		if d == nil || d.Name != "mock" || d.Value("target") != "Repo" {
			t.Fatalf("ParseComment returned %+v", d)
		}
	})

	t.Run("returns nil, nil for empty comments", func(t *testing.T) {
		t.Parallel()
		d, err := parser.ParseComment("// ", pos)
		assertNoError(t, err, "ParseComment")
		if d != nil {
			t.Fatalf("empty comment should return nil; got %+v", d)
		}
	})

	t.Run("returns nil, nil for non-directive comments", func(t *testing.T) {
		t.Parallel()
		d, err := parser.ParseComment("// regular comment", pos)
		assertNoError(t, err, "ParseComment")
		if d != nil {
			t.Fatalf("non-directive comment should return nil; got %+v", d)
		}
	})

	t.Run("propagates errors from a malformed directive comment", func(t *testing.T) {
		t.Parallel()
		_, err := parser.ParseComment("// gen:", pos)
		if !errors.Is(err, directive.ErrMalformedDirective) {
			t.Fatalf("err = %v, want ErrMalformedDirective", err)
		}
	})
}
