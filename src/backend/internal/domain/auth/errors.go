package auth

import "errors"

var (
	// ErrInvalidCredentials is returned when username or password is wrong.
	ErrInvalidCredentials = errors.New("invalid credentials")
	// ErrDisabledUser is returned when a disabled user tries to authenticate.
	ErrDisabledUser = errors.New("disabled user")
	// ErrUnauthorized is returned when a request has no valid session.
	ErrUnauthorized = errors.New("unauthorized")
	// ErrForbidden is returned when a user lacks a required permission.
	ErrForbidden = errors.New("forbidden")
)
