// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package shape

import (
	"slices"

	"go.thesmos.sh/eidos/core/diag"
	"go.thesmos.sh/eidos/core/meta"
	"go.thesmos.sh/eidos/core/position"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/sdk"
)

// ValidatorName is the stable identifier the framework uses for
// the contract-validation annotator that runs after the
// refinement resolver.
const ValidatorName = "shape.contract.validator"

// Validator is the validation-bucket companion to the umbrella
// plugin and the [Resolver]. It runs at
// [sdk.AnnotatorValidation] priority — strictly after the
// resolver — and enforces two kinds of contract invariants:
//
//   - **Required partners.** [Contract.Required] declares the
//     partner roles each self-role must supply; the validator
//     emits a positioned diagnostic for any missing partner key.
//   - **Custom invariants.** [Contract.Validate], when non-nil,
//     receives the resolved member set keyed by role and returns
//     [ContractViolation] entries the validator surfaces as
//     positioned diagnostics against the offending member.
//
// Construct via [Plugin.Validator] so the validator shares the
// umbrella plugin's contract registrations.
type Validator struct {
	contracts map[string]Contract

	// members accumulates the (callable, role, contract) triples
	// observed during the walk. Used by [Validator.AfterNodes] to
	// group callables into per-contract member sets before
	// invoking each [Contract.Validate] hook. Reset every
	// Annotate call.
	members map[string]map[string][]node.Node
}

// Validator returns a fresh [Validator] sharing p's contract
// registrations. Register alongside the umbrella plugin and the
// resolver so the three run in priority order:
//
//	s := shape.New().Detectors(...).Contracts(...)
//	pipe.Use(s)
//	pipe.Use(s.Resolver())
//	pipe.Use(s.Validator())
func (p *Plugin) Validator() *Validator {
	return &Validator{contracts: p.contracts}
}

// Name returns [ValidatorName].
func (*Validator) Name() string { return ValidatorName }

// Priority places the validator in the annotator-validation
// bucket so it runs strictly after the resolver.
func (*Validator) Priority() sdk.Priority { return sdk.AnnotatorValidation }

// Annotate delegates to the framework's annotator walk via
// [sdk.Walk]; per-callable required-partner checks live in
// [Validator.OnMethod] and [Validator.OnFunction]; per-contract
// member-set validation runs once in [Validator.AfterNodes].
func (v *Validator) Annotate(ctx *sdk.AnnotatorContext) error {
	return sdk.Walk(ctx, v)
}

// BeforeNodes resets the per-Annotate member accumulator so the
// validator stays stateless across runs.
func (v *Validator) BeforeNodes(*sdk.AnnotatorContext) {
	v.members = make(map[string]map[string][]node.Node)
}

// OnMethod runs the required-partner check on m for every
// contract it participates in, and accumulates m into the
// member set keyed by (contract, role).
func (v *Validator) OnMethod(ctx *sdk.AnnotatorContext, m *node.Method) {
	v.visit(ctx, m, m.Meta())
}

// OnFunction runs the required-partner check on fn for every
// contract it participates in, and accumulates fn into the
// member set keyed by (contract, role).
func (v *Validator) OnFunction(ctx *sdk.AnnotatorContext, fn *node.Function) {
	v.visit(ctx, fn, fn.Meta())
}

// AfterNodes invokes each registered [Contract.Validate] hook
// against its accumulated member set and surfaces any returned
// [ContractViolation] entries as positioned diagnostics on
// ctx.Diag.
func (v *Validator) AfterNodes(ctx *sdk.AnnotatorContext) {
	sink := ctx.Diag.For(ValidatorName)
	for contractName, members := range v.members {
		spec, ok := v.contracts[contractName]
		if !ok || spec.Validate == nil {
			continue
		}
		for _, violation := range spec.Validate(members) {
			sink.Errorf(posOf(violation.Host),
				"shape.contract %q: %s", contractName, violation.Message)
		}
	}
}

// visit runs the per-callable required-partner check and
// accumulates host into the member set.
func (v *Validator) visit(ctx *sdk.AnnotatorContext, host node.Node, bag *meta.Bag) {
	memberships := Contracts(bag)
	if len(memberships) == 0 {
		return
	}
	sink := ctx.Diag.For(ValidatorName)
	for _, contractName := range memberships {
		spec, ok := v.contracts[contractName]
		if !ok {
			continue
		}
		role, _ := ContractRoleKey(spec.Name).Get(bag)
		v.checkRequired(host, bag, role, spec, sink)
		v.accumulate(contractName, role, host)
	}
}

// checkRequired emits a diagnostic for each partner role
// declared in spec.Required[role] that is missing a stamped
// partner key on bag.
func (*Validator) checkRequired(
	host node.Node,
	bag *meta.Bag,
	role string,
	spec Contract,
	sink *diag.PluginSink,
) {
	required, ok := spec.Required[role]
	if !ok {
		return
	}
	for _, partnerRole := range required {
		got, _ := ContractPartnerKey(spec.Name, partnerRole).Get(bag)
		if got != "" {
			continue
		}
		sink.Errorf(host.Pos(),
			"shape.contract %q: role %q requires partner %q, none stamped",
			spec.Name, role, partnerRole)
	}
}

// accumulate records host as a member of (contractName, role) in
// the per-contract member set. Roles are deduplicated by host
// pointer so the same callable joining a contract twice (via
// self-stamp + back-stamp) appears once per role.
func (v *Validator) accumulate(contractName, role string, host node.Node) {
	byRole, ok := v.members[contractName]
	if !ok {
		byRole = make(map[string][]node.Node)
		v.members[contractName] = byRole
	}
	if slices.Contains(byRole[role], host) {
		return
	}
	byRole[role] = append(byRole[role], host)
}

// posOf returns n's source position via the [node.Node.Pos]
// method. Defensive guard: nil hosts produce a zero position.
func posOf(n node.Node) position.Pos {
	if n == nil {
		return position.Pos{}
	}
	return n.Pos()
}
