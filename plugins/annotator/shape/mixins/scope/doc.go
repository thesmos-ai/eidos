// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package scope recognises the scope mixin — the assertion that
// the annotated callable's effect is confined to the named
// scope (request, session, tenant, etc.) and never leaks into
// another scope.
//
// The recognised directive is:
//
//	//+gen:mixin scope name=request
package scope
