// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package shape_test

import (
	"reflect"
	"testing"

	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/core/meta"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/plugins/annotator/shape"
)

// txContract is the canonical multi-role test contract used by
// most stamping tests below — three roles, no per-role partner
// requirements (validation is the resolver's job, not Phase 1).
func txContract() shape.Contract {
	return shape.Contract{
		Name:  "tx",
		Roles: []string{"begin", "commit", "rollback"},
	}
}

// outboxContract is the secondary test contract used to assert
// that a single callable can carry multiple memberships without
// interference between them.
func outboxContract() shape.Contract {
	return shape.Contract{
		Name:  "outbox",
		Roles: []string{"append", "subscribe"},
	}
}

// TestContract_DirectiveStamping covers Phase 1 of the contract
// pipeline: each non-negated `+gen:contract` directive on a
// callable stamps the role, partner refs, and contract list
// without interfering with structural shape stamps.
func TestContract_DirectiveStamping(t *testing.T) {
	t.Parallel()

	t.Run("stamps role and partner refs for a registered contract", func(t *testing.T) {
		t.Parallel()
		fn := contractFn("Begin",
			&directive.Directive{
				Name: shape.ContractDirectiveName,
				Args: []string{"tx"},
				KV: map[string]string{
					"role":     "begin",
					"commit":   "Commit",
					"rollback": "Rollback",
				},
			},
		)
		runAnnotate(t, shape.New().Contracts(txContract()), pkgWithFunction(fn))

		assertContracts(t, fn.Meta(), []string{"tx"})
		assertMeta(t, fn.Meta(), shape.ContractRoleKey("tx"), "begin")
		assertMeta(t, fn.Meta(), shape.ContractPartnerKey("tx", "commit"), "Commit")
		assertMeta(t, fn.Meta(), shape.ContractPartnerKey("tx", "rollback"), "Rollback")
	})

	t.Run("multiple contracts on one callable: both stamped, ordered, no interference", func(t *testing.T) {
		t.Parallel()
		fn := contractFn("Mixed",
			&directive.Directive{
				Name: shape.ContractDirectiveName,
				Args: []string{"tx"},
				KV:   map[string]string{"role": "begin", "commit": "C"},
			},
			&directive.Directive{
				Name: shape.ContractDirectiveName,
				Args: []string{"outbox"},
				KV:   map[string]string{"role": "append", "subscribe": "Sub"},
			},
		)
		runAnnotate(t,
			shape.New().Contracts(txContract(), outboxContract()),
			pkgWithFunction(fn),
		)

		assertContracts(t, fn.Meta(), []string{"tx", "outbox"})
		assertMeta(t, fn.Meta(), shape.ContractRoleKey("tx"), "begin")
		assertMeta(t, fn.Meta(), shape.ContractRoleKey("outbox"), "append")
		assertMeta(t, fn.Meta(), shape.ContractPartnerKey("tx", "commit"), "C")
		assertMeta(t, fn.Meta(), shape.ContractPartnerKey("outbox", "subscribe"), "Sub")
	})

	t.Run("contract membership stamps even when structural shape detector also matches", func(t *testing.T) {
		t.Parallel()
		// readerFunc carries a reader signature so the reader
		// detector also fires; both stamps must land.
		fn := readerFunc("Find")
		fn.DirectiveList = []*directive.Directive{
			{
				Name: shape.ContractDirectiveName,
				Args: []string{"tx"},
				KV:   map[string]string{"role": "begin"},
			},
		}
		runAnnotate(t,
			shape.New().
				Detectors(testReaderDetector()).
				Contracts(txContract()),
			pkgWithFunction(fn),
		)
		// Structural shape stamped by detector
		assertShape(t, fn.Meta(), "reader")
		// Contract membership stamped alongside
		assertContracts(t, fn.Meta(), []string{"tx"})
		assertMeta(t, fn.Meta(), shape.ContractRoleKey("tx"), "begin")
	})

	t.Run("unknown contract name is silently skipped (resolver diagnoses later)", func(t *testing.T) {
		t.Parallel()
		fn := contractFn("X",
			&directive.Directive{
				Name: shape.ContractDirectiveName,
				Args: []string{"never-registered"},
				KV:   map[string]string{"role": "any"},
			},
		)
		// Plugin has no contracts registered — directive is a no-op.
		runAnnotate(t, shape.New(), pkgWithFunction(fn))
		if got := shape.Contracts(fn.Meta()); len(got) != 0 {
			t.Fatalf("expected no contract stamps; got %v", got)
		}
	})

	t.Run("missing role= is silently skipped at Phase 1", func(t *testing.T) {
		t.Parallel()
		fn := contractFn("Begin",
			&directive.Directive{
				Name: shape.ContractDirectiveName,
				Args: []string{"tx"},
				KV:   map[string]string{"commit": "Commit"},
			},
		)
		runAnnotate(t, shape.New().Contracts(txContract()), pkgWithFunction(fn))
		if got := shape.Contracts(fn.Meta()); len(got) != 0 {
			t.Fatalf("expected no contract stamps with missing role=; got %v", got)
		}
	})

	t.Run("negated directive is ignored", func(t *testing.T) {
		t.Parallel()
		fn := contractFn("Begin",
			&directive.Directive{
				Name:    shape.ContractDirectiveName,
				Args:    []string{"tx"},
				KV:      map[string]string{"role": "begin"},
				Negated: true,
			},
		)
		runAnnotate(t, shape.New().Contracts(txContract()), pkgWithFunction(fn))
		if got := shape.Contracts(fn.Meta()); len(got) != 0 {
			t.Fatalf("expected negated directive to be ignored; got %v", got)
		}
	})

	t.Run("partner refs with empty value are skipped", func(t *testing.T) {
		t.Parallel()
		fn := contractFn("Begin",
			&directive.Directive{
				Name: shape.ContractDirectiveName,
				Args: []string{"tx"},
				KV: map[string]string{
					"role":     "begin",
					"commit":   "Commit",
					"rollback": "",
				},
			},
		)
		runAnnotate(t, shape.New().Contracts(txContract()), pkgWithFunction(fn))

		assertMeta(t, fn.Meta(), shape.ContractPartnerKey("tx", "commit"), "Commit")
		if _, ok := shape.ContractPartnerKey("tx", "rollback").Get(fn.Meta()); ok {
			t.Fatalf("expected empty partner ref to be unstamped")
		}
	})

	t.Run("duplicate contract directive does not duplicate the contracts list entry", func(t *testing.T) {
		t.Parallel()
		fn := contractFn("Begin",
			&directive.Directive{
				Name: shape.ContractDirectiveName,
				Args: []string{"tx"},
				KV:   map[string]string{"role": "begin"},
			},
			&directive.Directive{
				Name: shape.ContractDirectiveName,
				Args: []string{"tx"},
				KV:   map[string]string{"role": "begin"},
			},
		)
		runAnnotate(t, shape.New().Contracts(txContract()), pkgWithFunction(fn))
		assertContracts(t, fn.Meta(), []string{"tx"})
	})

	t.Run("method-bound contracts stamp the same as free functions", func(t *testing.T) {
		t.Parallel()
		m := &node.Method{
			Name: "Begin",
			BaseNode: node.BaseNode{
				DirectiveList: []*directive.Directive{
					{
						Name: shape.ContractDirectiveName,
						Args: []string{"tx"},
						KV:   map[string]string{"role": "begin", "commit": "Commit"},
					},
				},
			},
		}
		s := &node.Struct{Name: "Repo", Package: "x", Methods: []*node.Method{m}}
		runAnnotate(t, shape.New().Contracts(txContract()), pkgWithStruct(s))

		assertContracts(t, m.Meta(), []string{"tx"})
		assertMeta(t, m.Meta(), shape.ContractRoleKey("tx"), "begin")
		assertMeta(t, m.Meta(), shape.ContractPartnerKey("tx", "commit"), "Commit")
	})

	t.Run("Contracts helper returns nil for an unstamped bag", func(t *testing.T) {
		t.Parallel()
		if got := shape.Contracts(nil); got != nil {
			t.Fatalf("Contracts(nil) = %v, want nil", got)
		}
		if got := shape.Contracts(meta.NewBag()); got != nil {
			t.Fatalf("Contracts(empty) = %v, want nil", got)
		}
	})
}

// contractFn returns a free-function node carrying the supplied
// directives — used by every test that exercises directive-driven
// contract stamping.
func contractFn(name string, dirs ...*directive.Directive) *node.Function {
	return &node.Function{
		Name: name, Package: "x",
		BaseNode: node.BaseNode{DirectiveList: dirs},
	}
}

// assertContracts fails the test when the contract list stamped
// on bag does not deep-equal want.
func assertContracts(t *testing.T, bag *meta.Bag, want []string) {
	t.Helper()
	got := shape.Contracts(bag)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Contracts = %v, want %v", got, want)
	}
}
