package config

import (
	"context"
	"mch_api/internal/app"
	"mch_api/internal/domain"
	"strings"
)

// Repository supplies configuration persistence.
type Repository interface {
	List(context.Context) ([]domain.Config, error)
	Details(context.Context, domain.ConfigSlugRequest) (domain.Config, error)
	Insert(context.Context, domain.ConfigWriteRequest) (domain.ConfigSlugRequest, error)
	Update(context.Context, domain.ConfigWriteRequest) error
	Delete(context.Context, domain.ConfigSlugRequest) error
}

// Service validates immutable slugs and complete configuration replacements.
type Service struct{ repo Repository }

// NewService injects persistence.
func NewService(repo Repository) *Service { return &Service{repo: repo} }

// List returns all configurations ordered by slug.
func (s *Service) List(ctx context.Context) ([]domain.Config, error) { return s.repo.List(ctx) }

// Details returns the exact supplied slug without falling back to default.
func (s *Service) Details(ctx context.Context, req domain.ConfigSlugRequest) (domain.Config, error) {
	if strings.TrimSpace(req.Slug) == "" {
		return domain.Config{}, app.ErrConfigInvalidInput
	}
	return s.repo.Details(ctx, req)
}

func validWrite(req domain.ConfigWriteRequest) bool {
	if strings.TrimSpace(req.Slug) == "" {
		return false
	}
	for _, values := range [][]string{req.ProjectDocs, req.EpicDocs, req.ChangeDocs, req.ChangePhases, req.ChangeColors, req.ChangeTypes} {
		if values == nil {
			return false
		}
		for _, value := range values {
			if strings.TrimSpace(value) == "" {
				return false
			}
		}
	}
	return true
}

// Insert creates a configuration with a caller-provided slug.
func (s *Service) Insert(ctx context.Context, req domain.ConfigWriteRequest) (domain.ConfigSlugRequest, error) {
	if !validWrite(req) {
		return domain.ConfigSlugRequest{}, app.ErrConfigInvalidInput
	}
	return s.repo.Insert(ctx, req)
}

// Update replaces all arrays for the existing slug.
func (s *Service) Update(ctx context.Context, req domain.ConfigWriteRequest) error {
	if !validWrite(req) {
		return app.ErrConfigInvalidInput
	}
	return s.repo.Update(ctx, req)
}

// Delete leaves the atomic reference check to PostgreSQL.
func (s *Service) Delete(ctx context.Context, req domain.ConfigSlugRequest) error {
	if strings.TrimSpace(req.Slug) == "" {
		return app.ErrConfigInvalidInput
	}
	return s.repo.Delete(ctx, req)
}
