package site

import "context"

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
	return s.repository.ListSummaries(ctx)
}

// GetTopology returns a nested charger topology for a site.
func (s *Service) GetTopology(ctx context.Context, siteID string) (Topology, error) {
	return s.repository.GetTopology(ctx, siteID)
}
