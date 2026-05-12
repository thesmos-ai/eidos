package blog

// Box is a generic envelope used by Comment to demonstrate that the
// builder generator handles type-parameter-carrying fields. The
// envelope holds a Value of an arbitrary type.
type Box[T any] struct {
	Value T
}

// Comment is the builder-only fixture entity — annotated for builder
// generation but not for repository or registry. Its generic Payload
// field forces the builder generator to render the field's type with
// its type parameter intact.
//
// +gen:builder
type Comment struct {
	// ID identifies the comment.
	ID int

	// Article references the article this comment belongs to.
	Article string

	// Author is the display name of whoever left the comment.
	Author string

	// Payload carries the comment text inside a generic envelope.
	Payload Box[string]
}
