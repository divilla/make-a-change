package project

import (
	"context"
	"mch_api/internal/domain"
	apperror "mch_api/internal/error"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServiceRejectsInvalidProjectInput(t *testing.T) {
	service := &Service{}
	_, err := service.GetProject(context.Background(), domain.ProjectIDRequest{})
	require.ErrorIs(t, err, apperror.ErrProjectInvalidInput)
	_, err = service.CreateProject(context.Background(), domain.ProjectCreateRequest{Name: "   "})
	require.ErrorIs(t, err, apperror.ErrProjectInvalidInput)
	_, err = service.UpdateProject(context.Background(), domain.ProjectUpdateRequest{ID: 1, Name: "   "})
	require.ErrorIs(t, err, apperror.ErrProjectInvalidInput)
	err = service.DeleteProject(context.Background(), domain.ProjectIDRequest{})
	require.ErrorIs(t, err, apperror.ErrProjectInvalidInput)
}

func TestServiceNormalizesProjectRequests(t *testing.T) {
	repo := &fakeProjectRepository{}
	service := NewService(repo)
	_, err := service.ListProjects(context.Background())
	require.NoError(t, err)
	assert.True(t, repo.listed)
	_, err = service.GetProject(context.Background(), domain.ProjectIDRequest{ID: 1})
	require.NoError(t, err)
	_, err = service.CreateProject(context.Background(), domain.ProjectCreateRequest{Name: " Project Name "})
	require.NoError(t, err)
	assert.Equal(t, "Project Name", repo.name)
	_, err = service.UpdateProject(context.Background(), domain.ProjectUpdateRequest{ID: 1, Name: " Updated Name "})
	require.NoError(t, err)
	assert.Equal(t, "Updated Name", repo.name)
	err = service.DeleteProject(context.Background(), domain.ProjectIDRequest{ID: 1})
	require.NoError(t, err)
}

type fakeProjectRepository struct {
	err    error
	id     int
	name   string
	listed bool
}

func (r *fakeProjectRepository) List(_ context.Context) ([]domain.Project, error) {
	r.listed = true
	return []domain.Project{}, r.err
}

func (r *fakeProjectRepository) Get(_ context.Context, id int) (domain.Project, error) {
	r.id = id
	return domain.Project{ID: id, Name: "Project"}, r.err
}

func (r *fakeProjectRepository) Create(_ context.Context, name string) (domain.Project, error) {
	r.name = name
	return domain.Project{ID: 1, Name: name}, r.err
}

func (r *fakeProjectRepository) Update(_ context.Context, id int, name string) (domain.Project, error) {
	r.id, r.name = id, name
	return domain.Project{ID: id, Name: name}, r.err
}
func (r *fakeProjectRepository) Delete(_ context.Context, id int) error { r.id = id; return r.err }
