// Package briefprocess runs the fixed brief prompts in an owned scratch directory.
package briefprocess

import (
	"bytes"
	"cli/internal/dto"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

const maxOutput = 1 << 20

// Runner invokes codex with explicit files. Command arguments are never agent text.
type Runner struct{ Executable string }

type limitedWriter struct {
	b        bytes.Buffer
	n        int
	ctx      context.Context
	progress chan<- string
	stream   string
}

func (w *limitedWriter) Write(p []byte) (int, error) {
	if w.n < maxOutput {
		keep := min(len(p), maxOutput-w.n)
		_, _ = w.b.Write(p[:keep])
		w.n += keep
	}
	if w.progress != nil && w.ctx.Err() == nil {
		message := strings.TrimSpace(string(p[:min(len(p), 256)]))
		if message != "" {
			select {
			case <-w.ctx.Done():
			case w.progress <- w.stream + ": " + message:
			default:
			}
		}
	}
	return len(p), nil
}

// Run prepares inputs, invokes the child, validates its file, and retains files on failure.
func (r Runner) Run(ctx context.Context, req dto.BriefRequest) (dto.BriefOutput, string, error) {
	if req.Root == "" || (req.Prompt != "brief-rewrite" && req.Prompt != "brief-resolve") {
		return dto.BriefOutput{}, "", errors.New("invalid brief operation")
	}
	dir, err := requestDirectory(req)
	if err != nil {
		return dto.BriefOutput{}, "", err
	}
	defer func() {
		if ctx.Err() != nil {
			_ = CleanupOwned(req.Root, dir)
		}
	}()
	promptPath := filepath.Join(req.Root, ".mch", "default", "prompts", req.Prompt+".md")
	prompt, err := os.ReadFile(promptPath)
	if err != nil {
		return dto.BriefOutput{}, dir, fmt.Errorf("read %s: %w", promptPath, err)
	}
	originalPath, briefPath, contextPath := req.OriginalPath, req.InputPath, req.ContextPath
	questionsPath, answersPath, outputPath := req.QuestionsPath, req.AnswersPath, req.OutputPath
	for _, path := range []string{originalPath, briefPath, contextPath, questionsPath, answersPath, outputPath} {
		if _, err := os.Lstat(path); err == nil {
			return dto.BriefOutput{}, dir, fmt.Errorf("refuse existing operation file %s", path)
		} else if !errors.Is(err, os.ErrNotExist) {
			return dto.BriefOutput{}, dir, err
		}
	}
	contextData, _ := json.Marshal(map[string]any{"input_revision": req.Revision, "project_id": req.ProjectID, "change_id": req.ChangeID, "document_id": req.DocumentID})
	questions := req.Questions
	if questions == nil {
		questions = []dto.BriefQuestion{}
	}
	questionsData, _ := json.Marshal(questions)
	answers := map[string]string{}
	for _, question := range req.Questions {
		if question.Answer != "" {
			answers[question.ID] = question.Answer
		}
	}
	answersData, _ := json.Marshal(answers)
	for path, data := range map[string][]byte{originalPath: []byte(req.Original), briefPath: []byte(req.Brief), contextPath: contextData, questionsPath: questionsData, answersPath: answersData} {
		if err := writeExclusive(path, data); err != nil {
			return dto.BriefOutput{}, dir, err
		}
	}
	executable := r.Executable
	if executable == "" {
		executable = "codex"
	}
	instruction := string(prompt) + "\n\nInput revision: " + fmt.Sprint(req.Revision) + "\nOriginal user brief path: " + originalPath + "\nCurrent brief path: " + briefPath + "\nContext path: " + contextPath + "\nQuestions path: " + questionsPath + "\nAnswers path: " + answersPath + "\nOutput path: " + outputPath + "\nWrite exactly one JSON object to the output path."
	cmd := exec.CommandContext(ctx, executable, "exec", "--sandbox", "workspace-write", "--ephemeral", "--cd", req.Root, instruction)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.WaitDelay = 2 * time.Second
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return os.ErrProcessDone
		}
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
	stdout := limitedWriter{ctx: ctx, progress: req.Progress, stream: "agent stdout"}
	stderr := limitedWriter{ctx: ctx, progress: req.Progress, stream: "agent stderr"}
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		return dto.BriefOutput{}, dir, fmt.Errorf("brief agent failed: %w: %s", err, strings.TrimSpace(stderr.b.String()))
	}
	info, err := os.Lstat(outputPath)
	if err != nil {
		return dto.BriefOutput{}, dir, fmt.Errorf("agent output missing: %w", err)
	}
	if !info.Mode().IsRegular() || info.Size() > maxOutput {
		return dto.BriefOutput{}, dir, errors.New("agent output is not a bounded regular file")
	}
	file, err := os.Open(outputPath)
	if err != nil {
		return dto.BriefOutput{}, dir, err
	}
	defer func() { _ = file.Close() }()
	decoder := json.NewDecoder(io.LimitReader(file, maxOutput+1))
	start, err := decoder.Token()
	if err != nil {
		return dto.BriefOutput{}, dir, fmt.Errorf("invalid agent output: %w", err)
	}
	if start != json.Delim('{') {
		return dto.BriefOutput{}, dir, errors.New("agent output must be a JSON object")
	}
	raw := make(map[string]json.RawMessage)
	for decoder.More() {
		keyToken, err := decoder.Token()
		if err != nil {
			return dto.BriefOutput{}, dir, fmt.Errorf("invalid agent output: %w", err)
		}
		key := keyToken.(string)
		if _, exists := raw[key]; exists {
			return dto.BriefOutput{}, dir, fmt.Errorf("agent output has duplicate field %q", key)
		}
		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			return dto.BriefOutput{}, dir, fmt.Errorf("invalid agent output: %w", err)
		}
		raw[key] = value
	}
	if _, err := decoder.Token(); err != nil {
		return dto.BriefOutput{}, dir, fmt.Errorf("invalid agent output: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return dto.BriefOutput{}, dir, errors.New("agent output contains trailing data")
	}
	for _, key := range []string{"input_revision", "rewritten_brief", "questions", "unresolved", "ready_for_spec"} {
		if _, ok := raw[key]; !ok {
			return dto.BriefOutput{}, dir, fmt.Errorf("agent output missing %s", key)
		}
	}
	if len(raw) != 5 {
		return dto.BriefOutput{}, dir, errors.New("agent output has unexpected fields")
	}
	ready := strings.TrimSpace(string(raw["ready_for_spec"]))
	if ready != "true" && ready != "false" {
		return dto.BriefOutput{}, dir, errors.New("agent ready_for_spec must be a boolean")
	}
	content, err := json.Marshal(raw)
	if err != nil {
		return dto.BriefOutput{}, dir, err
	}
	decoder = json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	var out dto.BriefOutput
	if err := decoder.Decode(&out); err != nil {
		return dto.BriefOutput{}, dir, fmt.Errorf("invalid agent output: %w", err)
	}
	if out.Questions == nil || out.Unresolved == nil {
		return dto.BriefOutput{}, dir, errors.New("agent questions and unresolved must be arrays")
	}
	return out, dir, nil
}

