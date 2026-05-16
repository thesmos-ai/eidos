// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// preflightWorkspaceCheck emits a notice to env.Stderr when the
// binary is invoked inside a Go module whose directory is NOT
// listed in the nearest enclosing go.work's `use ()` block. This
// is the canonical configuration for testdata fixtures parked
// inside a parent project, and it produces a confusing low-level
// loader error (`pattern ./...: directory prefix . does not
// contain modules listed in go.work`) without this nudge.
//
// Called from [buildPipeline] so every source-loading command
// (run / plan / check / explain / prune) inherits the warning
// without each binary author wiring it up. The version command
// builds no pipeline and so naturally skips the check.
//
// The check is best-effort — any failure to read or parse the
// workspace silently no-ops. Pre-flight UX is sugar; the
// pipeline still runs and produces its real diagnostic if the
// loader rejects the layout.
func preflightWorkspaceCheck(env *Env) {
	if os.Getenv("GOWORK") == "off" {
		// User has explicitly opted out of workspace
		// resolution; the loader will not consult go.work at
		// all, so there's no conflict to warn about.
		return
	}
	modDir, ok := findUp(env.Workdir, "go.mod")
	if !ok {
		return
	}
	workPath, ok := findUp(modDir, "go.work")
	if !ok {
		// No enclosing workspace; the module stands alone, no
		// conflict possible.
		return
	}
	if filepath.Dir(workPath) == modDir {
		// The workspace lives at the same level as the module —
		// the workspace IS the module's own. No conflict.
		return
	}
	listed, ok := moduleInWorkspaceUse(workPath, modDir)
	if !ok || listed {
		return
	}
	emitWorkspaceNotice(env, modDir, workPath)
}

// findUp walks up from start looking for the first directory
// containing a file named filename. Returns the directory path
// (not the file path) and whether a hit was found.
func findUp(start, filename string) (string, bool) {
	if start == "" {
		return "", false
	}
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", false
	}
	for {
		if _, statErr := os.Stat(filepath.Join(dir, filename)); statErr == nil {
			if filename == "go.work" {
				return filepath.Join(dir, filename), true
			}
			return dir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

// moduleInWorkspaceUse parses the workfile at workPath and
// reports whether modDir is listed in any `use ()` directive.
// The second return is false on read / parse failure; the caller
// treats that as "unknown, suppress the warning."
//
// The parser is deliberately minimal — it understands the
// canonical `use (` … `)` block and bare `use <path>` lines,
// strips comments, and matches absolute paths after resolving
// each entry against workPath's directory. Anything stranger
// (continuation lines, conditional uses) is out of scope; this
// is a best-effort UX nudge, not a definitive parser.
func moduleInWorkspaceUse(workPath, modDir string) (bool, bool) {
	f, err := os.Open(workPath) //nolint:gosec // workPath came from findUp; controlled.
	if err != nil {
		return false, false
	}
	defer func() { _ = f.Close() }()

	workDir := filepath.Dir(workPath)
	modAbs, err := filepath.Abs(modDir)
	if err != nil {
		return false, false
	}

	scanner := bufio.NewScanner(f)
	inBlock := false
	for scanner.Scan() {
		line := stripComment(strings.TrimSpace(scanner.Text()))
		if line == "" {
			continue
		}
		switch {
		case strings.HasPrefix(line, "use ("):
			inBlock = true
		case line == ")" && inBlock:
			inBlock = false
		case strings.HasPrefix(line, "use "):
			// Single-line `use <path>` form.
			entry := strings.TrimSpace(strings.TrimPrefix(line, "use "))
			if pathMatches(workDir, entry, modAbs) {
				return true, true
			}
		case inBlock:
			if pathMatches(workDir, line, modAbs) {
				return true, true
			}
		}
	}
	if scanErr := scanner.Err(); scanErr != nil {
		return false, false
	}
	return false, true
}

// stripComment drops a trailing `// …` comment from line and
// trims residual whitespace.
func stripComment(line string) string {
	if i := strings.Index(line, "//"); i >= 0 {
		line = line[:i]
	}
	return strings.TrimSpace(line)
}

// pathMatches reports whether an entry (relative or absolute)
// from a go.work use directive resolves to modAbs when resolved
// against workDir. Trailing whitespace and surrounding quotes
// are tolerated.
func pathMatches(workDir, entry, modAbs string) bool {
	entry = strings.TrimSpace(entry)
	entry = strings.Trim(entry, `"`)
	if entry == "" {
		return false
	}
	resolved := entry
	if !filepath.IsAbs(resolved) {
		resolved = filepath.Join(workDir, entry)
	}
	abs, err := filepath.Abs(resolved)
	if err != nil {
		return false
	}
	return filepath.Clean(abs) == filepath.Clean(modAbs)
}

// emitWorkspaceNotice writes the user-facing notice to
// env.Stderr. Format matches the binary's diagnostic style and
// names the fix paths explicitly.
func emitWorkspaceNotice(env *Env, modDir, workPath string) {
	relWork, err := filepath.Rel(modDir, workPath)
	if err != nil {
		relWork = workPath
	}
	relMod, err := filepath.Rel(filepath.Dir(workPath), modDir)
	if err != nil {
		relMod = modDir
	}
	fmt.Fprintf(env.Stderr,
		"%s: notice: this directory's go.mod (%s) is not listed in the workspace at %s.\n"+
			"  the Go frontend's packages.Load will likely fail with\n"+
			"  'pattern ./...: directory prefix . does not contain modules listed in go.work'.\n"+
			"  fix: set GOWORK=off in your environment, OR add `ignore_workspace: \"true\"` to\n"+
			"  the golang frontend's options in %s, OR add %q to the workspace's use() block.\n",
		env.Brand, modDir, relWork, env.ConfigFileName(), relMod)
}
