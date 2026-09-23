package integration_test

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

const testRefUUID = "11111111-1111-1111-1111-111111111111"

type flowConfig struct {
	Steps []struct {
		Slug   string `yaml:"slug"`
		Stage  string `yaml:"stage"`
		Help   string `yaml:"help"`
		Type   string `yaml:"type"`
		Prompt string `yaml:"prompt"`
	} `yaml:"steps"`
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	require.NoError(t, err)
	return strings.TrimSpace(string(out))
}

func TestDefaultFlowPromptAndStageIntegration(t *testing.T) {
	root := repositoryRoot(t)
	flowDir := filepath.Join(root, ".mch", "default")
	body, err := os.ReadFile(filepath.Join(flowDir, "flow.yaml"))
	require.NoError(t, err)
	var flow flowConfig
	require.NoError(t, yaml.Unmarshal(body, &flow))
	expectedSteps := map[string]struct {
		stage    string
		help     string
		stepType string
		prompt   string
	}{
		"def-create":          {stage: "artifact", help: "Create the initial Change definition.", stepType: "editor"},
		"def-edit":            {stage: "artifact", help: "Edit the Change definition.", stepType: "editor"},
		"spec-edit":           {stage: "artifact", help: "Edit the Change specification.", stepType: "editor"},
		"pr-edit":             {stage: "artifact", help: "Edit the pull request description.", stepType: "editor"},
		"def-rewrite":         {stage: "artifact", help: "Draft or revise the Change definition.", stepType: "exec", prompt: "prompts/def-rewrite.md"},
		"def-review":          {stage: "artifact", help: "Review the Change definition for clarity, completeness, and scope.", stepType: "exec", prompt: "prompts/def-review.md"},
		"branch-init":         {stage: "artifact", help: "Initialize and check out the Change branch.", stepType: "script"},
		"init-code-chat":      {stage: "artifact", help: "Discuss the initial implementation context before specification work.", stepType: "chat"},
		"spec-write":          {stage: "artifact", help: "Draft or revise the Change specification.", stepType: "exec", prompt: "prompts/spec-write.md"},
		"spec-write-chat":     {stage: "artifact", help: "Discuss and refine the current Change artifact.", stepType: "chat"},
		"spec-review":         {stage: "spec-review", help: "Review the Change specification for implementation readiness.", stepType: "exec", prompt: "prompts/spec-review.md"},
		"spec-review-chat":    {stage: "spec-review", help: "Resolve specification review findings.", stepType: "chat"},
		"code-implement":      {stage: "code", help: "Implement the approved Change specification.", stepType: "exec", prompt: "prompts/code-implement.md"},
		"code-chat":           {stage: "code", help: "Discuss implementation progress, decisions, and blockers.", stepType: "chat"},
		"pr-write":            {stage: "artifact", help: "Draft the pull request description from the specification and branch diff.", stepType: "exec", prompt: "prompts/pr-write.md"},
		"pr-publish":          {stage: "artifact", help: "Publish the pull request.", stepType: "script"},
		"code-review":         {stage: "code-review", help: "Review the implementation against the Change specification.", stepType: "exec", prompt: "prompts/code-review.md"},
		"code-review-publish": {stage: "code-review", help: "Publish the implementation review findings.", stepType: "script"},
		"code-review-chat":    {stage: "code-review", help: "Resolve implementation review findings.", stepType: "chat"},
		"pr-update":           {stage: "artifact", help: "Update the pull request description to reflect the final implementation.", stepType: "exec", prompt: "prompts/pr-update.md"},
	}
	require.Len(t, flow.Steps, len(expectedSteps))
	for _, step := range flow.Steps {
		expected, ok := expectedSteps[step.Slug]
		require.True(t, ok, "unexpected Flow step %q", step.Slug)
		assert.Equal(t, expected.stage, step.Stage)
		assert.Equal(t, expected.help, step.Help)
		assert.Equal(t, expected.stepType, step.Type)
		assert.Equal(t, expected.prompt, step.Prompt)
		delete(expectedSteps, step.Slug)
		if strings.HasPrefix(step.Prompt, "prompts/") && step.Slug != "pr-update" {
			require.FileExists(t, filepath.Join(flowDir, step.Prompt))
		}
	}
	assert.Empty(t, expectedSteps)

	mappings := map[string]string{
		"def-write":   "artifact",
		"def-review":  "artifact",
		"spec-write":  "artifact",
		"pr-write":    "artifact",
		"spec-review": "spec-review",
	}
	for step, stage := range mappings {
		t.Run(step, func(t *testing.T) {
			cmd := exec.Command("make", "--no-print-directory", "-f", filepath.Join(flowDir, "Makefile"), step+"-exec")
			cmd.Dir = root
			out, err := cmd.CombinedOutput()
			require.NoError(t, err, string(out))
			assert.Contains(t, string(out), fmt.Sprintf("mch demo exec: %s (%s)", step, stage))
		})
	}

	for _, target := range []string{"spec-exec", "ready-exec", "docs-exec", "sync-exec"} {
		cmd := exec.Command("make", "--no-print-directory", "-f", filepath.Join(flowDir, "Makefile"), target)
		cmd.Dir = root
		out, err := cmd.CombinedOutput()
		require.Error(t, err, "%s unexpectedly remained available: %s", target, out)
	}

	assert.NoFileExists(t, filepath.Join(flowDir, "scripts", "show-prompt.sh"))
	for _, stage := range []string{"def-write", "def-review", "spec-write", "pr-write", "spec-review"} {
		cmd := exec.Command("make", "--no-print-directory", "-f", filepath.Join(flowDir, "Makefile"), stage+"-prompt")
		cmd.Dir = root
		out, err := cmd.CombinedOutput()
		require.Error(t, err, "%s prompt target unexpectedly remained available: %s", stage, out)
		assert.Contains(t, string(out), "No rule to make target")
	}

	for _, name := range []string{"docs-update.md", "code-docs-spec-update.md"} {
		assert.NoFileExists(t, filepath.Join(flowDir, "prompts", name))
	}
}

