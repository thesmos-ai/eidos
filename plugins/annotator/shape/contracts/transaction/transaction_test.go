// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package transaction_test

import (
	"testing"

	"go.thesmos.sh/eidos/plugins/annotator/shape/contracts/internal/contracttest"
	"go.thesmos.sh/eidos/plugins/annotator/shape/contracts/transaction"
)

func TestContract_Identity(t *testing.T) {
	t.Parallel()
	contracttest.AssertIdentity(t,
		transaction.Contract(),
		transaction.Name, transaction.Roles)
}
