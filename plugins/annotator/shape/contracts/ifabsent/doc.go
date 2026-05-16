// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package ifabsent recognises the if-absent conditional-writer
// contract — a writer that succeeds only when no record with the
// declared key exists. Single-role marker; no params.
//
// The recognised directive is:
//
//	//+gen:contract if-absent role=writer
package ifabsent
