package finance

import "errors"

// ErrNotFound is returned when a finance resource cannot be found.
var ErrNotFound = errors.New("finance resource not found")

// ErrInvalidReview is returned when a review or correction request is invalid.
var ErrInvalidReview = errors.New("invalid finance review")
