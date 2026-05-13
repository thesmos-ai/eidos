// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package buildfixture supplies a proto3 source plus the matching
// hand-written Go types the protobuf-frontend + protogo-bridge +
// Go-backend pipeline expects when rendering reference-plugin
// output for the fixture's messages. The fixture exercises the
// three composition rules the bridge handles past the trivial
// scalar pass-through:
//
//   - a well-known reference (`google.protobuf.Timestamp`),
//   - an optional field (proto3 explicit-presence),
//   - a nested-message reference (`User.Profile`).
//
// Rendered output is produced into this directory at test time
// (see [eidostest/protopipe.TestToolchain_GoBuildAgainstRenderedOutput])
// and removed by the test's cleanup. The committed Go source is
// the stubs alone, so `go build ./...` against the eidos module
// succeeds whether or not a test run is in progress.
package buildfixture
