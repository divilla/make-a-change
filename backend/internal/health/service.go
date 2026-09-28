package health

import (
	"context"
	"mch_api/internal/domain"
	apperror "mch_api/internal/error"
	"time"

	"github.com/rs/zerolog/log"
)

type (
	// Service defines Service values.
	Service struct {
		repo Repository
	}
)

// NewService initializes or executes NewService behavior.
func NewService(healthRepository Repository) *Service {
	return &Service{
		repo: healthRepository,
	}
}

// Check executes Check behavior.
func (s *Service) Check(ctx context.Context) domain.Health {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	res := domain.Health{
		Status:   "ok",
		API:      "ok",
		Database: "ok",
	}

	if err := s.repo.Ping(ctx); err != nil {
		log.Warn().Err(apperror.Wrap(err, "database health check")).Msg("database health check failed")
		res.Status = "degraded"
		res.Database = "error"
		res.Error = "database unavailable"
	}

	return res
}
