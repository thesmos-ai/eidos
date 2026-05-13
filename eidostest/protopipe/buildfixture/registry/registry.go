// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package registry is the toolchain-compile stub for the
// registrygen registration call. Rendered output references
// `registry.Register(...)`; the stub gives those calls an
// existing target so `go vet ./...` against the buildfixture
// directory succeeds.
package registry

// Register is the registry-gen's configured call target.
func Register(_ string, _ any) {}
