package billing

import "errors"

// ErrNotFound is returned when a billing resource cannot be found.
var ErrNotFound = errors.New("billing resource not found")

// ErrInvalidDraft is returned when a billing draft cannot be generated.
var ErrInvalidDraft = errors.New("invalid billing draft")
