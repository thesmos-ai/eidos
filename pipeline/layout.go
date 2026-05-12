// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package pipeline

import (
	"cmp"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"go.thesmos.sh/eidos/core/diag"
	"go.thesmos.sh/eidos/core/position"
	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/manifest"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/plugin"
	"go.thesmos.sh/eidos/store"
)

// runLayout resolves [emit.Target] on every routable emit entity in
// the store and materialises pending origin-anchored slot
// contributions onto the resulting files. The phase runs after the
// generator phase and before [store.EmitView.Freeze]; it is the
// single place where output routing decisions are made.
//
// The phase walks every routable emit kind through an explicit
// per-kind dispatch, composes a [emit.Target] per decl from the
// precedence layers — framework default, plugin filename suffix,
// resolved layout policy, source-side `+gen:out` directive, CLI
// overrides — then drains the pending origin-anchored slot list
// and routes each tuple through the same composition. The
// one-file-one-package invariant is enforced once both passes
// complete. The byTarget index is rebuilt unconditionally so the
// backend's file grouping observes the resolved routing — even
// when the phase recorded routing errors, the rebuild excludes
// the offending Targets and renders the non-conflicting subset.
//
// Routing errors surface as Error diagnostics attributed to the
// pipeline's "pipeline.layout" channel. The run continues; the
// offending decls drop from the byTarget index so the backend
// skips them but remain in their per-kind buckets so debugging
// tools can still resolve them.
func (p *Pipeline) runLayout(s *store.Store) {
	ps := p.diag.For("pipeline.layout")
	v := s.Emit()
	suffixes := p.collectFilenameSuffixes()

	routeDecls(s, v, ps, suffixes, p)
	materialiseOriginSlots(s, v, ps, suffixes, p)
	enforceOneFileOnePackage(v, ps)
	v.RebuildByTarget()
}

// routeDecls dispatches per emit kind and stamps the resolved
// Target onto each routable decl. The dispatch table is the
// complete list of framework-defined routable kinds; adding a new
// kind requires registering it here explicitly. [emit.Alias] is
// special-cased because it stores its file routing in `.File`
// rather than `.Target` (the alias's `.Target` field holds the
// aliased type Ref).
func routeDecls(
	s *store.Store,
	v *store.EmitView,
	ps *diag.PluginSink,
	suffixes map[string]string,
	p *Pipeline,
) {
	v.Structs().Range(func(e *emit.Struct) bool {
		e.Target = composeOrZero(s, p, ps, suffixes, e.SetBy(), e.Origin(), "struct", e.QName())
		return true
	})
	v.Interfaces().Range(func(e *emit.Interface) bool {
		e.Target = composeOrZero(s, p, ps, suffixes, e.SetBy(), e.Origin(), "interface", e.QName())
		return true
	})
	v.Functions().Range(func(e *emit.Function) bool {
		e.Target = composeOrZero(s, p, ps, suffixes, e.SetBy(), e.Origin(), "function", e.QName())
		return true
	})
	v.Variables().Range(func(e *emit.Variable) bool {
		e.Target = composeOrZero(s, p, ps, suffixes, e.SetBy(), e.Origin(), "variable", e.QName())
		return true
	})
	v.Constants().Range(func(e *emit.Constant) bool {
		e.Target = composeOrZero(s, p, ps, suffixes, e.SetBy(), e.Origin(), "constant", e.QName())
		return true
	})
	v.Enums().Range(func(e *emit.Enum) bool {
		e.Target = composeOrZero(s, p, ps, suffixes, e.SetBy(), e.Origin(), "enum", e.QName())
		return true
	})
	v.Aliases().Range(func(e *emit.Alias) bool {
		e.File = composeOrZero(s, p, ps, suffixes, e.SetBy(), e.Origin(), "alias", e.QName())
		return true
	})
}

