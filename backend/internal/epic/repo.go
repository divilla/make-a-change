package epic

import (
	"context"
	"mch_api/internal/domain"
	apperror "mch_api/internal/error"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repo defines Repo values.
type Repo struct {
	pool epicPool
}

const epicColumns = "id, project_id, name, done_tc, total_tc, change_count, created, modified"

// NewRepo initializes or executes NewRepo behavior.
func NewRepo(pool *pgxpool.Pool) *Repo {
	return &Repo{pool: pool}
}

// List executes List behavior.
func (r *Repo) List(ctx context.Context, req domain.EpicListRequest) ([]domain.Epic, error) {
	rows, err := r.pool.Query(ctx, `
		select `+epicColumns+`
		from public.vw_epic
		where project_id = $1
		order by created, id
	`, req.ProjectID)
	if err != nil {
		return nil, apperror.Database(err, nil, nil)
	}
	defer rows.Close()
	epics := make([]domain.Epic, 0)
	for rows.Next() {
		epic, err := scanEpic(rows)
		if err != nil {
			return nil, err
		}
		epics = append(epics, epic)
	}
	return epics, apperror.Database(rows.Err(), nil, nil)
}

// Get executes Get behavior.
func (r *Repo) Get(ctx context.Context, req domain.EpicIDRequest) (domain.Epic, error) {
	epic, err := scanEpic(r.pool.QueryRow(ctx, "select "+epicColumns+" from public.vw_epic where id = $1", req.ID))
	if err != nil {
		return domain.Epic{}, apperror.Database(err, apperror.ErrEpicNotFound, nil)
	}
	return epic, nil
}

// Create inserts one epic and returns only its ID.
func (r *Repo) Create(ctx context.Context, req domain.EpicCreateRequest) (domain.EpicIDRequest, error) {
	var result domain.EpicIDRequest
	err := r.pool.QueryRow(ctx, "insert into public.epic (project_id, name) select id, $2 from public.project where id = $1 returning id", req.ProjectID, req.Name).Scan(&result.ID)
	return result, apperror.Database(err, apperror.ErrEpicNotFound, apperror.ErrEpicNotFound)
}

// Update changes the name and timestamp, including same-name updates.
func (r *Repo) Update(ctx context.Context, req domain.EpicUpdateRequest) error {
	tag, err := r.pool.Exec(ctx, "update public.epic set name = $2, modified = now() where id = $1", req.ID, req.Name)
	if err != nil {
		return apperror.Database(err, nil, nil)
	}
	if tag.RowsAffected() == 0 {
		return apperror.ErrEpicNotFound
	}
	return nil
}

// Delete lets PostgreSQL enforce parent dependencies in one statement.
func (r *Repo) Delete(ctx context.Context, req domain.EpicIDRequest) error {
	tag, err := r.pool.Exec(ctx, "delete from public.epic where id = $1", req.ID)
	if err != nil {
		return apperror.Database(err, nil, apperror.ErrEpicHasChanges)
	}
	if tag.RowsAffected() == 0 {
		return apperror.ErrEpicNotFound
	}
	return nil
}

func scanEpic(row pgx.Row) (domain.Epic, error) {
	var epic domain.Epic
	err := row.Scan(
		&epic.ID, &epic.ProjectID, &epic.Name, &epic.DoneTC,
		&epic.TotalTC, &epic.ChangeCount, &epic.Created, &epic.Modified,
	)
	return epic, apperror.Database(err, nil, nil)
}

type epicPool interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}
