// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package sideeffect_test

import (
	"testing"

	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/internal/mixintest"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/sideeffect"
)

func TestMixin_Identity(t *testing.T) {
	t.Parallel()
	mixintest.AssertIdentity(t, sideeffect.Mixin(), sideeffect.Name, nil)
}
