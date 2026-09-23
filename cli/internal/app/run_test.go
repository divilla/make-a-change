package app

import (
    "bytes"
    "context"
    "errors"
    "fmt"
    "os"
    "os/exec"
    "path/filepath"
    "strings"
    "testing"

    "cli/internal/agent"
    "cli/internal/changes"
    "cli/internal/dto"
    "cli/internal/projects"
    "cli/internal/styles"

    "github.com/charmbracelet/bubbles/cursor"
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
    "github.com/gofrs/uuid/v5"
    "github.com/muesli/termenv"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

type fakeClient struct {
    projects                 []dto.Option
    projectRows              []dto.Project
    createdProject           dto.Project
    updatedProject           dto.Project
    gotProject               dto.Project
    changeRows               []dto.Change
    createdChange            dto.Change
    gotChange                dto.Change
    epics                    []dto.Option
    phases                   []dto.Option
    types                    []dto.Option
    err                      error
    createErr                error
    updateErr                error
    getErr                   error
    changeCreateErr          error
    changeReferenceErr       error
    changeUpdateErr          error
    changeTypesUpdateErr     error
    changeGetErr             error
    changeDeleteErr          error
    epicErr                  error
    projectID                string
    listCalls                int
    rowListCalls             int
    changeListCalls          int
    changeCreateCalls        int
    changeReferenceCalls     int
    changeTitleUpdateCalls   int
    changeDefUpdateCalls     int
    changeSpecUpdateCalls    int
    changePRUpdateCalls      int
    changePRUrlUpdateCalls   int
    changeTypesUpdateCalls   int
    changePhaseUpdateCalls   int
    changeOpenUpdateCalls    int
    changeEpicUpdateCalls    int
    testCaseCreateCalls      int
    testCaseUpdateCalls      int
    testCaseDoneCalls        int
    testCaseDeleteCalls      int
    changeDeleteCalls        int
    changeGetCalls           int
    createCalls              int
    updateCalls              int
    getCalls                 int
    phaseCalls               int
    typeCalls                int
    epicCalls                int
    createNames              []string
    updateIDs                []int
    updateNames              []string
    getIDs                   []int
    changeListProjectIDs     []string
    changeCreateInputs       []dto.ChangeCreateInput
    changeReferenceIDs       []int
    changeTitleUpdates       []string
    changeDefUpdates         []string
    changeSpecUpdates        []string
    changePRUpdates          []string
    changeArtifactAgentEdits []bool
    changePRUrlUpdates       []string
    changeTypesUpdates       [][]string
    changePhaseUpdates       []string
    changeOpenUpdates        []bool
    changeEpicUpdates        []*int
    testCaseCreateInputs     []dto.TestCase
    testCaseUpdateInputs     []dto.TestCase
    testCaseDoneUpdates      []bool
    testCaseDoneIDs          []int
    testCaseDeleteIDs        []int
    changeDeleteIDs          []int
    changeGetIDs             []int
    requestOrder             []string
}

type fakeAgentRunner struct {
    repoRoot          string
    rewriteResult     agent.RewriteResult
    rewriteErr        error
    resolveErr        error
    rewriteSessions   []string
    rewriteWorkspaces []agent.Workspace
    initRepoRoot      string
    initSessionID     string
}

func (f *fakeAgentRunner) ResolveRepoRoot(context.Context) (string, error) {
    if f.resolveErr != nil {
        return "", f.resolveErr
    }
    if f.repoRoot == "" {
        return "/repo", nil
    }
    return f.repoRoot, nil
}

func (f *fakeAgentRunner) Rewrite(_ context.Context, repoRoot string, sessionID string, workspace agent.Workspace, progress agent.RewriteProgress) (agent.RewriteResult, error) {
    f.rewriteSessions = append(f.rewriteSessions, sessionID)
    f.rewriteWorkspaces = append(f.rewriteWorkspaces, workspace)
    if progress != nil && f.rewriteResult.CommandOutput != "" {
        progress(f.rewriteResult.CommandOutput)
    }
    if f.rewriteErr != nil {
        return agent.RewriteResult{}, f.rewriteErr
    }
    result := f.rewriteResult
    if result.RepoRoot == "" {
        result.RepoRoot = repoRoot
    }
    return result, nil
}

func (f *fakeAgentRunner) InitCommand(repoRoot string, sessionID string) *exec.Cmd {
    f.initRepoRoot = repoRoot
    f.initSessionID = sessionID
    return exec.Command("true")
}

func (f *fakeClient) ListProjects() ([]dto.Option, error) {
    f.listCalls++
    return f.projects, f.err
}

func (f *fakeClient) ListProjectRows() ([]dto.Project, error) {
    f.rowListCalls++
    return f.projectRows, f.err
}

func (f *fakeClient) GetProject(id int) (dto.Project, error) {
    f.getCalls++
    f.getIDs = append(f.getIDs, id)
    if f.getErr != nil {
        return dto.Project{}, f.getErr
    }
    if f.err != nil {
        return dto.Project{}, f.err
    }
    return f.gotProject, nil
}

func (f *fakeClient) CreateProject(name string) (dto.Project, error) {
    f.createCalls++
    f.createNames = append(f.createNames, name)
    if f.createErr != nil {
        return dto.Project{}, f.createErr
    }
    if f.err != nil {
        return dto.Project{}, f.err
    }
    return f.createdProject, nil
}

func (f *fakeClient) UpdateProject(id int, name string) (dto.Project, error) {
    f.updateCalls++
    f.updateIDs = append(f.updateIDs, id)
    f.updateNames = append(f.updateNames, name)
    if f.updateErr != nil {
        return dto.Project{}, f.updateErr
    }
    if f.err != nil {
        return dto.Project{}, f.err
    }
    return f.updatedProject, nil
}

func (f *fakeClient) ListChangeRows(projectID string) ([]dto.Change, error) {
    f.changeListCalls++
    f.changeListProjectIDs = append(f.changeListProjectIDs, projectID)
    if f.err != nil {
        return nil, f.err
    }
    return f.changeRows, nil
}

func (f *fakeClient) GetChange(id int) (dto.Change, error) {
    f.requestOrder = append(f.requestOrder, "change/get")
    f.changeGetCalls++
    f.changeGetIDs = append(f.changeGetIDs, id)
    if f.changeGetErr != nil {
        return dto.Change{}, f.changeGetErr
    }
    if f.err != nil {
        return dto.Change{}, f.err
    }
    return f.gotChange, nil
}

func (f *fakeClient) CreateChange(input dto.ChangeCreateInput) (dto.Change, error) {
    f.requestOrder = append(f.requestOrder, "change/create")
    f.changeCreateCalls++
    f.changeCreateInputs = append(f.changeCreateInputs, input)
    if f.changeCreateErr != nil {
        return dto.Change{}, f.changeCreateErr
    }
    if f.err != nil {
        return dto.Change{}, f.err
    }
    return f.createdChange, nil
}

func (f *fakeClient) ReferenceChange(id int) (dto.Change, error) {
    f.requestOrder = append(f.requestOrder, "change/reference")
    f.changeReferenceCalls++
    f.changeReferenceIDs = append(f.changeReferenceIDs, id)
    if f.changeReferenceErr != nil {
        return dto.Change{}, f.changeReferenceErr
    }
    if f.err != nil {
        return dto.Change{}, f.err
    }
    if f.gotChange.ID != "" {
        return f.gotChange, nil
    }
    return dto.Change{ID: fmt.Sprint(id), Ref: "3", Slug: "3-change"}, nil
}

func (f *fakeClient) UpdateChangeTitle(id int, title string) (dto.Change, error) {
    f.changeTitleUpdateCalls++
    f.changeTitleUpdates = append(f.changeTitleUpdates, title)
    if f.changeUpdateErr != nil {
        return dto.Change{}, f.changeUpdateErr
    }
    return dto.Change{ID: fmt.Sprint(id), Title: title}, nil
}

func (f *fakeClient) UpdateChangeDef(id int, def string, agentEdit bool) (dto.Change, error) {
    f.requestOrder = append(f.requestOrder, "change/update-def")
    f.changeDefUpdateCalls++
    f.changeDefUpdates = append(f.changeDefUpdates, def)
    f.changeArtifactAgentEdits = append(f.changeArtifactAgentEdits, agentEdit)
    if f.changeUpdateErr != nil {
        return dto.Change{}, f.changeUpdateErr
    }
    return dto.Change{ID: fmt.Sprint(id), Def: def, AgentEdit: agentEdit}, nil
}

func (f *fakeClient) UpdateChangeSpec(id int, spec string, agentEdit bool) (dto.Change, error) {
    f.requestOrder = append(f.requestOrder, "change/update-spec")
    f.changeSpecUpdateCalls++
    f.changeSpecUpdates = append(f.changeSpecUpdates, spec)
    f.changeArtifactAgentEdits = append(f.changeArtifactAgentEdits, agentEdit)
    if f.changeUpdateErr != nil {
        return dto.Change{}, f.changeUpdateErr
    }
    return dto.Change{ID: fmt.Sprint(id), Spec: spec, AgentEdit: agentEdit}, nil
}

func (f *fakeClient) UpdateChangePR(id int, pr string, agentEdit bool) (dto.Change, error) {
    f.changePRUpdateCalls++
    f.changePRUpdates = append(f.changePRUpdates, pr)
    f.changeArtifactAgentEdits = append(f.changeArtifactAgentEdits, agentEdit)
    if f.changeUpdateErr != nil {
        return dto.Change{}, f.changeUpdateErr
    }
    return dto.Change{ID: fmt.Sprint(id), PR: pr, AgentEdit: agentEdit}, nil
}

func (f *fakeClient) UpdateChangePRUrl(id int, prURL string) (dto.Change, error) {
    f.changePRUrlUpdateCalls++
    f.changePRUrlUpdates = append(f.changePRUrlUpdates, prURL)
    if f.changeUpdateErr != nil {
        return dto.Change{}, f.changeUpdateErr
    }
    return dto.Change{ID: fmt.Sprint(id), PRUrl: prURL}, nil
}

func (f *fakeClient) UpdateChangeTypes(id int, changeTypes []string) (dto.Change, error) {
    f.requestOrder = append(f.requestOrder, "change/update-change-types")
    f.changeTypesUpdateCalls++
    f.changeTypesUpdates = append(f.changeTypesUpdates, append([]string{}, changeTypes...))
    if f.changeTypesUpdateErr != nil {
        return dto.Change{}, f.changeTypesUpdateErr
    }
    if f.changeUpdateErr != nil {
        return dto.Change{}, f.changeUpdateErr
    }
    return dto.Change{ID: fmt.Sprint(id), ChangeTypes: changeTypes}, nil
}

func (f *fakeClient) UpdateChangePhase(id int, changePhase string) (dto.Change, error) {
    f.changePhaseUpdateCalls++
    f.changePhaseUpdates = append(f.changePhaseUpdates, changePhase)
    if f.changeUpdateErr != nil {
        return dto.Change{}, f.changeUpdateErr
    }
    return dto.Change{ID: fmt.Sprint(id), ChangePhase: changePhase}, nil
}

func (f *fakeClient) UpdateChangeOpen(id int, open bool) (dto.Change, error) {
    f.changeOpenUpdateCalls++
    f.changeOpenUpdates = append(f.changeOpenUpdates, open)
    if f.changeUpdateErr != nil {
        return dto.Change{}, f.changeUpdateErr
    }
    return dto.Change{ID: fmt.Sprint(id), Open: open}, nil
}

func (f *fakeClient) UpdateChangeEpic(id int, epicID *int) (dto.Change, error) {
    f.changeEpicUpdateCalls++
    f.changeEpicUpdates = append(f.changeEpicUpdates, epicID)
    if f.changeUpdateErr != nil {
        return dto.Change{}, f.changeUpdateErr
    }
    return dto.Change{ID: fmt.Sprint(id)}, nil
}

func (f *fakeClient) CreateTestCase(changeID int, scenario string) (dto.Change, error) {
    f.requestOrder = append(f.requestOrder, "test-case/create")
    f.testCaseCreateCalls++
    f.testCaseCreateInputs = append(f.testCaseCreateInputs, dto.TestCase{ChangeID: fmt.Sprint(changeID), Scenario: scenario})
    if f.changeUpdateErr != nil {
        return dto.Change{}, f.changeUpdateErr
    }
    return f.gotChange, nil
}

func (f *fakeClient) UpdateTestCase(id int, scenario string) (dto.Change, error) {
    f.testCaseUpdateCalls++
    f.testCaseUpdateInputs = append(f.testCaseUpdateInputs, dto.TestCase{ID: fmt.Sprint(id), Scenario: scenario})
    if f.changeUpdateErr != nil {
        return dto.Change{}, f.changeUpdateErr
    }
    return f.gotChange, nil
}

func (f *fakeClient) UpdateTestCaseDone(id int, done bool) (dto.Change, error) {
    f.testCaseDoneCalls++
    f.testCaseDoneIDs = append(f.testCaseDoneIDs, id)
    f.testCaseDoneUpdates = append(f.testCaseDoneUpdates, done)
    if f.changeUpdateErr != nil {
        return dto.Change{}, f.changeUpdateErr
    }
    return dto.Change{ID: fmt.Sprint(id)}, nil
}

func (f *fakeClient) DeleteTestCase(id int) (dto.Change, error) {
    f.testCaseDeleteCalls++
    f.testCaseDeleteIDs = append(f.testCaseDeleteIDs, id)
    if f.changeUpdateErr != nil {
        return dto.Change{}, f.changeUpdateErr
    }
    return f.gotChange, nil
}

func (f *fakeClient) DeleteChange(id int) error {
    f.changeDeleteCalls++
    f.changeDeleteIDs = append(f.changeDeleteIDs, id)
    if f.changeDeleteErr != nil {
        return f.changeDeleteErr
    }
    if f.err != nil {
        return f.err
    }
    return nil
}

func (f *fakeClient) ListEpics(projectID string) ([]dto.Option, error) {
    f.epicCalls++
    f.projectID = projectID
    if f.epicErr != nil {
        return nil, f.epicErr
    }
    if f.err != nil {
        return nil, f.err
    }
    if projectID == "" {
        return nil, errors.New("current project is required")
    }
    return f.epics, nil
}

func (f *fakeClient) ListPhases() ([]dto.Option, error) {
    f.phaseCalls++
    return f.phases, f.err
}

func (f *fakeClient) ListTypes() ([]dto.Option, error) {
    f.typeCalls++
    return f.types, f.err
}

func newModelWithOptionCatalog(client *fakeClient) Model {
    m := NewModelWithClient(client)
    m.optionCatalog = optionCatalog{phases: client.phases, types: client.types, loaded: true}
    return m
}

func TestRunVersionPrintsVersion(t *testing.T) {
    var out bytes.Buffer

    require.NoError(t, Run([]string{"--version"}, &out))

    got := out.String()
    assert.Contains(t, got, "mch")
    assert.Contains(t, got, Version)
}

func TestRunReturnsVerboseConfigErrorBeforeStartingTUI(t *testing.T) {
    root := t.TempDir()
    require.NoError(t, exec.Command("git", "init", root).Run())
    previous, err := os.Getwd()
    require.NoError(t, err)
    require.NoError(t, os.Chdir(root))
    t.Cleanup(func() {
        require.NoError(t, os.Chdir(previous))
    })
    var out bytes.Buffer

    err = Run(nil, &out)

    require.Error(t, err)
    assert.Contains(t, err.Error(), "failed to load repository configuration")
    assert.Contains(t, err.Error(), filepath.Join(root, ".mch", "config.yaml"))
    assert.Empty(t, out.String())
}

func TestNewModelStartupState(t *testing.T) {
    m := NewModel()

    assert.Equal(t, MainState, m.state)
    assert.True(t, m.input.Focused())
    assert.Contains(t, m.View(), "MainScreen")
}

func TestShellChromeRendersTitleAndCurrentProjectInFooter(t *testing.T) {
    m := newModelWithConfig(&fakeClient{}, testAppConfig(appConfig{ProjectID: 7}))
    m.currentProject = dto.Option{ID: "7", Label: "Project Seven"}
    m.width = 180

    view := stripANSI(m.View())

    assert.Contains(t, view, "Make a change v0.1")
    assert.NotContains(t, view, "\nversion 0.1")
    assert.NotContains(t, view, "\nProject: ")
    assert.Contains(t, view, "Current Project: #7 Project Seven")
    assert.Contains(t, view, "0 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16")
    assert.Contains(t, m.View(), lipgloss.NewStyle().Background(lipgloss.Color("5")).Foreground(lipgloss.Color("15")).Render("5"))
    assert.Contains(t, m.View(), lipgloss.NewStyle().Background(lipgloss.Color("9")).Foreground(lipgloss.Color("15")).Render("9"))
    assert.Contains(t, m.View(), lipgloss.NewStyle().Background(lipgloss.Color("12")).Foreground(lipgloss.Color("0")).Render("12"))
}

func TestStartupTriggersProjectSelectionWhenProjectIDIsUnset(t *testing.T) {
    client := &fakeClient{
        projects: []dto.Option{{ID: "7", Label: "Project Seven"}},
    }
    m := NewModelWithClient(client)

    cmd := m.Init()
    require.NotNil(t, cmd)
    got := applyCommand(m, cmd)
    assert.Equal(t, SelectProjectDropDown, got.state)
    assert.Equal(t, selectorProjects, got.dropdown.source)

    load := selectorCommand(client, got.dropdown.source, got.currentProject.ID)
    got = applyMsg(got, load())

    assert.Equal(t, SelectProjectDropDown, got.state)
    assert.Equal(t, []dto.Option{{ID: "7", Label: "Project Seven"}}, got.dropdown.options)
}

func TestStartupSkipsProjectSelectionWhenProjectIDIsSaved(t *testing.T) {
    client := &fakeClient{gotProject: dto.Project{ID: "7", Name: "Project Seven"}}
    m := newModelWithConfig(client, testAppConfig(appConfig{ProjectID: 7}))
    m.width = 120

    require.NotNil(t, m.Init())
    got := applyCommand(m, m.Init())
    assert.Equal(t, MainState, m.state)
    assert.Equal(t, MainState, got.state)
    assert.Equal(t, "7", got.currentProject.ID)
    assert.Equal(t, "Project Seven", got.currentProject.Label)
    assert.Contains(t, stripANSI(got.View()), "Current Project: #7 Project Seven")
}

func TestStartupLoadsChangeOptionCatalog(t *testing.T) {
    previousTypes := ChangeTypes()
    t.Cleanup(func() { cacheChangeTypes(previousTypes) })
    flowDir := t.TempDir()
    require.NoError(t, os.Mkdir(filepath.Join(flowDir, "prompts"), 0o755))
    client := &fakeClient{
        phases: []dto.Option{{ID: "todo", Label: "todo", Color: "12"}},
        types:  []dto.Option{{ID: "feature", Label: "Feature"}, {ID: "fix", Label: "Fix"}},
    }
    m := newModelWithConfig(client, appConfig{FlowDir: flowDir})

    got := applyCommand(m, m.Init())

    assert.Equal(t, 1, client.phaseCalls)
    assert.Equal(t, 1, client.typeCalls)
    assert.True(t, got.optionCatalog.loaded)
    assert.Equal(t, client.phases, got.optionCatalog.phases)
    assert.Equal(t, client.types, got.optionCatalog.types)
    assert.Equal(t, "# Change Types\n\n- feature\n- fix\n", readTestFile(t, filepath.Join(flowDir, "prompts", changeTypeSlugsPrompt)))
    loadedTypes := ChangeTypes()
    assert.Equal(t, client.types, loadedTypes)
    loadedTypes[0].ID = "mutated"
    assert.Equal(t, "feature", ChangeTypes()[0].ID)
}

func TestRebuildChangeTypeSlugsFileOverwritesStaleContentExactly(t *testing.T) {
    flowDir := t.TempDir()
    promptsDir := filepath.Join(flowDir, "prompts")
    require.NoError(t, os.Mkdir(promptsDir, 0o755))
    path := filepath.Join(promptsDir, changeTypeSlugsPrompt)
    require.NoError(t, os.WriteFile(path, []byte("stale content that must be removed"), 0o644))

    err := rebuildChangeTypeSlugsFile(flowDir, []dto.Option{
        {ID: "feature", Label: "ignored label"},
        {ID: "fix", Label: "also ignored"},
        {ID: "refactor", Label: "Refactor"},
    })

    require.NoError(t, err)
    assert.Equal(t, "# Change Types\n\n- feature\n- fix\n- refactor\n", readTestFile(t, path))
}

func TestReplaceFileAtomicallyPreservesExistingFileAfterPartialWriteFailure(t *testing.T) {
    directory := t.TempDir()
    path := filepath.Join(directory, changeTypeSlugsPrompt)
    require.NoError(t, os.WriteFile(path, []byte("previous catalog\n"), 0o644))
    writeErr := errors.New("disk full after partial write")

    err := replaceFileAtomically(path, 0o644, func(file *os.File) error {
        _, err := file.WriteString("partial replacement")
        require.NoError(t, err)
        return writeErr
    })

    require.ErrorIs(t, err, writeErr)
    assert.Equal(t, "previous catalog\n", readTestFile(t, path))
    entries, err := os.ReadDir(directory)
    require.NoError(t, err)
    require.Len(t, entries, 1)
    assert.Equal(t, changeTypeSlugsPrompt, entries[0].Name())
}

func TestStartupReportsChangeTypeSlugsFileWriteFailure(t *testing.T) {
    previousTypes := ChangeTypes()
    t.Cleanup(func() { cacheChangeTypes(previousTypes) })
    cacheChangeTypes([]dto.Option{{ID: "existing"}})
    flowDir := filepath.Join(t.TempDir(), "missing-flow-directory")
    client := &fakeClient{
        phases: []dto.Option{{ID: "todo"}},
        types:  []dto.Option{{ID: "feature"}},
    }
    m := newModelWithConfig(client, appConfig{FlowDir: flowDir})

    got := applyMsg(m, optionCatalogCommand(client, flowDir)())

    assert.False(t, got.optionCatalog.loaded)
    assert.Error(t, got.optionCatalog.err)
    assert.Contains(t, got.err, "rebuild Change type slugs file")
    assert.Equal(t, "option catalog failed", got.status)
    assert.Equal(t, []dto.Option{{ID: "existing"}}, ChangeTypes())
}

func TestStartupDistinguishesEmptyAndFailedTypeCatalogResponses(t *testing.T) {
    t.Run("successful empty response", func(t *testing.T) {
        m := NewModelWithClient(&fakeClient{})

        got := applyMsg(m, optionCatalogLoadedMsg{})

        assert.True(t, got.optionCatalog.loaded)
        assert.Empty(t, got.optionCatalog.types)
        assert.NoError(t, got.optionCatalog.err)
    })

    t.Run("failed response", func(t *testing.T) {
        loadErr := errors.New("type catalog failed")
        m := NewModelWithClient(&fakeClient{})

        got := applyMsg(m, optionCatalogLoadedMsg{err: loadErr})

        assert.False(t, got.optionCatalog.loaded)
        assert.ErrorIs(t, got.optionCatalog.err, loadErr)
        assert.Equal(t, loadErr.Error(), got.err)
    })
}

func TestStartupProjectSelectionShowsErrorWhenNoProjectsExist(t *testing.T) {
    client := &fakeClient{}
    m := NewModelWithClient(client)

    cmd := m.Init()
    require.NotNil(t, cmd)
    got := applyCommand(m, cmd)
    load := selectorCommand(client, got.dropdown.source, got.currentProject.ID)
    got = applyMsg(got, load())

    assert.Equal(t, MainState, got.state)
    assert.Empty(t, got.dropdown.kind)
    assert.Equal(t, noProjectsToSelectError, got.err)
}

func TestInputBandUsesCliProtoFullWidthBackground(t *testing.T) {
    m := NewModel()
    m.width = 40
    assert.Equal(t, 1, m.input.Width())
    assert.NotEqual(t, "252", fmt.Sprint(styles.Default.Surface.GetForeground()))
    assert.NotEqual(t, "235", fmt.Sprint(styles.Default.Surface.GetBackground()))

    band := m.inputBand(40)
    lines := strings.Split(band, "\n")
    require.Len(t, lines, 3)
    assert.Contains(t, band, "Type / for commands")
    for i, line := range lines {
        visible := stripANSI(line)
        assert.Falsef(t, strings.TrimSpace(visible) == "" && len(visible) < 40, "blank input band line %d too short: %q", i, visible)
    }
    assert.True(t, strings.HasPrefix(stripANSI(lines[1]), "> Type / for commands"))

    m = m.setPromptValue("typed text")
    typedBand := m.inputBand(40)
    assert.NotContains(t, typedBand, "48;5;0")
    assert.NotContains(t, typedBand, "[40m")
    typedLine := stripANSI(strings.Split(typedBand, "\n")[1])
    assert.True(t, strings.HasPrefix(typedLine, "> typed text"))
    assert.Equal(t, "15", fmt.Sprint(m.input.FocusedStyle.Text.GetForeground()))
    assert.Equal(t, "15", fmt.Sprint(m.input.FocusedStyle.CursorLine.GetForeground()))
    assert.Equal(t, "0", fmt.Sprint(m.input.FocusedStyle.Placeholder.GetForeground()))
    assert.Equal(t, cursor.CursorStatic, m.input.Cursor.Mode())

    wideBand := m.inputBand(180)
    wideLines := strings.Split(wideBand, "\n")
    require.Len(t, wideLines, 3)
    assert.Len(t, stripANSI(wideLines[0]), 180)
    assert.Len(t, stripANSI(wideLines[1]), 180)
    assert.Len(t, stripANSI(wideLines[2]), 180)
}

func TestPromptTextareaGrowsForExplicitNewlines(t *testing.T) {
    m := NewModel()
    m = m.setPromptValue("first line\nsecond line\n")

    band := stripANSI(m.inputBand(40))
    lines := strings.Split(band, "\n")

    require.Len(t, lines, 5)
    assert.True(t, strings.HasPrefix(lines[1], "> first line"))
    assert.True(t, strings.HasPrefix(lines[2], "> second line"))
    assert.True(t, strings.HasPrefix(lines[3], "> "))
}

func TestPromptNewlineKeyAddsBlankPromptLine(t *testing.T) {
    m := NewModel()
    m = m.setPromptValue("first line")

    got, cmd := sendKeyMsg(m, tea.KeyMsg{Type: tea.KeyEnter, Alt: true})

    assert.Nil(t, cmd)
    assert.Equal(t, "first line\n", got.input.Value())
    band := got.inputBand(40)
    assert.NotContains(t, band, "48;5;0")
    assert.NotContains(t, band, "[40m")
    assert.Equal(t, 4, len(strings.Split(stripANSI(band), "\n")))
}

func TestPromptShiftEnterEscapeSequenceAddsNewline(t *testing.T) {
    m := NewModel()
    m = m.setPromptValue("first line")

    got, cmd := sendKeyMsg(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'O'}, Alt: true})
    assert.Nil(t, cmd)
    assert.Equal(t, "first line", got.input.Value())
    assert.True(t, got.pendingAltO)

    got, cmd = sendKeyMsg(got, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'M'}})

    assert.Nil(t, cmd)
    assert.False(t, got.pendingAltO)
    assert.Equal(t, "first line\n", got.input.Value())
    assert.NotContains(t, got.input.Value(), "OM")
    assert.Equal(t, 4, len(strings.Split(stripANSI(got.inputBand(40)), "\n")))
}

