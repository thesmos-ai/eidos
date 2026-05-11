// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package writer

import "errors"

// ErrAliasAfterImp is returned by [ImportSet.Alias] when an
// explicit alias is registered for a path that already has an
// auto-derived alias assigned by a prior [ImportSet.Imp] call.
// Alias overrides must arrive before the path is first imported
// so the rest of the file uses the chosen alias consistently.
var ErrAliasAfterImp = errors.New("writer: alias registered after path was already imported")

// ErrEmptyPath is returned by [ImportSet.Imp] and [ImportSet.Alias]
// when path is the empty string. Imports are keyed by path and an
// empty path has no resolvable alias.
var ErrEmptyPath = errors.New("writer: empty import path")
