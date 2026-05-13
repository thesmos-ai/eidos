// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package buildfixture

import (
	"google.golang.org/protobuf/types/known/timestamppb"

	"go.thesmos.sh/eidos/eidostest/protopipe/buildfixture/extras"
)

// Status mirrors the fixture proto's `enum Status`. proto3
// enums map to int32-underlied Go types per the protoc-gen-go
// convention; the bridge stamps go.type / go.import meta on
// enum-typed type-refs so the rendered output references this
// type by name.
type Status int32

// Status variant constants mirror the proto-declared values.
// The bridge does not currently emit the variant constants
// (they live on the source-side node.EnumVariant entries); the
// stubs declare them so test-package code can reference the
// canonical names without a hand-rolled package.
const (
	StatusUnknown Status = 0
	StatusActive  Status = 1
	StatusBanned  Status = 2
)

// User mirrors the fixture proto's `message User`. The field
// types match the Go-side forms the protogo bridge composes
// through the scalar, well-known, optional-wrap, nested-
// reference, enum-reference, and cross-package-reference rules.
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
	State      Status
	ExtrasTag  extras.Tag
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

// GetUserRequest mirrors the request message for the proto
// UserService.GetUser RPC. The rendered mockgen output
// references this type by name; the stub gives the rendered
// `_mock_test.go` an existing target so `go vet` against the
// fixture directory succeeds.
type GetUserRequest struct {
	ID string
}
