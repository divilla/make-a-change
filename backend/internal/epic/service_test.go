package epic

import (
	"context"
	"testing"

	"mch_api/internal/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServiceRejectsInvalidEpicInput(t *testing.T) {
	service := &Service{}
	_, err := service.ListEpics(context.Background(), domain.EpicListRequest{})
	require.ErrorIs(t, err, ErrInvalidInput)
	_, err = service.GetEpic(context.Background(), domain.EpicIDRequest{})
	require.ErrorIs(t, err, ErrInvalidInput)
	_, err = service.CreateEpic(context.Background(), domain.EpicCreateRequest{ProjectID: 1, Name: "   "})
	require.ErrorIs(t, err, ErrInvalidInput)
	_, err = service.UpdateEpic(context.Background(), domain.EpicUpdateRequest{ID: 1, Name: "   "})
	require.ErrorIs(t, err, ErrInvalidInput)
	err = service.DeleteEpic(context.Background(), domain.EpicIDRequest{})
	require.ErrorIs(t, err, ErrInvalidInput)
}

func TestServiceNormalizesEpicRequests(t *testing.T) {
	repo := &fakeEpicRepository{}
	service := NewService(repo)

	_, err := service.ListEpics(context.Background(), domain.EpicListRequest{ProjectID: 2})
	require.NoError(t, err)
	assert.Equal(t, 2, repo.projectID)
	_, err = service.GetEpic(context.Background(), domain.EpicIDRequest{ID: 3})
	require.NoError(t, err)
	assert.Equal(t, 3, repo.id)
	_, err = service.CreateEpic(context.Background(), domain.EpicCreateRequest{ProjectID: 2, Name: " Epic Name "})
	require.NoError(t, err)
	assert.Equal(t, "Epic Name", repo.createReq.Name)
	_, err = service.UpdateEpic(context.Background(), domain.EpicUpdateRequest{ID: 3, Name: " Updated Epic "})
	require.NoError(t, err)
	assert.Equal(t, "Updated Epic", repo.updateReq.Name)
	err = service.DeleteEpic(context.Background(), domain.EpicIDRequest{ID: 3})
	require.NoError(t, err)
	assert.Equal(t, 3, repo.id)
}

type fakeEpicRepository struct {
	projectID int
	id        int
	createReq domain.EpicCreateRequest
	updateReq domain.EpicUpdateRequest
}

func (r *fakeEpicRepository) List(_ context.Context, projectID int) ([]domain.Epic, error) {
	r.projectID = projectID
	return []domain.Epic{}, nil
}

func (r *fakeEpicRepository) Get(_ context.Context, id int) (domain.Epic, error) {
	r.id = id
	return domain.Epic{ID: id}, nil
}

func (r *fakeEpicRepository) Create(_ context.Context, req domain.EpicCreateRequest) (domain.Epic, error) {
	r.createReq = req
	return domain.Epic{ID: 1, ProjectID: req.ProjectID, Name: req.Name}, nil
}

func (r *fakeEpicRepository) Update(_ context.Context, req domain.EpicUpdateRequest) (domain.Epic, error) {
	r.id = req.ID
	r.updateReq = req
	return domain.Epic{ID: req.ID, Name: req.Name}, nil
}

func (r *fakeEpicRepository) Delete(_ context.Context, id int) error {
	r.id = id
	return nil
}
