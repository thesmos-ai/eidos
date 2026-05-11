// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package cache

import "errors"

// ErrInvalidKey is returned by [Disk.Put] and [Disk.Get] when key is
// empty. Cache keys must be non-empty hex digests; an empty key is
// always a programmer error in the caller.
var ErrInvalidKey = errors.New("cache: invalid key")
