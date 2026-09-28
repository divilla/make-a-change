package briefprocess

import (
	"cli/internal/dto"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func testRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".mch/default/prompts"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".mch/default/prompts/brief-rewrite.md"), []byte("Rewrite only"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".mch/default/prompts/brief-resolve.md"), []byte("Resolve only"), 0o600))
	return root
}

func testRequest(t *testing.T, req dto.BriefRequest) dto.BriefRequest {
	t.Helper()
	base := filepath.Join(req.Root, ".mch", "tmp")
	dir := filepath.Join(base, "brief-owned")
	if _, err := os.Lstat(base); os.IsNotExist(err) {
		require.NoError(t, os.Mkdir(base, 0o700))
	}
	if info, err := os.Lstat(base); err == nil && info.IsDir() && info.Mode()&os.ModeSymlink == 0 {
		var makeErr error
		dir, makeErr = os.MkdirTemp(base, "brief-")
		require.NoError(t, makeErr)
	}
	req.OriginalPath = filepath.Join(dir, "original.md")
	req.InputPath = filepath.Join(dir, "brief.md")
	req.ContextPath = filepath.Join(dir, "context.json")
	req.QuestionsPath = filepath.Join(dir, "questions.json")
	req.AnswersPath = filepath.Join(dir, "answers.json")
	req.OutputPath = filepath.Join(dir, "result.json")
	return req
}

func TestP803PromptSelectionAndStructuredOutput(t *testing.T) {
	root := testRoot(t)
	executable := filepath.Join(t.TempDir(), "codex-stub")
	script := "#!/bin/sh\nfor arg do last=$arg; done\npath=$(printf '%s\\n' \"$last\" | sed -n 's/^Output path: //p')\nprintf '%s' '{\"input_revision\":4,\"rewritten_brief\":\"Clear brief\",\"questions\":[],\"unresolved\":[],\"ready_for_spec\":true}' > \"$path\"\n"
	require.NoError(t, os.WriteFile(executable, []byte(script), 0o700))
	out, dir, err := (Runner{Executable: executable}).Run(context.Background(), testRequest(t, dto.BriefRequest{Root: root, Prompt: "brief-rewrite", Revision: 4, ProjectID: 7, ChangeID: 21, DocumentID: 31, Original: "\tOriginal\n```sh\nrm example\n```\n", Brief: "Current draft", Questions: []dto.BriefQuestion{{ID: "Q1", Text: "Which?", Context: "intro", Answer: " Alpha "}}}))
	require.NoError(t, err)
	require.Equal(t, "Clear brief", out.RewrittenBrief)
	stored, err := os.ReadFile(filepath.Join(dir, "brief.md"))
	require.NoError(t, err)
	require.Equal(t, "Current draft", string(stored))
	stored, err = os.ReadFile(filepath.Join(dir, "original.md"))
	require.NoError(t, err)
	require.Equal(t, "\tOriginal\n```sh\nrm example\n```\n", string(stored))
	stored, err = os.ReadFile(filepath.Join(dir, "answers.json"))
	require.NoError(t, err)
	require.JSONEq(t, `{"Q1":" Alpha "}`, string(stored))
	stored, err = os.ReadFile(filepath.Join(dir, "context.json"))
	require.NoError(t, err)
	require.JSONEq(t, `{"input_revision":4,"project_id":7,"change_id":21,"document_id":31}`, string(stored))
	require.FileExists(t, filepath.Join(dir, "result.json"))
}

func TestP803MissingPromptsAndMalformedOutputStayIncomplete(t *testing.T) {
	root := testRoot(t)
	require.NoError(t, os.Remove(filepath.Join(root, ".mch/default/prompts/brief-resolve.md")))
	_, path, err := (Runner{Executable: "/bin/true"}).Run(context.Background(), testRequest(t, dto.BriefRequest{Root: root, Prompt: "brief-resolve", Revision: 1, Brief: "Text"}))
	require.ErrorContains(t, err, "brief-resolve.md")
	require.DirExists(t, path)
	_, path, err = (Runner{Executable: "/bin/true"}).Run(context.Background(), testRequest(t, dto.BriefRequest{Root: root, Prompt: "brief-rewrite", Revision: 1, Brief: "Text"}))
	require.ErrorContains(t, err, "output missing")
	require.DirExists(t, path)
	executable := filepath.Join(t.TempDir(), "malformed-runner")
	script := "#!/bin/sh\nfor arg do last=$arg; done\npath=$(printf '%s\\n' \"$last\" | sed -n 's/^Output path: //p')\nprintf '%s' '{\"input_revision\":1,\"rewritten_brief\":\"Draft\",\"questions\":[],\"unresolved\":[]}' > \"$path\"\n"
	require.NoError(t, os.WriteFile(executable, []byte(script), 0o700))
	_, path, err = (Runner{Executable: executable}).Run(context.Background(), testRequest(t, dto.BriefRequest{Root: root, Prompt: "brief-rewrite", Revision: 1, Brief: "Text"}))
	require.ErrorContains(t, err, "missing ready_for_spec")
	require.FileExists(t, filepath.Join(path, "result.json"))
}

