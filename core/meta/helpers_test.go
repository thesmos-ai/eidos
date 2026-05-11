// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package meta_test

import (
	"encoding/json"
	"testing"

	"go.thesmos.sh/eidos/core/meta"
)

// assertEqualString fails the test if got and want differ.
func assertEqualString(t *testing.T, got, want string) {
	t.Helper()
	if got != want {
		t.Fatalf("string mismatch:\n got:  %q\n want: %q", got, want)
	}
}

// assertNoError fails the test if err is non-nil. The message
// includes context for debugging without forcing every call site to
// write a t.Fatalf.
func assertNoError(t *testing.T, err error, context string) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: unexpected error: %v", context, err)
	}
}

// Bag JSON tests share these typed keys. Declared at package level
// because [meta.NewKey] panics on duplicate registration — the keys
// must live as singletons across the test binary.
var (
	bagJSONScratchKey = meta.NewKey("test.scratch", meta.BoolParser)
	bagJSONNumKey     = meta.NewKey("test.num", meta.IntParser)
	bagJSONStrKey     = meta.NewKey("test.str", meta.StringParser)
	bagJSONBrokenKey  = meta.NewKey("test.broken", meta.IntParser)
	bagJSONChanKey    = meta.NewKey("test.chan", channelParser)
	bagJSONRawKey     = meta.NewKey("test.raw", rawMessageParser)
)

// channelParser is an inert parser for the chan-int key used to test
// JSON encoding failures — JSON has no shape for channels, so the
// parser is never invoked but the key needs a Parser to register.
func channelParser(string) (chan int, error) { return nil, meta.ErrParse }

// rawMessageParser is an inert parser for the [json.RawMessage] key
// used to inject malformed JSON into a [meta.Bag] for the outer-
// Marshal error path. The parser body is never invoked at runtime;
// the key only exists so [meta.NewKey] accepts it.
func rawMessageParser(string) (json.RawMessage, error) { return nil, meta.ErrParse }
