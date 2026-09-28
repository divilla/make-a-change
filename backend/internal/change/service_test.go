package change

import (
	"context"
	"errors"
	"mch_api/internal/domain"
	apperror "mch_api/internal/error"
	"mch_api/pkg/markdown"
	"testing"

	"github.com/gofrs/uuid/v5"
	"github.com/stretchr/testify/require"
)

type fakeChangeRepository struct {
	calls                    []string
	requests                 []any
	contexts                 []context.Context
	err                      error
	failAt                   string
	projectID, epicProjectID int
	list                     []domain.ChangeListItem
	details                  domain.ChangeDetails
	docs                     []domain.ChangeDocument
	artifacts                []domain.ChangeArtifactSource
}

func (r *fakeChangeRepository) record(ctx context.Context, op string, req any) error {
	r.calls = append(r.calls, op)
	r.requests = append(r.requests, req)
	r.contexts = append(r.contexts, ctx)
	if r.failAt == "" || r.failAt == op {
		return r.err
	}
	return nil
}

func (r *fakeChangeRepository) List(c context.Context, q domain.ChangeListRequest) ([]domain.ChangeListItem, error) {
	v := r.list
	if v == nil {
		v = []domain.ChangeListItem{}
	}
	return v, r.record(c, "List", q)
}

func (r *fakeChangeRepository) Details(c context.Context, q domain.ChangeIDRequest) (domain.ChangeDetails, error) {
	return r.details, r.record(c, "Details", q)
}

func (r *fakeChangeRepository) Exists(c context.Context, q domain.ChangeIDRequest) error {
	return r.record(c, "Exists", q)
}

func (r *fakeChangeRepository) Project(c context.Context, q domain.ChangeIDRequest) (domain.ProjectIDRequest, error) {
	return domain.ProjectIDRequest{ID: r.projectID}, r.record(c, "Project", q)
}

func (r *fakeChangeRepository) EpicProject(c context.Context, q domain.EpicIDRequest) (domain.ProjectIDRequest, error) {
	return domain.ProjectIDRequest{ID: r.epicProjectID}, r.record(c, "EpicProject", q)
}

func (r *fakeChangeRepository) Documents(c context.Context, q domain.ChangeIDRequest) ([]domain.ChangeDocument, error) {
	v := r.docs
	if v == nil {
		v = []domain.ChangeDocument{}
	}
	return v, r.record(c, "Documents", q)
}

func (r *fakeChangeRepository) Artifacts(c context.Context, q domain.ChangeRenderedArtifactsRequest) ([]domain.ChangeArtifactSource, error) {
	return r.artifacts, r.record(c, "Artifacts", q)
}

func (r *fakeChangeRepository) Create(c context.Context, q domain.ChangeCreateRequest) (domain.ChangeIDRequest, error) {
	return domain.ChangeIDRequest{ID: 2}, r.record(c, "Create", q)
}

func (r *fakeChangeRepository) UpdateTitle(c context.Context, q domain.ChangeUpdateTitleRequest) error {
	return r.record(c, "UpdateTitle", q)
}

func (r *fakeChangeRepository) UpdatePhase(c context.Context, q domain.ChangeUpdatePhaseRequest) error {
	return r.record(c, "UpdatePhase", q)
}

func (r *fakeChangeRepository) UpdateEpic(c context.Context, q domain.ChangeUpdateEpicRequest) error {
	return r.record(c, "UpdateEpic", q)
}

func (r *fakeChangeRepository) UpdateOpen(c context.Context, q domain.ChangeUpdateOpenRequest) error {
	return r.record(c, "UpdateOpen", q)
}

func (r *fakeChangeRepository) UpdateChangeTypes(c context.Context, q domain.ChangeUpdateChangeTypesRequest) error {
	return r.record(c, "UpdateChangeTypes", q)
}

