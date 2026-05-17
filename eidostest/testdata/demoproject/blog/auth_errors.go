// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Authentication-related errors the blog package exposes. The
// `+gen:sentinel` directive on article.go's package clause
// opts the blog package in for the sentinel plugin's automatic
// _sentinel_test.go generation — every Err* var below and every
// exported error-implementing struct in the package gains pinned
// invariants (prefix, uniqueness, non-overlap, wrap-chain
// preservation, errors.As round-trip, format strictness).
package blog

import (
	"errors"
	"fmt"
)

// ErrUnauthorised is returned when the caller's token does not
// authorise the requested operation.
var ErrUnauthorised = errors.New("blog: unauthorised")

// ErrTokenExpired is returned when the caller's token is past
// its declared expiry.
var ErrTokenExpired = errors.New("blog: token expired")

// ValidationError carries the per-field reason a structured
// payload failed validation. The string-keyed Field plus the
// human-readable Reason make the error self-describing for both
// callers (errors.As) and operators (logged Error() output).
type ValidationError struct {
	// Field is the offending field's identifier (e.g. "Title").
	Field string

	// Reason is the human-readable validation failure
	// description.
	Reason string
}

// Error implements the error interface — `blog: validation
// failed: <Field>: <Reason>`. The format includes both fields
// so log scrapers can locate the offending field without
// reaching for errors.As.
func (e *ValidationError) Error() string {
	return fmt.Sprintf("blog: validation failed: %s: %s", e.Field, e.Reason)
}
