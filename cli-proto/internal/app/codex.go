package app

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

type CodexRunner interface {
	Run(ctx context.Context, req CodexRequest) (CodexResult, error)
}

type CodexRequest struct {
	RepoRoot  string
	Prompt    string
	SessionID string
}

type CodexResult struct {
	Output    string
	SessionID string
}

type CommandCodexRunner struct {
	Timeout time.Duration
}

func (r CommandCodexRunner) Run(ctx context.Context, req CodexRequest) (CodexResult, error) {
	timeout := r.Timeout
	if timeout == 0 {
		timeout = 5 * time.Minute
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	args := []string{"exec", "-C", req.RepoRoot}
	if req.SessionID != "" {
		args = append(args, "resume", req.SessionID)
	}
	args = append(args, "-")
	cmd := exec.CommandContext(ctx, "codex", args...)
	cmd.Stdin = strings.NewReader(req.Prompt)
	var combined bytes.Buffer
	cmd.Stdout = &combined
	cmd.Stderr = &combined
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return CodexResult{Output: combined.String()}, ctx.Err()
		}
		return CodexResult{Output: combined.String()}, err
	}
	output := combined.String()
	sessionID := req.SessionID
	if sessionID == "" {
		var ok bool
		sessionID, ok = ExtractSessionID(output)
		if !ok {
			return CodexResult{Output: output}, errors.New("codex session ID line is missing")
		}
	}
	return CodexResult{Output: output, SessionID: sessionID}, nil
}

var sessionIDPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?im)^\s*session[_ ]id\s*[:=]\s*([A-Za-z0-9._:-]+)\s*$`),
	regexp.MustCompile(`(?i)"session_id"\s*:\s*"([^"]+)"`),
}

func ExtractSessionID(output string) (string, bool) {
	for _, pattern := range sessionIDPatterns {
		matches := pattern.FindStringSubmatch(output)
		if len(matches) == 2 && strings.TrimSpace(matches[1]) != "" {
			return strings.TrimSpace(matches[1]), true
		}
	}
	return "", false
}

func BuildInitialPrompt(template, idea string) string {
	const placeholder = "[Describe the feature/change in 2-5 sentences.]"
	if strings.Contains(template, placeholder) {
		return strings.Replace(template, placeholder, strings.TrimSpace(idea), 1)
	}
	return strings.TrimSpace(template) + "\n\nInitial idea:\n" + strings.TrimSpace(idea) + "\n"
}
