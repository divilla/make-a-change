package epic

import (
	"context"
	"errors"
	"mch_api/internal/app"
	"mch_api/internal/domain"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
)

type boundaryPool struct {
	t     *testing.T
	ctx   context.Context
	args  []any
	sql   string
	calls int
	row   pgx.Row
	rows  pgx.Rows
	err   error
	tag   string
}

func (p *boundaryPool) check(ctx context.Context, sql string, args []any) {
	p.calls++
	require.Equal(p.t, 1, p.calls, "unexpected follow-up database call")
	require.Same(p.t, p.ctx, ctx)
	require.Equal(p.t, p.args, args)
	p.sql = strings.Join(strings.Fields(sql), " ")
}

func (p *boundaryPool) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	p.check(ctx, sql, args)
	return p.rows, p.err
}

func (p *boundaryPool) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	p.check(ctx, sql, args)
	return p.row
}

func (p *boundaryPool) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	p.check(ctx, sql, args)
	return pgconn.NewCommandTag(p.tag), p.err
}

type valueRow struct {
	t      *testing.T
	values []any
	err    error
}

func (r valueRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	require.Len(r.t, dest, len(r.values))
	for i, v := range r.values {
		require.Equal(r.t, reflect.TypeOf(v), reflect.TypeOf(dest[i]).Elem())
		reflect.ValueOf(dest[i]).Elem().Set(reflect.ValueOf(v))
	}
	return nil
}

type valueRows struct {
	pgx.Rows
	row       valueRow
	remaining int
	closed    bool
	err       error
}

func (r *valueRows) Next() bool {
	if r.remaining == 0 {
		return false
	}
	r.remaining--
	return true
}
func (r *valueRows) Scan(dest ...any) error { return r.row.Scan(dest...) }
func (r *valueRows) Close()                 { r.closed = true }
func (r *valueRows) Err() error             { return r.err }
func newBoundary(t *testing.T) *boundaryPool {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	return &boundaryPool{t: t, ctx: ctx}
}

func TestRepositoryReads(t *testing.T) {
	now := time.Now()
	failure := errors.New("read failed")
	values := []any{7, 9, "Name", int64(70000), int64(100000), 80000, now, now, false}
	for _, op := range []string{"details", "list", "list-inactive"} {
		scenarios := []string{"success", "scan", "missing", "wrapped missing"}
		if op != "details" {
			scenarios = []string{"success", "empty", "scan", "query", "iteration"}
		}
		for _, scenario := range scenarios {
			t.Run(op+"/"+scenario, func(t *testing.T) {
				p := newBoundary(t)
				p.args = []any{7}
				r := &Repo{pool: p}
				row := valueRow{t: t, values: values}
				rows := &valueRows{row: row, remaining: 1}
				switch scenario {
				case "empty":
					rows.remaining = 0
				case "scan":
					row.err = failure
					rows.row.err = failure
				case "query":
					p.err = failure
					row.err = failure
				case "iteration":
					rows.err = failure
				case "missing":
					row.err = pgx.ErrNoRows
				case "wrapped missing":
					row.err = app.WrapError(pgx.ErrNoRows, "query")
				}
				p.row = row
				p.rows = rows
				if op == "details" {
					got, err := r.Details(p.ctx, domain.EpicIDRequest{ID: 7})
					switch scenario {
					case "scan", "query":
						require.ErrorIs(t, err, failure)
					case "missing", "wrapped missing":
						require.ErrorIs(t, err, pgx.ErrNoRows)
						require.ErrorIs(t, err, app.ErrEpicNotFound)
					default:
						require.NoError(t, err)
						require.Equal(t, domain.Epic{ID: 7, ProjectID: 9, Name: "Name", DoneTC: 70000, TotalTC: 100000, ChangeCount: 80000, CreatedAt: now, UpdatedAt: now}, got)
					}
					require.Contains(t, p.sql, "where v.id = $1")
				} else {

					var got []domain.Epic
					var err error
					if op == "list-inactive" {
						got, err = r.ListInactive(p.ctx, domain.EpicListRequest{ProjectID: 7})
					} else {
						got, err = r.List(p.ctx, domain.EpicListRequest{ProjectID: 7})
					}
					switch scenario {
					case "scan", "query", "iteration":
						require.ErrorIs(t, err, failure)
					default:
						require.NoError(t, err)
						require.NotNil(t, got)
						if scenario == "empty" {
							require.Empty(t, got)
						} else {
							require.Len(t, got, 1)
							require.Equal(t, domain.Epic{ID: 7, ProjectID: 9, Name: "Name", DoneTC: 70000, TotalTC: 100000, ChangeCount: 80000, CreatedAt: now, UpdatedAt: now}, got[0])
						}
					}
					require.Equal(t, scenario != "query", rows.closed)
					require.Contains(t, p.sql, "order by v.name, v.id")
				}
				if op == "details" {
					require.Contains(t, p.sql, "from public.vw_epic_list v join public.epic e on e.id = v.id where v.id = $1")
					require.Contains(t, p.sql, "union all select "+epicColumns+" from public.vw_epic_inactive_list")
					require.NotContains(t, p.sql, "e.active =")
				} else {
					view := "public.vw_epic_list"
					if op == "list-inactive" {
						view = "public.vw_epic_inactive_list"
					}
					require.Contains(t, p.sql, "from "+view+" v join public.epic e on e.id = v.id")
				}
			})
		}
	}
	require.Nil(t, NewRepo(nil).pool)
}

func TestRepositorySingleStatementMutations(t *testing.T) {
	failure := errors.New("write failed")
	fk := &pgconn.PgError{Code: "23503"}
	for _, op := range []string{"create", "update", "delete"} {
		for _, cause := range []error{nil, failure, pgx.ErrNoRows, app.WrapError(fk, "constraint")} {
			affectedRows := []string{"1"}
			if cause == nil && op != "create" {
				affectedRows = []string{"0", "1"}
			}
			for _, affected := range affectedRows {
				p := newBoundary(t)
				p.err = cause
				p.tag = "UPDATE " + affected
				p.row = valueRow{t: t, values: []any{7}, err: cause}
				r := &Repo{pool: p}
				var err error
				want := cause
				switch op {
				case "create":
					p.args = []any{7, "Name"}
					var id domain.EpicIDRequest
					id, err = r.Create(p.ctx, domain.EpicCreateRequest{ProjectID: 7, Name: "Name"})
					if cause == nil {
						require.Equal(t, 7, id.ID)
					}
					require.Equal(t, "insert into public.epic (project_id, name) select id, $2 from public.project where id = $1 returning id", p.sql)
					if errors.Is(cause, pgx.ErrNoRows) || errors.Is(cause, fk) {
						want = app.ErrEpicNotFound
					}
				case "update":
					p.args = []any{7, "Name"}
					err = r.Update(p.ctx, domain.EpicUpdateRequest{ID: 7, Name: "Name"})
					require.Equal(t, "update public.epic set name = $2, updated_at = now() where id = $1", p.sql)
					if cause == nil && affected == "0" {
						want = app.ErrEpicNotFound
					}
				case "delete":
					p.args = []any{7}
					err = r.Delete(p.ctx, domain.EpicIDRequest{ID: 7})
					require.Equal(t, "delete from public.epic where id = $1", p.sql)
					if cause == nil && affected == "0" {
						want = app.ErrEpicNotFound
					}
					if errors.Is(cause, fk) {
						want = app.ErrEpicHasChanges
					}
				}
				require.ErrorIs(t, err, want)
				if cause != nil {
					require.ErrorIs(t, err, cause)
				}
			}
		}
	}
}
