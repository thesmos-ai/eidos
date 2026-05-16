// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package concurrentreaders recognises the concurrent-readers
// mixin — the assertion that the annotated callable is safe to
// invoke from multiple readers concurrently, even when writers
// are not safe to mix in.
//
// The recognised directive is:
//
//	//+gen:mixin concurrentreaders
package concurrentreaders