func TestPromptsReadStartupGeneratedChangeTypes(t *testing.T) {
	root := repositoryRoot(t)
	tests := []struct {
		path string
		want string
	}{
		{path: filepath.Join(root, ".mch", "default", "prompts", "spec-write.md"), want: "/def-dir/prompts/change-types.md"},
		{path: filepath.Join(root, "agent", "prompts", "build-requirement-with-agent.md"), want: ".mch/default/prompts/change-types.md"},
		{path: filepath.Join(root, "agent", "prompts", "build-requirement-with-agent-example.md"), want: ".mch/default/prompts/change-types.md"},
	}

	for _, tt := range tests {
		content := readFile(t, tt.path)
		assert.Contains(t, content, tt.want)
		assert.NotContains(t, content, "/api/v1/options/change-types-list")
	}

	specWrite := readFile(t, filepath.Join(root, ".mch", "default", "prompts", "spec-write.md"))
	assert.Contains(t, specWrite, "Require the file to begin with `# Change Types`")
	assert.Contains(t, specWrite, "the complete set of allowed Change type options")
	assert.Contains(t, specWrite, "Always include a populated `Types:` line")

	structure := readFile(t, filepath.Join(root, ".mch", "default", "prompts", "spec-file-structure.md"))
	assert.Contains(t, structure, "An optional `Types:` metadata line")
	assert.Contains(t, structure, "Types: feature|fix|test")
	assert.Contains(t, structure, "does not restrict which slugs may")
	assert.NotContains(t, structure, "change-types.md")

	for _, pattern := range []string{"http://", "https://", "localhost", "api/v1/", "curl "} {
		for _, dir := range []string{
			filepath.Join(root, ".mch", "default", "prompts"),
			filepath.Join(root, "agent", "prompts"),
		} {
			paths, err := filepath.Glob(filepath.Join(dir, "*.md"))
			require.NoError(t, err)
			for _, path := range paths {
				assert.NotContains(t, strings.ToLower(readFile(t, path)), pattern, path)
			}
		}
	}
}

func TestWorkflowPromptsRequireExplicitDocumentationScope(t *testing.T) {
	root := repositoryRoot(t)
	promptDirs := []string{
		filepath.Join(root, ".mch", "default", "prompts"),
		filepath.Join(root, "agent", "prompts"),
	}
	staleContracts := []string{
		"read every relevant doc",
		"documentation as the source of truth",
		"documentation under `docs/` as the behavioral reference",
		"relevant branch documentation under `docs/`",
		"treat current `docs/` changes as the behavioral reference",
		"repository documentation",
		"documented change merge workflow",
		"documented stage-promotion workflow",
		"documented master-promotion workflow",
		"foundation for high-quality documentation",
	}

	for _, dir := range promptDirs {
		paths, err := filepath.Glob(filepath.Join(dir, "*.md"))
		require.NoError(t, err)
		for _, path := range paths {
			content := strings.ToLower(readFile(t, path))
			for _, stale := range staleContracts {
				assert.NotContains(t, content, stale, path)
			}
		}
	}

	explicitContracts := map[string]string{
		filepath.Join(root, ".mch", "default", "prompts", "code-implement.md"):             "Documentation is outside the default Change Flow.",
		filepath.Join(root, ".mch", "default", "prompts", "code-fix.md"):                   "Documentation is outside the default Change Flow.",
		filepath.Join(root, ".mch", "default", "prompts", "code-review.md"):                "Documentation is not a default review input.",
		filepath.Join(root, ".mch", "default", "prompts", "spec-write.md"):                 "Documentation is outside the default Change",
		filepath.Join(root, ".mch", "default", "prompts", "spec-review.md"):                "Documentation is outside the default Change Flow.",
		filepath.Join(root, ".mch", "default", "prompts", "pr-write.md"):                   "Documentation is outside the default Change Flow.",
		filepath.Join(root, ".mch", "default", "prompts", "spec-file-structure.md"):        "Do not add documentation work as a routine Change stage.",
		filepath.Join(root, "agent", "prompts", "change-file-init-prompt.md"):              "Documentation is outside the default Change Flow.",
		filepath.Join(root, "agent", "prompts", "change-file-code-prompt.md"):              "Documentation is outside the default Change Flow.",
		filepath.Join(root, "agent", "prompts", "change-file-fix-prompt.md"):               "Documentation is outside the default Change Flow.",
		filepath.Join(root, "agent", "prompts", "change-file-review-prompt.md"):            "Documentation is not a default review input.",
		filepath.Join(root, "agent", "prompts", "change-file-pr-prompt.md"):                "Documentation is outside the default Change Flow.",
		filepath.Join(root, "agent", "prompts", "build-requirement-with-agent.md"):         "Do not inspect documentation unless the user",
		filepath.Join(root, "agent", "prompts", "build-requirement-with-agent-example.md"): "Do not inspect documentation unless the user",
	}

	for path, contract := range explicitContracts {
		assert.Contains(t, readFile(t, path), contract, path)
	}
}

