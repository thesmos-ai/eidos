package blog

import "context"

// Searcher is the multi-method fixture interface targeted by the
// mock generator. Its `+gen:mock` directive opts the interface into
// mocking without any repository chain — verifying the mock
// generator targets user-authored interfaces directly, not only the
// ones repogen synthesises.
//
// +gen:mock
type Searcher interface {
	// Find returns the article matching id, or nil when absent.
	Find(ctx context.Context, id string) (*Article, error)

	// Query returns every article whose Title contains q.
	Query(ctx context.Context, q string) ([]*Article, error)
}
