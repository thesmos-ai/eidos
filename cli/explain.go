// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"sort"
	"strings"

	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/pipeline"
	"go.thesmos.sh/eidos/plugin"
	"go.thesmos.sh/eidos/sink"
	"go.thesmos.sh/eidos/store"
)

// ExplainConfig holds the inputs for [ExplainCommand]. The
// selector identifies the entity, slot, or meta key the command
// inspects.
type ExplainConfig struct {
	File     *Config
	Plugins  []plugin.Plugin
	Selector string
	Format   DiagFormat
	Verbose  bool
	Quiet    bool
}

// ExplainCommand prints a provenance trace for an entity, slot,
// or meta key. Runs the pipeline against an in-memory sink so the
// inspection is non-destructive.
//
// Selector forms:
//
//   - `pkg.QName` — entity by qualified name.
//   - `pkg.QName:slot` — slot contributions on that entity.
//   - `pkg.QName#key` — meta-key trail on that entity.
type ExplainCommand struct{ Config ExplainConfig }

// RegisterFlags binds [ExplainCommand]'s flags into fs. The
// selector is a positional argument, not a flag.
func (c *ExplainCommand) RegisterFlags(fs *flag.FlagSet) {
	fs.Var(&c.Config.Format, FlagDiagFormat, UsageDiagFormat)
	fs.BoolVar(&c.Config.Verbose, FlagVerbose, false, UsageVerbose)
	fs.BoolVar(&c.Config.Quiet, FlagQuiet, false, UsageQuiet)
}

// Execute resolves the selector against the emit graph and prints
// the corresponding trace.
func (c *ExplainCommand) Execute(ctx context.Context, env *Env) (exit int) {
	defer recoverInto(env, &exit)

	if c.Config.Selector == "" {
		writeErr(env, "explain: selector is required "+
			"(e.g. \"users.User\", \"users.User:methods\", \"users.User#shape.writer\")")
		return ExitUserError
	}
	sel, err := parseSelector(c.Config.Selector)
	if err != nil {
		writeErr(env, "explain: %v", err)
		return ExitUserError
	}
	cfg := c.Config.File
	if cfg == nil {
		cfg = DefaultConfig()
	}
	p, err := buildPipeline(env, cfg, c.Config.Plugins, pipelineOverride{
		Verbose:      c.Config.Verbose,
		SinkOverride: sink.NewMemory(),
	})
	if err != nil {
		writeErr(env, "%v", err)
		return ExitUserError
	}
	runErr := p.Run(ctx, patternsOrDefault(cfg)...)
	rerr := RenderDiagnostics(env.Stderr, p.Diag(), c.Config.Format, c.Config.Verbose, c.Config.Quiet)
	if rerr != nil {
		writeErr(env, "%v", rerr)
	}
	if runErr != nil && !errors.Is(runErr, pipeline.ErrRunHadErrors) {
		writeErr(env, "%v", runErr)
		return ExitPipelineError
	}
	return c.report(env, p.Store(), sel)
}

// selectorKind discriminates the three selector forms.
type selectorKind int

const (
	selectorEntity selectorKind = iota
	selectorSlot
	selectorMeta
)

// selector is a parsed explain target. Kind selects which Anchor
// + Modifier interpretation applies:
//
//   - selectorEntity: Anchor is "<pkg.QName>", Modifier empty.
//   - selectorSlot:   Anchor is "<pkg.QName>", Modifier is the
//     slot name.
//   - selectorMeta:   Anchor is "<pkg.QName>", Modifier is the
//     meta-key name.
type selector struct {
	Kind     selectorKind
	Anchor   string
	Modifier string
}

// parseSelector accepts one of the three documented forms and
// produces a [selector]. The pkg-and-qname portion is the anchor;
// the optional `:slot` or `#key` suffix is the modifier.
//
// The first colon or hash decides the modifier kind. An empty
// anchor or empty modifier returns a user error so the cli
// surfaces a positioned diagnostic rather than silently producing
// an unhelpful empty trace.
func parseSelector(raw string) (selector, error) {
	if raw == "" {
		return selector{}, errors.New("empty selector")
	}
	if anchor, mod, ok := strings.Cut(raw, "#"); ok {
		if anchor == "" || mod == "" {
			return selector{}, fmt.Errorf("meta-key selector %q: anchor and key are both required", raw)
		}
		return selector{Kind: selectorMeta, Anchor: anchor, Modifier: mod}, nil
	}
	if anchor, mod, ok := strings.Cut(raw, ":"); ok {
		if anchor == "" || mod == "" {
			return selector{}, fmt.Errorf("slot selector %q: anchor and slot name are both required", raw)
		}
		return selector{Kind: selectorSlot, Anchor: anchor, Modifier: mod}, nil
	}
	return selector{Kind: selectorEntity, Anchor: raw}, nil
}

