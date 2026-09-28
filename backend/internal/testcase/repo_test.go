package testcase

import (
	"context"
	"errors"
	"fmt"
	"mch_api/internal/domain"
	apperror "mch_api/internal/error"
	"reflect"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
)

type schemaRow []any

func (r schemaRow) Scan(dest ...any) error {
	if len(dest) != len(r) {
		return fmt.Errorf("got %d destinations for %d columns", len(dest), len(r))
	}
	for i, v := range r {
		reflect.ValueOf(dest[i]).Elem().Set(reflect.ValueOf(v))
	}
	return nil
}

type errorRow struct{ err error }

func (r errorRow) Scan(...any) error { return r.err }

type testRows struct {
	pgx.Rows
	data   []pgx.Row
	index  int
	closed bool
	err    error
}

func (r *testRows) Next() bool             { r.index++; return r.index <= len(r.data) }
func (r *testRows) Scan(dest ...any) error { return r.data[r.index-1].Scan(dest...) }
func (r *testRows) Close()                 { r.closed = true }
func (r *testRows) Err() error             { return r.err }

type expectedCall struct {
	method, sql string
	args        []any
	row         pgx.Row
	rows        pgx.Rows
	err         error
	tag         string
}
type testPool struct {
	t     *testing.T
	ctx   context.Context
	calls []expectedCall
}

func (p *testPool) take(ctx context.Context, method, sql string, args []any) expectedCall {
	p.t.Helper()
	require.NotEmpty(p.t, p.calls, "unexpected %s: %s", method, sql)
	call := p.calls[0]
	p.calls = p.calls[1:]
	require.Equal(p.t, call.method, method)
	require.Equal(p.t, call.sql, sql)
	require.Equal(p.t, call.args, args)
	require.Same(p.t, p.ctx, ctx)
	return call
}

func (p *testPool) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return p.take(ctx, "row", sql, args).row
}

func (p *testPool) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	c := p.take(ctx, "query", sql, args)
	return c.rows, c.err
}

func (p *testPool) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	c := p.take(ctx, "exec", sql, args)
	return pgconn.NewCommandTag(c.tag), c.err
}

func newTestRepo(t *testing.T, calls ...expectedCall) (*Repo, context.Context) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	p := &testPool{t: t, ctx: ctx, calls: calls}
	t.Cleanup(func() { require.Empty(t, p.calls, "missing database calls") })
	return NewRepo(p), ctx
}

func TestRepositoryCurrentSixColumnList(t *testing.T) {
	failure := errors.New("database failure")
	now := time.Now()
	modified := now.Add(time.Hour)
	for _, tc := range []struct {
		name           string
		parent         pgx.Row
		rows           *testRows
		queryErr, want error
		query          bool
		count          int
	}{
		{"ordered six columns", schemaRow{true}, &testRows{data: []pgx.Row{schemaRow{1, 42, "first", false, now, modified}, schemaRow{1 << 40, 42, "second", true, now, modified}}}, nil, nil, true, 2},
		{"empty live parent or deletion race", schemaRow{true}, &testRows{}, nil, nil, true, 0},
		{"missing parent", schemaRow{false}, nil, nil, apperror.ErrTestCaseNotFound, false, 0},
		{"parent scan", errorRow{failure}, nil, nil, failure, false, 0},
		{"query", schemaRow{true}, nil, failure, failure, true, 0},
		{"scan", schemaRow{true}, &testRows{data: []pgx.Row{errorRow{failure}}}, nil, failure, true, 0},
		{"iteration", schemaRow{true}, &testRows{data: []pgx.Row{schemaRow{1, 42, "first", false, now, modified}}, err: failure}, nil, failure, true, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := []expectedCall{{method: "row", sql: "select exists(select 1 from public.change where id = $1)", args: []any{42}, row: tc.parent}}
			if tc.query {
				calls = append(calls, expectedCall{method: "query", sql: "select id, change_id, scenario, done, created, modified from public.testcase where change_id = $1 order by id", args: []any{42}, rows: tc.rows, err: tc.queryErr})
			}
			r, ctx := newTestRepo(t, calls...)
			got, err := r.List(ctx, domain.TestCaseListRequest{ChangeID: 42})
			require.ErrorIs(t, err, tc.want)
			if tc.rows != nil {
				require.True(t, tc.rows.closed)
			}
			if tc.want == nil {
				require.NotNil(t, got)
				require.Len(t, got, tc.count)
			}
			if tc.count == 2 {
				require.Equal(t, []domain.TestCase{{ID: 1, ChangeID: 42, Scenario: "first", Created: now, Modified: modified}, {ID: 1 << 40, ChangeID: 42, Scenario: "second", Done: true, Created: now, Modified: modified}}, got)
			}
		})
	}
}

