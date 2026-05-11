// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package meta_test

import (
	"errors"
	"testing"

	"go.thesmos.sh/eidos/core/meta"
	"go.thesmos.sh/eidos/core/position"
)

// Key tests share keys declared at package level so the global key
// registry is populated exactly once per test process. Per-test
// NewKey calls would re-register on -count > 1 and panic on the
// duplicate. The "key." prefix segregates these names from the
// other test files.
var (
	keyNewBasic                  = meta.NewKey("key.new.basic", meta.BoolParser)
	keyNewDuplicate              = meta.NewKey("key.new.duplicate", meta.BoolParser)
	keyLookupFound               = meta.NewKey("key.lookup.found", meta.BoolParser)
	keyGetBasic                  = meta.NewKey("key.get.basic", meta.StringParser)
	keyHasBasic                  = meta.NewKey("key.has.basic", meta.BoolParser)
	keySetBasic                  = meta.NewKey("key.set.basic", meta.BoolParser)
	keySetAtBasic                = meta.NewKey("key.setat.basic", meta.BoolParser)
	keySetDirectiveBasic         = meta.NewKey("key.setdirective.basic", meta.BoolParser)
	keySetManualBasic            = meta.NewKey("key.setmanual.basic", meta.BoolParser)
	keyTombstoneBasic            = meta.NewKey("key.tombstone.basic", meta.BoolParser)
	keyTombstoneAtBasic          = meta.NewKey("key.tombstoneat.basic", meta.BoolParser)
	keyTombstoneDirectiveBasic   = meta.NewKey("key.tombstonedirective.basic", meta.BoolParser)
	keyTombstoneManualBasic      = meta.NewKey("key.tombstonemanual.basic", meta.BoolParser)
	keySetDirectiveFromStringOk  = meta.NewKey("key.setdirectivefromstring.ok", meta.BoolParser)
	keySetDirectiveFromStringBad = meta.NewKey("key.setdirectivefromstring.bad", meta.BoolParser)
	keyAnyTombstoneDirective     = meta.NewKey("key.anykey.tombstonedirective", meta.BoolParser)
	keyEnsurePrior               = meta.NewKey("key.ensure.prior", meta.BoolParser)
	keyEnsureMismatch            = meta.NewKey("key.ensure.mismatch", meta.BoolParser)
)

func TestNewKey(t *testing.T) {
	t.Parallel()

	t.Run("returns a Key whose Name matches the registration", func(t *testing.T) {
		t.Parallel()
		if got := keyNewBasic.Name(); got != "key.new.basic" {
			t.Fatalf("Name = %q, want %q", got, "key.new.basic")
		}
	})

	t.Run("panics with ErrDuplicateKey when name is already registered", func(t *testing.T) {
		t.Parallel()
		// keyNewDuplicate was registered at package init; the second
		// attempt with the same name must panic with ErrDuplicateKey.
		defer func() {
			r := recover()
			if r == nil {
				t.Fatalf("expected panic on duplicate registration")
			}
			err, ok := r.(error)
			if !ok {
				t.Fatalf("panic value should be an error; got %T: %v", r, r)
			}
			if !errors.Is(err, meta.ErrDuplicateKey) {
				t.Fatalf("panic error = %v, want ErrDuplicateKey", err)
			}
		}()
		_ = meta.NewKey(keyNewDuplicate.Name(), meta.BoolParser)
	})
}

func TestLookup(t *testing.T) {
	t.Parallel()

	t.Run("returns the registered AnyKey", func(t *testing.T) {
		t.Parallel()
		got, err := meta.Lookup(keyLookupFound.Name())
		assertNoError(t, err, "Lookup")
		if got.Name() != keyLookupFound.Name() {
			t.Fatalf("Lookup returned %q, want %q", got.Name(), keyLookupFound.Name())
		}
	})

	t.Run("returns ErrUnregisteredKey for an unknown name", func(t *testing.T) {
		t.Parallel()
		_, err := meta.Lookup("definitely-not-registered")
		if !errors.Is(err, meta.ErrUnregisteredKey) {
			t.Fatalf("Lookup err = %v, want ErrUnregisteredKey", err)
		}
	})
}

func TestKey_Get(t *testing.T) {
	t.Parallel()

	t.Run("returns false for an unset key", func(t *testing.T) {
		t.Parallel()
		b := meta.NewBag()
		if _, ok := keyGetBasic.Get(b); ok {
			t.Fatalf("Get on unset key should be false")
		}
	})

	t.Run("returns the value set via Set", func(t *testing.T) {
		t.Parallel()
		b := meta.NewBag()
		keyGetBasic.Set(b, "hello", "test")
		got, ok := keyGetBasic.Get(b)
		if !ok {
			t.Fatalf("Get should return ok after Set")
		}
		assertEqualString(t, got, "hello")
	})

	t.Run("returns false when winning entry is a tombstone", func(t *testing.T) {
		t.Parallel()
		b := meta.NewBag()
		keyGetBasic.Set(b, "hello", "test")
		keyGetBasic.Tombstone(b, "test")
		if _, ok := keyGetBasic.Get(b); ok {
			t.Fatalf("Get should be false when value is tombstoned")
		}
	})
}

