// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package pipeline_test

import (
	"errors"
	"strings"
	"testing"
	"text/template"

	"go.thesmos.sh/eidos/pipeline"
)

func TestTemplateFuncCollision(t *testing.T) {
	t.Parallel()

	t.Run("returns ErrTemplateFuncCollision when two plugins claim the same func name", func(t *testing.T) {
		t.Parallel()
		gen := &stubGenWithTemplates{
			name:  "gen",
			funcs: template.FuncMap{"shared": func() string { return "" }},
		}
		be := &stubBEWithTemplates{
			name:  "be",
			lang:  "stub",
			funcs: template.FuncMap{"shared": func() string { return "" }},
		}
		_, err := pipeline.New().
			WithFrontend(&stubFE{name: "fe"}).
			WithGenerator(gen).
			WithBackend(be).
			Build()
		if !errors.Is(err, pipeline.ErrTemplateFuncCollision) {
			t.Fatalf("Build should return ErrTemplateFuncCollision; got %v", err)
		}
		if !strings.Contains(err.Error(), "shared") {
			t.Fatalf("error should name the colliding func; got %q", err.Error())
		}
	})

	t.Run("plugins claiming distinct func names compose without collision", func(t *testing.T) {
		t.Parallel()
		gen := &stubGenWithTemplates{
			name:  "gen",
			funcs: template.FuncMap{"upper": func() string { return "" }},
		}
		be := &stubBEWithTemplates{
			name:  "be",
			lang:  "stub",
			funcs: template.FuncMap{"lower": func() string { return "" }},
		}
		_, err := pipeline.New().
			WithFrontend(&stubFE{name: "fe"}).
			WithGenerator(gen).
			WithBackend(be).
			Build()
		assertNoError(t, err)
	})

	t.Run("intentional overrides do not trigger the collision check", func(t *testing.T) {
		t.Parallel()
		// Both plugins claim "shared" but one routes through
		// TemplateOverrides (intentional) so no collision diagnostic
		// fires.
		gen := &stubGenWithTemplates{
			name:      "gen",
			funcs:     template.FuncMap{"shared": func() string { return "" }},
			overrides: template.FuncMap{},
		}
		be := &stubBEWithTemplates{
			name:      "be",
			lang:      "stub",
			funcs:     template.FuncMap{},
			overrides: template.FuncMap{"shared": func() string { return "" }},
		}
		_, err := pipeline.New().
			WithFrontend(&stubFE{name: "fe"}).
			WithGenerator(gen).
			WithBackend(be).
			Build()
		assertNoError(t, err)
	})
}
