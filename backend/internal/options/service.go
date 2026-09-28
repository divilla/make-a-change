package options

import (
	"context"

	"mch_api/internal/domain"
)

// Repository defines Repository values.
type Repository interface {
	ChangePhases(ctx context.Context) ([]domain.ChangePhase, error)
	ChangeTypes(ctx context.Context) ([]domain.ChangeType, error)
}

// Service defines Service values.
type Service struct {
	repo Repository
}

// NewService initializes Service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// ChangePhases executes ChangePhases behavior.
func (s *Service) ChangePhases(ctx context.Context) ([]domain.ChangePhase, error) {
	return s.repo.ChangePhases(ctx)
}

// ChangeTypes executes ChangeTypes behavior.
func (s *Service) ChangeTypes(ctx context.Context) ([]domain.ChangeType, error) {
	return s.repo.ChangeTypes(ctx)
}
