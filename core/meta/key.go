// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package meta

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"go.thesmos.sh/eidos/core/position"
)

// AnyKey is the type-erased view of a [Key]. The directive-override
// step uses it to stamp values into a Bag without knowing the static
// type the key carries — it parses the directive's raw string via the
// key's registered Parser, then writes a typed value into the
// underlying storage.
//
// Plugin authors never implement AnyKey directly; the generic
// [NewKey] constructor produces a [Key][T] whose method set satisfies
// the interface.
type AnyKey interface {
	Name() string
	SetDirectiveFromString(b *Bag, raw string, pos position.Pos) error
	TombstoneDirective(b *Bag, pos position.Pos)
}

// Key is a typed metadata key. Construct via [NewKey] (which also
// registers the key globally so the directive-override step can find
// it by name).
//
// Keys are value types and safe to share across goroutines. The
// underlying storage is the per-owner [Bag] supplied to each method.
type Key[T any] struct {
	name   string
	parser Parser[T]
}

// ErrDuplicateKey is returned by [NewKey] when a key with the same
// name has already been registered.
//
// The duplicate-registration check is strict: even an identical Parser
// is rejected, because callers should declare each key exactly once
// in a package-level var. Re-registration usually indicates two
// packages claiming the same name, which would silently shadow each
// other at runtime.
var ErrDuplicateKey = errors.New("meta: duplicate key registration")

// ErrUnregisteredKey is returned by [Lookup] when no key matches the
// requested name. The directive-override step uses it to skip
// unknown +gen:meta directives with a positioned diagnostic rather
// than crashing the run.
var ErrUnregisteredKey = errors.New("meta: key not registered")

// keyRegistry holds every Key declared in the running process, keyed
// by name. Access is mutexed; registration is once per Key and reads
// dominate in the steady state.
var keyRegistry = struct {
	mu   sync.RWMutex
	keys map[string]AnyKey
}{keys: map[string]AnyKey{}}

// NewKey declares a typed key with the given name and parser, and
// registers it globally so [Lookup] can find it.
//
// Calling NewKey twice with the same name is a programmer error: it
// panics with [ErrDuplicateKey] wrapped with the offending name. Use
// package-level `var` declarations so duplicate registration shows up
// as a build/init failure rather than a runtime data race.
//
// Plugin authors declare keys once per package:
//
//	var Detected = meta.NewKey[bool]("shape.writer", meta.BoolParser)
func NewKey[T any](name string, parser Parser[T]) Key[T] {
	k := Key[T]{name: name, parser: parser}
	register(k)
	return k
}

// register adds k to the global registry. Duplicate registration is
// a programmer error (two distinct package-level vars claiming the
// same name); the function panics with [ErrDuplicateKey] so the
// problem surfaces at init rather than as silent value shadowing
// later. Tests cover the panic path via recover; production code
// never observes it.
func register(k AnyKey) {
	keyRegistry.mu.Lock()
	defer keyRegistry.mu.Unlock()
	if _, exists := keyRegistry.keys[k.Name()]; exists {
		// Init-time programmer error; documented contract.
		err := fmt.Errorf("%w: %q", ErrDuplicateKey, k.Name())
		panic(err) //nolint:forbidigo
	}
	keyRegistry.keys[k.Name()] = k
}

// Lookup returns the registered AnyKey with the given name, or
// [ErrUnregisteredKey] wrapped with the offending name when no such
// key exists.
func Lookup(name string) (AnyKey, error) {
	keyRegistry.mu.RLock()
	defer keyRegistry.mu.RUnlock()
	k, ok := keyRegistry.keys[name]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrUnregisteredKey, name)
	}
	return k, nil
}

// Name returns the key's registered name.
func (k Key[T]) Name() string { return k.name }

