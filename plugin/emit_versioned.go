// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package plugin

// EmitVersioned is the optional capability for plugins that declare
// which major versions of the [emit] contract they support. The
// pipeline checks compatibility at Build time: a plugin whose
// declared majors do not include [emit.Major] is rejected with a
// positioned diagnostic referencing both the plugin's declared
// versions and the in-tree emit major.
//
// Plugins that don't implement EmitVersioned are assumed compatible
// with every emit major — appropriate during development, where the
// contract has not yet hit a 1.0 release. After v1.0 ships,
// production plugins should declare their supported majors so
// breaking changes surface as Build errors rather than runtime
// surprises.
//
// Example:
//
//	func (p *Plugin) EmitVersions() []string { return []string{"1"} }
type EmitVersioned interface {
	// EmitVersions returns the [emit] major versions this plugin
	// supports. Strings, not integers, so future schemes ("1-rc1",
	// pre-release tags) compose without a type change.
	EmitVersions() []string
}
