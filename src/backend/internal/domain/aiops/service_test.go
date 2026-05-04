package aiops

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"charging-ops/backend/internal/domain/auth"
)

func TestServiceExplainSessionUsesGenerator(t *testing.T) {
	repository := fakeRepository{session: testSessionContext()}
	service := NewService(repository, fakeGenerator{content: "模型解释"})

	insight, err := service.ExplainSession(authorizedContext(), SessionExplanationParams{SessionID: "CS-1"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if insight.Content != "模型解释" {
		t.Fatalf("expected generator content, got %q", insight.Content)
	}
	if insight.Provider != "fake" || insight.Target.Code != "CS-1" {
		t.Fatalf("unexpected insight metadata: %+v", insight)
	}
}

func TestServiceFallsBackWhenGeneratorFails(t *testing.T) {
	repository := fakeRepository{session: testSessionContext()}
	service := NewService(repository, fakeGenerator{err: errors.New("provider unavailable")})

	insight, err := service.ExplainSession(authorizedContext(), SessionExplanationParams{SessionID: "CS-1"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if insight.Provider != "local-fallback" {
		t.Fatalf("expected fallback provider, got %s", insight.Provider)
	}
	if !strings.Contains(insight.Content, "CS-1") {
		t.Fatalf("expected fallback to include session number, got %q", insight.Content)
	}
}

func TestServiceRejectsUnauthorizedSite(t *testing.T) {
	repository := fakeRepository{session: testSessionContext()}
	service := NewService(repository, nil)

	_, err := service.ExplainSession(unauthorizedContext(), SessionExplanationParams{SessionID: "CS-1"})
	if !errors.Is(err, auth.ErrForbidden) {
		t.Fatalf("expected forbidden, got %v", err)
	}
}

func TestServiceStationQARequiresQuestion(t *testing.T) {
	service := NewService(fakeRepository{site: testSiteContext()}, nil)

	_, err := service.AnswerStationQuestion(authorizedContext(), StationQAParams{SiteID: "HQ-CAMPUS"})
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("expected invalid request, got %v", err)
	}
}

type fakeRepository struct {
	session   sessionContext
	workOrder workOrderContext
	site      siteContext
}

func (f fakeRepository) FindSessionContext(ctx context.Context, sessionID string) (sessionContext, error) {
	return f.session, nil
}

func (f fakeRepository) FindWorkOrderContext(ctx context.Context, workOrderID string) (workOrderContext, error) {
	return f.workOrder, nil
}

func (f fakeRepository) FindSiteContext(ctx context.Context, siteID string) (siteContext, error) {
	return f.site, nil
}

type fakeGenerator struct {
	content string
	err     error
}

func (f fakeGenerator) Generate(ctx context.Context, prompt Prompt) (GeneratedContent, error) {
	if f.err != nil {
		return GeneratedContent{}, f.err
	}
	return GeneratedContent{Content: f.content, Provider: "fake", Model: "test"}, nil
}

func authorizedContext() context.Context {
	principal := auth.Principal{
		UserID:      "user-1",
		Permissions: []string{auth.PermissionAIRead},
		Sites:       []auth.AuthorizedSite{{ID: "site-1", Code: "HQ-CAMPUS"}},
	}
	return auth.WithPrincipal(context.Background(), principal)
}

func unauthorizedContext() context.Context {
	principal := auth.Principal{
		UserID:      "user-2",
		Permissions: []string{auth.PermissionAIRead},
		Sites:       []auth.AuthorizedSite{{ID: "site-2", Code: "OTHER"}},
	}
	return auth.WithPrincipal(context.Background(), principal)
}

func testSessionContext() sessionContext {
	now := time.Now().UTC()
	return sessionContext{
		Site: SiteRef{ID: "site-1", Code: "HQ-CAMPUS", Name: "总部园区"},
		Session: sessionFact{
			ID:            "session-1",
			SessionNo:     "CS-1",
			Status:        "pending_review",
			ChargerCode:   "AC-N-001",
			ConnectorCode: "AC-N-001-01",
			UpdatedAt:     now,
		},
		Events: []eventFact{{EventType: "RemoteStartSent", Source: "operations", OccurredAt: now}},
	}
}

func testSiteContext() siteContext {
	return siteContext{
		Site:                SiteRef{ID: "site-1", Code: "HQ-CAMPUS", Name: "总部园区"},
		LoadLimitKW:         620,
		CurrentLoadKW:       320,
		ActiveSessions:      3,
		AvailableConnectors: 4,
	}
}
