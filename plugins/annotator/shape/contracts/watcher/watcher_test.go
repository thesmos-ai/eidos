// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package watcher_test

import (
	"testing"

	"go.thesmos.sh/eidos/plugins/annotator/shape/contracts/internal/contracttest"
	"go.thesmos.sh/eidos/plugins/annotator/shape/contracts/watcher"
)

func TestContract_Identity(t *testing.T) {
	t.Parallel()
	contracttest.AssertIdentity(t,
		watcher.Contract(),
		watcher.Name, watcher.Roles)
}
