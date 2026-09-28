package health

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type (
	// Repo defines Repo values.
	Repo struct {
		pool *pgxpool.Pool
	}

	// Repository defines Repository values.
	Repository interface {
		Ping(ctx context.Context) error
	}
)

// NewRepo initializes or executes NewRepo behavior.
func NewRepo(pool *pgxpool.Pool) *Repo {
	return &Repo{
		pool: pool,
	}
}

// Ping executes Ping behavior.
func (r *Repo) Ping(ctx context.Context) error {
	return r.pool.Ping(ctx)
}
