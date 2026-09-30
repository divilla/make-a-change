package change

import (
	"context"
	"errors"
	"mch_api/internal/app"
	"mch_api/internal/domain"
	"net/url"
	"strconv"
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

func (r *fakeChangeRepository) Create(c context.Context, q domain.ChangeCreateRequest) (domain.ChangeIDRequest, error) {
	return domain.ChangeIDRequest{ID: 2}, r.record(c, "Create", q)
}

func (r *fakeChangeRepository) UpdateTitle(c context.Context, q domain.ChangeUpdateTitleRequest) error {
	return r.record(c, "UpdateTitle", q)
}

func (r *fakeChangeRepository) UpdateSlug(c context.Context, q domain.ChangeUpdateSlugRequest) error {
	return r.record(c, "UpdateSlug", q)
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

func (r *fakeChangeRepository) UpdateTypes(c context.Context, q domain.ChangeUpdateTypesRequest) error {
	return r.record(c, "UpdateTypes", q)
}

func (r *fakeChangeRepository) UpdatePRUrl(c context.Context, q domain.ChangeUpdatePRUrlRequest) error {
	return r.record(c, "UpdatePRUrl", q)
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

func TestUpdateSlugValidatesEditableSuffix(t *testing.T) {
	for _, tc := range []struct {
		id, slug string
		valid    bool
	}{
		{"7", "full-slug_2", true},
		{"7", "Upper", false},
		{"7", "has space", false},
		{"7", "", false},
		{"0", "valid", false},
	} {
		t.Run(tc.slug+"/"+tc.id, func(t *testing.T) {
			r := &fakeChangeRepository{}
			id, _ := strconv.Atoi(tc.id)
			err := NewService(r, defaultConfig()).UpdateSlug(context.Background(), domain.ChangeUpdateSlugRequest{ID: id, Slug: tc.slug})
			if tc.valid {
				require.NoError(t, err)
				require.Equal(t, []string{"UpdateSlug"}, r.calls)
			} else {
				require.ErrorIs(t, err, app.ErrChangeInvalidInput)
				require.NotContains(t, r.calls, "UpdateSlug")
			}
		})
	}
	r := &fakeChangeRepository{}
	require.NoError(t, NewService(r, defaultConfig()).UpdateSlug(context.Background(), domain.ChangeUpdateSlugRequest{ID: 7, Slug: "valid"}))
	require.Equal(t, []string{"UpdateSlug"}, r.calls)
}

func TestServiceCreateIdentityDefaultsAndFailures(t *testing.T) {
	ctx := context.Background()
	supplied := uuid.Must(uuid.NewV4())
	for _, id := range []*uuid.UUID{nil, &supplied} {
		r := &fakeChangeRepository{}
		p := defaultConfig()
		s := NewService(r, p)
		got, err := s.Create(ctx, domain.ChangeCreateRequest{ProjectID: 9, RefUUID: id, Title: " Title ", Brief: " Brief "})
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
		s := NewService(r, &configFake{config: c})
		_, err := s.Create(ctx, domain.ChangeCreateRequest{ProjectID: 9, Title: "Title", Brief: "Brief"})
		require.ErrorIs(t, err, app.ErrChangeInvalidReference)
		require.Empty(t, r.calls)
	}
	failure := errors.New("entropy failure")
	r := &fakeChangeRepository{}
	s := NewService(r, defaultConfig())
	s.newUUID = func() (uuid.UUID, error) { return uuid.Nil, failure }
	_, err := s.Create(ctx, domain.ChangeCreateRequest{ProjectID: 9, Title: "Title", Brief: "Brief"})
	require.ErrorIs(t, err, failure)
	require.Contains(t, err.Error(), "generate change UUID")
	require.Empty(t, r.calls)
}

func TestServiceDerivesWideCompletion(t *testing.T) {
	for _, tc := range []struct{ done, total, want int64 }{{0, 0, 0}, {1, 2, 50}, {2, 3, 66}, {70000, 100000, 70}} {
		item := domain.ChangeListItem{ID: 7, DoneTC: tc.done, TotalTC: tc.total}
		r := &fakeChangeRepository{list: []domain.ChangeListItem{item}, details: domain.ChangeDetails{ChangeListItem: item}}
		s := NewService(r, nil)
		list, err := s.List(context.Background(), domain.ChangeListRequest{ProjectID: 9})
		require.NoError(t, err)
		require.Equal(t, tc.want, list[0].Completed)
		got, err := s.Details(context.Background(), domain.ChangeIDRequest{ID: 7})
		require.NoError(t, err)
		require.Equal(t, tc.want, got.Completed)
	}
}

func TestServiceSelectedConfigurationValidation(t *testing.T) {
	for _, available := range [][]string{{"fix", "feature"}, {"experiment", "maintenance"}} {
		for _, values := range [][]string{nil, {}, {" " + available[0] + " ", available[1], available[0], ""}} {
			p := &configFake{config: domain.Config{ChangeTypes: available}}
			r := &fakeChangeRepository{projectID: 19}
			require.NoError(t, NewService(r, p).UpdateTypes(context.Background(), domain.ChangeUpdateTypesRequest{ID: 7, ChangeTypes: values}))
			want := []string{}
			if len(values) > 0 {
				want = available
			}
			require.Equal(t, domain.ChangeUpdateTypesRequest{ID: 7, ChangeTypes: want}, r.requests[1])
			require.Equal(t, []string{"Project", "UpdateTypes"}, r.calls)
			require.Equal(t, []domain.ProjectIDRequest{{ID: 19}}, p.ids)
		}
		for _, values := range [][]string{{"unknown"}, {available[0], "unknown"}} {
			r := &fakeChangeRepository{projectID: 19}
			p := &configFake{config: domain.Config{ChangeTypes: available}}
			require.ErrorIs(t, NewService(r, p).UpdateTypes(context.Background(), domain.ChangeUpdateTypesRequest{ID: 7, ChangeTypes: values}), app.ErrChangeInvalidReference)
			require.Equal(t, []string{"Project"}, r.calls)
		}
	}
}

func TestServicePreflightAndAssociation(t *testing.T) {
	for _, epic := range []*int{nil, intPtr(4)} {
		r := &fakeChangeRepository{projectID: 9, epicProjectID: 9}
		s := NewService(r, nil)
		require.NoError(t, s.UpdateEpic(context.Background(), domain.ChangeUpdateEpicRequest{ID: 7, EpicID: epic}))
		want := []string{"Project", "UpdateEpic"}
		if epic != nil {
			want = []string{"Project", "EpicProject", "UpdateEpic"}
		}
		require.Equal(t, want, r.calls)
	}
	r := &fakeChangeRepository{projectID: 9, epicProjectID: 10}
	s := NewService(r, nil)
	require.ErrorIs(t, s.UpdateEpic(context.Background(), domain.ChangeUpdateEpicRequest{ID: 7, EpicID: intPtr(4)}), app.ErrChangeInvalidReference)
	require.Equal(t, []string{"Project", "EpicProject"}, r.calls)
	r = &fakeChangeRepository{}
	s = NewService(r, nil)
	require.NoError(t, s.UpdateTitle(context.Background(), domain.ChangeUpdateTitleRequest{ID: 7, Title: " A  title "}))
	require.Equal(t, []string{"Exists", "UpdateTitle"}, r.calls)
	require.Equal(t, domain.ChangeUpdateTitleRequest{ID: 7, Title: "A  title"}, r.requests[1])
}

func TestServiceRejectsInvalidDirectInput(t *testing.T) {
	ctx := context.Background()
	s := &Service{}
	for _, id := range []int{0, -1} {
		_, err := s.List(ctx, domain.ChangeListRequest{ProjectID: id})
		require.ErrorIs(t, err, app.ErrChangeInvalidInput)
		_, err = s.Details(ctx, domain.ChangeIDRequest{ID: id})
		require.ErrorIs(t, err, app.ErrChangeInvalidInput)
		_, err = s.Create(ctx, domain.ChangeCreateRequest{ProjectID: id, Title: "Title", Brief: "Brief"})
		require.ErrorIs(t, err, app.ErrChangeInvalidInput)
		for _, err = range []error{
			s.UpdateTitle(ctx, domain.ChangeUpdateTitleRequest{ID: id, Title: "Title"}),
			s.UpdatePhase(ctx, domain.ChangeUpdatePhaseRequest{ID: id, ChangePhase: "backlog"}),
			s.UpdateEpic(ctx, domain.ChangeUpdateEpicRequest{ID: id}),
			s.UpdateOpen(ctx, domain.ChangeUpdateOpenRequest{ID: id, Open: boolPtr(false)}),
			s.UpdateTypes(ctx, domain.ChangeUpdateTypesRequest{ID: id}),
			s.UpdatePRUrl(ctx, domain.ChangeUpdatePRUrlRequest{ID: id, PRUrl: "https://pr"}),
			s.Delete(ctx, domain.ChangeIDRequest{ID: id}),
		} {
			require.ErrorIs(t, err, app.ErrChangeInvalidInput)
		}
	}
	for _, req := range []domain.ChangeCreateRequest{{ProjectID: 9, Title: " ", Brief: "Brief"}, {ProjectID: 9, Title: "Title", Brief: " "}} {
		_, err := s.Create(ctx, req)
		require.ErrorIs(t, err, app.ErrChangeInvalidInput)
	}
	for _, err := range []error{
		s.UpdateTitle(ctx, domain.ChangeUpdateTitleRequest{ID: 7, Title: " "}),
		s.UpdatePhase(ctx, domain.ChangeUpdatePhaseRequest{ID: 7, ChangePhase: " "}),
		s.UpdateEpic(ctx, domain.ChangeUpdateEpicRequest{ID: 7, EpicID: intPtr(0)}),
		s.UpdateEpic(ctx, domain.ChangeUpdateEpicRequest{ID: 7, EpicID: intPtr(-1)}),
		s.UpdateOpen(ctx, domain.ChangeUpdateOpenRequest{ID: 7}),
	} {
		require.ErrorIs(t, err, app.ErrChangeInvalidInput)
	}
	for _, u := range []string{" ", "%", "https:///missing-host", "javascript:alert(1)", "ftp://host", "/relative"} {
		require.ErrorIs(t, s.UpdatePRUrl(ctx, domain.ChangeUpdatePRUrlRequest{ID: 7, PRUrl: u}), app.ErrChangeInvalidInput)
	}
	for _, u := range []string{" https://pr ", "http://pr", "HTTP://pr"} {
		r := &fakeChangeRepository{}
		service := NewService(r, nil)
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
			_, err := s.List(ctx, domain.ChangeListRequest{ProjectID: 9})
			return err
		}, []string{"List"}, false},
		{"details", func(s *Service) error { _, err := s.Details(ctx, domain.ChangeIDRequest{ID: 7}); return err }, []string{"Details"}, false},
		{"create", func(s *Service) error {
			_, err := s.Create(ctx, domain.ChangeCreateRequest{ProjectID: 9, Title: "Title", Brief: "Brief"})
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
		{"types", func(s *Service) error { return s.UpdateTypes(ctx, domain.ChangeUpdateTypesRequest{ID: 7}) }, []string{"Project", "UpdateTypes"}, true},
		{"open", func(s *Service) error {
			return s.UpdateOpen(ctx, domain.ChangeUpdateOpenRequest{ID: 7, Open: boolPtr(false)})
		}, []string{"UpdateOpen"}, false},
		{"url", func(s *Service) error {
			return s.UpdatePRUrl(ctx, domain.ChangeUpdatePRUrlRequest{ID: 7, PRUrl: "https://pr"})
		}, []string{"UpdatePRUrl"}, false},
		{"delete", func(s *Service) error { return s.Delete(ctx, domain.ChangeIDRequest{ID: 7}) }, []string{"Delete"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			failure := errors.New("collaborator failure")
			for _, step := range append([]string{"success"}, tc.ops...) {
				r := &fakeChangeRepository{projectID: 9, epicProjectID: 9}
				p := defaultConfig()
				p.err = app.ErrProjectConfigNotFound
				if tc.config {
					p.err = nil
				}
				if step != "success" {
					r.err = failure
					r.failAt = step
				}
				err := tc.call(NewService(r, p))
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
				for _, cause := range []error{app.ErrProjectConfigNotFound, failure} {
					r := &fakeChangeRepository{projectID: 9}
					p := defaultConfig()
					p.err = cause
					require.ErrorIs(t, tc.call(NewService(r, p)), cause)
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

func TestServiceRejectsUnknownPhase(t *testing.T) {
	for _, config := range []domain.Config{defaultConfig().config, {ChangePhases: []string{"discover"}, ChangeDocs: []string{"analysis"}}} {
		r := &fakeChangeRepository{projectID: 9}
		s := NewService(r, &configFake{config: config})
		require.ErrorIs(t, s.UpdatePhase(context.Background(), domain.ChangeUpdatePhaseRequest{ID: 7, ChangePhase: "unknown"}), app.ErrChangeInvalidReference)
		require.Equal(t, []string{"Project"}, r.calls)
	}
}

func TestUpdatePRURLValidationCauses(t *testing.T) {
	failure := errors.New("repository failure")
	for _, tc := range []struct {
		name, input, normalized string
		id                      int
		parser, escape          bool
		repoError               error
	}{
		{name: "escape", input: "https://host/%zz", id: 7, parser: true, escape: true},
		{name: "syntax", input: "https://[host", id: 7, parser: true},
		{name: "invalid ID before parser", input: "https://host/%zz", id: 0},
		{name: "negative ID before parser", input: "https://host/%zz", id: -1},
		{name: "empty", input: "", id: 7},
		{name: "blank", input: " \t ", id: 7},
		{name: "missing host", input: "https:///path", id: 7},
		{name: "unsupported scheme", input: "ftp://host", id: 7},
		{name: "relative", input: "/path", id: 7},
		{name: "uppercase", input: "HTTPS://host", normalized: "HTTPS://host", id: 7},
		{name: "trimmed", input: " \t https://host/path ", normalized: "https://host/path", id: 7},
		{name: "userinfo query fragment", input: "http://user:pass@host/path?q=1#fragment", normalized: "http://user:pass@host/path?q=1#fragment", id: 7},
		{name: "repository failure", input: "https://host", normalized: "https://host", id: 7, repoError: failure},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &fakeChangeRepository{err: tc.repoError}
			config := defaultConfig()
			err := NewService(repo, config).UpdatePRUrl(context.Background(), domain.ChangeUpdatePRUrlRequest{ID: tc.id, PRUrl: tc.input})
			require.Empty(t, config.ids)
			if tc.normalized != "" {
				require.Equal(t, []string{"UpdatePRUrl"}, repo.calls)
				require.Equal(t, []any{domain.ChangeUpdatePRUrlRequest{ID: tc.id, PRUrl: tc.normalized}}, repo.requests)
				if tc.repoError != nil {
					require.Same(t, tc.repoError, err)
				} else {
					require.NoError(t, err)
				}
				return
			}
			require.Empty(t, repo.calls)
			require.ErrorIs(t, err, app.ErrChangeInvalidInput)
			var parser *url.Error
			require.Equal(t, tc.parser, errors.As(err, &parser))
			if tc.parser {
				require.Equal(t, tc.input, parser.URL)
				require.Equal(t, "parse", parser.Op)
				var escape url.EscapeError
				require.Equal(t, tc.escape, errors.As(err, &escape))
				if tc.escape {
					require.Equal(t, url.EscapeError("%zz"), escape)
				}
			} else {
				require.Same(t, app.ErrChangeInvalidInput, err)
			}
		})
	}
}

func (r *fakeChangeRepository) UpdateAfterChange(c context.Context, q domain.ChangeUpdateAfterChangeRequest) error {
	return r.record(c, "UpdateAfterChange", q)
}

func TestUpdateAfterChange(t *testing.T) {
	for _, id := range []*int{nil, intPtr(9)} {
		repo := &fakeChangeRepository{}
		req := domain.ChangeUpdateAfterChangeRequest{ID: 7, AfterChangeID: id}
		require.NoError(t, NewService(repo, nil).UpdateAfterChange(context.Background(), req))
		require.Equal(t, []any{req}, repo.requests)
	}
	for _, req := range []domain.ChangeUpdateAfterChangeRequest{{}, {ID: -1}, {ID: 7, AfterChangeID: intPtr(0)}, {ID: 7, AfterChangeID: intPtr(-1)}} {
		require.ErrorIs(t, (&Service{}).UpdateAfterChange(context.Background(), req), app.ErrChangeInvalidInput)
	}
	failure := errors.New("failure")
	require.ErrorIs(t, NewService(&fakeChangeRepository{err: failure}, nil).UpdateAfterChange(context.Background(), domain.ChangeUpdateAfterChangeRequest{ID: 7}), failure)
}
