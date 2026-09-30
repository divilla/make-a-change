package app

import (
	"cli/internal/dto"
	"cli/internal/projects"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestP205SelectionCatalogScopeAndPersistenceFailure(t *testing.T) {
	client := &fakeClient{projects: []dto.Option{{ID: "8", Label: "Eight"}}, phases: []dto.Option{{ID: "eight", Label: "eight"}}, types: []dto.Option{{ID: "fix", Label: "fix"}}}
	m := newModelWithConfig(client, appConfig{ProjectID: 7, ConfigPath: t.TempDir() + "/missing/config.yaml"})
	m.optionCatalog = optionCatalog{loaded: true, phases: []dto.Option{{ID: "seven", Label: "seven"}}}
	m.changesFilters = changesFilters{
		phase: dto.Option{ID: "backlog", Label: "backlog"},
		typ:   dto.Option{ID: "feature", Label: "feature"},
		epic:  dto.Option{ID: "5", Label: "Epic Five"},
		find:  "migration",
	}
	savedFilters := m.changesFilters
	m, cmd := sendCommand(m, "/select-project")
	m = applyCommand(m, cmd)
	m, cmd = sendKey(m, tea.KeyEnter)
	require.Equal(t, "8", m.currentProject.ID)
	assert.Equal(t, savedFilters, m.changesFilters)
	assert.False(t, m.optionCatalog.loaded)
	assert.Empty(t, m.optionCatalog.phases)
	m = applyCommand(m, cmd)
	assert.True(t, m.optionCatalog.loaded)
	assert.Equal(t, client.phases, m.optionCatalog.phases)
	assert.Contains(t, m.err, "selected in memory")
	failure := errors.New("no configuration assigned")
	m = applyMsg(m, optionCatalogLoadedMsg{id: 8, generation: m.catalogGeneration, err: failure})
	assert.False(t, m.optionCatalog.loaded)
	assert.Empty(t, m.optionCatalog.phases)
	assert.Contains(t, m.err, "selected in memory")
	assert.Contains(t, m.err, "no configuration assigned")
	m = applyMsg(m, currentProjectLoadedMsg{id: 8, generation: m.selectionGeneration, err: errors.New("detail unavailable")})
	assert.Contains(t, m.err, "selected in memory")
	assert.Contains(t, m.err, "detail unavailable")
	other := newModelWithConfig(client, appConfig{ProjectID: 8})
	assert.False(t, other.optionCatalog.loaded)
}

func TestP206SelectionIdentityRejectsDelayedCatalogAndDetail(t *testing.T) {
	m := newModelWithConfig(&fakeClient{}, appConfig{ProjectID: 7})
	m.selectionGeneration = 2
	m.catalogGeneration = 2
	m.optionCatalog = optionCatalog{loaded: true, phases: []dto.Option{{ID: "new", Label: "new"}}}
	for _, r := range []optionCatalogLoadedMsg{
		{id: 8, generation: 2, phases: []dto.Option{{ID: "stale"}}},
		{id: 7, generation: 1, phases: []dto.Option{{ID: "stale"}}},
		{id: 7, generation: 1, err: errors.New("stale error")},
	} {
		got := applyMsg(m, r)
		assert.Equal(t, m.optionCatalog, got.optionCatalog)
		assert.Empty(t, got.err)
	}
	got := applyMsg(m, currentProjectLoadedMsg{id: 7, generation: 1, project: dto.Project{ID: 7, Name: "stale"}})
	assert.Equal(t, m.currentProject, got.currentProject)
	// Opening the selector twice must invalidate even same-project list results.
	m, old := sendCommand(m, "/select-project")
	m = applyMsg(m, tea.KeyMsg{Type: tea.KeyEsc})
	m, newer := sendCommand(m, "/select-project")
	m = applyCommand(m, newer)
	got = applyCommand(m, old)
	assert.Equal(t, m.dropdown.options, got.dropdown.options)
}

func TestP205ConfigReadRecoversSelectedProjectSelectors(t *testing.T) {
	for _, command := range []string{"/project-config", "/retry"} {
		t.Run(command, func(t *testing.T) {
			client := &fakeClient{
				phases: []dto.Option{{ID: "review", Label: "review", Color: "blue"}},
				types:  []dto.Option{{ID: "fix", Label: "fix"}},
				err:    errors.New("configuration unavailable"),
			}
			m := newModelWithConfig(client, appConfig{ProjectID: 7})
			m = applyCommand(m, optionCatalogCommand(m.ctx, client, 7, m.catalogGeneration))
			require.False(t, m.optionCatalog.loaded)
			m.state = ProjectDetailsState
			m.projectList.Detail = dto.Project{ID: 7}
			if command == "/retry" {
				var cmd tea.Cmd
				m, cmd = sendCommand(m, "/project-config")
				m = applyCommand(m, cmd)
				require.Error(t, m.projectList.Err)
			}
			client.err = nil
			m, cmd := sendCommand(m, command)
			m = applyCommand(m, cmd)
			require.Equal(t, "loaded project configuration", m.status)
			require.Empty(t, m.err)
			require.True(t, m.optionCatalog.loaded)
			require.NoError(t, m.optionCatalog.err)
			for _, selector := range []struct {
				state State
				want  []dto.Option
			}{
				{SelectPhaseDropDown, client.phases},
				{SelectTypesDropDown, client.types},
			} {
				next, load := m.beginSelector(selector.state)
				got := applyCommand(next.(Model), load)
				assert.Empty(t, got.err)
				assert.Equal(t, selector.want, got.dropdown.options)
			}
		})
	}
}

func TestP206ConfigReadPreservesCatalogOnUnrelatedOrStaleResults(t *testing.T) {
	for _, scenario := range []string{"other project", "stale generation", "failed read"} {
		t.Run(scenario, func(t *testing.T) {
			client := &fakeClient{phases: []dto.Option{{ID: "replacement"}}}
			m := newModelWithConfig(client, appConfig{ProjectID: 7})
			m.optionCatalog = optionCatalog{loaded: true, phases: []dto.Option{{ID: "current"}}}
			want := m.optionCatalog
			id := 7
			if scenario == "other project" {
				id = 8
			}
			if scenario == "failed read" {
				client.err = errors.New("configuration unavailable")
			}
			next, cmd := m.beginProject(projects.Config, id, "")
			m = next.(Model)
			result := cmd()
			if scenario == "stale generation" {
				m.projectList = m.projectList.Invalidate()
			}
			m = applyMsg(m, result)
			assert.Equal(t, want, m.optionCatalog)
		})
	}
}

func TestP206ManualConfigSupersedesPendingSelectionCatalog(t *testing.T) {
	for _, source := range []string{"startup", "selection"} {
		for _, failed := range []bool{false, true} {
			t.Run(source+map[bool]string{false: "/success", true: "/failure"}[failed], func(t *testing.T) {
				client := &fakeClient{projects: []dto.Option{{ID: "7", Label: "Seven"}}, phases: []dto.Option{{ID: "old", Label: "old"}}}
				m := newModelWithConfig(client, appConfig{ProjectID: 7})
				pending := m.Init()
				if source == "selection" {
					var cmd tea.Cmd
					m, cmd = sendCommand(m, "/select-project")
					m = applyCommand(m, cmd)
					m, pending = sendKey(m, tea.KeyEnter)
				}
				var catalog optionCatalogLoadedMsg
				var detail currentProjectLoadedMsg
				for _, cmd := range pending().(tea.BatchMsg) {
					msg := cmd()
					switch result := msg.(type) {
					case optionCatalogLoadedMsg:
						catalog = result
					case currentProjectLoadedMsg:
						detail = result
					default:
						m = applyMsg(m, msg)
					}
				}
				require.Equal(t, 7, catalog.id)
				if failed {
					catalog.err = errors.New("obsolete configuration failure")
				}
				client.phases = []dto.Option{{ID: "new", Label: "new", Color: "blue"}}
				client.types = []dto.Option{{ID: "fix", Label: "fix"}}
				m.state = ProjectDetailsState
				m.projectList.Detail = dto.Project{ID: 7}
				m, cmd := sendCommand(m, "/project-config")
				m = applyCommand(m, cmd)
				require.True(t, m.optionCatalog.loaded)
				want := m.optionCatalog
				m = applyMsg(m, catalog)
				assert.Equal(t, want, m.optionCatalog)
				assert.Empty(t, m.err)
				assert.Equal(t, "loaded project configuration", m.status)
				// Superseding configuration must not discard the independent name read.
				detail.project = dto.Project{ID: 7, Name: "Loaded name"}
				m = applyMsg(m, detail)
				assert.Equal(t, "Loaded name", m.currentProject.Label)
				for _, state := range []State{SelectPhaseDropDown, SelectTypesDropDown} {
					next, load := m.beginSelector(state)
					got := applyCommand(next.(Model), load)
					assert.Empty(t, got.err)
					assert.NotEmpty(t, got.dropdown.options)
				}
			})
		}
	}
}

func TestP206CanceledManualConfigPreservesPendingCatalog(t *testing.T) {
	for _, source := range []string{"startup", "selection"} {
		for _, timing := range []string{"before cancel", "after cancel"} {
			t.Run(source+"/"+timing, func(t *testing.T) {
				client := &fakeClient{
					projects: []dto.Option{{ID: "7", Label: "Seven"}},
					phases:   []dto.Option{{ID: "review", Label: "review", Color: "blue"}},
					types:    []dto.Option{{ID: "fix", Label: "fix"}},
				}
				m := newModelWithConfig(client, appConfig{ProjectID: 7})
				pending := m.Init()
				if source == "selection" {
					var cmd tea.Cmd
					m, cmd = sendCommand(m, "/select-project")
					m = applyCommand(m, cmd)
					m, pending = sendKey(m, tea.KeyEnter)
				}
				m.state = ProjectDetailsState
				m.projectList.Detail = dto.Project{ID: 7}
				m, manual := sendCommand(m, "/project-config")
				require.NotNil(t, manual)
				// Hold a successful manual response until its screen has been left.
				late := manual()
				if timing == "before cancel" {
					m = applyCommand(m, pending)
				}
				m, next := sendCommand(m, "/return")
				m = applyCommand(m, next)
				require.Equal(t, ProjectsListState, m.state)
				m = applyMsg(m, late)
				if timing == "after cancel" {
					m = applyCommand(m, pending)
				}
				require.True(t, m.optionCatalog.loaded)
				assert.False(t, m.projectList.ShowConfig)
				for _, selector := range []struct {
					state State
					want  []dto.Option
				}{
					{SelectPhaseDropDown, client.phases},
					{SelectTypesDropDown, client.types},
				} {
					next, load := m.beginSelector(selector.state)
					got := applyCommand(next.(Model), load)
					assert.Empty(t, got.err)
					assert.Equal(t, selector.want, got.dropdown.options)
				}
			})
		}
	}
}

func TestP203ProjectReloadCannotSelectHiddenCachedRow(t *testing.T) {
	m := newModelWithConfig(&fakeClient{projectRows: []dto.Project{{ID: 8, Name: "Refreshed"}}}, appConfig{ProjectID: 7})
	m.projectList.Rows = []dto.Project{{ID: 6}, {ID: 7, Name: "Cached"}}
	m.projectList.Selected = 1
	m, refresh := sendCommand(m, "/projects")
	require.True(t, m.projectList.Loading)
	assert.Empty(t, m.projectList.Rows)
	assert.Zero(t, m.projectList.Selected)
	m, detail := sendKey(m, tea.KeyEnter)
	assert.Equal(t, ProjectsListState, m.state)
	assert.Nil(t, detail)
	assert.True(t, m.projectList.Loading)
	m = applyCommand(m, refresh)
	require.Equal(t, "loaded projects", m.status)
	m, detail = sendKey(m, tea.KeyEnter)
	assert.Equal(t, ProjectDetailsState, m.state)
	require.NotNil(t, detail)
	assert.Equal(t, 8, m.projectList.Detail.ID)
}

func TestP204DeleteSelectedProjectClearsScopedCatalogAndPersists(t *testing.T) {
	m := newModelWithConfig(&fakeClient{}, appConfig{ProjectID: 7})
	m.state = ProjectDetailsState
	m.projectList.Detail = dto.Project{ID: 7}
	m.optionCatalog.loaded = true
	next, cmd := m.beginProject(projects.Delete, 7, "")
	m = next.(Model)
	result := cmd()
	next, save := m.Update(result)
	m = next.(Model)
	assert.Equal(t, ProjectsListState, m.state)
	assert.Zero(t, m.appConfig.ProjectID)
	assert.Empty(t, m.currentProject.ID)
	assert.False(t, m.optionCatalog.loaded)
	require.NotNil(t, save)
	m = applyCommand(m, save)
	assert.Contains(t, m.status, "deleted project")
}

type deleteProjectClient struct {
	*fakeClient
	deletes int
}

func (f *deleteProjectClient) DeleteProject(context.Context, int) error {
	f.deletes++
	return nil
}

func TestP204DeleteSelectedProjectPreservesOutcomeOnConfigFailure(t *testing.T) {
	for _, refreshFails := range []bool{false, true} {
		for _, queued := range []bool{false, true} {
			t.Run(fmt.Sprintf("refreshFails=%t/queued=%t", refreshFails, queued), func(t *testing.T) {
				root := t.TempDir()
				writeMCHFixture(t, root, "backend_url: http://backend.test\nproject_id: 7\n")
				cfg, err := loadAppConfig(root)
				require.NoError(t, err)
				client := &deleteProjectClient{fakeClient: &fakeClient{}}
				if refreshFails {
					client.err = errors.New("list refresh unavailable")
				}
				m := newModelWithConfig(client, cfg)
				var pending tea.Msg
				if queued {
					var save tea.Cmd
					m, save = m.persistCurrentProject()
					pending = save()
					require.NoError(t, pending.(configSavedMsg).err)
				}
				// A nonempty directory at the destination makes atomic replacement fail,
				// including when tests run with permission to bypass mode bits.
				require.NoError(t, os.Rename(cfg.ConfigPath, cfg.ConfigPath+".original"))
				require.NoError(t, os.Mkdir(cfg.ConfigPath, 0o755))
				require.NoError(t, os.WriteFile(filepath.Join(cfg.ConfigPath, "keep"), []byte("keep"), 0o600))
				next, remove := m.beginProject(projects.Delete, 7, "")
				m = next.(Model)
				next, save := m.Update(remove())
				m = next.(Model)
				if queued {
					require.Nil(t, save)
					// Complete the earlier selection save before the queued clear.
					next, save = m.Update(pending)
					m = next.(Model)
				}
				require.NotNil(t, save)
				msg := save().(configSavedMsg)
				require.Error(t, msg.err)
				m.quitRequested = true
				m = applyMsg(m, msg)
				assert.Zero(t, m.appConfig.ProjectID)
				assert.Empty(t, m.currentProject.ID)
				assert.False(t, m.quitRequested)
				assert.Contains(t, m.status, "deleted project")
				assert.Contains(t, m.status, "config save failed")
				assert.Contains(t, m.err, "project selection cleared in memory; failed to save project_id:")
				assert.Contains(t, m.err, msg.err.Error())
				assert.NotContains(t, m.err, "project selected in memory")
				if refreshFails {
					assert.Contains(t, m.status, "refresh failed — /retry reads only")
					assert.Contains(t, m.err, "list refresh unavailable")
				}
				client.err = nil
				m, retry := sendCommand(m, "/retry")
				m = applyCommand(m, retry)
				assert.Empty(t, m.err)
				assert.Equal(t, 1, client.deletes)
				assert.Equal(t, 2, client.rowListCalls)
				assert.Contains(t, readTestFile(t, cfg.ConfigPath+".original"), "project_id: 7")
				assert.Equal(t, "keep", readTestFile(t, filepath.Join(cfg.ConfigPath, "keep")))
			})
		}
	}
}

func TestP206ShellReadNavigationCancelsObsoleteProjectResult(t *testing.T) {
	m := newModelWithConfig(&fakeClient{gotProject: dto.Project{ID: 7, Name: "stale"}}, appConfig{})
	next, read := m.beginProject(projects.Details, 7, "")
	m = next.(Model)
	next, _ = m.arrive(MainState, "main")
	m = next.(Model)
	m = applyCommand(m, read)
	assert.Equal(t, MainState, m.state)
	assert.Empty(t, m.projectList.Detail.Name)
	assert.Equal(t, "main", m.status)
}

func TestP206ProgramContextReachesProjectCommands(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	client := &contextProjectClient{fakeClient: &fakeClient{}, t: t}
	m := newModelWithConfig(client, appConfig{})
	m.ctx = ctx
	next, cmd := m.beginProject(projects.Details, 7, "")
	m = applyCommand(next.(Model), cmd)
	assert.Contains(t, m.err, "context canceled")
}

type contextProjectClient struct {
	*fakeClient
	t *testing.T
}

func (c *contextProjectClient) GetProject(ctx context.Context, _ int) (dto.Project, error) {
	require.ErrorIs(c.t, ctx.Err(), context.Canceled)
	return dto.Project{}, ctx.Err()
}
