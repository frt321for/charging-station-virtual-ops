package loadcontrol

import "errors"

// ErrNotFound is returned when a load-control object cannot be found.
var ErrNotFound = errors.New("load-control object not found")

// ErrInvalidRecord is returned when a control record request is invalid.
var ErrInvalidRecord = errors.New("invalid load-control record")
