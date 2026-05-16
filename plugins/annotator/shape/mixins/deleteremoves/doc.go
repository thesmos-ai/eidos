// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package deleteremoves recognises the delete-removes mixin —
// the assertion that a delete operation actually removes the
// entity such that a subsequent read returns not-found.
//
// The recognised directive is:
//
//	//+gen:mixin deleteremoves
package deleteremoves
