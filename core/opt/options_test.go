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

// optionsForDecode is the test-side options struct decoded against
// throughout the Decode tests.
type optionsForDecode struct {
	OutputPackage string        `eidos:"output_package,required"`
	InterfaceName string        `eidos:"interface_name,default=Repository"`
	Naming        string        `eidos:"naming,one_of=Pascal|Camel,default=Pascal"`
	Retries       int           `eidos:"retries,default=3"`
	Verbose       bool          `eidos:"verbose,default=false"`
	Tags          []string      `eidos:"tags"`
	Timeout       time.Duration `eidos:"timeout,default=30s"`
}

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("binds the supplied schema and values without validation", func(t *testing.T) {
		t.Parallel()
		s := opt.Reflect(optionsForDecode{})
		o := opt.New(s, map[string]string{"output_package": "repo"})
		if o.Schema().Names()[0] != "output_package" {
			t.Fatalf("Schema().Names()[0] = %v", o.Schema().Names())
		}
	})
}

func TestOptions_Has(t *testing.T) {
	t.Parallel()

	t.Run("returns true for a present key", func(t *testing.T) {
		t.Parallel()
		o := opt.New(opt.Schema{}, map[string]string{"k": "v"})
		if !o.Has("k") {
			t.Fatalf("Has should be true for a present key")
		}
	})

	t.Run("returns false for an absent key", func(t *testing.T) {
		t.Parallel()
		o := opt.New(opt.Schema{}, map[string]string{"k": "v"})
		if o.Has("missing") {
			t.Fatalf("Has should be false for an absent key")
		}
	})
}

