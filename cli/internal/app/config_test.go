package app

import (
	"cli/internal/dto"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppConfigAllowsMissingAndZeroProjectID(t *testing.T) {
	root := t.TempDir()
	writeMCHFixture(t, root, "backend_url: http://backend.test\n"+"temp_dir: /workspace/custom-mch\n")

	missing, err := loadAppConfig(root)
	require.NoError(t, err)
	assert.Zero(t, missing.ProjectID)

	require.NoError(t, os.WriteFile(filepath.Join(root, ".mch", "config.yaml"), []byte("backend_url: http://backend.test\n"+"temp_dir: /workspace/custom-mch\n"+"project_id: 0\n"), 0o644))
	zero, err := loadAppConfig(root)
	require.NoError(t, err)
	assert.Zero(t, zero.ProjectID)
}

func TestSaveAppConfigPersistsRepositoryProjectIDAndDropsLegacyTempDir(t *testing.T) {
	root := t.TempDir()
	writeMCHFixture(t, root, "backend_url: http://backend.test\n"+"temp_dir: /workspace/custom-mch\n"+"project_id: 0\n")
	path := filepath.Join(root, ".mch", "config.yaml")
	cfg, err := loadAppConfig(root)
	require.NoError(t, err)

	cfg.ProjectID = 11
	require.NoError(t, saveAppConfig(path, cfg))

	loaded, err := loadAppConfig(root)
	require.NoError(t, err)
	assert.Equal(t, "http://backend.test", loaded.BackendURL)
	assert.Equal(t, 11, loaded.ProjectID)
	body, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(body), "backend_url: http://backend.test")
	assert.NotContains(t, string(body), "temp_dir:")
	assert.Contains(t, string(body), "project_id: 11")
}

func TestResolveGitRepositoryRootFindsRootFromNestedDirectory(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, exec.Command("git", "init", root).Run())
	nested := filepath.Join(root, "one", "two")
	require.NoError(t, os.MkdirAll(nested, 0o755))

	previous, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(nested))
	t.Cleanup(func() {
		require.NoError(t, os.Chdir(previous))
	})

	got, err := resolveGitRepositoryRoot(t.Context())

	require.NoError(t, err)
	assert.Equal(t, root, got)
}

func TestAppConfigErrorsWithoutFallbackToLegacyConfig(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "cli", ".config"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "cli", ".config", "config.yaml"), []byte("backend_url: http://legacy.test\nproject_id: 99\n"), 0o644))

	cfg, err := loadAppConfig(root)

	require.Error(t, err)
	assert.Contains(t, err.Error(), ".mch/config.yaml")
	assert.Equal(t, root, cfg.RepositoryRoot)
	assert.Equal(t, filepath.Join(root, ".mch", "config.yaml"), cfg.ConfigPath)
	_, statErr := os.Stat(filepath.Join(root, ".mch", "config.yaml"))
	assert.True(t, os.IsNotExist(statErr))
}

func testAppConfig(overrides appConfig) appConfig {
	if overrides.RepositoryRoot == "" {
		overrides.RepositoryRoot = "/repo"
	}
	if overrides.ConfigPath == "" {
		overrides.ConfigPath = "/repo/.mch/config.yaml"
	}
	if overrides.BackendURL == "" {
		overrides.BackendURL = defaultBackendURL
	}
	return overrides
}

func writeMCHFixture(t *testing.T, root, config string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".mch"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".mch", "config.yaml"), []byte(config), 0o644))
}

func TestStartupConfigIndependentOfAgentAndFlow(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PATH", t.TempDir())
	t.Setenv("BACKEND_URL", "http://ignored.test")
	writeMCHFixture(t, root, "backend_url: http://configured.test\nproject_id: 7\n")
	cfg, err := loadAppConfig(root)
	require.NoError(t, err)
	assert.Equal(t, "http://configured.test", cfg.BackendURL)
	assert.Equal(t, 7, cfg.ProjectID)
	assert.NoDirExists(t, filepath.Join(root, ".mch/default"))
	assert.NoDirExists(t, filepath.Join(root, ".mch/tmp"))
}

func TestConfigRequiredAndMalformedValues(t *testing.T) {
	for _, body := range []string{"", "backend_url: ''\n", "backend_url: [\n", "backend_url: http://test\nproject_id: nope\n"} {
		t.Run(body, func(t *testing.T) {
			root := t.TempDir()
			writeMCHFixture(t, root, body)
			_, err := loadAppConfig(root)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "config.yaml")
		})
	}
	_, err := loadAppConfig("")
	require.ErrorContains(t, err, "repository root is required")
}

