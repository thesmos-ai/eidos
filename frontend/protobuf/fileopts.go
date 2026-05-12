// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package protobuf

import (
	"google.golang.org/protobuf/reflect/protoreflect"

	"go.thesmos.sh/eidos/core/diag"
	"go.thesmos.sh/eidos/core/meta"
	"go.thesmos.sh/eidos/core/position"
	"go.thesmos.sh/eidos/node"
)

// stampFileOptions walks fd's file-level options and stamps each
// set option on pkg's meta bag under the documented
// `proto.option.<full-name>` channel. Standard options (defined
// on google.protobuf.FileOptions) stamp under their short name —
// `proto.option.go_package`, `proto.option.deprecated`. Custom
// extension options stamp under the extension's proto FQN —
// `proto.option.acme.cfg.feature_flag`. Value types follow the
// documented value-type mapping.
//
// File-level options accumulate across multiple .proto files
// sharing one proto package: later files' options merge into the
// already-stamped package bag. The merge is order-deterministic
// because [convertFiles] sorts descriptors alphabetically by file
// path before invoking this helper. Duplicate non-repeated options
// across files set the same key twice and the last write wins; the
// protocompile parser already rejects same-file duplicate
// non-repeated options at parse time, so this only applies to
// genuinely-different files setting the same option, which is
// well-defined under proto3.
func stampFileOptions(ps *diag.PluginSink, pkg *node.Package, fd protoreflect.FileDescriptor) {
	opts := fd.Options()
	if opts == nil {
		return
	}
	opts.ProtoReflect().Range(func(field protoreflect.FieldDescriptor, value protoreflect.Value) bool {
		stampOneOption(ps, pkg, fd, field, value)
		return true
	})
}

// stampOneOption stamps one option (field, value) pair on pkg's
// meta bag. The key namespace and the typed value follow the
// option-channel convention; unsupported value forms surface as
// positioned diag.Warn so an under-handled option doesn't
// silently vanish from the meta surface.
//
// Cross-file collisions — the same option key set on two .proto
// files contributing to one proto package — surface as a Warn so
// users see the silent last-writer-wins behaviour rather than
// inheriting one file's value at random.
func stampOneOption(
	ps *diag.PluginSink, pkg *node.Package, fd protoreflect.FileDescriptor,
	field protoreflect.FieldDescriptor, value protoreflect.Value,
) {
	name := optionKeyName(field)
	pos := position.Pos{File: fd.Path()}
	switch field.Kind() {
	case protoreflect.StringKind:
		stampStringOption(ps, pkg, name, value.String(), fd.Path(), pos)
	case protoreflect.BoolKind:
		stampBoolOption(ps, pkg, name, value.Bool(), fd.Path(), pos)
	case protoreflect.EnumKind:
		variant := enumVariantName(field, value.Enum())
		if variant == "" {
			ps.Warnf(
				pos,
				"protobuf: file option %s carries unknown enum number %d; skipped",
				name, value.Enum(),
			)
			return
		}
		stampStringOption(ps, pkg, name, variant, fd.Path(), pos)
	default:
		// Other value forms (int / uint / float / bytes / message /
		// repeated / map) ride the same channel but their typed
		// stamping paths land alongside the host kinds that
		// exercise them. Until a host kind populates this switch,
		// callers receive a positioned diag.Warn rather than a
		// silent drop.
		ps.Warnf(
			pos,
			"protobuf: file option %s carries unsupported value kind %s; skipped",
			name, field.Kind(),
		)
	}
}

// stampStringOption stamps a string-typed option, surfacing a Warn
// when a sibling file in the same proto package already stamped a
// different value under the same key.
func stampStringOption(
	ps *diag.PluginSink, pkg *node.Package,
	name, value, file string, pos position.Pos,
) {
	key := meta.EnsureKey(MetaOptionPrefix+name, meta.StringParser)
	if prior, ok := key.Get(pkg.Meta()); ok && prior != value {
		ps.Warnf(
			pos,
			"protobuf: file option %s = %q on %s overwrites prior value %q from sibling file",
			name, value, file, prior,
		)
	}
	key.SetAt(pkg.Meta(), value, meta.AuthorityPlugin, FrontendName, pos)
}

// stampBoolOption stamps a bool-typed option with the same
// cross-file-overwrite guard [stampStringOption] applies.
func stampBoolOption(
	ps *diag.PluginSink, pkg *node.Package,
	name string, value bool, file string, pos position.Pos,
) {
	key := meta.EnsureKey(MetaOptionPrefix+name, meta.BoolParser)
	if prior, ok := key.Get(pkg.Meta()); ok && prior != value {
		ps.Warnf(
			pos,
			"protobuf: file option %s = %t on %s overwrites prior value %t from sibling file",
			name, value, file, prior,
		)
	}
	key.SetAt(pkg.Meta(), value, meta.AuthorityPlugin, FrontendName, pos)
}

// enumVariantName resolves an enum-typed option value to the
// variant's source-form name. Returns the empty string when the
// numeric value doesn't match any declared variant — protocompile
// normally rejects such inputs at parse time; the empty return is
// the defensive fallback callers surface as a Warn.
func enumVariantName(field protoreflect.FieldDescriptor, number protoreflect.EnumNumber) string {
	v := field.Enum().Values().ByNumber(number)
	if v == nil {
		return ""
	}
	return string(v.Name())
}

