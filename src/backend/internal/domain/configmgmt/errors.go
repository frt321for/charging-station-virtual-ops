package configmgmt

import "errors"

var (
	ErrForbidden      = errors.New("forbidden")
	ErrInvalidRequest = errors.New("invalid request")
	ErrNotFound       = errors.New("configuration target not found")
)
