// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package ifabsent_test

import (
	"testing"

	"go.thesmos.sh/eidos/plugins/annotator/shape/contracts/ifabsent"
	"go.thesmos.sh/eidos/plugins/annotator/shape/contracts/internal/contracttest"
)

func TestContract_Identity(t *testing.T) {
	t.Parallel()
	contracttest.AssertIdentity(t, ifabsent.Contract(), ifabsent.Name, ifabsent.Roles)
}
