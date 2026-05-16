// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package pool recognises the pool contract — an acquire / release
// pair (Get / Put) with the invariant that every Get balances
// exactly one Put. Carries a [Validate] hook that flags pool
// declarations missing either side.
//
// The recognised directive (on the get side) is:
//
//	//+gen:contract pool role=get put=Put
package pool
