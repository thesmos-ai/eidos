package blog

import "example.com/demoproject/extras"

// User authors articles. Carries the repo and builder directives but
// not register — keeping it out of the registry-generator targets so
// the fixture exercises both "all three" (Article) and "two of three"
// (User) generator-directive combinations.
//
// +gen:repo
// +gen:builder
type User struct {
	// ID identifies the user in storage.
	ID extras.UUID

	// Name is the user's display name.
	Name string
}
