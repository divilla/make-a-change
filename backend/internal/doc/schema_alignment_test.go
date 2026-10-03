package doc

import (
	"context"
	"errors"
	"mch_api/internal/app"
	"mch_api/internal/domain"
	"mch_api/pkg/markdown"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/require"
)

func textPtr(v string) *string { return &v }

func TestCommentAndDeleteAPIContracts(t *testing.T) {
	for _, op := range []string{"comment-list", "comment-insert", "comment-update", "comment-undelete", "delete"} {
		for _, scenario := range []string{"success", "malformed", "wrong-type", "invalid", "failure", "missing"} {
			t.Run(op+"/"+scenario, func(t *testing.T) {
				r := &fakeRepo{docs: []domain.Doc{}}
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				input := `{"id":8,"ref_id":7,"ref_table":"change","body":"raw","agent_edit":false}`
				status := 200
				switch op {
				case "comment-insert":
					status = 201
				case "comment-update", "comment-undelete", "delete":
					status = 204
				}
				switch scenario {
				case "malformed":
					input = "{"
					status = 400
				case "wrong-type":
					input = `{"id":"wrong","ref_id":"wrong","body":2}`
					status = 400
				case "invalid":
					input = `{}`
					status = 400
				case "failure":
					r.err = errors.New("private failure")
					status = 500
				case "missing":
					r.err = app.ErrDocNotFound
					status = 404
				}
				e := echo.New()
				NewAPI(e, NewService(r, Renderer{}, nil))
				rec := httptest.NewRecorder()
				req := httptest.NewRequest("POST", "/api/v1/doc/"+op, strings.NewReader(input)).WithContext(ctx)
				req.Header.Set("Content-Type", "application/json")
				e.ServeHTTP(rec, req)
				require.Equal(t, status, rec.Code, rec.Body.String())
				if status == 204 {
					require.Empty(t, rec.Body.String())
				}
				if scenario == "success" && op == "comment-insert" {
					require.JSONEq(t, `{"id":8}`, rec.Body.String())
				}
				if scenario == "success" && op == "comment-list" {
					require.JSONEq(t, `[]`, rec.Body.String())
				}
				if status == 400 {
					require.Empty(t, r.calls)
				} else {
					for _, gotCtx := range r.contexts {
						require.Same(t, ctx, gotCtx)
					}
				}
				require.NotContains(t, rec.Body.String(), "private")
			})
		}
	}
}

func TestCommentUndeleteAPIRequiresPositiveID(t *testing.T) {
	for _, body := range []string{`{"id":0}`, `{"id":-1}`, `{"id":null}`, `{"id":[]}`} {
		r := &fakeRepo{}
		e := echo.New()
		NewAPI(e, NewService(r, Renderer{}, nil))
		rec := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/api/v1/doc/comment-undelete", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		e.ServeHTTP(rec, req)
		require.Equal(t, 400, rec.Code, rec.Body.String())
		require.Empty(t, r.calls)
	}
}

func TestCommentUpdateAcceptsExplicitEmptyBody(t *testing.T) {
	r := &fakeRepo{}
	e := echo.New()
	NewAPI(e, NewService(r, Renderer{}, nil))
	for _, tc := range []struct {
		body   string
		status int
	}{
		{`{"id":8,"body":""}`, 204}, {`{"id":8}`, 400}, {`{"id":8,"body":null}`, 400},
	} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/api/v1/doc/comment-update", strings.NewReader(tc.body))
		req.Header.Set("Content-Type", "application/json")
		e.ServeHTTP(rec, req)
		require.Equal(t, tc.status, rec.Code, rec.Body.String())
	}
	require.Equal(t, domain.DocCommentUpdateRequest{ID: 8, Body: textPtr("")}, r.req)
}

