package integration_test

import (
    "bytes"
    "context"
    "encoding/json"
    "errors"
    "fmt"
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

    "cli/internal/agent"
    "cli/internal/app"

    "github.com/gofrs/uuid/v5"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

const programRefUUID = "0198a86f-9b8a-7d89-ae5b-6f25b528b04c"

func TestCLIProgramNewChangeScenarios(t *testing.T) {
    def := "# Program Change\n\nTypes: feature\n\nInitial definition"

    t.Run("config ignores legacy temp and creates UUID stage files through editor handoff", func(t *testing.T) {
        backend := newProgramBackend(t)
        root := t.TempDir()
        legacy := filepath.Join(t.TempDir(), "legacy")
        writeProgramConfig(t, root, backend.URL, legacy)
        session := startProgram(t, root, programRefUUID, &programRunner{root: root}, def)

        session.openNewChange(t)
        session.waitFor(t, "Create Change?")

        workspace := programWorkspace(root, programRefUUID)
        require.FileExists(t, filepath.Join(workspace, "input.md"))
        require.FileExists(t, filepath.Join(workspace, "output.md"))
        assert.Equal(t, def, readProgramFile(t, filepath.Join(workspace, "output.md")))
        assert.NoDirExists(t, legacy)
        session.selectSecondAndFinishFromChanges(t)
    })

    t.Run("unchanged and no remove workspace without create", func(t *testing.T) {
        for _, scenario := range []struct {
            name   string
            output string
            no     bool
        }{{name: "unchanged"}, {name: "no", output: def, no: true}} {
            t.Run(scenario.name, func(t *testing.T) {
                backend := newProgramBackend(t)
                root := t.TempDir()
                writeProgramConfig(t, root, backend.URL, "")
                session := startProgram(t, root, programRefUUID, &programRunner{root: root}, scenario.output)

                changesBefore := session.openNewChange(t)
                if scenario.no {
                    session.waitFor(t, "Create Change?")
                    session.selectSecond(t)
                }
                session.output.waitForCount(t, "ChangesListScreen", changesBefore+1)

                assert.NoDirExists(t, filepath.Join(root, agent.TempDir, programRefUUID))
                assert.Zero(t, backend.createCount())
                session.finishFromChanges(t)
            })
        }
    })

    t.Run("title and body validate only after yes", func(t *testing.T) {
        for _, invalid := range []string{
            "missing title",
            "# Bad\n\nTypes:",
        } {
            backend := newProgramBackend(t)
            root := t.TempDir()
            writeProgramConfig(t, root, backend.URL, "")
            session := startProgram(t, root, programRefUUID, &programRunner{root: root}, invalid)

            session.openNewChange(t)
            session.waitFor(t, "Create Change?")
            assert.NotContains(t, session.output.String(), "/fix")
            session.selectFirst(t)
            session.waitFor(t, "/fix")

            workspace := programWorkspace(root, programRefUUID)
            require.FileExists(t, filepath.Join(workspace, "input.md"))
            require.FileExists(t, filepath.Join(workspace, "output.md"))
            assert.Zero(t, backend.createCount())
            session.selectSecondAndFinishFromChanges(t)
        }
    })

    t.Run("def validation does not require spec layout", func(t *testing.T) {
        for _, scenario := range []struct {
            name            string
            def             string
            wantTypeUpdates int
        }{
            {name: "leading blank before title", def: "\n# Program Change\n\nInitial definition"},
            {name: "no blank after title", def: "# Program Change\nTypes: feature\n\nInitial definition", wantTypeUpdates: 1},
        } {
            t.Run(scenario.name, func(t *testing.T) {
                backend := newProgramBackend(t)
                root := t.TempDir()
                writeProgramConfig(t, root, backend.URL, "")
                runner := &programRunner{root: root, rewritten: "# Program Change\n\nRewritten definition"}
                session := startProgram(t, root, programRefUUID, runner, scenario.def)

                session.openNewChange(t)
                session.waitFor(t, "Create Change?")
                session.selectFirst(t)
                session.waitFor(t, "ChangeDetailsScreen")

                assert.Equal(t, 1, backend.createCount())
                assert.Equal(t, scenario.wantTypeUpdates, backend.typeUpdateCount())
                session.finishFromDetails(t)
            })
        }
    })

    t.Run("epic and artifact types bypass extra option requests", func(t *testing.T) {
        backend := newProgramBackend(t)
        root := t.TempDir()
        writeProgramConfig(t, root, backend.URL, "")
        artifact := "# Ordered Change\n\nTypes: unsupported!\n\nEpic: Epic Five\n\nInitial definition"
        session := startProgram(t, root, programRefUUID, &programRunner{root: root}, artifact)

        session.openNewChange(t)
        session.waitFor(t, "Create Change?")

        assert.Equal(t, 1, backend.typeCatalogCount(), "startup is the only type-catalog caller")
        assert.Zero(t, backend.epicCount())
        assert.Zero(t, backend.createCount())
        session.selectSecondAndFinishFromChanges(t)
    })

    t.Run("present types create update and rewrite use complete program sequencing", func(t *testing.T) {
        backend := newProgramBackend(t)
        root := t.TempDir()
        writeProgramConfig(t, root, backend.URL, "")
        runner := &programRunner{root: root, rewritten: "# Typed Change\n\nTypes: feature\n\nRewritten definition"}
        typedDef := "# Typed Change\n\nTypes: fix|feature|unsupported!\n\nInitial definition"
        session := startProgram(t, root, programRefUUID, runner, typedDef)

        session.openNewChange(t)
        session.waitFor(t, "Create Change?")
        session.selectFirst(t)
        session.waitFor(t, "ChangeDetailsScreen")

        assert.Equal(t, [][]string{{"fix", "feature", "unsupported"}, {"feature"}}, backend.typeUpdatesCopy())
        assert.Equal(t, []string{
            "/api/v1/change/create",
            "/api/v1/change/update-change-types",
            "/api/v1/change/update-def",
            "/api/v1/change/update-change-types",
            "/api/v1/change/get",
        }, backend.changeMutationPaths())
        assert.Equal(t, 1, backend.typeCatalogCount())
        session.finishFromDetails(t)
    })

    t.Run("post-create type failure recovers as existing change", func(t *testing.T) {
        for _, action := range []string{"fix unchanged", "fix invalid then cancel", "cancel"} {
            t.Run(action, func(t *testing.T) {
                backend := newProgramBackend(t)
                backend.failTypeUpdate = true
                root := t.TempDir()
                writeProgramConfig(t, root, backend.URL, "")
                session := startProgram(t, root, programRefUUID, &programRunner{root: root}, def)

                session.openNewChange(t)
                session.waitFor(t, "Create Change?")
                session.selectFirst(t)
                session.waitFor(t, "/fix")
                assert.Contains(t, session.output.String(), "ChangeDetailsScreen")

                switch action {
                case "fix unchanged":
                    session.setEditorOutput(t, def)
                    session.selectFirst(t)
                    session.output.waitForCount(t, "ChangeDetailsScreen", 2)
                case "fix invalid then cancel":
                    session.setEditorOutput(t, "missing title")
                    session.selectFirst(t)
                    session.waitFor(t, "definition title is required")
                    session.selectSecond(t)
                default:
                    session.selectSecond(t)
                }

                assert.Equal(t, 1, backend.createCount())
                assert.Zero(t, backend.defUpdateCount())
                require.DirExists(t, filepath.Join(root, agent.TempDir, programRefUUID))
                session.finishFromDetails(t)
            })
        }
    })

    t.Run("omitted types skip type updates", func(t *testing.T) {
        backend := newProgramBackend(t)
        root := t.TempDir()
        writeProgramConfig(t, root, backend.URL, "")
        artifact := "# No Typed Change\n\nInitial definition"
        runner := &programRunner{root: root, rewritten: "# No Typed Change\n\nRewritten definition"}
        session := startProgram(t, root, programRefUUID, runner, artifact)

        session.openNewChange(t)
        session.waitFor(t, "Create Change?")
        session.selectFirst(t)
        session.waitFor(t, "ChangeDetailsScreen")

        assert.Zero(t, backend.typeUpdateCount())
        assert.Equal(t, 1, backend.typeCatalogCount())
        session.finishFromDetails(t)
    })

    t.Run("fix requires another editor pass and unchanged cancels", func(t *testing.T) {
        backend := newProgramBackend(t)
        root := t.TempDir()
        writeProgramConfig(t, root, backend.URL, "")
        invalid := "missing title"
        session := startProgram(t, root, programRefUUID, &programRunner{root: root}, invalid)

        session.openNewChange(t)
        session.waitFor(t, "Create Change?")
        session.selectFirst(t)
        session.waitFor(t, "/fix")
        session.setEditorOutput(t, invalid)
        changesBefore := session.output.count("ChangesListScreen")
        session.selectFirst(t)
        session.output.waitForCount(t, "ChangesListScreen", changesBefore+1)

        assert.NoDirExists(t, filepath.Join(root, agent.TempDir, programRefUUID))
        assert.Zero(t, backend.createCount())
        session.finishFromChanges(t)
    })

    t.Run("yes preserves identity and completes rewrite", func(t *testing.T) {
        backend := newProgramBackend(t)
        root := t.TempDir()
        writeProgramConfig(t, root, backend.URL, "")
        runner := &programRunner{root: root, rewritten: "# Program Change\n\nTypes: feature\n\nRewritten definition"}
        session := startProgram(t, root, programRefUUID, runner, def)

        session.openNewChange(t)
        session.waitFor(t, "Create Change?")
        session.selectFirst(t)
        session.waitFor(t, "ChangeDetailsScreen")

        workspace := programWorkspace(root, programRefUUID)
        assert.Equal(t, programRefUUID, backend.lastRefUUID())
        assert.Equal(t, def, readProgramFile(t, filepath.Join(workspace, "input.md")))
        assert.Equal(t, runner.rewritten, readProgramFile(t, filepath.Join(workspace, "output.md")))
        require.DirExists(t, workspace)
        assert.Equal(t, map[string]string{
            "MCH_DEFAULT_DIR": agent.DefaultDir,
            "MCH_TEMP_DIR":    agent.TempDir,
            "MCH_REF_UUID":    programRefUUID,
            "MCH_STAGE":       agent.ArtifactStage,
        }, runner.environment())
        session.finishFromDetails(t)
    })

    t.Run("UUID and workspace failures stop before request", func(t *testing.T) {
        t.Run("UUID", func(t *testing.T) {
            backend := newProgramBackend(t)
            root := t.TempDir()
            writeProgramConfig(t, root, backend.URL, "")
            session := startProgramWithUUID(t, root, func() (uuid.UUID, error) {
                return uuid.Nil, errors.New("uuid failed")
            }, &programRunner{root: root}, "")

            session.openNewChange(t)
            session.waitFor(t, "uuid failed")
            assert.NoDirExists(t, filepath.Join(root, agent.TempDir))
            assert.Zero(t, backend.createCount())
            session.finishFromChanges(t)
        })

        t.Run("directory", func(t *testing.T) {
            backend := newProgramBackend(t)
            root := t.TempDir()
            writeProgramConfig(t, root, backend.URL, "")
            require.NoError(t, os.WriteFile(filepath.Join(root, agent.TempDir), []byte("blocked"), 0o644))
            session := startProgram(t, root, programRefUUID, &programRunner{root: root}, "")

            session.openNewChange(t)
            session.waitFor(t, "not a directory")
            assert.Zero(t, backend.createCount())
            session.finishFromChanges(t)
        })

        t.Run("file collision", func(t *testing.T) {
            backend := newProgramBackend(t)
            root := t.TempDir()
            writeProgramConfig(t, root, backend.URL, "")
            stageDir := programWorkspace(root, programRefUUID)
            require.NoError(t, os.MkdirAll(stageDir, 0o755))
            require.NoError(t, os.WriteFile(filepath.Join(stageDir, "input.md"), []byte("collision"), 0o644))
            session := startProgram(t, root, programRefUUID, &programRunner{root: root}, "")

            session.openNewChange(t)
            session.waitFor(t, "file exists")
            assert.NoDirExists(t, filepath.Join(root, agent.TempDir, programRefUUID))
            assert.Zero(t, backend.createCount())
            session.finishFromChanges(t)
        })
    })

    t.Run("def agent failure remains visible in the complete program", func(t *testing.T) {
        backend := newProgramBackend(t)
        root := t.TempDir()
        writeProgramConfig(t, root, backend.URL, "")
        runner := &programRunner{root: root, err: errors.New("codex failed")}
        session := startProgram(t, root, programRefUUID, runner, def)

        session.openNewChange(t)
        session.waitFor(t, "Create Change?")
        session.selectFirst(t)
        session.waitFor(t, "codex failed")

        assert.Contains(t, session.output.String(), "AgentRunningScreen")
        assert.Equal(t, agent.ArtifactStage, runner.environment()["MCH_STAGE"])
        session.stop(t)
    })

    t.Run("API failure preserves files and offers fix or cancel", func(t *testing.T) {
        backend := newProgramBackend(t)
        backend.failCreate = true
        root := t.TempDir()
        writeProgramConfig(t, root, backend.URL, "")
        session := startProgram(t, root, programRefUUID, &programRunner{root: root}, def)

        session.openNewChange(t)
        session.waitFor(t, "Create Change?")
        session.selectFirst(t)
        session.waitFor(t, "/fix")

        workspace := programWorkspace(root, programRefUUID)
        require.FileExists(t, filepath.Join(workspace, "input.md"))
        require.FileExists(t, filepath.Join(workspace, "output.md"))
        assert.NotContains(t, session.output.String(), "ChangeDetailsScreen")
        session.selectSecondAndFinishFromChanges(t)
    })

    t.Run("existing workspace remains untouched", func(t *testing.T) {
        backend := newProgramBackend(t)
        root := t.TempDir()
        writeProgramConfig(t, root, backend.URL, "")
        old := programWorkspace(root, "11111111-1111-1111-1111-111111111111")
        require.NoError(t, os.MkdirAll(old, 0o755))
        require.NoError(t, os.WriteFile(filepath.Join(old, "input.md"), []byte("old input"), 0o644))
        require.NoError(t, os.WriteFile(filepath.Join(old, "output.md"), []byte("old output"), 0o644))
        session := startProgram(t, root, programRefUUID, &programRunner{root: root}, "")

        changesBefore := session.openNewChange(t)
        session.output.waitForCount(t, "ChangesListScreen", changesBefore+1)

        assert.Equal(t, "old input", readProgramFile(t, filepath.Join(old, "input.md")))
        assert.Equal(t, "old output", readProgramFile(t, filepath.Join(old, "output.md")))
        session.finishFromChanges(t)
    })
}

type programSession struct {
    input        *os.File
    output       *synchronizedBuffer
    done         chan error
    cancel       context.CancelFunc
    editorSource string
    controller   chan app.ProgramController
}

func startProgram(t *testing.T, root, refUUID string, runner agent.Runner, editorOutput string) *programSession {
    t.Helper()
    parsed, err := uuid.FromString(refUUID)
    require.NoError(t, err)
    return startProgramWithUUID(t, root, func() (uuid.UUID, error) { return parsed, nil }, runner, editorOutput)
}

func startProgramWithUUID(t *testing.T, root string, newUUID func() (uuid.UUID, error), runner agent.Runner, editorOutput string) *programSession {
    t.Helper()
    reader, writer, err := os.Pipe()
    require.NoError(t, err)
    output := newSynchronizedBuffer()
    ctx, cancel := context.WithCancel(context.Background())
    editorSource := filepath.Join(t.TempDir(), "editor-output.md")
    require.NoError(t, os.WriteFile(editorSource, []byte(editorOutput), 0o644))
    t.Setenv("EDITOR", "cp "+editorSource)
    session := &programSession{
        input: writer, output: output, done: make(chan error, 1), cancel: cancel,
        editorSource: editorSource, controller: make(chan app.ProgramController, 1),
    }
    go func() {
        session.done <- app.RunProgramWithIO(nil, reader, output, app.ProgramOptions{
            Context:        ctx,
            RepositoryRoot: root,
            NewChangeUUID:  newUUID,
            AgentRunner:    runner,
            ProgramReady: func(controller app.ProgramController) {
                session.controller <- controller
            },
        })
    }()
    t.Cleanup(func() {
        cancel()
        _ = writer.Close()
    })
    output.waitFor(t, "MainScreen")
    return session
}

func (s *programSession) openNewChange(t *testing.T) int {
    t.Helper()
    changesBefore := s.output.count("ChangesListScreen")
    s.send(t, "/changes\r")
    s.output.waitForCount(t, "ChangesListScreen", changesBefore+1)
    changesAfterNavigation := s.output.count("ChangesListScreen")
    s.send(t, "\x0e")
    return changesAfterNavigation
}

func (s *programSession) setEditorOutput(t *testing.T, content string) {
    t.Helper()
    require.NoError(t, os.WriteFile(s.editorSource, []byte(content), 0o644))
}

func (s *programSession) selectFirst(t *testing.T) {
    t.Helper()
    s.send(t, "\r")
}

func (s *programSession) selectSecond(t *testing.T) {
    t.Helper()
    s.send(t, "\x1b[B\r")
}

func (s *programSession) selectSecondAndFinishFromChanges(t *testing.T) {
    t.Helper()
    changesBefore := s.output.count("ChangesListScreen")
    s.selectSecond(t)
    s.output.waitForCount(t, "ChangesListScreen", changesBefore+1)
    s.finishFromChanges(t)
}

func (s *programSession) finishFromDetails(t *testing.T) {
    t.Helper()
    changesBefore := s.output.count("ChangesListScreen")
    s.send(t, "\x03")
    s.output.waitForCount(t, "ChangesListScreen", changesBefore+1)
    s.finishFromChanges(t)
}

func (s *programSession) finishFromChanges(t *testing.T) {
    t.Helper()
    mainBefore := s.output.count("MainScreen")
    s.send(t, "\x03")
    s.output.waitForCount(t, "MainScreen", mainBefore+1)
    s.send(t, "\x03")
    require.NoError(t, s.waitDone(t))
}

func (s *programSession) stop(t *testing.T) {
    t.Helper()
    select {
    case controller := <-s.controller:
        controller.Quit()
    case <-time.After(5 * time.Second):
        require.FailNow(t, "CLI program controller was not ready")
    }
    require.NoError(t, s.waitDone(t))
}

func (s *programSession) waitFor(t *testing.T, marker string) {
    t.Helper()
    s.output.waitFor(t, marker)
}

func (s *programSession) send(t *testing.T, keys string) {
    t.Helper()
    _, err := io.WriteString(s.input, keys)
    require.NoError(t, err)
}

func (s *programSession) waitDone(t *testing.T) error {
    t.Helper()
    select {
    case err := <-s.done:
        return err
    case <-time.After(5 * time.Second):
        require.Fail(t, "CLI program did not exit")
        return nil
    }
}

type synchronizedBuffer struct {
    mu      sync.Mutex
    buffer  bytes.Buffer
    changed chan struct{}
}

func newSynchronizedBuffer() *synchronizedBuffer {
    return &synchronizedBuffer{changed: make(chan struct{}, 1)}
}

func (b *synchronizedBuffer) Write(p []byte) (int, error) {
    b.mu.Lock()
    n, err := b.buffer.Write(p)
    b.mu.Unlock()
    select {
    case b.changed <- struct{}{}:
    default:
    }
    return n, err
}

func (b *synchronizedBuffer) String() string {
    b.mu.Lock()
    defer b.mu.Unlock()
    return b.buffer.String()
}

func (b *synchronizedBuffer) count(marker string) int {
    return strings.Count(b.String(), marker)
}

func (b *synchronizedBuffer) waitFor(t *testing.T, marker string) {
    t.Helper()
    b.waitForCount(t, marker, 1)
}

func (b *synchronizedBuffer) waitForCount(t *testing.T, marker string, count int) {
    t.Helper()
    timer := time.NewTimer(5 * time.Second)
    defer timer.Stop()
    for b.count(marker) < count {
        select {
        case <-b.changed:
        case <-timer.C:
            require.FailNow(t, "timed out waiting for CLI output", "marker %q count %d; output:\n%s", marker, count, b.String())
        }
    }
}

type programBackend struct {
    *httptest.Server
    mu             sync.Mutex
    creates        []map[string]any
    typeUpdates    [][]string
    typeCatalogs   int
    defUpdates     int
    paths          []string
    currentDef     string
    failCreate     bool
    failTypeUpdate bool
    epics          int
}

func newProgramBackend(t *testing.T) *programBackend {
    t.Helper()
    b := &programBackend{}
    b.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        var payload map[string]any
        _ = json.NewDecoder(r.Body).Decode(&payload)
        w.Header().Set("Content-Type", "application/json")
        b.recordPath(r.URL.Path)
        switch r.URL.Path {
        case "/api/v1/options/change-phases-list":
            writeProgramJSON(w, map[string]any{"phases": []map[string]any{{"slug": "backlog"}}})
        case "/api/v1/options/change-types-list":
            b.mu.Lock()
            b.typeCatalogs++
            b.mu.Unlock()
            writeProgramJSON(w, map[string]any{"types": []map[string]any{{"slug": "feature"}}})
        case "/api/v1/project/get":
            writeProgramJSON(w, map[string]any{"project": map[string]any{"id": 7, "name": "Program Project"}})
        case "/api/v1/change/list":
            writeProgramJSON(w, map[string]any{"changes": []any{}})
        case "/api/v1/epic/list":
            b.mu.Lock()
            b.epics++
            b.mu.Unlock()
            writeProgramJSON(w, map[string]any{"epics": []map[string]any{{"id": 5, "name": "Epic Five"}}})
        case "/api/v1/change/create":
            b.mu.Lock()
            b.creates = append(b.creates, payload)
            b.currentDef, _ = payload["def"].(string)
            fail := b.failCreate
            b.mu.Unlock()
            if fail {
                http.Error(w, "create failed", http.StatusInternalServerError)
                return
            }
            writeProgramJSON(w, map[string]any{"change": b.change(payload["title"], nil)})
        case "/api/v1/change/update-def":
            b.mu.Lock()
            b.defUpdates++
            b.currentDef, _ = payload["def"].(string)
            b.mu.Unlock()
            writeProgramJSON(w, map[string]any{"change": b.change("Program Change", nil)})
        case "/api/v1/change/update-change-types":
            values, _ := payload["change_types"].([]any)
            types := make([]string, 0, len(values))
            for _, value := range values {
                types = append(types, fmt.Sprint(value))
            }
            b.mu.Lock()
            b.typeUpdates = append(b.typeUpdates, types)
            fail := b.failTypeUpdate
            b.mu.Unlock()
            if fail {
                http.Error(w, "type update failed", http.StatusInternalServerError)
                return
            }
            writeProgramJSON(w, map[string]any{"change": b.change("Program Change", types)})
        case "/api/v1/change/get":
            writeProgramJSON(w, map[string]any{"change": b.change("Program Change", nil), "test_cases": []any{}})
        default:
            http.Error(w, r.URL.Path, http.StatusNotFound)
        }
    }))
    t.Cleanup(b.Close)
    return b
}

