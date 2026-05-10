// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package opt_test

import (
	"errors"
	"slices"
	"testing"
	"time"

	"go.thesmos.sh/eidos/core/opt"
)

type basicOptions struct {
	OutputPackage string        `eidos:"output_package,required"`
	InterfaceName string        `eidos:"interface_name,default=Repository"`
	Naming        string        `eidos:"naming,one_of=Pascal|Camel,default=Pascal"`
	Retries       int           `eidos:"retries,default=3"`
	Verbose       bool          `eidos:"verbose,default=false"`
	Tags          []string      `eidos:"tags"`
	Timeout       time.Duration `eidos:"timeout,default=30s"`
}

func TestReflect(t *testing.T) {
	t.Parallel()

	t.Run("derives a schema with one field per tagged struct field", func(t *testing.T) {
		t.Parallel()
		s := opt.Reflect(basicOptions{})
		if len(s.Fields) != 7 {
			t.Fatalf("Fields = %d, want 7", len(s.Fields))
		}
	})

	t.Run("maps tag options into Field flags", func(t *testing.T) {
		t.Parallel()
		s := opt.Reflect(basicOptions{})
		out, ok := s.Lookup("output_package")
		if !ok {
			t.Fatalf("output_package not in schema")
		}
		if !out.Required {
			t.Fatalf("output_package should be Required")
		}
	})

	t.Run("captures default values verbatim", func(t *testing.T) {
		t.Parallel()
		s := opt.Reflect(basicOptions{})
		got, _ := s.Lookup("interface_name")
		if !got.HasDefault || got.DefaultStr != "Repository" {
			t.Fatalf("interface_name default = %+v", got)
		}
	})

	t.Run("parses one_of into a slice", func(t *testing.T) {
		t.Parallel()
		s := opt.Reflect(basicOptions{})
		got, _ := s.Lookup("naming")
		if !slices.Equal(got.OneOf, []string{"Pascal", "Camel"}) {
			t.Fatalf("naming OneOf = %v, want [Pascal Camel]", got.OneOf)
		}
	})

	t.Run("snake-cases the option name when the tag's name segment is empty", func(t *testing.T) {
		t.Parallel()
		type derivedName struct {
			OutputPackage string `eidos:",required"`
		}
		s := opt.Reflect(derivedName{})
		if _, ok := s.Lookup("output_package"); !ok {
			t.Fatalf("expected output_package derived from field name; got %v", s.Names())
		}
	})

	t.Run("derives a name when the tag is absent entirely", func(t *testing.T) {
		t.Parallel()
		type untagged struct {
			OutputPackage string
		}
		s := opt.Reflect(untagged{})
		if _, ok := s.Lookup("output_package"); !ok {
			t.Fatalf("expected output_package derived from untagged field; got %v", s.Names())
		}
	})

	t.Run("skips fields tagged with -", func(t *testing.T) {
		t.Parallel()
		type withSkip struct {
			Visible string `eidos:"visible"`
			Hidden  string `eidos:"-"`
		}
		s := opt.Reflect(withSkip{})
		if len(s.Fields) != 1 || s.Fields[0].Name != "visible" {
			t.Fatalf("Fields = %+v, want only visible", s.Fields)
		}
	})

	t.Run("skips unexported fields", func(t *testing.T) {
		t.Parallel()
		type withUnexported struct {
			Visible string `eidos:"visible"`
			hidden  string //nolint:unused // intentionally tests unexported skipping
		}
		_ = withUnexported{}
		s := opt.Reflect(withUnexported{})
		if len(s.Fields) != 1 || s.Fields[0].Name != "visible" {
			t.Fatalf("Fields = %+v, want only visible", s.Fields)
		}
	})

	t.Run("captures description from desc= tag option", func(t *testing.T) {
		t.Parallel()
		type withDesc struct {
			V string `eidos:"v,desc=hello world"`
		}
		s := opt.Reflect(withDesc{})
		got, _ := s.Lookup("v")
		assertEqualString(t, got.Description, "hello world")
	})

	t.Run("recognises time.Duration as KindDuration", func(t *testing.T) {
		t.Parallel()
		s := opt.Reflect(basicOptions{})
		got, _ := s.Lookup("timeout")
		if got.Kind != opt.KindDuration {
			t.Fatalf("timeout Kind = %v, want Duration", got.Kind)
		}
	})

	t.Run("accepts a pointer to a struct example", func(t *testing.T) {
		t.Parallel()
		type small struct {
			V string
		}
		s := opt.Reflect(&small{})
		if _, ok := s.Lookup("v"); !ok {
			t.Fatalf("Reflect(*struct) should produce a schema")
		}
	})
}