func TestPromptInputUsesTerminalWidthForTyping(t *testing.T) {
    m := NewModel()
    m.width = 40

    got, cmd := sendRune(m, 'a')

    assert.Nil(t, cmd)
    assert.Equal(t, "a", got.input.Value())
    assert.Greater(t, got.input.Width(), 1)
}

func TestPromptUpDownMovesVisibleCursorBetweenLines(t *testing.T) {
    m := NewModel()
    m = m.setPromptValue("first\nsecond")

    got, cmd := sendKey(m, tea.KeyUp)

    assert.Nil(t, cmd)
    assert.Equal(t, 0, got.promptCursorRow)
    assert.Equal(t, len("first"), got.promptCursorCol)

    got, cmd = sendKey(got, tea.KeyDown)

    assert.Nil(t, cmd)
    assert.Equal(t, 1, got.promptCursorRow)
    assert.Equal(t, len("first"), got.promptCursorCol)
}

func TestViewAddsBlankLineBetweenPromptAndFooter(t *testing.T) {
    m := NewModelWithClient(&fakeClient{})
    m.width = 40

    lines := strings.Split(stripANSI(m.View()), "\n")
    var promptLine int
    for i, line := range lines {
        if strings.HasPrefix(line, "> Type / for commands") {
            promptLine = i
            break
        }
    }
    require.NotZero(t, promptLine)
    require.Greater(t, len(lines), promptLine+3)
    assert.Empty(t, strings.TrimSpace(lines[promptLine+1]))
    assert.Empty(t, strings.TrimSpace(lines[promptLine+2]))
    assert.Contains(t, lines[promptLine+3], "</> command")
}

func TestNewProjectUsesNamePlaceholder(t *testing.T) {
    m := NewModelWithClient(&fakeClient{})
    m.state = ProjectsListState

    got, _ := sendCommand(m, "/new-project")

    assert.Equal(t, ProjectCreateState, got.state)
    assert.Equal(t, "Write a Name", got.input.Placeholder)
    assert.Contains(t, stripANSI(got.inputBand(40)), "> Write a Name")

    got, _ = sendCommand(got, "/cancel")
    assert.Equal(t, defaultInputPlaceholder, got.input.Placeholder)
}

func TestProjectFormsExposeEditorCommandFirst(t *testing.T) {
    assert.Equal(t, []string{"/editor", "/save", "/cancel"}, commandsByState[ProjectCreateState])
    assert.Equal(t, []string{"/editor", "/save", "/cancel"}, commandsByState[ProjectUpdateState])
}

func TestProjectEditorSavesResultWithoutReturningToPrompt(t *testing.T) {
    m := NewModelWithClient(&fakeClient{})
    m.state = ProjectCreateState
    m.input.SetValue("Initial Name")

    updated, cmd := m.Update(editorFinishedMsg{source: ProjectCreateState, content: "Edited\nName\n"})
    got := updated.(Model)

    require.NotNil(t, cmd)
    assert.Equal(t, "Initial Name", got.input.Value())
    assert.Equal(t, "saving", got.status)
}

func TestChangeEditorPreservesEditedMarkdownAfterFailedSave(t *testing.T) {
    tests := []struct {
        name     string
        source   State
        original string
        edited   string
    }{
        {
            name:   "create",
            source: ChangeCreateState,
            edited: "# Edited Change\n\nTypes: unknown\n\n## Problem Statement\nKeep this edit.",
        },
        {
            name:     "update",
            source:   ChangeUpdateState,
            original: "# Original Change\n\nTypes: feature\n\n## Problem Statement\nOriginal spec.",
            edited:   "# Edited Change\n\nTypes: feature\n\n## Problem Statement\nKeep this edit.",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            m := NewModelWithClient(&fakeClient{})
            m.currentProject = dto.Option{ID: "7", Label: "Project Seven"}
            m.state = tt.source
            m.input.SetValue(tt.original)
            m.changeList.Detail = dto.Change{
                ID:          "12",
                Title:       "Original Change",
                Spec:        tt.original,
                ChangeTypes: []string{"feature"},
            }

            updated, cmd := m.Update(editorFinishedMsg{source: tt.source, content: tt.edited})
            got := updated.(Model)

            require.NotNil(t, cmd)
            assert.Equal(t, tt.edited, got.input.Value())

            got = applyMsg(got, changeSavedMsg{source: tt.source, err: errors.New("save failed")})
            assert.Equal(t, tt.source, got.state)
            assert.Equal(t, "save failed", got.status)
            assert.Equal(t, tt.edited, got.input.Value())
        })
    }
}

func TestProjectEditorIgnoresStaleResultAndReportsErrors(t *testing.T) {
    m := NewModelWithClient(&fakeClient{})
    m.state = ProjectUpdateState
    m.input.SetValue("Current")

    got := applyMsg(m, editorFinishedMsg{source: ProjectCreateState, content: "Stale"})
    assert.Equal(t, "Current", got.input.Value())

    got = applyMsg(got, editorFinishedMsg{source: ProjectUpdateState, err: errors.New("nano failed")})
    assert.Equal(t, "Current", got.input.Value())
    assert.Equal(t, "nano failed", got.err)
    assert.Equal(t, "editor failed", got.status)
}

func TestProjectEditorUsesEditorEnvWithNanoFallback(t *testing.T) {
    t.Setenv("EDITOR", "")
    fallback := editorCommand("/tmp/project.md")
    assert.Equal(t, "nano", fallback.Args[0])
    assert.Equal(t, "/tmp/project.md", fallback.Args[1])

    t.Setenv("EDITOR", "vim -f")
    fromEnv := editorCommand("/tmp/project.md")
    assert.Equal(t, "sh", fromEnv.Args[0])
    assert.Equal(t, []string{"sh", "-c", "$EDITOR \"$1\"", "mch-editor", "/tmp/project.md"}, fromEnv.Args)
    assert.Contains(t, fromEnv.Env, "EDITOR=vim -f")
}

func TestPromptEnterSavesProjectFormRawMultilineValue(t *testing.T) {
    client := &fakeClient{
        createdProject: dto.Project{ID: "7"},
        gotProject:     dto.Project{ID: "7", Name: "Line 1\nLine 2"},
    }
    m := NewModelWithClient(client)
    m.state = ProjectCreateState
    m.input.SetValue("Line 1\nLine 2")

    updated, cmd := sendKey(m, tea.KeyEnter)
    got := updated

    require.NotNil(t, cmd)
    assert.Equal(t, ProjectCreateState, got.state)
    assert.Equal(t, "saving", got.status)

    got = applyMsg(got, cmd())

    assert.Equal(t, ProjectDetailsState, got.state)
    assert.Equal(t, []string{"Line 1\nLine 2"}, client.createNames)
}

func TestPromptCtrlCClearsBeforeCancelingOrQuitting(t *testing.T) {
    m := NewModelWithClient(&fakeClient{})
    m.state = ProjectCreateState
    m.input.SetValue("draft")

    got, cmd := sendKey(m, tea.KeyCtrlC)
    assert.Nil(t, cmd)
    assert.Equal(t, ProjectCreateState, got.state)
    assert.Empty(t, got.input.Value())
    assert.Equal(t, "prompt cleared", got.status)

    got, cmd = sendKey(got, tea.KeyCtrlC)
    assert.NotNil(t, cmd)
    assert.Equal(t, ProjectsListState, got.state)

    got, cmd = sendKey(NewModelWithClient(&fakeClient{}), tea.KeyCtrlC)
    assert.NotNil(t, cmd)
    assert.Equal(t, DoneState, got.state)
    assert.True(t, got.quitting)
}

func TestMainCommandsTransition(t *testing.T) {
    tests := []struct {
        command string
        want    State
        quit    bool
    }{
        {command: "/changes", want: ChangesListState},
        {command: "/epics", want: EpicsListState},
        {command: "/projects", want: ProjectsListState},
        {command: "/help", want: MainHelpState},
        {command: "/quit", want: DoneState, quit: true},
    }

    for _, tt := range tests {
        t.Run(tt.command, func(t *testing.T) {
            got, cmd := sendCommand(NewModel(), tt.command)
            assert.Equal(t, tt.want, got.state)
            if tt.quit && cmd == nil {
                require.NotNil(t, cmd)
            }
        })
    }
}

func TestProjectsCommandReloadsAndRendersSelectableTable(t *testing.T) {
    client := &fakeClient{
        projectRows: []dto.Project{
            {
                ID:          "7",
                Name:        "Project Seven",
                ChangeCount: 3,
                Created:     "2026-06-29T08:15:00Z",
                Modified:    "2026-06-29T10:45:00Z",
            },
            {
                ID:          "8",
                Name:        "Project Eight",
                ChangeCount: 0,
                Created:     "bad timestamp",
                Modified:    "",
            },
        },
    }
    m := newModelWithOptionCatalog(client)

    got, cmd := sendCommand(m, "/projects")
    require.Equal(t, ProjectsListState, got.state)
    require.NotNil(t, cmd)
    assert.True(t, got.projectList.Loading)

    got = applyMsg(got, cmd())

    assert.Equal(t, 1, client.rowListCalls)
    assert.False(t, got.projectList.Loading)
    assert.Equal(t, 0, got.projectList.Selected)
    view := stripANSI(got.View())
    assert.Contains(t, view, "ProjectsListScreen")
    assert.Contains(t, view, "id")
    assert.Contains(t, view, "Name")
    assert.Contains(t, view, "Changes")
    assert.Contains(t, view, "Created")
    assert.Contains(t, view, "Modified")
    assert.Contains(t, view, "     7  Project Seven")
    assert.Contains(t, view, "Project Seven")
    assert.Contains(t, view, "3")
    assert.Contains(t, view, "2026-06-29")
    assert.Contains(t, view, "not a date")

    got, _ = sendCommand(got, "/return")
    got, cmd = sendCommand(got, "/projects")
    require.NotNil(t, cmd)
    got = applyMsg(got, cmd())
    assert.Equal(t, 2, client.rowListCalls)
    assert.Equal(t, ProjectsListState, got.state)
}

func TestProjectsTableUsesDynamicNameWidthAndTrimsVeryLongNames(t *testing.T) {
    longName := "This is a project with a real name that is long enough to resize the name column"
    tooLongName := longName + " and has additional words on the right that must be removed"
    m := NewModelWithClient(&fakeClient{})
    m.state = ProjectsListState
    m.projectList.Rows = []dto.Project{
        {ID: "1", Name: "demo1", ChangeCount: 2, Created: "2026-06-23T04:51:00Z", Modified: "2026-06-23T04:51:00Z"},
        {ID: "350", Name: longName, ChangeCount: 0, Created: "2026-06-29T15:57:00Z", Modified: "2026-06-29T15:57:00Z"},
        {ID: "351", Name: tooLongName, ChangeCount: 1, Created: "2026-06-29T15:58:00Z", Modified: "2026-06-29T15:58:00Z"},
    }

    rendered := stripANSI(projects.TableView(m.projectList, 160))
    lines := strings.Split(rendered, "\n")
    require.Len(t, lines, 4)

    createdColumn := strings.Index(lines[0], "Created")
    require.NotEqual(t, -1, createdColumn)
    assert.Equal(t, createdColumn, strings.Index(lines[1], "2026-"))
    assert.Equal(t, createdColumn, strings.Index(lines[2], "2026-"))
    assert.Equal(t, createdColumn, strings.Index(lines[3], "2026-"))
    assert.Contains(t, lines[2], longName)
    assert.NotContains(t, lines[3], "must be removed")
    trimmedName := projects.ProjectTableName(tooLongName)
    assert.True(t, strings.HasSuffix(trimmedName, "..."))
    assert.Less(t, len([]rune(trimmedName)), 78)
    assert.Contains(t, lines[3], trimmedName)
}

func TestProjectsTableSelectionIsBounded(t *testing.T) {
    m := NewModelWithClient(&fakeClient{})
    m.state = ProjectsListState
    m.projectList.Rows = []dto.Project{
        {ID: "1", Name: "One"},
        {ID: "2", Name: "Two"},
    }

    got, _ := sendKey(m, tea.KeyUp)
    assert.Equal(t, 0, got.projectList.Selected)

    got, _ = sendKey(got, tea.KeyDown)
    assert.Equal(t, 1, got.projectList.Selected)

    got, _ = sendKey(got, tea.KeyDown)
    assert.Equal(t, 1, got.projectList.Selected)

    got, _ = sendKey(got, tea.KeyUp)
    assert.Equal(t, 0, got.projectList.Selected)
}

func TestProjectsEnterOpensDetailsWithoutMutatingCurrentProject(t *testing.T) {
    current := dto.Option{ID: "99", Label: "Current Project"}
    client := &fakeClient{
        gotProject: dto.Project{ID: "8", Name: "Fresh Project Eight", ChangeCount: 5, Created: "2026-06-30T08:15:00Z", Modified: "2026-06-30T11:45:00Z"},
    }
    m := newModelWithOptionCatalog(client)
    m.state = ProjectsListState
    m.currentProject = current
    m.projectList.Rows = []dto.Project{
        {ID: "7", Name: "Project Seven", ChangeCount: 3, Created: "2026-06-29T08:15:00Z", Modified: "2026-06-29T10:45:00Z"},
        {ID: "8", Name: "Project Eight", ChangeCount: 4, Created: "2026-06-30T08:15:00Z", Modified: "2026-06-30T10:45:00Z"},
    }
    m.projectList.Selected = 1

    got, cmd := sendKey(m, tea.KeyEnter)

    assert.Equal(t, ProjectDetailsState, got.state)
    assert.Equal(t, current, got.currentProject)
    assert.Equal(t, dto.Project{ID: "8", Name: "Project Eight", ChangeCount: 4, Created: "2026-06-30T08:15:00Z", Modified: "2026-06-30T10:45:00Z"}, got.projectList.Detail)
    require.NotNil(t, cmd)
    got = applyMsg(got, cmd())
    assert.Equal(t, []int{8}, client.getIDs)
    assert.Equal(t, client.gotProject, got.projectList.Detail)
    view := stripANSI(got.View())
    assert.Contains(t, view, "ProjectDetailsScreen")
    assert.Contains(t, view, "         #ID: 8")
    assert.Contains(t, view, "        Name: Fresh Project Eight")
    assert.Contains(t, view, "Changes: 5")
}

func TestProjectDetailsRenderRequiredLabelsAndTimestampFallback(t *testing.T) {
    m := NewModelWithClient(&fakeClient{})
    m.state = ProjectDetailsState
    m.width = 32
    m.projectList.Detail = dto.Project{
        ID:          "7",
        Name:        "Project Seven",
        ChangeCount: 3,
        Created:     "2026-06-29T13:04:59.999Z",
        Modified:    "malformed",
    }

    rawDetails := projects.DetailsView(m.projectList.Detail, 32)
    view := stripANSI(m.View())
    whiteValue := lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
    pinkValue := lipgloss.NewStyle().Foreground(lipgloss.Color("218"))
    timestampValue := lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
    createdValue := projects.FormatTimestamp("2026-06-29T13:04:59.999Z")

    assert.Contains(t, view, "         #ID: 7")
    assert.Contains(t, view, "        Name: Project Seven")
    assert.Contains(t, view, "     Changes: 3")
    assert.Contains(t, view, "     Created: "+createdValue)
    assert.Contains(t, view, "    Modified: not a date")
    assert.Contains(t, rawDetails, pinkValue.Render("7"))
    assert.Contains(t, rawDetails, whiteValue.Render("3"))
    assert.Contains(t, rawDetails, timestampValue.Render(createdValue))
    assert.Contains(t, rawDetails, timestampValue.Render("not a date"))
    assert.Contains(t, rawDetails, styles.Default.AccentCyan.Render("Project Seven"))
    for _, line := range strings.Split(stripANSI(rawDetails), "\n") {
        assert.LessOrEqual(t, len(line), 32)
    }
}

func TestProjectDetailsWrapsNameAtEightyCharactersWithoutBreakingWords(t *testing.T) {
    name := "This project name is deliberately long and should wrap onto the next line without breaking any words in half"
    m := NewModelWithClient(&fakeClient{})
    m.state = ProjectDetailsState
    m.width = 120
    m.projectList.Detail = dto.Project{ID: "7", Name: name}

    view := stripANSI(m.View())

    assert.Contains(t, view, "        Name: This project name is deliberately long and should wrap onto the next line")
    assert.Contains(t, view, "\n              without breaking any words in half")
    assert.NotContains(t, view, "witho\n")
}

func TestProjectDetailsPreservesExplicitNameNewlines(t *testing.T) {
    m := NewModelWithClient(&fakeClient{})
    m.state = ProjectDetailsState
    m.width = 120
    m.projectList.Detail = dto.Project{ID: "7", Name: "First line\nSecond line"}

    view := stripANSI(m.View())

    assert.Contains(t, view, "        Name: First line")
    assert.Contains(t, view, "\n              Second line")
}

func TestProjectPagesReloadOnArrival(t *testing.T) {
    client := &fakeClient{
        projectRows: []dto.Project{{ID: "7", Name: "Reloaded List Project"}},
        gotProject:  dto.Project{ID: "7", Name: "Reloaded Detail Project"},
    }

    m := newModelWithOptionCatalog(client)
    m.state = ProjectDetailsState
    m.projectList.Detail = dto.Project{ID: "7", Name: "Stale Detail Project"}
    got, cmd := sendCommand(m, "/return")
    require.NotNil(t, cmd)
    assert.Equal(t, ProjectsListState, got.state)
    assert.True(t, got.projectList.Loading)
    got = applyMsg(got, cmd())
    assert.Equal(t, 1, client.rowListCalls)
    assert.Equal(t, []dto.Project{{ID: "7", Name: "Reloaded List Project"}}, got.projectList.Rows)

    got.state = ProjectUpdateState
    got.projectList.Detail = dto.Project{ID: "7", Name: "Stale Detail Project"}
    got, cmd = sendCommand(got, "/cancel")
    require.NotNil(t, cmd)
    assert.Equal(t, ProjectDetailsState, got.state)
    got = applyMsg(got, cmd())
    assert.Equal(t, []int{7}, client.getIDs)
    assert.Equal(t, client.gotProject, got.projectList.Detail)
}

func TestProjectsEnterWithNoSelectableRowErrors(t *testing.T) {
    m := NewModelWithClient(&fakeClient{})
    m.state = ProjectsListState

    got, _ := sendKey(m, tea.KeyEnter)

    assert.Equal(t, ProjectsListState, got.state)
    assert.NotEmpty(t, got.err)
}

func TestProjectsLoadFailureAndEmptyListAreDeterministic(t *testing.T) {
    failing := &fakeClient{err: errors.New("backend unavailable")}
    m := NewModelWithClient(failing)

    got, cmd := sendCommand(m, "/projects")
    require.NotNil(t, cmd)
    got = applyMsg(got, cmd())

    assert.Equal(t, ProjectsListState, got.state)
    assert.False(t, got.projectList.Loading)
    assert.Equal(t, "backend unavailable", got.err)
    assert.Contains(t, stripANSI(got.View()), "No projects.")

    empty := NewModelWithClient(&fakeClient{})
    got, cmd = sendCommand(empty, "/projects")
    require.NotNil(t, cmd)
    got = applyMsg(got, cmd())

    assert.Equal(t, ProjectsListState, got.state)
    assert.Contains(t, stripANSI(got.View()), "No projects.")
}

func TestProjectCreateSavePersistsFetchesDetailsAndDoesNotMutateConfig(t *testing.T) {
    path := filepath.Join(t.TempDir(), ".mch", "config.yaml")
    require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
    require.NoError(t, saveAppConfig(path, testAppConfig(appConfig{ProjectID: 99})))
    client := &fakeClient{
        createdProject: dto.Project{ID: "7"},
        gotProject: dto.Project{
            ID:          "7",
            Name:        "New Project",
            ChangeCount: 0,
            Created:     "2026-06-29T11:04:59Z",
            Modified:    "2026-06-29T11:04:59Z",
        },
    }
    m := newModelWithConfig(client, testAppConfig(appConfig{ProjectID: 99, ConfigPath: path}))
    m.state = ProjectCreateState
    m.input.SetValue("  New\nProject  ")

    updated, cmd := m.executeCommandFrom(ProjectCreateState, "/save")
    got := updated.(Model)
    require.NotNil(t, cmd)
    assert.Equal(t, ProjectCreateState, got.state)
    assert.Equal(t, "saving", got.status)

    got = applyMsg(got, cmd())

    assert.Equal(t, ProjectDetailsState, got.state)
    assert.Equal(t, []string{"  New\nProject  "}, client.createNames)
    assert.Equal(t, []int{7}, client.getIDs)
    assert.Equal(t, client.gotProject, got.projectList.Detail)
    assert.Equal(t, "99", got.currentProject.ID)
    loaded, err := loadConfigFile(path)
    require.NoError(t, err)
    assert.Equal(t, 99, loaded.ProjectID)
    view := stripANSI(got.View())
    assert.Contains(t, view, "Name: New Project")
    assert.Contains(t, view, "Changes: 0")
}

func TestProjectCreateValidationDoesNotCallBackend(t *testing.T) {
    client := &fakeClient{}
    m := NewModelWithClient(client)
    m.state = ProjectCreateState
    m.input.SetValue("   ")

    got, cmd := sendKey(m, tea.KeyEnter)

    assert.Nil(t, cmd)
    assert.Equal(t, ProjectCreateState, got.state)
    assert.Contains(t, got.err, "project name is required")
    assert.Zero(t, client.createCalls)
    assert.Zero(t, client.getCalls)
}

func TestProjectUpdateSavePersistsFetchesDetailsAndDoesNotMutateConfig(t *testing.T) {
    path := filepath.Join(t.TempDir(), ".mch", "config.yaml")
    require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
    require.NoError(t, saveAppConfig(path, testAppConfig(appConfig{ProjectID: 99})))
    client := &fakeClient{
        updatedProject: dto.Project{ID: "7"},
        gotProject: dto.Project{
            ID:          "7",
            Name:        "Renamed Project",
            ChangeCount: 2,
            Created:     "2026-06-29T08:15:00Z",
            Modified:    "2026-06-29T13:04:59Z",
        },
    }
    m := newModelWithConfig(client, testAppConfig(appConfig{ProjectID: 99, ConfigPath: path}))
    m.state = ProjectDetailsState
    m.projectList.Detail = dto.Project{ID: "7", Name: "Old Project", ChangeCount: 2}

    got, _ := sendCommand(m, "/edit")
    assert.Equal(t, ProjectUpdateState, got.state)
    assert.Equal(t, "Old Project", got.input.Value())
    got.input.SetValue("  Renamed\nProject  ")

    updated, cmd := got.executeCommandFrom(ProjectUpdateState, "/save")
    got = updated.(Model)
    require.NotNil(t, cmd)
    got = applyMsg(got, cmd())

    assert.Equal(t, ProjectDetailsState, got.state)
    assert.Equal(t, []int{7}, client.updateIDs)
    assert.Equal(t, []string{"  Renamed\nProject  "}, client.updateNames)
    assert.Equal(t, []int{7}, client.getIDs)
    assert.Equal(t, client.gotProject, got.projectList.Detail)
    assert.Equal(t, "99", got.currentProject.ID)
    loaded, err := loadConfigFile(path)
    require.NoError(t, err)
    assert.Equal(t, 99, loaded.ProjectID)
}

func TestProjectUpdateValidationDoesNotCallBackend(t *testing.T) {
    tests := []dto.Project{
        {},
        {ID: "0", Name: "Zero"},
        {ID: "-1", Name: "Negative"},
        {ID: "not-a-number", Name: "Bad"},
    }

    for _, project := range tests {
        t.Run(project.ID, func(t *testing.T) {
            client := &fakeClient{}
            m := NewModelWithClient(client)
            m.state = ProjectUpdateState
            m.projectList.Detail = project
            m.input.SetValue("Renamed")

            updated, cmd := m.executeCommandFrom(ProjectUpdateState, "/save")
            got := updated.(Model)

            assert.Nil(t, cmd)
            assert.Equal(t, ProjectUpdateState, got.state)
            assert.Contains(t, got.err, "project ID must be a valid positive number")
            assert.Zero(t, client.updateCalls)
            assert.Zero(t, client.getCalls)
        })
    }
}

func TestProjectSaveBackendFailurePreservesRecoverableFormState(t *testing.T) {
    client := &fakeClient{createErr: errors.New("invalid project payload")}
    m := NewModelWithClient(client)
    m.state = ProjectCreateState
    m.input.SetValue("New Project")

    updated, cmd := m.executeCommandFrom(ProjectCreateState, "/save")
    got := updated.(Model)
    require.NotNil(t, cmd)
    got = applyMsg(got, cmd())

    assert.Equal(t, ProjectCreateState, got.state)
    assert.Equal(t, "New Project", got.input.Value())
    assert.Equal(t, "invalid project payload", got.err)
    assert.Equal(t, 1, client.createCalls)
    assert.Zero(t, client.getCalls)

    client = &fakeClient{
        updatedProject: dto.Project{ID: "7"},
        getErr:         errors.New("project not found"),
    }
    m = newModelWithOptionCatalog(client)
    m.state = ProjectUpdateState
    m.projectList.Detail = dto.Project{ID: "7", Name: "Old Project"}
    m.input.SetValue("Renamed Project")

    updated, cmd = m.executeCommandFrom(ProjectUpdateState, "/save")
    got = updated.(Model)
    require.NotNil(t, cmd)
    got = applyMsg(got, cmd())

    assert.Equal(t, ProjectUpdateState, got.state)
    assert.Equal(t, "Renamed Project", got.input.Value())
    assert.Equal(t, "project not found", got.err)
    assert.Equal(t, 1, client.updateCalls)
    assert.Equal(t, 1, client.getCalls)
}

func TestProjectCancelDoesNotCallPersistence(t *testing.T) {
    client := &fakeClient{}
    m := NewModelWithClient(client)
    m.state = ProjectCreateState
    m.input.SetValue("New Project")

    got, _ := sendCommand(m, "/cancel")

    assert.Equal(t, ProjectsListState, got.state)
    assert.Zero(t, client.createCalls)
    assert.Zero(t, client.updateCalls)

    m = newModelWithOptionCatalog(client)
    m.state = ProjectUpdateState
    m.projectList.Detail = dto.Project{ID: "7", Name: "Old Project"}
    m.input.SetValue("Renamed Project")

    got, _ = sendKey(m, tea.KeyEsc)

    assert.Equal(t, ProjectDetailsState, got.state)
    assert.Zero(t, client.createCalls)
    assert.Zero(t, client.updateCalls)
}

