// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package referenceacceptance hosts the end-to-end acceptance test
// that drives every reference plugin together against the
// demoproject fixture. The package has no production surface — its
// purpose is to anchor a single test that exercises the
// cross-plugin interaction no individual plugin package can own
// on its own. Sibling to [eidostest/demopipe] (the harness) and
// [eidostest/demoproject] (the fixture).
package referenceacceptance
