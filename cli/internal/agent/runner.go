package agent

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
)

// RewriteResult stores the observed result of a Codex rewrite run.
type RewriteResult struct {
	RepoRoot      string
	SessionID     string
	Output        string
	CommandOutput string
}

// RewriteProgress receives command output while a rewrite command is still running.
type RewriteProgress func(output string)

// Runner executes external commands for agent-assisted workflows.
type Runner interface {
	ResolveRepoRoot(ctx context.Context) (string, error)
	Rewrite(ctx context.Context, repoRoot string, sessionID string, workspace Workspace, progress RewriteProgress) (RewriteResult, error)
	InitCommand(repoRoot string, sessionID string) *exec.Cmd
}

// ProcessRunner runs the real Codex and Git commands.
type ProcessRunner struct{}

// NewProcessRunner creates a runner backed by local processes.
func NewProcessRunner() ProcessRunner {
	return ProcessRunner{}
}

// ResolveRepoRoot resolves the current Git repository root.
func (ProcessRunner) ResolveRepoRoot(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	repoRoot := strings.TrimSpace(string(output))
	if repoRoot == "" {
		return "", fmt.Errorf("repository root is required")
	}
	return repoRoot, nil
}

// Rewrite runs Codex in JSON event mode and captures output/session data.
func (ProcessRunner) Rewrite(ctx context.Context, repoRoot string, sessionID string, workspace Workspace, progress RewriteProgress) (RewriteResult, error) {
	args := []string{"exec"}
	prompt := RewritePrompt(workspace)
	if sessionID == "" {
		args = append(args, "--json", "-C", repoRoot, "-o", workspace.AgentOutputPath(), prompt)
	} else {
		args = append(args, "resume", "-o", workspace.AgentOutputPath(), sessionID, prompt)
	}
	cmd := exec.CommandContext(ctx, "codex", args...)
	cmd.Dir = repoRoot
	cmd.Env = workflowEnvironment(os.Environ(), workspace)
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return RewriteResult{RepoRoot: repoRoot}, err
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return RewriteResult{RepoRoot: repoRoot}, err
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := cmd.Start(); err != nil {
		return RewriteResult{RepoRoot: repoRoot}, err
	}
	var wg sync.WaitGroup
	var stdoutErr error
	var stderrErr error
	wg.Add(2)
	go func() {
		defer wg.Done()
		stdoutErr = captureCommandOutput(stdoutPipe, "", &stdout, progress)
	}()
	go func() {
		defer wg.Done()
		stderrErr = captureCommandOutput(stderrPipe, "stderr:\n", &stderr, progress)
	}()
	runErr := cmd.Wait()
	wg.Wait()
	commandOutput := commandOutputText(stdout.String(), stderr.String())
	if runErr == nil {
		if stdoutErr != nil {
			return RewriteResult{RepoRoot: repoRoot, CommandOutput: commandOutput}, stdoutErr
		}
		if stderrErr != nil {
			return RewriteResult{RepoRoot: repoRoot, CommandOutput: commandOutput}, stderrErr
		}
	}
	if sessionID == "" {
		if err := writeFile(workspace.LogPath(), stdout.Bytes()); err != nil && runErr == nil {
			return RewriteResult{RepoRoot: repoRoot, CommandOutput: commandOutput}, err
		}
		sessionID = ExtractSessionID(stdout.String())
	}
	output, err := readFile(workspace.AgentOutputPath())
	if err != nil && runErr == nil {
		return RewriteResult{RepoRoot: repoRoot, SessionID: sessionID, CommandOutput: commandOutput}, err
	}
	result := RewriteResult{RepoRoot: repoRoot, SessionID: sessionID, Output: output, CommandOutput: commandOutput}
	if runErr != nil {
		if strings.TrimSpace(stderr.String()) != "" {
			return result, fmt.Errorf("%w: %s", runErr, strings.TrimSpace(stderr.String()))
		}
		return result, runErr
	}
	return result, nil
}

func workflowEnvironment(base []string, workspace Workspace) []string {
	overrides := map[string]string{
		"MCH_DEFAULT_DIR": DefaultDir,
		"MCH_TEMP_DIR":    TempDir,
		"MCH_REF_UUID":    workspace.RefUUID,
		"MCH_STAGE":       ArtifactStage,
	}
	result := make([]string, 0, len(base)+len(overrides))
	for _, entry := range base {
		key, _, ok := strings.Cut(entry, "=")
		if ok {
			if _, replaced := overrides[key]; replaced {
				continue
			}
		}
		result = append(result, entry)
	}
	for _, key := range []string{"MCH_DEFAULT_DIR", "MCH_TEMP_DIR", "MCH_REF_UUID", "MCH_STAGE"} {
		result = append(result, key+"="+overrides[key])
	}
	return result
}

func captureCommandOutput(reader io.Reader, prefix string, output *bytes.Buffer, progress RewriteProgress) error {
	buffered := bufio.NewReader(reader)
	for {
		chunk, err := buffered.ReadString('\n')
		if chunk != "" {
			output.WriteString(chunk)
			if progress != nil {
				progress(prefix + chunk)
			}
		}
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
	}
}