func TestChangesCommandLoadsAndRendersBackendRows(t *testing.T) {
    client := &fakeClient{
        changeRows: []dto.Change{
            {
                ID:          "11",
                Ref:         "3",
                Slug:        "change-three",
                Title:       "Backend Change",
                ChangePhase: "backlog",
                ChangeTypes: []string{"feature", "test"},
                EpicID:      "5",
                EpicName:    "Epic Five",
                Spec:        "Backend spec",
                Done:        2,
                Total:       5,
                Completed:   40,
                Modified:    "2026-06-29T10:45:00Z",
            },
        },
        gotChange: dto.Change{
            ID:          "11",
            RefUUID:     "11111111-2222-4333-8444-555555555555",
            Ref:         "3",
            Slug:        "change-three",
            Title:       "Backend Change",
            ChangePhase: "backlog",
            ChangeTypes: []string{"feature", "test"},
            EpicID:      "5",
            EpicName:    "Epic Five",
            Spec:        "# Backend Change\n\nTypes: feature|test\n\nEpic: Epic Five\n\n## Problem Statement\nBody.",
            PR:          "Pull request summary.",
            PRUrl:       "https://github.com/divilla/project-manager/pull/107",
            AgentEdit:   true,
            Open:        true,
            Created:     "2026-06-29T08:15:00Z",
            Modified:    "2026-06-29T10:45:00Z",
        },
    }
    m := newModelWithOptionCatalog(client)
    m.currentProject = dto.Option{ID: "7", Label: "Project Seven"}
    m.width = 120

    got, cmd := sendCommand(m, "/changes")
    require.Equal(t, ChangesListState, got.state)
    require.NotNil(t, cmd)
    assert.True(t, got.changeList.Loading)

    got = applyMsg(got, cmd())

    assert.Equal(t, []string{"7"}, client.changeListProjectIDs)
    view := stripANSI(got.View())
    assert.Contains(t, view, "/filter-phase")
    assert.Contains(t, view, "/filter-type")
    assert.Contains(t, view, "/filter-epic")
    assert.Contains(t, view, "/filter-find")
    assert.Contains(t, view, "#Ref")
    assert.Contains(t, view, "Phase")
    assert.Contains(t, view, "Types")
    assert.Contains(t, view, "Epic")
    assert.Contains(t, view, "Title")
    assert.Contains(t, view, "Don")
    assert.Contains(t, view, "Tot")
    assert.Contains(t, view, "%")
    assert.Contains(t, view, "Modified")
    assert.Contains(t, view, "000003")
    assert.Contains(t, view, "backlog")
    assert.Contains(t, view, "Backend Change")
    assert.Contains(t, view, "feature|test")
    assert.Contains(t, view, "Epic Five")
    assert.Contains(t, view, "  2")
    assert.Contains(t, view, "  5")
    assert.Contains(t, view, " 40")
    assert.Contains(t, view, "2026-06-29 10.45")

    got, cmd = sendKey(got, tea.KeyEnter)
    require.NotNil(t, cmd)
    assert.Equal(t, ChangeDetailsState, got.state)
    got = applyMsg(got, cmd())

    assert.Equal(t, []int{11}, client.changeGetIDs)
    rawView := got.View()
    view = stripANSI(rawView)
    assert.Contains(t, view, "ChangeDetailsScreen")
    assert.Contains(t, view, "ID │ 11")
    assert.Contains(t, view, "Ref UUID │ 11111111-2222-4333-8444-555555555555")
    assert.Contains(t, view, "Ref │ 000003")
    assert.Contains(t, view, "Slug │ change-three")
    assert.Contains(t, view, "Phase │ backlog")
    assert.Contains(t, view, "Epic │ Epic Five")
    assert.Contains(t, view, "Types │ feature|test")
    assert.Contains(t, view, "Title │ Backend Change")
    assert.Contains(t, view, "───────────┼")
    assert.NotContains(t, view, "Epic Five                                                                                              \n───────────┼")
    assert.Less(t, strings.Index(view, "ID │ 11"), strings.Index(view, "Ref UUID │ 11111111-2222-4333-8444-555555555555"))
    assert.Less(t, strings.Index(view, "Ref UUID │ 11111111-2222-4333-8444-555555555555"), strings.Index(view, "Ref │ 000003"))
    assert.Less(t, strings.Index(view, "Ref │ 000003"), strings.Index(view, "Slug │ change-three"))
    assert.Less(t, strings.Index(view, "Slug │ change-three"), strings.Index(view, "Phase │ backlog"))
    assert.Less(t, strings.Index(view, "Phase │ backlog"), strings.Index(view, "Epic │ Epic Five"))
    assert.Less(t, strings.Index(view, "Epic │ Epic Five"), strings.Index(view, "Types │ feature|test"))
    assert.Less(t, strings.Index(view, "Types │ feature|test"), strings.Index(view, "Title │ Backend Change"))
    assert.NotContains(t, view, "Rows 1-")
    assert.Contains(t, rawView, lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Render("Backend Change"))

    got, _ = sendKey(got, tea.KeyPgDown)
    view = stripANSI(got.View())
    assert.Contains(t, view, "Spec │ # Backend Change")
    assert.Contains(t, view, "PR │ Pull request summary.")
    assert.Less(t, strings.Index(view, "Spec │ # Backend Change"), strings.Index(view, "PR │ Pull request summary."))

    got, _ = sendKey(got, tea.KeyPgDown)
    view = stripANSI(got.View())
    assert.Contains(t, view, "PR URL │ https://github.com/divilla/project-manager/pull/107")
    assert.Contains(t, view, "Agent Edit │ ✔")
    assert.Contains(t, view, "Complete │ 0/0 - 0%")
    assert.Less(t, strings.Index(view, "PR URL │ https://github.com/divilla/project-manager/pull/107"), strings.Index(view, "Agent Edit │ ✔"))
    assert.Less(t, strings.Index(view, "Agent Edit │ ✔"), strings.Index(view, "Complete │ 0/0 - 0%"))

    got, _ = sendKey(got, tea.KeyPgDown)
    view = stripANSI(got.View())
    assert.Contains(t, view, "Open │ ✅")
    assert.Contains(t, view, "Created │ 2026-06-29 08.15")
    assert.Contains(t, view, "Modified │ 2026-06-29 10.45")
}

func TestChangesTableTruncatesEpicAndTitleAtMaxWidth(t *testing.T) {
    longEpic := strings.Repeat("E", 25)
    longTitle := strings.Repeat("T", 90)
    m := NewModelWithClient(&fakeClient{})
    m.state = ChangesListState
    m.width = 220
    m.changeList = m.changeList.WithRows([]dto.Change{{
        ID:       "1",
        Ref:      "1",
        EpicName: longEpic,
        Title:    longTitle,
    }})

    view := stripANSI(m.View())

    assert.Contains(t, view, "Title")
    assert.Contains(t, view, strings.Repeat("E", 20))
    assert.NotContains(t, view, strings.Repeat("E", 21))
    assert.Contains(t, view, strings.Repeat("T", 80))
    assert.NotContains(t, view, strings.Repeat("T", 81))
    assert.NotContains(t, view, "...")
}

func TestChangesTableUsesNaturalWidthUntilTerminalIsSmaller(t *testing.T) {
    view := stripANSI(changes.TableView(changes.Model{}.WithRows([]dto.Change{{
        ID:          "1",
        Ref:         "1",
        ChangeTypes: []string{strings.Repeat("Y", 35)},
        EpicName:    strings.Repeat("E", 25),
        Title:       strings.Repeat("T", 90),
    }}), changes.Filters{}, 220, 1))
    lines := strings.Split(view, "\n")
    require.NotEmpty(t, lines)

    require.GreaterOrEqual(t, len(lines), 2)
    assert.Equal(t, 181, lipgloss.Width(lines[1]))
    assert.Contains(t, view, strings.Repeat("Y", 30))
    assert.NotContains(t, view, strings.Repeat("Y", 31))

    narrow := stripANSI(changes.TableView(changes.Model{}.WithRows([]dto.Change{{
        ID:          "1",
        Ref:         "1",
        ChangeTypes: []string{strings.Repeat("Y", 35)},
        EpicName:    strings.Repeat("E", 25),
        Title:       strings.Repeat("T", 90),
    }}), changes.Filters{}, 120, 1))
    narrowLines := strings.Split(narrow, "\n")
    require.NotEmpty(t, narrowLines)
    require.GreaterOrEqual(t, len(narrowLines), 2)
    assert.Equal(t, 120, lipgloss.Width(narrowLines[1]))
}

func TestChangesTableRendersPhaseColumnWidthAndColors(t *testing.T) {
    model := changes.Model{}.WithRows([]dto.Change{
        {ID: "1", Ref: "1", ChangePhase: "backlog", Title: "Backlog", Completed: 10},
        {ID: "2", Ref: "2", ChangePhase: "progress", Title: "Progress", Completed: 75},
    })

    raw := changes.TableView(model, changes.Filters{}, 220, 2)
    view := stripANSI(raw)

    assert.Contains(t, view, "backlog   ")
    assert.Contains(t, view, "progress  ")
    assert.Contains(t, raw, lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Render("progress  "))
    assert.Contains(t, raw, lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Render("Progress"))
    assert.Contains(t, raw, lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Render(" 75"))
}

func TestChangesListViewUsesLoadedPhaseColors(t *testing.T) {
    client := &fakeClient{
        phases: []dto.Option{
            {ID: "backlog", Label: "backlog", Color: "15"},
            {ID: "progress", Label: "progress", Color: "10"},
        },
    }
    m := newModelWithOptionCatalog(client)
    m.state = ChangesListState
    m.width = 220
    m.changeList = m.changeList.WithRows([]dto.Change{
        {ID: "1", Ref: "1", ChangePhase: "backlog", Title: "Backlog", Completed: 10},
        {ID: "2", Ref: "2", ChangePhase: "progress", Title: "Progress", Completed: 75},
    })

    raw := m.View()

    assert.Contains(t, raw, lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Render("progress  "))
}

func TestChangesTableKeyboardSelectionMatchesProjects(t *testing.T) {
    client := &fakeClient{
        gotChange: dto.Change{ID: "2", Title: "Second Change"},
    }
    m := newModelWithOptionCatalog(client)
    m.state = ChangesListState
    m.changeList = m.changeList.WithRows([]dto.Change{
        {ID: "1", Ref: "1", Title: "First Change"},
        {ID: "2", Ref: "2", Title: "Second Change"},
    })

    got, _ := sendKey(m, tea.KeyUp)
    assert.Equal(t, 0, got.changeList.Selected)

    got, _ = sendKey(got, tea.KeyDown)
    assert.Equal(t, 1, got.changeList.Selected)

    got, _ = sendKey(got, tea.KeyDown)
    assert.Equal(t, 1, got.changeList.Selected)

    got, _ = sendKey(got, tea.KeyUp)
    assert.Equal(t, 0, got.changeList.Selected)

    got, _ = sendKey(got, tea.KeyDown)
    got, cmd := sendKeyMsg(got, tea.KeyMsg{Type: tea.KeyCtrlJ})
    require.NotNil(t, cmd)
    assert.Equal(t, ChangeDetailsState, got.state)

    got = applyMsg(got, cmd())
    assert.Equal(t, []int{2}, client.changeGetIDs)
    assert.Equal(t, client.gotChange, got.changeList.Detail)
}

func TestChangesTableIsBoxedAndScrollsSelectedRowIntoView(t *testing.T) {
    m := NewModelWithClient(&fakeClient{})
    m.state = ChangesListState
    m.height = 15
    m.width = 120
    m.changeList = m.changeList.WithRows([]dto.Change{
        {ID: "1", Ref: "1", Title: "Change One"},
        {ID: "2", Ref: "2", Title: "Change Two"},
        {ID: "3", Ref: "3", Title: "Change Three"},
        {ID: "4", Ref: "4", Title: "Change Four"},
        {ID: "5", Ref: "5", Title: "Change Five"},
    })

    view := stripANSI(m.View())
    assert.Contains(t, view, "┌")
    assert.Contains(t, view, "└")
    assert.Contains(t, view, "Change One")
    assert.Contains(t, view, "Change Three")
    assert.NotContains(t, view, "Change Four")
    assert.Contains(t, view, "Rows 1-3 of 5")

    got, _ := sendKey(m, tea.KeyDown)
    got, _ = sendKey(got, tea.KeyDown)
    got, _ = sendKey(got, tea.KeyDown)

    assert.Equal(t, 3, got.changeList.Selected)
    assert.Equal(t, 1, got.changeList.Offset)
    view = stripANSI(got.View())
    assert.NotContains(t, view, "Change One")
    assert.Contains(t, view, "Change Four")
    assert.Contains(t, view, "Rows 2-4 of 5")

    got, _ = sendKey(got, tea.KeyPgDown)
    assert.Equal(t, 4, got.changeList.Selected)
    assert.Equal(t, 2, got.changeList.Offset)
    view = stripANSI(got.View())
    assert.Contains(t, view, "Change Five")
    assert.Contains(t, view, "Rows 3-5 of 5")

    got, _ = sendKey(got, tea.KeyPgUp)
    assert.Equal(t, 1, got.changeList.Selected)
    assert.Equal(t, 1, got.changeList.Offset)
    view = stripANSI(got.View())
    assert.Contains(t, view, "Change Two")
    assert.Contains(t, view, "Rows 2-4 of 5")
}

func TestChangesEnterWithNoSelectableRowErrors(t *testing.T) {
    m := NewModelWithClient(&fakeClient{})
    m.state = ChangesListState

    got, _ := sendKey(m, tea.KeyEnter)

    assert.Equal(t, ChangesListState, got.state)
    assert.NotEmpty(t, got.err)
}

func TestNewChangeRequiresCurrentProject(t *testing.T) {
    client := &fakeClient{}
    m := newModelWithOptionCatalog(client)
    m.state = ChangesListState
    m.agentWorkspace = t.TempDir()

    got, cmd := sendCommand(m, "/new-change")

    require.Nil(t, cmd)
    assert.Equal(t, ChangesListState, got.state)
    assert.Equal(t, "current project must be numeric", got.err)
    assert.Zero(t, client.changeCreateCalls)
}

func TestNewChangeCreatesWorkspaceAndOpensDefEditor(t *testing.T) {
    workspaceDir := t.TempDir()
    refUUID := uuid.Must(uuid.FromString("0198a86f-9b8a-7d89-ae5b-6f25b528b04c"))
    m := NewModelWithClient(&fakeClient{})
    m.state = ChangesListState
    m.currentProject = dto.Option{ID: "7", Label: "Project Seven"}
    m.agentWorkspace = workspaceDir
    m.newChangeUUID = func() (uuid.UUID, error) { return refUUID, nil }

    got, cmd := sendCommand(m, "/new-change")

    require.NotNil(t, cmd)
    assert.Equal(t, CreateDefState, got.state)
    assert.Equal(t, agent.StageDefEntry, got.agentFlow.Stage)
    stageDir := filepath.Join(workspaceDir, refUUID.String(), agent.ArtifactStage)
    assert.Equal(t, stageDir, got.agentFlow.Workspace.Dir)
    content, err := os.ReadFile(filepath.Join(stageDir, agent.OutputFileName))
    require.NoError(t, err)
    assert.Empty(t, string(content))
    require.FileExists(t, filepath.Join(stageDir, agent.InputFileName))
}

func TestNewChangeUsesRepositoryRelativeTempDirAndIgnoresConfiguredTempDir(t *testing.T) {
    repositoryRoot := t.TempDir()
    refUUID := uuid.Must(uuid.FromString("0198a86f-9b8a-7d89-ae5b-6f25b528b04c"))
    m := newModelWithConfig(&fakeClient{}, testAppConfig(appConfig{RepositoryRoot: repositoryRoot}))
    m.state = ChangesListState
    m.currentProject = dto.Option{ID: "7", Label: "Project Seven"}
    m.newChangeUUID = func() (uuid.UUID, error) { return refUUID, nil }

    got, cmd := sendCommand(m, "/new-change")

    require.NotNil(t, cmd)
    assert.Equal(t, CreateDefState, got.state)
    expected := filepath.Join(repositoryRoot, agent.TempDir, refUUID.String(), agent.ArtifactStage)
    assert.Equal(t, expected, got.agentFlow.Workspace.Dir)
    require.FileExists(t, filepath.Join(expected, agent.OutputFileName))
}

func TestNewChangeReplacesRegularWorkspaceFile(t *testing.T) {
    parent := t.TempDir()
    workspaceDir := filepath.Join(parent, "mch")
    require.NoError(t, os.WriteFile(workspaceDir, []byte("file"), 0o644))
    m := NewModelWithClient(&fakeClient{})
    m.state = ChangesListState
    m.currentProject = dto.Option{ID: "7", Label: "Project Seven"}
    m.agentWorkspace = workspaceDir

    got, cmd := sendCommand(m, "/new-change")

    require.Nil(t, cmd)
    assert.Equal(t, ChangesListState, got.state)
    info, err := os.Stat(workspaceDir)
    require.NoError(t, err)
    assert.True(t, info.Mode().IsRegular())
    assert.Contains(t, got.err, "not a directory")
}

func TestNewChangeDoesNotReuseExistingWorkspace(t *testing.T) {
    workspaceDir := t.TempDir()
    oldRef := "0198a86f-9b8a-7d89-ae5b-6f25b528b04b"
    oldOutput := filepath.Join(workspaceDir, oldRef, agent.ArtifactStage, agent.OutputFileName)
    require.NoError(t, os.MkdirAll(filepath.Dir(oldOutput), 0o755))
    require.NoError(t, os.WriteFile(oldOutput, []byte("# Existing def"), 0o644))
    newRef := uuid.Must(uuid.FromString("0198a86f-9b8a-7d89-ae5b-6f25b528b04c"))
    m := NewModelWithClient(&fakeClient{})
    m.state = ChangesListState
    m.currentProject = dto.Option{ID: "7", Label: "Project Seven"}
    m.agentWorkspace = workspaceDir
    m.newChangeUUID = func() (uuid.UUID, error) { return newRef, nil }

    got, cmd := sendCommand(m, "/new-change")

    require.NotNil(t, cmd)
    assert.Equal(t, CreateDefState, got.state)
    require.FileExists(t, filepath.Join(workspaceDir, newRef.String(), agent.ArtifactStage, agent.InputFileName))
    content, err := os.ReadFile(oldOutput)
    require.NoError(t, err)
    assert.Equal(t, "# Existing def", string(content))
}

func TestNewChangeUUIDWorkspaceCreateAndRewriteContract(t *testing.T) {
    client := &fakeClient{
        types:         []dto.Option{{ID: "feature", Label: "feature"}},
        createdChange: dto.Change{ID: "12", Title: "New Change", Def: "# New Change\n\nTypes: feature\n\nDef"},
    }
    root := t.TempDir()
    refUUID := uuid.Must(uuid.FromString("0198a86f-9b8a-7d89-ae5b-6f25b528b04c"))
    m := newModelWithOptionCatalog(client)
    m.state = ChangesListState
    m.currentProject = dto.Option{ID: "7", Label: "Project Seven"}
    m.agentWorkspace = root
    m.newChangeUUID = func() (uuid.UUID, error) { return refUUID, nil }

    started, editorCmd := sendCommand(m, "/new-change")
    require.NotNil(t, editorCmd)
    workspace := started.agentFlow.Workspace
    def := "# New Change\n\nTypes: feature\n\nDef"
    require.NoError(t, os.WriteFile(workspace.OutputPath(), []byte(def), 0o644))
    confirmed := applyMsg(started, editorFinishedMsg{source: CreateDefState, content: def})
    assert.Equal(t, "Create Change?", confirmed.dropdown.label)
    assert.Zero(t, client.changeCreateCalls)

    updated, createCmd := confirmed.confirmDropdown()
    require.NotNil(t, createCmd)
    createdModel, typesCmd := updated.(Model).Update(createCmd())
    created := createdModel.(Model)
    require.Len(t, client.changeCreateInputs, 1)
    assert.Equal(t, dto.ChangeCreateInput{ProjectID: 7, RefUUID: refUUID.String(), Title: "New Change", Def: def}, client.changeCreateInputs[0])
    assert.Equal(t, ChangeDetailsState, created.state)
    assert.Equal(t, "12", created.changeList.Detail.ID)
    assert.Empty(t, client.changeTypesUpdates)
    require.NotNil(t, typesCmd)

    created = applyMsg(created, typesCmd())
    assert.Equal(t, [][]string{{"feature"}}, client.changeTypesUpdates)
    assert.Empty(t, client.changeGetIDs)
    assert.Equal(t, RewriteDefState, created.state)
    input, err := os.ReadFile(workspace.InputPath())
    require.NoError(t, err)
    assert.Equal(t, def, string(input))
    require.DirExists(t, workspace.RootDir)
}

func TestNewChangePostCreateFailureRecoversAsExistingChange(t *testing.T) {
    const def = "# New Change\n\nTypes: feature\n\nDef"
    newFailedModel := func(t *testing.T) (Model, *fakeClient, agent.Workspace) {
        t.Helper()
        client := &fakeClient{
            createdChange:   dto.Change{ID: "12", RefUUID: "0198a86f-9b8a-7d89-ae5b-6f25b528b04c", Title: "New Change", Def: def},
            gotChange:       dto.Change{ID: "12", RefUUID: "0198a86f-9b8a-7d89-ae5b-6f25b528b04c", Title: "New Change", Def: def, ChangeTypes: []string{"feature"}},
            changeUpdateErr: errors.New("type update failed"),
        }
        m := NewModelWithClient(client)
        m.state = ChangesListState
        m.currentProject = dto.Option{ID: "7"}
        m.agentWorkspace = t.TempDir()
        refUUID := uuid.Must(uuid.FromString("0198a86f-9b8a-7d89-ae5b-6f25b528b04c"))
        m.newChangeUUID = func() (uuid.UUID, error) { return refUUID, nil }

        started, _ := sendCommand(m, "/new-change")
        workspace := started.agentFlow.Workspace
        require.NoError(t, os.WriteFile(workspace.OutputPath(), []byte(def), 0o644))
        confirmed := applyMsg(started, editorFinishedMsg{source: CreateDefState, content: def})
        creating, createCmd := confirmed.confirmDropdown()
        require.NotNil(t, createCmd)
        createdModel, typesCmd := creating.(Model).Update(createCmd())
        created := createdModel.(Model)
        require.NotNil(t, typesCmd)
        failed := applyMsg(created, typesCmd())

        assert.Equal(t, ChangeDetailsState, failed.state)
        assert.Equal(t, "12", failed.changeList.Detail.ID)
        assert.Equal(t, dropdownPersistedDef, failed.dropdown.kind)
        assert.Equal(t, 1, client.changeCreateCalls)
        return failed, client, workspace
    }

    t.Run("cancel stays on details", func(t *testing.T) {
        m, client, workspace := newFailedModel(t)
        m.dropdown.highlighted = 1
        cancelled, cmd := m.confirmDropdown()
        got := cancelled.(Model)

        require.Nil(t, cmd)
        assert.Equal(t, ChangeDetailsState, got.state)
        assert.Equal(t, "12", got.changeList.Detail.ID)
        assert.Equal(t, 1, client.changeCreateCalls)
        require.DirExists(t, workspace.RootDir)
    })

    t.Run("fix with unchanged definition stays on details", func(t *testing.T) {
        m, client, workspace := newFailedModel(t)
        fixing, editorCmd := m.confirmDropdown()
        fixed := fixing.(Model)

        require.NotNil(t, editorCmd)
        assert.Equal(t, UpdateDefState, fixed.state)
        got := applyMsg(fixed, editorFinishedMsg{source: UpdateDefState, content: def})
        assert.Equal(t, ChangeDetailsState, got.state)
        assert.Equal(t, 1, client.changeCreateCalls)
        assert.Zero(t, client.changeDefUpdateCalls)
        require.DirExists(t, workspace.RootDir)
    })

    t.Run("fix with invalid def keeps persisted recovery", func(t *testing.T) {
        m, client, workspace := newFailedModel(t)
        fixing, editorCmd := m.confirmDropdown()
        fixed := fixing.(Model)
        require.NotNil(t, editorCmd)

        invalid := "missing title"
        require.NoError(t, os.WriteFile(workspace.OutputPath(), []byte(invalid), 0o644))
        failed := applyMsg(fixed, editorFinishedMsg{source: UpdateDefState, content: invalid})

        assert.Equal(t, ChangeDetailsState, failed.state)
        assert.Equal(t, "12", failed.changeList.Detail.ID)
        assert.Equal(t, dropdownPersistedDef, failed.dropdown.kind)
        assert.Equal(t, []dto.Option{{ID: "/fix", Label: "/fix"}, {ID: "/cancel", Label: "/cancel"}}, failed.dropdown.options)
        assert.Equal(t, 1, client.changeCreateCalls)
        assert.Zero(t, client.changeDefUpdateCalls)
        require.DirExists(t, workspace.RootDir)

        failed.dropdown.highlighted = 1
        cancelled, cmd := failed.confirmDropdown()
        got := cancelled.(Model)
        require.Nil(t, cmd)
        assert.Equal(t, ChangeDetailsState, got.state)
        assert.Equal(t, "12", got.changeList.Detail.ID)
        require.DirExists(t, workspace.RootDir)
    })

    t.Run("fix with changed definition updates existing change", func(t *testing.T) {
        m, client, workspace := newFailedModel(t)
        client.changeUpdateErr = nil
        fixing, editorCmd := m.confirmDropdown()
        fixed := fixing.(Model)
        require.NotNil(t, editorCmd)

        changed := "# New Change\n\nTypes: feature\n\nCorrected def"
        require.NoError(t, os.WriteFile(workspace.OutputPath(), []byte(changed), 0o644))
        updated, saveCmd := fixed.Update(editorFinishedMsg{source: UpdateDefState, content: changed})
        require.NotNil(t, saveCmd)
        saved := applyMsg(updated.(Model), changeDefUpdateForRewriteCommand(client, 12, changed)())

        assert.Equal(t, 1, client.changeCreateCalls)
        assert.Equal(t, []string{changed}, client.changeDefUpdates)
        assert.Equal(t, RewriteDefState, saved.state)
    })
}