func TestChangeFileInitPromptPreservesOrdinaryIdeaWording(t *testing.T) {
	root := repositoryRoot(t)
	prompt := readFile(t, filepath.Join(root, "agent", "prompts", "change-file-init-prompt.md"))

	assert.Contains(t, prompt, "related but non-essential ideas")
	assert.NotContains(t, prompt, "related but non-essential definitions")
}

func TestDefaultFlowSessionScriptIntegration(t *testing.T) {
	root := repositoryRoot(t)
	scripts := filepath.Join(root, ".mch", "default", "scripts")
	fixture := newFlowFixture(t)

	t.Run("required and unsafe values", func(t *testing.T) {
		out, err := runScript(fixture, filepath.Join(scripts, "codex-exec-new-session.sh"), []string{"MCH_DEFAULT_DIR="})
		require.Error(t, err)
		assert.Contains(t, out, "missing MCH_DEFAULT_DIR")

		out, err = runScript(fixture, filepath.Join(scripts, "codex-exec-new-session.sh"), []string{"MCH_TEMP_DIR=../unsafe"})
		require.Error(t, err)
		assert.Contains(t, out, "invalid MCH_TEMP_DIR")

		out, err = runScript(fixture, filepath.Join(scripts, "codex-exec-new-session.sh"), []string{"MCH_REF_UUID=bad"})
		require.Error(t, err)
		assert.Contains(t, out, "invalid MCH_REF_UUID")

	})

	t.Run("stage names are delegated to configured resources", func(t *testing.T) {
		for _, name := range []string{
			"codex-no-session.sh",
			"codex-exec-new-session.sh",
			"codex-exec-restore-session.sh",
			"codex-exec-resume-session.sh",
			"codex-resume-session.sh",
		} {
			t.Run(name, func(t *testing.T) {
				out, err := runScript(newFlowFixture(t), filepath.Join(scripts, name), []string{"MCH_STAGE=custom-stage"})
				require.Error(t, err)
				assert.NotContains(t, out, "invalid MCH_STAGE")
				assert.Contains(t, out, "/custom-stage/")
			})
		}
	})

	t.Run("every entry point requires a stage", func(t *testing.T) {
		for _, name := range []string{
			"codex-no-session.sh",
			"codex-exec-new-session.sh",
			"codex-exec-restore-session.sh",
			"codex-exec-resume-session.sh",
			"codex-resume-session.sh",
		} {
			t.Run(name, func(t *testing.T) {
				out, err := runScript(newFlowFixture(t), filepath.Join(scripts, name), []string{"MCH_STAGE="})
				require.Error(t, err)
				assert.Contains(t, out, "missing MCH_STAGE")
			})
		}
	})

	t.Run("new and restored session artifacts", func(t *testing.T) {
		out, err := runScript(fixture, filepath.Join(scripts, "codex-exec-new-session.sh"), nil)
		require.NoError(t, err, out)
		for _, name := range []string{"session-id", "agent-output.md", "events.jsonl", "error.log"} {
			require.FileExists(t, filepath.Join(fixture.stageDir, name))
		}
		sessionID := strings.TrimSpace(readFile(t, filepath.Join(fixture.stageDir, "session-id")))
		require.NoError(t, os.MkdirAll(filepath.Join(fixture.codexHome, "sessions"), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(fixture.codexHome, "sessions", "rollout-"+sessionID+".jsonl"), []byte("{}"), 0o644))
		out, err = runScript(fixture, filepath.Join(scripts, "codex-exec-restore-session.sh"), nil)
		require.NoError(t, err, out)
		assert.Contains(t, readFile(t, filepath.Join(fixture.stageDir, "events.jsonl")), sessionID)
	})

	t.Run("failure status is preserved", func(t *testing.T) {
		out, err := runScript(fixture, filepath.Join(scripts, "codex-exec-restore-session.sh"), []string{"FAKE_CODEX_STATUS=7"})
		require.Error(t, err, out)
		var exitErr *exec.ExitError
		require.True(t, errors.As(err, &exitErr))
		assert.Equal(t, 7, exitErr.ExitCode())
		require.FileExists(t, filepath.Join(fixture.stageDir, "events.jsonl"))
	})

	t.Run("unknown saved session is rejected", func(t *testing.T) {
		require.NoError(t, os.WriteFile(filepath.Join(fixture.stageDir, "session-id"), []byte("22222222-2222-2222-2222-222222222222\n"), 0o644))
		out, err := runScript(fixture, filepath.Join(scripts, "codex-resume-session.sh"), nil)
		require.Error(t, err)
		assert.Contains(t, out, "unknown Codex session-id")
	})
}

func TestDefaultMakefileAcceptsConfiguredStageNames(t *testing.T) {
	root := repositoryRoot(t)
	flowMakefile := filepath.Join(root, ".mch", "default", "Makefile")
	repo := newGitRemoteFixture(t)

	cmd := exec.Command("make", "--no-print-directory", "-f", flowMakefile, "init")
	cmd.Dir = repo
	cmd.Env = append(os.Environ(),
		"MCH_TEMP_DIR=.mch/tmp",
		"MCH_REF_UUID="+testRefUUID,
		"MCH_STAGE=custom-stage",
	)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	require.DirExists(t, filepath.Join(repo, ".mch", "tmp", testRefUUID, "custom-stage"))

	cmd = exec.Command("make", "--no-print-directory", "-f", flowMakefile, "init")
	cmd.Dir = repo
	cmd.Env = append(os.Environ(),
		"MCH_TEMP_DIR=.mch/tmp",
		"MCH_REF_UUID="+testRefUUID,
		"MCH_STAGE=",
	)
	out, err = cmd.CombinedOutput()
	require.Error(t, err)
	assert.Contains(t, string(out), "missing MCH_STAGE")
}