func TestOptions_Decode(t *testing.T) {
	t.Parallel()

	schema := opt.Reflect(optionsForDecode{})

	t.Run("populates destination struct from supplied values", func(t *testing.T) {
		t.Parallel()
		var dst optionsForDecode
		err := opt.New(schema, map[string]string{
			"output_package": "repo",
			"interface_name": "Repository",
			"naming":         "Camel",
			"retries":        "5",
			"verbose":        "true",
			"tags":           "alpha,beta",
			"timeout":        "10s",
		}).Decode(&dst)
		assertNoError(t, err, "Decode")
		if dst.OutputPackage != "repo" || dst.Naming != "Camel" || dst.Retries != 5 {
			t.Fatalf("decoded struct mismatch: %+v", dst)
		}
		if !dst.Verbose {
			t.Fatalf("verbose should be true")
		}
		if !slices.Equal(dst.Tags, []string{"alpha", "beta"}) {
			t.Fatalf("tags = %v", dst.Tags)
		}
		if dst.Timeout != 10*time.Second {
			t.Fatalf("timeout = %v, want 10s", dst.Timeout)
		}
	})

	t.Run("applies defaults to absent optional fields", func(t *testing.T) {
		t.Parallel()
		var dst optionsForDecode
		err := opt.New(schema, map[string]string{
			"output_package": "repo",
		}).Decode(&dst)
		assertNoError(t, err, "Decode")
		if dst.InterfaceName != "Repository" {
			t.Fatalf("interface_name default not applied: %q", dst.InterfaceName)
		}
		if dst.Retries != 3 {
			t.Fatalf("retries default not applied: %d", dst.Retries)
		}
		if dst.Timeout != 30*time.Second {
			t.Fatalf("timeout default not applied: %v", dst.Timeout)
		}
	})

	t.Run("returns ErrMissingRequired when a required field is absent", func(t *testing.T) {
		t.Parallel()
		var dst optionsForDecode
		err := opt.New(schema, map[string]string{}).Decode(&dst)
		if !errors.Is(err, opt.ErrMissingRequired) {
			t.Fatalf("err = %v, want ErrMissingRequired", err)
		}
	})

	t.Run("returns ErrUnknownField for a key not in the schema", func(t *testing.T) {
		t.Parallel()
		var dst optionsForDecode
		err := opt.New(schema, map[string]string{
			"output_package": "repo",
			"intruder":       "x",
		}).Decode(&dst)
		if !errors.Is(err, opt.ErrUnknownField) {
			t.Fatalf("err = %v, want ErrUnknownField", err)
		}
	})

	t.Run("returns ErrInvalidValue for a value outside one_of", func(t *testing.T) {
		t.Parallel()
		var dst optionsForDecode
		err := opt.New(schema, map[string]string{
			"output_package": "repo",
			"naming":         "Snake",
		}).Decode(&dst)
		if !errors.Is(err, opt.ErrInvalidValue) {
			t.Fatalf("err = %v, want ErrInvalidValue", err)
		}
	})

	t.Run("returns ErrInvalidValue for a non-numeric int", func(t *testing.T) {
		t.Parallel()
		var dst optionsForDecode
		err := opt.New(schema, map[string]string{
			"output_package": "repo",
			"retries":        "many",
		}).Decode(&dst)
		if !errors.Is(err, opt.ErrInvalidValue) {
			t.Fatalf("err = %v, want ErrInvalidValue", err)
		}
	})

	t.Run("returns ErrInvalidValue for an invalid bool", func(t *testing.T) {
		t.Parallel()
		var dst optionsForDecode
		err := opt.New(schema, map[string]string{
			"output_package": "repo",
			"verbose":        "maybe",
		}).Decode(&dst)
		if !errors.Is(err, opt.ErrInvalidValue) {
			t.Fatalf("err = %v, want ErrInvalidValue", err)
		}
	})

	t.Run("returns ErrInvalidValue for an unparseable duration", func(t *testing.T) {
		t.Parallel()
		var dst optionsForDecode
		err := opt.New(schema, map[string]string{
			"output_package": "repo",
			"timeout":        "not-a-duration",
		}).Decode(&dst)
		if !errors.Is(err, opt.ErrInvalidValue) {
			t.Fatalf("err = %v, want ErrInvalidValue", err)
		}
	})

	t.Run("returns ErrInvalidDecodeTarget for a non-pointer", func(t *testing.T) {
		t.Parallel()
		err := opt.New(schema, map[string]string{}).Decode(optionsForDecode{})
		if !errors.Is(err, opt.ErrInvalidDecodeTarget) {
			t.Fatalf("err = %v, want ErrInvalidDecodeTarget", err)
		}
	})

	t.Run("returns ErrInvalidDecodeTarget for a nil pointer", func(t *testing.T) {
		t.Parallel()
		var nilPtr *optionsForDecode
		err := opt.New(schema, map[string]string{}).Decode(nilPtr)
		if !errors.Is(err, opt.ErrInvalidDecodeTarget) {
			t.Fatalf("err = %v, want ErrInvalidDecodeTarget", err)
		}
	})

	t.Run("returns ErrInvalidDecodeTarget for a pointer to non-struct", func(t *testing.T) {
		t.Parallel()
		i := 0
		err := opt.New(schema, map[string]string{}).Decode(&i)
		if !errors.Is(err, opt.ErrInvalidDecodeTarget) {
			t.Fatalf("err = %v, want ErrInvalidDecodeTarget", err)
		}
	})

	t.Run("empty string list decodes to an empty slice not nil", func(t *testing.T) {
		t.Parallel()
		var dst optionsForDecode
		err := opt.New(schema, map[string]string{
			"output_package": "repo",
			"tags":           "",
		}).Decode(&dst)
		assertNoError(t, err, "Decode")
		if dst.Tags == nil || len(dst.Tags) != 0 {
			t.Fatalf("tags = %v, want empty slice", dst.Tags)
		}
	})

	t.Run("returns ErrInvalidValue for an unknown FieldKind", func(t *testing.T) {
		t.Parallel()
		type tinyTarget struct {
			V string
		}
		// Hand-built schema with a deliberately invalid kind to cover
		// parseValue's default branch. Reflect can never produce
		// this, but Schema is a public struct so misuse is possible.
		s := opt.Schema{
			Fields: []opt.Field{{
				Name:        "v",
				GoFieldName: "V",
				Kind:        opt.FieldKind(99),
			}},
		}
		var d tinyTarget
		err := opt.New(s, map[string]string{"v": "anything"}).Decode(&d)
		if !errors.Is(err, opt.ErrInvalidValue) {
			t.Fatalf("err = %v, want ErrInvalidValue", err)
		}
	})
}