func TestKey_Has(t *testing.T) {
	t.Parallel()

	t.Run("returns true when the key resolves to a value", func(t *testing.T) {
		t.Parallel()
		b := meta.NewBag()
		keyHasBasic.Set(b, true, "test")
		if !keyHasBasic.Has(b) {
			t.Fatalf("Has should be true after Set")
		}
	})

	t.Run("returns false when the key is unset", func(t *testing.T) {
		t.Parallel()
		b := meta.NewBag()
		if keyHasBasic.Has(b) {
			t.Fatalf("Has should be false on unset key")
		}
	})
}

func TestKey_Set(t *testing.T) {
	t.Parallel()

	t.Run("records the value at AuthorityPlugin", func(t *testing.T) {
		t.Parallel()
		b := meta.NewBag()
		keySetBasic.Set(b, true, "writershape")
		p, ok := b.Winning(keySetBasic.Name())
		if !ok {
			t.Fatalf("Winning should report ok")
		}
		if p.Authority != meta.AuthorityPlugin {
			t.Fatalf("authority = %v, want Plugin", p.Authority)
		}
		if p.SetBy != "writershape" {
			t.Fatalf("setBy = %q, want %q", p.SetBy, "writershape")
		}
	})
}

func TestKey_SetAt(t *testing.T) {
	t.Parallel()

	t.Run("records at the supplied authority and position", func(t *testing.T) {
		t.Parallel()
		b := meta.NewBag()
		pos := position.At("x.go", 7, 3)
		keySetAtBasic.SetAt(b, true, meta.AuthorityDirective, "directive", pos)
		p, ok := b.Winning(keySetAtBasic.Name())
		if !ok {
			t.Fatalf("Winning should report ok")
		}
		if p.Authority != meta.AuthorityDirective || p.Pos != pos || p.SetBy != "directive" {
			t.Fatalf("provenance = %+v, expected directive at %v by directive", p, pos)
		}
	})
}

func TestKey_SetDirective(t *testing.T) {
	t.Parallel()

	t.Run("records at AuthorityDirective with directive setBy", func(t *testing.T) {
		t.Parallel()
		b := meta.NewBag()
		keySetDirectiveBasic.SetDirective(b, true, position.At("d.go", 1, 1))
		p, ok := b.Winning(keySetDirectiveBasic.Name())
		if !ok {
			t.Fatalf("Winning should report ok")
		}
		if p.Authority != meta.AuthorityDirective {
			t.Fatalf("authority = %v, want Directive", p.Authority)
		}
		if p.SetBy != "directive" {
			t.Fatalf("setBy = %q, want \"directive\"", p.SetBy)
		}
	})
}

func TestKey_SetManual(t *testing.T) {
	t.Parallel()

	t.Run("records at AuthorityManual with the supplied setBy", func(t *testing.T) {
		t.Parallel()
		b := meta.NewBag()
		keySetManualBasic.SetManual(b, true, "test-harness")
		p, ok := b.Winning(keySetManualBasic.Name())
		if !ok {
			t.Fatalf("Winning should report ok")
		}
		if p.Authority != meta.AuthorityManual {
			t.Fatalf("authority = %v, want Manual", p.Authority)
		}
		if p.SetBy != "test-harness" {
			t.Fatalf("setBy = %q, want \"test-harness\"", p.SetBy)
		}
	})
}

func TestKey_Tombstone(t *testing.T) {
	t.Parallel()

	t.Run("records a plugin-authority tombstone", func(t *testing.T) {
		t.Parallel()
		b := meta.NewBag()
		keyTombstoneBasic.Tombstone(b, "writershape")
		p, ok := b.Winning(keyTombstoneBasic.Name())
		if !ok || !p.Tombstone {
			t.Fatalf("expected a tombstone; got %+v ok=%v", p, ok)
		}
		if p.Authority != meta.AuthorityPlugin {
			t.Fatalf("authority = %v, want Plugin", p.Authority)
		}
	})
}

func TestKey_TombstoneAt(t *testing.T) {
	t.Parallel()

	t.Run("records a tombstone at the supplied authority", func(t *testing.T) {
		t.Parallel()
		b := meta.NewBag()
		keyTombstoneAtBasic.TombstoneAt(b, meta.AuthorityManual, "test", position.Pos{})
		p, ok := b.Winning(keyTombstoneAtBasic.Name())
		if !ok || !p.Tombstone || p.Authority != meta.AuthorityManual {
			t.Fatalf("expected Manual tombstone; got %+v ok=%v", p, ok)
		}
	})
}

func TestKey_TombstoneDirective(t *testing.T) {
	t.Parallel()

	t.Run("records a directive-authority tombstone", func(t *testing.T) {
		t.Parallel()
		b := meta.NewBag()
		keyTombstoneDirectiveBasic.TombstoneDirective(b, position.At("d.go", 1, 1))
		p, ok := b.Winning(keyTombstoneDirectiveBasic.Name())
		if !ok || !p.Tombstone || p.Authority != meta.AuthorityDirective {
			t.Fatalf("expected Directive tombstone; got %+v ok=%v", p, ok)
		}
	})
}

