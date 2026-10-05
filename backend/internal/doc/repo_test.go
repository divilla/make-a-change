package doc

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

func TestDocRepositoryReads(t *testing.T) {
	now := time.Now()
	d := domain.Doc{ID: 9, RefID: 7, RefTable: "epic", DocType: "prd", Body: "raw", AgentEdit: false, CreatedAt: now, UpdatedAt: now}
	values := []any{d.ID, d.RefID, d.RefTable, d.DocType, d.Body, d.AgentEdit, d.CreatedAt, d.UpdatedAt, d.DeletedAt}
	for _, op := range []string{"List", "ListActive", "CommentList", "Details"} {
		for _, scenario := range []string{"values", "empty", "scan", "query", "iteration", "missing"} {
			t.Run(op+scenario, func(t *testing.T) {
				failure := errors.New("failure")
				p := newBoundary(t)
				p.args = []any{7, "epic"}
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
				if scenario == "missing" && op == "Details" {
					row.err = pgx.ErrNoRows
				}
				if scenario == "iteration" {
					rows.err = failure
				}
				if scenario == "empty" {
					rows.rows = nil
				}
				p.rows = rows
				p.row = row
				r := &Repo{pool: p}
				var err error
				if op == "Details" {
					p.args = []any{9}
					var got domain.Doc
					got, err = r.Details(p.ctx, domain.DocIDRequest{ID: 9})
					if err == nil {
						require.Equal(t, d, got)
					}
					require.Contains(t, p.sql, "where id = $1")
				} else {
					var got []domain.Doc
					q := domain.DocListRequest{RefID: 7, RefTable: "epic"}
					switch op {
					case "List":
						got, err = r.List(p.ctx, q)
					case "CommentList":
						got, err = r.CommentList(p.ctx, q)
					default:
						got, err = r.ListActive(p.ctx, q)
					}
					if err == nil {
						require.NotNil(t, got)
						if scenario == "empty" {
							require.Empty(t, got)
						} else {
							require.Equal(t, []domain.Doc{d}, got)
						}
					}
					require.Contains(t, p.sql, "where ref_id = $1 and ref_table = $2")
					if op == "ListActive" {
						require.Contains(t, p.sql, "from public.vw_doc_active")
						require.Contains(t, p.sql, "order by doc_id desc")
					} else {
						require.Contains(t, p.sql, "order by id desc")
					}
					require.Equal(t, op == "CommentList", strings.Contains(p.sql, "doc_type = 'comment'"))
					require.NotContains(t, p.sql, "current")
					require.Equal(t, scenario != "query", rows.closed)
				}
				if scenario == "scan" || scenario == "query" || scenario == "iteration" && op != "Details" {
					require.ErrorIs(t, err, failure)
				} else if scenario == "missing" && op == "Details" {
					require.ErrorIs(t, err, app.ErrDocNotFound)
				} else {
					require.NoError(t, err)
				}
				require.NotContains(t, p.sql, "join")
				require.NotContains(t, p.sql, "select *")
			})
		}
	}
	require.Nil(t, NewRepo(nil).pool)
}

func TestDocRepositoryInsertAndProject(t *testing.T) {
	for _, table := range []string{"project", "epic", "change"} {
		for _, cause := range []error{nil, errors.New("failure"), pgx.ErrNoRows} {
			p := newBoundary(t)
			p.args = []any{7, table}
			p.row = valueRow{t: t, values: []any{9}, err: cause}
			got, err := (&Repo{pool: p}).Project(p.ctx, domain.DocListRequest{RefID: 7, RefTable: table})
			require.ErrorIs(t, err, cause)
			if cause == nil {
				require.Equal(t, 9, got.ID)
			}
			if errors.Is(cause, pgx.ErrNoRows) {
				require.ErrorIs(t, err, app.ErrDocParentNotFound)
			}
		}
	}
	for _, scenario := range []string{"insert", "error"} {
		p := newBoundary(t)
		f := false
		p.args = []any{"change", 7, "spec", "raw", &f}
		id := 9
		value := id
		failure := errors.New("failure")
		var cause error
		if scenario == "error" {
			cause = failure
		}
		p.row = valueRow{t: t, values: []any{value}, err: cause}
		got, err := (&Repo{pool: p}).Insert(p.ctx, domain.DocInsertRequest{RefID: 7, RefTable: "change", DocType: "spec", Body: "raw", AgentEdit: &f})
		require.Equal(t, "select public.fn_doc_insert($1::text,$2::bigint,$3::text,$4::text,$5::boolean)", p.sql)
		switch scenario {
		case "insert":
			require.NoError(t, err)
			require.Equal(t, 9, got.ID)
		case "error":
			require.ErrorIs(t, err, failure)
		}
	}
}

func TestCommentUndeleteRepositoryRestrictsTypeAndPreservesContent(t *testing.T) {
	for _, cause := range []error{nil, pgx.ErrNoRows, context.Canceled, errors.New("write failed")} {
		p := newBoundary(t)
		p.args = []any{8}
		p.row = valueRow{t: t, values: []any{8}, err: cause}
		err := (&Repo{pool: p}).CommentUndelete(p.ctx, domain.DocIDRequest{ID: 8})
		if errors.Is(cause, pgx.ErrNoRows) {
			require.ErrorIs(t, err, app.ErrDocNotFound)
		} else {
			require.ErrorIs(t, err, cause)
		}
		require.Equal(t, "update public.doc set deleted_by = null, deleted_at = null where id = $1 and doc_type = 'comment' returning id", p.sql)
	}
}

func TestActiveSetRepositoryCallsExistingProcedure(t *testing.T) {
	for _, cause := range []error{nil, context.Canceled, errors.New("procedure failed")} {
		p := newBoundary(t)
		p.args = []any{"change", 7, "spec", 8}
		p.err = cause
		err := (&Repo{pool: p}).ActiveSet(p.ctx, domain.Doc{ID: 8, RefID: 7, RefTable: "change", DocType: "spec"})
		require.ErrorIs(t, err, cause)
		require.Equal(t, "call public.sp_doc_active_set($1::text,$2::bigint,$3::text,$4::bigint)", p.sql)
	}
}
