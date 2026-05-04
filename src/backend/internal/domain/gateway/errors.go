package gateway

import "errors"

// Domain errors returned by the protocol gateway module.
var (
	ErrNotFound       = errors.New("gateway target not found")
	ErrInvalidRequest = errors.New("invalid gateway request")
)
