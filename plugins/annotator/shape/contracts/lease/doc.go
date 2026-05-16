// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package lease recognises the lease contract — an acquire
// callable paired with a release callable. Every successful
// acquire must be balanced by exactly one release; the validator
// surfaces missing-partner errors via Required.
//
// The recognised directive (on the acquire side) is:
//
//	//+gen:contract lease role=acquire release=Release
package lease
