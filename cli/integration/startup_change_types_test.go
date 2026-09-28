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

func TestCLIStartupWithoutFlowResources(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
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
	require.NoError(t, os.MkdirAll(filepath.Join(repo, ".mch"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(repo, ".mch", "config.yaml"), []byte("backend_url: "+server.URL+"\nproject_id: 0\n"), 0o644))

	binPath := os.Getenv("MCH_COVER_BINARY")
	if binPath == "" {
		binPath = filepath.Join(t.TempDir(), "mch")
		build := exec.Command("go", "build", "-o", binPath, "./cmd/mch")
		build.Dir = repositoryRoot(t) + string(os.PathSeparator) + "cli"
		buildOutput, err := build.CombinedOutput()
		require.NoError(t, err, string(buildOutput))
	}

	outputPath := filepath.Join(t.TempDir(), "mch-output.log")
	output, err := os.Create(outputPath)
	require.NoError(t, err)
	defer func() { require.NoError(t, output.Close()) }()
	cmd := exec.Command(binPath)
	cmd.Dir = repo
	cmd.Env = append(os.Environ(), "TERM=xterm-256color", "GOCOVERDIR="+os.Getenv("MCH_COVER_DIR"))
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

	waitForProgramOutput(t, outputPath, "No projects to select from", wait)
	assert.NoDirExists(t, flowDir)
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
