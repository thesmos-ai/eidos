// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package pool_test

import (
	"reflect"
	"testing"

	"go.thesmos.sh/eidos/core/diag"
	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/plugins/annotator/shape"
	"go.thesmos.sh/eidos/plugins/annotator/shape/contracts/internal/contracttest"
	"go.thesmos.sh/eidos/plugins/annotator/shape/contracts/pool"
)

func TestContract_Identity(t *testing.T) {
	t.Parallel()
	contracttest.AssertIdentity(t,
		pool.Contract(),
		pool.Name, pool.Roles)
}

func TestContract_DeclaresRequiredAndValidate(t *testing.T) {
	t.Parallel()
	c := pool.Contract()
	wantRequired := map[string][]string{"get": {"put"}}
	if !reflect.DeepEqual(c.Required, wantRequired) {
		t.Fatalf("Required = %v, want %v", c.Required, wantRequired)
	}
	if c.Validate == nil {
		t.Fatalf("Validate hook missing")
	}
}

func TestContract_ValidateAcceptsExactlyOneEach(t *testing.T) {
	t.Parallel()
	c := pool.Contract()
	members := map[string][]shape.ContractMember{
		"get": {{Host: &node.Function{Name: "Get"}}},
		"put": {{Host: &node.Function{Name: "Put"}}},
	}
	if got := c.Validate(members); len(got) != 0 {
		t.Fatalf("Validate(one-each) = %+v; want no violations", got)
	}
}

// TestContract_PipelineRoundTrip exercises the happy path of one
// Get + one Put through umbrella → resolver → validator.
func TestContract_PipelineRoundTrip(t *testing.T) {
	t.Parallel()
	get := &node.Function{
		Name: "Get", Package: "x",
		BaseNode: node.BaseNode{
			DirectiveList: []*directive.Directive{
				contracttest.HostDirective(pool.Name, "get", map[string]string{
					"put": "Put",
				}),
			},
		},
	}
	put := &node.Function{Name: "Put", Package: "x"}
	pkg := &node.Package{
		Name: "x", Path: "x",
		Functions: []*node.Function{get, put},
	}
	diags := contracttest.RunPipeline(t, pool.Contract(), pkg)
	contracttest.AssertNoErrorDiag(t, diags)

	contracttest.AssertRole(t, get.Meta(), pool.Name, "get")
	contracttest.AssertPartner(t, get.Meta(), pool.Name, "put", "x.Put")
	contracttest.AssertRole(t, put.Meta(), pool.Name, "put")
	contracttest.AssertPartner(t, put.Meta(), pool.Name, "get", "x.Get")
}

// TestContract_ValidatorFlagsDuplicateGet exercises the Validate
// hook through the actual [shape.Validator] annotator — two Get
// callables share one Put, and the validator must emit a
// diagnostic naming the duplicate.
func TestContract_ValidatorFlagsDuplicateGet(t *testing.T) {
	t.Parallel()
	getA := &node.Function{
		Name: "GetA", Package: "x",
		BaseNode: node.BaseNode{
			DirectiveList: []*directive.Directive{
				contracttest.HostDirective(pool.Name, "get", map[string]string{
					"put": "Put",
				}),
			},
		},
	}
	getB := &node.Function{
		Name: "GetB", Package: "x",
		BaseNode: node.BaseNode{
			DirectiveList: []*directive.Directive{
				contracttest.HostDirective(pool.Name, "get", map[string]string{
					"put": "Put",
				}),
			},
		},
	}
	put := &node.Function{Name: "Put", Package: "x"}
	pkg := &node.Package{
		Name: "x", Path: "x",
		Functions: []*node.Function{getA, getB, put},
	}
	diags := contracttest.RunPipeline(t, pool.Contract(), pkg)
	contracttest.AssertContainsDiag(t, diags, diag.Error, "exactly one get")
}
