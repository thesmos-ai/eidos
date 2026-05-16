// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package streamreflectsmutations recognises the
// stream-reflects-mutations mixin — the assertion that an
// already-iterating stream observes mutations applied
// concurrently to the underlying data source.
//
// The recognised directive is:
//
//	//+gen:mixin streamreflectsmutations
package streamreflectsmutations
