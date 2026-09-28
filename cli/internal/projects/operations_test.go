package projects

import (
	"cli/internal/dto"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type projectAPI struct {
	writes, reads     int
	writeErr, readErr error
	get               func(context.Context, int) (dto.Project, error)
}

func (f *projectAPI) ListProjectRows(context.Context) ([]dto.Project, error) {
	f.reads++
	return []dto.Project{{ID: 7, Name: "Listed"}}, f.readErr
}

func (f *projectAPI) GetProject(ctx context.Context, id int) (dto.Project, error) {
	f.reads++
	if f.get != nil {
		return f.get(ctx, id)
	}
	return dto.Project{ID: id, Name: "Loaded", Config: "custom"}, f.readErr
}

func (f *projectAPI) GetProjectConfig(context.Context, int) (dto.ProjectConfig, error) {
	f.reads++
	return dto.ProjectConfig{Slug: "custom", ProjectDocs: []string{"readme"}, EpicDocs: []string{}, ChangeDocs: []string{"brief", "spec"}, ChangePhases: []string{"todo", "done"}, ChangeColors: []string{"12"}, ChangeTypes: []string{"fix"}}, f.readErr
}

func (f *projectAPI) CreateProject(context.Context, string) (int, error) {
	f.writes++
	return 9, f.writeErr
}

func (f *projectAPI) UpdateProject(context.Context, int, string) error { f.writes++; return f.writeErr }

func (f *projectAPI) DeleteProject(context.Context, int) error { f.writes++; return f.writeErr }

func TestP203ProjectActionsStatesAndAllDisplayedFields(t *testing.T) {
	for _, op := range []Operation{List, Details, Config, Create, Edit, Delete} {
		t.Run(string(op), func(t *testing.T) {
			api := &projectAPI{}
			m, cmd := (Model{}).Begin(context.Background(), api, op, 7, "new")
			require.True(t, m.Loading)
			require.NotNil(t, cmd)
			m, accepted := m.Apply(cmd().(Result))
			require.True(t, accepted)
			require.False(t, m.Loading)
			require.NoError(t, m.Err)
			if op == Config {
				view := ConfigView(m.Catalog)
				for _, s := range []string{"custom", "Project docs: readme", "Epic docs:", "Change docs: brief, spec", "Change phases: todo, done", "Change colors: 12", "Change types: fix"} {
					assert.Contains(t, view, s)
				}
			}
		})
	}
	p := dto.Project{ID: 7, Name: "Project", Config: "custom", LastRef: 42, ChangeCount: 3, CreatedAt: time.Date(2026, 9, 28, 10, 0, 0, 0, time.UTC), UpdatedAt: time.Date(2026, 9, 28, 11, 0, 0, 0, time.UTC)}
	for _, s := range []string{"#ID", "Project", "custom", "42", "Changes", "Created", "Modified"} {
		assert.Contains(t, DetailsView(p, 160), s)
	}
	assert.Contains(t, ListCommands(), "/new-project")
	for _, s := range []string{"/edit", "/delete", "/project-config", "/retry"} {
		assert.Contains(t, DetailCommands(), s)
	}
	assert.Contains(t, TableView(Model{}, 80), "No projects")
	assert.Contains(t, TableView(Model{Loading: true}, 80), "loading")
}

func TestP204MutationSuccessFailurePartialSuccessAndReadOnlyRetry(t *testing.T) {
	failure := errors.New("unavailable")
	for _, op := range []Operation{Create, Edit, Delete} {
		for _, mode := range []string{"success", "write failure", "refresh failure"} {
			t.Run(string(op)+mode, func(t *testing.T) {
				api := &projectAPI{}
				if mode == "write failure" {
					api.writeErr = failure
				}
				if mode == "refresh failure" {
					api.readErr = failure
				}
				m, cmd := (Model{}).Begin(context.Background(), api, op, 7, "draft")
				// Double Enter cannot schedule a duplicate mutation.
				held, duplicate := m.Begin(context.Background(), api, op, 7, "draft")
				require.Nil(t, duplicate)
				assert.Equal(t, m.Generation, held.Generation)
				r := cmd().(Result)
				m, ok := m.Apply(r)
				require.True(t, ok)
				assert.Equal(t, 1, api.writes)
				if mode == "write failure" {
					assert.ErrorIs(t, m.Err, failure)
					assert.Zero(t, api.reads)
					if op != Delete {
						assert.Equal(t, "draft", m.Draft)
					}
					return
				}
				assert.True(t, r.Committed)
				if op == Create {
					assert.Equal(t, 9, m.Detail.ID)
				}
				if mode == "refresh failure" {
					assert.ErrorIs(t, m.Err, failure)
					assert.Contains(t, m.Status, "refresh failed")
					if op == Delete {
						assert.Contains(t, m.Status, "deleted")
					} else {
						assert.Contains(t, m.Status, "saved")
						assert.Equal(t, "draft", m.Detail.Name)
					}
					api.readErr = nil
					retry := Details
					if op == Delete {
						retry = List
					}
					m, cmd = m.Begin(context.Background(), api, retry, m.Detail.ID, "")
					require.NotNil(t, cmd)
					m, ok = m.Apply(cmd().(Result))
					require.True(t, ok)
					require.NoError(t, m.Err)
					assert.Equal(t, 1, api.writes)
				}
			})
		}
	}
	for _, op := range []Operation{Create, Edit} {
		m, cmd := (Model{}).Begin(context.Background(), &projectAPI{}, op, 7, "  \n")
		assert.Nil(t, cmd)
		assert.Equal(t, "  \n", m.Draft)
		assert.Equal(t, "validation failed", m.Status)
	}
	for _, op := range []Operation{Edit, Delete, Details, Config} {
		m, cmd := (Model{}).Begin(context.Background(), &projectAPI{}, op, 0, "draft")
		assert.Nil(t, cmd)
		assert.ErrorContains(t, m.Err, "positive number")
	}
}

func TestP205OrderedProjectCatalogsWithoutFallback(t *testing.T) {
	cfg := dto.ProjectConfig{ChangePhases: []string{"done", "todo"}, ChangeColors: []string{"12"}, ChangeTypes: []string{"fix", "feature"}}
	phases, types := CatalogOptions(cfg)
	assert.Equal(t, []dto.Option{{ID: "done", Label: "done", Color: "12"}, {ID: "todo", Label: "todo"}}, phases)
	assert.Equal(t, []dto.Option{{ID: "fix", Label: "fix"}, {ID: "feature", Label: "feature"}}, types)
	assert.Equal(t, []dto.Option{{ID: "7", Label: "Seven"}}, Options([]dto.Project{{ID: 7, Name: "Seven"}}))
	m := Model{Catalog: cfg}
	api := &projectAPI{readErr: errors.New("configuration missing")}
	m, cmd := m.Begin(context.Background(), api, Config, 8, "")
	assert.Empty(t, m.Catalog.ChangePhases)
	m, _ = m.Apply(cmd().(Result))
	assert.ErrorContains(t, m.Err, "configuration missing")
	assert.Empty(t, m.Catalog.ChangePhases)
}

func TestP206DelayedResultsCannotOverwriteNewOperation(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	canceled := make(chan struct{})
	api := &projectAPI{get: func(ctx context.Context, id int) (dto.Project, error) {
		close(started)
		<-release
		if ctx.Err() != nil {
			close(canceled)
		}
		return dto.Project{ID: id, Name: "old"}, nil
	}}
	m, cmd := (Model{}).Begin(context.Background(), api, Details, 7, "")
	result := make(chan Result, 1)
	go func() { result <- cmd().(Result) }()
	<-started
	m, next := m.Begin(context.Background(), &projectAPI{}, Details, 8, "")
	m, _ = m.Apply(next().(Result))
	close(release)
	old := <-result
	m, accepted := m.Apply(old)
	assert.False(t, accepted)
	assert.Equal(t, 8, m.Detail.ID)
	select {
	case <-canceled:
	case <-time.After(time.Second):
		t.Fatal("obsolete read was not canceled")
	}
	for _, op := range []Operation{List, Config, Create, Edit, Delete} {
		stale := Result{Generation: m.Generation - 1, Operation: op, ID: 8, Project: dto.Project{ID: 9}, Rows: []dto.Project{{ID: 9}}, Config: dto.ProjectConfig{Slug: "stale"}, Committed: true}
		got, ok := m.Apply(stale)
		assert.False(t, ok)
		assert.Equal(t, m.Detail, got.Detail)
		assert.Equal(t, m.Rows, got.Rows)
		assert.Equal(t, m.Catalog, got.Catalog)
	}
	for _, r := range []Result{{Generation: m.Generation, Operation: List, ID: 8}, {Generation: m.Generation, Operation: Details, ID: 7}} {
		_, ok := m.Apply(r)
		assert.False(t, ok)
	}
}

func TestP207ProjectReadFailuresAndEmptyResults(t *testing.T) {
	for _, op := range []Operation{List, Details, Config} {
		failure := errors.New("read failed")
		m, cmd := (Model{}).Begin(context.Background(), &projectAPI{readErr: failure}, op, 7, "")
		m, ok := m.Apply(cmd().(Result))
		require.True(t, ok)
		assert.ErrorIs(t, m.Err, failure)
		assert.Equal(t, "load failed", m.Status)
	}
	m, cmd := (Model{}).Begin(context.Background(), &projectAPI{}, List, 0, "")
	r := cmd().(Result)
	r.Rows = nil
	m, _ = m.Apply(r)
	assert.Equal(t, "no projects", m.Status)
}
