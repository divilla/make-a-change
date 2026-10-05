// Package briefprocess supplies Codex processes and owned temporary drafts.
package briefprocess

import (
	"bytes"
	"cli/internal/dto"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/charmbracelet/x/ansi"
)

const maxOutput = 1 << 20

// Runner invokes the fixed Codex commands without a shell.
type Runner struct{ Executable string }

func (r Runner) command(ctx context.Context, args ...string) *exec.Cmd {
	executable := r.Executable
	if executable == "" {
		executable = "codex"
	}
	cmd := exec.CommandContext(ctx, executable, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.WaitDelay = 2 * time.Second
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return os.ErrProcessDone
		}
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
	return cmd
}

// Interactive returns the command for the shell's terminal handoff.
func (r Runner) Interactive(ctx context.Context, args ...string) *exec.Cmd {
	executable := r.Executable
	if executable == "" {
		executable = "codex"
	}
	return InteractiveCommand(ctx, executable, args...)
}

type streamWriter struct {
	mu       sync.Mutex
	buffer   bytes.Buffer
	ctx      context.Context
	progress chan<- string
}

func (w *streamWriter) Write(data []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.buffer.Len() < maxOutput {
		_, _ = w.buffer.Write(data[:min(len(data), maxOutput-w.buffer.Len())])
	}
	if w.progress != nil {
		select {
		case w.progress <- string(data):
		case <-w.ctx.Done():
		}
	}
	return len(data), nil
}

var sessionID = regexp.MustCompile(`(?m)session id:\s*([0-9a-fA-F-]{36})`)

// Exec streams colored output and reads the session's final message separately.
func (r Runner) Exec(ctx context.Context, root, briefPath, prompt string, progress chan<- string) (dto.AgentOutput, error) {
	scratchDir := filepath.Dir(briefPath)
	finalPath := filepath.Join(scratchDir, "final.txt")
	if err := writeExclusive(finalPath, nil); err != nil {
		return dto.AgentOutput{}, err
	}
	// Spec drafts live outside the repository workspace and need explicit write access.
	cmd := r.command(ctx, "exec", "-C", root, "--color", "always", "--output-last-message", finalPath,
		"--sandbox", "workspace-write", "--add-dir", scratchDir, prompt)
	stream := &streamWriter{ctx: ctx, progress: progress}
	cmd.Stdout, cmd.Stderr = stream, stream
	err := cmd.Run()
	output := dto.AgentOutput{}
	if match := sessionID.FindStringSubmatch(ansi.Strip(stream.buffer.String())); len(match) > 1 {
		output.SessionID = match[1]
	}
	if err != nil {
		return output, fmt.Errorf("codex process failed: %w: %s", err, strings.TrimSpace(ansi.Strip(stream.buffer.String())))
	}
	info, err := os.Lstat(finalPath)
	if err != nil {
		return output, err
	}
	if !info.Mode().IsRegular() || info.Size() > maxOutput {
		return output, fmt.Errorf("final message is not a bounded regular file")
	}
	final, err := os.ReadFile(finalPath)
	output.Final = strings.TrimSpace(string(final))
	return output, err
}
