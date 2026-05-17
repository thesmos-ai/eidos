// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package sdk

import "go.thesmos.sh/eidos/core/opt"

// Holder is the canonical typed-options holder plugins embed
// so they satisfy [plugin.OptionsProvider] without
// duplicating the schema / SetOptions boilerplate. Construct
// via [BindOptions].
//
//	type Plugin struct {
//	    *sdk.Holder[Options]
//	    opts Options
//	}
//
//	func New() *Plugin {
//	    p := &Plugin{}
//	    p.Holder = sdk.BindOptions(&p.opts)
//	    return p
//	}
type Holder[T any] = opt.Holder[T]

// BindOptions returns a [*Holder] bound to target. The
// reflected schema is derived from *target at Bind time and
// cached on the returned holder so subsequent
// [Holder.OptionsSchema] / [Holder.SetOptions] calls reuse
// it.
func BindOptions[T any](target *T) *Holder[T] {
	return opt.Bind(target)
}
