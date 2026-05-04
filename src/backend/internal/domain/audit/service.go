package audit

import "context"

type repository interface {
	List(ctx context.Context, params ListParams) (Page, error)
}

// Service coordinates audit queries.
type Service struct {
	repository repository
}

// NewService creates an audit service.
func NewService(repository repository) *Service {
	return &Service{repository: repository}
}

// List returns audit logs.
func (s *Service) List(ctx context.Context, params ListParams) (Page, error) {
	return s.repository.List(ctx, params)
}