func (r *fakeChangeRepository) UpdatePRUrl(c context.Context, q domain.ChangeUpdatePRUrlRequest) error {
	return r.record(c, "UpdatePRUrl", q)
}

func (r *fakeChangeRepository) SetDocument(c context.Context, q domain.ChangeDocumentSetRequest) error {
	return r.record(c, "SetDocument", q)
}

func (r *fakeChangeRepository) Delete(c context.Context, q domain.ChangeIDRequest) error {
	return r.record(c, "Delete", q)
}

type configFake struct {
	config   domain.Config
	err      error
	ids      []domain.ProjectIDRequest
	contexts []context.Context
}

func (p *configFake) Config(c context.Context, q domain.ProjectIDRequest) (domain.Config, error) {
	p.ids = append(p.ids, q)
	p.contexts = append(p.contexts, c)
	return p.config, p.err
}

func defaultConfig() *configFake {
	return &configFake{config: domain.Config{ChangePhases: []string{"backlog", "review"}, ChangeDocs: []string{"brief", "spec", "pr", "plan"}, ChangeTypes: []string{"fix", "feature"}}}
}
func boolPtr(v bool) *bool { return &v }
func intPtr(v int) *int    { return &v }

func TestServiceCreateIdentityDefaultsAndFailures(t *testing.T) {
	ctx := context.Background()
	supplied := uuid.Must(uuid.NewV4())
	for _, id := range []*uuid.UUID{nil, &supplied} {
		r := &fakeChangeRepository{}
		p := defaultConfig()
		s := NewService(r, Renderer{}, p)
		got, err := s.CreateChange(ctx, domain.ChangeCreateRequest{ProjectID: 9, RefUUID: id, Title: " Title ", Brief: " Brief "})
		require.NoError(t, err)
		require.Equal(t, domain.ChangeIDRequest{ID: 2}, got)
		require.Equal(t, []string{"Create"}, r.calls)
		req := r.requests[0].(domain.ChangeCreateRequest)
		require.Equal(t, "Title", req.Title)
		require.Equal(t, "Brief", req.Brief)
		if id == nil {
			require.Equal(t, byte(7), req.RefUUID.Version())
		} else {
			require.Equal(t, id, req.RefUUID)
		}
		require.Equal(t, []domain.ProjectIDRequest{{ID: 9}}, p.ids)
	}
	for _, c := range []domain.Config{{ChangeDocs: []string{"brief"}}, {ChangePhases: []string{"backlog"}}} {
		r := &fakeChangeRepository{}
		s := NewService(r, Renderer{}, &configFake{config: c})
		_, err := s.CreateChange(ctx, domain.ChangeCreateRequest{ProjectID: 9, Title: "Title", Brief: "Brief"})
		require.ErrorIs(t, err, apperror.ErrChangeInvalidReference)
		require.Empty(t, r.calls)
	}
	failure := errors.New("entropy failure")
	r := &fakeChangeRepository{}
	s := NewService(r, Renderer{}, defaultConfig())
	s.newUUID = func() (uuid.UUID, error) { return uuid.Nil, failure }
	_, err := s.CreateChange(ctx, domain.ChangeCreateRequest{ProjectID: 9, Title: "Title", Brief: "Brief"})
	require.ErrorIs(t, err, failure)
	require.Contains(t, err.Error(), "generate change UUID")
	require.Empty(t, r.calls)
}

func TestServiceDerivesWideCompletion(t *testing.T) {
	for _, tc := range []struct{ done, total, want int64 }{{0, 0, 0}, {1, 2, 50}, {2, 3, 66}, {70000, 100000, 70}} {
		item := domain.ChangeListItem{ID: 7, DoneTC: tc.done, TotalTC: tc.total}
		r := &fakeChangeRepository{list: []domain.ChangeListItem{item}, details: domain.ChangeDetails{ChangeListItem: item}}
		s := NewService(r, Renderer{}, nil)
		list, err := s.ListChanges(context.Background(), domain.ChangeListRequest{ProjectID: 9})
		require.NoError(t, err)
		require.Equal(t, tc.want, list[0].Completed)
		got, err := s.GetChange(context.Background(), domain.ChangeIDRequest{ID: 7})
		require.NoError(t, err)
		require.Equal(t, tc.want, got.Completed)
	}
}

