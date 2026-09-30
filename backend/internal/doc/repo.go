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

const columns = `id, ref_id, ref_table, doc_type, body, agent_edit, current, created_at, updated_at`

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

// Current reads current rows in deterministic descending ID order.
func (r *Repo) Current(ctx context.Context, req domain.DocListRequest) ([]domain.Doc, error) {
	rows, err := r.pool.Query(ctx, `select `+columns+` from public.doc where ref_id = $1 and ref_table = $2 and current = true order by id desc`, req.RefID, req.RefTable)
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
	err := row.Scan(&d.ID, &d.RefID, &d.RefTable, &d.DocType, &d.Body, &d.AgentEdit, &d.Current, &d.CreatedAt, &d.UpdatedAt)
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
	var id *int
	err := r.pool.QueryRow(ctx, `select public.fn_doc_insert($1,$2,$3,$4,$5)`, req.RefID, req.RefTable, req.DocType, req.Body, req.AgentEdit).Scan(&id)
	if err != nil {
		return domain.DocIDRequest{}, app.DatabaseError(err, nil, nil)
	}
	if id == nil {
		return domain.DocIDRequest{}, app.ErrDocParentNotFound
	}
	return domain.DocIDRequest{ID: *id}, nil
}
