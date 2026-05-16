// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package cursor_test

import (
	"testing"

	"go.thesmos.sh/eidos/plugins/annotator/shape/contracts/cursor"
	"go.thesmos.sh/eidos/plugins/annotator/shape/contracts/internal/contracttest"
)

func TestContract_Identity(t *testing.T) {
	t.Parallel()
	contracttest.AssertIdentity(t,
		cursor.Contract(),
		cursor.Name, cursor.Roles)
}
