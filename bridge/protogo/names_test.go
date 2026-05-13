// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package protogo

import "testing"

// TestGoFieldName_Initialisms covers the initialism preservation
// rule the goFieldName helper applies. Each row asserts one
// snake_case → PascalCase translation under a documented
// initialism segment.
func TestGoFieldName_Initialisms(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in, want string
	}{
		{"user_id", "UserID"},
		{"http_status", "HTTPStatus"},
		{"api_key", "APIKey"},
		{"user_uuid", "UserUUID"},
		{"json_payload", "JSONPayload"},
		{"http_url", "HTTPURL"},
		{"id", "ID"},
		{"created_at", "CreatedAt"},
		{"", ""},
	}
	for _, tc := range cases {
		t.Run(tc.in+" → "+tc.want, func(t *testing.T) {
			t.Parallel()
			if got := goFieldName(tc.in); got != tc.want {
				t.Fatalf("goFieldName(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// TestGoPackageName covers the three resolution paths the
// helper supports: semicolon-suffix (the proto3-spec form),
// slash-form (path/to/pkg → pkg), and dot-form (dotted proto
// qualifier → last segment).
func TestGoPackageName(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in, want string
	}{
		{"github.com/example/foo;namedfoo", "namedfoo"},
		{"github.com/example/foo", "foo"},
		{"acme.cfg.bar", "bar"},
		{"plain", "plain"},
		{"", ""},
	}
	for _, tc := range cases {
		t.Run(tc.in+" → "+tc.want, func(t *testing.T) {
			t.Parallel()
			if got := goPackageName(tc.in); got != tc.want {
				t.Fatalf("goPackageName(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// TestGoImportPath covers the semicolon-suffix trim rule. The
// non-semicolon path returns the value unchanged.
func TestGoImportPath(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in, want string
	}{
		{"github.com/example/foo;namedfoo", "github.com/example/foo"},
		{"github.com/example/foo", "github.com/example/foo"},
		{"", ""},
	}
	for _, tc := range cases {
		t.Run(tc.in+" → "+tc.want, func(t *testing.T) {
			t.Parallel()
			if got := goImportPath(tc.in); got != tc.want {
				t.Fatalf("goImportPath(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
