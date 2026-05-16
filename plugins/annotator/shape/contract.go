// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package shape

import (
	"slices"

	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/core/meta"
	"go.thesmos.sh/eidos/node"
)

// Contract is one named multi-callable protocol with a fixed role
// vocabulary. Per-protocol sub-packages (`persister`, `tx`,
// `saga`, `pool`, …) export one of these from their public API;
// the consumer composes them into the umbrella plugin via
// [Plugin.Contracts].
//
// Contract is orthogonal to [Detector]: a callable can carry both
// a structural shape (from a detector) and one or more contract
// memberships (from `+gen:contract` directives) without either
// overwriting the other.
type Contract struct {
	// Name is the contract's stable identifier (e.g. `"tx"`,
	// `"persister"`, `"saga"`). Used as the contract-name argument
	// to the `+gen:contract` directive and as a path component in
	// every per-contract meta key.
	Name string

	// Roles enumerates the role vocabulary the contract recognises
	// (e.g. `["begin", "commit", "rollback"]` for tx;
	// `["writer", "reader"]` for persister). The refinement
	// resolver rejects directives that name an undeclared role.
	Roles []string

	// Required maps a role to the partner roles that must be
	// specified when this role's directive appears. The refinement
	// resolver emits a positioned diagnostic when any required
	// partner is missing from the directive.
	//
	// Leave empty when no per-role partner-presence requirements
	// apply (e.g. for the simple two-member persister contract
	// where the structural-shape detector already enforces
	// signature shape on each side).
	Required map[string][]string

	// Validate, when non-nil, runs in the validation-priority
	// annotator pass after sibling resolution completes. Receives
	// the per-contract-instance member set keyed by role and
	// returns any invariants the implementation violated. Use for
	// structural checks like "Pool has exactly one Get and one
	// Put" or "every Saga step has a compensation".
	Validate ContractValidator
}

// ContractValidator is the signature of the optional per-contract
// invariant check the [Validator] annotator runs after sibling
// resolution completes. Returns the list of violations found
// (empty / nil on success); the validator annotator attaches each
// violation to its host node's diagnostic sink.
type ContractValidator func(members map[string][]node.Node) []ContractViolation

// ContractViolation is one invariant breach reported by a
// [ContractValidator]. The [Validator] annotator surfaces it as
// a positioned diagnostic against the host node.
type ContractViolation struct {
	// Host is the node the diagnostic attaches to. Pick the
	// member that most directly demonstrates the failure (e.g.
	// the orphan Get when "no matching Put" is violated).
	Host node.Node

	// Message is the human-readable violation summary.
	Message string
}

// MetaContracts is the per-callable list of contracts the callable
// participates in. Populated by [Plugin.applyContracts] each time a
// non-negated `+gen:contract` directive is recognised; consumers
// iterate this list to discover every contract the callable is
// part of, then read the per-contract role + partner keys.
//
//nolint:gochecknoglobals // registry-singleton key
var MetaContracts = meta.EnsureKey("shape.contracts", meta.StringListParser)

// ContractRoleKey returns the typed meta key carrying the
// callable's role within the named contract — stamped at
// `shape.contract.<name>.role`. Constructed on demand via
// [meta.EnsureKey] so multiple per-contract sub-packages
// referencing the same name resolve to one canonical key.
func ContractRoleKey(name string) meta.Key[string] {
	return meta.EnsureKey("shape.contract."+name+".role", meta.StringParser)
}

// ContractPartnerKey returns the typed meta key carrying the
// partner callable filling the named role within the contract —
// stamped at `shape.contract.<contract>.partner.<role>`. The
// stamped value is a raw sibling name as the umbrella plugin
// records it and a qualified name after the refinement resolver
// rewrites it.
func ContractPartnerKey(contract, role string) meta.Key[string] {
	return meta.EnsureKey(
		"shape.contract."+contract+".partner."+role,
		meta.StringParser,
	)
}

// contractStampedBy is the setBy attribution used for every
// contract-related stamp this plugin writes. Distinct from
// [PluginName] so meta provenance distinguishes structural-shape
// stamps from contract-membership stamps.
const contractStampedBy = PluginName + ".contract"

// applyContracts stamps every non-negated `+gen:contract` directive
// on bag. Unknown contract names are silently skipped — the
// refinement resolver surfaces them as positioned
// diagnostics. Unknown KV keys besides reserved ones are stamped
// verbatim as partner refs so the resolver has the raw data
// needed to diagnose them.
//
// The function is permissive because the framework's directive
// validator handles schema-level enforcement (missing `role=`,
// malformed positional, etc.) at Build time; this pass concerns
// itself only with meta stamping for callables whose directives
// already passed parse-time validation.
func (p *Plugin) applyContracts(bag *meta.Bag, dirs []*directive.Directive) {
	for _, d := range dirs {
		if d == nil || d.Name != ContractDirectiveName || d.Negated {
			continue
		}
		name := contractNameFromDirective(d)
		if name == "" {
			continue
		}
		if _, registered := p.contracts[name]; !registered {
			continue
		}
		role := d.KV["role"]
		if role == "" {
			continue
		}
		ContractRoleKey(name).Set(bag, role, contractStampedBy)
		for k, v := range d.KV {
			if k == "role" || v == "" {
				continue
			}
			ContractPartnerKey(name, k).Set(bag, v, contractStampedBy)
		}
		appendContract(bag, name)
	}
}

// contractNameFromDirective returns the contract name declared by
// d — the first positional argument. Returns empty when no
// positional was supplied (the framework validator rejects this
// at parse time; the runtime guard is defence-in-depth).
func contractNameFromDirective(d *directive.Directive) string {
	if len(d.Args) > 0 {
		return d.Args[0]
	}
	return ""
}

// appendContract adds name to the [MetaContracts] list on bag,
// preserving insertion order and skipping duplicates. Idempotent:
// repeated calls with the same name leave the list unchanged.
func appendContract(bag *meta.Bag, name string) {
	current, _ := MetaContracts.Get(bag)
	if slices.Contains(current, name) {
		return
	}
	MetaContracts.Set(bag, append(current, name), contractStampedBy)
}

// Contracts returns the contracts the callable participates in,
// in insertion order. Returns empty when the callable carries no
// contract memberships.
//
// Consumers wanting per-contract role + partner data combine the
// list with [ContractRoleKey] and [ContractPartnerKey]:
//
//	for _, name := range shape.Contracts(m.Meta()) {
//	    role, _ := shape.ContractRoleKey(name).Get(m.Meta())
//	    // …
//	}
func Contracts(bag *meta.Bag) []string {
	if bag == nil {
		return nil
	}
	out, _ := MetaContracts.Get(bag)
	return out
}