func TestAtomicConfigReplacementAndFailureCauses(t *testing.T) {
	root := t.TempDir()
	writeMCHFixture(t, root, "backend_url: http://old\nproject_id: 7\n")
	path := filepath.Join(root, ".mch/config.yaml")
	old, err := os.Open(path)
	require.NoError(t, err)
	defer func() { require.NoError(t, old.Close()) }()
	require.NoError(t, saveAppConfig(path, appConfig{BackendURL: "http://new", ProjectID: 8}))
	oldBody, err := io.ReadAll(old)
	require.NoError(t, err)
	assert.Contains(t, string(oldBody), "http://old")
	current, err := loadAppConfig(root)
	require.NoError(t, err)
	assert.Equal(t, 8, current.ProjectID)
	require.ErrorContains(t, saveAppConfig(path, appConfig{}), "backend_url is required")
	assert.Contains(t, readTestFile(t, path), "http://new")
	err = saveAppConfig(filepath.Join(root, "missing/config.yaml"), current)
	require.ErrorIs(t, err, os.ErrNotExist)
	assert.Contains(t, err.Error(), "save config")
	destination := filepath.Join(root, "directory")
	require.NoError(t, os.Mkdir(destination, 0o755))
	require.Error(t, saveAppConfig(destination, current))
	entries, err := filepath.Glob(filepath.Join(root, ".directory.tmp-*"))
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestAtomicConfigReplacementPreservesPermissions(t *testing.T) {
	for _, mode := range []os.FileMode{0o600, 0o640, 0o644} {
		t.Run(fmt.Sprintf("%o", mode), func(t *testing.T) {
			root := t.TempDir()
			writeMCHFixture(t, root, "backend_url: http://old\nproject_id: 7\n")
			path := filepath.Join(root, defaultConfigPath)
			require.NoError(t, os.Chmod(path, mode))
			require.NoError(t, saveAppConfig(path, appConfig{BackendURL: "http://new", ProjectID: 8}))
			info, err := os.Stat(path)
			require.NoError(t, err)
			assert.Equal(t, mode, info.Mode().Perm())
			cfg, err := loadAppConfig(root)
			require.NoError(t, err)
			assert.Equal(t, 8, cfg.ProjectID)
		})
	}
	t.Run("new file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "config.yaml")
		require.NoError(t, saveAppConfig(path, appConfig{BackendURL: "http://new"}))
		info, err := os.Stat(path)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o644), info.Mode().Perm())
	})
	t.Run("stat failure", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "config.yaml")
		require.NoError(t, os.Symlink(path, path))
		_, cause := os.Stat(path)
		require.Error(t, cause)
		err := saveAppConfig(path, appConfig{BackendURL: "http://new"})
		require.ErrorIs(t, err, cause.(*os.PathError).Err)
		assert.Contains(t, err.Error(), "save config")
		target, err := os.Readlink(path)
		require.NoError(t, err)
		assert.Equal(t, path, target, "failed stat must not replace the existing path")
	})
}

func TestProjectSelectionSaveIsAsynchronousAndKeepsPartialSuccess(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(fmt.Sprint(fail), func(t *testing.T) {
			root := t.TempDir()
			writeMCHFixture(t, root, "backend_url: http://backend.test\nproject_id: 7\n")
			path := filepath.Join(root, ".mch/config.yaml")
			cfg, err := loadAppConfig(root)
			require.NoError(t, err)
			if fail {
				cfg.ConfigPath = filepath.Join(root, "missing/config.yaml")
			}
			m := newModelWithConfig(&fakeClient{projects: []dto.Option{{ID: "8", Label: "Eight"}}}, cfg)
			m, cmd := sendCommand(m, "/select-project")
			m = applyCommand(m, cmd)
			m, save := sendKey(m, tea.KeyEnter)
			require.NotNil(t, save)
			assert.Equal(t, "8", m.currentProject.ID)
			assert.Equal(t, 8, m.appConfig.ProjectID)
			assert.Contains(t, readTestFile(t, path), "project_id: 7", "Update and View must not write config")
			_ = m.View()
			assert.Contains(t, readTestFile(t, path), "project_id: 7")
			msg := save().(configSavedMsg)
			if fail {
				require.ErrorIs(t, msg.err, os.ErrNotExist)
			}
			m = applyMsg(m, msg)
			assert.Equal(t, "8", m.currentProject.ID)
			if fail {
				assert.Contains(t, m.err, "selected in memory")
				assert.Contains(t, readTestFile(t, path), "project_id: 7")
			} else {
				assert.Empty(t, m.err)
				assert.Contains(t, readTestFile(t, path), "project_id: 8")
			}
		})
	}
}

func TestProjectSelectionSerializesOverlappingSaves(t *testing.T) {
	root := t.TempDir()
	writeMCHFixture(t, root, "backend_url: http://test\nproject_id: 7\n")
	cfg, err := loadAppConfig(root)
	require.NoError(t, err)
	m := newModelWithConfig(&fakeClient{}, cfg)
	m.appConfig.ProjectID = 8
	m, first := m.persistCurrentProject()
	m.appConfig.ProjectID = 9
	m, second := m.persistCurrentProject()
	require.Nil(t, second)
	next, last := m.Update(first())
	m = next.(Model)
	require.NotNil(t, last)
	m = applyCommand(m, last)
	assert.False(t, m.configSaveInFlight)
	saved, err := loadAppConfig(root)
	require.NoError(t, err)
	assert.Equal(t, 9, saved.ProjectID)
}

