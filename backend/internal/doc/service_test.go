package doc

import (
	"context"
	"errors"
	"mch_api/internal/app"
	"mch_api/internal/domain"
	"mch_api/pkg/markdown"
	"testing"

	"github.com/stretchr/testify/require"
)

type fakeRepo struct {
	calls    []string
	contexts []context.Context
	req      any
	err      error
	fail     string
	docs     []domain.Doc
}

func (r *fakeRepo) record(ctx context.Context, op string, q any) error {
	r.calls = append(r.calls, op)
	r.contexts = append(r.contexts, ctx)
	r.req = q
	if r.fail == "" || r.fail == op {
		return r.err
	}
	return nil
}

func (r *fakeRepo) List(ctx context.Context, q domain.DocListRequest) ([]domain.Doc, error) {
	return r.docs, r.record(ctx, "List", q)
}

func (r *fakeRepo) ListActive(ctx context.Context, q domain.DocListRequest) ([]domain.Doc, error) {
	return r.docs, r.record(ctx, "ListActive", q)
}

func (r *fakeRepo) Details(ctx context.Context, q domain.DocIDRequest) (domain.Doc, error) {
	d := domain.Doc{}
	if len(r.docs) > 0 {
		d = r.docs[0]
	}
	return d, r.record(ctx, "Details", q)
}

func (r *fakeRepo) Project(ctx context.Context, q domain.DocListRequest) (domain.ProjectIDRequest, error) {
	return domain.ProjectIDRequest{ID: 9}, r.record(ctx, "Project", q)
}

func (r *fakeRepo) Insert(ctx context.Context, q domain.DocInsertRequest) (domain.DocIDRequest, error) {
	return domain.DocIDRequest{ID: 8}, r.record(ctx, "Insert", q)
}

type fakeConfig struct {
	err error
	req domain.ProjectIDRequest
}

func (c *fakeConfig) Config(_ context.Context, q domain.ProjectIDRequest) (domain.Config, error) {
	c.req = q
	return domain.Config{ProjectDocs: []string{"prd"}, EpicDocs: []string{"plan"}, ChangeDocs: []string{"spec"}}, c.err
}
func flag(v bool) *bool { return &v }

func TestReadsValidateReferenceAndRenderWithoutParentLookups(t *testing.T) {
	ctx := context.Background()
	raw := "**safe** <script>alert(1)</script> [bad](javascript:alert(1))"
	renderer := NewRenderer(markdown.NewGoldmarkParser(), markdown.NewBluemondaySanitizer())
	for _, table := range []string{"project", "epic", "change"} {
		for _, current := range []bool{false, true} {
			r := &fakeRepo{docs: []domain.Doc{{ID: 8, RefID: 7, RefTable: table, Body: raw}}}
			s := NewService(r, renderer, nil)
			req := domain.DocListRequest{RefID: 7, RefTable: table}
			var docs []domain.Doc
			var err error
			if current {
				docs, err = s.ListActive(ctx, req)
			} else {
				docs, err = s.List(ctx, req)
			}
			require.NoError(t, err)
			require.Equal(t, req, r.req)
			require.Len(t, r.calls, 1)
			require.Equal(t, raw, docs[0].Body)
			require.Contains(t, docs[0].HTML, "<strong>safe</strong>")
			require.NotContains(t, docs[0].HTML, "alert")
			d, err := s.Details(ctx, domain.DocIDRequest{ID: 8})
			require.NoError(t, err)
			require.Equal(t, docs[0], d)
		}
	}
	for _, q := range []domain.DocListRequest{{}, {RefID: -1, RefTable: "project"}, {RefID: 1, RefTable: "user"}, {RefID: 1, RefTable: "change;drop"}} {
		s := &Service{}
		_, err := s.List(ctx, q)
		require.ErrorIs(t, err, app.ErrDocInvalidInput)
		_, err = s.ListActive(ctx, q)
		require.ErrorIs(t, err, app.ErrDocInvalidInput)
	}
	_, err := (&Service{}).Details(ctx, domain.DocIDRequest{})
	require.ErrorIs(t, err, app.ErrDocInvalidInput)
	failure := errors.New("read failure")
	s := NewService(&fakeRepo{err: failure}, renderer, nil)
	_, err = s.List(ctx, domain.DocListRequest{RefID: 7, RefTable: "change"})
	require.ErrorIs(t, err, failure)
	_, err = s.ListActive(ctx, domain.DocListRequest{RefID: 7, RefTable: "change"})
	require.ErrorIs(t, err, failure)
	_, err = s.Details(ctx, domain.DocIDRequest{ID: 8})
	require.ErrorIs(t, err, failure)
	require.Empty(t, (Renderer{}).Render("raw"))
	require.Empty(t, renderer.Render(""))
}

func TestInsertConfiguredKindsAndFailures(t *testing.T) {
	ctx := context.Background()
	failure := errors.New("failed")
	for table, kind := range map[string]string{"project": "prd", "epic": "plan", "change": "spec"} {
		for _, fail := range []string{"", "Project", "Config", "Insert"} {
			r := &fakeRepo{}
			p := &fakeConfig{}
			if fail == "Config" {
				p.err = failure
			} else if fail != "" {
				r.err = failure
				r.fail = fail
			}
			s := NewService(r, Renderer{}, p)
			q := domain.DocInsertRequest{RefID: 7, RefTable: table, DocType: " " + kind + " ", Body: " Raw ", AgentEdit: flag(false)}
			got, err := s.Insert(ctx, q)
			if fail != "" {
				require.ErrorIs(t, err, failure)
				continue
			}
			require.Equal(t, domain.DocIDRequest{ID: 8}, got)
			q.DocType = kind
			q.Body = "Raw"
			require.Equal(t, q, r.req)
			require.Equal(t, domain.ProjectIDRequest{ID: 9}, p.req)
			q.DocType = "unknown"
			_, err = s.Insert(ctx, q)
			require.ErrorIs(t, err, app.ErrDocInvalidReference)
		}
	}
	good := domain.DocInsertRequest{RefID: 7, RefTable: "change", DocType: "spec", Body: "raw", AgentEdit: flag(true)}
	for _, field := range []string{"id", "table", "kind", "body", "agent"} {
		q := good
		switch field {
		case "id":
			q.RefID = 0
		case "table":
			q.RefTable = "user"
		case "kind":
			q.DocType = " "
		case "body":
			q.Body = " "
		case "agent":
			q.AgentEdit = nil
		}
		_, err := (&Service{}).Insert(ctx, q)
		require.ErrorIs(t, err, app.ErrDocInvalidInput)
	}
}

func (r *fakeRepo) CommentList(ctx context.Context, q domain.DocListRequest) ([]domain.Doc, error) {
	return r.docs, r.record(ctx, "CommentList", q)
}

func (r *fakeRepo) CommentInsert(ctx context.Context, q domain.DocCommentInsertRequest) (domain.DocIDRequest, error) {
	return domain.DocIDRequest{ID: 8}, r.record(ctx, "CommentInsert", q)
}

func (r *fakeRepo) CommentUpdate(ctx context.Context, q domain.DocCommentUpdateRequest) error {
	return r.record(ctx, "CommentUpdate", q)
}

func (r *fakeRepo) Delete(ctx context.Context, q domain.DocIDRequest) error {
	return r.record(ctx, "Delete", q)
}
