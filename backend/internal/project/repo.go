package project

import (
	"context"
	"mch_api/internal/domain"
	apperror "mch_api/internal/error"

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
		Get(ctx context.Context, id int) (domain.Project, error)
		Create(ctx context.Context, name string) (domain.Project, error)
		Update(ctx context.Context, id int, name string) (domain.Project, error)
		Delete(ctx context.Context, id int) error
	}
)

// NewRepo initializes or executes NewRepo behavior.
func NewRepo(pool *pgxpool.Pool) *Repo {
	return &Repo{pool: pool}
}

const projectColumns = "id, name, last_ref, created, modified, change_count"

// List executes List behavior.
func (r *Repo) List(ctx context.Context) ([]domain.Project, error) {
	rows, err := r.pool.Query(ctx, `
		select `+projectColumns+`
		from public.vw_project
	`)
	if err != nil {
		return nil, apperror.Database(err, nil, nil)
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
	return projects, apperror.Database(rows.Err(), nil, nil)
}

// Get executes Get behavior.
func (r *Repo) Get(ctx context.Context, id int) (domain.Project, error) {
	project, err := scanProject(r.pool.QueryRow(ctx, `
		select `+projectColumns+`
		from public.vw_project
		where id = $1
	`, id))
	if err != nil {
		return domain.Project{}, apperror.Database(err, apperror.ErrProjectNotFound, nil)
	}
	return project, nil
}

// Create executes Create behavior.
func (r *Repo) Create(ctx context.Context, name string) (domain.Project, error) {
	var id int
	if err := r.pool.QueryRow(ctx, "insert into public.project (name) values ($1) returning id", name).Scan(&id); err != nil {
		return domain.Project{}, apperror.Database(err, nil, nil)
	}
	return r.Get(ctx, id)
}

// Update executes Update behavior.
func (r *Repo) Update(ctx context.Context, id int, name string) (domain.Project, error) {
	tag, err := r.pool.Exec(ctx, `
		update public.project
		set name = $2,
		    modified = now()
		where id = $1
	`, id, name)
	if err != nil {
		return domain.Project{}, apperror.Database(err, nil, nil)
	}
	if tag.RowsAffected() == 0 {
		return domain.Project{}, apperror.ErrProjectNotFound
	}
	return r.Get(ctx, id)
}

// Delete executes Delete behavior.
func (r *Repo) Delete(ctx context.Context, id int) error {
	tag, err := r.pool.Exec(ctx, `
		delete from public.project
		where id = $1
		  and not exists (
		    select 1
		    from public.change
		    where change.project_id = project.id
		  )
		  and not exists (
		    select 1
		    from public.epic
		    where epic.project_id = project.id
		  )
	`, id)
	if err != nil {
		return apperror.Database(err, nil, nil)
	}
	if tag.RowsAffected() > 0 {
		return nil
	}

	var exists bool
	if err := r.pool.QueryRow(ctx, "select exists(select 1 from public.project where id = $1)", id).Scan(&exists); err != nil {
		return apperror.Database(err, nil, nil)
	}
	if !exists {
		return apperror.ErrProjectNotFound
	}
	return apperror.ErrProjectHasChanges
}

func scanProject(row pgx.Row) (domain.Project, error) {
	var project domain.Project
	err := row.Scan(&project.ID, &project.Name, &project.LastRef, &project.Created, &project.Modified, &project.ChangeCount)
	return project, apperror.Database(err, nil, nil)
}

type projectPool interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}
