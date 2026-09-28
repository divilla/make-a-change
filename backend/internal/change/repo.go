package change

import (
	"context"
	"mch_api/internal/app"
	"mch_api/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repo executes current change SQL.
type Repo struct{ pool changePool }

// Repository is the service's database boundary.
type Repository interface {
	List(context.Context, domain.ChangeListRequest) ([]domain.ChangeListItem, error)
	Details(context.Context, domain.ChangeIDRequest) (domain.ChangeDetails, error)
	Exists(context.Context, domain.ChangeIDRequest) error
	Project(context.Context, domain.ChangeIDRequest) (domain.ProjectIDRequest, error)
	EpicProject(context.Context, domain.EpicIDRequest) (domain.ProjectIDRequest, error)
	Create(context.Context, domain.ChangeCreateRequest) (domain.ChangeIDRequest, error)
	UpdateTitle(context.Context, domain.ChangeUpdateTitleRequest) error
	UpdatePhase(context.Context, domain.ChangeUpdatePhaseRequest) error
	UpdateEpic(context.Context, domain.ChangeUpdateEpicRequest) error
	UpdateAfterChange(context.Context, domain.ChangeUpdateAfterChangeRequest) error
	UpdateOpen(context.Context, domain.ChangeUpdateOpenRequest) error
	UpdateTypes(context.Context, domain.ChangeUpdateTypesRequest) error
	UpdatePRUrl(context.Context, domain.ChangeUpdatePRUrlRequest) error
	Delete(context.Context, domain.ChangeIDRequest) error
}

type changePool interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

var _ Repository = (*Repo)(nil)

const changeListColumns = `id, ref_uuid, ref, slug, project_id, change_phase, change_types,
 epic_id, epic_name, title, open, done_tc, total_tc, updated_at`

// NewRepo constructs the PostgreSQL repository.
func NewRepo(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool} }

// List reads current view fields in modification order.
func (r *Repo) List(ctx context.Context, req domain.ChangeListRequest) ([]domain.ChangeListItem, error) {
	rows, err := r.pool.Query(ctx, `select `+changeListColumns+` from public.vw_change_list where project_id = $1 order by updated_at desc, id`, req.ProjectID)
	if err != nil {
		return nil, app.Database(err, nil, nil)
	}
	defer rows.Close()
	result := make([]domain.ChangeListItem, 0)
	for rows.Next() {
		var c domain.ChangeListItem
		if err = rows.Scan(&c.ID, &c.RefUUID, &c.Ref, &c.Slug, &c.ProjectID, &c.ChangePhase, &c.ChangeTypes, &c.EpicID, &c.EpicName, &c.Title, &c.Open, &c.DoneTC, &c.TotalTC, &c.UpdatedAt); err != nil {
			return nil, app.Database(err, nil, nil)
		}
		result = append(result, c)
	}
	return result, app.Database(rows.Err(), nil, nil)
}

// Details reads current fields without doc or configuration dependencies.
func (r *Repo) Details(ctx context.Context, req domain.ChangeIDRequest) (domain.ChangeDetails, error) {
	var c domain.ChangeDetails
	err := r.pool.QueryRow(ctx, `select `+changeListColumns+`, pr_url, created_at, after_change_id from public.vw_change_details where id = $1`, req.ID).Scan(&c.ID, &c.RefUUID, &c.Ref, &c.Slug, &c.ProjectID, &c.ChangePhase, &c.ChangeTypes, &c.EpicID, &c.EpicName, &c.Title, &c.Open, &c.DoneTC, &c.TotalTC, &c.UpdatedAt, &c.PRUrl, &c.CreatedAt, &c.AfterChangeID)
	return c, app.Database(err, app.ErrChangeNotFound, nil)
}

// Exists supplies the preflight required by procedures that silently ignore missing rows.
func (r *Repo) Exists(ctx context.Context, req domain.ChangeIDRequest) error {
	var id int
	err := r.pool.QueryRow(ctx, `select id from public.change where id = $1`, req.ID).Scan(&id)
	return app.Database(err, app.ErrChangeNotFound, nil)
}

// Project reads only the parent needed for configuration and association checks.
func (r *Repo) Project(ctx context.Context, req domain.ChangeIDRequest) (domain.ProjectIDRequest, error) {
	var project domain.ProjectIDRequest
	err := r.pool.QueryRow(ctx, `select project_id from public.change where id = $1`, req.ID).Scan(&project.ID)
	return project, app.Database(err, app.ErrChangeNotFound, nil)
}