func TestServiceSelectedConfigurationFiltering(t *testing.T) {
	for _, available := range [][]string{{"fix", "feature"}, {"experiment", "maintenance"}} {
		for _, values := range [][]string{nil, {}, {"unknown"}, {" fix ", "experiment", "fix", "", " feature ", "experiment"}} {
			p := &configFake{config: domain.Config{ChangeTypes: available}}
			r := &fakeChangeRepository{projectID: 19}
			s := NewService(r, Renderer{}, p)
			require.NoError(t, s.UpdateChangeTypes(context.Background(), domain.ChangeUpdateChangeTypesRequest{ID: 7, ChangeTypes: values}))
			want := []string{}
			if len(values) > 1 {
				if available[0] == "fix" {
					want = []string{"fix", "feature"}
				} else {
					want = []string{"experiment"}
				}
			}
			require.Equal(t, domain.ChangeUpdateChangeTypesRequest{ID: 7, ChangeTypes: want}, r.requests[1])
			require.Equal(t, []string{"Project", "UpdateChangeTypes"}, r.calls)
			require.Equal(t, []domain.ProjectIDRequest{{ID: 19}}, p.ids)
		}
	}
}

func TestServiceDocumentMappingAndRepeatedWrites(t *testing.T) {
	ctx := context.Background()
	flag := false
	for _, kind := range []string{"brief", "spec", "pr", "plan"} {
		r := &fakeChangeRepository{projectID: 9}
		s := NewService(r, Renderer{}, defaultConfig())
		for range 2 {
			var err error
			switch kind {
			case "brief":
				err = s.UpdateBrief(ctx, domain.ChangeUpdateBriefRequest{ID: 7, Brief: " Raw ", AgentEdit: &flag})
			case "spec":
				err = s.UpdateSpec(ctx, domain.ChangeUpdateSpecRequest{ID: 7, Spec: " Raw ", AgentEdit: &flag})
			case "pr":
				err = s.UpdatePR(ctx, domain.ChangeUpdatePRRequest{ID: 7, PR: " Raw ", AgentEdit: &flag})
			case "plan":
				err = s.SetDocument(ctx, domain.ChangeDocumentSetRequest{ID: 7, DocType: " plan ", Body: " Raw ", AgentEdit: &flag})
			}
			require.NoError(t, err)
		}
		require.Equal(t, []string{"Project", "SetDocument", "Project", "SetDocument"}, r.calls)
		require.Equal(t, domain.ChangeDocumentSetRequest{ID: 7, DocType: kind, Body: "Raw", AgentEdit: &flag}, r.requests[1])
		require.Equal(t, r.requests[1], r.requests[3])
	}
}

func TestServicePreflightAndAssociation(t *testing.T) {
	for _, epic := range []*int{nil, intPtr(4)} {
		r := &fakeChangeRepository{projectID: 9, epicProjectID: 9}
		s := NewService(r, Renderer{}, nil)
		require.NoError(t, s.UpdateEpic(context.Background(), domain.ChangeUpdateEpicRequest{ID: 7, EpicID: epic}))
		want := []string{"Project", "UpdateEpic"}
		if epic != nil {
			want = []string{"Project", "EpicProject", "UpdateEpic"}
		}
		require.Equal(t, want, r.calls)
	}
	r := &fakeChangeRepository{projectID: 9, epicProjectID: 10}
	s := NewService(r, Renderer{}, nil)
	require.ErrorIs(t, s.UpdateEpic(context.Background(), domain.ChangeUpdateEpicRequest{ID: 7, EpicID: intPtr(4)}), apperror.ErrChangeInvalidReference)
	require.Equal(t, []string{"Project", "EpicProject"}, r.calls)
	r = &fakeChangeRepository{}
	s = NewService(r, Renderer{}, nil)
	require.NoError(t, s.UpdateTitle(context.Background(), domain.ChangeUpdateTitleRequest{ID: 7, Title: " A  title "}))
	require.Equal(t, []string{"Exists", "UpdateTitle"}, r.calls)
	require.Equal(t, domain.ChangeUpdateTitleRequest{ID: 7, Title: "A  title"}, r.requests[1])
}

