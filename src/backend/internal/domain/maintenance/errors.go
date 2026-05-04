package maintenance

import "errors"

// ErrNotFound is returned when a maintenance resource cannot be found.
var ErrNotFound = errors.New("maintenance resource not found")

// ErrInvalidWorkOrder is returned when a work-order request is invalid.
var ErrInvalidWorkOrder = errors.New("invalid work order")