func requestDirectory(req dto.BriefRequest) (string, error) {
	dir := filepath.Dir(req.InputPath)
	base := filepath.Join(req.Root, ".mch", "tmp")
	rel, err := filepath.Rel(base, dir)
	if req.InputPath == "" || err != nil || filepath.Dir(rel) != "." || !strings.HasPrefix(rel, "brief-") {
		return "", errors.New("brief request has no owned operation paths")
	}
	paths := map[string]string{
		req.OriginalPath: "original.md", req.InputPath: "brief.md", req.ContextPath: "context.json",
		req.QuestionsPath: "questions.json", req.AnswersPath: "answers.json", req.OutputPath: "result.json",
	}
	if len(paths) != 6 {
		return "", errors.New("brief request has incomplete operation paths")
	}
	for path, name := range paths {
		if path != filepath.Join(dir, name) {
			return "", errors.New("brief request has paths outside its operation directory")
		}
	}
	for _, path := range []string{filepath.Join(req.Root, ".mch"), base, dir} {
		info, err := os.Lstat(path)
		if err != nil {
			return "", err
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("scratch path %s is not an owned directory", path)
		}
	}
	return dir, nil
}

func writeExclusive(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	if _, err = f.Write(data); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

// CleanupOwned removes only expected regular files inside an owned operation directory.
func CleanupOwned(root, dir string) error {
	base := filepath.Join(root, ".mch", "tmp")
	for _, path := range []string{filepath.Join(root, ".mch"), base} {
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("refuse unowned scratch ancestor %s", path)
		}
	}
	rel, err := filepath.Rel(base, dir)
	if err != nil || filepath.Dir(rel) != "." || !strings.HasPrefix(rel, "brief-") {
		return fmt.Errorf("refuse unowned scratch directory %s", dir)
	}
	info, err := os.Lstat(dir)
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("refuse non-directory scratch path %s", dir)
	}
	for _, name := range []string{"original.md", "brief.md", "context.json", "questions.json", "answers.json", "result.json"} {
		path := filepath.Join(dir, name)
		info, err := os.Lstat(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("refuse non-regular operation file %s", path)
		}
		if err := os.Remove(path); err != nil {
			return err
		}
	}
	return os.Remove(dir)
}
