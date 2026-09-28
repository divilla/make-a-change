package change

import (
	"context"
	"mch_api/internal/domain"
	apperror "mch_api/internal/error"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repo executes current change and document SQL.
type Repo struct{ pool changePool }

// Repository is the service's database boundary.
type Repository interface {
	List(context.Context, domain.ChangeListRequest) ([]domain.ChangeListItem, error)
	Details(context.Context, domain.ChangeIDRequest) (domain.ChangeDetails, error)
	Exists(context.Context, domain.ChangeIDRequest) error
	Project(context.Context, domain.ChangeIDRequest) (domain.ProjectIDRequest, error)
	EpicProject(context.Context, domain.EpicIDRequest) (domain.ProjectIDRequest, error)
	Artifacts(context.Context, domain.ChangeRenderedArtifactsRequest) ([]domain.ChangeArtifactSource, error)
	Documents(context.Context, domain.ChangeIDRequest) ([]domain.ChangeDocument, error)
	Create(context.Context, domain.ChangeCreateRequest) (domain.ChangeIDRequest, error)
	UpdateTitle(context.Context, domain.ChangeUpdateTitleRequest) error
	UpdatePhase(context.Context, domain.ChangeUpdatePhaseRequest) error
	UpdateEpic(context.Context, domain.ChangeUpdateEpicRequest) error
	UpdateOpen(context.Context, domain.ChangeUpdateOpenRequest) error
	UpdateChangeTypes(context.Context, domain.ChangeUpdateChangeTypesRequest) error
	UpdatePRUrl(context.Context, domain.ChangeUpdatePRUrlRequest) error
	SetDocument(context.Context, domain.ChangeDocumentSetRequest) error
	Delete(context.Context, domain.ChangeIDRequest) error
}

type changePool interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

var _ Repository = (*Repo)(nil)

const changeListColumns = `id, ref_uuid, ref, slug, project_id, change_phase, change_types,
 epic_id, epic_name, title, open, done_tc, total_tc, modified`

// NewRepo constructs the PostgreSQL repository.
func NewRepo(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool} }

// List reads current view fields in modification order.
func (r *Repo) List(ctx context.Context, req domain.ChangeListRequest) ([]domain.ChangeListItem, error) {
	rows, err := r.pool.Query(ctx, `select `+changeListColumns+` from public.vw_change_list where project_id = $1 order by modified desc, id`, req.ProjectID)
	if err != nil {
		return nil, apperror.Database(err, nil, nil)
	}
	defer rows.Close()
	result := make([]domain.ChangeListItem, 0)
	for rows.Next() {
		var c domain.ChangeListItem
		if err = rows.Scan(&c.ID, &c.RefUUID, &c.Ref, &c.Slug, &c.ProjectID, &c.ChangePhase, &c.ChangeTypes, &c.EpicID, &c.EpicName, &c.Title, &c.Open, &c.DoneTC, &c.TotalTC, &c.Modified); err != nil {
			return nil, apperror.Database(err, nil, nil)
		}
		result = append(result, c)
	}
	return result, apperror.Database(rows.Err(), nil, nil)
}

// Details reads current fields without document or configuration dependencies.
func (r *Repo) Details(ctx context.Context, req domain.ChangeIDRequest) (domain.ChangeDetails, error) {
	var c domain.ChangeDetails
	err := r.pool.QueryRow(ctx, `select `+changeListColumns+`, pr_url, created from public.vw_change_details where id = $1`, req.ID).Scan(&c.ID, &c.RefUUID, &c.Ref, &c.Slug, &c.ProjectID, &c.ChangePhase, &c.ChangeTypes, &c.EpicID, &c.EpicName, &c.Title, &c.Open, &c.DoneTC, &c.TotalTC, &c.Modified, &c.PRUrl, &c.Created)
	return c, apperror.Database(err, apperror.ErrChangeNotFound, nil)
}

// Exists supplies the preflight required by procedures that silently ignore missing rows.
func (r *Repo) Exists(ctx context.Context, req domain.ChangeIDRequest) error {
	var id int
	err := r.pool.QueryRow(ctx, `select id from public.change where id = $1`, req.ID).Scan(&id)
	return apperror.Database(err, apperror.ErrChangeNotFound, nil)
}

// Project reads only the parent needed for configuration and association checks.
func (r *Repo) Project(ctx context.Context, req domain.ChangeIDRequest) (domain.ProjectIDRequest, error) {
	var project domain.ProjectIDRequest
	err := r.pool.QueryRow(ctx, `select project_id from public.change where id = $1`, req.ID).Scan(&project.ID)
	return project, apperror.Database(err, apperror.ErrChangeNotFound, nil)
}

// EpicProject reads the reference's parent without requiring project configuration.
func (r *Repo) EpicProject(ctx context.Context, req domain.EpicIDRequest) (domain.ProjectIDRequest, error) {
	var project domain.ProjectIDRequest
	err := r.pool.QueryRow(ctx, `select project_id from public.epic where id = $1`, req.ID).Scan(&project.ID)
	return project, apperror.Database(err, apperror.ErrChangeInvalidReference, nil)
}

// Documents returns current rows only for a live change. Service preflight supplies 404.
func (r *Repo) Documents(ctx context.Context, req domain.ChangeIDRequest) ([]domain.ChangeDocument, error) {
	rows, err := r.pool.Query(ctx, `select d.id, d.doc_type, d.body, d.agent_edit, d.created
 from public.doc d join public.change c on c.id = d.ref_id
 where c.id = $1 and d.ref_table = 'change' and d.current
 order by d.doc_type, d.id`, req.ID)
	if err != nil {
		return nil, apperror.Database(err, nil, nil)
	}
	defer rows.Close()
	result := make([]domain.ChangeDocument, 0)
	for rows.Next() {
		var d domain.ChangeDocument
		if err = rows.Scan(&d.ID, &d.DocType, &d.Body, &d.AgentEdit, &d.Created); err != nil {
			return nil, apperror.Database(err, nil, nil)
		}
		result = append(result, d)
	}
	return result, apperror.Database(rows.Err(), nil, nil)
}

