// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package errors recognises the errors mixin — the assertion that
// the annotated callable's error returns are part of its contract
// (callers are expected to inspect and handle them) rather than
// "shouldn't happen" sentinels.
//
// The recognised directive is:
//
//	//+gen:mixin errors
package errors
