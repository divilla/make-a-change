package integration_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCLIStartupRebuildsChangeTypeSlugsPrompt(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/options/change-phases-list":
			_ = json.NewEncoder(w).Encode([]map[string]any{{"slug": "backlog"}})
		case "/api/v1/options/change-types-list":
			_ = json.NewEncoder(w).Encode([]map[string]any{{"slug": "fix"}, {"slug": "feature"}})
		case "/api/v1/project/list":
			_ = json.NewEncoder(w).Encode([]any{})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	repo := t.TempDir()
	require.NoError(t, exec.Command("git", "init", repo).Run())
	flowDir := filepath.Join(repo, ".mch", "default")
	promptsDir := filepath.Join(flowDir, "prompts")
	require.NoError(t, os.MkdirAll(promptsDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(repo, ".mch", "config.yaml"), []byte("backend_url: "+server.URL+"\nproject_id: 0\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(flowDir, "flow.yaml"), []byte("version: 1\nslug: default\nhelp: help.yaml\nmakefile: Makefile\nsteps:\n  - slug: def\n    type: edit\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(flowDir, "help.yaml"), []byte("version: 1\n"), 0o644))
	promptPath := filepath.Join(promptsDir, "change-types.md")
	require.NoError(t, os.WriteFile(promptPath, []byte("stale\n"), 0o644))

	binPath := filepath.Join(t.TempDir(), "mch")
	build := exec.Command("go", "build", "-o", binPath, "./cmd/mch")
	build.Dir = repositoryRoot(t) + string(os.PathSeparator) + "cli"
	buildOutput, err := build.CombinedOutput()
	require.NoError(t, err, string(buildOutput))

	outputPath := filepath.Join(t.TempDir(), "mch-output.log")
	output, err := os.Create(outputPath)
	require.NoError(t, err)
	defer output.Close()
	cmd := exec.Command(binPath)
	cmd.Dir = repo
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")
	stdin, err := cmd.StdinPipe()
	require.NoError(t, err)
	cmd.Stdout = output
	cmd.Stderr = output
	require.NoError(t, cmd.Start())
	t.Cleanup(func() {
		if cmd.ProcessState == nil {
			_ = cmd.Process.Kill()
		}
	})
	wait := make(chan error, 1)
	go func() { wait <- cmd.Wait() }()

	want := "# Change Types\n\n- fix\n- feature\n"
	waitForFileContent(t, promptPath, want, wait)
	require.NoError(t, output.Sync())
	assert.Equal(t, want, readFile(t, promptPath))

	_, err = stdin.Write([]byte{3})
	require.NoError(t, err)
	require.NoError(t, stdin.Close())
	select {
	case err := <-wait:
		require.NoError(t, err, readFile(t, outputPath))
	case <-time.After(5 * time.Second):
		require.NoError(t, cmd.Process.Kill())
		t.Fatal("mch did not exit after ctrl+c")
	}
}

func waitForFileContent(t *testing.T, path string, want string, processExit <-chan error) {
	t.Helper()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	timeout := time.NewTimer(10 * time.Second)
	defer timeout.Stop()

	for {
		content, err := os.ReadFile(path)
		if err == nil && string(content) == want {
			return
		}
		select {
		case err := <-processExit:
			t.Fatalf("mch exited before rebuilding %s: %v", path, err)
		case <-ticker.C:
		case <-timeout.C:
			t.Fatalf("timed out waiting for %s to contain the rebuilt Change type slugs", path)
		}
	}
}
