// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package pagination recognises the paginated-reader contract —
// a reader whose protocol cursors via a named field. The
// directive carries a `cursor=` param naming the field; the
// resolver leaves the value as an opaque string.
//
// The recognised directive is:
//
//	//+gen:contract pagination role=reader cursor=Cursor
package pagination
