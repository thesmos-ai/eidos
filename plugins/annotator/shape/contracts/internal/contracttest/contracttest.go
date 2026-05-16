// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package contracttest provides shared assertions for the
// per-contract sub-packages. Each contract test file boils down
// to a single [AssertIdentity] call plus, optionally, contract-
// specific tests that exercise the validator hook.
//
// Internal — importable only by [shape/contracts/...] children;
// not part of the shape library's public API.
package contracttest

import (
	"reflect"
	"testing"

	"go.thesmos.sh/eidos/plugins/annotator/shape"
)

// AssertIdentity fails the test when c does not match the
// expected name + roles. Use as the canonical body of every
// per-contract `TestContract_Identity` test:
//
//	contracttest.AssertIdentity(t, watcher.Contract(),
//	    watcher.Name, watcher.Roles)
func AssertIdentity(t *testing.T, c shape.Contract, wantName string, wantRoles []string) {
	t.Helper()
	if c.Name != wantName {
		t.Fatalf("Contract().Name = %q, want %q", c.Name, wantName)
	}
	if !reflect.DeepEqual(c.Roles, wantRoles) {
		t.Fatalf("Contract().Roles = %v, want %v", c.Roles, wantRoles)
	}
}
