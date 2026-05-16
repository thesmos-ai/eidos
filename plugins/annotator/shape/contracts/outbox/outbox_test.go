// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package outbox_test

import (
	"testing"

	"go.thesmos.sh/eidos/plugins/annotator/shape/contracts/internal/contracttest"
	"go.thesmos.sh/eidos/plugins/annotator/shape/contracts/outbox"
)

func TestContract_Identity(t *testing.T) {
	t.Parallel()
	contracttest.AssertIdentity(t,
		outbox.Contract(),
		outbox.Name, outbox.Roles)
}
