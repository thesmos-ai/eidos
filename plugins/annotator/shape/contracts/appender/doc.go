// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package appender recognises the append-only writer contract —
// a single-role marker declaring that the annotated callable's
// effect is purely additive (no overwrite, no delete).
//
// The recognised directive is:
//
//	//+gen:contract appender role=fn
package appender
