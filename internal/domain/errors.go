package domain

import "errors"

// ErrNotFound is returned by service-layer ownership checks when an
// owner-scoped lookup (e.g. verifying that a foreign-key reference such as a
// project, area, section, location, tag, or task belongs to the requesting
// user) finds no matching row. Handlers map this to HTTP 404.
var ErrNotFound = errors.New("not found")
