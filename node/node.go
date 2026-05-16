// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package node

import (
	"go.thesmos.sh/eidos/core/contract"
)

// Node is the common interface every concrete source-side node
// kind satisfies. The shape — a kind discriminator, source
// position, documentation, directives, and a metadata bag — is
// identical to [contract.Node]; Node is exposed as a type alias
// so existing callers continue to spell `node.Node` while
// cross-graph framework components (the routing layer, the
// owner-resolve pass, plugin author cookbooks) can speak
// `contract.Node` and accept either source-side or emit-side
// values.
//
// Kind-specific accessors (Methods, Fields, Params, …) live on
// the concrete types and are reached via type assertion or
// [Walk].
type Node = contract.Node
