// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package integrationonly_test

import (
	"testing"

	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/integrationonly"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/internal/mixintest"
)

func TestMixin_Identity(t *testing.T) {
	t.Parallel()
	mixintest.AssertIdentity(t, integrationonly.Mixin(), integrationonly.Name, nil)
}
