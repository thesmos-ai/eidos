// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package manifest

import "errors"

// ErrUnsupportedVersion is returned by [Read] when the on-disk
// manifest reports a [Manifest.Version] this build does not
// understand. Older readers refuse to interpret newer formats
// rather than silently dropping fields they don't know about.
var ErrUnsupportedVersion = errors.New("manifest: unsupported version")