// report dispatches on the selector kind and prints the matching
// trace. An unresolvable selector exits ExitUserError so CI gates
// distinguish "I can't find this" from "the pipeline failed".
func (c *ExplainCommand) report(env *Env, s *store.Store, sel selector) int {
	entity := findEntity(s, sel.Anchor)
	if entity == nil {
		writeErr(env, "explain: %q resolves to no emit entity", sel.Anchor)
		return ExitUserError
	}
	switch sel.Kind {
	case selectorEntity:
		return c.reportEntity(env, entity)
	case selectorSlot:
		return c.reportSlot(env, entity, sel.Modifier)
	case selectorMeta:
		return c.reportMeta(env, entity, sel.Modifier)
	default:
		writeErr(env, "explain: unknown selector kind %d", sel.Kind)
		return ExitUserError
	}
}

// findEntity walks every emit-store bucket for an entity whose
// QName equals anchor. Returns the first match or nil.
//
// Multiple kinds may carry the same QName (a struct and an alias
// with identical names, for example); the function picks the
// first match in bucket-iteration order, which is deterministic
// across runs but doesn't disambiguate. Callers needing
// disambiguation should pass a kind-qualified selector — out of
// scope for this iteration.
func findEntity(s *store.Store, anchor string) emit.Node {
	if s == nil {
		return nil
	}
	v := s.Emit()
	var found emit.Node
	scan := func(n emit.Node, qname string) bool {
		if qname == anchor {
			found = n
			return false
		}
		return true
	}
	v.Structs().Range(func(e *emit.Struct) bool { return scan(e, e.QName()) })
	if found != nil {
		return found
	}
	v.Interfaces().Range(func(e *emit.Interface) bool { return scan(e, e.QName()) })
	if found != nil {
		return found
	}
	v.Functions().Range(func(e *emit.Function) bool { return scan(e, e.QName()) })
	if found != nil {
		return found
	}
	v.Variables().Range(func(e *emit.Variable) bool { return scan(e, e.QName()) })
	if found != nil {
		return found
	}
	v.Constants().Range(func(e *emit.Constant) bool { return scan(e, e.QName()) })
	if found != nil {
		return found
	}
	v.Enums().Range(func(e *emit.Enum) bool { return scan(e, e.QName()) })
	if found != nil {
		return found
	}
	v.Aliases().Range(func(e *emit.Alias) bool { return scan(e, e.QName()) })
	return found
}

// reportEntity prints kind / origin / directive count / meta-key
// count for the entity. Mirrors the spec's sketched output shape.
func (*ExplainCommand) reportEntity(env *Env, n emit.Node) int {
	w := env.Stdout
	fmt.Fprintf(w, "Kind:       %s\n", n.Kind())
	if origin := n.Origin(); origin != nil {
		pos := origin.Pos()
		if pos.File != "" {
			fmt.Fprintf(w, "Origin:     %s\n", pos.String())
		}
	}
	if dirs := n.Directives(); len(dirs) > 0 {
		fmt.Fprintln(w, "Directives:")
		for _, d := range dirs {
			fmt.Fprintf(w, "  - %s\n", d.Name)
		}
	}
	if meta := n.Meta(); meta != nil {
		names := meta.Names()
		if len(names) > 0 {
			sort.Strings(names)
			fmt.Fprintln(w, "Meta keys:")
			for _, name := range names {
				fmt.Fprintf(w, "  - %s\n", name)
			}
		}
	}
	return ExitOK
}

// reportSlot prints the contributions of a named slot on the
// entity, each line prefixed with its index and the
// Provenance.SetBy / Provenance.ID values when present.
func (*ExplainCommand) reportSlot(env *Env, host emit.Node, name string) int {
	owner, ok := host.(emit.SlotHost)
	if !ok {
		writeErr(env, "explain: %s does not own slots", host.Kind())
		return ExitUserError
	}
	slot := owner.Slot(name)
	if slot == nil || slot.Len() == 0 {
		writeErr(env, "explain: slot %q has no contributions", name)
		return ExitUserError
	}
	fmt.Fprintf(env.Stdout, "Slot %q (%d contributions):\n", name, slot.Len())
	for i := range slot.Len() {
		prov := slot.ProvenanceAt(i)
		setBy := prov.SetBy
		if setBy == "" {
			setBy = "(unattributed)"
		}
		idSuffix := ""
		if prov.ID != "" {
			idSuffix = " id=" + prov.ID
		}
		fmt.Fprintf(env.Stdout, "  [%d] set by %s%s\n", i, setBy, idSuffix)
	}
	return ExitOK
}

// reportMeta prints the meta-key's value and authority on the
// entity. Unset keys exit ExitUserError so CI gates can detect
// "I asked about a key that doesn't exist".
func (*ExplainCommand) reportMeta(env *Env, n emit.Node, key string) int {
	meta := n.Meta()
	if meta == nil {
		writeErr(env, "explain: entity has no meta bag")
		return ExitUserError
	}
	v, ok := meta.RawValue(key)
	if !ok {
		writeErr(env, "explain: meta key %q is unset on %s", key, n.Kind())
		return ExitUserError
	}
	fmt.Fprintf(env.Stdout, "Meta %q = %v\n", key, v)
	return ExitOK
}
