// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package emit

import "strings"

// Version is the semantic version of the [emit] contract. Bumping
// it signals compatibility expectations to plugins that declare
// [plugin.EmitVersioned]: a bump to the major number is a breaking
// change; a bump to the minor number is additive; a bump to the
// patch number is implementation-only.
const Version = "1.0.0"

// Major returns the leading numeric segment of [Version] — the
// value plugins compare against in their declared
// [plugin.EmitVersioned.EmitVersions] list. For Version "1.0.0"
// this returns "1". [strings.Cut] returns the whole string when no
// separator is present, so a hypothetical malformed Version like
// "1" still yields a usable major.
func Major() string {
	head, _, _ := strings.Cut(Version, ".")
	return head
}
