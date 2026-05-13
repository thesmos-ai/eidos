// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package extras supplies the cross-package Go stub the
// buildfixture proto references through its
// `import "extras.proto";` directive. Rendered output from the
// proto-acceptance and toolchain compile tests references
// extras.Tag through the bridge-translated Go import path; the
// stub gives those references an existing target so the
// compilation check succeeds.
package extras
