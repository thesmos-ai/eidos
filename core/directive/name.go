// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package directive

// Name is the bare name of a directive — what follows the "+gen:"
// or "-gen:" prefix and precedes the first whitespace. Defined as
// its own string type so [Schema], [Directive], and [Registry]
// signatures are self-documenting; literal strings convert
// implicitly.
type Name string
