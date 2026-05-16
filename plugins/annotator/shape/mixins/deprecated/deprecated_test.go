// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package deprecated_test

import (
	"testing"

	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/deprecated"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/internal/mixintest"
)

func TestMixin_Identity(t *testing.T) {
	t.Parallel()
	mixintest.AssertIdentity(t, deprecated.Mixin(), deprecated.Name, nil)
}
