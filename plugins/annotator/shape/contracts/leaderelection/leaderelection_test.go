// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package leaderelection_test

import (
	"reflect"
	"testing"

	"go.thesmos.sh/eidos/plugins/annotator/shape/contracts/internal/contracttest"
	"go.thesmos.sh/eidos/plugins/annotator/shape/contracts/leaderelection"
)

func TestContract_Identity(t *testing.T) {
	t.Parallel()
	contracttest.AssertIdentity(t,
		leaderelection.Contract(),
		leaderelection.Name, leaderelection.Roles)
}

func TestContract_RequiresResignAndIsLeader(t *testing.T) {
	t.Parallel()
	got := leaderelection.Contract().Required
	want := map[string][]string{"campaign": {"resign", "isleader"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Required = %v, want %v", got, want)
	}
}
