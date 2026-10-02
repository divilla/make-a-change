package project

import (
	"context"
	"mch_api/internal/app"
	"mch_api/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type (
	// Repo defines Repo values.
	Repo struct {
		pool projectPool
	}

	// Repository defines Repository values.
	Repository interface {
		List(ctx context.Context) ([]domain.Project, error)
		Details(ctx context.Context, req domain.ProjectIDRequest) (domain.Project, error)
		Create(ctx context.Context, req domain.ProjectCreateRequest) (domain.ProjectIDRequest, error)
		Update(ctx context.Context, req domain.ProjectUpdateRequest) error
		Delete(ctx context.Context, req domain.ProjectIDRequest) error
		Deactivate(ctx context.Context, req domain.ProjectIDRequest) error
		Config(ctx context.Context, req domain.ProjectIDRequest) (domain.Config, error)
	}
)

// NewRepo initializes or executes NewRepo behavior.
func NewRepo(pool *pgxpool.Pool) *Repo {
	return &Repo{pool: pool}
}

const projectColumns = "v.id, v.name, p.config_slug, p.last_ref, v.created_at, v.updated_at, v.change_count, p.active"

// List executes List behavior.
func (r *Repo) List(ctx context.Context) ([]domain.Project, error) {
	rows, err := r.pool.Query(ctx, `
		select `+projectColumns+`
		from public.vw_project_list v join public.project p on p.id = v.id
 order by v.active desc, v.updated_at desc, v.id desc
	`)
	if err != nil {
		return nil, app.DatabaseError(err, nil, nil)
	}
	defer rows.Close()
	projects := make([]domain.Project, 0)
	for rows.Next() {
		project, err := scanProject(rows)
		if err != nil {
			return nil, err
		}
		projects = append(projects, project)
	}
	return projects, app.DatabaseError(rows.Err(), nil, nil)
}

// Details executes Details behavior.
func (r *Repo) Details(ctx context.Context, req domain.ProjectIDRequest) (domain.Project, error) {
	project, err := scanProject(r.pool.QueryRow(ctx, `
		select `+projectColumns+`
		from public.vw_project_list v join public.project p on p.id = v.id
		where v.id = $1
	`, req.ID))
	if err != nil {
		return domain.Project{}, app.DatabaseError(err, app.ErrProjectNotFound, nil)
	}
	return project, nil
}

// Create inserts one project and returns only its ID.
func (r *Repo) Create(ctx context.Context, req domain.ProjectCreateRequest) (domain.ProjectIDRequest, error) {
	var result domain.ProjectIDRequest
	err := r.pool.QueryRow(ctx, "insert into public.project (name) values ($1) returning id", req.Name).Scan(&result.ID)
	return result, app.DatabaseError(err, nil, nil)
}

// Update changes the name and timestamp, including same-name updates.
func (r *Repo) Update(ctx context.Context, req domain.ProjectUpdateRequest) error {
	tag, err := r.pool.Exec(ctx, "update public.project set name = $2, updated_at = now() where id = $1", req.ID, req.Name)
	if err != nil {
		return app.DatabaseError(err, nil, nil)
	}
	if tag.RowsAffected() == 0 {
		return app.ErrProjectNotFound
	}
	return nil
}

// Delete lets PostgreSQL enforce parent dependencies in one statement.
func (r *Repo) Delete(ctx context.Context, req domain.ProjectIDRequest) error {
	tag, err := r.pool.Exec(ctx, "delete from public.project where id = $1", req.ID)
	if err != nil {
		return app.DatabaseError(err, nil, app.ErrProjectHasChanges)
	}
	if tag.RowsAffected() == 0 {
		return app.ErrProjectNotFound
	}
	return nil
}

func scanProject(row pgx.Row) (domain.Project, error) {
	var project domain.Project
	err := row.Scan(&project.ID, &project.Name, &project.ConfigSlug, &project.LastRef, &project.CreatedAt, &project.UpdatedAt, &project.ChangeCount, &project.Active)
	return project, app.DatabaseError(err, nil, nil)
}

type projectPool interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

// Config resolves exactly the project's stored slug. Missing joins never fall back.
func (r *Repo) Config(ctx context.Context, req domain.ProjectIDRequest) (domain.Config, error) {
	var result domain.Config
	err := r.pool.QueryRow(ctx, `select c.slug, c.project_docs, c.epic_docs, c.change_docs,
 c.change_phases, c.change_colors, c.change_types
 from public.project p join public.config c on c.slug = p.config_slug where p.id = $1`, req.ID).Scan(
		&result.Slug, &result.ProjectDocs, &result.EpicDocs, &result.ChangeDocs,
		&result.ChangePhases, &result.ChangeColors, &result.ChangeTypes)
	return result, app.DatabaseError(err, app.ErrProjectConfigNotFound, nil)
}

// Deactivate retains a referenced project after its DELETE was refused by PostgreSQL.
func (r *Repo) Deactivate(ctx context.Context, req domain.ProjectIDRequest) error {
	tag, err := r.pool.Exec(ctx, "update public.project set active = false, updated_at = now() where id = $1", req.ID)
	if err != nil {
		return app.DatabaseError(err, nil, nil)
	}
	if tag.RowsAffected() == 0 {
		return app.ErrProjectNotFound
	}
	return nil
}