func TestReflectChecked(t *testing.T) {
	t.Parallel()

	t.Run("rejects a nil example with ErrInvalidTag", func(t *testing.T) {
		t.Parallel()
		_, err := opt.ReflectChecked(nil)
		if !errors.Is(err, opt.ErrInvalidTag) {
			t.Fatalf("err = %v, want ErrInvalidTag", err)
		}
	})

	t.Run("rejects a non-struct example with ErrInvalidTag", func(t *testing.T) {
		t.Parallel()
		_, err := opt.ReflectChecked("nope")
		if !errors.Is(err, opt.ErrInvalidTag) {
			t.Fatalf("err = %v, want ErrInvalidTag", err)
		}
	})

	t.Run("rejects an unsupported field type", func(t *testing.T) {
		t.Parallel()
		type withUnsupported struct {
			V map[string]string
		}
		_, err := opt.ReflectChecked(withUnsupported{})
		if !errors.Is(err, opt.ErrUnsupportedFieldType) {
			t.Fatalf("err = %v, want ErrUnsupportedFieldType", err)
		}
	})

	t.Run("rejects an unsupported slice element type", func(t *testing.T) {
		t.Parallel()
		type withUnsupportedSlice struct {
			V []int
		}
		_, err := opt.ReflectChecked(withUnsupportedSlice{})
		if !errors.Is(err, opt.ErrUnsupportedFieldType) {
			t.Fatalf("err = %v, want ErrUnsupportedFieldType", err)
		}
	})

	t.Run("rejects an unknown tag option", func(t *testing.T) {
		t.Parallel()
		type withUnknownTag struct {
			V string `eidos:"v,bogus=1"`
		}
		_, err := opt.ReflectChecked(withUnknownTag{})
		if !errors.Is(err, opt.ErrInvalidTag) {
			t.Fatalf("err = %v, want ErrInvalidTag", err)
		}
	})

	t.Run("rejects one_of on a non-string field", func(t *testing.T) {
		t.Parallel()
		type oneOfInt struct {
			V int `eidos:"v,one_of=1|2"`
		}
		_, err := opt.ReflectChecked(oneOfInt{})
		if !errors.Is(err, opt.ErrInvalidTag) {
			t.Fatalf("err = %v, want ErrInvalidTag", err)
		}
	})

	t.Run("rejects a default that is not in one_of", func(t *testing.T) {
		t.Parallel()
		type defaultNotInOneOf struct {
			V string `eidos:"v,one_of=A|B,default=C"`
		}
		_, err := opt.ReflectChecked(defaultNotInOneOf{})
		if !errors.Is(err, opt.ErrInvalidTag) {
			t.Fatalf("err = %v, want ErrInvalidTag", err)
		}
	})

	t.Run("rejects a default that fails per-kind parsing", func(t *testing.T) {
		t.Parallel()
		type defaultBadInt struct {
			V int `eidos:"v,default=not-a-number"`
		}
		_, err := opt.ReflectChecked(defaultBadInt{})
		if !errors.Is(err, opt.ErrInvalidTag) {
			t.Fatalf("err = %v, want ErrInvalidTag", err)
		}
	})

	t.Run("rejects a default duration that fails to parse", func(t *testing.T) {
		t.Parallel()
		type defaultBadDuration struct {
			V time.Duration `eidos:"v,default=not-a-duration"`
		}
		_, err := opt.ReflectChecked(defaultBadDuration{})
		if !errors.Is(err, opt.ErrInvalidTag) {
			t.Fatalf("err = %v, want ErrInvalidTag", err)
		}
	})
}

func TestReflect_PanicsOnError(t *testing.T) {
	t.Parallel()

	t.Run("panics with the same error ReflectChecked returns", func(t *testing.T) {
		t.Parallel()
		defer func() {
			r := recover()
			if r == nil {
				t.Fatalf("expected panic on invalid example")
			}
			err, ok := r.(error)
			if !ok {
				t.Fatalf("panic value should be an error; got %T: %v", r, r)
			}
			if !errors.Is(err, opt.ErrInvalidTag) {
				t.Fatalf("panic err = %v, want ErrInvalidTag", err)
			}
		}()
		_ = opt.Reflect("nope")
	})
}
