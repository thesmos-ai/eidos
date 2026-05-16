// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package integrationonly recognises the integration-only mixin —
// the assertion that the annotated callable's test suite must run
// against the integration target rather than a unit-level mock.
//
// The recognised directive is:
//
//	//+gen:mixin integrationonly
package integrationonly
