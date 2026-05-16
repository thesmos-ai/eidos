// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package partition recognises the partition mixin — the
// assertion that the annotated callable observes a partition
// boundary (e.g. tenant, shard) and never serves data from a
// different partition.
//
// The recognised directive is:
//
//	//+gen:mixin partition
package partition
