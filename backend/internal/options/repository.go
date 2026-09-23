package options

import (
	"context"

	"aipm/internal/dto"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Repo defines Repo values.
type Repo struct {
	pool *pgxpool.Pool
}

// NewRepo initializes Repo.
func NewRepo(pool *pgxpool.Pool) *Repo {
	return &Repo{pool: pool}
}

// ChangePhases executes ChangePhases behavior.
func (r *Repo) ChangePhases(ctx context.Context) ([]dto.ChangePhase, error) {
	rows, err := r.pool.Query(ctx, `
		select slug,
		       priority,
		       case slug
		           when 'backlog' then '15'
		           when 'progress' then '10'
		           when 'review' then '11'
		           when 'staging' then '12'
		           when 'production' then '13'
		           when 'rejected' then '9'
		           else ''
		       end as color
		from public.change_phase
		order by priority, slug
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]dto.ChangePhase, 0)
	for rows.Next() {
		var item dto.ChangePhase
		if err := rows.Scan(&item.Slug, &item.Priority, &item.Color); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// ChangeTypes executes ChangeTypes behavior.
func (r *Repo) ChangeTypes(ctx context.Context) ([]dto.ChangeType, error) {
	rows, err := r.pool.Query(ctx, "select slug, priority from public.change_type order by priority, slug")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]dto.ChangeType, 0)
	for rows.Next() {
		var item dto.ChangeType
		if err := rows.Scan(&item.Slug, &item.Priority); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
