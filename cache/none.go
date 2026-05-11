// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package cache

// None is a no-op [Cache]: every Get is a miss and Put is a no-op.
// Use it in hermetic CI runs that should not consume or emit cache
// state, or when caching is explicitly disabled in configuration.
//
// None is concurrent-safe — it has no state.
type None struct{}

// NewNone returns a None cache. The constructor exists for symmetry
// with [NewDisk]; passing a zero-value None directly works equally.
func NewNone() *None { return &None{} }

// Get always returns (nil, false). The boolean indicates "not
// cached", which is the entire contract of the None cache.
func (*None) Get(string) ([]byte, bool) { return nil, false }

// Put always returns nil. The supplied value is discarded.
func (*None) Put(string, []byte) error { return nil }
