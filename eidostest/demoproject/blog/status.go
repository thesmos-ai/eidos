package blog

// Status enumerates the publication states an Article moves through.
// The typed-iota declaration exercises the backend's enum-rendering
// path (typed underlying + iota promotion) on a fixture type that
// other fixture entities reference.
type Status int

const (
	// Draft indicates the article is still being written.
	Draft Status = iota

	// Published indicates the article is live.
	Published

	// Archived indicates the article is no longer current.
	Archived
)
