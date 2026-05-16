// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package cursor recognises the cursor contract — a next-element
// reader paired with the close callable that releases the
// underlying iterator's resources.
//
// The recognised directive (on the next side) is:
//
//	//+gen:contract cursor role=next close=Close
package cursor
