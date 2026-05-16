// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package updater_test

import (
	"testing"

	"go.thesmos.sh/eidos/plugins/annotator/shape/contracts/internal/contracttest"
	"go.thesmos.sh/eidos/plugins/annotator/shape/contracts/updater"
)

func TestContract_Identity(t *testing.T) {
	t.Parallel()
	contracttest.AssertIdentity(t,
		updater.Contract(),
		updater.Name, updater.Roles)
}