func (b *programBackend) change(title any, types []string) map[string]any {
    b.mu.Lock()
    defer b.mu.Unlock()
    refUUID := ""
    if len(b.creates) > 0 {
        refUUID, _ = b.creates[len(b.creates)-1]["ref_uuid"].(string)
    }
    return map[string]any{
        "id": 12, "project_id": 7, "ref_uuid": refUUID, "title": title,
        "def": b.currentDef, "change_types": types,
    }
}

func (b *programBackend) recordPath(path string) {
    b.mu.Lock()
    b.paths = append(b.paths, path)
    b.mu.Unlock()
}

func (b *programBackend) createCount() int {
    b.mu.Lock()
    defer b.mu.Unlock()
    return len(b.creates)
}

func (b *programBackend) epicCount() int {
    b.mu.Lock()
    defer b.mu.Unlock()
    return b.epics
}

func (b *programBackend) typeCatalogCount() int {
    b.mu.Lock()
    defer b.mu.Unlock()
    return b.typeCatalogs
}

func (b *programBackend) typeUpdateCount() int {
    b.mu.Lock()
    defer b.mu.Unlock()
    return len(b.typeUpdates)
}

func (b *programBackend) typeUpdatesCopy() [][]string {
    b.mu.Lock()
    defer b.mu.Unlock()
    result := make([][]string, len(b.typeUpdates))
    for i := range b.typeUpdates {
        result[i] = append([]string(nil), b.typeUpdates[i]...)
    }
    return result
}