// composeOrZero is the per-kind dispatch's stamp helper: it routes
// a single decl through the precedence pipeline, records the
// resolved [manifest.ResolvedLayout] on the pipeline for manifest
// composition, and returns the composed Target — or the zero
// Target when the decl cannot be routed. Routing failures
// (synthetic Origin, missing [plugin.FilenameProvider], unknown
// SetBy attribution) surface an Error diagnostic via ps before the
// helper returns the zero Target. The zero return value drops the
// decl from the byTarget rebuild and from the manifest; the
// per-kind bucket still retains it so debugging tools can still
// resolve it.
func composeOrZero(
	s *store.Store,
	p *Pipeline,
	ps *diag.PluginSink,
	suffixes map[string]string,
	setBy string,
	origin node.Node,
	kind, qname string,
) emit.Target {
	if origin == nil {
		ps.Errorf(position.Pos{},
			"synthetic %s %q has no Origin; cannot route", kind, qname)
		return emit.Target{}
	}
	suffix, ok := suffixes[setBy]
	if !ok {
		ps.Errorf(position.Pos{},
			"%s: %s %q emitted by %q",
			ErrMissingFilenameProvider.Error(), kind, qname, setBy)
		return emit.Target{}
	}
	t, rl, ok := composeTarget(s, p, setBy, suffix, origin)
	if !ok {
		return emit.Target{}
	}
	p.recordResolvedLayout(t, rl)
	return t
}

// composeTarget runs the precedence pipeline for one (plugin,
// origin) pair. Layer order, lowest priority first:
//
//  1. Framework default: alongside-source layout — Dir from
//     filepath.Dir(origin.Pos().File); Package + ImportPath from the
//     origin's source package.
//  2. Plugin filename suffix: Filename = <src-basename><suffix>.
//  3. Resolved layout policy: switches to centralised when policy
//     selects it; under alongside-source a non-empty policy.Package
//     overrides Package without changing Dir.
//  4. Per-source `+gen:out` directive on origin: overrides Filename.
//  5. CLI -o: overrides Filename for every routed decl in scope.
//  6. CLI -p (folded into policy.Package): wins for Package
//     unconditionally when set.
//
// composeTarget is the single helper both decl routing and pending
// slot-tuple routing call so the two paths cannot drift. The
// returned [manifest.ResolvedLayout] records the per-field
// precedence-layer attribution for the manifest's observability
// block.
func composeTarget(
	s *store.Store,
	p *Pipeline,
	pluginName, suffix string,
	origin node.Node,
) (emit.Target, manifest.ResolvedLayout, bool) {
	policy := p.LayoutPolicyFor(pluginName)
	srcPkg := originSourcePackage(s, origin)
	srcDir, basename := originSourceDirBasename(origin)

	var t emit.Target
	rl := manifest.ResolvedLayout{
		Layout:       policy.Layout,
		ResolvedFrom: map[string]manifest.Layer{},
	}

	// Filename: framework basename + plugin-declared suffix. The
	// suffix is guaranteed non-empty here because composeOrZero /
	// materialiseOriginSlots filter empty entries out of the
	// FilenameProvider lookup table; reaching this point with an
	// empty suffix is a framework bug.
	t.Filename = basename + suffix
	rl.ResolvedFrom["filename"] = manifest.LayerPluginSuffix
	rl.ResolvedFrom["layout"] = policy.LayoutFrom

	// Dir + Package + ImportPath per resolved policy. The switch
	// is exhaustive: an unrecognised Layout value is a framework
	// bug — Builder.resolveLayoutPolicy is the only producer and
	// normalises to one of the two valid values.
	switch policy.Layout {
	case LayoutCentralised:
		t.Dir = policy.Dir
		rl.ResolvedFrom["dir"] = policy.DirFrom
		if t.Dir == "" {
			// Dir defaults from Package when unset. The
			// attribution follows the Package source since that
			// is what supplied the directory's final value.
			t.Dir = policy.Package
			rl.ResolvedFrom["dir"] = policy.PackageFrom
		}
		t.Package = policy.Package
		rl.ResolvedFrom["package"] = policy.PackageFrom
		// Centralised ImportPath is not derivable without a module
		// root; left empty so the renderer's same-package elision
		// stays inert under centralised layout until config carries
		// the value forward.
	case LayoutAlongsideSource:
		t.Dir = srcDir
		rl.ResolvedFrom["dir"] = manifest.LayerFramework
		if srcPkg != nil {
			t.Package = srcPkg.Name
			t.ImportPath = srcPkg.Path
		}
		rl.ResolvedFrom["package"] = manifest.LayerFramework
		// A non-empty policy.Package under alongside-source pins
		// the package without changing the directory. Attribution
		// follows the layer that supplied the Package value.
		if policy.Package != "" {
			t.Package = policy.Package
			rl.ResolvedFrom["package"] = policy.PackageFrom
		}
	default:
		p.diag.Internalf(position.Pos{},
			"pipeline.layout: unknown layout %q for plugin %q; cannot route",
			policy.Layout, pluginName)
		return emit.Target{}, manifest.ResolvedLayout{}, false
	}

	// +gen:out directive on origin overrides Filename.
	if fn, ok := outDirectiveFilename(origin); ok {
		t.Filename = fn
		rl.ResolvedFrom["filename"] = manifest.LayerDirective
	}

	// CLI -o overrides Filename for every decl in scope.
	if p.OutputFilename() != "" {
		t.Filename = p.OutputFilename()
		rl.ResolvedFrom["filename"] = manifest.LayerCLI
	}

	rl.Package = t.Package
	rl.Dir = t.Dir
	rl.Filename = t.Filename
	return t, rl, true
}

