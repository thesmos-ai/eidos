// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package protogo

import "go.thesmos.sh/eidos/node"

// AnnotateStructForTest exposes the package-private
// [annotateStruct] for the in-package test surface. The
// black-box [protogo_test] package uses it to drive the
// idempotency contract against a hand-built node graph without
// the annotator-context plumbing.
func AnnotateStructForTest(s *node.Struct) { annotateStruct(s) }
