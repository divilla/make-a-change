package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkspaceEnsureReplacesRegularFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mch")
	require.NoError(t, os.WriteFile(path, []byte("not a directory"), 0o644))

	workspace := Workspace{Dir: path}
	require.NoError(t, workspace.Ensure())

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestWorkspaceDefFileLifecycle(t *testing.T) {
	workspace := Workspace{Dir: t.TempDir()}

	exists, err := workspace.DefExists()
	require.NoError(t, err)
	assert.False(t, exists)

	require.NoError(t, workspace.ResetDef())
	exists, err = workspace.DefExists()
	require.NoError(t, err)
	assert.True(t, exists)

	content, err := workspace.ReadDef()
	require.NoError(t, err)
	assert.Empty(t, content)
}

func TestExtractSessionID(t *testing.T) {
	tests := []struct {
		name string
		log  string
		want string
	}{
		{name: "thread started thread id", log: `{"type":"thread.started","thread_id":"thread-1"}`, want: "thread-1"},
		{name: "thread started wins over earlier event id", log: "{\"type\":\"item.started\",\"id\":\"item-1\"}\n{\"type\":\"thread.started\",\"thread_id\":\"thread-1\"}", want: "thread-1"},
		{name: "top level session_id", log: `{"session_id":"abc"}`, want: "abc"},
		{name: "nested session id", log: `{"session":{"id":"nested"}}`, want: "nested"},
		{name: "top level id", log: `{"id":"top"}`, want: "top"},
		{name: "first valid line", log: "not-json\n{\"session_id\":\"later\"}", want: "later"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ExtractSessionID(tt.log))
		})
	}
}

func TestFormatCommandOutputHumanizesJSONLines(t *testing.T) {
	output := strings.Join([]string{
		`{"type":"thread.started","thread_id":"thread-1"}`,
		`{"type":"turn.started"}`,
		`{"type":"item.completed","item":{"id":"item_0","type":"agent_message","text":"Reading def."}}`,
		`{"type":"item.started","item":{"id":"item_1","type":"command_execution","command":"sed -n '1,40p' /tmp/mch/initial-def.md","aggregated_output":"","exit_code":null,"status":"in_progress"}}`,
		`{"type":"item.completed","item":{"id":"item_1","type":"command_execution","command":"sed -n '1,40p' /tmp/mch/initial-def.md","aggregated_output":"# Definition\n\nBody.","exit_code":0,"status":"completed"}}`,
		`{"type":"item.completed","item":{"id":"item_2","type":"file_change","changes":[{"path":"/tmp/mch/initial-def.md","kind":"update"}],"status":"completed"}}`,
		`{"type":"turn.completed","usage":{"input_tokens":10,"cached_input_tokens":5,"output_tokens":2,"reasoning_output_tokens":1}}`,
		`{"type":"error","message":"failed"}`,
		`plain text`,
		``,
	}, "\n")

	formatted := FormatCommandOutput(output)

	assert.Contains(t, formatted, "thread started: thread-1")
	assert.Contains(t, formatted, "turn started")
	assert.Contains(t, formatted, "assistant: Reading def.")
	assert.Contains(t, formatted, "running command: sed -n '1,40p' /tmp/mch/initial-def.md")
	assert.Contains(t, formatted, "command completed (completed): exit 0")
	assert.Contains(t, formatted, "output:\n  # Definition\n  \n  Body.")
	assert.Contains(t, formatted, "file change completed:\n  update: /tmp/mch/initial-def.md")
	assert.Contains(t, formatted, "turn completed (input=10, cached_input=5, output=2, reasoning_output=1)")
	assert.Contains(t, formatted, "error: failed")
	assert.Contains(t, formatted, "plain text")
	assert.NotContains(t, formatted, `"type":`)
}

