// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package pool_test

import (
	"reflect"
	"testing"

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

func TestContract_ValidateFlagsDuplicateGet(t *testing.T) {
	t.Parallel()
	c := pool.Contract()
	getA := &node.Function{Name: "GetA", Package: "x"}
	getB := &node.Function{Name: "GetB", Package: "x"}
	put := &node.Function{Name: "Put", Package: "x"}
	members := map[string][]node.Node{
		"get": {getA, getB},
		"put": {put},
	}
	violations := c.Validate(members)
	if len(violations) != 1 || violations[0].Host != getB {
		t.Fatalf("Validate(duplicate-get) = %+v; want one violation on getB", violations)
	}
}

func TestContract_ValidateAcceptsExactlyOneEach(t *testing.T) {
	t.Parallel()
	c := pool.Contract()
	members := map[string][]node.Node{
		"get": {&node.Function{Name: "Get"}},
		"put": {&node.Function{Name: "Put"}},
	}
	if got := c.Validate(members); len(got) != 0 {
		t.Fatalf("Validate(one-each) = %+v; want no violations", got)
	}
}

// _ ensures the package-level Contract value satisfies
// [shape.Contract]'s expected shape at compile time.
var _ shape.Contract = pool.Contract()
