package project

import (
	"context"
	"testing"

	"aipm/internal/dto"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServiceRejectsInvalidProjectInput(t *testing.T) {
	service := &Service{}
	_, err := service.GetProject(context.Background(), dto.ProjectIDRequest{})
	require.ErrorIs(t, err, ErrInvalidInput)
	_, err = service.CreateProject(context.Background(), dto.ProjectCreateRequest{Name: "   "})
	require.ErrorIs(t, err, ErrInvalidInput)
	_, err = service.UpdateProject(context.Background(), dto.ProjectUpdateRequest{ID: 1, Name: "   "})
	require.ErrorIs(t, err, ErrInvalidInput)
	err = service.DeleteProject(context.Background(), dto.ProjectIDRequest{})
	require.ErrorIs(t, err, ErrInvalidInput)
}

func TestServiceNormalizesProjectRequests(t *testing.T) {
	repo := &fakeProjectRepository{}
	service := NewService(repo)
	_, err := service.ListProjects(context.Background())
	require.NoError(t, err)
	assert.True(t, repo.listed)
	_, err = service.GetProject(context.Background(), dto.ProjectIDRequest{ID: 1})
	require.NoError(t, err)
	_, err = service.CreateProject(context.Background(), dto.ProjectCreateRequest{Name: " Project Name "})
	require.NoError(t, err)
	assert.Equal(t, "Project Name", repo.name)
	_, err = service.UpdateProject(context.Background(), dto.ProjectUpdateRequest{ID: 1, Name: " Updated Name "})
	require.NoError(t, err)
	assert.Equal(t, "Updated Name", repo.name)
	err = service.DeleteProject(context.Background(), dto.ProjectIDRequest{ID: 1})
	require.NoError(t, err)
}

type fakeProjectRepository struct {
	id     int
	name   string
	listed bool
}

func (r *fakeProjectRepository) List(_ context.Context) ([]dto.Project, error) {
	r.listed = true
	return []dto.Project{}, nil
}
func (r *fakeProjectRepository) Get(_ context.Context, id int) (dto.Project, error) {
	r.id = id
	return dto.Project{ID: id, Name: "Project"}, nil
}
func (r *fakeProjectRepository) Create(_ context.Context, name string) (dto.Project, error) {
	r.name = name
	return dto.Project{ID: 1, Name: name}, nil
}
func (r *fakeProjectRepository) Update(_ context.Context, id int, name string) (dto.Project, error) {
	r.id, r.name = id, name
	return dto.Project{ID: id, Name: name}, nil
}
func (r *fakeProjectRepository) Delete(_ context.Context, id int) error { r.id = id; return nil }
