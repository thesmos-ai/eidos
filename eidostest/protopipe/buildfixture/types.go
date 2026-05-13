// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package buildfixture

import "google.golang.org/protobuf/types/known/timestamppb"

// User mirrors the fixture proto's `message User`. The field
// types match the Go-side forms the protogo bridge composes
// through the scalar, well-known, optional-wrap, and nested-
// reference rules.
//
// ProfileRef is a value-typed reference to the nested message
// rather than the pointer-typed form protoc-gen-go emits.
// proto-message pointer-wrap is a downstream code-generation
// convention rather than a bridge-translation rule, so the
// fixture matches the bridge's current output verbatim —
// optional fields wrap (`Age *int32`), non-optional message
// fields do not.
type User struct {
	Name       string
	Age        *int32
	CreatedAt  *timestamppb.Timestamp
	ProfileRef User_Profile
}

// User_Profile mirrors the fixture proto's nested `message
// Profile`. The Go-side identifier underscore-joins the outer
// and nested names per the protoc-gen-go convention; the
// bridge's render-site rule produces the same form.
//
//revive:disable-next-line:var-naming // proto nested-name convention.
//nolint:staticcheck // ST1003: proto nested-name convention.
type User_Profile struct {
	Bio string
}