func TestNewChangeValidatesArtifactOnlyAfterYes(t *testing.T) {
    tests := []struct {
        name    string
        def     string
        wantErr string
    }{
        {
            name:    "missing title",
            def:     "missing title",
            wantErr: "definition title is required",
        },
        {
            name:    "missing body",
            def:     "# New Change\n\n",
            wantErr: "definition body is required",
        },
        {
            name:    "types without body",
            def:     "# New Change\n\nTypes:",
            wantErr: "definition body is required",
        },
        {
            name: "leading blank before title",
            def:  "\n# New Change\n\nDef",
        },
        {
            name: "supported type",
            def:  "# New Change\n\nTypes: feature\n\nDef",
        },
        {
            name: "blank types",
            def:  "# New Change\n\nTypes:\n\nDef",
        },
        {
            name: "epic is ordinary body text",
            def:  "# New Change\n\nTypes: feature\n\nEpic: Epic Five\n\nDef",
        },
        {
            name: "missing types",
            def:  "# New Change\n\nDef",
        },
        {
            name: "types moved below body",
            def:  "# New Change\n\nDef\n\nTypes: feature",
        },
        {
            name: "no blank after title",
            def:  "# New Change\nTypes: feature\n\nDef",
        },
        {
            name: "extra blank after title",
            def:  "# New Change\n\n\nTypes: feature\n\nDef",
        },
        {
            name: "no blank after types",
            def:  "# New Change\n\nTypes: feature\nDef",
        },
        {
            name: "extra blank after types",
            def:  "# New Change\n\nTypes: feature\n\n\nDef",
        },
        {
            name: "unsupported type",
            def:  "# New Change\n\nTypes: unsupported\n\nDef",
        },
    }

    for index, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            client := &fakeClient{
                types:         []dto.Option{{ID: "feature", Label: "feature"}},
                createdChange: dto.Change{ID: "12"},
                gotChange:     dto.Change{ID: "12"},
            }
            m := newModelWithOptionCatalog(client)
            m.state = ChangesListState
            m.currentProject = dto.Option{ID: "7", Label: "Project Seven"}
            m.agentWorkspace = t.TempDir()
            refUUID := uuid.Must(uuid.FromBytes([]byte{1, 152, 168, 111, 155, 138, 125, 137, 174, 91, 111, 37, 181, 40, 176, byte(index + 1)}))
            m.newChangeUUID = func() (uuid.UUID, error) { return refUUID, nil }

            started, editorCmd := sendCommand(m, "/new-change")
            require.NotNil(t, editorCmd)
            require.NoError(t, os.WriteFile(started.agentFlow.Workspace.OutputPath(), []byte(tt.def), 0o644))
            got := applyMsg(started, editorFinishedMsg{source: CreateDefState, content: tt.def})

            assert.Zero(t, client.epicCalls)
            assert.Zero(t, client.changeCreateCalls)
            assert.Empty(t, got.err)
            assert.Equal(t, agent.StageCreateConfirmation, got.agentFlow.Stage)
            assert.Equal(t, "Create Change?", got.dropdown.label)

            creating, createCmd := got.confirmDropdown()
            require.NotNil(t, createCmd)
            created := applyMsg(creating.(Model), createCmd())
            if tt.wantErr != "" {
                assert.Equal(t, tt.wantErr, created.err)
                assert.Equal(t, []dto.Option{{ID: "/fix", Label: "/fix"}, {ID: "/cancel", Label: "/cancel"}}, created.dropdown.options)
                assert.Zero(t, client.changeCreateCalls)
                return
            }
            assert.Empty(t, created.err)
            assert.Equal(t, 1, client.changeCreateCalls)
        })
    }
}

func TestNewChangeYesValidatesFinalRereadBeforeCreate(t *testing.T) {
    client := &fakeClient{createdChange: dto.Change{ID: "12"}}
    m := NewModelWithClient(client)
    m.state = ChangesListState
    m.currentProject = dto.Option{ID: "7"}
    m.agentWorkspace = t.TempDir()
    refUUID := uuid.Must(uuid.FromString("0198a86f-9b8a-7d89-ae5b-6f25b528b04c"))
    m.newChangeUUID = func() (uuid.UUID, error) { return refUUID, nil }

    started, _ := sendCommand(m, "/new-change")
    def := "# New Change\n\nValid body"
    require.NoError(t, os.WriteFile(started.agentFlow.Workspace.OutputPath(), []byte(def), 0o644))
    confirmed := applyMsg(started, editorFinishedMsg{source: CreateDefState, content: def})
    assert.Equal(t, "Create Change?", confirmed.dropdown.label)

    require.NoError(t, os.WriteFile(started.agentFlow.Workspace.OutputPath(), []byte("# New Change\n\nTypes:"), 0o644))
    creating, createCmd := confirmed.confirmDropdown()
    require.NotNil(t, createCmd)
    failed := applyMsg(creating.(Model), createCmd())

    assert.Equal(t, "definition body is required", failed.err)
    assert.Equal(t, []dto.Option{{ID: "/fix", Label: "/fix"}, {ID: "/cancel", Label: "/cancel"}}, failed.dropdown.options)
    assert.Zero(t, client.changeCreateCalls)
}

func TestNewChangeCancellationAndFixContract(t *testing.T) {
    newModel := func(t *testing.T, suffix byte) Model {
        t.Helper()
        client := &fakeClient{types: []dto.Option{{ID: "feature", Label: "feature"}}}
        m := newModelWithOptionCatalog(client)
        m.state = ChangesListState
        m.currentProject = dto.Option{ID: "7"}
        m.agentWorkspace = t.TempDir()
        ref := uuid.Must(uuid.FromBytes([]byte{1, 152, 168, 111, 155, 138, 125, 137, 174, 91, 111, 37, 181, 40, 176, suffix}))
        m.newChangeUUID = func() (uuid.UUID, error) { return ref, nil }
        return m
    }

    t.Run("unchanged output", func(t *testing.T) {
        m, _ := sendCommand(newModel(t, 1), "/new-change")
        root := m.agentFlow.Workspace.RootDir
        got := applyMsg(m, editorFinishedMsg{source: CreateDefState})
        assert.Equal(t, ChangesListState, got.state)
        assert.NoDirExists(t, root)
    })

    t.Run("no", func(t *testing.T) {
        m, _ := sendCommand(newModel(t, 2), "/new-change")
        workspace := m.agentFlow.Workspace
        def := "# New Change\n\nTypes: feature\n\nDef"
        require.NoError(t, os.WriteFile(workspace.OutputPath(), []byte(def), 0o644))
        m = applyMsg(m, editorFinishedMsg{source: CreateDefState, content: def})
        m.dropdown.highlighted = 1
        updated, cmd := m.confirmDropdown()
        require.Nil(t, cmd)
        assert.Equal(t, ChangesListState, updated.(Model).state)
        assert.NoDirExists(t, workspace.RootDir)
    })

    t.Run("fix then unchanged", func(t *testing.T) {
        m, _ := sendCommand(newModel(t, 3), "/new-change")
        workspace := m.agentFlow.Workspace
        require.NoError(t, os.WriteFile(workspace.OutputPath(), []byte("missing title"), 0o644))
        m = applyMsg(m, editorFinishedMsg{source: CreateDefState, content: "missing title"})
        creating, createCmd := m.confirmDropdown()
        require.NotNil(t, createCmd)
        m = applyMsg(creating.(Model), createCmd())
        assert.Equal(t, []dto.Option{{ID: "/fix", Label: "/fix"}, {ID: "/cancel", Label: "/cancel"}}, m.dropdown.options)
        updated, editorCmd := m.confirmDropdown()
        require.NotNil(t, editorCmd)
        fixed := updated.(Model)
        baseline, err := os.ReadFile(workspace.InputPath())
        require.NoError(t, err)
        assert.Equal(t, "missing title", string(baseline))
        cancelled := applyMsg(fixed, editorFinishedMsg{source: CreateDefState, content: "missing title"})
        assert.Equal(t, ChangesListState, cancelled.state)
        assert.NoDirExists(t, workspace.RootDir)
    })
}

func TestNewChangeFailuresPreserveOrCleanWorkspaceAsSpecified(t *testing.T) {
    m := newModelWithOptionCatalog(&fakeClient{types: []dto.Option{{ID: "feature", Label: "feature"}}, changeCreateErr: errors.New("create failed")})
    m.state = ChangesListState
    m.currentProject = dto.Option{ID: "7"}
    m.agentWorkspace = t.TempDir()
    refUUID := uuid.Must(uuid.FromString("0198a86f-9b8a-7d89-ae5b-6f25b528b04c"))
    m.newChangeUUID = func() (uuid.UUID, error) { return refUUID, nil }
    m, _ = sendCommand(m, "/new-change")
    workspace := m.agentFlow.Workspace
    def := "# New Change\n\nTypes: feature\n\nDef"
    require.NoError(t, os.WriteFile(workspace.OutputPath(), []byte(def), 0o644))
    m = applyMsg(m, editorFinishedMsg{source: CreateDefState, content: def})
    updated, createCmd := m.confirmDropdown()
    m = applyMsg(updated.(Model), createCmd())
    assert.Equal(t, []dto.Option{{ID: "/fix", Label: "/fix"}, {ID: "/cancel", Label: "/cancel"}}, m.dropdown.options)
    require.FileExists(t, workspace.InputPath())
    require.FileExists(t, workspace.OutputPath())

    m = NewModelWithClient(&fakeClient{})
    m.state = ChangesListState
    m.currentProject = dto.Option{ID: "7"}
    m.agentWorkspace = t.TempDir()
    m.newChangeUUID = func() (uuid.UUID, error) { return uuid.Nil, errors.New("uuid failed") }
    failed, cmd := sendCommand(m, "/new-change")
    require.Nil(t, cmd)
    assert.Equal(t, "uuid failed", failed.err)
    assert.NoDirExists(t, filepath.Join(m.agentWorkspace, uuid.Nil.String()))
}

func TestDefPreviewMarkdownColorsFinalView(t *testing.T) {
    previousProfile := lipgloss.ColorProfile()
    t.Cleanup(func() {
        lipgloss.SetColorProfile(previousProfile)
    })
    lipgloss.SetColorProfile(termenv.ANSI256)
    workspaceDir := t.TempDir()
    workspace := agent.Workspace{Dir: workspaceDir}
    require.NoError(t, workspace.WriteDef("# Colored definition\n\nPlain text with `code` and [link](https://example.test).\n\n- item"))

    m := NewModelWithClient(&fakeClient{})
    m.state = CreateDefState
    m.agentFlow = agent.NewModelWithWorkspace(workspaceDir)
    m.openDropdown(CreateDefState, dropdownAgent, CreateDefState, CreateDefState, "Resume definition?", []dto.Option{
        {ID: "/resume", Label: "/resume"},
        {ID: "/clear", Label: "/clear"},
        {ID: "/cancel", Label: "/cancel"},
    }, false)

    view := m.View()

    assertDefBeforePrompt(t, view, "# Colored definition", "Resume definition?")
    assert.Contains(t, view, markdownPreviewStyles.heading.Render("# Colored definition"))
    assert.Contains(t, view, markdownPreviewStyles.text.Render("Plain text with "))
    assert.Contains(t, view, markdownPreviewStyles.code.Render("`code`"))
    assert.Contains(t, view, markdownPreviewStyles.link.Render("[link](https://example.test)"))
    assert.Contains(t, view, markdownPreviewStyles.listMarker.Render("- "))
}

func TestAgentDefEmptyValidatesAfterYesWithoutCodexOrCreate(t *testing.T) {
    client := &fakeClient{}
    runner := &fakeAgentRunner{}
    m := NewModelWithClient(client)
    m.state = CreateDefState
    m.currentProject = dto.Option{ID: "7", Label: "Project Seven"}
    m.agentWorkspace = t.TempDir()
    m.agentRunner = runner
    m.agentFlow = agent.NewModelWithWorkspace(m.agentWorkspace)
    m.agentFlow.Stage = agent.StageDefEntry

    got := applyMsg(m, editorFinishedMsg{source: CreateDefState, content: " \n"})

    assert.Equal(t, CreateDefState, got.state)
    assert.Equal(t, dropdownDef, got.dropdown.kind)
    assert.Equal(t, "Create Change?", got.dropdown.label)
    creating, createCmd := got.confirmDropdown()
    require.NotNil(t, createCmd)
    failed := applyMsg(creating.(Model), createCmd())
    assert.Equal(t, "definition title is required", failed.err)
    assert.Equal(t, []dto.Option{{ID: "/fix", Label: "/fix"}, {ID: "/cancel", Label: "/cancel"}}, failed.dropdown.options)
    assert.Empty(t, runner.rewriteSessions)
    assert.Zero(t, client.changeCreateCalls)
}

func TestAgentResumeUnchangedValidDefPromptsCreateConfirmation(t *testing.T) {
    client := &fakeClient{}
    runner := &fakeAgentRunner{}
    m := NewModelWithClient(client)
    m.state = CreateDefState
    m.currentProject = dto.Option{ID: "7", Label: "Project Seven"}
    m.agentWorkspace = t.TempDir()
    m.agentRunner = runner
    m.agentFlow = agent.NewModelWithWorkspace(m.agentWorkspace)
    m.agentFlow.Stage = agent.StageDefEntry
    m.agentFlow.DefEntryContent = "# Existing Definition\n\nexisting definition"

    got := applyMsg(m, editorFinishedMsg{source: CreateDefState, content: "# Existing Definition\n\nexisting definition"})

    assert.Equal(t, CreateDefState, got.state)
    assert.Equal(t, agent.StageCreateConfirmation, got.agentFlow.Stage)
    assert.Equal(t, "Create Change?", got.dropdown.label)
    assert.Empty(t, runner.rewriteSessions)
    assert.Zero(t, client.changeCreateCalls)
}

func TestAgentResumeChangedPromptsCreateConfirmation(t *testing.T) {
    runner := &fakeAgentRunner{}
    m := NewModelWithClient(&fakeClient{})
    m.state = CreateDefState
    m.currentProject = dto.Option{ID: "7", Label: "Project Seven"}
    m.agentWorkspace = t.TempDir()
    m.agentRunner = runner
    m.agentFlow = agent.NewModelWithWorkspace(m.agentWorkspace)
    m.agentFlow.Stage = agent.StageDefEntry
    m.agentFlow.DefEntryContent = "existing definition"

    got, cmd := m.Update(editorFinishedMsg{source: CreateDefState, content: "# Changed Definition\n\nchanged definition"})
    model := got.(Model)

    require.NotNil(t, cmd)
    assert.Equal(t, CreateDefState, model.state)
    assert.Equal(t, agent.StageCreateConfirmation, model.agentFlow.Stage)
    assert.Equal(t, dropdownDef, model.dropdown.kind)
    assert.Equal(t, "Create Change?", model.dropdown.label)
    assert.Empty(t, runner.rewriteSessions)
}

func TestAgentRunningViewRendersInsteadOfChangeList(t *testing.T) {
    m := NewModelWithClient(&fakeClient{})
    m.state = ChangesListState
    m.width = 120
    m.changeList = m.changeList.WithRows([]dto.Change{{ID: "12", Title: "Hidden While Running"}})
    m.agentFlow = agent.NewModelWithWorkspace(t.TempDir())
    m.agentFlow.Stage = agent.StageAIRunning
    m.agentFlow.SessionID = "session-1"
    m.agentFlow.CommandOutput = agent.FormatCommandOutput("{\"type\":\"thread.started\",\"thread_id\":\"session-1\"}") + "\nfinal output:\nNope"
    m.agentElapsed = 3
    m.agentMessageCount = 2

    view := stripANSI(m.View())

    assert.Contains(t, view, "AgentRunningScreen")
    assert.Contains(t, view, "def-write - session-1 - started - 00:03 - ..·")
    assert.Contains(t, view, "Codex output:")
    assert.Contains(t, view, "thread started: session-1")
    assert.Contains(t, view, "final output:")
    assert.Contains(t, view, "Nope")
    assert.NotContains(t, view, "Hidden While Running")
}

func TestAgentRunningViewUsesBlackColoredViewportWithProgressAndPromptBelow(t *testing.T) {
    previousProfile := lipgloss.ColorProfile()
    t.Cleanup(func() {
        lipgloss.SetColorProfile(previousProfile)
    })
    lipgloss.SetColorProfile(termenv.ANSI256)

    m := NewModelWithClient(&fakeClient{})
    m.state = RewriteDefState
    m.width = 100
    m.height = 19
    m.agentFlow = agent.NewModelWithWorkspace(t.TempDir())
    m.agentFlow.Stage = agent.StageAIRunning
    m.agentFlow.SessionID = "resumed-session"
    m.agentSessionResumed = true
    m.agentFlow.CommandOutput = strings.Join([]string{
        "thread started: session-1",
        "turn started",
        "assistant: reviewing the def",
        "running command: git status --short",
        "command completed (completed): exit 0",
        "turn completed",
        "final output:",
        "Done.",
    }, "\n")
    m.agentElapsed = 7
    m.agentMessageCount = 8
    m.refreshAgentViewport(true)

    view := m.View()
    plain := stripANSI(view)

    assert.Contains(t, view, "\x1b[40m")
    assert.NotContains(t, view, "\x1b[48;5;240m")
    assert.Contains(t, view, styles.Default.AccentPurple.Bold(true).Render("Codex output:"))
    assert.Contains(t, view, styles.Default.AccentCyan.Bold(true).Render("def-write - resumed-session - resumed - 00:07 - ........·"))
    assert.Contains(t, plain, defaultInputPlaceholder)
    assert.Less(t, strings.Index(plain, "Done."), strings.Index(plain, "def-write - resumed-session - resumed"))
    assert.Less(t, strings.Index(plain, "def-write - resumed-session - resumed"), strings.Index(plain, defaultInputPlaceholder))

    slashed, cmd := sendRune(m, '/')
    require.Nil(t, cmd)
    assert.True(t, slashed.hasDropdown())
    assert.Empty(t, slashed.input.Value())
    assert.Contains(t, stripANSI(slashed.View()), "Codex output:")
    assert.Contains(t, stripANSI(slashed.View()), "Commands: no options")
}

func TestAgentRunningViewCyclesMessageProgressAtAvailableWidth(t *testing.T) {
    const (
        width     = 100
        sessionID = "33333333-3333-3333-3333-333333333333"
    )
    m := NewModelWithClient(&fakeClient{})
    m.state = RewriteDefState
    m.width = width
    m.agentFlow = agent.NewModelWithWorkspace(t.TempDir())
    m.agentFlow.Stage = agent.StageAIRunning
    m.agentFlow.SessionID = sessionID

    prefix := "def-write - " + sessionID + " - started - 00:00 - "
    availableDotWidth := width - lipgloss.Width(prefix) - lipgloss.Width(m.agentSpinner.View())
    require.Positive(t, availableDotWidth)

    m.agentMessageCount = availableDotWidth
    lines := strings.Split(stripANSI(m.agentRunningView(width)), "\n")
    progress := lines[len(lines)-1]
    assert.Equal(t, prefix+"·", strings.TrimRight(progress, " "))
    assert.Equal(t, width, lipgloss.Width(progress))

    m.agentMessageCount++
    lines = strings.Split(stripANSI(m.agentRunningView(width)), "\n")
    progress = lines[len(lines)-1]
    assert.Equal(t, prefix+".·", strings.TrimRight(progress, " "))
    assert.Equal(t, width, lipgloss.Width(progress))
}

func TestAgentRunningViewportScrollsAndDoesNotStealPositionOnOutput(t *testing.T) {
    m := NewModelWithClient(&fakeClient{})
    m.state = RewriteDefState
    m.width = 80
    m.height = 13
    m.agentFlow = agent.NewModelWithWorkspace(t.TempDir())
    m.agentFlow.Stage = agent.StageAIRunning
    for line := 1; line <= 20; line++ {
        m.agentFlow.CommandOutput = appendCommandOutput(m.agentFlow.CommandOutput, fmt.Sprintf("output line %02d", line))
    }
    m.refreshAgentViewport(false)
    m.agentViewport.GotoTop()

    top := stripANSI(m.View())
    assert.Contains(t, top, "output line 01")
    assert.NotContains(t, top, "output line 20")

    paged, _ := sendKey(m, tea.KeyPgDown)
    pagedView := stripANSI(paged.View())
    assert.NotContains(t, pagedView, "output line 01")

    scrolled, _ := paged.Update(tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonWheelDown})
    mouseView := stripANSI(scrolled.(Model).View())
    assert.NotEqual(t, pagedView, mouseView)

    atTop := applyMsg(m, agentCommandOutputMsg{output: "output line 21"})
    assert.Zero(t, atTop.agentViewport.YOffset)
    assert.Equal(t, 1, atTop.agentMessageCount)
    assert.Contains(t, stripANSI(atTop.View()), "output line 01")

    atTop.agentViewport.GotoBottom()
    atBottom := applyMsg(atTop, agentCommandOutputMsg{
        output: `{"type":"thread.started","thread_id":"live-session"}`,
    })
    assert.True(t, atBottom.agentViewport.AtBottom())
    assert.Equal(t, 2, atBottom.agentMessageCount)
    assert.Equal(t, "live-session", atBottom.agentFlow.SessionID)
    assert.Contains(t, stripANSI(atBottom.View()), "thread started: live-session")
}

func TestAgentCreateDefValidatesAfterYesBeforeCreate(t *testing.T) {
    workspace := agent.Workspace{Dir: t.TempDir()}
    require.NoError(t, workspace.ResetDef())
    client := &fakeClient{}
    m := NewModelWithClient(client)
    m.state = CreateDefState
    m.currentProject = dto.Option{ID: "7", Label: "Project Seven"}
    m.agentWorkspace = workspace.Dir
    m.agentFlow = agent.NewModelWithWorkspace(workspace.Dir)
    m.agentFlow.Stage = agent.StageDefEntry

    got := applyMsg(m, editorFinishedMsg{source: CreateDefState, content: "missing title"})

    assert.Equal(t, CreateDefState, got.state)
    assert.Equal(t, dropdownDef, got.dropdown.kind)
    assert.Equal(t, "Create Change?", got.dropdown.label)
    assert.Empty(t, got.err)
    creating, createCmd := got.confirmDropdown()
    require.NotNil(t, createCmd)
    failed := applyMsg(creating.(Model), createCmd())
    assert.Equal(t, "definition title is required", failed.err)
    assert.Equal(t, []dto.Option{{ID: "/fix", Label: "/fix"}, {ID: "/cancel", Label: "/cancel"}}, failed.dropdown.options)
    assertDefBeforePrompt(t, failed.View(), "missing title", "definition title is required")
    assert.Zero(t, client.changeCreateCalls)
}

func TestAgentCreateDefValidPromptsCreateConfirmation(t *testing.T) {
    workspace := agent.Workspace{Dir: t.TempDir()}
    require.NoError(t, workspace.ResetDef())
    client := &fakeClient{}
    m := NewModelWithClient(client)
    m.state = CreateDefState
    m.currentProject = dto.Option{ID: "7", Label: "Project Seven"}
    m.agentWorkspace = workspace.Dir
    m.agentFlow = agent.NewModelWithWorkspace(workspace.Dir)
    m.agentFlow.Stage = agent.StageDefEntry

    got := applyMsg(m, editorFinishedMsg{source: CreateDefState, content: "# New Change\n\nInitial definition"})

    assert.Equal(t, CreateDefState, got.state)
    assert.Equal(t, agent.StageCreateConfirmation, got.agentFlow.Stage)
    assert.Equal(t, dropdownDef, got.dropdown.kind)
    assert.Equal(t, "Create Change?", got.dropdown.label)
    assert.Equal(t, []dto.Option{{ID: "/yes", Label: "/yes"}, {ID: "/no", Label: "/no"}}, got.dropdown.options)
    assertDefBeforePrompt(t, got.View(), "New Change", "Create Change?")
    assert.Zero(t, client.changeCreateCalls)
}

func TestAgentCreateDefNoDiscardsDefAndReturnsToList(t *testing.T) {
    workspace := agent.Workspace{Dir: t.TempDir()}
    require.NoError(t, workspace.WriteDef("# New Change\n\nInitial definition"))
    client := &fakeClient{}
    m := NewModelWithClient(client)
    m.state = CreateDefState
    m.agentWorkspace = workspace.Dir
    m.agentFlow = agent.NewModelWithWorkspace(workspace.Dir)
    m.agentFlow.Stage = agent.StageCreateConfirmation
    m.openDropdown(CreateDefState, dropdownDef, CreateDefState, CreateDefState, "Create Change?", []dto.Option{
        {ID: "/yes", Label: "/yes"},
        {ID: "/no", Label: "/no"},
    }, false)
    m.dropdown.highlighted = 1

    updated, cmd := m.confirmDropdown()
    got := updated.(Model)

    require.Nil(t, cmd)
    assert.Equal(t, ChangesListState, got.state)
    assert.False(t, got.agentFlow.Active())
    exists, err := workspace.DefExists()
    require.NoError(t, err)
    assert.False(t, exists)
    assert.Zero(t, client.changeCreateCalls)
}

func TestAgentCreateDefCancelCommandDiscardsDefAndReturnsToList(t *testing.T) {
    workspace := agent.Workspace{Dir: t.TempDir()}
    require.NoError(t, workspace.WriteDef("# New Change\n\nInitial definition"))
    client := &fakeClient{}
    m := NewModelWithClient(client)
    m.state = CreateDefState
    m.agentWorkspace = workspace.Dir
    m.agentFlow = agent.NewModelWithWorkspace(workspace.Dir)
    m.agentFlow.Stage = agent.StageCreateConfirmation

    updated, cmd := m.executeCommandFrom(CreateDefState, "/cancel")
    got := updated.(Model)

    require.Nil(t, cmd)
    assert.Equal(t, ChangesListState, got.state)
    assert.False(t, got.agentFlow.Active())
    exists, err := workspace.DefExists()
    require.NoError(t, err)
    assert.False(t, exists)
    assert.Zero(t, client.changeCreateCalls)
}

func TestAgentCreateDefYesCreatesThenRunsRewrite(t *testing.T) {
    workspace := agent.Workspace{Dir: t.TempDir()}
    require.NoError(t, workspace.WriteDef("# New Change\n\nInitial definition"))
    client := &fakeClient{createdChange: dto.Change{ID: "12", Title: "New Change", Def: "# New Change\n\nInitial definition"}}
    runner := &fakeAgentRunner{rewriteResult: agent.RewriteResult{SessionID: "session-1", Output: "Done."}}
    m := NewModelWithClient(client)
    m.state = CreateDefState
    m.currentProject = dto.Option{ID: "7", Label: "Project Seven"}
    m.agentWorkspace = workspace.Dir
    m.agentRunner = runner
    m.agentFlow = agent.NewModelWithWorkspace(workspace.Dir)
    m.agentFlow.Stage = agent.StageCreateConfirmation
    m.openDropdown(CreateDefState, dropdownDef, CreateDefState, CreateDefState, "Create Change?", []dto.Option{
        {ID: "/yes", Label: "/yes"},
        {ID: "/no", Label: "/no"},
    }, false)

    updated, cmd := m.confirmDropdown()
    model := updated.(Model)
    require.NotNil(t, cmd)
    model = applyMsg(model, cmd())

    require.Len(t, client.changeCreateInputs, 1)
    assert.Equal(t, 7, client.changeCreateInputs[0].ProjectID)
    assert.Equal(t, "New Change", client.changeCreateInputs[0].Title)
    assert.Equal(t, "# New Change\n\nInitial definition", client.changeCreateInputs[0].Def)
    assert.Equal(t, RewriteDefState, model.state)
    assert.Equal(t, agent.StageAIRunning, model.agentFlow.Stage)
    assert.Equal(t, client.createdChange, model.changeList.Detail)
}