func TestDocumentSerializationHasNullableDeletionTimestamp(t *testing.T) {
	deleted := time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC)
	for _, timestamp := range []*time.Time{nil, &deleted} {
		r := &fakeRepo{docs: []domain.Doc{{ID: 8, DeletedAt: timestamp}}}
		e := echo.New()
		NewAPI(e, NewService(r, Renderer{}, nil))
		rec := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/api/v1/doc/details", strings.NewReader(`{"id":8}`))
		req.Header.Set("Content-Type", "application/json")
		e.ServeHTTP(rec, req)
		require.Equal(t, 200, rec.Code)
		if timestamp == nil {
			require.Contains(t, rec.Body.String(), `"deleted_at":null`)
		} else {
			require.Contains(t, rec.Body.String(), `"deleted_at":"2026-10-02T00:00:00Z"`)
		}
		require.Contains(t, rec.Body.String(), `"updated_at":`)
		require.NotContains(t, rec.Body.String(), `"current":`)
	}
}

func TestNewDocServiceValidationAndFailurePropagation(t *testing.T) {
	ctx := context.Background()
	failure := errors.New("persistence failed")
	for _, table := range []string{"project", "epic", "change"} {
		for _, fail := range []string{"", "Project", "CommentInsert"} {
			r := &fakeRepo{fail: fail}
			if fail != "" {
				r.err = failure
			}
			q := domain.DocCommentInsertRequest{RefID: 7, RefTable: table, Body: " Raw ", AgentEdit: flag(false)}
			got, err := NewService(r, Renderer{}, nil).CommentInsert(ctx, q)
			if fail != "" {
				require.ErrorIs(t, err, failure)
			} else {
				require.Equal(t, 8, got.ID)
				q.Body = "Raw"
				require.Equal(t, q, r.req)
				require.Equal(t, []string{"Project", "CommentInsert"}, r.calls)
			}
		}
	}
	for _, q := range []domain.DocCommentInsertRequest{
		{},
		{RefID: 0, RefTable: "project", Body: "raw", AgentEdit: flag(true)},
		{RefID: 1, RefTable: "user", Body: "raw", AgentEdit: flag(true)},
		{RefID: 1, RefTable: "project", Body: " \t ", AgentEdit: flag(true)},
		{RefID: 1, RefTable: "project", Body: "raw"},
	} {
		_, err := (&Service{}).CommentInsert(ctx, q)
		require.ErrorIs(t, err, app.ErrDocInvalidInput)
	}
	for _, q := range []domain.DocListRequest{{}, {RefID: -1, RefTable: "project"}, {RefID: 1, RefTable: "user"}} {
		_, err := (&Service{}).CommentList(ctx, q)
		require.ErrorIs(t, err, app.ErrDocInvalidInput)
	}
	for _, q := range []domain.DocCommentUpdateRequest{{}, {ID: 0, Body: textPtr("raw")}, {ID: 1}} {
		require.ErrorIs(t, (&Service{}).CommentUpdate(ctx, q), app.ErrDocInvalidInput)
	}
	require.ErrorIs(t, (&Service{}).Delete(ctx, domain.DocIDRequest{}), app.ErrDocInvalidInput)
	for _, cause := range []error{nil, failure, app.ErrDocNotFound} {
		r := &fakeRepo{err: cause}
		s := NewService(r, Renderer{}, nil)
		require.ErrorIs(t, s.CommentUpdate(ctx, domain.DocCommentUpdateRequest{ID: 8, Body: textPtr("")}), cause)
		require.ErrorIs(t, s.Delete(ctx, domain.DocIDRequest{ID: 8}), cause)
		_, err := s.CommentList(ctx, domain.DocListRequest{RefID: 7, RefTable: "epic"})
		require.ErrorIs(t, err, cause)
	}
}

func TestCommentsRenderRetainedHistory(t *testing.T) {
	deleted := time.Now()
	r := &fakeRepo{docs: []domain.Doc{{ID: 8, DocType: "comment", Body: "**safe** <script>bad()</script>", DeletedAt: &deleted}}}
	renderer := NewRenderer(markdown.NewGoldmarkParser(), markdown.NewBluemondaySanitizer())
	docs, err := NewService(r, renderer, nil).CommentList(context.Background(), domain.DocListRequest{RefID: 7, RefTable: "project"})
	require.NoError(t, err)
	require.Contains(t, docs[0].HTML, "<strong>safe</strong>")
	require.NotContains(t, docs[0].HTML, "bad()")
	require.Equal(t, &deleted, docs[0].DeletedAt)
	require.Equal(t, []string{"CommentList"}, r.calls)
}

