package project

import (
	"context"
	"errors"
	"mch_api/internal/app"
	"mch_api/internal/domain"
	"testing"

	"github.com/stretchr/testify/require"
)

type fakeProjectRepository struct {
	err    error
	calls  []string
	ctx    context.Context
	req    any
	item   domain.Project
	config domain.Config
}

func (r *fakeProjectRepository) record(ctx context.Context, op string, req any) {
	r.calls = append(r.calls, op)
	r.ctx = ctx
	r.req = req
}

func (r *fakeProjectRepository) List(ctx context.Context) ([]domain.Project, error) {
	r.record(ctx, "list", nil)
	return []domain.Project{r.item}, r.err
}

func (r *fakeProjectRepository) Details(ctx context.Context, req domain.ProjectIDRequest) (domain.Project, error) {
	r.record(ctx, "details", req)
	return r.item, r.err
}

func (r *fakeProjectRepository) Create(ctx context.Context, req domain.ProjectCreateRequest) (domain.ProjectIDRequest, error) {
	r.record(ctx, "create", req)
	return domain.ProjectIDRequest{ID: 7}, r.err
}

func (r *fakeProjectRepository) Update(ctx context.Context, req domain.ProjectUpdateRequest) error {
	r.record(ctx, "update", req)
	return r.err
}

func (r *fakeProjectRepository) Delete(ctx context.Context, req domain.ProjectIDRequest) error {
	r.record(ctx, "delete", req)
	return r.err
}

func (r *fakeProjectRepository) Config(ctx context.Context, req domain.ProjectIDRequest) (domain.Config, error) {
	r.record(ctx, "config", req)
	return r.config, r.err
}

func TestServiceRequestsAndErrors(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	for _, cause := range []error{nil, errors.New("repository failure")} {
		for _, op := range []string{"list", "details", "create", "update", "delete", "config"} {
			t.Run(op, func(t *testing.T) {
				r := &fakeProjectRepository{err: cause}
				s := NewService(r)
				var err error
				var want any
				switch op {
				case "list":
					_, err = s.List(ctx)
					want = nil
				case "details":
					want = domain.ProjectIDRequest{ID: 7}
					_, err = s.Details(ctx, want.(domain.ProjectIDRequest))
				case "create":
					want = domain.ProjectCreateRequest{Name: "Name"}
					var id domain.ProjectIDRequest
					id, err = s.Create(ctx, domain.ProjectCreateRequest{Name: " Name "})
					require.Equal(t, 7, id.ID)
				case "update":
					want = domain.ProjectUpdateRequest{ID: 7, Name: "Name"}
					err = s.UpdateProject(ctx, domain.ProjectUpdateRequest{ID: 7, Name: " Name "})
				case "delete":
					want = domain.ProjectIDRequest{ID: 7}
					err = s.Delete(ctx, want.(domain.ProjectIDRequest))
				case "config":
					want = domain.ProjectIDRequest{ID: 7}
					_, err = s.Config(ctx, want.(domain.ProjectIDRequest))
				}
				require.ErrorIs(t, err, cause)
				require.Equal(t, []string{op}, r.calls)
				require.Same(t, ctx, r.ctx)
				require.Equal(t, want, r.req)
			})
		}
	}
}

func TestServiceRejectsInvalidProjectInput(t *testing.T) {
	s := NewService(nil)
	ctx := context.Background()
	for _, id := range []int{0, -1} {
		_, err := s.Details(ctx, domain.ProjectIDRequest{ID: id})
		require.ErrorIs(t, err, app.ErrProjectInvalidInput)
		require.ErrorIs(t, s.Delete(ctx, domain.ProjectIDRequest{ID: id}), app.ErrProjectInvalidInput)
		require.ErrorIs(t, s.UpdateProject(ctx, domain.ProjectUpdateRequest{ID: id, Name: "Valid"}), app.ErrProjectInvalidInput)
		_, err = s.Config(ctx, domain.ProjectIDRequest{ID: id})
		require.ErrorIs(t, err, app.ErrProjectInvalidInput)
	}
	for _, name := range []string{"", " ", "\t\n"} {
		_, err := s.Create(ctx, domain.ProjectCreateRequest{Name: name})
		require.ErrorIs(t, err, app.ErrProjectInvalidInput)
		require.ErrorIs(t, s.UpdateProject(ctx, domain.ProjectUpdateRequest{ID: 7, Name: name}), app.ErrProjectInvalidInput)
	}
}

func TestServiceConfigNeverSubstitutes(t *testing.T) {
	r := &fakeProjectRepository{err: app.ErrProjectConfigNotFound}
	_, err := NewService(r).Config(context.Background(), domain.ProjectIDRequest{ID: 7})
	require.ErrorIs(t, err, app.ErrProjectConfigNotFound)
	require.Equal(t, []string{"config"}, r.calls)
}
