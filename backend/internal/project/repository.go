package project

import (
	"context"
	"errors"

	"aipm/internal/dto"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type (
	// Repo defines Repo values.
	Repo struct {
		pool *pgxpool.Pool
	}

	// Repository defines Repository values.
	Repository interface {
		List(ctx context.Context) ([]dto.Project, error)
		Get(ctx context.Context, id int) (dto.Project, error)
		Create(ctx context.Context, name string) (dto.Project, error)
		Update(ctx context.Context, id int, name string) (dto.Project, error)
		Delete(ctx context.Context, id int) error
	}
)

// NewRepo initializes or executes NewRepo behavior.
func NewRepo(pool *pgxpool.Pool) *Repo {
	return &Repo{pool: pool}
}

const projectColumns = "id, name, last_ref, created, modified, change_count"

// List executes List behavior.
func (r *Repo) List(ctx context.Context) ([]dto.Project, error) {
	rows, err := r.pool.Query(ctx, `
		select `+projectColumns+`
		from public.vw_project
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	projects := make([]dto.Project, 0)
	for rows.Next() {
		project, err := scanProject(rows)
		if err != nil {
			return nil, err
		}
		projects = append(projects, project)
	}
	return projects, rows.Err()
}

// Get executes Get behavior.
func (r *Repo) Get(ctx context.Context, id int) (dto.Project, error) {
	project, err := scanProject(r.pool.QueryRow(ctx, `
		select `+projectColumns+`
		from public.vw_project
		where id = $1
	`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return dto.Project{}, ErrNotFound
	}
	return project, err
}

// Create executes Create behavior.
func (r *Repo) Create(ctx context.Context, name string) (dto.Project, error) {
	var id int
	if err := r.pool.QueryRow(ctx, "insert into public.project (name) values ($1) returning id", name).Scan(&id); err != nil {
		return dto.Project{}, err
	}
	return r.Get(ctx, id)
}

// Update executes Update behavior.
func (r *Repo) Update(ctx context.Context, id int, name string) (dto.Project, error) {
	tag, err := r.pool.Exec(ctx, `
		update public.project
		set name = $2,
		    modified = now()
		where id = $1
	`, id, name)
	if err != nil {
		return dto.Project{}, err
	}
	if tag.RowsAffected() == 0 {
		return dto.Project{}, ErrNotFound
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
		return err
	}
	if tag.RowsAffected() > 0 {
		return nil
	}

	var exists bool
	if err := r.pool.QueryRow(ctx, "select exists(select 1 from public.project where id = $1)", id).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return ErrNotFound
	}
	return ErrProjectHasChanges
}

func scanProject(row pgx.Row) (dto.Project, error) {
	var project dto.Project
	err := row.Scan(&project.ID, &project.Name, &project.LastRef, &project.Created, &project.Modified, &project.ChangeCount)
	return project, err
}
