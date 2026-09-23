package integration_test

import (
    "encoding/json"
    "io"
    "net/http"
    "net/http/httptest"
    "os"
    "os/exec"
    "path/filepath"
    "strings"
    "sync"
    "testing"
    "time"

    "cli/internal/app"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestCLIProgramExistingSpecUsesUUIDArtifactWorkspaceAndSpecWrite(t *testing.T) {
    runCLIProgramExistingArtifactWrite(t, artifactProgramScenario{
        operation:    "spec-write",
        prompt:       "write spec",
        edited:       "Types: feature\n\nUser-edited spec\n",
        agentWritten: "Types: feature\n\nAgent-written spec\n",
        updatePath:   "/api/v1/change/update-spec",
        command:      "/edit-spec\r",
    })
}

func TestCLIProgramExistingDefinitionUsesUUIDArtifactWorkspaceAndDefWrite(t *testing.T) {
    runCLIProgramExistingArtifactWrite(t, artifactProgramScenario{
        operation:    "def-write",
        prompt:       "rewrite definition",
        edited:       "# Program Change\n\nTypes: feature\n\nUser-edited definition\n",
        agentWritten: "# Program Change\n\nTypes: feature\n\nAgent-written definition\n",
        updatePath:   "/api/v1/change/update-def",
        detailDown:   8,
    })
}

func TestCLIProgramExistingPRUsesUUIDArtifactWorkspaceAndPRWrite(t *testing.T) {
    runCLIProgramExistingArtifactWrite(t, artifactProgramScenario{
        operation:    "pr-write",
        prompt:       "write pull request",
        edited:       "Types: feature\n\nUser-edited PR\n",
        agentWritten: "Types: feature\n\nAgent-written PR\n",
        updatePath:   "/api/v1/change/update-pr",
        detailDown:   10,
    })
}

type artifactProgramScenario struct {
    operation    string
    prompt       string
    edited       string
    agentWritten string
    updatePath   string
    command      string
    detailDown   int
}

func runCLIProgramExistingArtifactWrite(t *testing.T, scenario artifactProgramScenario) {
    t.Helper()
    const refUUID = "0198a86f-9b8a-7d89-ae5b-6f25b528b04c"
    backend := newArtifactProgramBackend(t, refUUID)
    repo := t.TempDir()
    require.NoError(t, exec.Command("git", "init", repo).Run())
    flowDir := filepath.Join(repo, ".mch", "default")
    require.NoError(t, os.MkdirAll(filepath.Join(flowDir, "prompts"), 0o755))
    require.NoError(t, os.WriteFile(filepath.Join(repo, ".mch", "config.yaml"), []byte("backend_url: "+backend.URL+"\nproject_id: 7\n"), 0o644))
    require.NoError(t, os.WriteFile(filepath.Join(flowDir, "flow.yaml"), []byte("version: 1\nslug: default\nhelp: help.yaml\nmakefile: Makefile\nsteps:\n  - slug: "+scenario.operation+"\n    stage: artifact\n    type: exec\n    prompt: prompts/"+scenario.operation+".md\n"), 0o644))
    require.NoError(t, os.WriteFile(filepath.Join(flowDir, "help.yaml"), []byte("version: 1\nstage_modes: []\ntask_statuses: []\ntask_steps: []\n"), 0o644))
    require.NoError(t, os.WriteFile(filepath.Join(flowDir, "prompts", scenario.operation+".md"), []byte(scenario.prompt), 0o644))

    workspace := filepath.Join(repo, ".mch", "tmp", refUUID, "artifact")
    require.NoError(t, os.MkdirAll(workspace, 0o755))
    require.NoError(t, os.WriteFile(filepath.Join(workspace, "session-id"), []byte("22222222-2222-2222-2222-222222222222\n"), 0o644))
    require.NoError(t, os.WriteFile(filepath.Join(workspace, "events.jsonl"), []byte("preserve"), 0o644))

    binDir := filepath.Join(t.TempDir(), "bin")
    require.NoError(t, os.MkdirAll(binDir, 0o755))
    editorPath := filepath.Join(binDir, "editor")
    codexPath := filepath.Join(binDir, "codex")
    invocationPath := filepath.Join(t.TempDir(), "codex-invocation")
    require.NoError(t, os.WriteFile(editorPath, []byte("#!/bin/sh\nprintf '%s' \"$MCH_TEST_EDITOR_TEXT\" > \"$1\"\n"), 0o755))
    require.NoError(t, os.WriteFile(codexPath, []byte("#!/bin/sh\noutput=''\nprevious=''\nfor argument in \"$@\"; do\n  if [ \"$previous\" = '-o' ]; then output=\"$argument\"; fi\n  previous=\"$argument\"\ndone\nworkspace=\"$PWD/$MCH_TEMP_DIR/$MCH_REF_UUID/$MCH_STAGE\"\nprintf '%s|%s|%s\\n' \"$MCH_STAGE\" \"$MCH_REF_UUID\" \"$*\" > \"$MCH_TEST_INVOCATION\"\nprintf '%s' \"$MCH_TEST_AGENT_TEXT\" > \"$workspace/output.md\"\nprintf 'Done.' > \"$output\"\nprintf '{\"type\":\"thread.started\",\"thread_id\":\"33333333-3333-3333-3333-333333333333\"}\\n'\n"), 0o755))

    previous, err := os.Getwd()
    require.NoError(t, err)
    require.NoError(t, os.Chdir(repo))
    t.Cleanup(func() { require.NoError(t, os.Chdir(previous)) })
    t.Setenv("TERM", "xterm-256color")
    t.Setenv("EDITOR", editorPath)
    t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
    t.Setenv("MCH_TEST_INVOCATION", invocationPath)
    t.Setenv("MCH_TEST_EDITOR_TEXT", scenario.edited)
    t.Setenv("MCH_TEST_AGENT_TEXT", scenario.agentWritten)

    inputReader, inputWriter, err := os.Pipe()
    require.NoError(t, err)
    outputPath := filepath.Join(t.TempDir(), "program-output")
    output, err := os.Create(outputPath)
    require.NoError(t, err)
    done := make(chan error, 1)
    go func() { done <- app.RunWithIO(nil, inputReader, output) }()

    waitForProgramOutput(t, outputPath, "MainScreen", done)
    _, err = io.WriteString(inputWriter, "/changes\r")
    require.NoError(t, err)
    waitForProgramOutput(t, outputPath, "ChangesListScreen", done)
    _, err = inputWriter.Write([]byte{'\r'})
    require.NoError(t, err)
    waitForProgramOutput(t, outputPath, "ChangeDetailsScreen", done)
    editOffset := programOutputSize(t, outputPath)
    if scenario.command != "" {
        _, err = io.WriteString(inputWriter, scenario.command)
    } else {
        _, err = io.WriteString(inputWriter, strings.Repeat("\x1b[B", scenario.detailDown)+"\r")
    }
    require.NoError(t, err)
    backend.waitForAgentArtifact(t, outputPath)
    waitForProgramOutputAfter(t, outputPath, "status save", editOffset, done)

    assert.Equal(t, []artifactProgramUpdate{
        {Path: scenario.updatePath, Text: scenario.edited, AgentEdit: false},
        {Path: scenario.updatePath, Text: scenario.agentWritten, AgentEdit: true},
    }, backend.updatesSnapshot())
    assert.Equal(t, scenario.agentWritten, readFile(t, filepath.Join(workspace, "output.md")))
    assert.Equal(t, "preserve", readFile(t, filepath.Join(workspace, "events.jsonl")))
    assert.Equal(t, "22222222-2222-2222-2222-222222222222", strings.TrimSpace(readFile(t, filepath.Join(workspace, "session-id"))))
    for _, stage := range []string{"def", "spec", "pr"} {
        assert.NoDirExists(t, filepath.Join(repo, ".mch", "tmp", refUUID, stage))
    }
    invocation := readFile(t, invocationPath)
    assert.Contains(t, invocation, "artifact|"+refUUID+"|")
    assert.Contains(t, invocation, scenario.operation+".md")
    assert.Contains(t, invocation, "22222222-2222-2222-2222-222222222222")

    returnOffset := programOutputSize(t, outputPath)
    _, err = io.WriteString(inputWriter, "/return\r")
    require.NoError(t, err)
    waitForProgramOutputAfter(t, outputPath, "ChangesListScreen", returnOffset, done)
    mainOffset := programOutputSize(t, outputPath)
    _, err = io.WriteString(inputWriter, "/return\r")
    require.NoError(t, err)
    waitForProgramOutputAfter(t, outputPath, "MainScreen", mainOffset, done)
    _, err = inputWriter.Write([]byte{3})
    require.NoError(t, err)
    require.NoError(t, inputWriter.Close())
    select {
    case err := <-done:
        require.NoError(t, err, readFile(t, outputPath))
    case <-time.After(5 * time.Second):
        t.Fatal("CLI program did not exit")
    }
    require.NoError(t, output.Close())
}

func TestCLIProgramDefReviewUsesDefinitionPromptAndSharedArtifactSession(t *testing.T) {
    invocation := runArtifactFlowTarget(t, "def-review")

    assert.Contains(t, invocation, "artifact|")
    assert.Contains(t, invocation, "exec -C")
    assert.Contains(t, invocation, "resume 22222222-2222-2222-2222-222222222222")
    assert.Contains(t, invocation, "Review the software change definition")
    assert.Contains(t, invocation, "/artifact/input.md")
    assert.Contains(t, invocation, "/artifact/output.md")
}

func TestCLIProgramArtifactChatResumesSharedArtifactSession(t *testing.T) {
    invocation := runArtifactFlowTarget(t, "artifact-chat")

    assert.Contains(t, invocation, "artifact|")
    assert.Contains(t, invocation, "resume 22222222-2222-2222-2222-222222222222")
    assert.NotContains(t, invocation, "def-review.md")
}

func runArtifactFlowTarget(t *testing.T, target string) string {
    t.Helper()
    const (
        refUUID   = "0198a86f-9b8a-7d89-ae5b-6f25b528b04c"
        sessionID = "22222222-2222-2222-2222-222222222222"
    )
    root := repositoryRoot(t)
    repo := t.TempDir()
    require.NoError(t, exec.Command("git", "init", repo).Run())
    workspace := filepath.Join(repo, ".mch", "tmp", refUUID, "artifact")
    require.NoError(t, os.MkdirAll(workspace, 0o755))
    require.NoError(t, os.WriteFile(filepath.Join(workspace, "input.md"), []byte("# Definition\n\nOriginal.\n"), 0o644))
    require.NoError(t, os.WriteFile(filepath.Join(workspace, "output.md"), []byte("# Definition\n\nCurrent.\n"), 0o644))
    require.NoError(t, os.WriteFile(filepath.Join(workspace, "session-id"), []byte(sessionID+"\n"), 0o644))

    codexHome := filepath.Join(repo, "codex-home")
    require.NoError(t, os.MkdirAll(filepath.Join(codexHome, "sessions"), 0o755))
    require.NoError(t, os.WriteFile(filepath.Join(codexHome, "sessions", "rollout-"+sessionID+".jsonl"), []byte("{}\n"), 0o644))
    binDir := filepath.Join(repo, "bin")
    require.NoError(t, os.MkdirAll(binDir, 0o755))
    invocationPath := filepath.Join(repo, "codex-invocation")
    codex := "#!/bin/sh\noutput=''\nprevious=''\nfor argument in \"$@\"; do\n  if [ \"$previous\" = '-o' ]; then output=\"$argument\"; fi\n  previous=\"$argument\"\ndone\nprintf '%s|%s|%s\\n' \"$MCH_STAGE\" \"$MCH_REF_UUID\" \"$*\" > \"$MCH_TEST_INVOCATION\"\nif [ -n \"$output\" ]; then printf 'Done.' > \"$output\"; fi\nprintf '{\"type\":\"thread.started\",\"thread_id\":\"" + sessionID + "\"}\\n'\n"
    require.NoError(t, os.WriteFile(filepath.Join(binDir, "codex"), []byte(codex), 0o755))

    cmd := exec.Command("make", "--no-print-directory", "-f", filepath.Join(root, ".mch", "default", "Makefile"), target)
    cmd.Dir = repo
    cmd.Env = append(os.Environ(),
        "PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"),
        "HOME="+repo,
        "CODEX_HOME="+codexHome,
        "MCH_DEFAULT_DIR=.mch/default",
        "MCH_TEMP_DIR=.mch/tmp",
        "MCH_REF_UUID="+refUUID,
        "MCH_TEST_INVOCATION="+invocationPath,
    )
    output, err := cmd.CombinedOutput()
    require.NoError(t, err, string(output))
    assert.Equal(t, sessionID, strings.TrimSpace(readFile(t, filepath.Join(workspace, "session-id"))))
    return readFile(t, invocationPath)
}