// materialiseOriginSlots drains the pending origin-anchored slot
// list, composes a Target for each tuple's origin via the same
// helper that decl routing uses, and appends each item to the
// matching named slot on the resolved [emit.File]. Iteration order
// is the registration order on the pending list — the EmitView
// preserves insertion order on append, and the generator phase
// iterates source nodes in store insertion order, so two runs
// over the same input produce the same materialisation order.
//
// A tuple whose contributing plugin lacks a filename suffix
// surfaces [ErrMissingFilenameProvider]; the tuple is dropped
// and the run continues.
func materialiseOriginSlots(
	s *store.Store,
	v *store.EmitView,
	ps *diag.PluginSink,
	suffixes map[string]string,
	p *Pipeline,
) {
	pending := v.PendingOriginSlots()
	for _, tup := range pending {
		setBy := tup.Prov.SetBy
		suffix, ok := suffixes[setBy]
		if !ok {
			ps.Errorf(position.Pos{},
				"%s: slot %q contribution from %q",
				ErrMissingFilenameProvider.Error(),
				tup.SlotName, setBy)
			continue
		}
		target, rl, ok := composeTarget(s, p, setBy, suffix, tup.Origin)
		if !ok {
			continue
		}
		p.recordResolvedLayout(target, rl)
		// FileFor only errors when the view is frozen; Layout runs
		// pre-freeze so the lookup-or-create cannot fail here.
		file, _ := v.FileFor(target)
		if err := file.Slot(tup.SlotName).Append(tup.Item, tup.Prov); err != nil {
			ps.Errorf(position.Pos{},
				"slot %q: append failed on %s: %v",
				tup.SlotName, target.JoinPath(), err)
		}
	}
}

// enforceOneFileOnePackage walks every routable emit entity, groups
// by (Dir, Filename), and surfaces an Error diagnostic for any
// group whose members disagree on Package. The offending decls have
// their Target cleared so the byTarget rebuild below excludes them;
// the manifest sink correspondingly omits the conflicted Target.
// The run continues for non-conflicting Targets.
func enforceOneFileOnePackage(v *store.EmitView, ps *diag.PluginSink) {
	type slot struct {
		key  string
		pkgs map[string][]string // package → qnames carrying that package value
	}
	groups := map[string]*slot{}
	visit := func(t emit.Target, qname string) {
		if t.Dir == "" || t.Filename == "" {
			return
		}
		key := t.Dir + "/" + t.Filename
		g, ok := groups[key]
		if !ok {
			g = &slot{key: key, pkgs: map[string][]string{}}
			groups[key] = g
		}
		g.pkgs[t.Package] = append(g.pkgs[t.Package], qname)
	}
	v.Structs().Range(func(e *emit.Struct) bool { visit(e.Target, e.QName()); return true })
	v.Interfaces().Range(func(e *emit.Interface) bool { visit(e.Target, e.QName()); return true })
	v.Functions().Range(func(e *emit.Function) bool { visit(e.Target, e.QName()); return true })
	v.Variables().Range(func(e *emit.Variable) bool { visit(e.Target, e.QName()); return true })
	v.Constants().Range(func(e *emit.Constant) bool { visit(e.Target, e.QName()); return true })
	v.Enums().Range(func(e *emit.Enum) bool { visit(e.Target, e.QName()); return true })
	v.Aliases().Range(func(e *emit.Alias) bool { visit(e.File, e.QName()); return true })

	for _, g := range groups {
		if len(g.pkgs) < 2 {
			continue
		}
		ps.Errorf(position.Pos{},
			"one-file-one-package violation at %s: conflicting package declarations %s",
			g.key, formatPackageConflict(g.pkgs))
		clearConflictedTargets(v, g.key)
	}
}