func TestKey_TombstoneManual(t *testing.T) {
	t.Parallel()

	t.Run("records a manual-authority tombstone", func(t *testing.T) {
		t.Parallel()
		b := meta.NewBag()
		keyTombstoneManualBasic.TombstoneManual(b, "test")
		p, ok := b.Winning(keyTombstoneManualBasic.Name())
		if !ok || !p.Tombstone || p.Authority != meta.AuthorityManual {
			t.Fatalf("expected Manual tombstone; got %+v ok=%v", p, ok)
		}
	})
}

func TestKey_SetDirectiveFromString(t *testing.T) {
	t.Parallel()

	t.Run("parses the raw value and stamps at AuthorityDirective", func(t *testing.T) {
		t.Parallel()
		b := meta.NewBag()
		err := keySetDirectiveFromStringOk.SetDirectiveFromString(b, "true", position.At("d.go", 1, 1))
		assertNoError(t, err, "SetDirectiveFromString")
		got, ok := keySetDirectiveFromStringOk.Get(b)
		if !ok || got != true {
			t.Fatalf("Get = (%v, %v), want (true, true)", got, ok)
		}
	})

	t.Run("returns the parser's error for an invalid raw value", func(t *testing.T) {
		t.Parallel()
		b := meta.NewBag()
		err := keySetDirectiveFromStringBad.SetDirectiveFromString(b, "bogus", position.At("d.go", 1, 1))
		if !errors.Is(err, meta.ErrParse) {
			t.Fatalf("err = %v, want ErrParse", err)
		}
	})
}

func TestEnsureKey(t *testing.T) {
	t.Parallel()

	t.Run("returns the existing Key when name was registered via NewKey", func(t *testing.T) {
		t.Parallel()
		// keyEnsurePrior was registered with BoolParser at package
		// init; EnsureKey with the same T must return that entry,
		// not register a new one.
		got := meta.EnsureKey(keyEnsurePrior.Name(), meta.BoolParser)
		if got.Name() != keyEnsurePrior.Name() {
			t.Fatalf("EnsureKey name = %q, want %q", got.Name(), keyEnsurePrior.Name())
		}
	})

	t.Run("registers and returns a fresh Key when name is unknown", func(t *testing.T) {
		t.Parallel()
		const name = "key.ensure.fresh"
		got := meta.EnsureKey(name, meta.StringParser)
		if got.Name() != name {
			t.Fatalf("EnsureKey name = %q, want %q", got.Name(), name)
		}
		// The fresh key must now be visible through the global registry.
		looked, err := meta.Lookup(name)
		assertNoError(t, err, "Lookup")
		if looked.Name() != name {
			t.Fatalf("Lookup returned %q, want %q", looked.Name(), name)
		}
	})

	t.Run("is idempotent: a second call reads state written via the first", func(t *testing.T) {
		t.Parallel()
		const name = "key.ensure.idempotent"
		first := meta.EnsureKey(name, meta.IntParser)
		second := meta.EnsureKey(name, meta.IntParser)
		b := meta.NewBag()
		first.Set(b, 42, "test")
		got, ok := second.Get(b)
		if !ok || got != 42 {
			t.Fatalf("Get via second EnsureKey = (%d, %v), want (42, true)", got, ok)
		}
	})

	t.Run("panics with ErrDuplicateKey when T differs from prior registration", func(t *testing.T) {
		t.Parallel()
		// keyEnsureMismatch was registered as Key[bool]; calling
		// EnsureKey with [string] must panic with ErrDuplicateKey.
		defer func() {
			r := recover()
			if r == nil {
				t.Fatalf("expected panic on type mismatch")
			}
			err, ok := r.(error)
			if !ok {
				t.Fatalf("panic value should be an error; got %T: %v", r, r)
			}
			if !errors.Is(err, meta.ErrDuplicateKey) {
				t.Fatalf("panic err = %v, want ErrDuplicateKey", err)
			}
		}()
		_ = meta.EnsureKey(keyEnsureMismatch.Name(), meta.StringParser)
	})
}

func TestAnyKey_TombstoneDirective(t *testing.T) {
	t.Parallel()

	t.Run("AnyKey.TombstoneDirective records a directive tombstone", func(t *testing.T) {
		t.Parallel()
		erased, err := meta.Lookup(keyAnyTombstoneDirective.Name())
		assertNoError(t, err, "Lookup")
		b := meta.NewBag()
		erased.TombstoneDirective(b, position.At("d.go", 1, 1))
		p, ok := b.Winning(keyAnyTombstoneDirective.Name())
		if !ok || !p.Tombstone || p.Authority != meta.AuthorityDirective {
			t.Fatalf("expected Directive tombstone via AnyKey; got %+v ok=%v", p, ok)
		}
	})
}
