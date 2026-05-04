package site

import (
	"context"

	"charging-ops/backend/internal/domain/auth"
)

type repository interface {
	ListSummaries(ctx context.Context) ([]Summary, error)
	GetTopology(ctx context.Context, siteID string) (Topology, error)
}

// Service coordinates site and topology queries.
type Service struct {
	repository repository
}

// NewService creates a site service.
func NewService(repository repository) *Service {
	return &Service{repository: repository}
}

// ListSummaries returns site overview rows.
func (s *Service) ListSummaries(ctx context.Context) ([]Summary, error) {
	summaries, err := s.repository.ListSummaries(ctx)
	if err != nil {
		return nil, err
	}
	filtered := summaries[:0]
	for _, summary := range summaries {
		if auth.CanAccessSite(ctx, summary.ID, summary.Code) {
			filtered = append(filtered, summary)
		}
	}
	return filtered, nil
}

// GetTopology returns a nested charger topology for a site.
func (s *Service) GetTopology(ctx context.Context, siteID string) (Topology, error) {
	topology, err := s.repository.GetTopology(ctx, siteID)
	if err != nil {
		return Topology{}, err
	}
	if !auth.CanAccessSite(ctx, topology.Site.ID, topology.Site.Code) {
		return Topology{}, auth.ErrForbidden
	}
	return topology, nil
}