// EpicProject reads the reference's parent without requiring project configuration.
func (r *Repo) EpicProject(ctx context.Context, req domain.EpicIDRequest) (domain.ProjectIDRequest, error) {
	var project domain.ProjectIDRequest
	err := r.pool.QueryRow(ctx, `select project_id from public.epic where id = $1`, req.ID).Scan(&project.ID)
	return project, app.Database(err, app.ErrChangeInvalidReference, nil)
}

// Create delegates atomic change and initial brief creation to PostgreSQL.
func (r *Repo) Create(ctx context.Context, req domain.ChangeCreateRequest) (domain.ChangeIDRequest, error) {
	var id domain.ChangeIDRequest
	err := r.pool.QueryRow(ctx, `select public.fn_change_insert($1,$2,$3,$4)`, req.ProjectID, req.RefUUID, req.Title, req.Brief).Scan(&id.ID)
	return id, app.ChangeCreate(err)
}

// UpdateTitle executes one database mutation without reloading the entity.
func (r *Repo) UpdateTitle(ctx context.Context, req domain.ChangeUpdateTitleRequest) error {
	_, err := r.pool.Exec(ctx, `call public.sp_change_title_update($1,$2)`, req.ID, req.Title)
	return app.Database(err, nil, nil)
}

// UpdatePhase executes one database mutation without reloading the entity.
func (r *Repo) UpdatePhase(ctx context.Context, req domain.ChangeUpdatePhaseRequest) error {
	_, err := r.pool.Exec(ctx, `call public.sp_change_phase_update($1,$2)`, req.ID, req.ChangePhase)
	return app.Database(err, nil, nil)
}

// UpdateEpic executes one database mutation without reloading the entity.
func (r *Repo) UpdateEpic(ctx context.Context, req domain.ChangeUpdateEpicRequest) error {
	_, err := r.pool.Exec(ctx, `call public.sp_change_epic_update($1,$2)`, req.ID, req.EpicID)
	return app.Database(err, nil, app.ErrChangeInvalidReference)
}

// UpdateOpen executes one database mutation without reloading the entity.
func (r *Repo) UpdateOpen(ctx context.Context, req domain.ChangeUpdateOpenRequest) error {
	tag, err := r.pool.Exec(ctx, `update public.change set open = $2, updated_at = now() where id = $1`, req.ID, req.Open)
	if err != nil {
		return app.Database(err, nil, nil)
	}
	if tag.RowsAffected() == 0 {
		return app.ErrChangeNotFound
	}
	return nil
}

// UpdateTypes executes one database mutation without reloading the entity.
func (r *Repo) UpdateTypes(ctx context.Context, req domain.ChangeUpdateTypesRequest) error {
	tag, err := r.pool.Exec(ctx, `update public.change set change_types = $2, updated_at = now() where id = $1`, req.ID, req.ChangeTypes)
	if err != nil {
		return app.Database(err, nil, nil)
	}
	if tag.RowsAffected() == 0 {
		return app.ErrChangeNotFound
	}
	return nil
}

// UpdatePRUrl executes one database mutation without reloading the entity.
func (r *Repo) UpdatePRUrl(ctx context.Context, req domain.ChangeUpdatePRUrlRequest) error {
	tag, err := r.pool.Exec(ctx, `update public.change set pr_url = $2, updated_at = now() where id = $1`, req.ID, req.PRUrl)
	if err != nil {
		return app.Database(err, nil, nil)
	}
	if tag.RowsAffected() == 0 {
		return app.ErrChangeNotFound
	}
	return nil
}

// Delete executes one database mutation without reloading the entity.
func (r *Repo) Delete(ctx context.Context, req domain.ChangeIDRequest) error {
	tag, err := r.pool.Exec(ctx, `delete from public.change where id = $1`, req.ID)
	if err != nil {
		return app.Database(err, nil, app.ErrChangeHasTestCases)
	}
	if tag.RowsAffected() == 0 {
		return app.ErrChangeNotFound
	}
	return nil
}

// UpdateAfterChange executes one mutation without reloading the change.
func (r *Repo) UpdateAfterChange(ctx context.Context, req domain.ChangeUpdateAfterChangeRequest) error {
	tag, err := r.pool.Exec(ctx, `update public.change set after_change_id = $2, updated_at = now() where id = $1`, req.ID, req.AfterChangeID)
	if err != nil {
		return app.Database(err, nil, app.ErrChangeInvalidReference)
	}
	if tag.RowsAffected() == 0 {
		return app.ErrChangeNotFound
	}
	return nil
}
