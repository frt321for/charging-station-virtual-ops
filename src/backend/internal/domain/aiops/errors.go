package aiops

import "errors"

// ErrNotFound is returned when an AI context target cannot be found.
var ErrNotFound = errors.New("ai context target not found")

// ErrInvalidRequest is returned when an AI request is incomplete.
var ErrInvalidRequest = errors.New("invalid ai request")
