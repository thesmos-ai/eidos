package blog

import "errors"

// errBlankTitle is the sentinel Article.Validate returns when the
// Title slot is empty. Kept in its own file so error declarations
// have a single home as the fixture grows.
var errBlankTitle = errors.New("blog: article title is required")
