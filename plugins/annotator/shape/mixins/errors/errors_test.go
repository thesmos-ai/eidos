// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package errors_test

import (
	"testing"

	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/errors"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/internal/mixintest"
)

func TestMixin_Identity(t *testing.T) {
	t.Parallel()
	mixintest.AssertIdentity(t, errors.Mixin(), errors.Name, nil)
}
