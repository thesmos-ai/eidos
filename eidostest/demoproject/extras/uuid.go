// Package extras supplies cross-package types referenced from the
// blog package. Its purpose in the fixture is to force the frontend
// to resolve an external import path and the backend's import
// machinery to register an alias when blog-package generators emit
// types that reference extras.UUID.
package extras

// UUID is a 16-byte universally unique identifier. The fixture uses
// it as the identity type on blog.Article and blog.User so generators
// emitting repository or builder code reference a non-builtin,
// cross-package type — exercising the writer.ImportSet path during
// rendering.
type UUID = [16]byte