func TestP803RunnerRejectsDuplicateTopLevelFields(t *testing.T) {
	root := testRoot(t)
	executable := filepath.Join(t.TempDir(), "duplicate-output-runner")
	script := "#!/bin/sh\nfor arg do last=$arg; done\npath=$(printf '%s\\n' \"$last\" | sed -n 's/^Output path: //p')\nprintf '%s' \"$AGENT_OUTPUT\" > \"$path\"\n"
	require.NoError(t, os.WriteFile(executable, []byte(script), 0o700))
	for _, tc := range []struct {
		name, output, field string
	}{
		{name: "contradictory readiness", output: `{"input_revision":1,"rewritten_brief":"Draft","questions":[],"unresolved":[],"ready_for_spec":false,"ready_for_spec":true}`, field: "ready_for_spec"},
		{name: "repeated revision", output: `{"input_revision":1,"input_revision":1,"rewritten_brief":"Draft","questions":[],"unresolved":[],"ready_for_spec":true}`, field: "input_revision"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("AGENT_OUTPUT", tc.output)
			_, dir, err := (Runner{Executable: executable}).Run(context.Background(), testRequest(t, dto.BriefRequest{Root: root, Prompt: "brief-rewrite", Revision: 1, Brief: "Text"}))
			require.ErrorContains(t, err, "duplicate field")
			require.ErrorContains(t, err, tc.field)
			require.FileExists(t, filepath.Join(dir, "result.json"))
		})
	}
}

func TestP803RunnerCancellationProgressAndReaping(t *testing.T) {
	root := testRoot(t)
	executable := filepath.Join(t.TempDir(), "sleeping-runner")
	require.NoError(t, os.WriteFile(executable, []byte("#!/bin/sh\nprintf 'started\\n'\nprintf 'checking context\\n' >&2\nsleep 30\n"), 0o700))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	progress := make(chan string, 1)
	type result struct {
		path string
		err  error
	}
	done := make(chan result, 1)
	start := time.Now()
	go func() {
		_, path, err := (Runner{Executable: executable}).Run(ctx, testRequest(t, dto.BriefRequest{Root: root, Prompt: "brief-rewrite", Revision: 1, Brief: "Text", Progress: progress}))
		done <- result{path: path, err: err}
	}()
	select {
	case message := <-progress:
		require.Contains(t, message, "started")
	case <-time.After(3 * time.Second):
		t.Fatal("no live agent progress")
	}
	cancel()
	var got result
	select {
	case got = <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("agent was not reaped after cancellation")
	}
	require.Error(t, got.err)
	require.Less(t, time.Since(start), 3*time.Second)
	require.True(t, strings.HasPrefix(got.path, filepath.Join(root, ".mch/tmp")))
	require.NoDirExists(t, got.path)
}

func TestP803ProgressCaptureIsBoundedAndNeverBlocks(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	progress := make(chan string, 1)
	w := limitedWriter{ctx: ctx, progress: progress, stream: "agent stderr"}
	chunk := strings.Repeat("x", maxOutput+10)
	done := make(chan struct{})
	go func() {
		_, _ = w.Write([]byte(chunk))
		_, _ = w.Write([]byte("later output"))
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("full progress channel blocked the writer")
	}
	require.Equal(t, maxOutput, w.b.Len())
	require.LessOrEqual(t, len(<-progress), len("agent stderr: ")+256)
	cancel()
	_, _ = w.Write([]byte("after cancellation"))
	require.Empty(t, progress)
}

func TestP803ScratchOwnershipRefusalAndCleanup(t *testing.T) {
	root := testRoot(t)
	base := filepath.Join(root, ".mch/tmp")
	require.NoError(t, os.WriteFile(base, []byte("user file"), 0o600))
	_, _, err := (Runner{}).Run(context.Background(), testRequest(t, dto.BriefRequest{Root: root, Prompt: "brief-rewrite", Revision: 1, Brief: "Text"}))
	require.ErrorContains(t, err, "not an owned directory")
	content, err := os.ReadFile(base)
	require.NoError(t, err)
	require.Equal(t, "user file", string(content))
	require.NoError(t, os.Remove(base))
	outside := t.TempDir()
	require.NoError(t, os.Symlink(outside, base))
	_, _, err = (Runner{}).Run(context.Background(), testRequest(t, dto.BriefRequest{Root: root, Prompt: "brief-rewrite", Revision: 1, Brief: "Text"}))
	require.ErrorContains(t, err, "not an owned directory")
	require.NoError(t, os.Remove(base))
	require.NoError(t, os.MkdirAll(base, 0o700))
	owned, err := os.MkdirTemp(base, "brief-")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(owned, "brief.md"), []byte("input"), 0o600))
	require.NoError(t, CleanupOwned(root, owned))
	require.NoDirExists(t, owned)
	owned, err = os.MkdirTemp(base, "brief-")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(owned, "unrelated"), []byte("keep"), 0o600))
	require.Error(t, CleanupOwned(root, owned))
	require.FileExists(t, filepath.Join(owned, "unrelated"))
}

func TestP803RunnerRejectsSymlinkOutputWithoutTouchingUserFile(t *testing.T) {
	root := testRoot(t)
	victim := filepath.Join(t.TempDir(), "user-document")
	require.NoError(t, os.WriteFile(victim, []byte("keep me"), 0o600))
	executable := filepath.Join(t.TempDir(), "symlink-runner")
	script := "#!/bin/sh\nfor arg do last=$arg; done\npath=$(printf '%s\\n' \"$last\" | sed -n 's/^Output path: //p')\nln -s \"$VICTIM\" \"$path\"\n"
	require.NoError(t, os.WriteFile(executable, []byte(script), 0o700))
	t.Setenv("VICTIM", victim)
	_, dir, err := (Runner{Executable: executable}).Run(context.Background(), testRequest(t, dto.BriefRequest{Root: root, Prompt: "brief-rewrite", Revision: 1, Brief: "Text"}))
	require.ErrorContains(t, err, "not a bounded regular file")
	require.DirExists(t, dir)
	content, err := os.ReadFile(victim)
	require.NoError(t, err)
	require.Equal(t, "keep me", string(content))
}
