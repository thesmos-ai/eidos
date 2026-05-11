// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package pipeline

import (
	"fmt"

	"go.thesmos.sh/eidos/plugin"
)

// validateTemplateFuncs returns one error per collision detected
// in [plugin.TemplateProvider.TemplateFuncs] for the supplied
// language. Two plugins claiming the same func name (without going
// through [plugin.TemplateProvider.TemplateOverrides]) is rejected
// at Build time so the backend never has to choose silently.
//
// Plugins that don't implement [plugin.TemplateProvider] are
// ignored. Plugins that return a nil or empty FuncMap for the
// language contribute no entries.
func validateTemplateFuncs(plugins []plugin.Plugin, lang string) []error {
	type owner struct {
		funcName   string
		pluginName string
	}
	seen := map[string]owner{}
	var errs []error
	for _, p := range plugins {
		tp, ok := p.(plugin.TemplateProvider)
		if !ok {
			continue
		}
		for name := range tp.TemplateFuncs(lang) {
			if prior, dup := seen[name]; dup {
				errs = append(errs, fmt.Errorf("%w: %q (%s) collides with %s",
					ErrTemplateFuncCollision, name, p.Name(), prior.pluginName))
				continue
			}
			seen[name] = owner{funcName: name, pluginName: p.Name()}
		}
	}
	return errs
}
