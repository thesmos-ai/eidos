// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package position_test

import "go.thesmos.sh/eidos/core/position"

// at builds a Pos with the four common dimensions populated. Tests use
// this rather than struct literals so adding a field to Pos surfaces in
// one place.
func at(file string, line, column, offset int) position.Pos {
	return position.Pos{File: file, Line: line, Column: column, Offset: offset}
}