func (b *programBackend) defUpdateCount() int {
    b.mu.Lock()
    defer b.mu.Unlock()
    return b.defUpdates
}

func (b *programBackend) lastRefUUID() string {
    b.mu.Lock()
    defer b.mu.Unlock()
    if len(b.creates) == 0 {
        return ""
    }
    value, _ := b.creates[len(b.creates)-1]["ref_uuid"].(string)
    return value
}

func (b *programBackend) changeMutationPaths() []string {
    b.mu.Lock()
    defer b.mu.Unlock()
    paths := make([]string, 0, len(b.paths))
    for _, path := range b.paths {
        if strings.HasPrefix(path, "/api/v1/change/") && path != "/api/v1/change/list" {
            paths = append(paths, path)
        }
    }
    return paths
}

type programRunner struct {
    mu           sync.Mutex
    root         string
    rewritten    string
    workspaceEnv map[string]string
    err          error
}

func (r *programRunner) ResolveRepoRoot(context.Context) (string, error) { return r.root, nil }

func (r *programRunner) Rewrite(_ context.Context, _ string, _ string, workspace agent.Workspace, _ agent.RewriteProgress) (agent.RewriteResult, error) {
    r.mu.Lock()
    r.workspaceEnv = map[string]string{
        "MCH_DEFAULT_DIR": agent.DefaultDir,
        "MCH_TEMP_DIR":    agent.TempDir,
        "MCH_REF_UUID":    workspace.RefUUID,
        "MCH_STAGE":       workspace.Stage,
    }
    err := r.err
    rewritten := r.rewritten
    root := r.root
    r.mu.Unlock()
    if err != nil {
        return agent.RewriteResult{}, err
    }
    if err := os.WriteFile(workspace.OutputPath(), []byte(rewritten), 0o644); err != nil {
        return agent.RewriteResult{}, err
    }
    return agent.RewriteResult{RepoRoot: root, SessionID: "33333333-3333-3333-3333-333333333333", Output: "Done."}, nil
}

