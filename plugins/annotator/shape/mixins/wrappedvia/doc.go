// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package wrappedvia recognises the wrapped-via mixin — the
// assertion that the annotated callable delegates to the named
// sibling, wrapping it with cross-cutting concerns (logging,
// metrics, etc.). The `fn` param names the wrapped sibling; the
// resolver rewrites it into a qualified name.
//
// The recognised directive is:
//
//	//+gen:mixin wrappedvia fn=Delegate
package wrappedvia
