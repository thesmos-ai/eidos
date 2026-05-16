// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package crdtmerge recognises the crdt-merge mixin — the
// assertion that concurrent writes to the annotated callable
// merge deterministically (CRDT-style) without conflict.
//
// The recognised directive is:
//
//	//+gen:mixin crdtmerge
package crdtmerge