func TestNewDocRepositoryMutations(t *testing.T) {
	failure := errors.New("write failed")
	for _, op := range []string{"CommentInsert", "CommentUpdate", "Delete"} {
		scenarios := []string{"success", "error", "no-row"}
		if op != "CommentInsert" {
			scenarios = append(scenarios, "missing")
		}
		for _, scenario := range scenarios {
			p := newBoundary(t)
			var err error
			var cause error
			if scenario == "error" {
				cause = failure
			}
			if scenario == "no-row" {
				cause = pgx.ErrNoRows
			}
			id := 8
			value := &id
			if scenario == "missing" {
				value = nil
			}
			row := valueRow{t: t, values: []any{value}, err: cause}
			q := domain.DocCommentUpdateRequest{ID: 8, Body: textPtr("raw")}
			switch op {
			case "CommentInsert":
				p.args = []any{7, "change", "raw", flag(false)}
				row.values = []any{8}
				p.row = row
				got, e := (&Repo{pool: p}).CommentInsert(p.ctx, domain.DocCommentInsertRequest{RefID: 7, RefTable: "change", Body: "raw", AgentEdit: flag(false)})
				err = e
				if cause == nil {
					require.Equal(t, 8, got.ID)
				}
				require.Equal(t, "select public.fn_doc_comment_insert($1::bigint,$2::text,$3::text,$4::boolean)", p.sql)
			case "CommentUpdate":
				p.args = []any{8, q.Body}
				p.row = row
				err = (&Repo{pool: p}).CommentUpdate(p.ctx, q)
				require.Equal(t, "select public.fn_doc_comment_update($1::bigint,$2::text)", p.sql)
			case "Delete":
				p.args = []any{8}
				p.row = row
				err = (&Repo{pool: p}).Delete(p.ctx, domain.DocIDRequest{ID: 8})
				require.Equal(t, "select public.fn_doc_delete($1::bigint)", p.sql)
			}
			if scenario == "missing" && op != "CommentInsert" || scenario == "no-row" && op != "CommentInsert" {
				require.ErrorIs(t, err, app.ErrDocNotFound)
			} else {
				require.ErrorIs(t, err, cause)
			}
		}
	}
}

func TestRenamedDocRouteIsAbsent(t *testing.T) {
	e := echo.New()
	NewAPI(e, nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest("POST", "/api/v1/doc/current", strings.NewReader(`{}`)))
	require.Equal(t, 404, rec.Code)
}

type commentConfig struct{}

func (commentConfig) Config(context.Context, domain.ProjectIDRequest) (domain.Config, error) {
	return domain.Config{ChangeDocs: []string{"comment"}}, nil
}

func TestRegularInsertRejectsCommentsEvenWhenConfigured(t *testing.T) {
	r := &fakeRepo{}
	_, err := NewService(r, Renderer{}, commentConfig{}).Insert(context.Background(), domain.DocInsertRequest{RefID: 7, RefTable: "change", DocType: "comment", Body: "raw", AgentEdit: flag(false)})
	require.ErrorIs(t, err, app.ErrDocInvalidReference)
	require.Equal(t, []string{"Project"}, r.calls)
}

func TestDocRepositoryReadsNullableAndDeletedTimestamps(t *testing.T) {
	now := time.Now()
	for _, deleted := range []*time.Time{nil, &now} {
		p := newBoundary(t)
		p.args = []any{8}
		p.row = valueRow{t: t, values: []any{8, 7, "change", "comment", "raw", false, now, now, deleted}}
		got, err := (&Repo{pool: p}).Details(p.ctx, domain.DocIDRequest{ID: 8})
		require.NoError(t, err)
		require.Equal(t, deleted, got.DeletedAt)
		require.NotContains(t, p.sql, "deleted_at is null")
	}
}