func TestCLIProgramExistingSpecKeepsPersistedArtifactAfterFollowUpFailure(t *testing.T) {
    const refUUID = "0198a86f-9b8a-7d89-ae5b-6f25b528b04c"
    tests := []struct {
        name             string
        failTypeUpdateAt int
        wantSpec         string
        wantUpdates      []artifactProgramUpdate
    }{
        {
            name:             "user save",
            failTypeUpdateAt: 1,
            wantSpec:         "User-edited spec",
            wantUpdates: []artifactProgramUpdate{
                {Path: "/api/v1/change/update-spec", Text: "Types: feature\n\nUser-edited spec\n", AgentEdit: false},
            },
        },
        {
            name:             "agent save",
            failTypeUpdateAt: 2,
            wantSpec:         "Agent-written spec",
            wantUpdates: []artifactProgramUpdate{
                {Path: "/api/v1/change/update-spec", Text: "Types: feature\n\nUser-edited spec\n", AgentEdit: false},
                {Path: "/api/v1/change/update-spec", Text: "Types: feature\n\nAgent-written spec\n", AgentEdit: true},
            },
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            backend := newArtifactProgramBackend(t, refUUID)
            backend.failTypeUpdateAt = tt.failTypeUpdateAt
            repo := t.TempDir()
            require.NoError(t, exec.Command("git", "init", repo).Run())
            flowDir := filepath.Join(repo, ".mch", "default")
            require.NoError(t, os.MkdirAll(filepath.Join(flowDir, "prompts"), 0o755))
            require.NoError(t, os.WriteFile(filepath.Join(repo, ".mch", "config.yaml"), []byte("backend_url: "+backend.URL+"\nproject_id: 7\n"), 0o644))
            require.NoError(t, os.WriteFile(filepath.Join(flowDir, "flow.yaml"), []byte("version: 1\nslug: default\nhelp: help.yaml\nmakefile: Makefile\nsteps:\n  - slug: spec-write\n    stage: artifact\n    type: exec\n    prompt: prompts/spec-write.md\n"), 0o644))
            require.NoError(t, os.WriteFile(filepath.Join(flowDir, "help.yaml"), []byte("version: 1\nstage_modes: []\ntask_statuses: []\ntask_steps: []\n"), 0o644))
            require.NoError(t, os.WriteFile(filepath.Join(flowDir, "prompts", "spec-write.md"), []byte("write spec"), 0o644))

            workspace := filepath.Join(repo, ".mch", "tmp", refUUID, "artifact")
            require.NoError(t, os.MkdirAll(workspace, 0o755))
            require.NoError(t, os.WriteFile(filepath.Join(workspace, "session-id"), []byte("22222222-2222-2222-2222-222222222222\n"), 0o644))

            binDir := filepath.Join(t.TempDir(), "bin")
            require.NoError(t, os.MkdirAll(binDir, 0o755))
            editorPath := filepath.Join(binDir, "editor")
            codexPath := filepath.Join(binDir, "codex")
            require.NoError(t, os.WriteFile(editorPath, []byte("#!/bin/sh\nprintf 'Types: feature\\n\\nUser-edited spec\\n' > \"$1\"\n"), 0o755))
            require.NoError(t, os.WriteFile(codexPath, []byte("#!/bin/sh\noutput=''\nprevious=''\nfor argument in \"$@\"; do\n  if [ \"$previous\" = '-o' ]; then output=\"$argument\"; fi\n  previous=\"$argument\"\ndone\nworkspace=\"$PWD/$MCH_TEMP_DIR/$MCH_REF_UUID/$MCH_STAGE\"\nprintf 'Types: feature\\n\\nAgent-written spec\\n' > \"$workspace/output.md\"\nprintf 'Done.' > \"$output\"\nprintf '{\"type\":\"thread.started\",\"thread_id\":\"33333333-3333-3333-3333-333333333333\"}\\n'\n"), 0o755))

            previous, err := os.Getwd()
            require.NoError(t, err)
            require.NoError(t, os.Chdir(repo))
            t.Cleanup(func() { require.NoError(t, os.Chdir(previous)) })
            t.Setenv("TERM", "xterm-256color")
            t.Setenv("EDITOR", editorPath)
            t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

            inputReader, inputWriter, err := os.Pipe()
            require.NoError(t, err)
            outputPath := filepath.Join(t.TempDir(), "program-output")
            output, err := os.Create(outputPath)
            require.NoError(t, err)
            done := make(chan error, 1)
            go func() { done <- app.RunWithIO(nil, inputReader, output) }()

            waitForProgramOutput(t, outputPath, "MainScreen", done)
            _, err = io.WriteString(inputWriter, "/changes\r")
            require.NoError(t, err)
            waitForProgramOutput(t, outputPath, "ChangesListScreen", done)
            _, err = inputWriter.Write([]byte{'\r'})
            require.NoError(t, err)
            waitForProgramOutput(t, outputPath, "ChangeDetailsScreen", done)
            editOffset := programOutputSize(t, outputPath)
            _, err = io.WriteString(inputWriter, "/edit-spec\r")
            require.NoError(t, err)
            waitForProgramOutputAfter(t, outputPath, "save failed", editOffset, done)
            detailOffset := programOutputSize(t, outputPath)
            _, err = io.WriteString(inputWriter, strings.Repeat("\x1b[B", 9))
            require.NoError(t, err)
            waitForProgramOutputAfter(t, outputPath, tt.wantSpec, detailOffset, done)

            assert.Equal(t, tt.wantUpdates, backend.updatesSnapshot())
            assert.Equal(t, tt.failTypeUpdateAt, backend.typeUpdateCount())

            returnOffset := programOutputSize(t, outputPath)
            _, err = inputWriter.Write([]byte{3})
            require.NoError(t, err)
            waitForProgramOutputAfter(t, outputPath, "ChangesListScreen", returnOffset, done)
            mainOffset := programOutputSize(t, outputPath)
            _, err = inputWriter.Write([]byte{3})
            require.NoError(t, err)
            waitForProgramOutputAfter(t, outputPath, "MainScreen", mainOffset, done)
            _, err = inputWriter.Write([]byte{3})
            require.NoError(t, err)
            require.NoError(t, inputWriter.Close())
            select {
            case err := <-done:
                require.NoError(t, err, readFile(t, outputPath))
            case <-time.After(5 * time.Second):
                t.Fatal("CLI program did not exit")
            }
            require.NoError(t, output.Close())
        })
    }
}