// formatPackageConflict renders the conflicting-package set in a
// deterministic, parser-friendly form for the diagnostic message:
// `pkgA={qname1,qname2}; pkgB={qname3}`. Package names are sorted
// lexicographically so the diagnostic stays byte-stable across
// runs even though the visit order through the buckets is not
// guaranteed to be deterministic. The qnames within each entry
// are left in insertion order since emit-view ranges yield decls
// in insertion order already.
func formatPackageConflict(pkgs map[string][]string) string {
	type entry struct {
		pkg    string
		qnames []string
	}
	out := make([]entry, 0, len(pkgs))
	for pkg, qnames := range pkgs {
		out = append(out, entry{pkg: pkg, qnames: qnames})
	}
	slices.SortFunc(out, func(a, b entry) int { return cmp.Compare(a.pkg, b.pkg) })
	parts := make([]string, 0, len(out))
	for _, e := range out {
		parts = append(parts, fmt.Sprintf("%s={%s}", e.pkg, strings.Join(e.qnames, ",")))
	}
	return strings.Join(parts, "; ")
}

// clearConflictedTargets zeroes every routable decl whose
// (Dir, Filename) joins to key. The cleared decls drop from the
// byTarget rebuild and from the manifest; they remain in their
// per-kind buckets so debugging tooling can still inspect them.
func clearConflictedTargets(v *store.EmitView, key string) {
	v.Structs().Range(func(e *emit.Struct) bool {
		if e.Target.JoinPath() == key {
			e.Target = emit.Target{}
		}
		return true
	})
	v.Interfaces().Range(func(e *emit.Interface) bool {
		if e.Target.JoinPath() == key {
			e.Target = emit.Target{}
		}
		return true
	})
	v.Functions().Range(func(e *emit.Function) bool {
		if e.Target.JoinPath() == key {
			e.Target = emit.Target{}
		}
		return true
	})
	v.Variables().Range(func(e *emit.Variable) bool {
		if e.Target.JoinPath() == key {
			e.Target = emit.Target{}
		}
		return true
	})
	v.Constants().Range(func(e *emit.Constant) bool {
		if e.Target.JoinPath() == key {
			e.Target = emit.Target{}
		}
		return true
	})
	v.Enums().Range(func(e *emit.Enum) bool {
		if e.Target.JoinPath() == key {
			e.Target = emit.Target{}
		}
		return true
	})
	v.Aliases().Range(func(e *emit.Alias) bool {
		if e.File.JoinPath() == key {
			e.File = emit.Target{}
		}
		return true
	})
}