func TestAgentRewriteInvalidOutputShowsGenericError(t *testing.T) {
    tests := []struct {
        name   string
        result agent.RewriteResult
    }{
        {
            name:   "missing session id",
            result: agent.RewriteResult{Output: "Done.", CommandOutput: "{\"type\":\"done\"}"},
        },
        {
            name:   "unexpected output",
            result: agent.RewriteResult{SessionID: "session-1", Output: "Nope", CommandOutput: "{\"session_id\":\"session-1\"}"},
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            m := NewModelWithClient(&fakeClient{})
            m.state = RewriteDefState
            m.agentFlow = agent.NewModelWithWorkspace(t.TempDir())
            m.agentFlow.Stage = agent.StageAIRunning

            got := applyMsg(m, agentRewriteFinishedMsg{result: tt.result})

            assert.Equal(t, RewriteDefState, got.state)
            assert.Equal(t, agent.GenericError, got.err)
            assert.NotEmpty(t, got.agentFlow.CommandOutput)
            view := stripANSI(got.View())
            assert.Contains(t, view, "AgentRunningScreen")
            assert.Contains(t, view, "Codex output:")
            for _, line := range strings.Split(agent.FormatCommandOutput(tt.result.CommandOutput), "\n") {
                assert.Contains(t, view, strings.TrimSpace(line))
            }
        })
    }
}

func TestUpdateDefSavesBeforeRewrite(t *testing.T) {
    workspace := agent.Workspace{Dir: t.TempDir()}
    def := "# Existing\n\nTypes: feature\n\nEdited"
    client := &fakeClient{gotChange: dto.Change{ID: "12", Def: def, ChangeTypes: []string{"feature"}}}
    m := NewModelWithClient(client)
    m.state = UpdateDefState
    m.agentWorkspace = workspace.Dir
    m.changeList.Detail = dto.Change{ID: "12", Title: "Existing", Def: "# Existing\n\nOld"}
    m.agentFlow = agent.NewModelWithWorkspace(workspace.Dir)
    m.agentFlow.Stage = agent.StageDefEntry

    updated, cmd := m.Update(editorFinishedMsg{source: UpdateDefState, content: def})
    model := updated.(Model)
    require.NotNil(t, cmd)
    model = applyMsg(model, changeDefUpdateForRewriteCommand(client, 12, def)())

    assert.Equal(t, []string{def}, client.changeDefUpdates)
    assert.Equal(t, [][]string{{"feature"}}, client.changeTypesUpdates)
    assert.Equal(t, []int{12}, client.changeGetIDs)
    assert.Equal(t, RewriteDefState, model.state)
    assert.Equal(t, agent.StageAIRunning, model.agentFlow.Stage)
}

func TestUpdateDefPromptsParseErrorBeforeSave(t *testing.T) {
    workspace := agent.Workspace{Dir: t.TempDir()}
    client := &fakeClient{}
    m := NewModelWithClient(client)
    m.state = UpdateDefState
    m.agentWorkspace = workspace.Dir
    m.changeList.Detail = dto.Change{ID: "12", Title: "Existing", Def: "# Existing\n\nOld"}
    m.agentFlow = agent.NewModelWithWorkspace(workspace.Dir)
    m.agentFlow.Stage = agent.StageDefEntry

    got := applyMsg(m, editorFinishedMsg{source: UpdateDefState, content: "missing title"})

    assert.Equal(t, UpdateDefState, got.state)
    assert.Equal(t, dropdownDef, got.dropdown.kind)
    assert.Equal(t, "error parsing title:", got.dropdown.label)
    assert.Equal(t, []dto.Option{{ID: "/edit", Label: "/edit"}, {ID: "/cancel", Label: "/cancel"}}, got.dropdown.options)
    assert.Equal(t, "error parsing title", got.err)
    assert.Empty(t, client.changeDefUpdates)
    assertDefBeforePrompt(t, got.View(), "missing title", "error parsing title:")

    content, err := os.ReadFile(workspace.DefPath())
    require.NoError(t, err)
    assert.Equal(t, "missing title", string(content))
}

func TestUpdateDefParseErrorCancelDiscardsDefAndReturnsToList(t *testing.T) {
    workspace := agent.Workspace{Dir: t.TempDir()}
    require.NoError(t, workspace.WriteDef("missing title"))
    client := &fakeClient{}
    m := NewModelWithClient(client)
    m.state = UpdateDefState
    m.agentWorkspace = workspace.Dir
    m.agentFlow = agent.NewModelWithWorkspace(workspace.Dir)
    m.agentFlow.Stage = agent.StageDefEntry
    m.agentFlow.DefEntryContent = "missing title"
    m.openDropdown(UpdateDefState, dropdownDef, UpdateDefState, UpdateDefState, "error parsing title:", []dto.Option{
        {ID: "/edit", Label: "/edit"},
        {ID: "/cancel", Label: "/cancel"},
    }, false)
    m.dropdown.highlighted = 1

    updated, cmd := m.confirmDropdown()
    got := updated.(Model)

    require.Nil(t, cmd)
    assert.Equal(t, ChangesListState, got.state)
    assert.False(t, got.agentFlow.Active())
    exists, err := workspace.DefExists()
    require.NoError(t, err)
    assert.False(t, exists)
    assert.Empty(t, client.changeDefUpdates)
}

func TestUpdateDefEscDiscardsDefAndReturnsToList(t *testing.T) {
    workspace := agent.Workspace{Dir: t.TempDir()}
    require.NoError(t, workspace.WriteDef("# Existing\n\nEdited def"))
    client := &fakeClient{}
    m := NewModelWithClient(client)
    m.state = UpdateDefState
    m.agentWorkspace = workspace.Dir
    m.agentFlow = agent.NewModelWithWorkspace(workspace.Dir)
    m.agentFlow.Stage = agent.StageDefEntry
    m.detailEditField = detailEditDef

    got, cmd := sendKey(m, tea.KeyEsc)

    require.Nil(t, cmd)
    assert.Equal(t, ChangesListState, got.state)
    assert.Empty(t, got.detailEditField)
    assert.False(t, got.agentFlow.Active())
    exists, err := workspace.DefExists()
    require.NoError(t, err)
    assert.False(t, exists)
    assert.Empty(t, client.changeDefUpdates)
}

func TestUpdateDefParseErrorEditReopensPersistentDefEditor(t *testing.T) {
    workspace := agent.Workspace{Dir: t.TempDir()}
    require.NoError(t, workspace.WriteDef("missing title"))
    m := NewModelWithClient(&fakeClient{})
    m.state = UpdateDefState
    m.agentWorkspace = workspace.Dir
    m.agentFlow = agent.NewModelWithWorkspace(workspace.Dir)
    m.agentFlow.Stage = agent.StageDefEntry
    m.agentFlow.DefEntryContent = "missing title"
    m.openDropdown(UpdateDefState, dropdownDef, UpdateDefState, UpdateDefState, "error parsing title:", []dto.Option{
        {ID: "/edit", Label: "/edit"},
        {ID: "/cancel", Label: "/cancel"},
    }, false)

    updated, cmd := m.confirmDropdown()
    got := updated.(Model)

    require.NotNil(t, cmd)
    assert.Equal(t, UpdateDefState, got.state)
    content, err := os.ReadFile(workspace.DefPath())
    require.NoError(t, err)
    assert.Equal(t, "missing title", string(content))
}

func TestAgentRewriteSuccessSavesDefAgentEditAndRemovesTempDef(t *testing.T) {
    workspace := agent.Workspace{Dir: t.TempDir()}
    def := "# Rewritten Change\n\nTypes: feature\n\nRewritten definition"
    require.NoError(t, workspace.WriteDef(def))
    client := &fakeClient{
        gotChange: dto.Change{ID: "12", Title: "Rewritten Change", Def: def, ChangeTypes: []string{"feature"}, AgentEdit: true},
    }
    m := NewModelWithClient(client)
    m.state = RewriteDefState
    m.changeList.Detail = dto.Change{ID: "12", Title: "Draft", Def: "# Draft\n\nInitial"}
    m.agentFlow = agent.NewModelWithWorkspace(workspace.Dir)
    m.agentFlow.Stage = agent.StageAIRunning

    got, saveCmd := m.Update(agentRewriteFinishedMsg{result: agent.RewriteResult{RepoRoot: "/repo", SessionID: "session-1", Output: "Done."}})
    model := got.(Model)
    require.NotNil(t, saveCmd)
    updated, clearCmd := model.Update(saveCmd())
    model = updated.(Model)
    require.NotNil(t, clearCmd)

    assert.Equal(t, []string{def}, client.changeDefUpdates)
    assert.Equal(t, [][]string{{"feature"}}, client.changeTypesUpdates)
    assert.Equal(t, []string{"change/update-def", "change/update-change-types", "change/get"}, client.requestOrder)
    assert.Equal(t, []int{12}, client.changeGetIDs)
    assert.Equal(t, ChangeDetailsState, model.state)
    assert.Equal(t, client.gotChange, model.changeList.Detail)
    assert.False(t, model.agentFlow.Active())
    assert.LessOrEqual(t, model.agentViewport.TotalLineCount(), 1)
    exists, err := workspace.DefExists()
    require.NoError(t, err)
    assert.False(t, exists)
}

func TestAgentRewriteSaveFailureStopsRunningState(t *testing.T) {
    workspace := agent.Workspace{Dir: t.TempDir()}
    require.NoError(t, workspace.WriteDef("# Rewritten Change\n\nRewritten definition"))
    client := &fakeClient{changeUpdateErr: errors.New("Internal Server Error")}
    m := NewModelWithClient(client)
    m.state = RewriteDefState
    m.changeList.Detail = dto.Change{ID: "12", Title: "Draft", Def: "# Draft\n\nInitial"}
    m.agentFlow = agent.NewModelWithWorkspace(workspace.Dir)
    m.agentFlow.Stage = agent.StageAIRunning

    got, cmd := m.Update(agentRewriteFinishedMsg{result: agent.RewriteResult{RepoRoot: "/repo", SessionID: "session-1", Output: "Done."}})
    model := got.(Model)
    require.NotNil(t, cmd)
    model = applyMsg(model, cmd())

    assert.Equal(t, ChangeDetailsState, model.state)
    assert.Equal(t, "Internal Server Error", model.err)
    assert.Equal(t, "save failed", model.status)
    assert.False(t, model.agentFlow.Active())
    assert.Equal(t, agent.StageIdle, model.agentFlow.Stage)
    assert.Zero(t, model.agentElapsed)
    view := stripANSI(model.View())
    assert.NotContains(t, view, "Agent running:")
}

func TestAgentSpecCreateCommandPersistsGeneratedSpecPath(t *testing.T) {
    workspace := agent.Workspace{Dir: t.TempDir()}
    require.NoError(t, workspace.Ensure())
    generated := "# Generated Change\n\nTypes: feature|test\n\n## Goal\nShip it.\n\n## QA Test Cases\n\n- First scenario.\n- Second scenario."
    require.NoError(t, os.WriteFile(workspace.GeneratedPath(), []byte(generated), 0o644))
    client := &fakeClient{
        createdChange: dto.Change{ID: "12", Title: "Generated Change"},
        gotChange:     dto.Change{ID: "12", Title: "Generated Change", Def: generated, Spec: generated, ChangeTypes: []string{"feature", "test"}},
    }
    m := NewModelWithClient(client)
    m.state = RewriteDefState
    m.agentWorkspace = workspace.Dir
    m.agentFlow = agent.NewModelWithWorkspace(workspace.Dir)
    m.agentFlow.Stage = agent.StageAIRunning

    got := applyMsg(m, agentSpecCreateCommand(client, 7, workspace)())

    assert.Equal(t, ChangeDetailsState, got.state)
    assert.Equal(t, client.gotChange, got.changeList.Detail)
    require.Len(t, client.changeCreateInputs, 1)
    assert.Equal(t, dto.ChangeCreateInput{ProjectID: 7, Title: "Generated Change", Def: generated}, client.changeCreateInputs[0])
    assert.Equal(t, []string{generated}, client.changeSpecUpdates)
    assert.Equal(t, [][]string{{"feature", "test"}}, client.changeTypesUpdates)
    assert.Equal(t, []dto.TestCase{
        {ChangeID: "12", Scenario: "First scenario."},
        {ChangeID: "12", Scenario: "Second scenario."},
    }, client.testCaseCreateInputs)
    assert.Equal(t, []string{
        "change/create",
        "change/update-spec",
        "change/update-change-types",
        "test-case/create",
        "test-case/create",
        "change/get",
    }, client.requestOrder)
}

func TestChangeCreateSaveExtractsTitleAndPreservesDef(t *testing.T) {
    spec := "# New Change\n\nTypes: feature|test\n\nEpic: Epic Five\n\n## Problem Statement\nKeep every section."
    client := &fakeClient{
        types:         []dto.Option{{ID: "feature", Label: "feature"}, {ID: "test", Label: "test"}},
        epics:         []dto.Option{{ID: "5", Label: "Epic Five"}},
        createdChange: dto.Change{ID: "12"},
        gotChange:     dto.Change{ID: "12", Title: "New Change", Spec: spec, ChangeTypes: []string{"feature", "test"}, EpicID: "5", EpicName: "Epic Five"},
    }
    m := newModelWithOptionCatalog(client)
    m.currentProject = dto.Option{ID: "7", Label: "Project Seven"}
    m.state = ChangeCreateState
    m.input.SetValue(spec)

    updated, cmd := m.executeCommandFrom(ChangeCreateState, "/save")
    got := updated.(Model)
    require.NotNil(t, cmd)
    got = applyMsg(got, cmd())

    require.Len(t, client.changeCreateInputs, 1)
    assert.Equal(t, 7, client.changeCreateInputs[0].ProjectID)
    assert.Equal(t, "New Change", client.changeCreateInputs[0].Title)
    assert.Equal(t, spec, client.changeCreateInputs[0].Def)
    assert.Equal(t, [][]string{{"feature", "test"}}, client.changeTypesUpdates)
    assert.Zero(t, client.epicCalls)
    assert.Equal(t, []int{12}, client.changeGetIDs)
    assert.Equal(t, ChangeDetailsState, got.state)
    assert.Equal(t, client.gotChange, got.changeList.Detail)
}

func TestChangeCreateSuccessWithReloadFailureOpensCreatedDetails(t *testing.T) {
    spec := "# New Change\n\nTypes: feature\n\n## Problem Statement\nKeep every section."
    client := &fakeClient{
        types:         []dto.Option{{ID: "feature", Label: "feature"}},
        createdChange: dto.Change{ID: "12", Title: "New Change", Spec: spec, ChangeTypes: []string{"feature"}},
        changeGetErr:  errors.New("temporary reload failure"),
    }
    m := newModelWithOptionCatalog(client)
    m.currentProject = dto.Option{ID: "7", Label: "Project Seven"}
    m.state = ChangeCreateState
    m.input.SetValue(spec)

    updated, cmd := m.executeCommandFrom(ChangeCreateState, "/save")
    got := updated.(Model)
    require.NotNil(t, cmd)
    got = applyMsg(got, cmd())

    require.Len(t, client.changeCreateInputs, 1)
    assert.Equal(t, []int{12}, client.changeGetIDs)
    assert.Equal(t, ChangeDetailsState, got.state)
    assert.Equal(t, client.createdChange, got.changeList.Detail)
    assert.Equal(t, "temporary reload failure", got.err)
    assert.Empty(t, got.input.Value())
}

func TestStandaloneChangeSaveDoesNotRequireEpicLookup(t *testing.T) {
    spec := "# Standalone Change\n\nTypes: feature\n\n## Problem Statement\nNo epic."
    client := &fakeClient{
        types:         []dto.Option{{ID: "feature", Label: "feature"}},
        epicErr:       errors.New("epics unavailable"),
        createdChange: dto.Change{ID: "12"},
        gotChange:     dto.Change{ID: "12", Title: "Standalone Change", Spec: spec, ChangeTypes: []string{"feature"}},
    }
    m := newModelWithOptionCatalog(client)
    m.currentProject = dto.Option{ID: "7", Label: "Project Seven"}
    m.state = ChangeCreateState
    m.input.SetValue(spec)

    updated, cmd := m.executeCommandFrom(ChangeCreateState, "/save")
    got := updated.(Model)
    require.NotNil(t, cmd)
    got = applyMsg(got, cmd())

    require.Len(t, client.changeCreateInputs, 1)
    assert.Zero(t, client.epicCalls)
    assert.Equal(t, ChangeDetailsState, got.state)

    updateSpec := "# Standalone Change\n\nTypes: feature\n\nEpic: \n\n## Problem Statement\nNo epic."
    original := dto.Change{
        ID:          "12",
        Title:       "Standalone Change",
        Spec:        spec,
        ChangeTypes: []string{"feature"},
    }
    client = &fakeClient{
        types:     []dto.Option{{ID: "feature", Label: "feature"}},
        epicErr:   errors.New("epics unavailable"),
        gotChange: dto.Change{ID: "12", Title: "Standalone Change", Spec: updateSpec, ChangeTypes: []string{"feature"}},
    }
    m = newModelWithOptionCatalog(client)
    m.currentProject = dto.Option{ID: "7", Label: "Project Seven"}
    m.state = ChangeUpdateState
    m.changeList.Detail = original
    m.input.SetValue(updateSpec)

    updated, cmd = m.executeCommandFrom(ChangeUpdateState, "/save")
    got = updated.(Model)
    require.NotNil(t, cmd)
    got = applyMsg(got, cmd())

    assert.Zero(t, client.epicCalls)
    assert.Equal(t, 1, client.changeSpecUpdateCalls)
    assert.Equal(t, ChangeDetailsState, got.state)
}

func TestChangeCreateValidationErrorsDoNotCallBackendCreate(t *testing.T) {
    tests := []struct {
        name string
        spec string
    }{
        {name: "missing title", spec: "Types: feature\n\n## Problem Statement\nSpec."},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            client := &fakeClient{
                types: []dto.Option{{ID: "feature", Label: "feature"}},
                epics: []dto.Option{{ID: "5", Label: "Epic Five"}},
            }
            m := NewModelWithClient(client)
            m.currentProject = dto.Option{ID: "7", Label: "Project Seven"}
            m.state = ChangeCreateState
            m.input.SetValue(tt.spec)

            updated, cmd := m.executeCommandFrom(ChangeCreateState, "/save")
            got := updated.(Model)
            require.NotNil(t, cmd)
            got = applyMsg(got, cmd())

            assert.Equal(t, ChangeCreateState, got.state)
            assert.NotEmpty(t, got.err)
            assert.Zero(t, client.changeCreateCalls)
            assert.Zero(t, client.changeGetCalls)
        })
    }
}

func TestChangeSaveStructuralValidationDoesNotFetchReferences(t *testing.T) {
    tests := []struct {
        name    string
        spec    string
        wantErr string
    }{
        {name: "missing title", spec: "Types: feature\n\n## Problem Statement\nSpec.", wantErr: "definition title is required"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            client := &fakeClient{err: errors.New("reference backend unavailable")}
            m := NewModelWithClient(client)
            m.currentProject = dto.Option{ID: "7", Label: "Project Seven"}
            m.state = ChangeCreateState
            m.input.SetValue(tt.spec)

            updated, cmd := m.executeCommandFrom(ChangeCreateState, "/save")
            got := updated.(Model)
            require.NotNil(t, cmd)
            got = applyMsg(got, cmd())

            assert.Equal(t, ChangeCreateState, got.state)
            assert.Equal(t, tt.wantErr, got.err)
            assert.Zero(t, client.typeCalls)
            assert.Zero(t, client.epicCalls)
            assert.Zero(t, client.changeCreateCalls)
        })
    }
}

func TestChangeUpdateStructuralValidationDoesNotFetchReferences(t *testing.T) {
    client := &fakeClient{err: errors.New("reference backend unavailable")}
    m := NewModelWithClient(client)
    m.currentProject = dto.Option{ID: "7", Label: "Project Seven"}
    m.state = ChangeUpdateState
    m.changeList.Detail = dto.Change{
        ID:          "12",
        Title:       "Existing Change",
        Spec:        "# Existing Change\n\nTypes: feature\n\n## Problem Statement\nSpec.",
        ChangeTypes: []string{"feature"},
    }
    m.input.SetValue("Types: feature\n\n## Problem Statement\nSpec.")

    updated, cmd := m.executeCommandFrom(ChangeUpdateState, "/save")
    got := updated.(Model)
    require.NotNil(t, cmd)
    got = applyMsg(got, cmd())

    assert.Equal(t, ChangeUpdateState, got.state)
    assert.Equal(t, "spec title is required", got.err)
    assert.Zero(t, client.typeCalls)
    assert.Zero(t, client.epicCalls)
    assert.Zero(t, client.changeTitleUpdateCalls)
    assert.Zero(t, client.changeSpecUpdateCalls)
    assert.Zero(t, client.changeTypesUpdateCalls)
    assert.Zero(t, client.changeEpicUpdateCalls)
}

func TestChangeUpdateSaveUpdatesChangedExtractedFieldsAndReloads(t *testing.T) {
    original := dto.Change{
        ID:          "12",
        Title:       "Old Change",
        Spec:        "# Old Change\n\nTypes: feature\n\n## Problem Statement\nOld spec.",
        ChangeTypes: []string{"feature"},
        EpicID:      "5",
        EpicName:    "Epic Five",
    }
    spec := "# New Change\n\nTypes: test\n\n## Problem Statement\nNew spec."
    client := &fakeClient{
        types:     []dto.Option{{ID: "feature", Label: "feature"}, {ID: "test", Label: "test"}},
        gotChange: dto.Change{ID: "12", Title: "New Change", Spec: spec, ChangeTypes: []string{"test"}, EpicID: "5", EpicName: "Epic Five"},
    }
    m := newModelWithOptionCatalog(client)
    m.currentProject = dto.Option{ID: "7", Label: "Project Seven"}
    m.state = ChangeUpdateState
    m.changeList.Detail = original
    m.input.SetValue(spec)

    updated, cmd := m.executeCommandFrom(ChangeUpdateState, "/save")
    got := updated.(Model)
    require.NotNil(t, cmd)
    got = applyMsg(got, cmd())

    assert.Equal(t, []string{"New Change"}, client.changeTitleUpdates)
    assert.Equal(t, []string{spec}, client.changeSpecUpdates)
    assert.Equal(t, [][]string{{"test"}}, client.changeTypesUpdates)
    assert.Zero(t, client.epicCalls)
    assert.Zero(t, client.changeEpicUpdateCalls)
    assert.Equal(t, []int{12}, client.changeGetIDs)
    assert.Equal(t, ChangeDetailsState, got.state)
}

func TestChangeUpdateSaveLeavesTypesUnchangedWhenMetadataIsOmitted(t *testing.T) {
    original := dto.Change{
        ID:          "12",
        Title:       "Old Change",
        ChangeTypes: []string{},
    }
    spec := "# New Change\n\n## Problem Statement\nNew spec."
    client := &fakeClient{
        types:     []dto.Option{{ID: "feature", Label: "feature"}},
        gotChange: dto.Change{ID: "12", Title: "New Change", Spec: spec, ChangeTypes: []string{}},
    }
    m := newModelWithOptionCatalog(client)
    m.currentProject = dto.Option{ID: "7", Label: "Project Seven"}
    m.state = ChangeUpdateState
    m.changeList.Detail = original
    m.input.SetValue(spec)

    updated, cmd := m.executeCommandFrom(ChangeUpdateState, "/save")
    got := updated.(Model)
    require.NotNil(t, cmd)
    got = applyMsg(got, cmd())

    assert.Empty(t, got.err)
    assert.Equal(t, []string{"New Change"}, client.changeTitleUpdates)
    assert.Equal(t, []string{spec}, client.changeSpecUpdates)
    assert.Zero(t, client.changeTypesUpdateCalls)
    assert.Equal(t, []int{12}, client.changeGetIDs)
    assert.Equal(t, ChangeDetailsState, got.state)
}

func TestChangeUpdateSaveTreatsBlankTypesAsEmpty(t *testing.T) {
    original := dto.Change{
        ID:          "12",
        Title:       "Existing Change",
        Spec:        "# Existing Change\n\nTypes: feature\n\n## Problem Statement\nOld spec.",
        ChangeTypes: []string{"feature"},
    }
    spec := "# Existing Change\n\nTypes:\n\n## Problem Statement\nOld spec."
    client := &fakeClient{
        types:     []dto.Option{{ID: "feature", Label: "feature"}},
        gotChange: dto.Change{ID: "12", Title: "Existing Change", Spec: spec, ChangeTypes: []string{}},
    }
    m := newModelWithOptionCatalog(client)
    m.currentProject = dto.Option{ID: "7", Label: "Project Seven"}
    m.state = ChangeUpdateState
    m.changeList.Detail = original
    m.input.SetValue(spec)

    updated, cmd := m.executeCommandFrom(ChangeUpdateState, "/save")
    got := updated.(Model)
    require.NotNil(t, cmd)
    got = applyMsg(got, cmd())

    assert.Zero(t, client.changeTitleUpdateCalls)
    assert.Equal(t, []string{spec}, client.changeSpecUpdates)
    require.Len(t, client.changeTypesUpdates, 1)
    assert.Empty(t, client.changeTypesUpdates[0])
    assert.Equal(t, []int{12}, client.changeGetIDs)
    assert.Equal(t, ChangeDetailsState, got.state)
}

func TestChangeUpdateOnlyCallsChangedFieldEndpoints(t *testing.T) {
    original := dto.Change{
        ID:          "12",
        Title:       "Old Change",
        Spec:        "# Old Change\n\nTypes: feature\n\n## Problem Statement\nOld spec.",
        ChangeTypes: []string{"feature"},
    }
    spec := "# Old Change\n\nTypes: feature\n\n## Problem Statement\nNew spec."
    client := &fakeClient{
        types:     []dto.Option{{ID: "feature", Label: "feature"}},
        gotChange: dto.Change{ID: "12", Title: "Old Change", Spec: spec, ChangeTypes: []string{"feature"}},
    }
    m := newModelWithOptionCatalog(client)
    m.currentProject = dto.Option{ID: "7", Label: "Project Seven"}
    m.state = ChangeUpdateState
    m.changeList.Detail = original
    m.input.SetValue(spec)

    updated, cmd := m.executeCommandFrom(ChangeUpdateState, "/save")
    got := updated.(Model)
    require.NotNil(t, cmd)
    got = applyMsg(got, cmd())

    assert.Zero(t, client.changeTitleUpdateCalls)
    assert.Equal(t, 1, client.changeSpecUpdateCalls)
    assert.Equal(t, [][]string{{"feature"}}, client.changeTypesUpdates)
    assert.Zero(t, client.changeEpicUpdateCalls)
    assert.Equal(t, ChangeDetailsState, got.state)
}