func TestServiceSanitizesExplicitReads(t *testing.T) {
	raw := "**safe** <script>alert(1)</script> [bad](javascript:alert(1))"
	r := &fakeChangeRepository{docs: []domain.ChangeDocument{{ID: 8, DocType: "spec", Body: raw}}, artifacts: []domain.ChangeArtifactSource{{ID: 9}, {ID: 7, Spec: raw, PR: raw}}}
	s := NewService(r, NewRenderer(markdown.NewGoldmarkParser(), markdown.NewBluemondaySanitizer()), nil)
	docs, err := s.Documents(context.Background(), domain.ChangeIDRequest{ID: 7})
	require.NoError(t, err)
	require.Equal(t, raw, docs[0].Body)
	require.Contains(t, docs[0].HTML, "<strong>safe</strong>")
	require.NotContains(t, docs[0].HTML, "script")
	require.NotContains(t, docs[0].HTML, "alert")
	got, err := s.RenderedArtifacts(context.Background(), domain.ChangeRenderedArtifactsRequest{IDs: []int{9, 404, 7, 9}})
	require.NoError(t, err)
	require.Equal(t, domain.ChangeRenderedArtifactsRequest{IDs: []int{9, 404, 7}}, r.requests[2])
	require.Equal(t, []domain.ChangeRenderedArtifact{{ID: 9}, {ID: 7, SpecHTML: docs[0].HTML, PRHtml: docs[0].HTML}}, got.Artifacts)
	require.Equal(t, raw, r.artifacts[1].Spec)
	r = &fakeChangeRepository{}
	s = NewService(r, Renderer{}, nil)
	empty, err := s.RenderedArtifacts(context.Background(), domain.ChangeRenderedArtifactsRequest{})
	require.NoError(t, err)
	require.Equal(t, []domain.ChangeRenderedArtifact{}, empty.Artifacts)
	require.Empty(t, r.calls)
	docs, err = s.Documents(context.Background(), domain.ChangeIDRequest{ID: 7})
	require.NoError(t, err)
	require.Equal(t, []domain.ChangeDocument{}, docs)
	require.Empty(t, (Renderer{}).Render("raw"))
}