type artifactProgramUpdate struct {
    Path      string
    Text      string
    AgentEdit bool
}

type artifactProgramBackend struct {
    *httptest.Server
    mu               sync.Mutex
    refUUID          string
    def              string
    spec             string
    pr               string
    updates          []artifactProgramUpdate
    typeUpdates      int
    failTypeUpdateAt int
}

func newArtifactProgramBackend(t *testing.T, refUUID string) *artifactProgramBackend {
    t.Helper()
    backend := &artifactProgramBackend{
        refUUID: refUUID,
        def:     "# Program Change\n\nOriginal definition",
        spec:    "Original spec",
        pr:      "Original PR",
    }
    backend.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        var payload map[string]any
        _ = json.NewDecoder(r.Body).Decode(&payload)
        switch r.URL.Path {
        case "/api/v1/options/change-phases-list":
            _ = json.NewEncoder(w).Encode([]map[string]any{{"slug": "backlog"}})
        case "/api/v1/options/change-types-list":
            _ = json.NewEncoder(w).Encode([]map[string]any{{"slug": "feature"}})
        case "/api/v1/project/get":
            _ = json.NewEncoder(w).Encode(map[string]any{"id": 7, "name": "Program Project"})
        case "/api/v1/change/list":
            _ = json.NewEncoder(w).Encode(map[string]any{"changes": []map[string]any{{"id": 12, "ref_uuid": refUUID, "ref": 3, "title": "Program Change", "spec": backend.currentSpec()}}})
        case "/api/v1/change/get":
            def, spec, pr := backend.currentArtifacts()
            _ = json.NewEncoder(w).Encode(map[string]any{"change": map[string]any{"id": 12, "project_id": 7, "ref_uuid": refUUID, "ref": 3, "title": "Program Change", "def": def, "spec": spec, "pr": pr}, "test_cases": []any{}})
        case "/api/v1/change/update-def":
            def, _ := payload["def"].(string)
            agentEdit, _ := payload["agent_edit"].(bool)
            backend.recordUpdate(r.URL.Path, def, agentEdit)
            _ = json.NewEncoder(w).Encode(map[string]any{"id": 12, "ref_uuid": refUUID, "def": def, "agent_edit": agentEdit})
        case "/api/v1/change/update-spec":
            spec, _ := payload["spec"].(string)
            agentEdit, _ := payload["agent_edit"].(bool)
            backend.recordUpdate(r.URL.Path, spec, agentEdit)
            _ = json.NewEncoder(w).Encode(map[string]any{"id": 12, "ref_uuid": refUUID, "spec": spec, "agent_edit": agentEdit})
        case "/api/v1/change/update-pr":
            pr, _ := payload["pr"].(string)
            agentEdit, _ := payload["agent_edit"].(bool)
            backend.recordUpdate(r.URL.Path, pr, agentEdit)
            _ = json.NewEncoder(w).Encode(map[string]any{"id": 12, "ref_uuid": refUUID, "pr": pr, "agent_edit": agentEdit})
        case "/api/v1/change/update-change-types":
            backend.mu.Lock()
            backend.typeUpdates++
            fail := backend.failTypeUpdateAt == backend.typeUpdates
            backend.mu.Unlock()
            if fail {
                http.Error(w, "type update failed", http.StatusInternalServerError)
                return
            }
            _ = json.NewEncoder(w).Encode(map[string]any{"id": 12, "change_types": payload["change_types"]})
        default:
            http.NotFound(w, r)
        }
    }))
    t.Cleanup(backend.Close)
    return backend
}

