// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package opt

// Holder binds a typed-options pointer to its reflected [Schema] and
// supplies the two methods that satisfy the option-provider contract
// — [Holder.OptionsSchema] and [Holder.SetOptions]. Plugin authors
// embed *Holder[T] on their plugin type so the plugin satisfies the
// contract without duplicating the boilerplate. Construct via [Bind].
//
//	type Plugin struct {
//	    *opt.Holder[Options]
//	    opts Options
//	}
//
//	func New() *Plugin {
//	    p := &Plugin{}
//	    p.Holder = opt.Bind(&p.opts)
//	    return p
//	}
//
// After Bind, [SetOptions] writes into the supplied target so the
// plugin reads its options through the same field it owns.
type Holder[T any] struct {
	schema Schema
	target *T
}

// Bind returns a *Holder bound to target. The reflected [Schema] is
// derived from *target at Bind time and cached on the returned
// holder so per-plugin schema reflection happens once per plugin
// instance.
//
// target must be a non-nil pointer to a struct of the type the
// generated schema describes — passing a value type is a programmer
// error and panics during the [Reflect] call.
func Bind[T any](target *T) *Holder[T] {
	return &Holder[T]{schema: Reflect(*target), target: target}
}

// OptionsSchema returns the cached reflected schema. Satisfies the
// [OptionsProvider.OptionsSchema] contract documented in the plugin
// package.
func (h *Holder[T]) OptionsSchema() Schema { return h.schema }

// SetOptions decodes opts into the bound target, surfacing any
// validation or assignment error from [Options.Decode] verbatim.
// Satisfies the [OptionsProvider.SetOptions] contract documented in
// the plugin package.
func (h *Holder[T]) SetOptions(opts Options) error {
	return opts.Decode(h.target)
}