func (*programRunner) InitCommand(string, string) *exec.Cmd { return exec.Command("true") }

func (r *programRunner) environment() map[string]string {
    r.mu.Lock()
    defer r.mu.Unlock()
    result := make(map[string]string, len(r.workspaceEnv))
    for key, value := range r.workspaceEnv {
        result[key] = value
    }
    return result
}

func writeProgramConfig(t *testing.T, root, backendURL, legacyTemp string) {
    t.Helper()
    flowDir := filepath.Join(root, agent.DefaultDir)
    require.NoError(t, os.MkdirAll(filepath.Join(flowDir, "prompts"), 0o755))
    config := "backend_url: " + backendURL + "\nproject_id: 7\n"
    if legacyTemp != "" {
        config += "temp_dir: " + legacyTemp + "\n"
    }
    require.NoError(t, os.WriteFile(filepath.Join(root, ".mch", "config.yaml"), []byte(config), 0o644))
    require.NoError(t, os.WriteFile(filepath.Join(flowDir, "flow.yaml"), []byte("version: 1\nslug: default\nname: Default\nhelp: help.yaml\nmakefile: Makefile\nsteps:\n  - slug: def-write\n    type: write\n    prompt: prompts/def-rewrite.md\n"), 0o644))
    require.NoError(t, os.WriteFile(filepath.Join(flowDir, "help.yaml"), []byte("version: 1\n"), 0o644))
}

func programWorkspace(root, refUUID string) string {
    return filepath.Join(root, agent.TempDir, refUUID, agent.ArtifactStage)
}

func writeProgramJSON(w http.ResponseWriter, value any) {
    _ = json.NewEncoder(w).Encode(value)
}

func readProgramFile(t *testing.T, path string) string {
    t.Helper()
    content, err := os.ReadFile(path)
    require.NoError(t, err)
    return string(content)
}