func TestChangeSpecEditUsesBackendArtifactWithoutSynthesizingMetadata(t *testing.T) {
    change := dto.Change{
        ID:          "12",
        RefUUID:     "0198a86f-9b8a-7d89-ae5b-6f25b528b04c",
        Title:       "Legacy Change",
        Spec:        "## Problem Statement\nLegacy spec.",
        ChangeTypes: []string{"feature", "test"},
        EpicName:    "Epic Five",
    }
    got := beginSpecArtifactEditor(t, change)

    assert.Equal(t, ChangeDetailsState, got.state)
    assert.Equal(t, change.Spec, got.input.Value())
    assert.Equal(t, change.Spec, readTestFile(t, got.agentFlow.Workspace.InputPath()))
}

func TestChangeSpecEditDoesNotInjectBackendEpic(t *testing.T) {
    change := dto.Change{
        ID:          "12",
        RefUUID:     "0198a86f-9b8a-7d89-ae5b-6f25b528b04c",
        Title:       "Existing Change",
        Spec:        "# Existing Change\n\nTypes: feature\n\n## Problem Statement\nExisting spec.",
        ChangeTypes: []string{"feature"},
        EpicID:      "5",
        EpicName:    "Epic Five",
    }
    got := beginSpecArtifactEditor(t, change)

    assert.Equal(t, ChangeDetailsState, got.state)
    assert.Equal(t, change.Spec, got.input.Value())
    assert.NotContains(t, got.input.Value(), "Epic Five")
}

func TestChangeSpecEditPreservesOmittedTypes(t *testing.T) {
    spec := "# Existing Change\n\n## Problem Statement\nExisting spec."
    change := dto.Change{
        ID:          "12",
        RefUUID:     "0198a86f-9b8a-7d89-ae5b-6f25b528b04c",
        Title:       "Existing Change",
        Spec:        spec,
        ChangeTypes: []string{},
    }
    got := beginSpecArtifactEditor(t, change)

    assert.Equal(t, ChangeDetailsState, got.state)
    assert.Equal(t, spec, got.input.Value())
}

func TestChangeSpecEditPreservesLongMarkdownOutsidePromptLimit(t *testing.T) {
    longSection := strings.Repeat("Full markdown line with details.\n", 12)
    spec := "# Long Change\n\nTypes: feature\n\n## Problem Statement\n" + longSection
    require.Greater(t, len(spec), defaultPromptCharLimit)

    change := dto.Change{
        ID:          "12",
        RefUUID:     "0198a86f-9b8a-7d89-ae5b-6f25b528b04c",
        Title:       "Long Change",
        Spec:        spec,
        ChangeTypes: []string{"feature"},
    }
    got := beginSpecArtifactEditor(t, change)

    assert.Equal(t, ChangeDetailsState, got.state)
    assert.Equal(t, spec, got.input.Value())
    assert.Zero(t, got.input.CharLimit)
}

func beginSpecArtifactEditor(t *testing.T, change dto.Change) Model {
    t.Helper()
    client := &fakeClient{gotChange: change}
    m := NewModelWithClient(client)
    m.agentWorkspace = t.TempDir()
    m.state = ChangeDetailsState
    m.changeList.Detail = change
    loading, loadCmd := sendCommand(m, "/edit-spec")
    require.NotNil(t, loadCmd)
    updated, editorCmd := loading.Update(loadCmd())
    require.NotNil(t, editorCmd)
    return updated.(Model)
}

func TestChangeUpdateDoesNotChangeBackendEpic(t *testing.T) {
    original := dto.Change{
        ID:          "12",
        Title:       "Existing Change",
        Spec:        "# Existing Change\n\nTypes: feature\n\n## Problem Statement\nExisting spec.",
        ChangeTypes: []string{"feature"},
        EpicID:      "5",
    }
    client := &fakeClient{
        types:     []dto.Option{{ID: "feature", Label: "feature"}},
        gotChange: original,
    }
    m := newModelWithOptionCatalog(client)
    m.currentProject = dto.Option{ID: "7", Label: "Project Seven"}
    m.state = ChangeUpdateState
    m.changeList.Detail = original

    updated, cmd := m.saveChangeUpdateValue(changes.SpecMarkdown(original))
    got := updated.(Model)
    require.NotNil(t, cmd)
    got = applyMsg(got, cmd())

    assert.Zero(t, client.changeTitleUpdateCalls)
    assert.Zero(t, client.changeSpecUpdateCalls)
    assert.Equal(t, [][]string{{"feature"}}, client.changeTypesUpdates)
    assert.Zero(t, client.epicCalls)
    assert.Zero(t, client.changeEpicUpdateCalls)
    assert.Equal(t, []int{12}, client.changeGetIDs)
    assert.Equal(t, ChangeDetailsState, got.state)
}

func TestChangeFindFilterNarrowsVisibleRowsAndClearRestoresList(t *testing.T) {
    m := NewModelWithClient(&fakeClient{})
    m.state = ChangesListState
    m.changeList = m.changeList.WithRows([]dto.Change{
        {ID: "1", Ref: "1", Title: "Alpha", ChangePhase: "backlog", ChangeTypes: []string{"feature"}, Spec: "first"},
        {ID: "2", Ref: "2", Title: "Beta", ChangePhase: "done", ChangeTypes: []string{"test"}, Spec: "second"},
    })

    got, _ := sendCommand(m, "/find-filter")
    assert.Equal(t, FindInputState, got.state)
    got.input.SetValue("beta")
    got, _ = sendKey(got, tea.KeyEnter)

    assert.Equal(t, ChangesListState, got.state)
    assert.Equal(t, "beta", got.changesFilters.find)
    view := stripANSI(got.View())
    assert.Contains(t, view, "Beta")
    assert.NotContains(t, view, "Alpha")

    got, _ = sendCommand(got, "/clear-filters")
    assert.Empty(t, got.changesFilters.find)
    view = stripANSI(got.View())
    assert.Contains(t, view, "Alpha")
    assert.Contains(t, view, "Beta")
}

func TestChangeFindFilterMatchesDisplayedPaddedRef(t *testing.T) {
    m := NewModelWithClient(&fakeClient{})
    m.state = ChangesListState
    m.changeList = m.changeList.WithRows([]dto.Change{
        {ID: "1", Ref: "3", Title: "Alpha", ChangePhase: "backlog", ChangeTypes: []string{"feature"}},
        {ID: "2", Ref: "4", Title: "Beta", ChangePhase: "done", ChangeTypes: []string{"test"}},
    })

    got, _ := sendCommand(m, "/find-filter")
    got.input.SetValue("000003")
    got, _ = sendKey(got, tea.KeyEnter)

    assert.Equal(t, ChangesListState, got.state)
    view := stripANSI(got.View())
    assert.Contains(t, view, "000003")
    assert.Contains(t, view, "Alpha")
    assert.NotContains(t, view, "Beta")
}

func TestChangeFindFilterClampsSelectedRow(t *testing.T) {
    client := &fakeClient{gotChange: dto.Change{ID: "2", Title: "Beta"}}
    m := NewModelWithClient(client)
    m.state = ChangesListState
    m.changeList = m.changeList.WithRows([]dto.Change{
        {ID: "1", Ref: "1", Title: "Alpha", ChangePhase: "backlog", ChangeTypes: []string{"feature"}},
        {ID: "2", Ref: "2", Title: "Beta", ChangePhase: "done", ChangeTypes: []string{"test"}},
        {ID: "3", Ref: "3", Title: "Gamma", ChangePhase: "review", ChangeTypes: []string{"feature"}},
    })
    m.changeList.Selected = 2

    got, _ := sendCommand(m, "/find-filter")
    got.input.SetValue("beta")
    got, _ = sendKey(got, tea.KeyEnter)

    assert.Equal(t, ChangesListState, got.state)
    assert.Equal(t, 0, got.changeList.Selected)

    updated, cmd := got.submitPromptValue("")
    got = updated.(Model)
    require.NotNil(t, cmd)
    got = applyMsg(got, cmd())

    assert.Equal(t, []int{2}, client.changeGetIDs)
    assert.Equal(t, ChangeDetailsState, got.state)
}

func TestProjectsTableNarrowWidthDoesNotOverflow(t *testing.T) {
    m := NewModelWithClient(&fakeClient{})
    m.state = ProjectsListState
    m.width = 24
    m.projectList.Rows = []dto.Project{{
        ID:          "777777",
        Name:        "Very Long Project Name That Must Be Truncated",
        ChangeCount: 123,
        Created:     "2026-06-29T08:15:00Z",
        Modified:    "2026-06-29T10:45:00Z",
    }}

    for _, line := range strings.Split(stripANSI(projects.TableView(m.projectList, 24)), "\n") {
        assert.LessOrEqual(t, len(line), 24)
    }
}

func TestMainNewChangeShortcutIsFirstCommand(t *testing.T) {
    commands := commandsByState[MainState]
    require.NotEmpty(t, commands)
    assert.Equal(t, "/changes", commands[0])
    assert.NotContains(t, commands, "/new-change")
    assert.Contains(t, commandsByState[ChangesListState], "/new-change")
}

func TestQuitOutsideMainIsRecoverableError(t *testing.T) {
    m := NewModel()
    m.state = ChangesListState

    got, cmd := sendCommand(m, "/quit")

    assert.Equal(t, ChangesListState, got.state)
    assert.NotEmpty(t, got.err)
    assert.Nil(t, cmd)
}

func TestUnknownCommandLeavesStateUnchanged(t *testing.T) {
    m := NewModel()
    m.state = ChangeDetailsState

    got, _ := sendCommand(m, "/bogus")

    assert.Equal(t, ChangeDetailsState, got.state)
    assert.NotEmpty(t, got.err)
}

func TestChangeDetailsTableSelectionMovesAcrossAllRows(t *testing.T) {
    m := NewModel()
    m.state = ChangeDetailsState
    m.changeList = m.changeList.WithDetail(dto.Change{
        ID:          "11",
        RefUUID:     "11111111-2222-4333-8444-555555555555",
        Ref:         "3",
        Slug:        "change-three",
        Title:       "Backend Change",
        ChangePhase: "backlog",
        ChangeTypes: []string{"feature", "test"},
        EpicName:    "Epic Five",
        Spec:        "Spec",
        PR:          "Pull request body",
        PRUrl:       "https://example.test/pr",
    })

    assert.Equal(t, -2, m.changeList.DetailSelected)

    got, _ := sendKey(m, tea.KeyUp)
    assert.Equal(t, -2, got.changeList.DetailSelected)

    got, _ = sendKey(got, tea.KeyDown)
    assert.Equal(t, -1, got.changeList.DetailSelected)

    got, _ = sendKey(got, tea.KeyEnter)
    assert.Equal(t, ChangeDetailsState, got.state)
    assert.Equal(t, "selected Ref UUID", got.status)

    got, _ = sendKey(got, tea.KeyDown)
    assert.Equal(t, 0, got.changeList.DetailSelected)

    got, _ = sendKey(got, tea.KeyDown)
    assert.Equal(t, 1, got.changeList.DetailSelected)

    got, _ = sendKey(got, tea.KeyDown)
    assert.Equal(t, 2, got.changeList.DetailSelected)
}

func TestChangeDetailsCopySelectedField(t *testing.T) {
    var copied []string
    previousWriteClipboard := writeClipboard
    writeClipboard = func(value string) error {
        copied = append(copied, value)
        return nil
    }
    defer func() {
        writeClipboard = previousWriteClipboard
    }()

    m := NewModel()
    m.state = ChangeDetailsState
    m.changeList = m.changeList.WithDetail(dto.Change{
        ID:      "11",
        RefUUID: "11111111-2222-4333-8444-555555555555",
        Ref:     "3",
        Title:   "Backend Change",
    })

    got, cmd := sendKeyMsg(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("ctrl+shift+c")})
    require.NotNil(t, cmd)
    assert.Equal(t, "copying ID", got.status)

    got = applyMsg(got, cmd())
    assert.Equal(t, []string{"11"}, copied)
    assert.Equal(t, "copied ID", got.status)

    got, _ = sendKey(got, tea.KeyDown)
    got, cmd = sendKeyMsg(got, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("ctrl+insert")})
    require.NotNil(t, cmd)
    got = applyMsg(got, cmd())

    assert.Equal(t, []string{"11", "11111111-2222-4333-8444-555555555555"}, copied)
    assert.Equal(t, "copied Ref UUID", got.status)
}

func TestChangeDetailsCopyReportsClipboardFailure(t *testing.T) {
    previousWriteClipboard := writeClipboard
    writeClipboard = func(string) error {
        return errors.New("clipboard unavailable")
    }
    defer func() {
        writeClipboard = previousWriteClipboard
    }()

    m := NewModel()
    m.state = ChangeDetailsState
    m.changeList = m.changeList.WithDetail(dto.Change{ID: "11", Title: "Backend Change"})

    got, cmd := sendKeyMsg(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("ctrl+insert")})
    require.NotNil(t, cmd)
    got = applyMsg(got, cmd())

    assert.Equal(t, "copy failed", got.status)
    assert.Equal(t, "clipboard unavailable", got.err)
}

func TestChangeDetailsPhaseSelectionSavesAndReloads(t *testing.T) {
    client := &fakeClient{
        phases: []dto.Option{{ID: "stage", Label: "stage"}, {ID: "backlog", Label: "backlog"}},
        gotChange: dto.Change{
            ID:          "12",
            Ref:         "3",
            Title:       "Backend Change",
            ChangePhase: "stage",
        },
    }
    m := newModelWithOptionCatalog(client)
    m.state = ChangeDetailsState
    m.changeList = m.changeList.WithDetail(dto.Change{
        ID:          "12",
        Ref:         "3",
        Title:       "Backend Change",
        ChangePhase: "backlog",
    })
    m.changeList.DetailSelected = 2

    got, cmd := sendKey(m, tea.KeyEnter)
    require.NotNil(t, cmd)
    assert.Equal(t, SelectPhaseDropDown, got.state)
    got = applyMsg(got, cmd())
    assert.Equal(t, 1, got.dropdown.highlighted)
    assert.Contains(t, stripANSI(got.dropdownView(80)), "    -stage")

    got, _ = sendKey(got, tea.KeyUp)
    got, cmd = sendKey(got, tea.KeyEnter)
    require.NotNil(t, cmd)
    assert.Equal(t, ChangeDetailsState, got.state)
    got = applyMsg(got, cmd())

    assert.Equal(t, []string{"stage"}, client.changePhaseUpdates)
    assert.Equal(t, []int{12}, client.changeGetIDs)
    assert.Equal(t, "stage", got.changeList.Detail.ChangePhase)
    assert.Equal(t, ChangeDetailsState, got.state)
    assert.Equal(t, 2, got.changeList.DetailSelected)
}

func TestChangeDetailsFieldSelectionEscapeCancelsWithoutSaving(t *testing.T) {
    client := &fakeClient{
        phases: []dto.Option{{ID: "stage", Label: "stage"}},
    }
    m := newModelWithOptionCatalog(client)
    m.state = ChangeDetailsState
    m.changeList = m.changeList.WithDetail(dto.Change{
        ID:          "12",
        Ref:         "3",
        Title:       "Backend Change",
        ChangePhase: "backlog",
    })
    m.changeList.DetailSelected = 2

    got, cmd := sendKey(m, tea.KeyEnter)
    require.NotNil(t, cmd)
    got = applyMsg(got, cmd())

    got, cmd = sendKey(got, tea.KeyEsc)
    require.Nil(t, cmd)

    assert.Equal(t, ChangeDetailsState, got.state)
    assert.Empty(t, got.dropdown.kind)
    assert.Zero(t, client.changePhaseUpdateCalls)
    assert.Zero(t, client.changeGetCalls)
    assert.Equal(t, "backlog", got.changeList.Detail.ChangePhase)
}

func TestChangeDetailsEpicNoneSelectionClearsEpic(t *testing.T) {
    client := &fakeClient{
        epics: []dto.Option{{ID: "4", Label: "Epic Four"}, {ID: "5", Label: "Epic Five"}},
        gotChange: dto.Change{
            ID:    "12",
            Ref:   "3",
            Title: "Backend Change",
        },
    }
    m := NewModelWithClient(client)
    m.currentProject = dto.Option{ID: "7", Label: "Project Seven"}
    m.state = ChangeDetailsState
    m.changeList = m.changeList.WithDetail(dto.Change{
        ID:       "12",
        Ref:      "3",
        Title:    "Backend Change",
        EpicID:   "5",
        EpicName: "Epic Five",
    })
    m.changeList.DetailSelected = 3

    got, cmd := sendKey(m, tea.KeyEnter)
    require.NotNil(t, cmd)
    assert.Equal(t, SelectEpicDropDown, got.state)
    got = applyMsg(got, cmd())
    assert.Equal(t, 1, got.dropdown.highlighted)
    assert.Contains(t, stripANSI(got.dropdownView(80)), "    @none")

    got, _ = sendKey(got, tea.KeyDown)
    got, cmd = sendKey(got, tea.KeyEnter)
    require.NotNil(t, cmd)
    got = applyMsg(got, cmd())

    require.Len(t, client.changeEpicUpdates, 1)
    assert.Nil(t, client.changeEpicUpdates[0])
    assert.Equal(t, []int{12}, client.changeGetIDs)
    assert.Empty(t, got.changeList.Detail.EpicID)
    assert.Equal(t, ChangeDetailsState, got.state)
    assert.Equal(t, 3, got.changeList.DetailSelected)
}

func TestChangeDetailsTitleSelectionOpensPromptAndSaves(t *testing.T) {
    client := &fakeClient{
        gotChange: dto.Change{
            ID:    "12",
            Ref:   "3",
            Title: "New Title",
        },
    }
    m := NewModelWithClient(client)
    m.state = ChangeDetailsState
    m.changeList = m.changeList.WithDetail(dto.Change{
        ID:    "12",
        Ref:   "3",
        Title: "Old Title",
    })
    m.changeList.DetailSelected = 5

    got, cmd := sendKey(m, tea.KeyEnter)
    require.Nil(t, cmd)
    assert.Equal(t, ChangeUpdateState, got.state)
    assert.Equal(t, detailEditTitle, got.detailEditField)
    assert.Equal(t, "Old Title", got.input.Value())
    assert.Equal(t, "Write a Title", got.input.Placeholder)
    assert.Contains(t, got.View(), "ChangeUpdateScreen")

    got = got.setPromptValue("New Title")
    got, cmd = sendKey(got, tea.KeyEnter)
    require.NotNil(t, cmd)
    got = applyMsg(got, cmd())

    assert.Equal(t, []string{"New Title"}, client.changeTitleUpdates)
    assert.Equal(t, []int{12}, client.changeGetIDs)
    assert.Equal(t, "New Title", got.changeList.Detail.Title)
    assert.Equal(t, ChangeDetailsState, got.state)
    assert.Equal(t, 5, got.changeList.DetailSelected)
    assert.Empty(t, got.detailEditField)
    assert.Empty(t, got.input.Value())
}

func TestChangeDetailsTitleCancelDoesNotSave(t *testing.T) {
    client := &fakeClient{gotChange: dto.Change{ID: "12", Ref: "3", Title: "Old Title"}}
    m := NewModelWithClient(client)
    m.state = ChangeDetailsState
    m.changeList = m.changeList.WithDetail(dto.Change{
        ID:    "12",
        Ref:   "3",
        Title: "Old Title",
    })
    m.changeList.DetailSelected = 5

    got, cmd := sendKey(m, tea.KeyEnter)
    require.Nil(t, cmd)
    require.Equal(t, ChangeUpdateState, got.state)
    assert.Equal(t, detailEditTitle, got.detailEditField)

    got = got.setPromptValue("/cancel")
    got, cmd = sendKey(got, tea.KeyEnter)

    require.NotNil(t, cmd)
    got = applyMsg(got, cmd())
    assert.Equal(t, ChangeDetailsState, got.state)
    assert.Empty(t, got.detailEditField)
    assert.Empty(t, got.input.Value())
    assert.Zero(t, client.changeTitleUpdateCalls)
    assert.Equal(t, []int{12}, client.changeGetIDs)
}

func TestExistingChangeArtifactWritesReuseUUIDArtifactWorkspaceAndSession(t *testing.T) {
    const refUUID = "0198a86f-9b8a-7d89-ae5b-6f25b528b04c"
    tests := []struct {
        name      string
        field     detailEditField
        operation agent.WriteOperation
        initial   string
        edited    string
        agentText string
        setValue  func(*dto.Change, string)
        updates   func(*fakeClient) []string
    }{
        {
            name: "def", field: detailEditDef, operation: agent.DefWriteOperation,
            initial: "# Existing\n\nOld def", edited: "# Existing\n\nTypes: feature\n\nEdited def",
            agentText: "# Existing\n\nTypes: feature\n\nRewritten definition",
            setValue:  func(change *dto.Change, value string) { change.Def = value },
            updates:   func(client *fakeClient) []string { return client.changeDefUpdates },
        },
        {
            name: "spec", field: detailEditSpec, operation: agent.SpecWriteOperation,
            initial: "Original spec", edited: "Types: feature\n\nEdited spec", agentText: "Types: feature\n\nWritten spec",
            setValue: func(change *dto.Change, value string) { change.Spec = value },
            updates:  func(client *fakeClient) []string { return client.changeSpecUpdates },
        },
        {
            name: "pr", field: detailEditPullRequest, operation: agent.PRWriteOperation,
            initial: "Original PR", edited: "Types: feature\n\nEdited PR", agentText: "Types: feature\n\nWritten PR",
            setValue: func(change *dto.Change, value string) { change.PR = value },
            updates:  func(client *fakeClient) []string { return client.changePRUpdates },
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            tempRoot := filepath.Join(t.TempDir(), ".mch", "tmp")
            workspace := agent.NewArtifactModel(tempRoot, refUUID, tt.operation).Workspace
            require.NoError(t, workspace.Ensure())
            require.NoError(t, os.WriteFile(workspace.SessionPath(), []byte("existing-session\n"), 0o644))
            require.NoError(t, os.WriteFile(filepath.Join(workspace.Dir, "events.jsonl"), []byte("keep"), 0o644))

            loadedChange := dto.Change{ID: "12", RefUUID: refUUID, Ref: "3", Title: "Backend Change"}
            tt.setValue(&loadedChange, tt.initial)
            client := &fakeClient{gotChange: loadedChange}
            m := NewModelWithClient(client)
            m.agentWorkspace = tempRoot
            m.state = ChangeDetailsState
            m.changeList = m.changeList.WithDetail(dto.Change{ID: "12", RefUUID: refUUID})

            loading, loadCmd := m.beginDetailTextEditor(tt.field)
            got := loading.(Model)
            require.NotNil(t, loadCmd)
            updated, editorCmd := got.Update(loadCmd())
            got = updated.(Model)
            require.NotNil(t, editorCmd)
            assert.Equal(t, ChangeDetailsState, got.state)
            assert.Equal(t, agent.StageArtifactEntry, got.agentFlow.Stage)
            assert.Equal(t, tt.operation, got.agentFlow.Workspace.Operation)
            assert.Equal(t, "artifact", got.agentFlow.Workspace.Stage)
            assert.Equal(t, filepath.Join(tempRoot, refUUID, "artifact"), got.agentFlow.Workspace.Dir)
            assert.Equal(t, "existing-session", got.agentFlow.SessionID)
            assert.Equal(t, tt.initial, readTestFile(t, workspace.InputPath()))
            assert.Equal(t, tt.initial, readTestFile(t, workspace.OutputPath()))
            assert.Equal(t, "keep", readTestFile(t, filepath.Join(workspace.Dir, "events.jsonl")))

            require.NoError(t, os.WriteFile(workspace.OutputPath(), []byte(tt.edited), 0o644))
            tt.setValue(&client.gotChange, tt.edited)
            updated, saveCmd := got.Update(editorFinishedMsg{source: ChangeDetailsState, content: tt.edited})
            got = updated.(Model)
            require.NotNil(t, saveCmd)
            got = applyCommand(got, saveCmd)
            assert.Equal(t, RewriteDefState, got.state)
            assert.Equal(t, agent.StageAIRunning, got.agentFlow.Stage)
            assert.Equal(t, []string{tt.edited}, tt.updates(client))
            assert.Equal(t, []bool{false}, client.changeArtifactAgentEdits)
            assert.Equal(t, [][]string{{"feature"}}, client.changeTypesUpdates)
            assert.Equal(t, tt.edited, readTestFile(t, workspace.InputPath()))

            require.NoError(t, os.WriteFile(workspace.OutputPath(), []byte(tt.agentText), 0o644))
            tt.setValue(&client.gotChange, tt.agentText)
            updated, agentSaveCmd := got.Update(agentRewriteFinishedMsg{result: agent.RewriteResult{
                RepoRoot: "/repo", SessionID: "new-session", Output: "Done.",
            }})
            got = updated.(Model)
            require.NotNil(t, agentSaveCmd)
            got = applyCommand(got, agentSaveCmd)

            assert.Equal(t, []string{tt.edited, tt.agentText}, tt.updates(client))
            assert.Equal(t, []bool{false, true}, client.changeArtifactAgentEdits)
            assert.Equal(t, ChangeDetailsState, got.state)
            assert.Equal(t, "new-session", strings.TrimSpace(readTestFile(t, workspace.SessionPath())))
            assert.Equal(t, "keep", readTestFile(t, filepath.Join(workspace.Dir, "events.jsonl")))
            assert.Equal(t, []int{12, 12, 12}, client.changeGetIDs)
        })
    }
}

func TestExistingChangeArtifactUnchangedReturnsToDetailsWithoutUpdate(t *testing.T) {
    const refUUID = "0198a86f-9b8a-7d89-ae5b-6f25b528b04c"
    change := dto.Change{ID: "12", RefUUID: refUUID, Title: "Existing", Spec: "Existing spec"}
    client := &fakeClient{gotChange: change}
    m := NewModelWithClient(client)
    m.agentWorkspace = t.TempDir()
    m.state = ChangeDetailsState
    m.changeList.Detail = change

    loading, loadCmd := m.beginDetailTextEditor(detailEditSpec)
    updated, _ := loading.Update(loadCmd())
    got := updated.(Model)
    updated, clearCmd := got.Update(editorFinishedMsg{source: ChangeDetailsState, content: change.Spec})
    got = updated.(Model)

    require.NotNil(t, clearCmd)
    assert.Equal(t, ChangeDetailsState, got.state)
    assert.Empty(t, got.detailEditField)
    assert.Equal(t, "cancel", got.status)
    assert.Zero(t, client.changeSpecUpdateCalls)
    assert.Equal(t, []int{12}, client.changeGetIDs)
    require.FileExists(t, filepath.Join(m.agentWorkspace, refUUID, "artifact", "input.md"))
}