func (b *artifactProgramBackend) currentSpec() string {
    b.mu.Lock()
    defer b.mu.Unlock()
    return b.spec
}

func (b *artifactProgramBackend) currentArtifacts() (string, string, string) {
    b.mu.Lock()
    defer b.mu.Unlock()
    return b.def, b.spec, b.pr
}

func (b *artifactProgramBackend) recordUpdate(path string, text string, agentEdit bool) {
    b.mu.Lock()
    defer b.mu.Unlock()
    switch path {
    case "/api/v1/change/update-def":
        b.def = text
    case "/api/v1/change/update-spec":
        b.spec = text
    case "/api/v1/change/update-pr":
        b.pr = text
    }
    b.updates = append(b.updates, artifactProgramUpdate{Path: path, Text: text, AgentEdit: agentEdit})
}

func (b *artifactProgramBackend) updatesSnapshot() []artifactProgramUpdate {
    b.mu.Lock()
    defer b.mu.Unlock()
    return append([]artifactProgramUpdate(nil), b.updates...)
}

func (b *artifactProgramBackend) typeUpdateCount() int {
    b.mu.Lock()
    defer b.mu.Unlock()
    return b.typeUpdates
}

func (b *artifactProgramBackend) waitForAgentArtifact(t *testing.T, outputPath string) {
    t.Helper()
    deadline := time.Now().Add(10 * time.Second)
    for time.Now().Before(deadline) {
        updates := b.updatesSnapshot()
        if len(updates) == 2 && updates[1].AgentEdit {
            return
        }
        time.Sleep(10 * time.Millisecond)
    }
    output, _ := os.ReadFile(outputPath)
    t.Fatalf("timed out waiting for user and agent artifact updates: %#v\n%s", b.updatesSnapshot(), output)
}

