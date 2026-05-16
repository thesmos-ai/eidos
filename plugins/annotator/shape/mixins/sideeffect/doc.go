// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package sideeffect recognises the side-effect mixin — the
// assertion that the annotated callable has observable
// side-effects beyond its return value (so the test suite must
// observe them externally rather than just inspecting the
// return).
//
// The recognised directive is:
//
//	//+gen:mixin sideeffect
package sideeffect