func TestExistingChangeArtifactFollowUpFailuresKeepPersistedArtifactCurrent(t *testing.T) {
    t.Run("user save type update failure", func(t *testing.T) {
        const edited = "Types: feature\n\nUser-edited spec"
        client := &fakeClient{changeTypesUpdateErr: errors.New("type update failed")}
        m := NewModelWithClient(client)
        m.state = ChangeDetailsState
        m.changeList = m.changeList.WithDetail(dto.Change{ID: "12", Spec: "Original spec"})
        m.agentFlow.Stage = agent.StageArtifactEntry
        m.detailEditField = detailEditSpec
        m = m.setPromptValue("Original spec")

        got := applyMsg(m, changeArtifactUpdateForWriteCommand(client, 12, detailEditSpec, edited)())

        assert.Equal(t, edited, got.changeList.Detail.Spec)
        assert.Equal(t, "type update failed", got.err)
        assert.Equal(t, "save failed", got.status)
        assert.Equal(t, ChangeDetailsState, got.state)
        assert.Equal(t, agent.StageIdle, got.agentFlow.Stage)
        assert.Empty(t, got.detailEditField)
        assert.Empty(t, got.input.Value())
        assert.Zero(t, client.changeGetCalls)
    })

    t.Run("agent save reload failure", func(t *testing.T) {
        const edited = "Types: feature\n\nAgent-written spec"
        workspace := agent.Workspace{Dir: t.TempDir()}
        require.NoError(t, workspace.WriteDef(edited))
        client := &fakeClient{changeGetErr: errors.New("reload failed")}
        m := NewModelWithClient(client)
        m.state = RewriteDefState
        m.changeList = m.changeList.WithDetail(dto.Change{ID: "12", Spec: "User-edited spec"})
        m.agentFlow = agent.NewModelWithWorkspace(workspace.Dir)
        m.agentFlow.Stage = agent.StageAIRunning
        m.detailEditField = detailEditSpec

        got := applyMsg(m, changeArtifactAgentEditSaveCommand(client, 12, detailEditSpec, workspace)())

        assert.Equal(t, edited, got.changeList.Detail.Spec)
        assert.Equal(t, "reload failed", got.err)
        assert.Equal(t, "save failed", got.status)
        assert.Equal(t, ChangeDetailsState, got.state)
        assert.Equal(t, agent.StageIdle, got.agentFlow.Stage)
        assert.Empty(t, got.detailEditField)
        assert.Empty(t, got.input.Value())
        assert.Equal(t, []int{12}, client.changeGetIDs)
    })
}

func TestChangeDetailsRejectsInvalidArtifactSavesBeforeBackend(t *testing.T) {
    tests := []struct {
        name      string
        field     detailEditField
        value     string
        wantError string
    }{
        {
            name:      "empty spec",
            field:     detailEditSpec,
            value:     "   ",
            wantError: "spec is required",
        },
        {
            name:      "empty pr",
            field:     detailEditPullRequest,
            value:     "   ",
            wantError: "PR is required",
        },
        {
            name:      "empty pr url",
            field:     detailEditPRUrl,
            value:     "   ",
            wantError: "PR URL is required",
        },
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            client := &fakeClient{}
            cmd := changeDetailTextUpdateCommand(client, ChangeDetailsState, dto.Change{ID: "12"}, tt.field, tt.value)
            require.NotNil(t, cmd)
            msg := cmd()
            saved, ok := msg.(changeSavedMsg)
            require.True(t, ok)
            require.Error(t, saved.err)
            assert.Equal(t, tt.wantError, saved.err.Error())
            assert.Zero(t, client.changeSpecUpdateCalls)
            assert.Zero(t, client.changePRUpdateCalls)
            assert.Zero(t, client.changePRUrlUpdateCalls)
            assert.Zero(t, client.changeGetCalls)
        })
    }
}

func TestDefSpecAndPRSavesApplyPresentTypesMetadata(t *testing.T) {
    fields := []detailEditField{detailEditDef, detailEditSpec, detailEditPullRequest}
    metadata := []struct {
        name   string
        line   string
        values []string
    }{
        {name: "empty", line: "Types:", values: []string{}},
        {name: "pipe-delimited and processed", line: "Types: fix|feature|unsupported!", values: []string{"fix", "feature", "unsupported"}},
    }

    for _, field := range fields {
        for _, tt := range metadata {
            t.Run(string(field)+"/"+tt.name, func(t *testing.T) {
                client := &fakeClient{gotChange: dto.Change{ID: "12"}}
                value := "# Artifact\n\n" + tt.line + "\n\nBody"
                msg := changeDetailTextUpdateCommand(client, ChangeDetailsState, dto.Change{ID: "12"}, field, value)()
                saved, ok := msg.(changeSavedMsg)
                require.True(t, ok)
                require.NoError(t, saved.err)
                assert.Equal(t, [][]string{tt.values}, client.changeTypesUpdates)
                assert.Equal(t, []int{12}, client.changeGetIDs)
            })
        }
    }
}

func TestChangeDetailsTypesSelectionAddsUnselectedType(t *testing.T) {
    client := &fakeClient{
        types: []dto.Option{
            {ID: "docs", Label: "docs"},
            {ID: "feature", Label: "feature"},
            {ID: "test", Label: "test"},
        },
        gotChange: dto.Change{
            ID:          "12",
            Ref:         "3",
            Title:       "Backend Change",
            ChangeTypes: []string{"docs", "feature"},
        },
    }
    m := newModelWithOptionCatalog(client)
    m.state = ChangeDetailsState
    m.changeList = m.changeList.WithDetail(dto.Change{
        ID:          "12",
        Ref:         "3",
        Title:       "Backend Change",
        ChangeTypes: []string{"feature"},
    })
    m.changeList.DetailSelected = 4

    got, cmd := sendKey(m, tea.KeyEnter)
    require.NotNil(t, cmd)
    assert.Equal(t, SelectTypesDropDown, got.state)
    got = applyMsg(got, cmd())
    assert.Equal(t, 1, got.dropdown.highlighted)
    view := stripANSI(got.dropdownView(80))
    assert.Less(t, strings.Index(view, "    +docs"), strings.Index(view, "    -feature"))
    assert.Less(t, strings.Index(view, "    -feature"), strings.Index(view, "    +test"))
    assert.Contains(t, view, "press <space> to change")

    got, _ = sendKey(got, tea.KeyUp)
    got, cmd = sendKey(got, tea.KeySpace)
    require.Nil(t, cmd)
    got, cmd = sendKey(got, tea.KeyEnter)
    require.NotNil(t, cmd)
    got = applyMsg(got, cmd())

    assert.Equal(t, [][]string{{"docs", "feature"}}, client.changeTypesUpdates)
    assert.Equal(t, []int{12}, client.changeGetIDs)
    assert.Equal(t, []string{"docs", "feature"}, got.changeList.Detail.ChangeTypes)
    assert.Equal(t, ChangeDetailsState, got.state)
    assert.Equal(t, 4, got.changeList.DetailSelected)
}

func TestChangeDetailsTypesSelectionRemovesSelectedType(t *testing.T) {
    client := &fakeClient{
        types: []dto.Option{
            {ID: "docs", Label: "docs"},
            {ID: "feature", Label: "feature"},
            {ID: "test", Label: "test"},
        },
        gotChange: dto.Change{
            ID:          "12",
            Ref:         "3",
            Title:       "Backend Change",
            ChangeTypes: []string{"test"},
        },
    }
    m := newModelWithOptionCatalog(client)
    m.state = ChangeDetailsState
    m.changeList = m.changeList.WithDetail(dto.Change{
        ID:          "12",
        Ref:         "3",
        Title:       "Backend Change",
        ChangeTypes: []string{"feature", "test"},
    })
    m.changeList.DetailSelected = 4

    got, cmd := sendKey(m, tea.KeyEnter)
    require.NotNil(t, cmd)
    got = applyMsg(got, cmd())
    assert.Equal(t, 1, got.dropdown.highlighted)
    assert.Contains(t, stripANSI(got.dropdownView(80)), "    -feature")

    got, cmd = sendKey(got, tea.KeySpace)
    require.Nil(t, cmd)
    got, cmd = sendKey(got, tea.KeyEnter)
    require.NotNil(t, cmd)
    got = applyMsg(got, cmd())

    assert.Equal(t, [][]string{{"test"}}, client.changeTypesUpdates)
    assert.Equal(t, []int{12}, client.changeGetIDs)
    assert.Equal(t, []string{"test"}, got.changeList.Detail.ChangeTypes)
    assert.Equal(t, ChangeDetailsState, got.state)
    assert.Equal(t, 4, got.changeList.DetailSelected)
}

func TestChangeDetailsTypesSelectionEnterWithoutToggleReturnsWithoutSaving(t *testing.T) {
    client := &fakeClient{
        types: []dto.Option{
            {ID: "feature", Label: "feature"},
            {ID: "test", Label: "test"},
        },
    }
    m := newModelWithOptionCatalog(client)
    m.state = ChangeDetailsState
    m.changeList = m.changeList.WithDetail(dto.Change{
        ID:          "12",
        Ref:         "3",
        Title:       "Backend Change",
        ChangeTypes: []string{"feature"},
    })
    m.changeList.DetailSelected = 4

    got, cmd := sendKey(m, tea.KeyEnter)
    require.NotNil(t, cmd)
    got = applyMsg(got, cmd())

    got, cmd = sendKey(got, tea.KeyEnter)
    require.Nil(t, cmd)

    assert.Equal(t, ChangeDetailsState, got.state)
    assert.Empty(t, got.dropdown.kind)
    assert.Zero(t, client.changeTypesUpdateCalls)
    assert.Zero(t, client.changeGetCalls)
    assert.Equal(t, []string{"feature"}, got.changeList.Detail.ChangeTypes)
}

func TestChangeDetailsOpenSpaceTogglesAndReloads(t *testing.T) {
    client := &fakeClient{
        gotChange: dto.Change{
            ID:    "12",
            Ref:   "3",
            Title: "Backend Change",
            Open:  false,
        },
    }
    m := NewModelWithClient(client)
    m.state = ChangeDetailsState
    m.changeList = m.changeList.WithDetail(dto.Change{
        ID:    "12",
        Ref:   "3",
        Title: "Backend Change",
        Open:  true,
    })
    m.changeList.DetailSelected = 12

    got, cmd := sendRune(m, ' ')
    require.NotNil(t, cmd)
    assert.Equal(t, "saving open", got.status)
    got = applyMsg(got, cmd())

    assert.Equal(t, []bool{false}, client.changeOpenUpdates)
    assert.Equal(t, []int{12}, client.changeGetIDs)
    assert.False(t, got.changeList.Detail.Open)
    assert.Equal(t, ChangeDetailsState, got.state)
    assert.Equal(t, 12, got.changeList.DetailSelected)
}

func TestChangeDetailsTestCaseSpaceTogglesAndReloads(t *testing.T) {
    client := &fakeClient{
        gotChange: dto.Change{
            ID:    "12",
            Ref:   "3",
            Title: "Backend Change",
            TestCases: []dto.TestCase{
                {ID: "31", Scenario: "first", Done: true},
                {ID: "32", Scenario: "second", Done: true},
            },
        },
    }
    m := NewModelWithClient(client)
    m.state = ChangeDetailsState
    m.changeList = m.changeList.WithDetail(dto.Change{
        ID:    "12",
        Ref:   "3",
        Title: "Backend Change",
        TestCases: []dto.TestCase{
            {ID: "31", Scenario: "first", Done: false},
            {ID: "32", Scenario: "second", Done: true},
        },
    })
    m.changeList.DetailSelected = 8

    got, cmd := sendRune(m, ' ')
    require.NotNil(t, cmd)
    assert.Equal(t, "saving test case", got.status)
    got = applyMsg(got, cmd())

    assert.Equal(t, []int{31}, client.testCaseDoneIDs)
    assert.Equal(t, []bool{true}, client.testCaseDoneUpdates)
    assert.Equal(t, []int{12}, client.changeGetIDs)
    assert.True(t, got.changeList.Detail.TestCases[0].Done)
    assert.Equal(t, ChangeDetailsState, got.state)
    assert.Equal(t, 8, got.changeList.DetailSelected)
}

func TestChangeDetailsNewTestcaseCreatesAndRefreshes(t *testing.T) {
    client := &fakeClient{
        gotChange: dto.Change{
            ID:    "12",
            Ref:   "3",
            Title: "Backend Change",
            TestCases: []dto.TestCase{
                {ID: "31", Scenario: "new scenario", Done: false, ChangeID: "12"},
            },
        },
    }
    m := NewModelWithClient(client)
    m.state = ChangeDetailsState
    m.changeList = m.changeList.WithDetail(dto.Change{ID: "12", Ref: "3", Title: "Backend Change"})

    got, cmd := sendCommand(m, "/new-testcase")
    require.Nil(t, cmd)
    assert.Equal(t, TestCaseCreateState, got.state)
    assert.Equal(t, "Write a Scenario", got.input.Placeholder)

    got.input.SetValue("new scenario")
    got, cmd = sendKey(got, tea.KeyEnter)
    require.NotNil(t, cmd)
    got = applyMsg(got, cmd())

    assert.Equal(t, ChangeDetailsState, got.state)
    assert.Equal(t, []dto.TestCase{{ChangeID: "12", Scenario: "new scenario"}}, client.testCaseCreateInputs)
    require.Len(t, got.changeList.Detail.TestCases, 1)
    assert.Equal(t, "new scenario", got.changeList.Detail.TestCases[0].Scenario)
}

func TestChangeDetailsTestcaseEnterEditsScenarioAndRefreshes(t *testing.T) {
    client := &fakeClient{
        gotChange: dto.Change{
            ID:    "12",
            Ref:   "3",
            Title: "Backend Change",
            TestCases: []dto.TestCase{
                {ID: "31", Scenario: "updated scenario", Done: false, ChangeID: "12"},
            },
        },
    }
    m := NewModelWithClient(client)
    m.state = ChangeDetailsState
    m.changeList = m.changeList.WithDetail(dto.Change{
        ID:    "12",
        Ref:   "3",
        Title: "Backend Change",
        TestCases: []dto.TestCase{
            {ID: "31", Scenario: "old scenario", Done: false, ChangeID: "12"},
        },
    })
    m.changeList.DetailSelected = 8

    got, cmd := sendKey(m, tea.KeyEnter)
    require.Nil(t, cmd)
    assert.Equal(t, TestCaseUpdateState, got.state)
    assert.Equal(t, "old scenario", got.input.Value())

    got.input.SetValue("updated scenario")
    got, cmd = sendKey(got, tea.KeyEnter)
    require.NotNil(t, cmd)
    got = applyMsg(got, cmd())

    assert.Equal(t, ChangeDetailsState, got.state)
    assert.Equal(t, []dto.TestCase{{ID: "31", Scenario: "updated scenario"}}, client.testCaseUpdateInputs)
    require.Len(t, got.changeList.Detail.TestCases, 1)
    assert.Equal(t, "updated scenario", got.changeList.Detail.TestCases[0].Scenario)
}

func TestChangeDetailsTestcaseDeleteConfirmsAndRefreshes(t *testing.T) {
    client := &fakeClient{
        gotChange: dto.Change{ID: "12", Ref: "3", Title: "Backend Change"},
    }
    m := NewModelWithClient(client)
    m.state = ChangeDetailsState
    m.changeList = m.changeList.WithDetail(dto.Change{
        ID:    "12",
        Ref:   "3",
        Title: "Backend Change",
        TestCases: []dto.TestCase{
            {ID: "31", Scenario: "old scenario", Done: false, ChangeID: "12"},
        },
    })
    m.changeList.DetailSelected = 8

    got, cmd := sendKey(m, tea.KeyDelete)
    require.Nil(t, cmd)
    assert.Equal(t, TestCaseDeleteConfirmation, got.state)
    assert.Equal(t, "Are you sure?", got.dropdown.label)

    got.dropdown.filter = "/yes"
    got, cmd = sendKey(got, tea.KeyEnter)
    require.NotNil(t, cmd)
    got = applyMsg(got, cmd())

    assert.Equal(t, ChangeDetailsState, got.state)
    assert.Equal(t, []int{31}, client.testCaseDeleteIDs)
    assert.Empty(t, got.changeList.Detail.TestCases)
}

func TestCtrlNShortcutsCreateChangeAndTestCase(t *testing.T) {
    changeList := NewModelWithClient(&fakeClient{})
    changeList.state = ChangesListState
    changeList.currentProject = dto.Option{ID: "7", Label: "Project Seven"}
    changeList.agentWorkspace = t.TempDir()
    got, cmd := sendKey(changeList, tea.KeyCtrlN)
    require.NotNil(t, cmd)
    assert.Equal(t, CreateDefState, got.state)
    assert.Equal(t, agent.StageDefEntry, got.agentFlow.Stage)

    detail := NewModelWithClient(&fakeClient{})
    detail.state = ChangeDetailsState
    detail.changeList = detail.changeList.WithDetail(dto.Change{ID: "12", Ref: "3", Title: "Backend Change"})
    got, cmd = sendKey(detail, tea.KeyCtrlN)
    require.Nil(t, cmd)
    assert.Equal(t, TestCaseCreateState, got.state)
    assert.Equal(t, "Write a Scenario", got.input.Placeholder)
}

func TestShortcutHelpRendersInFooter(t *testing.T) {
    m := NewModelWithClient(&fakeClient{})
    m.state = ChangeDetailsState
    m.width = 120
    m.changeList = m.changeList.WithDetail(dto.Change{ID: "12", Ref: "3", Title: "Backend Change"})

    lines := strings.Split(stripANSI(m.View()), "\n")
    require.NotEmpty(t, lines)
    footer := strings.Join(lines[max(0, len(lines)-4):], "\n")
    assert.Contains(t, lines[0], "ChangeDetailsScreen")
    assert.Contains(t, footer, "<ctrl+n> new testcase")
    assert.Contains(t, footer, "</> command")

    m.state = TestCaseUpdateState
    m.input.SetValue("scenario")
    lines = strings.Split(stripANSI(m.View()), "\n")
    footer = strings.Join(lines[max(0, len(lines)-4):], "\n")
    assert.Contains(t, footer, "<return> save")
    assert.Contains(t, footer, "<ctrl+c> delete prompt")

    m.state = TestCaseDeleteConfirmation
    m.openConfirmation(TestCaseDeleteConfirmation, ChangeDetailsState, ChangeDetailsState)
    lines = strings.Split(stripANSI(m.View()), "\n")
    footer = strings.Join(lines[max(0, len(lines)-4):], "\n")
    assert.Contains(t, footer, "<return> select")
    assert.Contains(t, footer, "<esc> or <ctrl+c> cancel")
}

func TestChangesListHeaderRendersFiltersAndTable(t *testing.T) {
    m := NewModelWithClient(&fakeClient{})
    m.state = ChangesListState
    m.width = 160
    m.changeList = m.changeList.WithRows([]dto.Change{{
        ID:          "12",
        Ref:         "3",
        Title:       "Backend Change",
        ChangePhase: "backlog",
        ChangeTypes: []string{"feature"},
        EpicID:      "5",
        EpicName:    "Epic Five",
        Spec:        "backend spec",
    }})
    m.changesFilters.phase = dto.Option{ID: "backlog", Label: "backlog"}
    m.changesFilters.typ = dto.Option{ID: "feature", Label: "feature"}
    m.changesFilters.epic = dto.Option{ID: "5", Label: "Epic Five"}
    m.changesFilters.find = "backend"

    lines := strings.Split(stripANSI(m.View()), "\n")
    require.GreaterOrEqual(t, len(lines), 7)
    assert.Contains(t, lines[0], "Make a change v0.1")
    assert.Contains(t, lines[0], "ChangesListScreen")
    assert.Contains(t, lines[1], "/filter-phase")
    assert.Contains(t, lines[1], "backlog")
    assert.Contains(t, lines[1], "/filter-type")
    assert.Contains(t, lines[1], "feature")
    assert.Contains(t, lines[1], "/filter-epic")
    assert.Contains(t, lines[1], "Epic Five")
    assert.Contains(t, lines[1], "/filter-find")
    assert.Contains(t, lines[1], "backend")
    assert.Contains(t, lines[2], "┌")
    assert.Equal(t, lipgloss.Width(lines[2]), lipgloss.Width(lines[1]))
    assert.Contains(t, lines[len(lines)-1], "<ctrl+n> new change")
    assert.Contains(t, lines[len(lines)-1], "</> command")
}

func TestChangesListFiltersRenderValuesPureWhite(t *testing.T) {
    m := NewModelWithClient(&fakeClient{})
    m.state = ChangesListState
    m.width = 160
    m.changeList = m.changeList.WithRows([]dto.Change{{
        ID:          "12",
        Ref:         "3",
        Title:       "Backend Change",
        ChangePhase: "backlog",
    }})
    m.changesFilters.typ = dto.Option{ID: "feature", Label: "feature"}

    view := m.View()
    assert.Contains(t, view, styles.Default.Muted.Render("/filter-type "))
    assert.Contains(t, view, lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Render("feature"))
}

func TestChangeDetailsTableTruncatesLongSpecAndPullRequestRows(t *testing.T) {
    m := NewModel()
    m.state = ChangeDetailsState
    m.width = 120
    m.height = 40
    m.changeList = m.changeList.WithDetail(dto.Change{
        ID:          "11",
        Ref:         "3",
        Slug:        "change-three",
        Title:       "Backend Change",
        ChangePhase: "backlog",
        EpicName:    "Epic Five",
        Spec:        strings.Repeat("spec content ", 180),
        PR:          "pull request start\n" + strings.Repeat("pull request middle ", 120) + "\npull request end",
        PRUrl:       "https://example.test/pr",
    })

    firstView := stripANSI(m.View())
    assert.Contains(t, firstView, "Ref │ 000003")
    assert.Contains(t, firstView, "Spec │ spec content")
    assert.Contains(t, firstView, "...")
    assert.NotContains(t, firstView, "pull request end")
    assert.Contains(t, firstView, "───────────┼")

    got, _ := sendKey(m, tea.KeyPgDown)
    scrolledView := stripANSI(got.View())
    assert.Contains(t, scrolledView, "PR │ pull request start")
    assert.Contains(t, scrolledView, "...")
    assert.NotContains(t, firstView, "pull request end")

    got, _ = sendKey(got, tea.KeyPgUp)
    backView := stripANSI(got.View())
    assert.Contains(t, backView, "Ref │ 000003")
}

func TestListSelectionDropdownTransitionsToDetails(t *testing.T) {
    m := NewModel()
    m.state = EpicsListState

    dropdown, _ := sendKey(m, tea.KeyEnter)
    require.Equal(t, ListSelectionDropDownState, dropdown.state)

    got, _ := sendKey(dropdown, tea.KeyEnter)
    assert.Equal(t, EpicDetailsState, got.state)
}

func TestCreateUpdateSaveCancelTransitions(t *testing.T) {
    tests := []struct {
        start   State
        command string
        want    State
    }{
        {start: ChangeCreateState, command: "/cancel", want: ChangesListState},
        {start: ChangeDetailsState, command: "/edit-spec", want: ChangeDetailsState},
        {start: ChangeUpdateState, command: "/cancel", want: ChangeDetailsState},
        {start: ChangeDetailsState, command: "/new-testcase", want: TestCaseCreateState},
        {start: TestCaseUpdateState, command: "/cancel", want: ChangeDetailsState},
        {start: TestCaseDetailsState, command: "/edit", want: TestCaseUpdateState},
        {start: EpicsListState, command: "/new-epic", want: EpicCreateState},
        {start: EpicCreateState, command: "/save", want: EpicDetailsState},
        {start: EpicDetailsState, command: "/edit", want: EpicUpdateState},
        {start: ProjectsListState, command: "/new-project", want: ProjectCreateState},
        {start: ProjectDetailsState, command: "/edit", want: ProjectUpdateState},
    }

    for _, tt := range tests {
        t.Run(string(tt.start)+tt.command, func(t *testing.T) {
            m := NewModel()
            m.state = tt.start

            got, _ := sendCommand(m, tt.command)

            assert.Equal(t, tt.want, got.state)
        })
    }
}

func TestChangeDetailsRejectsLegacyEditCommand(t *testing.T) {
    m := NewModel()
    m.state = ChangeDetailsState

    got, _ := sendCommand(m, "/edit")

    assert.Equal(t, ChangeDetailsState, got.state)
    assert.Contains(t, got.err, "unknown command: /edit")
}

func TestSlashCommandTransitionsByState(t *testing.T) {
    tests := []struct {
        start        State
        command      string
        want         State
        wantPrevious State
    }{
        {start: ChangesListState, command: "/help", want: ChangesHelpState},
        {start: ChangesListState, command: "/clear-filters", want: ChangesListState},
        {start: ChangesListState, command: "/return", want: MainState},
        {start: ChangeDetailsState, command: "/return", want: ChangesListState},
        {start: TestCaseDetailsState, command: "/new-testcase", want: TestCaseCreateState},
        {start: TestCaseDetailsState, command: "/save", want: TestCaseDetailsState},
        {start: TestCaseDetailsState, command: "/cancel", want: TestCaseDetailsState},
        {start: TestCaseDetailsState, command: "/return", want: ChangeDetailsState},
        {start: EpicsListState, command: "/help", want: EpicsHelpState},
        {start: EpicsListState, command: "/find", want: FindInputState, wantPrevious: EpicsListState},
        {start: EpicsListState, command: "/return", want: MainState},
        {start: EpicDetailsState, command: "/help", want: EpicsHelpState},
        {start: EpicDetailsState, command: "/find", want: FindInputState, wantPrevious: EpicDetailsState},
        {start: EpicDetailsState, command: "/return", want: EpicsListState},
        {start: EpicCreateState, command: "/cancel", want: EpicsListState},
        {start: EpicUpdateState, command: "/save", want: EpicDetailsState},
        {start: EpicUpdateState, command: "/cancel", want: EpicDetailsState},
        {start: ProjectsListState, command: "/help", want: ProjectsHelpState},
        {start: ProjectsListState, command: "/find", want: FindInputState, wantPrevious: ProjectsListState},
        {start: ProjectsListState, command: "/return", want: MainState},
        {start: ProjectDetailsState, command: "/help", want: ProjectsHelpState},
        {start: ProjectDetailsState, command: "/find", want: FindInputState, wantPrevious: ProjectDetailsState},
        {start: ProjectDetailsState, command: "/return", want: ProjectsListState},
        {start: ProjectCreateState, command: "/cancel", want: ProjectsListState},
        {start: ProjectUpdateState, command: "/cancel", want: ProjectDetailsState},
        {start: MainHelpState, command: "/return", want: MainState},
        {start: ChangesHelpState, command: "/return", want: ChangesListState},
        {start: EpicsHelpState, command: "/return", want: EpicsListState},
        {start: ProjectsHelpState, command: "/return", want: ProjectsListState},
    }

    for _, tt := range tests {
        t.Run(string(tt.start)+tt.command, func(t *testing.T) {
            m := NewModel()
            m.state = tt.start

            got, _ := sendCommand(m, tt.command)

            assert.Equal(t, tt.want, got.state)
            if tt.wantPrevious != "" {
                assert.Equal(t, tt.wantPrevious, got.previousState)
            }
        })
    }
}

