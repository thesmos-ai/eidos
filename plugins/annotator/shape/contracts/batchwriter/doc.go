// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package batchwriter recognises the batch-writer contract — a
// writer accepting a batch of records with a configured failure
// mode. The `mode` param is opaque (typically `all-or-nothing`
// or `best-effort`); the resolver leaves the value unchanged.
//
// The recognised directive is:
//
//	//+gen:contract batch-writer role=writer mode=all-or-nothing
package batchwriter