// collectFilenameSuffixes builds the plugin-name → filename-suffix
// lookup the Layout phase consults to compose Target.Filename. The
// map contains one entry per registered generator that implements
// [plugin.FilenameProvider] AND returns a non-empty suffix —
// generators that don't implement the capability OR that return
// the empty string are both absent from the map. The two cases
// are equivalent at the routing layer: both signal "I don't
// declare a routable suffix", and either kind of attribution
// failure surfaces [ErrMissingFilenameProvider] when a decl
// emitted by such a plugin reaches the Layout phase.
//
// Closing the empty-suffix case at the collection boundary
// (rather than tolerating it downstream) prevents the
// source-overwrite footgun the spec calls out: a routable decl
// composed with an empty suffix would produce Target.Filename =
// <source-basename>, which on a fresh write would clobber the
// originating source file on disk.
func (p *Pipeline) collectFilenameSuffixes() map[string]string {
	out := map[string]string{}
	for _, gen := range p.generators {
		fp, ok := any(gen).(plugin.FilenameProvider)
		if !ok {
			continue
		}
		suffix := fp.FilenameSuffix()
		if suffix == "" {
			continue
		}
		out[gen.Name()] = suffix
	}
	return out
}

// originSourcePackage walks origin's owning-package chain and
// returns the matching [*node.Package] from the node store. The
// lookup is a direct ByQName so scope predicates do not filter it
// out. Returns nil when origin has no resolvable package (a
// synthetic Origin built without a frontend, or a Node kind whose
// Owner chain never reaches a packaged ancestor).
//
// The helper is the single place that knows how to extract a
// packaged ancestor from a [node.Node] of any kind. Both decl
// routing and pending-slot-tuple routing call into it so the two
// paths cannot drift.
func originSourcePackage(s *store.Store, origin node.Node) *node.Package {
	path := originPackagePath(origin)
	if path == "" {
		return nil
	}
	pkg, _ := s.Nodes().Packages().ByQName(path)
	return pkg
}

// originPackagePath returns the import path of origin's owning
// source package. Top-level kinds (Struct, Interface, Function,
// Variable, Constant, Enum, Alias) read their Package field
// directly. File terminates at its Owner Package. EnumVariant
// terminates at its Owner Enum. Method and Field walk their Owner
// chain — typically one hop to the enclosing Struct or Interface
// — to reach a packaged ancestor. Returns the empty string for
// kinds (Param, TypeParam, Import, Package itself) whose
// Origin-as-routable use case has no documented routing semantics;
// adding an arm here requires the same kind to surface through
// [store.Reader] and at least one plugin's emit graph.
func originPackagePath(n node.Node) string {
	cur := n
	for cur != nil {
		switch v := cur.(type) {
		case *node.Struct:
			return v.Package
		case *node.Interface:
			return v.Package
		case *node.Function:
			return v.Package
		case *node.Variable:
			return v.Package
		case *node.Constant:
			return v.Package
		case *node.Enum:
			return v.Package
		case *node.Alias:
			return v.Package
		case *node.File:
			if v.Owner != nil {
				return v.Owner.Path
			}
			return ""
		case *node.Method:
			cur = v.Owner
			continue
		case *node.Field:
			cur = v.Owner
			continue
		case *node.EnumVariant:
			if v.Owner == nil {
				return ""
			}
			return v.Owner.Package
		}
		return ""
	}
	return ""
}

// originSourceDirBasename returns the directory and basename
// derived from origin's source position. The basename strips the
// `.go` extension when present so the Layout phase can append the
// plugin's filename suffix verbatim. Returns empty strings when the
// origin's source position carries no file — synthetic origins fall
// back to that case and the resulting Target needs an alternative
// filename source (CLI -o, +gen:out) to be routable.
func originSourceDirBasename(origin node.Node) (dir, basename string) {
	pos := origin.Pos()
	if pos.File == "" {
		return "", ""
	}
	base := filepath.Base(pos.File)
	if ext := filepath.Ext(base); ext != "" {
		base = base[:len(base)-len(ext)]
	}
	d := filepath.Dir(pos.File)
	if d == "." {
		d = ""
	}
	return d, base
}

// outDirectiveFilename returns the first positional argument of the
// `+gen:out` directive on n, or empty + false when n carries no
// such directive. The Layout phase calls this for both decl
// routing and slot-tuple routing so the directive's effect applies
// uniformly to whichever path routes the host origin.
func outDirectiveFilename(n node.Node) (string, bool) {
	for _, d := range n.Directives() {
		if d.Name != OutDirective {
			continue
		}
		if len(d.Args) > 0 {
			return d.Args[0], true
		}
	}
	return "", false
}