func TestServiceRejectsInvalidDirectInput(t *testing.T) {
	ctx := context.Background()
	s := &Service{}
	for _, id := range []int{0, -1} {
		_, err := s.ListChanges(ctx, domain.ChangeListRequest{ProjectID: id})
		require.ErrorIs(t, err, apperror.ErrChangeInvalidInput)
		_, err = s.GetChange(ctx, domain.ChangeIDRequest{ID: id})
		require.ErrorIs(t, err, apperror.ErrChangeInvalidInput)
		_, err = s.Documents(ctx, domain.ChangeIDRequest{ID: id})
		require.ErrorIs(t, err, apperror.ErrChangeInvalidInput)
		_, err = s.RenderedArtifacts(ctx, domain.ChangeRenderedArtifactsRequest{IDs: []int{7, id}})
		require.ErrorIs(t, err, apperror.ErrChangeInvalidInput)
		_, err = s.CreateChange(ctx, domain.ChangeCreateRequest{ProjectID: id, Title: "Title", Brief: "Brief"})
		require.ErrorIs(t, err, apperror.ErrChangeInvalidInput)
		for _, err = range []error{
			s.UpdateTitle(ctx, domain.ChangeUpdateTitleRequest{ID: id, Title: "Title"}),
			s.UpdatePhase(ctx, domain.ChangeUpdatePhaseRequest{ID: id, ChangePhase: "backlog"}),
			s.UpdateEpic(ctx, domain.ChangeUpdateEpicRequest{ID: id}),
			s.UpdateOpen(ctx, domain.ChangeUpdateOpenRequest{ID: id, Open: boolPtr(false)}),
			s.UpdateChangeTypes(ctx, domain.ChangeUpdateChangeTypesRequest{ID: id}),
			s.UpdatePRUrl(ctx, domain.ChangeUpdatePRUrlRequest{ID: id, PRUrl: "https://pr"}),
			s.SetDocument(ctx, domain.ChangeDocumentSetRequest{ID: id, DocType: "spec", Body: "Raw", AgentEdit: boolPtr(true)}),
			s.DeleteChange(ctx, domain.ChangeIDRequest{ID: id}),
		} {
			require.ErrorIs(t, err, apperror.ErrChangeInvalidInput)
		}
	}
	for _, req := range []domain.ChangeCreateRequest{{ProjectID: 9, Title: " ", Brief: "Brief"}, {ProjectID: 9, Title: "Title", Brief: " "}} {
		_, err := s.CreateChange(ctx, req)
		require.ErrorIs(t, err, apperror.ErrChangeInvalidInput)
	}
	for _, err := range []error{
		s.UpdateTitle(ctx, domain.ChangeUpdateTitleRequest{ID: 7, Title: " "}),
		s.UpdatePhase(ctx, domain.ChangeUpdatePhaseRequest{ID: 7, ChangePhase: " "}),
		s.UpdateEpic(ctx, domain.ChangeUpdateEpicRequest{ID: 7, EpicID: intPtr(0)}),
		s.UpdateEpic(ctx, domain.ChangeUpdateEpicRequest{ID: 7, EpicID: intPtr(-1)}),
		s.UpdateOpen(ctx, domain.ChangeUpdateOpenRequest{ID: 7}),
		s.UpdateBrief(ctx, domain.ChangeUpdateBriefRequest{ID: 7, Brief: "Raw"}),
		s.UpdateSpec(ctx, domain.ChangeUpdateSpecRequest{ID: 7, Spec: "Raw"}),
		s.UpdatePR(ctx, domain.ChangeUpdatePRRequest{ID: 7, PR: "Raw"}),
	} {
		require.ErrorIs(t, err, apperror.ErrChangeInvalidInput)
	}
	for _, req := range []domain.ChangeDocumentSetRequest{{ID: 7, DocType: " ", Body: "Raw", AgentEdit: boolPtr(false)}, {ID: 7, DocType: "spec", Body: " ", AgentEdit: boolPtr(false)}, {ID: 7, DocType: "spec", Body: "Raw"}} {
		require.ErrorIs(t, s.SetDocument(ctx, req), apperror.ErrChangeInvalidInput)
	}
	for _, u := range []string{" ", "%", "https:///missing-host", "javascript:alert(1)", "ftp://host", "/relative"} {
		require.ErrorIs(t, s.UpdatePRUrl(ctx, domain.ChangeUpdatePRUrlRequest{ID: 7, PRUrl: u}), apperror.ErrChangeInvalidInput)
	}
	for _, u := range []string{" https://pr ", "http://pr", "HTTP://pr"} {
		r := &fakeChangeRepository{}
		service := NewService(r, Renderer{}, nil)
		require.NoError(t, service.UpdatePRUrl(ctx, domain.ChangeUpdatePRUrlRequest{ID: 7, PRUrl: u}))
		if u == " https://pr " {
			require.Equal(t, domain.ChangeUpdatePRUrlRequest{ID: 7, PRUrl: "https://pr"}, r.requests[0])
		}
	}
}

func TestServiceCollaboratorFailuresAndContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cases := []struct {
		name   string
		call   func(*Service) error
		ops    []string
		config bool
	}{
		{"list", func(s *Service) error {
			_, err := s.ListChanges(ctx, domain.ChangeListRequest{ProjectID: 9})
			return err
		}, []string{"List"}, false},
		{"get", func(s *Service) error { _, err := s.GetChange(ctx, domain.ChangeIDRequest{ID: 7}); return err }, []string{"Details"}, false},
		{"artifacts", func(s *Service) error {
			_, err := s.RenderedArtifacts(ctx, domain.ChangeRenderedArtifactsRequest{IDs: []int{7}})
			return err
		}, []string{"Artifacts"}, false},
		{"documents", func(s *Service) error { _, err := s.Documents(ctx, domain.ChangeIDRequest{ID: 7}); return err }, []string{"Exists", "Documents"}, false},
		{"create", func(s *Service) error {
			_, err := s.CreateChange(ctx, domain.ChangeCreateRequest{ProjectID: 9, Title: "Title", Brief: "Brief"})
			return err
		}, []string{"Create"}, true},
		{"title", func(s *Service) error {
			return s.UpdateTitle(ctx, domain.ChangeUpdateTitleRequest{ID: 7, Title: "Title"})
		}, []string{"Exists", "UpdateTitle"}, false},
		{"epic", func(s *Service) error {
			return s.UpdateEpic(ctx, domain.ChangeUpdateEpicRequest{ID: 7, EpicID: intPtr(4)})
		}, []string{"Project", "EpicProject", "UpdateEpic"}, false},
		{"phase", func(s *Service) error {
			return s.UpdatePhase(ctx, domain.ChangeUpdatePhaseRequest{ID: 7, ChangePhase: " review "})
		}, []string{"Project", "UpdatePhase"}, true},
		{"types", func(s *Service) error { return s.UpdateChangeTypes(ctx, domain.ChangeUpdateChangeTypesRequest{ID: 7}) }, []string{"Project", "UpdateChangeTypes"}, true},
		{"set", func(s *Service) error {
			return s.SetDocument(ctx, domain.ChangeDocumentSetRequest{ID: 7, DocType: "spec", Body: "Raw", AgentEdit: boolPtr(false)})
		}, []string{"Project", "SetDocument"}, true},
		{"open", func(s *Service) error {
			return s.UpdateOpen(ctx, domain.ChangeUpdateOpenRequest{ID: 7, Open: boolPtr(false)})
		}, []string{"UpdateOpen"}, false},
		{"url", func(s *Service) error {
			return s.UpdatePRUrl(ctx, domain.ChangeUpdatePRUrlRequest{ID: 7, PRUrl: "https://pr"})
		}, []string{"UpdatePRUrl"}, false},
		{"delete", func(s *Service) error { return s.DeleteChange(ctx, domain.ChangeIDRequest{ID: 7}) }, []string{"Delete"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			failure := errors.New("collaborator failure")
			for _, step := range append([]string{"success"}, tc.ops...) {
				r := &fakeChangeRepository{projectID: 9, epicProjectID: 9}
				p := defaultConfig()
				p.err = apperror.ErrProjectConfigNotFound
				if tc.config {
					p.err = nil
				}
				if step != "success" {
					r.err = failure
					r.failAt = step
				}
				err := tc.call(NewService(r, Renderer{}, p))
				if step == "success" {
					require.NoError(t, err)
					require.Equal(t, tc.ops, r.calls)
				} else {
					require.ErrorIs(t, err, failure)
					require.Equal(t, step, r.calls[len(r.calls)-1])
				}
				for _, got := range r.contexts {
					require.Same(t, ctx, got)
				}
				if !tc.config {
					require.Empty(t, p.ids)
				}
				for _, got := range p.contexts {
					require.Same(t, ctx, got)
				}
			}
			if tc.config {
				for _, cause := range []error{apperror.ErrProjectConfigNotFound, failure} {
					r := &fakeChangeRepository{projectID: 9}
					p := defaultConfig()
					p.err = cause
					require.ErrorIs(t, tc.call(NewService(r, Renderer{}, p)), cause)
					if tc.name == "create" {
						require.Empty(t, r.calls)
					} else {
						require.Equal(t, []string{"Project"}, r.calls)
					}
				}
			}
		})
	}
}

