package testcase

import (
	"context"
	"mch_api/internal/app"
	"mch_api/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type pool interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

// Repo reads and writes current testcase rows.
type Repo struct{ pool pool }

// NewRepo constructs the testcase repository.
func NewRepo(pool pool) *Repo { return &Repo{pool: pool} }

// List reads current cases after checking the live parent. A concurrent parent
// deletion between these independent reads may yield an empty list.
func (r *Repo) List(ctx context.Context, req domain.TestCaseListRequest) ([]domain.TestCase, error) {
	var exists bool
	if err := r.pool.QueryRow(ctx, "select exists(select 1 from public.change where id = $1)", req.ChangeID).Scan(&exists); err != nil {
		return nil, app.Database(err, nil, nil)
	}
	if !exists {
		return nil, app.ErrTestCaseNotFound
	}
	rows, err := r.pool.Query(ctx, "select id, change_id, scenario, done, created_at, updated_at from public.testcase where change_id = $1 order by id", req.ChangeID)
	if err != nil {
		return nil, app.Database(err, nil, nil)
	}
	defer rows.Close()
	cases := make([]domain.TestCase, 0)
	for rows.Next() {
		var tc domain.TestCase
		if err := rows.Scan(&tc.ID, &tc.ChangeID, &tc.Scenario, &tc.Done, &tc.CreatedAt, &tc.UpdatedAt); err != nil {
			return nil, app.Database(err, nil, nil)
		}
		cases = append(cases, tc)
	}
	return cases, app.Database(rows.Err(), nil, nil)
}

// Create inserts one case, leaving default state and timestamps to the database.
func (r *Repo) Create(ctx context.Context, req domain.TestCaseCreateRequest) (domain.TestCaseIDRequest, error) {
	var result domain.TestCaseIDRequest
	err := r.pool.QueryRow(ctx, "insert into public.testcase(change_id,scenario) values($1,$2) returning id", req.ChangeID, req.Scenario).Scan(&result.ID)
	return result, app.Database(err, nil, app.ErrTestCaseNotFound)
}

// Update changes the scenario and timestamp, including same-value writes.
func (r *Repo) Update(ctx context.Context, req domain.TestCaseUpdateRequest) error {
	tag, err := r.pool.Exec(ctx, "update public.testcase set scenario=$2,updated_at=now() where id=$1", req.ID, req.Scenario)
	if err != nil {
		return app.Database(err, nil, nil)
	}
	if tag.RowsAffected() == 0 {
		return app.ErrTestCaseNotFound
	}
	return nil
}

// UpdateDone changes completion state and timestamp, including same-value writes.
func (r *Repo) UpdateDone(ctx context.Context, req domain.TestCaseUpdateDoneRequest) error {
	tag, err := r.pool.Exec(ctx, "update public.testcase set done=$2,updated_at=now() where id=$1", req.ID, req.Done)
	if err != nil {
		return app.Database(err, nil, nil)
	}
	if tag.RowsAffected() == 0 {
		return app.ErrTestCaseNotFound
	}
	return nil
}

// Delete removes only the requested testcase.
func (r *Repo) Delete(ctx context.Context, req domain.TestCaseIDRequest) error {
	tag, err := r.pool.Exec(ctx, "delete from public.testcase where id=$1", req.ID)
	if err != nil {
		return app.Database(err, nil, nil)
	}
	if tag.RowsAffected() == 0 {
		return app.ErrTestCaseNotFound
	}
	return nil
}
