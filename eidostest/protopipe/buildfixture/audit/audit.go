// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package audit is the toolchain-compile stub for the auditweaver
// prebody contribution. Rendered output from the proto-acceptance
// and toolchain-compile tests references `audit.Record(...)`; the
// stub gives those calls an existing target so `go vet ./...`
// against the buildfixture directory succeeds.
package audit

// Record is the audit-weaver's configured call target.
func Record(_ string, _ ...any) {}
