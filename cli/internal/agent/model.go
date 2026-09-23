package agent

import (
	"fmt"
	"path/filepath"
)

// Stage identifies where the user is in an agent-assisted workflow.
type Stage string

const (
	// StageIdle means no agent-assisted workflow is active.
	StageIdle Stage = ""
	// StageDefEntry means the user is editing the initial definition.
	StageDefEntry Stage = "def entry"
	// StageCreateConfirmation means the user is confirming whether to create a Change.
	StageCreateConfirmation Stage = "create confirmation"
	// StagePersistedDefEntry means the user is correcting a definition after its Change was created.
	StagePersistedDefEntry Stage = "persisted def entry"
	// StageArtifactEntry means the user is editing an existing Change artifact.
	StageArtifactEntry Stage = "artifact entry"
	// StageAIRunning means a Codex process is currently active.
	StageAIRunning Stage = "AI running"
)

// WriteOperation identifies the Flow prompt used to write an artifact.
type WriteOperation string

const (
	// DefWriteOperation rewrites a definition in the shared artifact workspace.
	DefWriteOperation WriteOperation = "def-write"
	// SpecWriteOperation writes a Spec in the shared artifact workspace.
	SpecWriteOperation WriteOperation = "spec-write"
	// PRWriteOperation writes a PR in the shared artifact workspace.
	PRWriteOperation WriteOperation = "pr-write"
)

const (
	// DefaultDir is the repository-relative default Flow resource directory.
	DefaultDir = ".mch/default"
	// TempDir is the repository-relative runtime workspace root.
	TempDir = ".mch/tmp"
	// ArtifactStage is the shared stage launched for definition, Spec, and PR artifacts.
	ArtifactStage = "artifact"
	// InputFileName is the immutable baseline for one editor pass.
	InputFileName = "input.md"
	// OutputFileName is the editable and rewritten new-Change definition artifact.
	OutputFileName = "output.md"
	// DefFileName is the legacy existing-Change editor artifact.
	DefFileName = "initial-def.md"
	// GeneratedFileName is the generated Change spec markdown file.
	GeneratedFileName = "initial-change.md"
	// CodexOutputName is the final text output file written by Codex exec.
	CodexOutputName = "codex-output.txt"
	// CodexRunLogName is the JSON event log file for the initial Codex exec.
	CodexRunLogName = "codex-run.jsonl"
	// SessionFileName stores the Codex session shared by artifact-write operations.
	SessionFileName = "session-id"
	// InitPrompt is the Codex prompt for generating the final Change spec.
	InitPrompt = "Use $change-spec-tmp."
	// GenericError is shown when Codex output does not satisfy the flow contract.
	GenericError = "something went wrong - please try again"
)

// RewritePrompt returns the Codex prompt for rewriting the configured definition file.
func RewritePrompt(workspace Workspace) string {
	if workspace.RootDir == "" {
		return fmt.Sprintf("Use $change-def-tmp. The temporary workspace is %q. Read and replace %q.", workspace.Dir, workspace.DefPath())
	}
	operation := workspace.Operation
	if operation == "" {
		operation = DefWriteOperation
	}
	return fmt.Sprintf("Execute the %s Flow prompt from %q. When following it, replace /stg-tmp-dir/ with %q and /def-dir/ with %q. Read %q and write the complete result to %q.", operation, filepath.Join(DefaultDir, "prompts", string(operation)+".md"), workspace.Dir, DefaultDir, workspace.InputPath(), workspace.OutputPath())
}

// Model stores agent-assisted workflow state.
type Model struct {
	Workspace       Workspace
	Stage           Stage
	SessionID       string
	RepoRoot        string
	DefEntryContent string
	CommandOutput   string
}

// NewModelWithWorkspace returns an idle agent model for a specific workspace.
func NewModelWithWorkspace(dir string) Model {
	return Model{Workspace: Workspace{Dir: dir}}
}

// NewChangeModel returns a model rooted in one UUID and artifact-stage workspace.
func NewChangeModel(tempRoot string, refUUID string) Model {
	return NewArtifactModel(tempRoot, refUUID, DefWriteOperation)
}

// NewArtifactModel returns a model for one existing Change artifact-write operation.
// All write operations deliberately reuse the artifact workspace and session.
func NewArtifactModel(tempRoot string, refUUID string, operation WriteOperation) Model {
	return Model{Workspace: Workspace{
		Dir:       filepath.Join(tempRoot, refUUID, ArtifactStage),
		RootDir:   filepath.Join(tempRoot, refUUID),
		RefUUID:   refUUID,
		Stage:     ArtifactStage,
		Operation: operation,
	}}
}

// Active reports whether an agent flow is in progress.
func (m Model) Active() bool {
	return m.Stage != StageIdle
}
