// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package appender_test

import (
	"testing"

	"go.thesmos.sh/eidos/plugins/annotator/shape/contracts/appender"
	"go.thesmos.sh/eidos/plugins/annotator/shape/contracts/internal/contracttest"
)

func TestContract_Identity(t *testing.T) {
	t.Parallel()
	contracttest.AssertIdentity(t,
		appender.Contract(),
		appender.Name, appender.Roles)
}
