package session

import (
	"errors"
	"testing"
)

func TestValidateTransition(t *testing.T) {
	tests := []struct {
		name    string
		from    Status
		to      Status
		wantErr error
	}{
		{
			name: "allows reserved to waiting arrival",
			from: StatusReserved,
			to:   StatusWaitingArrival,
		},
		{
			name: "allows charging to paused",
			from: StatusCharging,
			to:   StatusPaused,
		},
		{
			name:    "rejects billed to charging",
			from:    StatusBilled,
			to:      StatusCharging,
			wantErr: ErrInvalidTransition,
		},
		{
			name:    "rejects unknown target status",
			from:    StatusReserved,
			to:      Status("unknown"),
			wantErr: ErrInvalidStatus,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateTransition(test.from, test.to)
			if test.wantErr == nil && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if test.wantErr != nil && !errors.Is(err, test.wantErr) {
				t.Fatalf("expected %v, got %v", test.wantErr, err)
			}
		})
	}
}
