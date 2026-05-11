// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package golang

import (
	"encoding/json"
	"fmt"
	"go/types"
)

// constraintTermsFromUnion converts a [types.Union] (the type-checker
// representation of a `~int | ~string` constraint expression) into
// the [ConstraintTerm] slice [MetaConstraintTerms] carries.
func (c *converter) constraintTermsFromUnion(u *types.Union) []ConstraintTerm {
	out := make([]ConstraintTerm, 0, u.Len())
	for term := range u.Terms() {
		out = append(out, ConstraintTerm{
			Type:        c.typeRefOf(term.Type()),
			Approximate: term.Tilde(),
		})
	}
	return out
}

// jsonUnmarshalString decodes raw into v via [encoding/json]. Used
// by [meta.Parser] callbacks that accept structured JSON values
// off the directive-override path; programmatic stamping bypasses
// the parser entirely.
func jsonUnmarshalString(raw string, v any) error {
	if err := json.Unmarshal([]byte(raw), v); err != nil {
		return fmt.Errorf("go-frontend: parse meta value: %w", err)
	}
	return nil
}