func TestRepositoryCreateOnlyID(t *testing.T) {
	r, ctx := newTestRepo(t, expectedCall{method: "row", sql: "insert into public.testcase(change_id,scenario) values($1,$2) returning id", args: []any{1 << 40, "Scenario"}, row: schemaRow{1 << 40}})
	got, err := r.Create(ctx, domain.TestCaseCreateRequest{ChangeID: 1 << 40, Scenario: "Scenario"})
	require.NoError(t, err)
	require.Equal(t, domain.TestCaseIDRequest{ID: 1 << 40}, got)
}

func TestRepositoryTranslationKeepsExternalCauses(t *testing.T) {
	for _, cause := range []error{errors.New("private failure"), pgx.ErrNoRows, &pgconn.PgError{Code: "23503", Message: "private FK"}, &pgconn.PgError{Code: "23505"}} {
		t.Run(cause.Error(), func(t *testing.T) {
			nested := fmt.Errorf("nested: %w", cause)
			r, ctx := newTestRepo(t, expectedCall{method: "row", sql: "insert into public.testcase(change_id,scenario) values($1,$2) returning id", args: []any{42, "Scenario"}, row: errorRow{nested}})
			_, err := r.Create(ctx, domain.TestCaseCreateRequest{ChangeID: 42, Scenario: "Scenario"})
			require.ErrorIs(t, err, cause)
			code := 500
			if pg, ok := cause.(*pgconn.PgError); ok {
				var actual *pgconn.PgError
				require.ErrorAs(t, apperror.HTTP(err), &actual)
				require.Same(t, pg, actual)
				if pg.Code == "23503" {
					require.ErrorIs(t, err, apperror.ErrTestCaseNotFound)
					code = 404
				}
			}
			status, _ := apperror.Interpret(err)
			require.Equal(t, code, status)
		})
	}
}

func TestRepositorySingleStatementMutations(t *testing.T) {
	for _, op := range []struct {
		name, sql string
		args      []any
		run       func(*Repo, context.Context) error
	}{
		{"scenario and same value", "update public.testcase set scenario=$2,modified=now() where id=$1", []any{1 << 40, "Scenario"}, func(r *Repo, c context.Context) error {
			return r.Update(c, domain.TestCaseUpdateRequest{ID: 1 << 40, Scenario: "Scenario"})
		}},
		{"done true and same value", "update public.testcase set done=$2,modified=now() where id=$1", []any{1 << 40, true}, func(r *Repo, c context.Context) error {
			return r.UpdateDone(c, domain.TestCaseUpdateDoneRequest{ID: 1 << 40, Done: true})
		}},
		{"done false", "update public.testcase set done=$2,modified=now() where id=$1", []any{1 << 40, false}, func(r *Repo, c context.Context) error {
			return r.UpdateDone(c, domain.TestCaseUpdateDoneRequest{ID: 1 << 40})
		}},
		{"delete", "delete from public.testcase where id=$1", []any{1 << 40}, func(r *Repo, c context.Context) error { return r.Delete(c, domain.TestCaseIDRequest{ID: 1 << 40}) }},
	} {
		t.Run(op.name, func(t *testing.T) {
			for _, tc := range []struct {
				name, tag   string
				cause, want error
			}{
				{"affected", "UPDATE 1", nil, nil},
				{"missing", "UPDATE 0", nil, apperror.ErrTestCaseNotFound},
				{"unknown", "", errors.New("exec failure"), nil},
				{"constraint", "", &pgconn.PgError{Code: "23503"}, nil},
			} {
				t.Run(tc.name, func(t *testing.T) {
					r, ctx := newTestRepo(t, expectedCall{method: "exec", sql: op.sql, args: op.args, err: tc.cause, tag: tc.tag})
					err := op.run(r, ctx)
					want := tc.want
					if tc.cause != nil {
						want = tc.cause
					}
					require.ErrorIs(t, err, want)
					if pg, ok := tc.cause.(*pgconn.PgError); ok {
						var actual *pgconn.PgError
						require.ErrorAs(t, err, &actual)
						require.Same(t, pg, actual)
					}
				})
			}
		})
	}
}
