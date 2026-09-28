package project

import (
	"context"
	"errors"
	"mch_api/internal/domain"
	apperror "mch_api/internal/error"
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
	values := []any{7, "Name", "custom", int32(42), now, now, 70000}
	for _, op := range []string{"get", "list"} {
		scenarios := []string{"success", "scan", "missing", "wrapped missing"}
		if op == "list" {
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
					row.err = apperror.Wrap(pgx.ErrNoRows, "query")
				}
				p.row = row
				p.rows = rows
				if op == "get" {
					got, err := r.Get(p.ctx, domain.ProjectIDRequest{ID: 7})
					switch scenario {
					case "scan", "query":
						require.ErrorIs(t, err, failure)
					case "missing", "wrapped missing":
						require.ErrorIs(t, err, pgx.ErrNoRows)
						require.ErrorIs(t, err, apperror.ErrProjectNotFound)
					default:
						require.NoError(t, err)
						require.Equal(t, domain.Project{ID: 7, Name: "Name", Config: "custom", LastRef: 42, Created: now, Modified: now, ChangeCount: 70000}, got)
					}
					require.Contains(t, p.sql, "where v.id = $1")
				} else {
					p.args = nil
					got, err := r.List(p.ctx)
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
							require.Equal(t, domain.Project{ID: 7, Name: "Name", Config: "custom", LastRef: 42, Created: now, Modified: now, ChangeCount: 70000}, got[0])
						}
					}
					require.Equal(t, scenario != "query", rows.closed)
					require.Contains(t, p.sql, "order by v.modified desc, v.id desc")
				}
				require.Contains(t, p.sql, "v.id, v.name, p.config, p.last_ref, v.created, v.modified, v.change_count from public.vw_project v join public.project p on p.id = v.id")
			})
		}
	}
	require.Nil(t, NewRepo(nil).pool)
}

func TestRepositorySingleStatementMutations(t *testing.T) {
	failure := errors.New("write failed")
	fk := &pgconn.PgError{Code: "23503"}
	for _, op := range []string{"create", "update", "delete"} {
		for _, cause := range []error{nil, failure, pgx.ErrNoRows, apperror.Wrap(fk, "constraint")} {
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
					p.args = []any{"Name"}
					var id domain.ProjectIDRequest
					id, err = r.Create(p.ctx, domain.ProjectCreateRequest{Name: "Name"})
					if cause == nil {
						require.Equal(t, 7, id.ID)
					}
					require.Equal(t, "insert into public.project (name) values ($1) returning id", p.sql)

				case "update":
					p.args = []any{7, "Name"}
					err = r.Update(p.ctx, domain.ProjectUpdateRequest{ID: 7, Name: "Name"})
					require.Equal(t, "update public.project set name = $2, modified = now() where id = $1", p.sql)
					if cause == nil && affected == "0" {
						want = apperror.ErrProjectNotFound
					}
				case "delete":
					p.args = []any{7}
					err = r.Delete(p.ctx, domain.ProjectIDRequest{ID: 7})
					require.Equal(t, "delete from public.project where id = $1", p.sql)
					if cause == nil && affected == "0" {
						want = apperror.ErrProjectNotFound
					}
					if errors.Is(cause, fk) {
						want = apperror.ErrProjectHasChanges
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

func TestRepositorySelectedConfig(t *testing.T) {
	for _, cause := range []error{nil, pgx.ErrNoRows, errors.New("config read failed")} {
		p := newBoundary(t)
		p.args = []any{7}
		p.row = valueRow{t: t, values: []any{"custom", []string{"p2", "p1"}, []string{"e"}, []string{"c"}, []string{"phase"}, []string{"blue"}, []string{"type"}}, err: cause}
		got, err := (&Repo{pool: p}).Config(p.ctx, domain.ProjectIDRequest{ID: 7})
		if cause == nil {
			require.NoError(t, err)
			require.Equal(t, domain.Config{Slug: "custom", ProjectDocs: []string{"p2", "p1"}, EpicDocs: []string{"e"}, ChangeDocs: []string{"c"}, ChangePhases: []string{"phase"}, ChangeColors: []string{"blue"}, ChangeTypes: []string{"type"}}, got)
		} else {
			require.ErrorIs(t, err, cause)
		}
		if errors.Is(cause, pgx.ErrNoRows) {
			require.ErrorIs(t, err, apperror.ErrProjectConfigNotFound)
		}
		require.Contains(t, p.sql, "c.slug, c.project_docs, c.epic_docs, c.change_docs, c.change_phases, c.change_colors, c.change_types")
		require.Contains(t, p.sql, "join public.config c on c.slug = p.config where p.id = $1")
		require.NotContains(t, p.sql, "default")
	}
}
