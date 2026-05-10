// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package directive

// Kind is the symbolic name of a node kind a directive can attach to
// ("struct", "interface", "method", and so on).
//
// Defined as its own string type so [Schema.AppliesTo] is self-
// describing at call sites. The directive package does not enumerate
// kinds itself — downstream node-model packages declare their kind
// constants as Kind values, and this package stores them by string
// equality without needing to import node.
type Kind string
