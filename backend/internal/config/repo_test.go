package config

import (
	"context"
	"errors"
	"mch_api/internal/app"
	"mch_api/internal/domain"
	"reflect"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
)

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
		reflect.ValueOf(dest[i]).Elem().Set(reflect.ValueOf(v))
	}
	return nil
}

type valueRows struct {
	pgx.Rows
	rows   []valueRow
	index  int
	closed bool
	err    error
}

func (r *valueRows) Next() bool { return r.index < len(r.rows) }
func (r *valueRows) Scan(dest ...any) error {
	row := r.rows[r.index]
	r.index++
	return row.Scan(dest...)
}
func (r *valueRows) Close()     { r.closed = true }
func (r *valueRows) Err() error { return r.err }

type boundary struct {
	t     *testing.T
	ctx   context.Context
	sql   string
	args  []any
	calls int
	row   pgx.Row
	rows  pgx.Rows
	err   error
	tag   string
}

func (p *boundary) record(ctx context.Context, sql string, args []any) {
	require.Same(p.t, p.ctx, ctx)
	p.calls++
	require.Equal(p.t, 1, p.calls, "repository operation must not issue convenience queries")
	require.Equal(p.t, p.args, args)
	p.sql = strings.Join(strings.Fields(sql), " ")
}

func (p *boundary) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	p.record(ctx, sql, args)
	return p.row
}

func (p *boundary) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	p.record(ctx, sql, args)
	return p.rows, p.err
}

func (p *boundary) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	p.record(ctx, sql, args)
	return pgconn.NewCommandTag(p.tag), p.err
}

func newBoundary(t *testing.T) *boundary {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	return &boundary{t: t, ctx: ctx, tag: "UPDATE 1"}
}

func TestConfigRepositoryReads(t *testing.T) {
	q := validConfig()
	c := domain.Config(q)
	values := []any{c.Slug, c.ProjectDocs, c.EpicDocs, c.ChangeDocs, c.ChangePhases, c.ChangeColors, c.ChangeTypes}
	for _, details := range []bool{false, true} {
		for _, scenario := range []string{"values", "empty", "scan", "query", "iteration", "missing"} {
			p := newBoundary(t)
			if details {
				p.args = []any{"custom"}
			}
			failure := errors.New("failure")
			row := valueRow{t: t, values: values}
			rows := &valueRows{rows: []valueRow{row}}
			if scenario == "scan" {
				row.err = failure
				rows.rows = []valueRow{row}
			}
			if scenario == "query" {
				p.err = failure
				row.err = failure
			}
			if scenario == "missing" && details {
				row.err = pgx.ErrNoRows
			}
			if scenario == "iteration" {
				rows.err = failure
			}
			if scenario == "empty" {
				rows.rows = nil
			}
			p.row = row
			p.rows = rows
			r := &Repo{pool: p}
			var err error
			if details {
				var got domain.Config
				got, err = r.Details(p.ctx, domain.ConfigSlugRequest{Slug: "custom"})
				if err == nil {
					require.Equal(t, c, got)
				}
				require.Contains(t, p.sql, "where slug = $1")
			} else {
				var got []domain.Config
				got, err = r.List(p.ctx)
				if err == nil {
					require.NotNil(t, got)
					if scenario == "empty" {
						require.Empty(t, got)
					} else {
						require.Equal(t, []domain.Config{c}, got)
					}
				}
				require.Contains(t, p.sql, "order by slug")
				require.Equal(t, scenario != "query", rows.closed)
			}
			if scenario == "scan" || scenario == "query" || scenario == "iteration" && !details {
				require.ErrorIs(t, err, failure)
			} else if scenario == "missing" && details {
				require.ErrorIs(t, err, app.ErrConfigNotFound)
			} else {
				require.NoError(t, err)
			}
		}
	}
	require.Nil(t, NewRepo(nil).pool)
}

func TestConfigRepositoryMutations(t *testing.T) {
	for _, op := range []string{"Insert", "Update", "Delete"} {
		for _, scenario := range []string{"success", "missing", "failure", "constraint"} {
			p := newBoundary(t)
			q := validConfig()
			p.args = []any{q.Slug, q.ProjectDocs, q.EpicDocs, q.ChangeDocs, q.ChangePhases, q.ChangeColors, q.ChangeTypes}
			failure := errors.New("failure")
			var cause error
			if scenario == "failure" {
				cause = failure
			}
			if scenario == "constraint" {
				cause = &pgconn.PgError{Code: "23505", ConstraintName: "config_pkey"}
				if op == "Delete" {
					cause = &pgconn.PgError{Code: "23503"}
				}
			}
			p.err = cause
			if scenario == "missing" {
				p.tag = "UPDATE 0"
			}
			var err error
			r := &Repo{pool: p}
			switch op {
			case "Insert":
				p.row = valueRow{t: t, values: []any{q.Slug}, err: cause}
				var got domain.ConfigSlugRequest
				got, err = r.Insert(p.ctx, q)
				if err == nil {
					require.Equal(t, q.Slug, got.Slug)
				}
				require.Contains(t, p.sql, "returning slug")
				if scenario == "constraint" {
					require.ErrorIs(t, err, app.ErrConfigDuplicate)
				}
			case "Update":
				err = r.Update(p.ctx, q)
				require.Contains(t, p.sql, "where slug=$1")
				require.NotContains(t, p.sql, "set slug")
				if scenario == "missing" {
					require.ErrorIs(t, err, app.ErrConfigNotFound)
				}
			case "Delete":
				p.args = []any{q.Slug}
				p.row = valueRow{t: t, values: []any{scenario != "missing"}, err: cause}
				err = r.Delete(p.ctx, domain.ConfigSlugRequest{Slug: q.Slug})
				require.Equal(t, "select public.fn_config_delete($1)", p.sql)
				if scenario == "missing" {
					require.ErrorIs(t, err, app.ErrConfigNotFound)
				}
				if scenario == "constraint" {
					require.ErrorIs(t, err, app.ErrConfigInUse)
				}
			}
			if cause != nil {
				require.ErrorIs(t, err, cause)
			} else if scenario != "missing" || op == "Insert" {
				require.NoError(t, err)
			}
		}
	}
}