func TestOrdinaryDocumentSaveRetainsCommittedTextAfterFollowUpFailure(t *testing.T) {
	for _, field := range []detailEditField{detailEditDef, detailEditSpec, detailEditPullRequest} {
		t.Run(string(field), func(t *testing.T) {
			cause := errors.New("follow-up failed")
			for _, typesFailure := range []bool{false, true} {
				client := &fakeClient{}
				if typesFailure {
					client.changeTypesUpdateErr = cause
				} else {
					client.changeGetErr = cause
				}
				original := dto.Change{ID: "12", Title: "Existing"}
				text := "# Existing\n\nTypes: feature\n\nSaved text"
				msg := changeDetailTextUpdateCommand(client, ChangeDetailsState, original, field, text)().(changeSavedMsg)
				require.NoError(t, msg.err)
				require.ErrorIs(t, msg.reloadErr, cause)
				m := NewModelWithClient(client)
				m.state = ChangeDetailsState
				m = applyMsg(m, msg)
				switch field {
				case detailEditDef:
					assert.Equal(t, text, m.changeList.Detail.Def)
				case detailEditSpec:
					assert.Equal(t, text, m.changeList.Detail.Spec)
				case detailEditPullRequest:
					assert.Equal(t, text, m.changeList.Detail.PR)
				}
				assert.Contains(t, m.err, "saved")
			}
		})
	}
}

func TestQuitDrainsProjectSelectionSaves(t *testing.T) {
	for _, exit := range []string{"esc", "/quit", "ctrl+c"} {
		for _, queued := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/queued=%t", exit, queued), func(t *testing.T) {
				root := t.TempDir()
				writeMCHFixture(t, root, "backend_url: http://test\nproject_id: 7\n")
				cfg, err := loadAppConfig(root)
				require.NoError(t, err)
				m := newModelWithConfig(&fakeClient{}, cfg)
				m.appConfig.ProjectID = 8
				m, first := m.persistCurrentProject()
				want := 8
				if queued {
					want = 9
					m.appConfig.ProjectID = want
					var cmd tea.Cmd
					m, cmd = m.persistCurrentProject()
					require.Nil(t, cmd)
				}
				var quit tea.Cmd
				switch exit {
				case "esc":
					m, quit = sendKey(m, tea.KeyEsc)
				case "ctrl+c":
					m, quit = sendKey(m, tea.KeyCtrlC)
				default:
					m, quit = sendCommand(m, exit)
				}
				require.Nil(t, quit, "must await the save result before quitting")
				assert.False(t, m.quitting)
				assert.Contains(t, m.View(), "saving project selection before exit")
				// Repeated exit and editor input cannot interrupt the drain.
				m, quit = sendKey(m, tea.KeyEsc)
				require.Nil(t, quit)
				m, quit = sendKey(m, tea.KeyCtrlE)
				require.Nil(t, quit)
				next, cmd := m.Update(first())
				m = next.(Model)
				require.NotNil(t, cmd)
				if queued {
					assert.False(t, m.quitting)
					next, cmd = m.Update(cmd())
					m = next.(Model)
				}
				require.NotNil(t, cmd)
				assert.IsType(t, tea.QuitMsg{}, cmd())
				assert.True(t, m.quitting)
				assert.Equal(t, DoneState, m.state)
				assert.False(t, m.configSaveInFlight)
				saved, err := loadAppConfig(root)
				require.NoError(t, err)
				assert.Equal(t, want, saved.ProjectID)
			})
		}
	}
}

func TestQuitSaveFailureRemainsVisible(t *testing.T) {
	for _, failFirst := range []bool{false, true} {
		t.Run(fmt.Sprintf("first=%t", failFirst), func(t *testing.T) {
			m := NewModelWithClient(&fakeClient{})
			m.appConfig.ProjectID = 9
			m.currentProject = dto.Option{ID: "9", Label: "Nine"}
			m.configSaveInFlight = true
			m.configSavePending = failFirst
			m, quit := sendKey(m, tea.KeyEsc)
			require.Nil(t, quit)
			next, pending := m.Update(configSavedMsg{projectID: 8, err: errors.New("disk full")})
			m = next.(Model)
			assert.Contains(t, m.View(), "failed to save project_id: disk full")
			assert.False(t, m.quitting)
			if failFirst {
				require.NotNil(t, pending, "a failed old save must still drain the latest selection")
				next, pending = m.Update(pending())
				m = next.(Model)
			}
			require.Nil(t, pending, "failure must cancel automatic exit")
			assert.Equal(t, "9", m.currentProject.ID)
			assert.Contains(t, m.View(), "disk full")
			// The user can acknowledge the error with another exit request.
			m, quit = sendKey(m, tea.KeyEsc)
			require.NotNil(t, quit)
			assert.IsType(t, tea.QuitMsg{}, quit())
			assert.True(t, m.quitting)
		})
	}
}
