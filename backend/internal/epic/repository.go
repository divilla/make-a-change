package epic

import (
	"context"
	"errors"

	"aipm/internal/dto"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repo defines Repo values.
type Repo struct {
	pool *pgxpool.Pool
}

const epicColumns = "id, version, project_id, name, done_tc, total_tc, completed, change_count, created, modified"

// NewRepo initializes or executes NewRepo behavior.
func NewRepo(pool *pgxpool.Pool) *Repo {
	return &Repo{pool: pool}
}

// List executes List behavior.
func (r *Repo) List(ctx context.Context, projectID int) ([]dto.Epic, error) {
	rows, err := r.pool.Query(ctx, `
		select `+epicColumns+`
		from public.vw_epic
		where project_id = $1
		order by created, id
	`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	epics := make([]dto.Epic, 0)
	for rows.Next() {
		epic, err := scanEpic(rows)
		if err != nil {
			return nil, err
		}
		epics = append(epics, epic)
	}
	return epics, rows.Err()
}

// Get executes Get behavior.
func (r *Repo) Get(ctx context.Context, id int) (dto.Epic, error) {
	epic, err := scanEpic(r.pool.QueryRow(ctx, "select "+epicColumns+" from public.vw_epic where id = $1", id))
	if errors.Is(err, pgx.ErrNoRows) {
		return dto.Epic{}, ErrNotFound
	}
	return epic, err
}

// Create executes Create behavior.
func (r *Repo) Create(ctx context.Context, req dto.EpicCreateRequest) (dto.Epic, error) {
	if err := r.ensureProject(ctx, req.ProjectID); err != nil {
		return dto.Epic{}, err
	}
	var id int
	if err := r.pool.QueryRow(ctx, `
		insert into public.epic (project_id, name)
		values ($1, $2)
		returning id
	`, req.ProjectID, req.Name).Scan(&id); err != nil {
		return dto.Epic{}, err
	}
	return r.Get(ctx, id)
}

// Update executes Update behavior.
func (r *Repo) Update(ctx context.Context, req dto.EpicUpdateRequest) (dto.Epic, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return dto.Epic{}, err
	}
	defer tx.Rollback(ctx)

	current, err := getEpic(ctx, tx, req.ID)
	if err != nil {
		return dto.Epic{}, err
	}
	if current.Name == req.Name {
		if err := tx.Commit(ctx); err != nil {
			return dto.Epic{}, err
		}
		return current, nil
	}
	if _, err := tx.Exec(ctx, "call public.sp_epic_to_history($1, false)", req.ID); err != nil {
		return dto.Epic{}, err
	}
	tag, err := tx.Exec(ctx, `
		update public.epic
		set name = $2,
			version = version + 1,
			modified = now()
		where id = $1
	`, req.ID, req.Name)
	if err != nil {
		return dto.Epic{}, err
	}
	if tag.RowsAffected() == 0 {
		return dto.Epic{}, ErrNotFound
	}
	epic, err := getEpic(ctx, tx, req.ID)
	if err != nil {
		return dto.Epic{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return dto.Epic{}, err
	}
	return epic, nil
}

// Delete executes Delete behavior.
func (r *Repo) Delete(ctx context.Context, id int) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := getEpic(ctx, tx, id); err != nil {
		return err
	}
	var hasChanges bool
	if err := tx.QueryRow(ctx, "select exists(select 1 from public.change where epic_id = $1)", id).Scan(&hasChanges); err != nil {
		return err
	}
	if hasChanges {
		return ErrEpicHasChanges
	}
	if _, err := tx.Exec(ctx, "call public.sp_epic_to_history($1, true)", id); err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, "delete from public.epic where id = $1", id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return tx.Commit(ctx)
}

func (r *Repo) ensureProject(ctx context.Context, id int) error {
	var exists bool
	if err := r.pool.QueryRow(ctx, "select exists(select 1 from public.project where id = $1)", id).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return ErrNotFound
	}
	return nil
}

func getEpic(ctx context.Context, q queryer, id int) (dto.Epic, error) {
	epic, err := scanEpic(q.QueryRow(ctx, "select "+epicColumns+" from public.vw_epic where id = $1", id))
	if errors.Is(err, pgx.ErrNoRows) {
		return dto.Epic{}, ErrNotFound
	}
	return epic, err
}

func scanEpic(row pgx.Row) (dto.Epic, error) {
	var epic dto.Epic
	err := row.Scan(
		&epic.ID, &epic.Version, &epic.ProjectID, &epic.Name, &epic.DoneTC,
		&epic.TotalTC, &epic.Completed, &epic.ChangeCount, &epic.Created, &epic.Modified,
	)
	return epic, err
}

type queryer interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}
