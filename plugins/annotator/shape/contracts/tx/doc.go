// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package tx recognises the two-phase transaction contract — a
// begin / commit / rollback triple. The Begin host declares both
// partners; the validator emits a diagnostic when either is
// missing.
//
// The recognised directive (on the Begin side) is:
//
//	//+gen:contract tx role=begin commit=Commit rollback=Rollback
package tx
