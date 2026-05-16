// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package sdk

import "go.thesmos.sh/eidos/core/kind"

// Kind is the structural discriminator both [node.Node] and
// [emit.Node] return from their Kind method, and the type
// [DirectiveSchema.AppliesTo] scopes against. Concrete kind
// constants live in their owning package
// ([node.KindStruct], [emit.KindStruct], plugin-defined
// constants for plugin-introduced emit shapes).
//
// Example targeting a directive to source structs and
// interfaces:
//
//	sdk.NewDirective("repo").
//	    On(node.KindStruct).
//	    On(node.KindInterface).
//	    Build()
type Kind = kind.Kind
