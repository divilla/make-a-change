package app

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppConfigLoadsRepositoryMCHConfigFlowAndHelp(t *testing.T) {
	root := t.TempDir()
	writeMCHFixture(t, root, "backend_url: http://backend.test\n"+"temp_dir: /workspace/custom-mch\n"+"project_id: 7\n")

	cfg, err := loadAppConfig(root)

	require.NoError(t, err)
	assert.Equal(t, root, cfg.RepositoryRoot)
	assert.Equal(t, filepath.Join(root, ".mch", "config.yaml"), cfg.ConfigPath)
	assert.Equal(t, "http://backend.test", cfg.BackendURL)
	assert.Equal(t, 7, cfg.ProjectID)
	assert.Equal(t, filepath.Join(root, ".mch", "default"), cfg.FlowDir)
	assert.Equal(t, "default", cfg.Flow.Slug)
	require.Len(t, cfg.Flow.Steps, 3)
	assert.Equal(t, "def-write", cfg.Flow.Steps[0].Slug)
	assert.Equal(t, "edit", cfg.Flow.Steps[0].Type)
	assert.Equal(t, "def-review", cfg.Flow.Steps[1].Slug)
	assert.Equal(t, "make def-review-exec", cfg.Flow.Steps[1].Exec)
	assert.Equal(t, []string{"skip", "prompt", "exec"}, flowOptionSlugs(cfg.FlowHelp.StageModes))
	assert.Equal(t, []string{"queued", "running", "paused", "stopped", "waiting", "completed", "failed"}, flowOptionSlugs(cfg.FlowHelp.TaskStatuses))
	assert.Equal(t, []string{"none", "entry", "prompt", "agent", "exit", "done"}, flowOptionSlugs(cfg.FlowHelp.TaskSteps))
}

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

func TestAppConfigErrorsOnMalformedFlowAndEmptyHelpSlugs(t *testing.T) {
	root := t.TempDir()
	writeMCHFixture(t, root, "backend_url: http://backend.test\n"+"temp_dir: /workspace/custom-mch\n")
	require.NoError(t, os.WriteFile(filepath.Join(root, ".mch", "default", "flow.yaml"), []byte("version: 1\nslug: default\nname: Default\nhelp: help.yaml\nmakefile: Makefile\nsteps: []\n"), 0o644))

	_, err := loadAppConfig(root)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "flow steps are required")

	writeMCHFixture(t, root, "backend_url: http://backend.test\n"+"temp_dir: /workspace/custom-mch\n")
	require.NoError(t, os.WriteFile(filepath.Join(root, ".mch", "default", "help.yaml"), []byte("stage_modes:\n  - slug: ''\n"), 0o644))

	_, err = loadAppConfig(root)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "stage_modes option slug is required")
}

func TestAppConfigErrorsOnDuplicateFlowStepSlugAndMissingType(t *testing.T) {
	root := t.TempDir()
	writeMCHFixture(t, root, "backend_url: http://backend.test\n"+"temp_dir: /workspace/custom-mch\n")
	flow := `version: 1
slug: default
name: Default
help: help.yaml
makefile: Makefile
steps:
  - slug: custom
    type: edit
  - slug: custom
    type: exec
`
	require.NoError(t, os.WriteFile(filepath.Join(root, ".mch", "default", "flow.yaml"), []byte(flow), 0o644))

	_, err := loadAppConfig(root)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicates slug \"custom\"")

	writeMCHFixture(t, root, "backend_url: http://backend.test\n"+"temp_dir: /workspace/custom-mch\n")
	flow = `version: 1
slug: default
name: Default
help: help.yaml
makefile: Makefile
steps:
  - slug: custom
    type:
`
	require.NoError(t, os.WriteFile(filepath.Join(root, ".mch", "default", "flow.yaml"), []byte(flow), 0o644))

	_, err = loadAppConfig(root)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "flow step 1 type is required")
}

func TestAppConfigAllowsCustomAndMissingFlowHelpOptions(t *testing.T) {
	root := t.TempDir()
	writeMCHFixture(t, root, "backend_url: http://backend.test\n"+"temp_dir: /workspace/custom-mch\n")
	require.NoError(t, os.WriteFile(filepath.Join(root, ".mch", "default", "help.yaml"), []byte("stage_modes:\n  - slug: custom-mode\n    help: custom mode\n"), 0o644))

	cfg, err := loadAppConfig(root)

	require.NoError(t, err)
	assert.Equal(t, []string{"custom-mode"}, flowOptionSlugs(cfg.FlowHelp.StageModes))
	assert.Empty(t, cfg.FlowHelp.TaskStatuses)
	assert.Empty(t, cfg.FlowHelp.TaskSteps)
}

