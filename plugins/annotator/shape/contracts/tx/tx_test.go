// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package tx_test

import (
	"reflect"
	"testing"

	"go.thesmos.sh/eidos/plugins/annotator/shape/contracts/internal/contracttest"
	"go.thesmos.sh/eidos/plugins/annotator/shape/contracts/tx"
)

func TestContract_Identity(t *testing.T) {
	t.Parallel()
	contracttest.AssertIdentity(t,
		tx.Contract(),
		tx.Name, tx.Roles)
}

func TestContract_RequiresCommitAndRollback(t *testing.T) {
	t.Parallel()
	got := tx.Contract().Required
	want := map[string][]string{"begin": {"commit", "rollback"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Required = %v, want %v", got, want)
	}
}
