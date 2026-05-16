// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package workflow recognises the workflow contract — a callable
// whose execution follows a declared state-transition graph. The
// `transitions` param carries the encoded transitions (opaque to
// the resolver; the downstream codegen parses it).
//
// The recognised directive is:
//
//	//+gen:contract workflow role=fn transitions=start->step1->done
package workflow