func testAppConfig(overrides appConfig) appConfig {
	cfg := appConfig{
		RepositoryRoot: "/repo",
		ConfigPath:     "/repo/.mch/config.yaml",
		BackendURL:     defaultBackendURL,
		FlowDir:        "/repo/.mch/default",
		Flow: flowConfig{
			Version:        1,
			Slug:           "default",
			Name:           "Default Change Automation",
			Description:    "Default test Flow.",
			Help:           "help.yaml",
			Makefile:       "Makefile",
			Steps:          []flowStep{{Slug: "def", Help: "capture def", Type: "prompt", Prompt: "prompts/change-def.md", Entry: "make def-entry", Exec: "make def-exec", Exit: "make def-exit"}},
			UtilityPrompts: map[string]string{"change-def-tmp": "prompts/change-def-tmp.md"},
		},
		FlowHelp: flowHelpConfig{
			Version:      1,
			StageModes:   []flowOption{{Slug: "skip", Help: "skip"}, {Slug: "prompt", Help: "prompt"}, {Slug: "exec", Help: "exec"}},
			TaskStatuses: []flowOption{{Slug: "queued", Help: "queued"}, {Slug: "running", Help: "running"}, {Slug: "paused", Help: "paused"}, {Slug: "stopped", Help: "stopped"}, {Slug: "waiting", Help: "waiting"}, {Slug: "completed", Help: "completed"}, {Slug: "failed", Help: "failed"}},
			TaskSteps:    []flowOption{{Slug: "none", Help: "none"}, {Slug: "entry", Help: "entry"}, {Slug: "prompt", Help: "prompt"}, {Slug: "agent", Help: "agent"}, {Slug: "exit", Help: "exit"}, {Slug: "done", Help: "done"}},
		},
	}
	if overrides.RepositoryRoot != "" {
		cfg.RepositoryRoot = overrides.RepositoryRoot
	}
	if overrides.ConfigPath != "" {
		cfg.ConfigPath = overrides.ConfigPath
	}
	if overrides.BackendURL != "" {
		cfg.BackendURL = overrides.BackendURL
	}
	if overrides.ProjectID != 0 {
		cfg.ProjectID = overrides.ProjectID
	}
	if overrides.FlowDir != "" {
		cfg.FlowDir = overrides.FlowDir
	}
	if overrides.Flow.Slug != "" {
		cfg.Flow = overrides.Flow
	}
	if len(overrides.FlowHelp.StageModes) > 0 {
		cfg.FlowHelp = overrides.FlowHelp
	}
	return cfg
}

func writeMCHFixture(t *testing.T, root string, config string) {
	t.Helper()
	flowDir := filepath.Join(root, ".mch", "default")
	require.NoError(t, os.MkdirAll(filepath.Join(flowDir, "prompts"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".mch", "config.yaml"), []byte(config), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(flowDir, "flow.yaml"), []byte(testFlowYAML()), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(flowDir, "help.yaml"), []byte(testHelpYAML()), 0o644))
}

func testFlowYAML() string {
	return `version: 1
slug: default
name: Default Change Automation
description: Test Flow.
help: help.yaml
makefile: Makefile
steps:
  - slug: def-write
    help: write def
    type: edit
  - slug: def-review
    help: review def
    type: exec
    prompt: prompts/def-review.md
    entry: make def-review-entry
    exec: make def-review-exec
    exit: make def-review-exit
  - slug: def-refine
    help: refine def
    type: prompt
utility_prompts:
  change-def-tmp: prompts/change-def-tmp.md
`
}

func testHelpYAML() string {
	return `version: 1
stage_modes:
  - slug: skip
    help: stage will not execute
  - slug: prompt
    help: stage will run an interactive session
  - slug: exec
    help: stage will run an automated agent
task_statuses:
  - slug: queued
    help: task is waiting to start
  - slug: running
    help: task is actively executing
  - slug: paused
    help: task is temporarily paused
  - slug: stopped
    help: task was manually stopped
  - slug: waiting
    help: task is waiting for input
  - slug: completed
    help: task finished successfully
  - slug: failed
    help: task finished with an error
task_steps:
  - slug: none
    help: task has not started yet
  - slug: entry
    help: entry script is executing
  - slug: prompt
    help: prompt is being prepared or shown
  - slug: agent
    help: automated agent is executing
  - slug: exit
    help: exit script is executing
  - slug: done
    help: task has finished
`
}

func flowOptionSlugs(options []flowOption) []string {
	slugs := make([]string, 0, len(options))
	for _, option := range options {
		slugs = append(slugs, option.Slug)
	}
	return slugs
}
