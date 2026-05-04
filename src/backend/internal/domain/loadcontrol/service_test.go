package loadcontrol

import (
	"context"
	"errors"
	"testing"

	"charging-ops/backend/internal/domain/auth"
)

func TestServiceCreateRecord(t *testing.T) {
	tests := []struct {
		name      string
		params    CreateRecordParams
		principal *auth.Principal
		target    recordTarget
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
			target:    recordTarget{SiteID: "site-1", SiteCode: "HQ-CAMPUS"},
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
			target:  recordTarget{SiteID: "site-1", SiteCode: "HQ-CAMPUS"},
			wantErr: ErrInvalidRecord,
		},
		{
			name:    "rejects invalid status",
			params:  CreateRecordParams{SiteID: "HQ-CAMPUS", ActionType: "pause", Status: "bad"},
			target:  recordTarget{SiteID: "site-1", SiteCode: "HQ-CAMPUS"},
			wantErr: ErrInvalidRecord,
		},
		{
			name:    "rejects negative load",
			params:  CreateRecordParams{SiteID: "HQ-CAMPUS", ActionType: "pause", BeforeLoadKW: -1},
			target:  recordTarget{SiteID: "site-1", SiteCode: "HQ-CAMPUS"},
			wantErr: ErrInvalidRecord,
		},
		{
			name: "rejects session outside requested site before saving",
			params: CreateRecordParams{
				SiteID:       "HQ-CAMPUS",
				SessionID:    "CS-OTHER",
				ActionType:   "pause",
				BeforeLoadKW: 56.8,
				AfterLoadKW:  52.1,
			},
			target:  recordTarget{SiteID: "site-2", SiteCode: "REMOTE-CAMPUS"},
			wantErr: auth.ErrForbidden,
		},
		{
			name: "rejects unauthorized scoped user before saving",
			params: CreateRecordParams{
				SiteID:       "REMOTE-CAMPUS",
				ActionType:   "pause",
				BeforeLoadKW: 56.8,
				AfterLoadKW:  52.1,
			},
			principal: &auth.Principal{
				UserID: "user-1",
				Sites:  []auth.AuthorizedSite{{ID: "site-1", Code: "HQ-CAMPUS"}},
			},
			target:  recordTarget{SiteID: "site-2", SiteCode: "REMOTE-CAMPUS"},
			wantErr: auth.ErrForbidden,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := &fakeRepository{target: test.target}
			service := NewService(repository)
			ctx := context.Background()
			if test.principal != nil {
				ctx = auth.WithPrincipal(ctx, *test.principal)
			}

			_, err := service.CreateRecord(ctx, test.params)

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
	target recordTarget
}

func (f *fakeRepository) Snapshot(ctx context.Context, siteID string) (Snapshot, error) {
	return Snapshot{}, nil
}

func (f *fakeRepository) FindRecordTarget(ctx context.Context, siteID string, sessionID string) (recordTarget, error) {
	if f.target.SiteID == "" && f.target.SiteCode == "" {
		return recordTarget{SiteID: siteID, SiteCode: siteID}, nil
	}
	return f.target, nil
}

func (f *fakeRepository) CreateRecord(
	ctx context.Context,
	params CreateRecordParams,
) (LoadControlRecord, error) {
	f.saved = true
	f.params = params
	return LoadControlRecord{SiteID: params.SiteID, ActionType: params.ActionType}, nil
}
