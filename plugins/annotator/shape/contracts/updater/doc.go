// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package updater recognises the updater contract — a writer
// paired with a reader, where the reader fetches the just-updated
// entity to confirm the write took effect.
//
// The recognised directive (on the writer side) is:
//
//	//+gen:contract updater role=writer reader=GetByID
package updater
