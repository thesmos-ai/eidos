// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package streamreflectsmutations_test

import (
	"testing"

	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/internal/mixintest"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/streamreflectsmutations"
)

func TestMixin_Identity(t *testing.T) {
	t.Parallel()
	mixintest.AssertIdentity(t, streamreflectsmutations.Mixin(), streamreflectsmutations.Name, nil)
}
