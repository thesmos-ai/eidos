// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package cacheable_test

import (
	"testing"

	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/cacheable"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/internal/mixintest"
)

func TestMixin_Identity(t *testing.T) {
	t.Parallel()
	mixintest.AssertIdentity(t, cacheable.Mixin(), cacheable.Name, nil)
}
