// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package lifecycleafterclose recognises the
// lifecycle-after-close mixin — the assertion that the annotated
// callable continues to behave correctly (typically returning a
// sentinel error) when invoked after the host has been closed.
//
// The recognised directive is:
//
//	//+gen:mixin lifecycleafterclose
package lifecycleafterclose