// InitCommand builds the interactive Codex command that generates the final Change spec.
func (ProcessRunner) InitCommand(repoRoot string, sessionID string) *exec.Cmd {
	cmd := exec.Command("codex", "resume", sessionID, InitPrompt)
	cmd.Dir = repoRoot
	return cmd
}

var readFile = func(path string) (string, error) {
	output, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(output), nil
}

var writeFile = func(path string, content []byte) error {
	return os.WriteFile(path, content, 0o644)
}

func commandOutputText(stdout string, stderr string) string {
	parts := make([]string, 0, 2)
	if strings.TrimSpace(stdout) != "" {
		parts = append(parts, strings.TrimRight(stdout, "\n"))
	}
	if strings.TrimSpace(stderr) != "" {
		parts = append(parts, "stderr:\n"+strings.TrimRight(stderr, "\n"))
	}
	return strings.Join(parts, "\n")
}

// FormatCommandOutput formats JSON-line command output into a readable form.
func FormatCommandOutput(output string) string {
	lines := strings.Split(strings.ReplaceAll(output, "\r\n", "\n"), "\n")
	formatted := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		var event map[string]any
		if err := json.Unmarshal([]byte(trimmed), &event); err == nil {
			formatted = append(formatted, formatCodexEvent(event))
			continue
		}
		formatted = append(formatted, trimmed)
	}
	return strings.Join(formatted, "\n")
}

func formatCodexEvent(event map[string]any) string {
	eventType := stringField(event, "type")
	switch eventType {
	case "thread.started":
		if id := stringField(event, "thread_id"); id != "" {
			return "thread started: " + id
		}
		return "thread started"
	case "turn.started":
		return "turn started"
	case "turn.completed":
		if usage, ok := event["usage"].(map[string]any); ok {
			return "turn completed" + formatUsage(usage)
		}
		return "turn completed"
	case "error":
		if message := stringField(event, "message"); message != "" {
			return "error: " + message
		}
		return "error"
	case "item.started", "item.completed":
		item, ok := event["item"].(map[string]any)
		if !ok {
			return eventType
		}
		return formatCodexItem(eventType, item)
	default:
		if eventType != "" {
			return eventType
		}
		pretty, err := json.MarshalIndent(event, "", "  ")
		if err != nil {
			return fmt.Sprint(event)
		}
		return string(pretty)
	}
}

func formatCodexItem(eventType string, item map[string]any) string {
	itemType := stringField(item, "type")
	started := eventType == "item.started"
	switch itemType {
	case "agent_message":
		if text := stringField(item, "text"); text != "" {
			return "assistant: " + text
		}
		return "assistant message"
	case "command_execution":
		command := stringField(item, "command")
		if started {
			return "running command: " + command
		}
		parts := []string{"command completed"}
		if status := stringField(item, "status"); status != "" {
			parts[0] += " (" + status + ")"
		}
		if exitCode := stringField(item, "exit_code"); exitCode != "" && exitCode != "<nil>" {
			parts[0] += ": exit " + exitCode
		}
		if command != "" {
			parts = append(parts, "  "+command)
		}
		if output := strings.TrimSpace(stringField(item, "aggregated_output")); output != "" {
			parts = append(parts, "output:\n"+indentLines(output, "  "))
		}
		return strings.Join(parts, "\n")
	case "file_change":
		return formatFileChange(started, item)
	default:
		if itemType == "" {
			return eventType
		}
		if started {
			return itemType + " started"
		}
		return itemType + " completed"
	}
}

func formatFileChange(started bool, item map[string]any) string {
	changes, ok := item["changes"].([]any)
	if !ok || len(changes) == 0 {
		if started {
			return "file change started"
		}
		return "file change completed"
	}
	lines := make([]string, 0, len(changes))
	for _, change := range changes {
		values, ok := change.(map[string]any)
		if !ok {
			continue
		}
		kind := stringField(values, "kind")
		path := stringField(values, "path")
		if kind == "" {
			kind = "change"
		}
		if path == "" {
			lines = append(lines, kind)
			continue
		}
		lines = append(lines, kind+": "+path)
	}
	if len(lines) == 0 {
		if started {
			return "file change started"
		}
		return "file change completed"
	}
	prefix := "file change"
	if started {
		prefix = "file change started"
	} else {
		prefix = "file change completed"
	}
	return prefix + ":\n" + indentLines(strings.Join(lines, "\n"), "  ")
}

func formatUsage(usage map[string]any) string {
	parts := make([]string, 0, 4)
	for _, key := range []string{"input_tokens", "cached_input_tokens", "output_tokens", "reasoning_output_tokens"} {
		if value := stringField(usage, key); value != "" {
			parts = append(parts, strings.TrimSuffix(strings.ReplaceAll(key, "_tokens", ""), "_")+"="+value)
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return " (" + strings.Join(parts, ", ") + ")"
}

func indentLines(value string, prefix string) string {
	lines := strings.Split(value, "\n")
	for i, line := range lines {
		lines[i] = prefix + line
	}
	return strings.Join(lines, "\n")
}
