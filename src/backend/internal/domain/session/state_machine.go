package session

import (
	"errors"
	"fmt"
)

// ErrInvalidStatus is returned for an unknown session status.
var ErrInvalidStatus = errors.New("invalid session status")

// ErrInvalidTransition is returned when a lifecycle transition is not allowed.
var ErrInvalidTransition = errors.New("invalid session transition")

var allowedTransitions = map[Status][]Status{
	StatusReserved:       {StatusWaitingArrival, StatusCancelled},
	StatusWaitingArrival: {StatusPluggedIn, StatusCancelled},
	StatusPluggedIn:      {StatusStarting, StatusCancelled},
	StatusStarting:       {StatusCharging, StatusPendingReview, StatusCancelled},
	StatusCharging:       {StatusPaused, StatusStopping},
	StatusPaused:         {StatusCharging, StatusStopping},
	StatusStopping:       {StatusPendingBilling, StatusPendingReview},
	StatusPendingBilling: {StatusBilled, StatusPendingReview},
	StatusPendingReview:  {StatusBilled, StatusCancelled},
	StatusBilled:         {},
	StatusCancelled:      {},
}

// IsValidStatus reports whether the status is part of the documented lifecycle.
func IsValidStatus(status Status) bool {
	_, ok := allowedTransitions[status]
	return ok
}

// CanTransition reports whether a transition is allowed by the lifecycle.
func CanTransition(from Status, to Status) bool {
	targets, ok := allowedTransitions[from]
	if !ok {
		return false
	}
	for _, target := range targets {
		if target == to {
			return true
		}
	}
	return false
}

// ValidateTransition checks a status change against the session lifecycle.
func ValidateTransition(from Status, to Status) error {
	if !IsValidStatus(from) {
		return fmt.Errorf("%w: %s", ErrInvalidStatus, from)
	}
	if !IsValidStatus(to) {
		return fmt.Errorf("%w: %s", ErrInvalidStatus, to)
	}
	if !CanTransition(from, to) {
		return fmt.Errorf("%w: %s to %s", ErrInvalidTransition, from, to)
	}
	return nil
}