// Get returns the typed value of k in b, or the zero value of T and
// false when the resolution is "not set" (no entry or a winning
// tombstone). Get reflects authority precedence and prefix-tombstone
// rules — see [Bag.Winning] for the full algorithm.
//
// Type safety is enforced upstream: every name is bound to a single
// Key[T] by the registry, so the stored value's concrete type is
// always T for the registered key. The one exception is a value
// freshly unmarshalled by [Bag.UnmarshalJSON] — Bag preserves the
// raw JSON bytes there because it lacks static type information at
// that level. Get detects the raw form and decodes lazily into T,
// then returns the typed result.
//
// A raw-JSON decode failure returns the zero T and false; callers
// that need to distinguish "absent" from "corrupt" should inspect
// [Bag.Winning] directly.
func (k Key[T]) Get(b *Bag) (T, bool) {
	raw, ok := b.RawValue(k.name)
	if !ok {
		var zero T
		return zero, false
	}
	if rm, isRaw := raw.(json.RawMessage); isRaw {
		var v T
		if err := json.Unmarshal(rm, &v); err != nil {
			var zero T
			return zero, false
		}
		return v, true
	}
	return raw.(T), true
}

// Has reports whether k currently resolves to a value in b. Equivalent
// to discarding the value from [Key.Get].
func (k Key[T]) Has(b *Bag) bool {
	_, ok := k.Get(b)
	return ok
}

// Set records v at AuthorityPlugin attributed to setBy (typically the
// plugin's name).
func (k Key[T]) Set(b *Bag, v T, setBy string) {
	b.setEntry(k.name, v, AuthorityPlugin, setBy, position.Pos{})
}

// SetAt records v with the given authority attributed to setBy. Pos
// records where the operation originated (the directive's position
// for AuthorityDirective; usually zero for the others).
//
// Use this rather than the convenience wrappers when the authority
// is decided dynamically — e.g. inside the test harness.
func (k Key[T]) SetAt(b *Bag, v T, auth Authority, setBy string, pos position.Pos) {
	b.setEntry(k.name, v, auth, setBy, pos)
}

// SetDirective records v at AuthorityDirective, used by the built-in
// directive-override step.
func (k Key[T]) SetDirective(b *Bag, v T, pos position.Pos) {
	b.setEntry(k.name, v, AuthorityDirective, "directive", pos)
}

// SetManual records v at AuthorityManual, reserved for test harnesses
// and programmatic overrides. The setBy label distinguishes the
// caller in --explain output.
func (k Key[T]) SetManual(b *Bag, v T, setBy string) {
	b.setEntry(k.name, v, AuthorityManual, setBy, position.Pos{})
}

// Tombstone records a tombstone at AuthorityPlugin attributed to
// setBy. Within AuthorityPlugin the tombstone wins over any value
// recorded at the same authority.
func (k Key[T]) Tombstone(b *Bag, setBy string) {
	b.setTombstone(k.name, AuthorityPlugin, setBy, position.Pos{})
}

// TombstoneAt is the authority-parameterised tombstone variant,
// symmetric with [Key.SetAt].
func (k Key[T]) TombstoneAt(b *Bag, auth Authority, setBy string, pos position.Pos) {
	b.setTombstone(k.name, auth, setBy, pos)
}

// TombstoneDirective records a tombstone at AuthorityDirective. The
// directive-override step calls this for -gen:meta directives that
// target the exact key name; prefix tombstones go through
// [Bag.TombstonePrefix] directly.
func (k Key[T]) TombstoneDirective(b *Bag, pos position.Pos) {
	b.setTombstone(k.name, AuthorityDirective, "directive", pos)
}

// TombstoneManual records a tombstone at AuthorityManual.
func (k Key[T]) TombstoneManual(b *Bag, setBy string) {
	b.setTombstone(k.name, AuthorityManual, setBy, position.Pos{})
}

// SetDirectiveFromString parses raw via the registered Parser and
// stamps the typed result into b at AuthorityDirective with pos.
// Used by the directive-override step when it has the string form
// of a value (from `+gen:meta KEY=VALUE`) and a registered key but
// no static type.
//
// Returns [ErrParse] (wrapped) when raw cannot be parsed as T. The
// caller is expected to surface that as a positioned diagnostic.
func (k Key[T]) SetDirectiveFromString(b *Bag, raw string, pos position.Pos) error {
	v, err := k.parser(raw)
	if err != nil {
		return fmt.Errorf("meta: key %q: %w", k.name, err)
	}
	k.SetDirective(b, v, pos)
	return nil
}
