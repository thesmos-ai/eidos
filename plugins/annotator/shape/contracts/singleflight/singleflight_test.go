// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package singleflight_test

import (
	"testing"

	"go.thesmos.sh/eidos/plugins/annotator/shape/contracts/internal/contracttest"
	"go.thesmos.sh/eidos/plugins/annotator/shape/contracts/singleflight"
)

func TestContract_Identity(t *testing.T) {
	t.Parallel()
	contracttest.AssertIdentity(t,
		singleflight.Contract(),
		singleflight.Name, singleflight.Roles)
}
