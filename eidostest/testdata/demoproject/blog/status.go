// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package blog

// Status enumerates the publication states an Article moves through.
// The typed-iota declaration exercises the backend's enum-rendering
// path (typed underlying + iota promotion) on a fixture type that
// other fixture entities reference. The `+gen:enum` directive opts
// the type in for the production enum plugin's String / Parse /
// MarshalJSON / UnmarshalJSON surface and a paired test file.
//
//+gen:enum
type Status int

const (
	// Draft indicates the article is still being written.
	Draft Status = iota

	// Published indicates the article is live.
	Published

	// Archived indicates the article is no longer current.
	Archived
)
