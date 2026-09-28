package epic

import (
	"context"
	"errors"
	"mch_api/internal/domain"
	apperror "mch_api/internal/error"
	"testing"

	"github.com/stretchr/testify/require"
)

type fakeEpicRepository struct {
	err   error
	calls []string
	ctx   context.Context
	req   any
	item  domain.Epic
}

func (r *fakeEpicRepository) record(ctx context.Context, op string, req any) {
	r.calls = append(r.calls, op)
	r.ctx = ctx
	r.req = req
}

func (r *fakeEpicRepository) List(ctx context.Context, req domain.EpicListRequest) ([]domain.Epic, error) {
	r.record(ctx, "list", req)
	return []domain.Epic{r.item}, r.err
}

func (r *fakeEpicRepository) Get(ctx context.Context, req domain.EpicIDRequest) (domain.Epic, error) {
	r.record(ctx, "get", req)
	return r.item, r.err
}

func (r *fakeEpicRepository) Create(ctx context.Context, req domain.EpicCreateRequest) (domain.EpicIDRequest, error) {
	r.record(ctx, "create", req)
	return domain.EpicIDRequest{ID: 7}, r.err
}

func (r *fakeEpicRepository) Update(ctx context.Context, req domain.EpicUpdateRequest) error {
	r.record(ctx, "update", req)
	return r.err
}

func (r *fakeEpicRepository) Delete(ctx context.Context, req domain.EpicIDRequest) error {
	r.record(ctx, "delete", req)
	return r.err
}

func TestServiceRequestsAndErrors(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	for _, cause := range []error{nil, errors.New("repository failure")} {
		for _, op := range []string{"list", "get", "create", "update", "delete"} {
			t.Run(op, func(t *testing.T) {
				r := &fakeEpicRepository{err: cause}
				s := NewService(r)
				var err error
				var want any
				switch op {
				case "list":
					_, err = s.ListEpics(ctx, domain.EpicListRequest{ProjectID: 7})
					want = domain.EpicListRequest{ProjectID: 7}
				case "get":
					want = domain.EpicIDRequest{ID: 7}
					_, err = s.GetEpic(ctx, want.(domain.EpicIDRequest))
				case "create":
					want = domain.EpicCreateRequest{ProjectID: 7, Name: "Name"}
					var id domain.EpicIDRequest
					id, err = s.CreateEpic(ctx, domain.EpicCreateRequest{ProjectID: 7, Name: " Name "})
					require.Equal(t, 7, id.ID)
				case "update":
					want = domain.EpicUpdateRequest{ID: 7, Name: "Name"}
					err = s.UpdateEpic(ctx, domain.EpicUpdateRequest{ID: 7, Name: " Name "})
				case "delete":
					want = domain.EpicIDRequest{ID: 7}
					err = s.DeleteEpic(ctx, want.(domain.EpicIDRequest))

				}
				require.ErrorIs(t, err, cause)
				require.Equal(t, []string{op}, r.calls)
				require.Same(t, ctx, r.ctx)
				require.Equal(t, want, r.req)
			})
		}
	}
}

func TestServiceRejectsInvalidEpicInput(t *testing.T) {
	s := NewService(nil)
	ctx := context.Background()
	for _, id := range []int{0, -1} {
		_, err := s.GetEpic(ctx, domain.EpicIDRequest{ID: id})
		require.ErrorIs(t, err, apperror.ErrEpicInvalidInput)
		require.ErrorIs(t, s.DeleteEpic(ctx, domain.EpicIDRequest{ID: id}), apperror.ErrEpicInvalidInput)
		require.ErrorIs(t, s.UpdateEpic(ctx, domain.EpicUpdateRequest{ID: id, Name: "Valid"}), apperror.ErrEpicInvalidInput)
		_, err = s.ListEpics(ctx, domain.EpicListRequest{ProjectID: id})
		require.ErrorIs(t, err, apperror.ErrEpicInvalidInput)
		_, err = s.CreateEpic(ctx, domain.EpicCreateRequest{ProjectID: id, Name: "Valid"})
		require.ErrorIs(t, err, apperror.ErrEpicInvalidInput)
	}
	for _, name := range []string{"", " ", "\t\n"} {
		_, err := s.CreateEpic(ctx, domain.EpicCreateRequest{ProjectID: 7, Name: name})
		require.ErrorIs(t, err, apperror.ErrEpicInvalidInput)
		require.ErrorIs(t, s.UpdateEpic(ctx, domain.EpicUpdateRequest{ID: 7, Name: name}), apperror.ErrEpicInvalidInput)
	}
}

func TestServiceDerivesCompletion(t *testing.T) {
	for _, tc := range []struct{ done, total, want int64 }{{0, 0, 0}, {1, 2, 50}, {2, 3, 66}, {70000, 100000, 70}} {
		r := &fakeEpicRepository{item: domain.Epic{DoneTC: tc.done, TotalTC: tc.total, Completed: 99}}
		s := NewService(r)
		item, err := s.GetEpic(context.Background(), domain.EpicIDRequest{ID: 7})
		require.NoError(t, err)
		require.Equal(t, tc.want, item.Completed)
		items, err := s.ListEpics(context.Background(), domain.EpicListRequest{ProjectID: 7})
		require.NoError(t, err)
		require.Equal(t, tc.want, items[0].Completed)
	}
}
