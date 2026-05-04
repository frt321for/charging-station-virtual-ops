package command

import "errors"

// Domain errors returned by the command module.
var (
	ErrNotFound          = errors.New("command target not found")
	ErrInvalidCommand    = errors.New("invalid command")
	ErrInvalidReceipt    = errors.New("invalid command receipt")
	ErrChargerOffline    = errors.New("charger offline")
	ErrInvalidTransition = errors.New("invalid command transition")
)
