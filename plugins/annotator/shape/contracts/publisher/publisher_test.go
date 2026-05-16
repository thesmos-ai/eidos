// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package publisher_test

import (
	"testing"

	"go.thesmos.sh/eidos/plugins/annotator/shape/contracts/internal/contracttest"
	"go.thesmos.sh/eidos/plugins/annotator/shape/contracts/publisher"
)

func TestContract_Identity(t *testing.T) {
	t.Parallel()
	contracttest.AssertIdentity(t,
		publisher.Contract(),
		publisher.Name, publisher.Roles)
}
