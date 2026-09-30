package config

import (
	"context"
	"mch_api/internal/app"
	"mch_api/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type configPool interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

// Repo persists configuration in PostgreSQL.
type Repo struct{ pool configPool }

// NewRepo constructs the configuration repository.
func NewRepo(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool} }

const columns = `slug, project_docs, epic_docs, change_docs, change_phases, change_colors, change_types`

func scanConfig(row pgx.Row) (domain.Config, error) {
	var c domain.Config
	err := row.Scan(&c.Slug, &c.ProjectDocs, &c.EpicDocs, &c.ChangeDocs, &c.ChangePhases, &c.ChangeColors, &c.ChangeTypes)
	return c, app.DatabaseError(err, nil, nil)
}

// List returns every configuration in slug order.
func (r *Repo) List(ctx context.Context) ([]domain.Config, error) {
	rows, err := r.pool.Query(ctx, `select `+columns+` from public.config order by slug`)
	if err != nil {
		return nil, app.DatabaseError(err, nil, nil)
	}
	defer rows.Close()
	result := make([]domain.Config, 0)
	for rows.Next() {
		c, err := scanConfig(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, c)
	}
	return result, app.DatabaseError(rows.Err(), nil, nil)
}

// Details reads the supplied primary key.
func (r *Repo) Details(ctx context.Context, req domain.ConfigSlugRequest) (domain.Config, error) {
	c, err := scanConfig(r.pool.QueryRow(ctx, `select `+columns+` from public.config where slug = $1`, req.Slug))
	return c, app.DatabaseError(err, app.ErrConfigNotFound, nil)
}

// Insert returns only the inserted slug.
func (r *Repo) Insert(ctx context.Context, req domain.ConfigWriteRequest) (domain.ConfigSlugRequest, error) {
	var result domain.ConfigSlugRequest
	err := r.pool.QueryRow(ctx, `insert into public.config (`+columns+`) values ($1,$2,$3,$4,$5,$6,$7) returning slug`, req.Slug, req.ProjectDocs, req.EpicDocs, req.ChangeDocs, req.ChangePhases, req.ChangeColors, req.ChangeTypes).Scan(&result.Slug)
	return result, app.ConfigInsert(err)
}

// Update replaces all mutable arrays without changing the slug or reloading.
func (r *Repo) Update(ctx context.Context, req domain.ConfigWriteRequest) error {
	tag, err := r.pool.Exec(ctx, `update public.config set project_docs=$2, epic_docs=$3, change_docs=$4, change_phases=$5, change_colors=$6, change_types=$7, updated_at=now() where slug=$1`, req.Slug, req.ProjectDocs, req.EpicDocs, req.ChangeDocs, req.ChangePhases, req.ChangeColors, req.ChangeTypes)
	if err != nil {
		return app.DatabaseError(err, nil, nil)
	}
	if tag.RowsAffected() == 0 {
		return app.ErrConfigNotFound
	}
	return nil
}

// Delete atomically refuses deletion of a config referenced by any project.
func (r *Repo) Delete(ctx context.Context, req domain.ConfigSlugRequest) error {
	var deleted bool
	err := r.pool.QueryRow(ctx, `select public.fn_config_delete($1)`, req.Slug).Scan(&deleted)
	if err != nil {
		return app.DatabaseError(err, nil, app.ErrConfigInUse)
	}
	if !deleted {
		return app.ErrConfigNotFound
	}
	return nil
}
