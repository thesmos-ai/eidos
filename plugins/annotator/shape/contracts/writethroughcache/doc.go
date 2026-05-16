// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package writethroughcache recognises the write-through-cache
// contract — a cache callable paired with the backing-store
// callable it delegates misses to. The cache role requires the
// backing partner.
//
// The recognised directive (on the cache side) is:
//
//	//+gen:contract cache role=cache backing=GetFromDB
package writethroughcache