func TestDeleteCommandsOpenExpectedConfirmations(t *testing.T) {
    tests := []struct {
        start State
        want  State
    }{
        {start: ChangeDetailsState, want: ChangeDeleteConfirmation},
        {start: TestCaseDetailsState, want: TestCaseDeleteConfirmation},
        {start: EpicDetailsState, want: EpicDeleteConfirmation},
        {start: ProjectDetailsState, want: ProjectDeleteConfirmation},
    }

    for _, tt := range tests {
        t.Run(string(tt.start), func(t *testing.T) {
            m := NewModel()
            m.state = tt.start

            got, _ := sendCommand(m, "/delete")

            assert.Equal(t, tt.want, got.state)
        })
    }
}

func TestChangeDetailsCommandsAreExact(t *testing.T) {
    assert.Equal(t, []string{
        "/reference",
        "/new-testcase",
        "/phase",
        "/epic",
        "/types",
        "/edit-spec",
        "/delete",
        "/return",
    }, commandsByState[ChangeDetailsState])
}

func TestChangesListCommandsAreExact(t *testing.T) {
    assert.Equal(t, []string{
        "/new-change",
        "/phase-filter",
        "/epic-filter",
        "/type-filter",
        "/find-filter",
        "/clear-filters",
        "/help",
        "/return",
    }, commandsByState[ChangesListState])
}

func TestReturnAndEscapeTransitions(t *testing.T) {
    returnTests := []struct {
        start State
        want  State
    }{
        {start: ChangesListState, want: MainState},
        {start: ChangeDetailsState, want: ChangesListState},
        {start: TestCaseDetailsState, want: ChangeDetailsState},
        {start: EpicsListState, want: MainState},
        {start: EpicDetailsState, want: EpicsListState},
        {start: ProjectsListState, want: MainState},
        {start: ProjectDetailsState, want: ProjectsListState},
        {start: MainHelpState, want: MainState},
        {start: ChangesHelpState, want: ChangesListState},
        {start: EpicsHelpState, want: EpicsListState},
        {start: ProjectsHelpState, want: ProjectsListState},
    }

    for _, tt := range returnTests {
        t.Run("return "+string(tt.start), func(t *testing.T) {
            m := NewModel()
            m.state = tt.start

            got, _ := sendKey(m, tea.KeyEsc)

            assert.Equal(t, tt.want, got.state)
        })
    }

    m := NewModel()
    got, cmd := sendKey(m, tea.KeyEsc)
    assert.Equal(t, DoneState, got.state)
    assert.True(t, got.quitting)
    require.NotNil(t, cmd)

    m = NewModel()
    m.state = ChangeCreateState
    got, _ = sendKey(m, tea.KeyEsc)
    assert.Equal(t, ChangesListState, got.state)
}

func TestSelectorDropdownsLoadAndReturn(t *testing.T) {
    client := &fakeClient{
        projects:  []dto.Option{{ID: "7", Label: "Project Seven"}},
        phases:    []dto.Option{{ID: "backlog", Label: "backlog"}},
        types:     []dto.Option{{ID: "feature", Label: "feature"}},
        epics:     []dto.Option{{ID: "3", Label: "Epic Three"}},
        gotChange: dto.Change{ID: "12", Ref: "3", Title: "Backend Change"},
    }

    m := newModelWithOptionCatalog(client)
    got, cmd := sendCommand(m, "/select-project")
    require.Equal(t, SelectProjectDropDown, got.state)
    require.NotNil(t, cmd)
    got = applyMsg(got, cmd())
    got, _ = sendKey(got, tea.KeyEnter)
    assert.Equal(t, MainState, got.state)
    assert.Equal(t, "7", got.currentProject.ID)

    got.state = ChangeDetailsState
    got.changeList = got.changeList.WithDetail(dto.Change{ID: "12", Ref: "3", Title: "Backend Change"})
    got, cmd = sendCommand(got, "/phase")
    got = applyMsg(got, cmd())
    got, cmd = sendKey(got, tea.KeyEnter)
    require.NotNil(t, cmd)
    got = applyMsg(got, cmd())
    assert.Equal(t, ChangeDetailsState, got.state)
    assert.Zero(t, client.phaseCalls)
    assert.Equal(t, []string{"backlog"}, client.changePhaseUpdates)

    got, cmd = sendCommand(got, "/types")
    got = applyMsg(got, cmd())
    got, cmd = sendKey(got, tea.KeySpace)
    require.Nil(t, cmd)
    got, cmd = sendKey(got, tea.KeyEnter)
    require.NotNil(t, cmd)
    got = applyMsg(got, cmd())
    assert.Equal(t, ChangeDetailsState, got.state)
    assert.Zero(t, client.typeCalls)
    assert.Equal(t, [][]string{{"feature"}}, client.changeTypesUpdates)

    got, cmd = sendCommand(got, "/epic")
    got = applyMsg(got, cmd())
    got, _ = sendKey(got, tea.KeyUp)
    got, cmd = sendKey(got, tea.KeyEnter)
    require.NotNil(t, cmd)
    got = applyMsg(got, cmd())
    assert.Equal(t, ChangeDetailsState, got.state)
    assert.Equal(t, 1, client.epicCalls)
    assert.Equal(t, "7", client.projectID)
    require.Len(t, client.changeEpicUpdates, 1)
    require.NotNil(t, client.changeEpicUpdates[0])
    assert.Equal(t, 3, *client.changeEpicUpdates[0])
}

func TestSelectProjectPersistsProjectIDToConfig(t *testing.T) {
    root := t.TempDir()
    path := filepath.Join(root, ".mch", "config.yaml")
    legacyPath := filepath.Join(root, "cli", ".config", "config.yaml")
    require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
    require.NoError(t, saveAppConfig(path, testAppConfig(appConfig{})))
    client := &fakeClient{
        projects: []dto.Option{{ID: "7", Label: "Project Seven"}},
    }
    m := newModelWithConfig(client, testAppConfig(appConfig{ConfigPath: path}))

    got, cmd := sendCommand(m, "/select-project")
    require.NotNil(t, cmd)
    got = applyMsg(got, cmd())
    got, _ = sendKey(got, tea.KeyEnter)

    assert.Equal(t, MainState, got.state)
    assert.Equal(t, "7", got.currentProject.ID)
    loadedFile, err := loadConfigFile(path)
    require.NoError(t, err)
    assert.Equal(t, 7, loadedFile.ProjectID)
    _, statErr := os.Stat(legacyPath)
    assert.True(t, os.IsNotExist(statErr))
}

func TestSelectorFailureAndEscapePreservePreviousState(t *testing.T) {
    client := &fakeClient{err: errors.New("backend unavailable")}
    m := NewModelWithClient(client)
    m.state = ChangeDetailsState

    got, cmd := sendCommand(m, "/phase")
    got = applyMsg(got, cmd())
    assert.Equal(t, SelectPhaseDropDown, got.state)
    assert.NotEmpty(t, got.err)

    got, _ = sendKey(got, tea.KeyEsc)
    assert.Equal(t, ChangeDetailsState, got.state)
}

func TestFilterSelectorsReturnToChangesList(t *testing.T) {
    client := &fakeClient{
        phases: []dto.Option{{ID: "done", Label: "done"}},
        epics:  []dto.Option{{ID: "epic-1", Label: "Epic One"}},
        types:  []dto.Option{{ID: "test", Label: "test"}},
    }
    m := newModelWithOptionCatalog(client)
    m.state = ChangesListState
    m.currentProject = dto.Option{ID: "project-1", Label: "Project One"}

    got, cmd := sendCommand(m, "/phase-filter")
    require.NotNil(t, cmd)
    assert.Equal(t, ChangesListState, got.state)
    assert.Contains(t, got.View(), "ChangesListScreen")
    got = applyMsg(got, cmd())
    phaseDropdown := strings.Split(got.dropdownView(80), "\n")
    require.GreaterOrEqual(t, len(phaseDropdown), 3)
    assert.True(t, strings.HasPrefix(stripANSI(phaseDropdown[1]), "    -done"))
    assert.True(t, strings.HasPrefix(stripANSI(phaseDropdown[len(phaseDropdown)-1]), "    /clear"))
    got, _ = sendKey(got, tea.KeyEnter)
    assert.Equal(t, ChangesListState, got.state)
    assert.Equal(t, "done", got.changesFilters.phase.ID)
    assert.Equal(t, "done", got.changesFilters.phase.Label)

    got, cmd = sendCommand(got, "/epic-filter")
    require.NotNil(t, cmd)
    assert.Equal(t, ChangesListState, got.state)
    assert.Contains(t, got.View(), "ChangesListScreen")
    got = applyMsg(got, cmd())
    got, _ = sendKey(got, tea.KeyEnter)
    assert.Equal(t, ChangesListState, got.state)
    assert.Equal(t, "epic-1", got.changesFilters.epic.ID)

    got, cmd = sendCommand(got, "/type-filter")
    require.NotNil(t, cmd)
    assert.Equal(t, ChangesListState, got.state)
    assert.Contains(t, got.View(), "ChangesListScreen")
    got = applyMsg(got, cmd())
    got, _ = sendKey(got, tea.KeyEnter)
    assert.Equal(t, ChangesListState, got.state)
    assert.Equal(t, "test", got.changesFilters.typ.ID)

    got, cmd = sendCommand(got, "/phase-filter")
    require.NotNil(t, cmd)
    got = applyMsg(got, cmd())
    got.dropdown.filter = "/clear"
    got, _ = sendKey(got, tea.KeyEnter)
    assert.Equal(t, ChangesListState, got.state)
    assert.Empty(t, got.changesFilters.phase.ID)
    assert.Equal(t, "epic-1", got.changesFilters.epic.ID)
    assert.Equal(t, "test", got.changesFilters.typ.ID)

    got, _ = sendCommand(got, "/find-filter")
    assert.Equal(t, FindInputState, got.state)
    got.input.SetValue("needle")
    got, _ = sendKey(got, tea.KeyEnter)
    assert.Equal(t, ChangesListState, got.state)
    assert.Equal(t, "needle", got.changesFilters.find)

    got, _ = sendKey(got, tea.KeyCtrlF)
    assert.Equal(t, FindInputState, got.state)
    assert.Equal(t, ChangesListState, got.previousState)
    assert.Equal(t, "needle", got.input.Value())
    got, _ = sendKey(got, tea.KeyEnter)
    assert.Equal(t, ChangesListState, got.state)
    assert.Equal(t, "needle", got.changesFilters.find)

    got, _ = sendCommand(got, "/clear-filters")
    assert.Empty(t, got.changesFilters.epic.ID)
    assert.Empty(t, got.changesFilters.typ.ID)
    assert.Empty(t, got.changesFilters.find)
}

func TestFindInputHighlightsAndEmptyFindErrors(t *testing.T) {
    m := NewModel()
    m.state = MainHelpState

    got, _ := sendCommand(m, "/find")
    assert.Equal(t, FindInputState, got.state)
    assert.Equal(t, MainHelpState, got.previousState)

    got.input.SetValue("phase")
    got, _ = sendKey(got, tea.KeyEnter)
    assert.Equal(t, MainHelpState, got.state)
    assert.Equal(t, "phase", got.helpQuery)

    got, _ = sendCommand(got, "/find")
    got, _ = sendKey(got, tea.KeyEnter)
    assert.Equal(t, MainHelpState, got.state)
    assert.NotEmpty(t, got.err)
}

func TestConfirmationRequiresYesOrCancel(t *testing.T) {
    m := NewModelWithClient(&fakeClient{})
    m.state = ChangeDetailsState
    m.changeList = m.changeList.WithDetail(dto.Change{ID: "12", Title: "Backend Change"})

    got, _ := sendCommand(m, "/delete")
    assert.Equal(t, ChangeDeleteConfirmation, got.state)
    assert.Equal(t, "Are you sure?", got.dropdown.label)

    got.dropdown.filter = "/bogus"
    got, _ = sendKey(got, tea.KeyEnter)
    assert.Equal(t, ChangeDeleteConfirmation, got.state)
    assert.NotEmpty(t, got.err)

    got.dropdown.filter = "/no"
    got, _ = sendKey(got, tea.KeyEnter)
    assert.Equal(t, ChangeDetailsState, got.state)

    got, _ = sendCommand(m, "/delete")
    got, _ = sendKey(got, tea.KeyCtrlC)
    assert.Equal(t, ChangeDetailsState, got.state)
}

func TestChangeDeleteConfirmationDeletesAndReloadsList(t *testing.T) {
    client := &fakeClient{
        changeRows: []dto.Change{{ID: "13", Ref: "4", Title: "Remaining Change"}},
    }
    m := NewModelWithClient(client)
    m.currentProject = dto.Option{ID: "7", Label: "Project Seven"}
    m.state = ChangeDetailsState
    m.changeList = m.changeList.WithDetail(dto.Change{ID: "12", Ref: "3", Title: "Backend Change"})

    got, _ := sendCommand(m, "/delete")
    require.Equal(t, ChangeDeleteConfirmation, got.state)

    got.dropdown.filter = "/yes"
    got, cmd := sendKey(got, tea.KeyEnter)
    require.NotNil(t, cmd)
    assert.Equal(t, ChangeDetailsState, got.state)
    assert.Equal(t, "deleting change", got.status)

    updated, reload := got.Update(cmd())
    got = updated.(Model)
    require.Equal(t, ChangesListState, got.state)
    assert.True(t, got.changeList.Loading)
    assert.Equal(t, []int{12}, client.changeDeleteIDs)

    require.NotNil(t, reload)
    got = applyMsg(got, reload())

    assert.Equal(t, ChangesListState, got.state)
    assert.Equal(t, []string{"7"}, client.changeListProjectIDs)
    assert.Equal(t, []dto.Change{{ID: "13", Ref: "4", Title: "Remaining Change"}}, got.changeList.Rows)
}

func TestChangeDeleteFailurePreservesDetail(t *testing.T) {
    client := &fakeClient{changeDeleteErr: errors.New("delete failed")}
    m := NewModelWithClient(client)
    m.currentProject = dto.Option{ID: "7", Label: "Project Seven"}
    m.state = ChangeDetailsState
    m.changeList = m.changeList.WithDetail(dto.Change{ID: "12", Ref: "3", Title: "Backend Change"})

    got, _ := sendCommand(m, "/delete")
    got.dropdown.filter = "/yes"
    got, cmd := sendKey(got, tea.KeyEnter)
    require.NotNil(t, cmd)
    got = applyMsg(got, cmd())

    assert.Equal(t, ChangeDetailsState, got.state)
    assert.Equal(t, "delete failed", got.err)
    assert.Equal(t, []int{12}, client.changeDeleteIDs)
    assert.Zero(t, client.changeListCalls)
}

func TestCommandDropdownFiltersAndExecutesSelection(t *testing.T) {
    m := NewModel()

    got, _ := sendRune(m, '/')
    require.Equal(t, MainState, got.state)
    require.Equal(t, dropdownCommand, got.dropdown.kind)
    assert.Contains(t, got.View(), "MainScreen")
    assert.NotContains(t, got.View(), "CommandDropDownScreen")
    dropdown := got.dropdownView(80)
    lines := strings.Split(dropdown, "\n")
    require.GreaterOrEqual(t, len(lines), 3)
    assert.True(t, strings.HasPrefix(stripANSI(lines[1]), "    /changes"))
    assert.Equal(t, "15", fmt.Sprint(styles.Default.Selection.GetForeground()))
    assert.Empty(t, strings.TrimSpace(stripANSI(lines[len(lines)-1])))
    got, _ = sendRune(got, 'e')
    got, _ = sendRune(got, 'p')
    got, _ = sendKey(got, tea.KeyEnter)

    assert.Equal(t, EpicsListState, got.state)
}

func TestConfigCommandRendersResolvedConfigWithoutBackendCalls(t *testing.T) {
    client := &fakeClient{}
    cfg := testAppConfig(appConfig{ProjectID: 7})
    m := newModelWithConfig(client, cfg)
    m.width = 160

    got, cmd := sendCommand(m, "/config")

    require.Nil(t, cmd)
    assert.Equal(t, ConfigState, got.state)
    assert.Zero(t, client.listCalls)
    assert.Zero(t, client.rowListCalls)
    assert.Zero(t, client.changeListCalls)
    view := stripANSI(got.View())
    assert.Contains(t, view, "ConfigScreen")
    assert.Contains(t, view, "repository_root: /repo")
    assert.Contains(t, view, "config_path: /repo/.mch/config.yaml")
    assert.Contains(t, view, "backend_url: http://localhost:8080")
    assert.NotContains(t, view, "temp_dir:")
    assert.Contains(t, view, "project_id: 7")
    assert.Contains(t, view, "flow_dir: /repo/.mch/default")
    assert.Contains(t, view, "slug: def")
    assert.Contains(t, view, "prompt: prompts/change-def.md")
    assert.Contains(t, view, "entry: make def-entry")
    assert.Contains(t, view, "exec: make def-exec")
    assert.Contains(t, view, "exit: make def-exit")
    assert.Contains(t, view, "stage_modes:")
    assert.Contains(t, view, "task_statuses:")
    assert.Contains(t, view, "task_steps:")
}

func TestConfigViewReturnsWithoutSavingOrCallingBackend(t *testing.T) {
    root := t.TempDir()
    writeMCHFixture(t, root, "backend_url: http://backend.test\n"+"temp_dir: /tmp/custom-mch\n"+"project_id: 5\n")
    cfg, err := loadAppConfig(root)
    require.NoError(t, err)
    before, err := os.ReadFile(cfg.ConfigPath)
    require.NoError(t, err)
    client := &fakeClient{}
    m := newModelWithConfig(client, cfg)

    got, _ := sendCommand(m, "/config")
    got, cmd := sendCommand(got, "/return")
    require.Nil(t, cmd)
    assert.Equal(t, MainState, got.state)

    got, _ = sendCommand(m, "/config")
    got, cmd = sendKey(got, tea.KeyEsc)
    require.Nil(t, cmd)
    assert.Equal(t, MainState, got.state)

    got, _ = sendCommand(m, "/config")
    got, cmd = sendKey(got, tea.KeyCtrlC)
    require.Nil(t, cmd)
    assert.Equal(t, MainState, got.state)

    after, err := os.ReadFile(cfg.ConfigPath)
    require.NoError(t, err)
    assert.Equal(t, string(before), string(after))
    assert.Zero(t, client.listCalls)
    assert.Zero(t, client.rowListCalls)
    assert.Zero(t, client.changeListCalls)
}

func TestCommandDropdownPreservesUnderlyingScreenForEveryCommandState(t *testing.T) {
    for state := range commandsByState {
        t.Run(string(state), func(t *testing.T) {
            m := NewModel()
            m.state = state

            got, _ := sendRune(m, '/')

            assert.Equal(t, state, got.state)
            assert.Equal(t, dropdownCommand, got.dropdown.kind)
            assert.Equal(t, CommandDropDownState, got.dropdown.state)
            assert.Contains(t, got.View(), headerScreenName(state))
            assert.NotContains(t, got.View(), headerScreenName(CommandDropDownState))
        })
    }
}

func TestProjectsCommandMenuPreservesListTitle(t *testing.T) {
    m := NewModelWithClient(&fakeClient{})
    m.state = ProjectsListState
    m.projectList.Rows = []dto.Project{{ID: "7", Name: "Project Seven"}}

    got, _ := sendRune(m, '/')

    assert.Equal(t, ProjectsListState, got.state)
    assert.Equal(t, dropdownCommand, got.dropdown.kind)
    view := stripANSI(got.View())
    assert.Contains(t, view, "ProjectsListScreen")
    assert.Contains(t, view, "/new-project")
    assert.Contains(t, view, "/help")
    assert.Contains(t, view, "/find")
    assert.Contains(t, view, "/return")
}

func TestCreateStatesUseContextSpecificNewCommandVocabulary(t *testing.T) {
    createCommands := map[State]string{
        ChangesListState:     "/new-change",
        ChangeDetailsState:   "/new-testcase",
        TestCaseDetailsState: "/new-testcase",
        EpicsListState:       "/new-epic",
        ProjectsListState:    "/new-project",
    }
    for state, want := range createCommands {
        t.Run(string(state), func(t *testing.T) {
            commands := commandsByState[state]
            assert.Contains(t, commands, want)
            assert.NotContains(t, commands, "/new")
            assert.NotContains(t, commands, "/create")
        })
    }
}

func TestUpdateStatesUseEditCommandVocabulary(t *testing.T) {
    updateSources := []State{
        ChangeDetailsState,
        TestCaseDetailsState,
        EpicDetailsState,
        ProjectDetailsState,
    }
    for _, state := range updateSources {
        t.Run(string(state), func(t *testing.T) {
            commands := commandsByState[state]
            if state == ChangeDetailsState {
                assert.Contains(t, commands, "/edit-spec")
                assert.NotContains(t, commands, "/edit")
            } else {
                assert.Contains(t, commands, "/edit")
            }
            assert.NotContains(t, commands, "/update")
        })
    }
}

func TestNoPersistenceAPICallsForNavigationOnlyActions(t *testing.T) {
    client := &fakeClient{
        phases: []dto.Option{{ID: "backlog", Label: "backlog"}},
    }
    m := newModelWithOptionCatalog(client)
    m.state = ChangeDetailsState

    got, _ := sendCommand(m, "/save")
    got, _ = sendCommand(got, "/delete")
    got.dropdown.filter = "/yes"
    got, _ = sendKey(got, tea.KeyEnter)
    assert.Zero(t, client.listCalls)
    assert.Zero(t, client.rowListCalls)
    assert.Zero(t, client.phaseCalls)
    assert.Zero(t, client.typeCalls)
    assert.Zero(t, client.epicCalls)

    got.state = ChangesListState
    got, cmd := sendCommand(got, "/phase-filter")
    require.NotNil(t, cmd)
    got = applyMsg(got, cmd())
    got, _ = sendKey(got, tea.KeyEnter)
    assert.Zero(t, client.phaseCalls)
}

func TestEveryDummyScreenTitleRendersExactly(t *testing.T) {
    tests := []struct {
        state State
        title string
    }{
        {MainState, "MainScreen"},
        {ChangesListState, "ChangesListScreen"},
        {ChangeDetailsState, "ChangeDetailsScreen"},
        {TestCaseDetailsState, "TestCaseDetailsScreen - Title: Test Case Details"},
        {ChangeCreateState, "ChangeCreateScreen - Title: New Change"},
        {ChangeUpdateState, "ChangeUpdateScreen"},
        {TestCaseCreateState, "TestCaseCreateScreen - Title: New Test Case"},
        {TestCaseUpdateState, "TestCaseUpdateScreen - Title: Edit Test Case"},
        {EpicsListState, "EpicsListScreen - Title: Epics List"},
        {EpicDetailsState, "EpicDetailsScreen - Title: Epic Details"},
        {EpicCreateState, "EpicCreateScreen - Title: New Epic"},
        {EpicUpdateState, "EpicUpdateScreen - Title: Edit Epic"},
        {ProjectsListState, "ProjectsListScreen"},
        {ProjectDetailsState, "ProjectDetailsScreen"},
        {ProjectCreateState, "ProjectCreateScreen - Title: New Project"},
        {ProjectUpdateState, "ProjectUpdateScreen - Title: Edit Project"},
        {MainHelpState, "MainHelpScreen - Title: Main Help"},
        {ChangesHelpState, "ChangesHelpScreen - Title: Changes Help"},
        {EpicsHelpState, "EpicsHelpScreen - Title: Epics Help"},
        {ProjectsHelpState, "ProjectsHelpScreen - Title: Projects Help"},
        {FindInputState, "FindInputScreen - Title: Find"},
        {CommandDropDownState, "CommandDropDownScreen"},
        {ListSelectionDropDownState, "ListSelectionDropDownScreen - Title: Select Item"},
        {SelectProjectDropDown, "SelectProjectDropDownScreen - Title: Select Project"},
        {SelectPhaseDropDown, "SelectChangePhasesDropDownScreen - Title: Select Change Phases"},
        {SelectEpicDropDown, "SelectEpicDropDownScreen - Title: Select Epic"},
        {SelectTypesDropDown, "SelectChangeTypesDropDownScreen - Title: Select Change Types"},
        {ChangeDeleteConfirmation, "ChangeDeleteConfirmationScreen - Title: Are you sure?"},
        {TestCaseDeleteConfirmation, "TestCaseDeleteConfirmationScreen - Title: Are you sure?"},
        {EpicDeleteConfirmation, "EpicDeleteConfirmationScreen - Title: Are you sure?"},
        {ProjectDeleteConfirmation, "ProjectDeleteConfirmationScreen - Title: Are you sure?"},
    }

    for _, tt := range tests {
        t.Run(string(tt.state), func(t *testing.T) {
            m := NewModel()
            m.state = tt.state

            view := m.View()
            if tt.state == ChangesListState {
                assert.Contains(t, view, "/filter-phase")
            } else {
                assert.Contains(t, view, headerScreenName(tt.state))
            }
            assert.Contains(t, view, "Make a change v0.1")
        })
    }
}

func sendCommand(m Model, command string) (Model, tea.Cmd) {
    updated, cmd := m.executeCommand(command)
    return updated.(Model), cmd
}

func headerScreenName(state State) string {
    title := screenTitle(state)
    if before, _, ok := strings.Cut(title, " - "); ok {
        return before
    }
    return title
}

func sendRune(m Model, r rune) (Model, tea.Cmd) {
    updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
    return updated.(Model), cmd
}

func sendKey(m Model, key tea.KeyType) (Model, tea.Cmd) {
    return sendKeyMsg(m, tea.KeyMsg{Type: key})
}

func sendKeyMsg(m Model, msg tea.KeyMsg) (Model, tea.Cmd) {
    updated, cmd := m.Update(msg)
    return updated.(Model), cmd
}

func applyMsg(m Model, msg tea.Msg) Model {
    updated, _ := m.Update(msg)
    return updated.(Model)
}

func applyCommand(m Model, cmd tea.Cmd) Model {
    msg := cmd()
    if batch, ok := msg.(tea.BatchMsg); ok {
        for _, next := range batch {
            if next == nil {
                continue
            }
            m = applyMsg(m, next())
        }
        return m
    }
    return applyMsg(m, msg)
}

func readTestFile(t *testing.T, path string) string {
    t.Helper()
    content, err := os.ReadFile(path)
    require.NoError(t, err)
    return string(content)
}

func stripANSI(value string) string {
    var b strings.Builder
    inEscape := false
    for _, r := range value {
        if inEscape {
            if r == '[' || (r >= '0' && r <= '?') {
                continue
            }
            if r >= '@' && r <= '~' {
                inEscape = false
            }
            continue
        }
        if r == '\x1b' {
            inEscape = true
            continue
        }
        b.WriteRune(r)
    }
    return b.String()
}

func assertDefBeforePrompt(t *testing.T, rendered string, defText string, promptText string) {
    t.Helper()
    view := stripANSI(rendered)
    defIndex := strings.Index(view, defText)
    promptIndex := strings.Index(view, promptText)
    require.NotEqual(t, -1, defIndex, "expected def text %q in view:\n%s", defText, view)
    require.NotEqual(t, -1, promptIndex, "expected prompt text %q in view:\n%s", promptText, view)
    assert.Less(t, defIndex, promptIndex, "def preview should render before prompt")
}