func TestDefaultFlowSessionScriptsRenderPromptPaths(t *testing.T) {
	root := repositoryRoot(t)
	scripts := filepath.Join(root, ".mch", "default", "scripts")

	for _, name := range []string{
		"codex-no-session.sh",
		"codex-exec-new-session.sh",
		"codex-exec-restore-session.sh",
		"codex-exec-resume-session.sh",
		"codex-resume-session.sh",
	} {
		t.Run(name, func(t *testing.T) {
			fixture := newFlowFixture(t)
			sessionID := "22222222-2222-2222-2222-222222222222"
			require.NoError(t, os.WriteFile(filepath.Join(fixture.stageDir, "session-id"), []byte(sessionID+"\n"), 0o644))
			require.NoError(t, os.MkdirAll(filepath.Join(fixture.codexHome, "sessions"), 0o755))
			require.NoError(t, os.WriteFile(filepath.Join(fixture.codexHome, "sessions", "rollout-"+sessionID+".jsonl"), []byte("{}"), 0o644))
			logPath := filepath.Join(fixture.repo, "codex.log")

			out, err := runScript(fixture, filepath.Join(scripts, name), []string{"FAKE_CODEX_LOG=" + logPath})
			require.NoError(t, err, out)
			log := readFile(t, logPath)
			assert.Contains(t, log, "Read "+filepath.Join(fixture.stageDir, "input.md"))
			assert.Contains(t, log, ".mch/default/prompts/spec-file-structure.md")
			assert.NotContains(t, log, "/stg-tmp-dir/")
			assert.NotContains(t, log, "/def-dir/")
		})
	}
}