func waitForProgramOutput(t *testing.T, path, marker string, done <-chan error) {
    t.Helper()
    deadline := time.Now().Add(10 * time.Second)
    for time.Now().Before(deadline) {
        content, _ := os.ReadFile(path)
        if strings.Contains(string(content), marker) {
            return
        }
        select {
        case err := <-done:
            t.Fatalf("CLI exited before %q: %v\n%s", marker, err, content)
        default:
        }
        time.Sleep(10 * time.Millisecond)
    }
    t.Fatalf("timed out waiting for %q in %s", marker, path)
}

func waitForProgramOutputAfter(t *testing.T, path, marker string, offset int64, done <-chan error) {
    t.Helper()
    deadline := time.Now().Add(10 * time.Second)
    for time.Now().Before(deadline) {
        content, _ := os.ReadFile(path)
        if offset < int64(len(content)) && strings.Contains(string(content[offset:]), marker) {
            return
        }
        select {
        case err := <-done:
            t.Fatalf("CLI exited before new %q: %v\n%s", marker, err, content)
        default:
        }
        time.Sleep(10 * time.Millisecond)
    }
    t.Fatalf("timed out waiting for new %q in %s", marker, path)
}

func programOutputSize(t *testing.T, path string) int64 {
    t.Helper()
    info, err := os.Stat(path)
    require.NoError(t, err)
    return info.Size()
}