// Artifacts preserves request order and omits absent parents; latest current duplicates are deterministic.
func (r *Repo) Artifacts(ctx context.Context, req domain.ChangeRenderedArtifactsRequest) ([]domain.ChangeArtifactSource, error) {
	rows, err := r.pool.Query(ctx, `select c.id, coalesce(s.body, ''), coalesce(p.body, '')
 from public.change c
 left join lateral (select body from public.doc where ref_table = 'change' and ref_id = c.id and doc_type = 'spec' and current order by id desc limit 1) s on true
 left join lateral (select body from public.doc where ref_table = 'change' and ref_id = c.id and doc_type = 'pr' and current order by id desc limit 1) p on true
 where c.id = any($1::bigint[]) order by array_position($1::bigint[], c.id)`, req.IDs)
	if err != nil {
		return nil, apperror.Database(err, nil, nil)
	}
	defer rows.Close()
	result := make([]domain.ChangeArtifactSource, 0)
	for rows.Next() {
		var a domain.ChangeArtifactSource
		if err = rows.Scan(&a.ID, &a.Spec, &a.PR); err != nil {
			return nil, apperror.Database(err, nil, nil)
		}
		result = append(result, a)
	}
	return result, apperror.Database(rows.Err(), nil, nil)
}

// Create delegates atomic change and initial brief creation to PostgreSQL.
func (r *Repo) Create(ctx context.Context, req domain.ChangeCreateRequest) (domain.ChangeIDRequest, error) {
	var id domain.ChangeIDRequest
	err := r.pool.QueryRow(ctx, `select public.fn_change_insert($1,$2,$3,$4)`, req.ProjectID, req.RefUUID, req.Title, req.Brief).Scan(&id.ID)
	return id, apperror.ChangeCreate(err)
}

// UpdateTitle executes one database mutation without reloading the entity.
func (r *Repo) UpdateTitle(ctx context.Context, req domain.ChangeUpdateTitleRequest) error {
	_, err := r.pool.Exec(ctx, `call public.sp_change_title_update($1,$2)`, req.ID, req.Title)
	return apperror.Database(err, nil, nil)
}

// UpdatePhase executes one database mutation without reloading the entity.
func (r *Repo) UpdatePhase(ctx context.Context, req domain.ChangeUpdatePhaseRequest) error {
	_, err := r.pool.Exec(ctx, `call public.sp_change_phase_update($1,$2)`, req.ID, req.ChangePhase)
	return apperror.Database(err, nil, nil)
}

// UpdateEpic executes one database mutation without reloading the entity.
func (r *Repo) UpdateEpic(ctx context.Context, req domain.ChangeUpdateEpicRequest) error {
	_, err := r.pool.Exec(ctx, `call public.sp_change_epic_update($1,$2)`, req.ID, req.EpicID)
	return apperror.Database(err, nil, apperror.ErrChangeInvalidReference)
}

// SetDocument executes one database mutation without reloading the entity.
func (r *Repo) SetDocument(ctx context.Context, req domain.ChangeDocumentSetRequest) error {
	_, err := r.pool.Exec(ctx, `call public.sp_change_doc_set($1,$2,$3,$4)`, req.ID, req.DocType, req.Body, req.AgentEdit)
	return apperror.Database(err, nil, nil)
}

// UpdateOpen executes one database mutation without reloading the entity.
func (r *Repo) UpdateOpen(ctx context.Context, req domain.ChangeUpdateOpenRequest) error {
	tag, err := r.pool.Exec(ctx, `update public.change set open = $2, modified = now() where id = $1`, req.ID, req.Open)
	if err != nil {
		return apperror.Database(err, nil, nil)
	}
	if tag.RowsAffected() == 0 {
		return apperror.ErrChangeNotFound
	}
	return nil
}

// UpdateChangeTypes executes one database mutation without reloading the entity.
func (r *Repo) UpdateChangeTypes(ctx context.Context, req domain.ChangeUpdateChangeTypesRequest) error {
	tag, err := r.pool.Exec(ctx, `update public.change set change_types = $2, modified = now() where id = $1`, req.ID, req.ChangeTypes)
	if err != nil {
		return apperror.Database(err, nil, nil)
	}
	if tag.RowsAffected() == 0 {
		return apperror.ErrChangeNotFound
	}
	return nil
}

// UpdatePRUrl executes one database mutation without reloading the entity.
func (r *Repo) UpdatePRUrl(ctx context.Context, req domain.ChangeUpdatePRUrlRequest) error {
	tag, err := r.pool.Exec(ctx, `update public.change set pr_url = $2, modified = now() where id = $1`, req.ID, req.PRUrl)
	if err != nil {
		return apperror.Database(err, nil, nil)
	}
	if tag.RowsAffected() == 0 {
		return apperror.ErrChangeNotFound
	}
	return nil
}

// Delete executes one database mutation without reloading the entity.
func (r *Repo) Delete(ctx context.Context, req domain.ChangeIDRequest) error {
	tag, err := r.pool.Exec(ctx, `delete from public.change where id = $1`, req.ID)
	if err != nil {
		return apperror.Database(err, nil, apperror.ErrChangeHasTestCases)
	}
	if tag.RowsAffected() == 0 {
		return apperror.ErrChangeNotFound
	}
	return nil
}
