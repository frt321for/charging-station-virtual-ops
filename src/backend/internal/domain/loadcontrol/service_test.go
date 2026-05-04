package loadcontrol

import (
	"context"
	"errors"
	"testing"
)

func TestServiceCreateRecord(t *testing.T) {
	tests := []struct {
		name      string
		params    CreateRecordParams
		wantErr   error
		wantSaved bool
	}{
		{
			name: "saves valid manual limit",
			params: CreateRecordParams{
				SiteID:       "HQ-CAMPUS",
				SessionID:    "CS-1",
				ActionType:   "limit_power",
				BeforeLoadKW: 56.8,
				AfterLoadKW:  52.1,
			},
			wantSaved: true,
		},
		{
			name:    "requires site",
			params:  CreateRecordParams{ActionType: "pause"},
			wantErr: ErrInvalidRecord,
		},
		{
			name:    "rejects invalid action",
			params:  CreateRecordParams{SiteID: "HQ-CAMPUS", ActionType: "bad"},
			wantErr: ErrInvalidRecord,
		},
		{
			name:    "rejects invalid status",
			params:  CreateRecordParams{SiteID: "HQ-CAMPUS", ActionType: "pause", Status: "bad"},
			wantErr: ErrInvalidRecord,
		},
		{
			name:    "rejects negative load",
			params:  CreateRecordParams{SiteID: "HQ-CAMPUS", ActionType: "pause", BeforeLoadKW: -1},
			wantErr: ErrInvalidRecord,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := &fakeRepository{}
			service := NewService(repository)

			_, err := service.CreateRecord(context.Background(), test.params)

			if test.wantErr == nil && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if test.wantErr != nil && !errors.Is(err, test.wantErr) {
				t.Fatalf("expected %v, got %v", test.wantErr, err)
			}
			if repository.saved != test.wantSaved {
				t.Fatalf("expected saved=%v, got %v", test.wantSaved, repository.saved)
			}
			if test.wantSaved && repository.params.OperatorName != "operations" {
				t.Fatalf("expected default operator, got %s", repository.params.OperatorName)
			}
			if test.wantSaved && repository.params.Status != "applied" {
				t.Fatalf("expected default status, got %s", repository.params.Status)
			}
		})
	}
}

type fakeRepository struct {
	saved  bool
	params CreateRecordParams
}

func (f *fakeRepository) Snapshot(ctx context.Context, siteID string) (Snapshot, error) {
	return Snapshot{}, nil
}

func (f *fakeRepository) CreateRecord(
	ctx context.Context,
	params CreateRecordParams,
) (LoadControlRecord, error) {
	f.saved = true
	f.params = params
	return LoadControlRecord{SiteID: params.SiteID, ActionType: params.ActionType}, nil
}
