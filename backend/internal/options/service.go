package options

import (
	"context"

	"aipm/internal/dto"
)

// Repository defines Repository values.
type Repository interface {
	ChangePhases(ctx context.Context) ([]dto.ChangePhase, error)
	ChangeTypes(ctx context.Context) ([]dto.ChangeType, error)
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
func (s *Service) ChangePhases(ctx context.Context) ([]dto.ChangePhase, error) {
	return s.repo.ChangePhases(ctx)
}

// ChangeTypes executes ChangeTypes behavior.
func (s *Service) ChangeTypes(ctx context.Context) ([]dto.ChangeType, error) {
	return s.repo.ChangeTypes(ctx)
}
