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
// `proto.option.<full-name>` channel. Standard options (defined on
// google.protobuf.FileOptions) stamp under their short name —
// `proto.option.go_package`, `proto.option.deprecated`. Custom
// extension options stamp under the extension's proto FQN —
// `proto.option.acme.cfg.feature_flag`. Value types follow the
// spec's value-type table.
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
// meta bag. The key namespace and the typed value follow the spec's
// option-channel convention; unsupported value forms surface as
// positioned diag.Warn so an under-handled option doesn't silently
// vanish from the meta surface.
func stampOneOption(
	ps *diag.PluginSink, pkg *node.Package, fd protoreflect.FileDescriptor,
	field protoreflect.FieldDescriptor, value protoreflect.Value,
) {
	name := optionKeyName(field)
	pos := position.Pos{File: fd.Path()}
	switch field.Kind() {
	case protoreflect.StringKind:
		key := meta.EnsureKey(MetaOptionPrefix+name, meta.StringParser)
		key.SetAt(pkg.Meta(), value.String(), meta.AuthorityPlugin, FrontendName, pos)
	case protoreflect.BoolKind:
		key := meta.EnsureKey(MetaOptionPrefix+name, meta.BoolParser)
		key.SetAt(pkg.Meta(), value.Bool(), meta.AuthorityPlugin, FrontendName, pos)
	default:
		// Other value forms (int / uint / float / bytes / enum /
		// message / repeated / map) ride the same channel but their
		// typed-stamping paths land alongside the host kinds that
		// motivate them (message-level options, field-level options,
		// service-level options, …). The Phase-B fixture only
		// exercises string and bool file options; later milestones
		// extend this switch with the remaining value forms.
		ps.Warnf(
			pos,
			"protobuf: file option %s carries unsupported value kind %s; skipped",
			name, field.Kind(),
		)
	}
}

// optionKeyName returns the canonical option-channel key fragment
// for field. Custom extensions stamp under their proto FQN;
// standard options on google.protobuf.FileOptions stamp under the
// short well-known name (`go_package`, `deprecated`, …).
func optionKeyName(field protoreflect.FieldDescriptor) string {
	if field.IsExtension() {
		return string(field.FullName())
	}
	return string(field.Name())
}