func TestDefaultFlowSpecReviewChatResumesInReviewWorkspace(t *testing.T) {
	root := repositoryRoot(t)
	flowDir := filepath.Join(root, ".mch", "default")
	fixture := newFlowFixture(t)
	reviewDir := filepath.Join(fixture.repo, ".mch", "tmp", testRefUUID, "spec-review")
	require.NoError(t, os.MkdirAll(reviewDir, 0o755))
	for _, name := range []string{"input.md", "output.md"} {
		require.NoError(t, os.WriteFile(filepath.Join(reviewDir, name), []byte("# Spec\n"), 0o644))
	}
	sessionID := "22222222-2222-2222-2222-222222222222"
	require.NoError(t, os.WriteFile(filepath.Join(reviewDir, "session-id"), []byte(sessionID+"\n"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(fixture.codexHome, "sessions"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(fixture.codexHome, "sessions", "rollout-"+sessionID+".jsonl"), []byte("{}"), 0o644))
	logPath := filepath.Join(fixture.repo, "codex.log")
	cmd := exec.Command("make", "--no-print-directory", "-f", filepath.Join(flowDir, "Makefile"), "spec-review-chat")
	cmd.Dir = fixture.repo
	cmd.Env = append(os.Environ(),
		"PATH="+fixture.fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"),
		"HOME="+fixture.repo, "CODEX_HOME="+fixture.codexHome,
		"MCH_DEFAULT_DIR=.mch/default", "MCH_TEMP_DIR=.mch/tmp", "MCH_REF_UUID="+testRefUUID,
		"FAKE_CODEX_LOG="+logPath,
	)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	assert.Contains(t, readFile(t, logPath), "resume "+sessionID)
}

func TestChangeBranchInitializationGuards(t *testing.T) {
	root := repositoryRoot(t)
	script := filepath.Join(root, ".mch", "default", "scripts", "init-branch.sh")
	repo := newGitRemoteFixture(t)

	for _, slug := range []string{
		"bad/slug",
		"001-Name.2",
		"001-Name+2",
		"Name",
		"001-",
		"001-Name/Part",
	} {
		out, err := commandInRepo(repo, script, slug)
		require.Error(t, err, slug)
		assert.Contains(t, out, "invalid change slug", slug)
	}

	require.NoError(t, os.WriteFile(filepath.Join(repo, "dirty.txt"), []byte("dirty"), 0o644))
	out, err := commandInRepo(repo, script, "001-New_change-2")
	require.Error(t, err)
	assert.Contains(t, out, "uncommitted changes")
	require.NoError(t, os.Remove(filepath.Join(repo, "dirty.txt")))

	out, err = commandInRepo(repo, script, "001-New_change-2")
	require.NoError(t, err, out)
	assert.Equal(t, "change/001-New_change-2", gitOutput(t, repo, "branch", "--show-current"))
	out, err = commandInRepo(repo, script, "001-New_change-2")
	require.NoError(t, err, out)

	gitRun(t, repo, "checkout", "stage")
	gitRun(t, repo, "push", "origin", "stage:refs/heads/change/002-remote")
	out, err = commandInRepo(repo, script, "002-remote")
	require.NoError(t, err, out)
	assert.Equal(t, "change/002-remote", gitOutput(t, repo, "branch", "--show-current"))

	gitRun(t, repo, "checkout", "stage")
	gitRun(t, repo, "branch", "change/003-existing")
	out, err = commandInRepo(repo, script, "003-conflict")
	require.Error(t, err)
	assert.Contains(t, out, "already exists locally")

	gitRun(t, repo, "push", "origin", "stage:refs/heads/change/004-existing")
	gitRun(t, repo, "fetch", "origin")
	out, err = commandInRepo(repo, script, "004-conflict")
	require.Error(t, err)
	assert.Contains(t, out, "already exists remotely")
}

func TestDefaultFlowCommitScriptGuardsSlugAndPushes(t *testing.T) {
	root := repositoryRoot(t)
	script := filepath.Join(root, ".mch", "default", "scripts", "commit.sh")
	repo := newGitRemoteFixture(t)

	gitRun(t, repo, "checkout", "-b", "change/123-commit-script")
	gitRun(t, repo, "push", "-u", "origin", "change/123-commit-script")
	require.NoError(t, os.WriteFile(filepath.Join(repo, "work.txt"), []byte("committed\n"), 0o644))
	headBefore := gitOutput(t, repo, "rev-parse", "HEAD")

	out, err := commandInRepo(repo, script, "124-other-change", "Wrong commit")
	require.Error(t, err)
	assert.Contains(t, out, "branch slug 123-commit-script does not match change slug 124-other-change")
	assert.Equal(t, headBefore, gitOutput(t, repo, "rev-parse", "HEAD"))

	out, err = commandInRepo(repo, script, "123-commit-script", "Configured commit message")
	require.NoError(t, err, out)
	assert.Equal(t, "Configured commit message", gitOutput(t, repo, "log", "-1", "--format=%s"))
	assert.Equal(t, gitOutput(t, repo, "rev-parse", "HEAD"), gitOutput(t, repo, "rev-parse", "origin/change/123-commit-script"))
}

func TestDefaultFlowPRPublishCreatesPRAndPushesURL(t *testing.T) {
	root := repositoryRoot(t)
	script := filepath.Join(root, ".mch", "default", "scripts", "pr-publish.sh")
	repo := newGitRemoteFixture(t)
	gitRun(t, repo, "checkout", "-b", "change/123-publish")
	require.NoError(t, os.MkdirAll(filepath.Join(repo, "agent", "prs"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(repo, "agent", "prs", "123-publish.md"),
		[]byte("# Publish This Change\n\nPR body.\n"),
		0o644,
	))

	fakeBin := t.TempDir()
	ghLog := filepath.Join(fakeBin, "gh.log")
	fakeGH := `#!/usr/bin/env bash
set -euo pipefail
if [[ "$1" == "api" && "$2" == "user" ]]; then
  printf '%s\n' 'divilla'
  exit 0
fi
if [[ "$1" == "pr" && "$2" == "create" ]]; then
  printf '%s\n' "$@" > "$FAKE_GH_LOG"
  printf '%s\n' 'https://github.com/divilla/project-manager/pull/123'
  exit 0
fi
printf 'unexpected gh arguments: %s\n' "$*" >&2
exit 1
`
	require.NoError(t, os.WriteFile(filepath.Join(fakeBin, "gh"), []byte(fakeGH), 0o755))
	nestedDir := filepath.Join(repo, "nested")
	require.NoError(t, os.MkdirAll(nestedDir, 0o755))

	cmd := exec.Command(script)
	cmd.Dir = nestedDir
	cmd.Env = append(
		os.Environ(),
		"PATH="+fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"),
		"FAKE_GH_LOG="+ghLog,
	)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, string(output))
	assert.Equal(t, "Write PR URL for 123-publish by agent", gitOutput(t, repo, "log", "-1", "--format=%s"))
	assert.Equal(t, "Write PR for 123-publish by agent", gitOutput(t, repo, "log", "-1", "--format=%s", "--skip=1"))
	assert.Equal(t, gitOutput(t, repo, "rev-parse", "HEAD"), gitOutput(t, repo, "rev-parse", "origin/change/123-publish"))
	assert.Equal(
		t,
		"https://github.com/divilla/project-manager/pull/123\n",
		readFile(t, filepath.Join(repo, "agent", "prurls", "123-publish")),
	)
	ghArgs := readFile(t, ghLog)
	assert.Contains(t, ghArgs, "--base\nstage\n")
	assert.Contains(t, ghArgs, "--head\ndivilla:change/123-publish\n")
	assert.Contains(t, ghArgs, "--title\nPublish This Change\n")
	assert.Contains(t, ghArgs, "--body-file\nagent/prs/123-publish.md\n")

	out, err := commandInRepo(repo, script, "unexpected")
	require.Error(t, err)
	assert.Contains(t, out, "usage: pr-publish.sh")
}

func TestChangeSlugExtractionGrammar(t *testing.T) {
	root := repositoryRoot(t)
	scripts := []struct {
		path       string
		wantOutput string
	}{
		{path: filepath.Join(root, ".mch", "default", "scripts", "extract-slug.sh"), wantOutput: "005-Name_2-Part9"},
		{path: filepath.Join(root, "scripts", "extract.sh"), wantOutput: "specs/005-Name_2-Part9.md"},
	}
	repo := newGitRemoteFixture(t)
	require.NoError(t, os.MkdirAll(filepath.Join(repo, "specs"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(repo, "specs", "005-Name_2-Part9.md"), []byte("# Spec\n"), 0o644))
	gitRun(t, repo, "checkout", "-b", "change/005-Name_2-Part9")

	for _, script := range scripts {
		out, err := commandInRepo(repo, script.path)
		require.NoError(t, err, out)
		assert.Equal(t, script.wantOutput, strings.TrimSpace(out))
	}

	for _, branch := range []string{
		"change/006-Name.2",
		"change/007-Name+2",
		"change/008-Name/Part",
		"change/Name",
		"change/009-",
		"changes/010-Name",
	} {
		gitRun(t, repo, "checkout", "stage")
		gitRun(t, repo, "checkout", "-b", branch)
		for _, script := range scripts {
			out, err := commandInRepo(repo, script.path)
			require.Error(t, err, branch)
			assert.Contains(t, out, `^change/([0-9]+-[0-9A-Za-z_-]+)$`, branch)
		}
	}
}

func TestChangeSlugGrammarIsSharedByWorkflowEntryPoints(t *testing.T) {
	root := repositoryRoot(t)
	extractionPattern := `^change/([0-9]+-[0-9A-Za-z_-]+)$`
	for _, path := range []string{
		filepath.Join(root, ".mch", "default", "prompts", "spec-review.md"),
		filepath.Join(root, ".mch", "default", "prompts", "code-implement.md"),
		filepath.Join(root, ".mch", "default", "prompts", "pr-write.md"),
		filepath.Join(root, ".mch", "default", "prompts", "code-review.md"),
		filepath.Join(root, ".mch", "default", "prompts", "code-fix.md"),
		filepath.Join(root, "agent", "prompts", "change-file-init-prompt.md"),
		filepath.Join(root, ".mch", "default", "scripts", "extract-slug.sh"),
		filepath.Join(root, "scripts", "extract.sh"),
		filepath.Join(root, "scripts", "change-def.pl"),
		filepath.Join(root, "scripts", "change-spec.pl"),
		filepath.Join(root, "scripts", "change-code.pl"),
		filepath.Join(root, "scripts", "change-fix.pl"),
		filepath.Join(root, "scripts", "change-pr.pl"),
		filepath.Join(root, ".mch", "default", "scripts", "pr-publish.sh"),
		filepath.Join(root, "scripts", "change-update.pl"),
		filepath.Join(root, "scripts", "change-stage.pl"),
	} {
		assert.Contains(t, readFile(t, path), extractionPattern, path)
	}

	validationPattern := `^[0-9]+-[0-9A-Za-z_-]+$`
	for _, path := range []string{
		filepath.Join(root, ".mch", "default", "scripts", "init-branch.sh"),
		filepath.Join(root, "scripts", "change-new.pl"),
	} {
		assert.Contains(t, readFile(t, path), validationPattern, path)
	}
}

func TestChangeDefScriptCommitsAndPushesDefinition(t *testing.T) {
	root := repositoryRoot(t)
	script := filepath.Join(root, "scripts", "change-def.pl")
	repo := newGitRemoteFixture(t)
	gitRun(t, repo, "checkout", "-b", "change/123-definition")
	require.NoError(t, os.MkdirAll(filepath.Join(repo, "agent", "defs"), 0o755))
	definition := filepath.Join(repo, "agent", "defs", "123-definition.md")
	require.NoError(t, os.WriteFile(definition, []byte("# Definition\n"), 0o644))
	gitRun(t, repo, "add", "agent/defs/123-definition.md")
	gitRun(t, repo, "commit", "-m", "initial definition")
	gitRun(t, repo, "push", "-u", "origin", "change/123-definition")
	require.NoError(t, os.WriteFile(definition, []byte("# Definition\n\nUpdated.\n"), 0o644))

	out, err := commandInRepo(repo, script)
	require.NoError(t, err, out)
	assert.Equal(t, "Definition for 123-definition by user", gitOutput(t, repo, "log", "-1", "--format=%s"))
	assert.Equal(t, gitOutput(t, repo, "rev-parse", "HEAD"), gitOutput(t, repo, "rev-parse", "origin/change/123-definition"))

	gitRun(t, repo, "checkout", "stage")
	before := gitOutput(t, repo, "rev-parse", "HEAD")
	out, err = commandInRepo(repo, script)
	require.Error(t, err)
	assert.Contains(t, out, "current branch is not a change/<change-slug> branch")
	assert.Equal(t, before, gitOutput(t, repo, "rev-parse", "HEAD"))
}

func TestChangeNewScriptInitializesLocalBranchWithoutPublishing(t *testing.T) {
	root := repositoryRoot(t)
	script := filepath.Join(root, "scripts", "change-new.pl")
	repo := newGitRemoteFixture(t)
	for _, dir := range []string{"agent/defs", "agent/prs", "specs"} {
		require.NoError(t, os.MkdirAll(filepath.Join(repo, dir), 0o755))
	}

	out, err := commandInRepo(repo, script, "124-initialized")
	require.NoError(t, err, out)
	assert.Equal(t, "change/124-initialized", gitOutput(t, repo, "branch", "--show-current"))
	upstream := exec.Command("git", "-C", repo, "rev-parse", "--abbrev-ref", "@{upstream}")
	require.Error(t, upstream.Run())
	remoteRef := exec.Command(
		"git", "-C", repo, "ls-remote", "--exit-code", "origin",
		"refs/heads/change/124-initialized",
	)
	require.Error(t, remoteRef.Run())
	require.FileExists(t, filepath.Join(repo, "agent", "defs", "124-initialized.md"))
	require.FileExists(t, filepath.Join(repo, "specs", "124-initialized.md"))
	require.FileExists(t, filepath.Join(repo, "agent", "prs", "124-initialized.md"))
}

func TestChangeNewScriptReusesExistingBranchesWithoutPublishing(t *testing.T) {
	root := repositoryRoot(t)
	script := filepath.Join(root, "scripts", "change-new.pl")

	t.Run("untracked branch and commits remain local", func(t *testing.T) {
		repo := newGitRemoteFixture(t)
		for _, dir := range []string{"agent/defs", "agent/prs", "specs"} {
			require.NoError(t, os.MkdirAll(filepath.Join(repo, dir), 0o755))
		}

		gitRun(t, repo, "checkout", "-b", "change/125-existing-local")
		require.NoError(t, os.WriteFile(filepath.Join(repo, "local.txt"), []byte("local\n"), 0o644))
		gitRun(t, repo, "add", "local.txt")
		gitRun(t, repo, "commit", "-m", "unpublished local work")
		localHead := gitOutput(t, repo, "rev-parse", "HEAD")
		remoteBefore := gitOutput(t, repo, "ls-remote", "origin")
		gitRun(t, repo, "checkout", "stage")

		out, err := commandInRepo(repo, script, "125-existing-local")
		require.NoError(t, err, out)
		assert.Equal(t, "change/125-existing-local", gitOutput(t, repo, "branch", "--show-current"))
		assert.Equal(t, localHead, gitOutput(t, repo, "rev-parse", "HEAD"))
		upstream := exec.Command("git", "-C", repo, "rev-parse", "--abbrev-ref", "@{upstream}")
		require.Error(t, upstream.Run())
		remoteRef := exec.Command(
			"git", "-C", repo, "ls-remote", "--exit-code", "origin",
			"refs/heads/change/125-existing-local",
		)
		require.Error(t, remoteRef.Run())
		assert.Equal(t, remoteBefore, gitOutput(t, repo, "ls-remote", "origin"))
		require.FileExists(t, filepath.Join(repo, "agent", "defs", "125-existing-local.md"))
		require.FileExists(t, filepath.Join(repo, "specs", "125-existing-local.md"))
		require.FileExists(t, filepath.Join(repo, "agent", "prs", "125-existing-local.md"))
	})

	t.Run("tracked branch keeps its upstream without publishing new commits", func(t *testing.T) {
		repo := newGitRemoteFixture(t)
		for _, dir := range []string{"agent/defs", "agent/prs", "specs"} {
			require.NoError(t, os.MkdirAll(filepath.Join(repo, dir), 0o755))
		}

		gitRun(t, repo, "checkout", "-b", "change/126-existing-tracked")
		require.NoError(t, os.WriteFile(filepath.Join(repo, "tracked.txt"), []byte("published\n"), 0o644))
		gitRun(t, repo, "add", "tracked.txt")
		gitRun(t, repo, "commit", "-m", "published work")
		gitRun(t, repo, "push", "-u", "origin", "change/126-existing-tracked")
		remoteBefore := gitOutput(t, repo, "rev-parse", "origin/change/126-existing-tracked")
		require.NoError(t, os.WriteFile(filepath.Join(repo, "tracked.txt"), []byte("local update\n"), 0o644))
		gitRun(t, repo, "add", "tracked.txt")
		gitRun(t, repo, "commit", "-m", "unpublished tracked work")
		localHead := gitOutput(t, repo, "rev-parse", "HEAD")
		gitRun(t, repo, "checkout", "stage")

		out, err := commandInRepo(repo, script, "126-existing-tracked")
		require.NoError(t, err, out)
		assert.Equal(t, "change/126-existing-tracked", gitOutput(t, repo, "branch", "--show-current"))
		assert.Equal(t, localHead, gitOutput(t, repo, "rev-parse", "HEAD"))
		assert.Equal(t, "origin/change/126-existing-tracked", gitOutput(t, repo, "rev-parse", "--abbrev-ref", "@{upstream}"))
		assert.Equal(t, remoteBefore, gitOutput(t, repo, "rev-parse", "origin/change/126-existing-tracked"))
		assert.NotEqual(t, localHead, remoteBefore)
		require.FileExists(t, filepath.Join(repo, "agent", "defs", "126-existing-tracked.md"))
		require.FileExists(t, filepath.Join(repo, "specs", "126-existing-tracked.md"))
		require.FileExists(t, filepath.Join(repo, "agent", "prs", "126-existing-tracked.md"))
	})
}

func TestMasterPromotionRefusesUnsafeState(t *testing.T) {
	root := repositoryRoot(t)
	script := filepath.Join(root, ".mch", "default", "scripts", "deploy-master.sh")
	repo := newGitRemoteFixture(t)

	gitRun(t, repo, "checkout", "-b", "feature")
	out, err := commandInRepo(repo, script)
	require.Error(t, err)
	assert.Contains(t, out, "checkout stage")
	gitRun(t, repo, "checkout", "stage")
	require.NoError(t, os.WriteFile(filepath.Join(repo, "dirty.txt"), []byte("dirty"), 0o644))
	out, err = commandInRepo(repo, script)
	require.Error(t, err)
	assert.Contains(t, out, "uncommitted changes")
	require.NoError(t, os.Remove(filepath.Join(repo, "dirty.txt")))

	gitRun(t, repo, "checkout", "-b", "master")
	require.NoError(t, os.WriteFile(filepath.Join(repo, "master.txt"), []byte("master"), 0o644))
	gitRun(t, repo, "add", "master.txt")
	gitRun(t, repo, "commit", "-m", "master divergence")
	gitRun(t, repo, "push", "-u", "origin", "master")
	gitRun(t, repo, "checkout", "stage")
	require.NoError(t, os.WriteFile(filepath.Join(repo, "stage.txt"), []byte("stage"), 0o644))
	gitRun(t, repo, "add", "stage.txt")
	gitRun(t, repo, "commit", "-m", "stage divergence")
	gitRun(t, repo, "push", "origin", "stage")
	out, err = commandInRepo(repo, script)
	require.Error(t, err)
	assert.Contains(t, out, "master is not an ancestor")
}

func TestMasterPromotionRefusesStageThatMovesDuringPromotion(t *testing.T) {
	root := repositoryRoot(t)
	script := filepath.Join(root, ".mch", "default", "scripts", "deploy-master.sh")
	repo := newGitRemoteFixture(t)
	gitRun(t, repo, "branch", "master")
	gitRun(t, repo, "push", "origin", "master")
	require.NoError(t, os.WriteFile(filepath.Join(repo, "stage.txt"), []byte("stage"), 0o644))
	gitRun(t, repo, "add", "stage.txt")
	gitRun(t, repo, "commit", "-m", "advance stage")
	gitRun(t, repo, "push", "origin", "stage")

	realGit, err := exec.LookPath("git")
	require.NoError(t, err)
	binDir := filepath.Join(t.TempDir(), "bin")
	require.NoError(t, os.MkdirAll(binDir, 0o755))
	countPath := filepath.Join(binDir, "count")
	wrapper := fmt.Sprintf(`#!/usr/bin/env bash
set -euo pipefail
if [[ "$*" == *"ls-remote --heads origin stage"* ]]; then
  count=0
  [[ -f %q ]] && count="$(cat %q)"
  count=$((count+1)); printf '%%s' "$count" > %q
  if ((count >= 2)); then printf '0000000000000000000000000000000000000000\trefs/heads/stage\n'; exit 0; fi
fi
exec %q "$@"
`, countPath, countPath, countPath, realGit)
	require.NoError(t, os.WriteFile(filepath.Join(binDir, "git"), []byte(wrapper), 0o755))
	cmd := exec.Command(script)
	cmd.Dir = repo
	cmd.Env = append(os.Environ(), "PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	out, err := cmd.CombinedOutput()
	require.Error(t, err)
	assert.Contains(t, string(out), "origin/stage moved")
}

type flowFixture struct {
	repo      string
	stageDir  string
	prompt    string
	fakeBin   string
	codexHome string
}

func newFlowFixture(t *testing.T) flowFixture {
	t.Helper()
	repo := t.TempDir()
	require.NoError(t, exec.Command("git", "init", repo).Run())
	stageDir := filepath.Join(repo, ".mch", "tmp", testRefUUID, "artifact")
	require.NoError(t, os.MkdirAll(stageDir, 0o755))
	for _, name := range []string{"input.md", "output.md"} {
		require.NoError(t, os.WriteFile(filepath.Join(stageDir, name), []byte("# Definition\n"), 0o644))
	}
	prompt := filepath.Join(repo, "prompt.md")
	require.NoError(t, os.WriteFile(prompt, []byte("Read /stg-tmp-dir/input.md using /def-dir/prompts/spec-file-structure.md"), 0o644))
	fakeBin := filepath.Join(repo, "bin")
	require.NoError(t, os.MkdirAll(fakeBin, 0o755))
	fakeCodex := `#!/usr/bin/env bash
set -euo pipefail
output=""
session="11111111-1111-1111-1111-111111111111"
if [[ -n "${FAKE_CODEX_LOG:-}" ]]; then printf '%s\n' "$*" >> "${FAKE_CODEX_LOG}"; fi
while (($#)); do
  if [[ "$1" == "-o" ]]; then output="$2"; shift 2; continue; fi
  if [[ "$1" =~ ^[0-9a-f-]{36}$ ]]; then session="$1"; fi
  shift
done
if [[ -n "$output" ]]; then printf 'Done.' > "$output"; fi
printf '{"type":"thread.started","thread_id":"%s"}\n' "$session"
exit "${FAKE_CODEX_STATUS:-0}"
`
	require.NoError(t, os.WriteFile(filepath.Join(fakeBin, "codex"), []byte(fakeCodex), 0o755))
	return flowFixture{repo: repo, stageDir: stageDir, prompt: prompt, fakeBin: fakeBin, codexHome: filepath.Join(repo, "codex-home")}
}

func newGitRemoteFixture(t *testing.T) string {
	t.Helper()
	base := t.TempDir()
	bare := filepath.Join(base, "origin.git")
	repo := filepath.Join(base, "work")
	require.NoError(t, exec.Command("git", "init", "--bare", bare).Run())
	require.NoError(t, exec.Command("git", "init", repo).Run())
	gitRun(t, repo, "config", "user.email", "tests@example.test")
	gitRun(t, repo, "config", "user.name", "Flow Tests")
	require.NoError(t, os.WriteFile(filepath.Join(repo, "README.md"), []byte("fixture\n"), 0o644))
	gitRun(t, repo, "add", "README.md")
	gitRun(t, repo, "commit", "-m", "initial")
	gitRun(t, repo, "branch", "-M", "stage")
	gitRun(t, repo, "remote", "add", "origin", bare)
	gitRun(t, repo, "push", "-u", "origin", "stage")
	return repo
}

func gitRun(t *testing.T, repo string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
}

func gitOutput(t *testing.T, repo string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	return strings.TrimSpace(string(out))
}

func commandInRepo(repo string, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = repo
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func runScript(f flowFixture, script string, overrides []string) (string, error) {
	env := []string{
		"PATH=" + f.fakeBin + string(os.PathListSeparator) + os.Getenv("PATH"),
		"HOME=" + f.repo,
		"CODEX_HOME=" + f.codexHome,
		"MCH_DEFAULT_DIR=.mch/default",
		"MCH_TEMP_DIR=.mch/tmp",
		"MCH_REF_UUID=" + testRefUUID,
		"MCH_STAGE=artifact",
	}
	for _, override := range overrides {
		key := strings.SplitN(override, "=", 2)[0] + "="
		filtered := env[:0]
		for _, value := range env {
			if !strings.HasPrefix(value, key) {
				filtered = append(filtered, value)
			}
		}
		env = append(filtered, override)
	}
	cmd := exec.Command(script, f.prompt)
	cmd.Dir = f.repo
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	body, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(body)
}
