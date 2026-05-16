// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package transaction recognises the transaction contract — a
// single-role marker declaring that the annotated callable runs
// inside (or owns) a transactional scope.
//
// The recognised directive is:
//
//	//+gen:contract transaction role=fn
//
// Single-member contract: no partner siblings, no cross-callable
// resolution. The downstream codegen reads the contract
// membership to wrap the call in begin / commit / rollback
// scaffolding.
package transaction
