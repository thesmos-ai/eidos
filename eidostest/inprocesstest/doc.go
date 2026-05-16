// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package inprocesstest drives end-to-end scenarios through the
// in-process pipeline harness ([pipelinetest]) — wiring a real
// frontend, real generators, and a real backend to render against
// a self-contained fixture module on disk, then compiling the
// rendered output with `go vet` to verify byte-level correctness.
//
// The layer above this is [acceptancetest], which exercises the
// reference binary as a black box; the layer below is
// [pipelinetest], which builds in-process pipelines with stub
// plugins. inprocesstest sits between them — full pipeline plus
// real plugins, but no separate binary. Use it for scenarios
// where the rendered Go must compile (so a backend is required)
// and where speed matters (so a subprocess does not).
//
// Tests in this package may import any frontend, backend, and
// production plugin. That privilege keeps it firmly outside
// `plugins/...`, where the depguard rule forbids those imports
// to preserve plugin / backend independence.
package inprocesstest
