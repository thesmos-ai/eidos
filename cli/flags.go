// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package cli

// Flag-name constants. The cli package registers flags into a
// caller-supplied [flag.FlagSet] using these names; consumers
// using other CLI frameworks (cobra, kong, urfave-cli) reference
// the same constants when declaring equivalent flags so the
// canonical spelling stays consistent across binaries.
//
// Per-command flag-name sets:
//
//   - Every command: FlagConfig, FlagDiagFormat, FlagVerbose, FlagQuiet
//   - Run: + FlagNoCache, FlagVerifyCache, FlagTarget, FlagOutput
//   - Plan, Version: (none beyond the common set)
//   - Check, Prune: (none beyond the common set; Prune adds FlagDryRun)
//   - Explain: (none beyond the common set; selector is positional)
const (
	// FlagConfig is the path to the config file (overrides
	// .<brand>.yaml discovery).
	FlagConfig = "config"

	// FlagDiagFormat selects the diagnostic output format
	// ("text" or "json").
	FlagDiagFormat = "diag-format"

	// FlagVerbose surfaces Info diagnostics.
	FlagVerbose = "verbose"

	// FlagQuiet suppresses Warn diagnostics.
	FlagQuiet = "quiet"

	// FlagNoCache disables the build cache for a single invocation
	// (overrides the file's cache.enabled).
	FlagNoCache = "no-cache"

	// FlagVerifyCache recomputes output and asserts byte-identity
	// against the cached value.
	FlagVerifyCache = "verify-cache"

	// FlagTarget restricts the pipeline to a single scope:
	// "file=...", "interface=...", "struct=...", "package=...".
	FlagTarget = "target"

	// FlagOutput overrides the rendered filename for the entity
	// matched by the explain / run selector. Used by `go:generate`
	// style one-shot invocations.
	FlagOutput = "o"

	// FlagDryRun reports planned actions without performing them
	// (currently used by `prune`).
	FlagDryRun = "dry-run"
)

// Usage strings. Centralised so the wording stays consistent
// across the reference binary and any downstream binary that
// re-declares flags with another CLI framework.
const (
	UsageConfig      = "Path to the config file. Defaults to .<brand>.yaml discovered upward from the working directory."
	UsageDiagFormat  = "Diagnostic output format. One of: text, json."
	UsageVerbose     = "Surface Info diagnostics."
	UsageQuiet       = "Suppress Warn diagnostics."
	UsageNoCache     = "Disable the build cache for this invocation."
	UsageVerifyCache = "Recompute output and assert byte-identity against the cached value."
	UsageTarget      = "Restrict the pipeline to a single scope (file=, interface=, struct=, package=)."
	UsageOutput      = "Override the rendered filename for the matched entity."
	UsageDryRun      = "Report planned actions without performing them."
)