func TestServiceRejectsUnknownPhaseAndDocumentKind(t *testing.T) {
	for _, config := range []domain.Config{defaultConfig().config, {ChangePhases: []string{"discover"}, ChangeDocs: []string{"analysis"}}} {
		r := &fakeChangeRepository{projectID: 9}
		s := NewService(r, Renderer{}, &configFake{config: config})
		require.ErrorIs(t, s.UpdatePhase(context.Background(), domain.ChangeUpdatePhaseRequest{ID: 7, ChangePhase: "unknown"}), apperror.ErrChangeInvalidReference)
		require.ErrorIs(t, s.SetDocument(context.Background(), domain.ChangeDocumentSetRequest{ID: 7, DocType: "unknown", Body: "Raw", AgentEdit: boolPtr(true)}), apperror.ErrChangeInvalidReference)
		require.Equal(t, []string{"Project", "Project"}, r.calls)
	}
}

func TestP4RenderingAdapterRetainsLegacyContract(t *testing.T) {
	raw := domain.Change{ID: 7, Spec: "**Spec**", PR: "**PR** <script>unsafe()</script>"}
	r := NewRenderer(markdown.NewGoldmarkParser(), markdown.NewBluemondaySanitizer())
	got := r.RenderMutation(domain.TestCaseMutationResponse{Change: raw})
	require.Equal(t, raw.Spec, got.Change.Spec)
	require.Equal(t, raw.PR, got.Change.PR)
	require.Contains(t, got.Change.SpecHTML, "<strong>Spec</strong>")
	require.Contains(t, got.Change.PRHtml, "<strong>PR</strong>")
	require.NotContains(t, got.Change.PRHtml, "script")
	require.NotContains(t, got.Change.PRHtml, "unsafe")
	require.Equal(t, raw, (Renderer{}).RenderChange(raw))
	require.Equal(t, domain.Change{}, r.RenderChange(domain.Change{}))
}

func TestServiceCustomPhaseAndDocuments(t *testing.T) {
	ctx := context.Background()
	p := &configFake{config: domain.Config{ChangePhases: []string{"discover", "ship"}, ChangeDocs: []string{"analysis", "decision"}}}
	r := &fakeChangeRepository{projectID: 19}
	s := NewService(r, Renderer{}, p)
	require.NoError(t, s.UpdatePhase(ctx, domain.ChangeUpdatePhaseRequest{ID: 7, ChangePhase: " ship "}))
	require.Equal(t, domain.ChangeUpdatePhaseRequest{ID: 7, ChangePhase: "ship"}, r.requests[1])
	require.NoError(t, s.SetDocument(ctx, domain.ChangeDocumentSetRequest{ID: 7, DocType: " analysis ", Body: " Custom ", AgentEdit: boolPtr(true)}))
	require.Equal(t, domain.ChangeDocumentSetRequest{ID: 7, DocType: "analysis", Body: "Custom", AgentEdit: boolPtr(true)}, r.requests[3])
	require.ErrorIs(t, s.UpdatePhase(ctx, domain.ChangeUpdatePhaseRequest{ID: 7, ChangePhase: "backlog"}), apperror.ErrChangeInvalidReference)
	require.ErrorIs(t, s.UpdateBrief(ctx, domain.ChangeUpdateBriefRequest{ID: 7, Brief: "Raw", AgentEdit: boolPtr(false)}), apperror.ErrChangeInvalidReference)
	require.Equal(t, []string{"Project", "UpdatePhase", "Project", "SetDocument", "Project", "Project"}, r.calls)
}
