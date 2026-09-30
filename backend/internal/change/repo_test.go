package change

import (
	"context"
	"errors"
	"mch_api/internal/app"
	"mch_api/internal/domain"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid/v5"
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

func TestRepositoryCurrentReads(t *testing.T) {
	now := time.Now()
	refSlug := "012-slug"
	epic := 4
	name := "Epic"
	afterName := "Previous change #4"
	for _, op := range []string{"list", "details"} {
		for _, scenario := range []string{"values", "nullable", "empty", "scan", "query", "iteration", "missing"} {
			t.Run(op+"/"+scenario, func(t *testing.T) {
				failure := errors.New("read failed")
				p := newBoundary(t)
				p.args = []any{7}
				c := domain.ChangeListItem{ID: 7, RefUUID: "uuid", RefSlug: &refSlug, ProjectID: 9, ChangePhase: "backlog", ChangeTypes: []string{"fix"}, EpicID: &epic, EpicName: &name, Title: "Title", Open: true, DoneTC: 70000, TotalTC: 100000, UpdatedAt: now}
				if scenario == "nullable" {
					c.RefSlug = nil
					c.EpicID = nil
					c.EpicName = nil
				}
				values := []any{c.ID, c.RefUUID, c.RefSlug, c.ProjectID, c.ChangePhase, c.ChangeTypes, c.EpicID, c.EpicName, c.Title, c.Open, c.DoneTC, c.TotalTC, c.UpdatedAt}
				switch op {
				case "details":
					values = append(values, "https://pr", now, &epic, &afterName)
				}
				row := valueRow{t: t, values: values}
				rows := &valueRows{}
				if scenario == "scan" {
					row.err = failure
				}
				if scenario == "missing" && op == "details" {
					row.err = app.Wrap(pgx.ErrNoRows, "nested")
				}
				if scenario == "query" {
					row.err = failure
					p.err = failure
				}
				rows.rows = []valueRow{row}
				if scenario == "empty" {
					rows.rows = nil
				}
				if scenario == "iteration" {
					rows.err = failure
				}
				p.row = row
				p.rows = rows
				r := &Repo{pool: p}
				var err error
				switch op {
				case "list":
					var got []domain.ChangeListItem
					got, err = r.List(p.ctx, domain.ChangeListRequest{ProjectID: 7})
					if err == nil {
						require.NotNil(t, got)
						if scenario != "empty" {
							require.Equal(t, []domain.ChangeListItem{c}, got)
						} else {
							require.Empty(t, got)
						}
					}
					require.Contains(t, p.sql, "from public.vw_change_list where project_id = $1 order by updated_at desc, id")
				case "details":
					var got domain.ChangeDetails
					got, err = r.Details(p.ctx, domain.ChangeIDRequest{ID: 7})
					if err == nil {
						require.Equal(t, domain.ChangeDetails{ChangeListItem: c, PRUrl: "https://pr", CreatedAt: now, AfterChangeID: &epic, AfterChangeName: &afterName}, got)
					}
					require.Contains(t, p.sql, "from public.vw_change_details where id = $1")
				}
				if scenario == "scan" || scenario == "query" || scenario == "iteration" && op != "details" {
					require.ErrorIs(t, err, failure)
				} else if scenario == "missing" && op == "details" {
					require.ErrorIs(t, err, pgx.ErrNoRows)
					require.ErrorIs(t, err, app.ErrChangeNotFound)
				} else {
					require.NoError(t, err)
				}
				if op != "details" {
					require.Equal(t, scenario != "query", rows.closed)
				}
				for _, removed := range []string{"version", "completed", "select *", "100 *"} {
					require.NotContains(t, p.sql, removed)
				}
			})
		}
	}
	require.Nil(t, NewRepo(nil).pool)
}

func TestRepositoryTargetedContext(t *testing.T) {
	failure := errors.New("query failure")
	for _, op := range []string{"exists", "project", "epic"} {
		for _, cause := range []error{nil, failure, app.Wrap(pgx.ErrNoRows, "nested")} {
			p := newBoundary(t)
			p.args = []any{7}
			p.row = valueRow{t: t, values: []any{9}, err: cause}
			r := &Repo{pool: p}
			var err error
			var got domain.ProjectIDRequest
			want := app.ErrChangeNotFound
			switch op {
			case "exists":
				err = r.Exists(p.ctx, domain.ChangeIDRequest{ID: 7})
				require.Equal(t, "select id from public.change where id = $1", p.sql)
			case "project":
				got, err = r.Project(p.ctx, domain.ChangeIDRequest{ID: 7})
				require.Equal(t, "select project_id from public.change where id = $1", p.sql)
			case "epic":
				got, err = r.EpicProject(p.ctx, domain.EpicIDRequest{ID: 7})
				want = app.ErrChangeInvalidReference
				require.Equal(t, "select project_id from public.epic where id = $1", p.sql)
			}
			require.ErrorIs(t, err, cause)
			if errors.Is(cause, pgx.ErrNoRows) {
				require.ErrorIs(t, err, want)
			}
			if cause == nil && op != "exists" {
				require.Equal(t, domain.ProjectIDRequest{ID: 9}, got)
			}
		}
	}
}

func TestRepositoryCreateOnlyID(t *testing.T) {
	u := uuid.Must(uuid.NewV4())
	for _, cause := range []error{nil, errors.New("scan failed"), &pgconn.PgError{Code: "23505", ConstraintName: "change_ref_uuid_idx"}, &pgconn.PgError{Code: "23503"}, &pgconn.PgError{Code: "23502", TableName: "change", ColumnName: "project_id"}} {
		p := newBoundary(t)
		p.args = []any{9, &u, "Title", "Brief"}
		p.row = valueRow{t: t, values: []any{7}, err: cause}
		got, err := (&Repo{pool: p}).Create(p.ctx, domain.ChangeCreateRequest{ProjectID: 9, RefUUID: &u, Title: "Title", Brief: "Brief"})
		require.Equal(t, "select public.fn_change_insert($1,$2,$3,$4)", p.sql)
		require.ErrorIs(t, err, cause)
		if cause == nil {
			require.Equal(t, domain.ChangeIDRequest{ID: 7}, got)
		}
		var pgErr *pgconn.PgError
		if errors.As(cause, &pgErr) {
			if pgErr.Code == "23505" {
				require.ErrorIs(t, err, app.ErrChangeDuplicateUUID)
			} else {
				require.ErrorIs(t, err, app.ErrProjectNotFound)
			}
		}
	}
}

func TestRepositorySingleStatementMutations(t *testing.T) {
	flag := false
	epic := 4
	cases := []struct {
		name, sql string
		args      []any
		call      func(*Repo, context.Context) error
		direct    bool
		fk        error
	}{
		{"title", "call public.sp_change_title_update($1,$2)", []any{7, "Title"}, func(r *Repo, c context.Context) error {
			return r.UpdateTitle(c, domain.ChangeUpdateTitleRequest{ID: 7, Title: "Title"})
		}, false, nil},
		{"slug", "update public.change set slug = $2, updated_at = now() where id = $1", []any{7, "new-slug"}, func(r *Repo, c context.Context) error {
			return r.UpdateSlug(c, domain.ChangeUpdateSlugRequest{ID: 7, Slug: "new-slug"})
		}, true, nil},
		{"phase", "call public.sp_change_phase_update($1,$2)", []any{7, "todo"}, func(r *Repo, c context.Context) error {
			return r.UpdatePhase(c, domain.ChangeUpdatePhaseRequest{ID: 7, ChangePhase: "todo"})
		}, false, nil},
		{"epic", "call public.sp_change_epic_update($1,$2)", []any{7, &epic}, func(r *Repo, c context.Context) error {
			return r.UpdateEpic(c, domain.ChangeUpdateEpicRequest{ID: 7, EpicID: &epic})
		}, false, app.ErrChangeInvalidReference},
		{"after-change", "update public.change set after_change_id = $2, updated_at = now() where id = $1", []any{7, &epic}, func(r *Repo, c context.Context) error {
			return r.UpdateAfterChange(c, domain.ChangeUpdateAfterChangeRequest{ID: 7, AfterChangeID: &epic})
		}, true, app.ErrChangeInvalidReference},
		{"open", "update public.change set open = $2, updated_at = now() where id = $1", []any{7, &flag}, func(r *Repo, c context.Context) error {
			return r.UpdateOpen(c, domain.ChangeUpdateOpenRequest{ID: 7, Open: &flag})
		}, true, nil},
		{"types", "update public.change set change_types = $2, updated_at = now() where id = $1", []any{7, []string{}}, func(r *Repo, c context.Context) error {
			return r.UpdateTypes(c, domain.ChangeUpdateTypesRequest{ID: 7, ChangeTypes: []string{}})
		}, true, nil},
		{"pr-url", "update public.change set pr_url = $2, updated_at = now() where id = $1", []any{7, "https://pr"}, func(r *Repo, c context.Context) error {
			return r.UpdatePRUrl(c, domain.ChangeUpdatePRUrlRequest{ID: 7, PRUrl: "https://pr"})
		}, true, nil},
		{"delete", "delete from public.change where id = $1", []any{7}, func(r *Repo, c context.Context) error { return r.Delete(c, domain.ChangeIDRequest{ID: 7}) }, true, app.ErrChangeHasTestCases},
	}
	for _, tc := range cases {
		for _, tag := range []string{"UPDATE 0", "UPDATE 1"} {
			for _, cause := range []error{nil, errors.New("exec failed"), app.Wrap(&pgconn.PgError{Code: "23503"}, "nested")} {
				t.Run(tc.name+"/"+tag, func(t *testing.T) {
					p := newBoundary(t)
					p.args = tc.args
					p.tag = tag
					p.err = cause
					err := tc.call(&Repo{pool: p}, p.ctx)
					require.Equal(t, tc.sql, p.sql)
					if cause == nil && tc.direct && tag == "UPDATE 0" {
						require.ErrorIs(t, err, app.ErrChangeNotFound)
					} else {
						require.ErrorIs(t, err, cause)
					}
					var pgErr *pgconn.PgError
					if errors.As(cause, &pgErr) && tc.fk != nil {
						require.ErrorIs(t, err, tc.fk)
					}
				})
			}
		}
	}
}
