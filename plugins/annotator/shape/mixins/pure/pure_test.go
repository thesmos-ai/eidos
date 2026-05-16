// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package pure_test

import (
	"testing"

	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/internal/mixintest"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/pure"
)

func TestMixin_Identity(t *testing.T) {
	t.Parallel()
	mixintest.AssertIdentity(t, pure.Mixin(), pure.Name, nil)
}
