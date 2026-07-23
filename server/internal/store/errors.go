package store

import "errors"

// Sentinel errors the api layer maps to HTTP status codes. Keeping these
// in store (not api) lets store functions be the single source of truth
// for "does this satisfy the authorization checklist" per
// docs/plans/2026-07-20-go-rewrite/01-database.md §5.
var (
	ErrNotFound   = errors.New("not found")
	ErrConflict   = errors.New("conflict")
	ErrForbidden  = errors.New("forbidden")
	ErrValidation = errors.New("validation")
)
