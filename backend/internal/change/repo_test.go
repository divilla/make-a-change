package change

import (
	"context"
	"errors"
	"mch_api/internal/domain"
	apperror "mch_api/internal/error"
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
	ref := int32(12)
	slug := "slug"
	epic := 4
	name := "Epic"
	for _, op := range []string{"list", "get", "documents", "artifacts"} {
		for _, scenario := range []string{"values", "nullable", "empty", "scan", "query", "iteration", "missing"} {
			t.Run(op+"/"+scenario, func(t *testing.T) {
				failure := errors.New("read failed")
				p := newBoundary(t)
				p.args = []any{7}
				c := domain.ChangeListItem{ID: 7, RefUUID: "uuid", Ref: &ref, Slug: &slug, ProjectID: 9, ChangePhase: "backlog", ChangeTypes: []string{"fix"}, EpicID: &epic, EpicName: &name, Title: "Title", Open: true, DoneTC: 70000, TotalTC: 100000, Modified: now}
				if scenario == "nullable" {
					c.Ref = nil
					c.Slug = nil
					c.EpicID = nil
					c.EpicName = nil
				}
				values := []any{c.ID, c.RefUUID, c.Ref, c.Slug, c.ProjectID, c.ChangePhase, c.ChangeTypes, c.EpicID, c.EpicName, c.Title, c.Open, c.DoneTC, c.TotalTC, c.Modified}
				switch op {
				case "get":
					values = append(values, "https://pr", now)
				case "documents":
					values = []any{8, "brief", "Raw", true, now}
				case "artifacts":
					values = []any{7, "Spec", "PR"}
					p.args = []any{[]int{7, 8}}
				}
				row := valueRow{t: t, values: values}
				rows := &valueRows{}
				if scenario == "scan" {
					row.err = failure
				}
				if scenario == "missing" && op == "get" {
					row.err = apperror.Wrap(pgx.ErrNoRows, "nested")
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
					require.Contains(t, p.sql, "from public.vw_change_list where project_id = $1 order by modified desc, id")
				case "get":
					var got domain.ChangeDetails
					got, err = r.Details(p.ctx, domain.ChangeIDRequest{ID: 7})
					if err == nil {
						require.Equal(t, domain.ChangeDetails{ChangeListItem: c, PRUrl: "https://pr", Created: now}, got)
					}
					require.Contains(t, p.sql, "from public.vw_change_details where id = $1")
				case "documents":
					var got []domain.ChangeDocument
					got, err = r.Documents(p.ctx, domain.ChangeIDRequest{ID: 7})
					if err == nil {
						require.NotNil(t, got)
						if scenario != "empty" {
							require.Equal(t, []domain.ChangeDocument{{ID: 8, DocType: "brief", Body: "Raw", AgentEdit: true, Created: now}}, got)
						} else {
							require.Empty(t, got)
						}
					}
					require.Contains(t, p.sql, "select d.id, d.doc_type, d.body, d.agent_edit, d.created")
					require.Contains(t, p.sql, "join public.change c on c.id = d.ref_id")
					require.Contains(t, p.sql, "c.id = $1 and d.ref_table = 'change' and d.current order by d.doc_type, d.id")
				case "artifacts":
					var got []domain.ChangeArtifactSource
					got, err = r.Artifacts(p.ctx, domain.ChangeRenderedArtifactsRequest{IDs: []int{7, 8}})
					if err == nil {
						require.NotNil(t, got)
						if scenario != "empty" {
							require.Equal(t, []domain.ChangeArtifactSource{{ID: 7, Spec: "Spec", PR: "PR"}}, got)
						} else {
							require.Empty(t, got)
						}
					}
					require.Contains(t, p.sql, "select c.id, coalesce(s.body, ''), coalesce(p.body, '') from public.change c")
					for _, kind := range []string{"spec", "pr"} {
						require.Contains(t, p.sql, "ref_table = 'change' and ref_id = c.id and doc_type = '"+kind+"' and current order by id desc limit 1")
					}
					require.Contains(t, p.sql, "where c.id = any($1::bigint[]) order by array_position($1::bigint[], c.id)")
				}
				if scenario == "scan" || scenario == "query" || scenario == "iteration" && op != "get" {
					require.ErrorIs(t, err, failure)
				} else if scenario == "missing" && op == "get" {
					require.ErrorIs(t, err, pgx.ErrNoRows)
					require.ErrorIs(t, err, apperror.ErrChangeNotFound)
				} else {
					require.NoError(t, err)
				}
				if op != "get" {
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
		for _, cause := range []error{nil, failure, apperror.Wrap(pgx.ErrNoRows, "nested")} {
			p := newBoundary(t)
			p.args = []any{7}
			p.row = valueRow{t: t, values: []any{9}, err: cause}
			r := &Repo{pool: p}
			var err error
			var got domain.ProjectIDRequest
			want := apperror.ErrChangeNotFound
			switch op {
			case "exists":
				err = r.Exists(p.ctx, domain.ChangeIDRequest{ID: 7})
				require.Equal(t, "select id from public.change where id = $1", p.sql)
			case "project":
				got, err = r.Project(p.ctx, domain.ChangeIDRequest{ID: 7})
				require.Equal(t, "select project_id from public.change where id = $1", p.sql)
			case "epic":
				got, err = r.EpicProject(p.ctx, domain.EpicIDRequest{ID: 7})
				want = apperror.ErrChangeInvalidReference
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
				require.ErrorIs(t, err, apperror.ErrChangeDuplicateUUID)
			} else {
				require.ErrorIs(t, err, apperror.ErrProjectNotFound)
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
		{"phase", "call public.sp_change_phase_update($1,$2)", []any{7, "todo"}, func(r *Repo, c context.Context) error {
			return r.UpdatePhase(c, domain.ChangeUpdatePhaseRequest{ID: 7, ChangePhase: "todo"})
		}, false, nil},
		{"epic", "call public.sp_change_epic_update($1,$2)", []any{7, &epic}, func(r *Repo, c context.Context) error {
			return r.UpdateEpic(c, domain.ChangeUpdateEpicRequest{ID: 7, EpicID: &epic})
		}, false, apperror.ErrChangeInvalidReference},
		{"document", "call public.sp_change_doc_set($1,$2,$3,$4)", []any{7, "spec", "Raw", &flag}, func(r *Repo, c context.Context) error {
			return r.SetDocument(c, domain.ChangeDocumentSetRequest{ID: 7, DocType: "spec", Body: "Raw", AgentEdit: &flag})
		}, false, nil},
		{"open", "update public.change set open = $2, modified = now() where id = $1", []any{7, &flag}, func(r *Repo, c context.Context) error {
			return r.UpdateOpen(c, domain.ChangeUpdateOpenRequest{ID: 7, Open: &flag})
		}, true, nil},
		{"types", "update public.change set change_types = $2, modified = now() where id = $1", []any{7, []string{}}, func(r *Repo, c context.Context) error {
			return r.UpdateChangeTypes(c, domain.ChangeUpdateChangeTypesRequest{ID: 7, ChangeTypes: []string{}})
		}, true, nil},
		{"pr-url", "update public.change set pr_url = $2, modified = now() where id = $1", []any{7, "https://pr"}, func(r *Repo, c context.Context) error {
			return r.UpdatePRUrl(c, domain.ChangeUpdatePRUrlRequest{ID: 7, PRUrl: "https://pr"})
		}, true, nil},
		{"delete", "delete from public.change where id = $1", []any{7}, func(r *Repo, c context.Context) error { return r.Delete(c, domain.ChangeIDRequest{ID: 7}) }, true, apperror.ErrChangeHasTestCases},
	}
	for _, tc := range cases {
		for _, tag := range []string{"UPDATE 0", "UPDATE 1"} {
			for _, cause := range []error{nil, errors.New("exec failed"), apperror.Wrap(&pgconn.PgError{Code: "23503"}, "nested")} {
				t.Run(tc.name+"/"+tag, func(t *testing.T) {
					p := newBoundary(t)
					p.args = tc.args
					p.tag = tag
					p.err = cause
					err := tc.call(&Repo{pool: p}, p.ctx)
					require.Equal(t, tc.sql, p.sql)
					if cause == nil && tc.direct && tag == "UPDATE 0" {
						require.ErrorIs(t, err, apperror.ErrChangeNotFound)
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
