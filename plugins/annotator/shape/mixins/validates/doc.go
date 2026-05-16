// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package validates recognises the validates mixin — the
// assertion that the annotated callable's inputs are screened by
// the named validator before any business logic runs. The `fn`
// param names the validator sibling; the resolver rewrites it
// into a qualified name.
//
// The recognised directive is:
//
//	//+gen:mixin validates fn=ValidateInput
package validates
