package doc

import (
	"context"
	"mch_api/internal/app"
	"mch_api/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type docPool interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

// Repo executes PostgreSQL doc queries.
type Repo struct{ pool docPool }

// NewRepo constructs the shared doc repository.
func NewRepo(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool} }

const columns = `id, ref_id, ref_table, doc_type, body, agent_edit, created_at, updated_at, deleted_at`

// List reads all history for exactly one parent reference.
func (r *Repo) List(ctx context.Context, req domain.DocListRequest) ([]domain.Doc, error) {
	rows, err := r.pool.Query(ctx, `select `+columns+` from public.doc where ref_id = $1 and ref_table = $2 order by id desc`, req.RefID, req.RefTable)
	if err != nil {
		return nil, app.DatabaseError(err, nil, nil)
	}
	defer rows.Close()
	result := make([]domain.Doc, 0)
	for rows.Next() {
		d, err := scanDoc(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, d)
	}
	return result, app.DatabaseError(rows.Err(), nil, nil)
}

// ListActive reads the selected documents in deterministic descending ID order.
func (r *Repo) ListActive(ctx context.Context, req domain.DocListRequest) ([]domain.Doc, error) {
	rows, err := r.pool.Query(ctx, `select doc_id, ref_id, ref_table, doc_type, body, agent_edit, created_at, updated_at, deleted_at from public.vw_doc_active where ref_id = $1 and ref_table = $2 order by doc_id desc`, req.RefID, req.RefTable)
	if err != nil {
		return nil, app.DatabaseError(err, nil, nil)
	}
	defer rows.Close()
	result := make([]domain.Doc, 0)
	for rows.Next() {
		d, err := scanDoc(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, d)
	}
	return result, app.DatabaseError(rows.Err(), nil, nil)
}

func scanDoc(row pgx.Row) (domain.Doc, error) {
	var d domain.Doc
	err := row.Scan(&d.ID, &d.RefID, &d.RefTable, &d.DocType, &d.Body, &d.AgentEdit, &d.CreatedAt, &d.UpdatedAt, &d.DeletedAt)
	return d, app.DatabaseError(err, nil, nil)
}

// Details filters solely by the doc's primary key.
func (r *Repo) Details(ctx context.Context, req domain.DocIDRequest) (domain.Doc, error) {
	d, err := scanDoc(r.pool.QueryRow(ctx, `select `+columns+` from public.doc where id = $1`, req.ID))
	return d, app.DatabaseError(err, app.ErrDocNotFound, nil)
}

// Project resolves only the project needed for insert validation.
func (r *Repo) Project(ctx context.Context, req domain.DocListRequest) (domain.ProjectIDRequest, error) {
	var result domain.ProjectIDRequest
	err := r.pool.QueryRow(ctx, `select id from public.project where $2 = 'project' and id = $1
 union all select project_id from public.epic where $2 = 'epic' and id = $1
 union all select project_id from public.change where $2 = 'change' and id = $1`, req.RefID, req.RefTable).Scan(&result.ID)
	return result, app.DatabaseError(err, app.ErrDocParentNotFound, nil)
}

// Insert returns the ID from the database's atomic append operation.
func (r *Repo) Insert(ctx context.Context, req domain.DocInsertRequest) (domain.DocIDRequest, error) {
	var id domain.DocIDRequest
	err := r.pool.QueryRow(ctx, `select public.fn_doc_insert($1::text,$2::bigint,$3::text,$4::text,$5::boolean)`, req.RefTable, req.RefID, req.DocType, req.Body, req.AgentEdit).Scan(&id.ID)
	return id, app.DatabaseError(err, nil, nil)
}

// CommentList reads all comments, including soft-deleted history, for one reference.
func (r *Repo) CommentList(ctx context.Context, req domain.DocListRequest) ([]domain.Doc, error) {
	rows, err := r.pool.Query(ctx, `select `+columns+` from public.doc where ref_id = $1 and ref_table = $2 and doc_type = 'comment' order by id desc`, req.RefID, req.RefTable)
	if err != nil {
		return nil, app.DatabaseError(err, nil, nil)
	}
	defer rows.Close()
	result := make([]domain.Doc, 0)
	for rows.Next() {
		d, err := scanDoc(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, d)
	}
	return result, app.DatabaseError(rows.Err(), nil, nil)
}

// CommentInsert uses the separate database comment operation without an active selection.
func (r *Repo) CommentInsert(ctx context.Context, req domain.DocCommentInsertRequest) (domain.DocIDRequest, error) {
	var id domain.DocIDRequest
	err := r.pool.QueryRow(ctx, `select public.fn_doc_comment_insert($1::bigint,$2::text,$3::text,$4::boolean)`, req.RefID, req.RefTable, req.Body, req.AgentEdit).Scan(&id.ID)
	return id, app.DatabaseError(err, nil, nil)
}

// CommentUpdate only edits comments; a missing or non-comment ID returns no ID.
func (r *Repo) CommentUpdate(ctx context.Context, req domain.DocCommentUpdateRequest) error {
	var id *int
	err := r.pool.QueryRow(ctx, `select public.fn_doc_comment_update($1::bigint,$2::text)`, req.ID, req.Body).Scan(&id)
	if err != nil {
		return app.DatabaseError(err, app.ErrDocNotFound, nil)
	}
	if id == nil {
		return app.ErrDocNotFound
	}
	return nil
}

// Delete delegates soft deletion and removal of active selections to PostgreSQL.
func (r *Repo) Delete(ctx context.Context, req domain.DocIDRequest) error {
	var id *int
	err := r.pool.QueryRow(ctx, `select public.fn_doc_delete($1::bigint)`, req.ID).Scan(&id)
	if err != nil {
		return app.DatabaseError(err, app.ErrDocNotFound, nil)
	}
	if id == nil {
		return app.ErrDocNotFound
	}
	return nil
}
