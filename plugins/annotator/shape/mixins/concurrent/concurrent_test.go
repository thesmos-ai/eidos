// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package concurrent_test

import (
	"testing"

	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/concurrent"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/internal/mixintest"
)

func TestMixin_Identity(t *testing.T) {
	t.Parallel()
	mixintest.AssertIdentity(t, concurrent.Mixin(), concurrent.Name, nil)
}
