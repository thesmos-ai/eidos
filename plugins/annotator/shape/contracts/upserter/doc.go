// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package upserter recognises the upserter contract — a writer
// paired with a reader, where the writer's semantics combine
// insert and update (last-write-wins on the key).
//
// The recognised directive (on the writer side) is:
//
//	//+gen:contract upserter role=writer reader=GetByID
package upserter