func TestCodexPromptsUseExpectedSkills(t *testing.T) {
	workspace := Workspace{Dir: "/tmp/custom-mch"}
	assert.Equal(t, `Use $change-def-tmp. The temporary workspace is "/tmp/custom-mch". Read and replace "/tmp/custom-mch/initial-def.md".`, RewritePrompt(workspace))
	changeWorkspace := NewChangeModel("/repo/.mch/tmp", "0198a86f-9b8a-7d89-ae5b-6f25b528b04c").Workspace
	assert.Contains(t, RewritePrompt(changeWorkspace), changeWorkspace.InputPath())
	assert.Contains(t, RewritePrompt(changeWorkspace), changeWorkspace.OutputPath())
	for _, operation := range []WriteOperation{DefWriteOperation, SpecWriteOperation, PRWriteOperation} {
		workspace := NewArtifactModel("/repo/.mch/tmp", "0198a86f-9b8a-7d89-ae5b-6f25b528b04c", operation).Workspace
		assert.Equal(t, ArtifactStage, workspace.Stage)
		assert.Contains(t, RewritePrompt(workspace), filepath.Join(DefaultDir, "prompts", string(operation)+".md"))
	}
	assert.Equal(t, "Use $change-spec-tmp.", InitPrompt)
}

func TestExistingArtifactPreparationPreservesDefWorkspaceSession(t *testing.T) {
	workspace := NewArtifactModel(t.TempDir(), "0198a86f-9b8a-7d89-ae5b-6f25b528b04c", SpecWriteOperation).Workspace
	require.NoError(t, workspace.Ensure())
	require.NoError(t, os.WriteFile(workspace.SessionPath(), []byte("session-1\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(workspace.Dir, "events.jsonl"), []byte("events"), 0o644))
	require.NoError(t, workspace.PrepareArtifact("# Existing spec"))

	assert.Equal(t, "# Existing spec", readWorkspaceFile(t, workspace.InputPath()))
	assert.Equal(t, "# Existing spec", readWorkspaceFile(t, workspace.OutputPath()))
	sessionID, err := workspace.ReadSessionID()
	require.NoError(t, err)
	assert.Equal(t, "session-1", sessionID)
	assert.Equal(t, "events", readWorkspaceFile(t, filepath.Join(workspace.Dir, "events.jsonl")))
}

func readWorkspaceFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(content)
}

func TestWorkflowEnvironmentOverridesInheritedValues(t *testing.T) {
	workspace := Workspace{RefUUID: "0198a86f-9b8a-7d89-ae5b-6f25b528b04c"}
	env := workflowEnvironment([]string{"PATH=/bin", "MCH_STAGE=spec", "MCH_TEMP_DIR=/unsafe"}, workspace)
	assert.ElementsMatch(t, []string{
		"PATH=/bin",
		"MCH_DEFAULT_DIR=.mch/default",
		"MCH_TEMP_DIR=.mch/tmp",
		"MCH_REF_UUID=0198a86f-9b8a-7d89-ae5b-6f25b528b04c",
		"MCH_STAGE=artifact",
	}, env)
}

func TestParseGeneratedChange(t *testing.T) {
	spec := "# Generated Change\n\nTypes: feature|test\n\n## Goal\nShip it.\n\n## QA Test Cases\n\n- First scenario.\n- Second scenario spans\n  more detail.\n1. Numbered scenario.\n\n## Review Focus\n\n- Parser."

	parsed, err := ParseGeneratedChange(spec)
	require.NoError(t, err)

	assert.Equal(t, "Generated Change", parsed.Title)
	assert.Equal(t, []string{"feature", "test"}, parsed.ChangeTypes)
	assert.True(t, parsed.ChangeTypesPresent)
	assert.Equal(t, []string{"First scenario.", "Second scenario spans more detail.", "Numbered scenario."}, parsed.TestCases)
	assert.Equal(t, spec, parsed.Spec)
}

func TestParseGeneratedChangeSkipsNoneQATestCase(t *testing.T) {
	spec := "# Generated Change\n\nTypes: feature\n\n## QA Test Cases\n\n- None.\n\n## Review Focus\n\n- Parser."

	parsed, err := ParseGeneratedChange(spec)
	require.NoError(t, err)

	assert.Empty(t, parsed.TestCases)
}

func TestParseGeneratedChangeExtractsQATestCasesFromGeneratedBody(t *testing.T) {
	spec := `# Generated Change

Types: feature|test

## Relevant Specs

- ` + "`/tmp/mch/initial-change.md`" + `
- ` + "`.mch/default/prompts/spec-file-structure.md`" + `
- ` + "`docs/architecture/mch.md`" + `

## Verification

- From ` + "`cli`" + `: ` + "`make lint`" + `
- From ` + "`cli`" + `: ` + "`go test ./...`" + `
- From ` + "`cli`" + `: ` + "`go build -o /tmp/mch ./cmd/mch`" + `

## QA Test Cases

- Start ` + "`/new-change`" + ` with no valid current project and confirm the flow stops with a recoverable project-context error and no editor, Codex, or create request runs.
- Start ` + "`/new-change`" + ` when ` + "`/tmp/mch`" + ` is absent and confirm the directory and blank ` + "`initial-def.md`" + ` are created.
- Simulate a generated test case create failure after Change create and confirm the error is displayed above the prompt and created details are not opened.
- Complete a successful create and confirm ` + "`mch`" + ` reloads and renders the created Change detail using backend data.

## Review Focus

- Generated Change parser strictness for H1 title, ` + "`Types:`" + ` metadata, type slugs, full spec preservation, and ` + "`## QA Test Cases`" + ` extraction.`

	parsed, err := ParseGeneratedChange(spec)
	require.NoError(t, err)

	assert.Equal(t, []string{
		"Start `/new-change` with no valid current project and confirm the flow stops with a recoverable project-context error and no editor, Codex, or create request runs.",
		"Start `/new-change` when `/tmp/mch` is absent and confirm the directory and blank `initial-def.md` are created.",
		"Simulate a generated test case create failure after Change create and confirm the error is displayed above the prompt and created details are not opened.",
		"Complete a successful create and confirm `mch` reloads and renders the created Change detail using backend data.",
	}, parsed.TestCases)
}

func TestExtractQATestCasesFromSpecFragment(t *testing.T) {
	spec := `## Relevant Specs

- ` + "`/tmp/mch/initial-change.md`" + `
- ` + "`.mch/default/prompts/spec-file-structure.md`" + `

## Verification

- From ` + "`cli`" + `: ` + "`make lint`" + `
- From ` + "`cli`" + `: ` + "`go test ./...`" + `

## QA Test Cases

- Start ` + "`/new-change`" + ` with no valid current project and confirm the flow stops with a recoverable project-context error and no editor, Codex, or create request runs.
- Start ` + "`/new-change`" + ` when ` + "`/tmp/mch`" + ` is absent and confirm the directory and blank ` + "`initial-def.md`" + ` are created.
- Complete a successful create and confirm ` + "`mch`" + ` reloads and renders the created Change detail using backend data.

## Review Focus

- Generated Change parser strictness for H1 title, ` + "`Types:`" + ` metadata, type slugs, full spec preservation, and ` + "`## QA Test Cases`" + ` extraction.`

	assert.Equal(t, []string{
		"Start `/new-change` with no valid current project and confirm the flow stops with a recoverable project-context error and no editor, Codex, or create request runs.",
		"Start `/new-change` when `/tmp/mch` is absent and confirm the directory and blank `initial-def.md` are created.",
		"Complete a successful create and confirm `mch` reloads and renders the created Change detail using backend data.",
	}, ExtractQATestCases(spec))
}

func TestParseGeneratedChangeValidation(t *testing.T) {
	tests := []struct {
		name string
		spec string
	}{
		{name: "missing title", spec: "Types: feature\n\n## Goal\nShip it."},
		{name: "leading blank", spec: "\n# Generated Change\n\nTypes: feature\n\n## Goal\nShip it."},
		{name: "no blank after title", spec: "# Generated Change\nTypes: feature\n\n## Goal\nShip it."},
		{name: "missing body", spec: "# Generated Change\n\nTypes: feature"},
		{name: "blank body", spec: "# Generated Change\n\nTypes:\n\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseGeneratedChange(tt.spec)
			require.Error(t, err)
		})
	}
}

func TestParseGeneratedChangeAllowsOmittedEmptyAndUnsupportedTypes(t *testing.T) {
	tests := []struct {
		name    string
		spec    string
		present bool
		values  []string
	}{
		{name: "omitted", spec: "# Generated Change\n\n## Goal\nShip it."},
		{name: "empty", spec: "# Generated Change\n\nTypes:\n\n## Goal\nShip it.", present: true, values: []string{}},
		{name: "pipe-delimited unsupported and punctuation", spec: "# Generated Change\n\nTypes: fix|feature|unsupported!\n\n## Goal\nShip it.", present: true, values: []string{"fix", "feature", "unsupported"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := ParseGeneratedChange(tt.spec)
			require.NoError(t, err)
			assert.Equal(t, tt.present, parsed.ChangeTypesPresent)
			assert.Equal(t, tt.values, parsed.ChangeTypes)
		})
	}
}
