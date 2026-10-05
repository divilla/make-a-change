package app

import (
	"bytes"
	"cli/internal/changes"
	"cli/internal/dto"
	"cli/internal/projects"
	"cli/internal/styles"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/cursor"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeClient struct {
	projects                 []dto.Option
	projectRows              []dto.Project
	createdProject           dto.Project
	updatedProject           dto.Project
	gotProject               dto.Project
	changeRows               []dto.ChangeView
	createdChange            dto.ChangeView
	gotChange                dto.ChangeView
	epics                    []dto.Option
	phases                   []dto.Option
	types                    []dto.Option
	err                      error
	createErr                error
	updateErr                error
	getErr                   error
	changeCreateErr          error
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
	changeTitleUpdateCalls   int
	changeBriefUpdateCalls   int
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
	changeTitleUpdates       []string
	changeBriefUpdates       []string
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

func (f *fakeClient) ListProjectRows(context.Context) ([]dto.Project, error) {
	f.rowListCalls++
	f.listCalls++
	if f.projects != nil {
		rows := make([]dto.Project, 0, len(f.projects))
		for _, o := range f.projects {
			id, _ := strconv.Atoi(o.ID)
			rows = append(rows, dto.Project{ID: id, Name: o.Label})
		}
		return rows, f.err
	}
	return f.projectRows, f.err
}

func (f *fakeClient) GetProject(_ context.Context, id int) (dto.Project, error) {
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

func (f *fakeClient) CreateProject(_ context.Context, name string) (int, error) {
	f.createCalls++
	f.createNames = append(f.createNames, name)
	if f.createErr != nil {
		return 0, f.createErr
	}
	if f.err != nil {
		return 0, f.err
	}
	return f.createdProject.ID, nil
}

func (f *fakeClient) UpdateProject(_ context.Context, id int, name string) error {
	f.updateCalls++
	f.updateIDs = append(f.updateIDs, id)
	f.updateNames = append(f.updateNames, name)
	if f.updateErr != nil {
		return f.updateErr
	}
	if f.err != nil {
		return f.err
	}
	return nil
}

func (f *fakeClient) ListChangeRows(_ context.Context, project int) ([]dto.Change, error) {
	f.changeListCalls++
	f.changeListProjectIDs = append(f.changeListProjectIDs, strconv.Itoa(project))
	if f.err != nil {
		return nil, f.err
	}
	rows := make([]dto.Change, 0, len(f.changeRows))
	for _, v := range f.changeRows {
		w := fakeWire(v)
		w.ProjectID = project
		rows = append(rows, w)
	}
	return rows, nil
}

func (f *fakeClient) GetChange(_ context.Context, id int) (dto.Change, error) {
	f.requestOrder = append(f.requestOrder, "change/details")
	f.changeGetCalls++
	f.changeGetIDs = append(f.changeGetIDs, id)
	if f.changeGetErr != nil {
		return dto.Change{}, f.changeGetErr
	}
	if f.err != nil {
		return dto.Change{}, f.err
	}
	v := fakeWire(f.gotChange)
	if v.ID == 0 {
		v.ID = id
	}
	return v, nil
}

func (f *fakeClient) CreateChange(_ context.Context, input dto.ChangeCreateInput) (int, error) {
	f.requestOrder = append(f.requestOrder, "change/create")
	f.changeCreateCalls++
	f.changeCreateInputs = append(f.changeCreateInputs, input)
	if f.changeCreateErr != nil {
		return 0, f.changeCreateErr
	}
	if f.err != nil {
		return 0, f.err
	}
	id, _ := strconv.Atoi(f.createdChange.ID)
	return id, nil
}

func (f *fakeClient) UpdateChangeTitle(_ context.Context, _ int, title string) error {
	f.changeTitleUpdateCalls++
	f.changeTitleUpdates = append(f.changeTitleUpdates, title)
	if f.changeUpdateErr != nil {
		return f.changeUpdateErr
	}
	return nil
}

func (f *fakeClient) UpdateChangeSlug(_ context.Context, _ int, slug string) error {
	ref, _, ok := strings.Cut(f.gotChange.RefSlug, "-")
	if ok {
		f.gotChange.RefSlug = ref + "-" + slug
	}
	return nil
}

func (f *fakeClient) UpdateChangePRUrl(_ context.Context, _ int, prURL string) error {
	f.changePRUrlUpdateCalls++
	f.changePRUrlUpdates = append(f.changePRUrlUpdates, prURL)
	if f.changeUpdateErr != nil {
		return f.changeUpdateErr
	}
	return nil
}

func (f *fakeClient) UpdateChangeTypes(_ context.Context, _ int, changeTypes []string) error {
	f.requestOrder = append(f.requestOrder, "change/update-types")
	f.changeTypesUpdateCalls++
	f.changeTypesUpdates = append(f.changeTypesUpdates, append([]string{}, changeTypes...))
	if f.changeTypesUpdateErr != nil {
		return f.changeTypesUpdateErr
	}
	if f.changeUpdateErr != nil {
		return f.changeUpdateErr
	}
	return nil
}

func (f *fakeClient) UpdateChangePhase(_ context.Context, _ int, changePhase string) error {
	f.changePhaseUpdateCalls++
	f.changePhaseUpdates = append(f.changePhaseUpdates, changePhase)
	if f.changeUpdateErr != nil {
		return f.changeUpdateErr
	}
	return nil
}

func (f *fakeClient) UpdateChangeActive(_ context.Context, _ int, open bool) error {
	f.changeOpenUpdateCalls++
	f.changeOpenUpdates = append(f.changeOpenUpdates, open)
	if f.changeUpdateErr != nil {
		return f.changeUpdateErr
	}
	return nil
}

func (f *fakeClient) UpdateChangeEpic(_ context.Context, _ int, epicID *int) error {
	f.changeEpicUpdateCalls++
	f.changeEpicUpdates = append(f.changeEpicUpdates, epicID)
	if f.changeUpdateErr != nil {
		return f.changeUpdateErr
	}
	return nil
}

func (f *fakeClient) CreateTestCase(_ context.Context, changeID int, scenario string) (int, error) {
	f.requestOrder = append(f.requestOrder, "test-case/create")
	f.testCaseCreateCalls++
	f.testCaseCreateInputs = append(f.testCaseCreateInputs, dto.TestCase{ChangeID: changeID, Scenario: scenario})
	if f.changeUpdateErr != nil {
		return 0, f.changeUpdateErr
	}
	return 31, nil
}

func (f *fakeClient) UpdateTestCase(_ context.Context, id int, scenario string) error {
	f.testCaseUpdateCalls++
	f.testCaseUpdateInputs = append(f.testCaseUpdateInputs, dto.TestCase{ID: id, Scenario: scenario})
	return f.changeUpdateErr
}

func (f *fakeClient) UpdateTestCaseDone(_ context.Context, id int, done bool) error {
	f.testCaseDoneCalls++
	f.testCaseDoneIDs = append(f.testCaseDoneIDs, id)
	f.testCaseDoneUpdates = append(f.testCaseDoneUpdates, done)
	return f.changeUpdateErr
}

func (f *fakeClient) DeleteTestCase(_ context.Context, id int) error {
	f.testCaseDeleteCalls++
	f.testCaseDeleteIDs = append(f.testCaseDeleteIDs, id)
	return f.changeUpdateErr
}

func (f *fakeClient) DeleteChange(_ context.Context, id int) error {
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

func (f *fakeClient) ListEpics(_ context.Context, projectID int) ([]dto.Epic, error) {
	f.epicCalls++
	f.projectID = strconv.Itoa(projectID)
	if f.epicErr != nil {
		return nil, f.epicErr
	}
	if f.err != nil {
		return nil, f.err
	}
	if projectID <= 0 {
		return nil, errors.New("current project is required")
	}
	rows := make([]dto.Epic, 0, len(f.epics))
	for _, o := range f.epics {
		id, _ := strconv.Atoi(o.ID)
		rows = append(rows, dto.Epic{ID: id, ProjectID: projectID, Name: o.Label, Active: true})
	}
	return rows, nil
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

	load := got.selectorCommand(got.dropdown.source)
	got = applyMsg(got, load())

	assert.Equal(t, SelectProjectDropDown, got.state)
	assert.Equal(t, []dto.Option{{ID: "7", Label: "Project Seven"}}, got.dropdown.options)
}

func TestStartupSkipsProjectSelectionWhenProjectIDIsSaved(t *testing.T) {
	client := &fakeClient{gotProject: dto.Project{ID: 7, Name: "Project Seven"}}
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
	client := &fakeClient{
		phases: []dto.Option{{ID: "todo", Label: "todo", Color: "12"}},
		types:  []dto.Option{{ID: "feature", Label: "feature"}, {ID: "fix", Label: "fix"}},
	}
	m := newModelWithConfig(client, appConfig{ProjectID: 7})

	got := applyCommand(m, m.Init())

	assert.Equal(t, 1, client.phaseCalls)
	assert.Equal(t, 1, client.typeCalls)
	assert.True(t, got.optionCatalog.loaded)
	assert.Equal(t, client.phases, got.optionCatalog.phases)
	assert.Equal(t, client.types, got.optionCatalog.types)
}

func TestReplaceFileAtomicallyPreservesExistingFileAfterPartialWriteFailure(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "config.yaml")
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
	assert.Equal(t, "config.yaml", entries[0].Name())
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
	load := got.selectorCommand(got.dropdown.source)
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
	assert.Equal(t, lipgloss.Color("#454748"), styles.InputBackground)
	assert.Equal(t, styles.InputBackground, styles.Default.InputBand.GetBackground())
	assert.Equal(t, styles.Foreground, styles.Default.InputBand.GetForeground())
	assert.Equal(t, strings.Repeat("▄", 40), stripANSI(lines[0]))
	assert.Equal(t, strings.Repeat("▀", 40), stripANSI(lines[2]))
	assert.Contains(t, band, "Type / for commands")
	for i, line := range lines {
		visible := stripANSI(line)
		assert.Falsef(t, strings.TrimSpace(visible) == "" && len(visible) < 40, "blank input band line %d too short: %q", i, visible)
	}
	assert.True(t, strings.HasPrefix(stripANSI(lines[1]), " > Type / for commands"))

	m = m.setPromptValue("typed text")
	typedBand := m.inputBand(40)
	assert.NotContains(t, typedBand, "48;5;0")
	assert.NotContains(t, typedBand, "[40m")
	typedLine := stripANSI(strings.Split(typedBand, "\n")[1])
	assert.True(t, strings.HasPrefix(typedLine, " > typed text"))
	assert.Equal(t, styles.AccentPurple, m.input.FocusedStyle.Prompt.GetForeground())
	assert.Equal(t, styles.Foreground, m.input.FocusedStyle.Text.GetForeground())
	assert.Equal(t, styles.Foreground, m.input.FocusedStyle.CursorLine.GetForeground())
	assert.Equal(t, styles.Gray, m.input.FocusedStyle.Placeholder.GetForeground())
	assert.Equal(t, cursor.CursorStatic, m.input.Cursor.Mode())

	wideBand := m.inputBand(180)
	wideLines := strings.Split(wideBand, "\n")
	require.Len(t, wideLines, 3)
	assert.Equal(t, 180, lipgloss.Width(wideLines[0]))
	assert.Equal(t, 180, lipgloss.Width(wideLines[1]))
	assert.Equal(t, 180, lipgloss.Width(wideLines[2]))
}

func TestPromptTextareaGrowsForExplicitNewlines(t *testing.T) {
	m := NewModel()
	m = m.setPromptValue("first line\nsecond line\n")

	band := stripANSI(m.inputBand(40))
	lines := strings.Split(band, "\n")

	require.Len(t, lines, 5)
	assert.True(t, strings.HasPrefix(lines[1], " > first line"))
	assert.True(t, strings.HasPrefix(lines[2], " > second line"))
	assert.True(t, strings.HasPrefix(lines[3], " > "))
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
		if strings.HasPrefix(line, " > Type / for commands") {
			promptLine = i
			break
		}
	}
	require.NotZero(t, promptLine)
	require.Greater(t, len(lines), promptLine+3)
	assert.Equal(t, strings.Repeat("▀", 40), lines[promptLine+1])
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
	assert.Equal(t, "Edited\nName\n", got.input.Value())
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
			m.changeList.Detail = dto.ChangeView{
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
		createdProject: dto.Project{ID: 7},
		gotProject:     dto.Project{ID: 7, Name: "Line 1\nLine 2"},
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
				ID:          7,
				Name:        "Project Seven",
				ChangeCount: 3,
				CreatedAt:   projectTime("2026-06-29T08:15:00Z"),
				UpdatedAt:   projectTime("2026-06-29T10:45:00Z"),
			},
			{
				ID:          8,
				Name:        "Project Eight",
				ChangeCount: 0,
				CreatedAt:   time.Time{},
				UpdatedAt:   time.Time{},
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
		{ID: 1, Name: "demo1", ChangeCount: 2, CreatedAt: projectTime("2026-06-23T04:51:00Z"), UpdatedAt: projectTime("2026-06-23T04:51:00Z")},
		{ID: 350, Name: longName, ChangeCount: 0, CreatedAt: projectTime("2026-06-29T15:57:00Z"), UpdatedAt: projectTime("2026-06-29T15:57:00Z")},
		{ID: 351, Name: tooLongName, ChangeCount: 1, CreatedAt: projectTime("2026-06-29T15:58:00Z"), UpdatedAt: projectTime("2026-06-29T15:58:00Z")},
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
		{ID: 1, Name: "One"},
		{ID: 2, Name: "Two"},
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
		gotProject: dto.Project{ID: 8, Name: "Fresh Project Eight", ChangeCount: 5, CreatedAt: projectTime("2026-06-30T08:15:00Z"), UpdatedAt: projectTime("2026-06-30T11:45:00Z")},
	}
	m := newModelWithOptionCatalog(client)
	m.state = ProjectsListState
	m.currentProject = current
	m.projectList.Rows = []dto.Project{
		{ID: 7, Name: "Project Seven", ChangeCount: 3, CreatedAt: projectTime("2026-06-29T08:15:00Z"), UpdatedAt: projectTime("2026-06-29T10:45:00Z")},
		{ID: 8, Name: "Project Eight", ChangeCount: 4, CreatedAt: projectTime("2026-06-30T08:15:00Z"), UpdatedAt: projectTime("2026-06-30T10:45:00Z")},
	}
	m.projectList.Selected = 1

	got, cmd := sendKey(m, tea.KeyEnter)

	assert.Equal(t, ProjectDetailsState, got.state)
	assert.Equal(t, current, got.currentProject)
	assert.Equal(t, dto.Project{ID: 8, Name: "Project Eight", ChangeCount: 4, CreatedAt: projectTime("2026-06-30T08:15:00Z"), UpdatedAt: projectTime("2026-06-30T10:45:00Z")}, got.projectList.Detail)
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
		ID:          7,
		Name:        "Project Seven",
		ChangeCount: 3,
		CreatedAt:   projectTime("2026-06-29T13:04:59.999Z"),
		UpdatedAt:   time.Time{},
	}

	rawDetails := projects.DetailsView(m.projectList.Detail, 32)
	view := stripANSI(m.View())
	whiteValue := lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
	pinkValue := lipgloss.NewStyle().Foreground(lipgloss.Color("218"))
	timestampValue := lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
	createdValue := projects.FormatTimestamp(projectTime("2026-06-29T13:04:59.999Z"))

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
	m.projectList.Detail = dto.Project{ID: 7, Name: name}

	view := stripANSI(m.View())

	assert.Contains(t, view, "        Name: This project name is deliberately long and should wrap onto the next line")
	assert.Contains(t, view, "\n              without breaking any words in half")
	assert.NotContains(t, view, "witho\n")
}

func TestProjectDetailsPreservesExplicitNameNewlines(t *testing.T) {
	m := NewModelWithClient(&fakeClient{})
	m.state = ProjectDetailsState
	m.width = 120
	m.projectList.Detail = dto.Project{ID: 7, Name: "First line\nSecond line"}

	view := stripANSI(m.View())

	assert.Contains(t, view, "        Name: First line")
	assert.Contains(t, view, "\n              Second line")
}

func TestProjectPagesReloadOnArrival(t *testing.T) {
	client := &fakeClient{
		projectRows: []dto.Project{{ID: 7, Name: "Reloaded List Project"}},
		gotProject:  dto.Project{ID: 7, Name: "Reloaded Detail Project"},
	}

	m := newModelWithOptionCatalog(client)
	m.state = ProjectDetailsState
	m.projectList.Detail = dto.Project{ID: 7, Name: "Stale Detail Project"}
	got, cmd := sendCommand(m, "/return")
	require.NotNil(t, cmd)
	assert.Equal(t, ProjectsListState, got.state)
	assert.True(t, got.projectList.Loading)
	got = applyMsg(got, cmd())
	assert.Equal(t, 1, client.rowListCalls)
	assert.Equal(t, []dto.Project{{ID: 7, Name: "Reloaded List Project"}}, got.projectList.Rows)

	got.state = ProjectUpdateState
	got.projectList.Detail = dto.Project{ID: 7, Name: "Stale Detail Project"}
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
		createdProject: dto.Project{ID: 7},
		gotProject: dto.Project{
			ID:          7,
			Name:        "New Project",
			ChangeCount: 0,
			CreatedAt:   projectTime("2026-06-29T11:04:59Z"),
			UpdatedAt:   projectTime("2026-06-29T11:04:59Z"),
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
		updatedProject: dto.Project{ID: 7},
		gotProject: dto.Project{
			ID:          7,
			Name:        "Renamed Project",
			ChangeCount: 2,
			CreatedAt:   projectTime("2026-06-29T08:15:00Z"),
			UpdatedAt:   projectTime("2026-06-29T13:04:59Z"),
		},
	}
	m := newModelWithConfig(client, testAppConfig(appConfig{ProjectID: 99, ConfigPath: path}))
	m.state = ProjectDetailsState
	m.projectList.Detail = dto.Project{ID: 7, Name: "Old Project", ChangeCount: 2}

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
		{ID: 0, Name: "Zero"},
		{ID: -1, Name: "Negative"},
		{ID: 0, Name: "Missing ID"},
	}

	for _, project := range tests {
		t.Run(strconv.Itoa(project.ID), func(t *testing.T) {
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
		updatedProject: dto.Project{ID: 7},
		getErr:         errors.New("project not found"),
	}
	m = newModelWithOptionCatalog(client)
	m.state = ProjectUpdateState
	m.projectList.Detail = dto.Project{ID: 7, Name: "Old Project"}
	m.input.SetValue("Renamed Project")

	updated, cmd = m.executeCommandFrom(ProjectUpdateState, "/save")
	got = updated.(Model)
	require.NotNil(t, cmd)
	got = applyMsg(got, cmd())

	assert.Equal(t, ProjectDetailsState, got.state)
	assert.Equal(t, "Renamed Project", got.projectList.Detail.Name)
	assert.Contains(t, got.status, "saved project; refresh failed")
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
	m.projectList.Detail = dto.Project{ID: 7, Name: "Old Project"}
	m.input.SetValue("Renamed Project")

	got, _ = sendKey(m, tea.KeyEsc)
	got, _ = sendKey(got, tea.KeyEsc)

	assert.Equal(t, ProjectDetailsState, got.state)
	assert.Zero(t, client.createCalls)
	assert.Zero(t, client.updateCalls)
}

func TestChangesCommandLoadsAndRendersBackendRows(t *testing.T) {
	modified, err := time.Parse(time.RFC3339, "2026-06-29T10:45:00Z")
	require.NoError(t, err)
	expectedModified := modified.In(time.Local).Format("2006-01-02 15:04")
	client := &fakeClient{
		changeRows: []dto.ChangeView{
			{
				ID:          "11",
				Ref:         "3",
				RefSlug:     "003-change-three",
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
		gotChange: dto.ChangeView{
			ID:          "11",
			RefUUID:     "11111111-2222-4333-8444-555555555555",
			Ref:         "3",
			RefSlug:     "003-change-three",
			Title:       "Backend Change",
			ChangePhase: "backlog",
			ChangeTypes: []string{"feature", "test"},
			EpicID:      "5",
			EpicName:    "Epic Five",
			Spec:        "# Backend Change\n\nTypes: feature|test\n\nEpic: Epic Five\n\n## Problem Statement\nBody.",
			PR:          "Pull request summary.",
			PRUrl:       "https://github.com/divilla/project-manager/pull/107",

			Active:   true,
			Created:  "2026-06-29T08:15:00Z",
			Modified: "2026-06-29T10:45:00Z",
		},
	}
	m := newChangeTestModel(client)
	m.currentProject = dto.Option{ID: "7", Label: "Project Seven"}
	m.width = 120

	got, cmd := sendCommand(m, "/changes")
	require.Equal(t, ChangesListState, got.state)
	require.NotNil(t, cmd)
	assert.True(t, got.changeList.Loading)

	got = applyMsg(got, cmd())

	assert.Equal(t, []string{"7"}, client.changeListProjectIDs)
	view := stripANSI(got.View())
	assert.Contains(t, view, "/phase-filter")
	assert.Contains(t, view, "/types-filter")
	assert.Contains(t, view, "/epic-filter")
	assert.Contains(t, view, "/find-filter")
	assert.Contains(t, view, "#Ref")
	assert.Contains(t, view, "Phase")
	assert.Contains(t, view, "Types")
	assert.Contains(t, view, "Epic")
	assert.Contains(t, view, "Title")
	assert.Contains(t, view, "Don")
	assert.Contains(t, view, "Tot")
	assert.Contains(t, view, "%")
	assert.Contains(t, view, "Modified")
	assert.Contains(t, view, "3")
	assert.Contains(t, view, "backlog")
	assert.Contains(t, view, "Backend Change")
	assert.Contains(t, view, "feature|test")
	assert.Contains(t, view, "Epic Five")
	assert.Contains(t, view, "  2")
	assert.Contains(t, view, "  5")
	assert.Contains(t, view, " 40")
	assert.Contains(t, view, expectedModified)

	got, cmd = sendKey(got, tea.KeyEnter)
	require.NotNil(t, cmd)
	assert.Equal(t, ChangeDetailsState, got.state)
	got = applyMsg(got, cmd())

	assert.Equal(t, []int{11}, client.changeGetIDs)
	got.height = 50
	rawView := got.View()
	view = stripANSI(rawView)
	for _, value := range []string{"ChangeDetailsScreen", "ID │ 11", "Ref UUID │ 11111111-2222-4333-8444-555555555555", "Slug │ 003-change-three", "Epic │ Epic Five", "Phase │ backlog", "Types │ feature|test", "Title │ Backend Change", "[✓] brief", "[✓] spec", "[✓] pr", "PR URL │ https://github.com/divilla/project-manager/pull/107", "Completed │ ---=== 0/0 - 0% ===---", "Active │ ✅", "Modified: " + expectedModified} {
		assert.Contains(t, view, value)
	}
	assert.Less(t, strings.Index(view, "Title │"), strings.Index(view, "Docs │"))
	assert.Less(t, strings.Index(view, "Comments │"), strings.Index(view, "Ref UUID │"))
	assert.NotContains(t, view, "Ref │")
	assert.NotContains(t, view, "Epic ID │")
}

func TestChangesTableTruncatesEpicAndTitleAtMaxWidth(t *testing.T) {
	longEpic := strings.Repeat("E", 25)
	longTitle := strings.Repeat("T", 90)
	m := NewModelWithClient(&fakeClient{})
	m.state = ChangesListState
	m.width = 220
	m.changeList = m.changeList.WithRows([]dto.ChangeView{{
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
	view := stripANSI(changes.TableView(changes.Model{}.WithRows([]dto.ChangeView{{
		ID:          "1",
		Ref:         "1",
		ChangeTypes: []string{strings.Repeat("Y", 35)},
		EpicName:    strings.Repeat("E", 25),
		Title:       strings.Repeat("T", 90),
	}}), changes.Filters{}, 220, 1))
	lines := strings.Split(view, "\n")
	require.NotEmpty(t, lines)

	require.GreaterOrEqual(t, len(lines), 2)
	assert.Equal(t, 182, lipgloss.Width(lines[1]))
	assert.Contains(t, view, strings.Repeat("Y", 30))
	assert.NotContains(t, view, strings.Repeat("Y", 31))

	narrow := stripANSI(changes.TableView(changes.Model{}.WithRows([]dto.ChangeView{{
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
	model := changes.Model{}.WithRows([]dto.ChangeView{
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
	m := newChangeTestModel(client)
	m.state = ChangesListState
	m.width = 220
	m.changeList = m.changeList.WithRows([]dto.ChangeView{
		{ID: "1", Ref: "1", ChangePhase: "backlog", Title: "Backlog", Completed: 10},
		{ID: "2", Ref: "2", ChangePhase: "progress", Title: "Progress", Completed: 75},
	})

	raw := m.View()

	assert.Contains(t, raw, lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Render("progress  "))
}

func TestChangesTableKeyboardSelectionMatchesProjects(t *testing.T) {
	client := &fakeClient{
		gotChange: dto.ChangeView{ID: "2", Title: "Second Change"},
	}
	m := newChangeTestModel(client)
	m.state = ChangesListState
	m.changeList = m.changeList.WithRows([]dto.ChangeView{
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
	assert.Equal(t, client.gotChange.ID, got.changeList.Detail.ID)
	assert.Equal(t, client.gotChange.Title, got.changeList.Detail.Title)
}

func TestChangesTableIsBoxedAndScrollsSelectedRowIntoView(t *testing.T) {
	m := NewModelWithClient(&fakeClient{})
	m.state = ChangesListState
	m.height = 15
	m.width = 120
	m.changeList = m.changeList.WithRows([]dto.ChangeView{
		{ID: "1", Ref: "1", Title: "Change One"},
		{ID: "2", Ref: "2", Title: "Change Two"},
		{ID: "3", Ref: "3", Title: "Change Three"},
		{ID: "4", Ref: "4", Title: "Change Four"},
		{ID: "5", Ref: "5", Title: "Change Five"},
	})

	view := stripANSI(m.View())
	assert.Contains(t, view, "┌")
	assert.Contains(t, view, "└")
	assert.Contains(t, view, "Change Two")
	assert.Contains(t, view, "Change Three")
	assert.NotContains(t, view, "Change Four")
	assert.NotContains(t, view, "Change Five")
	assert.Contains(t, view, "Rows 1-3 of 5")
	lines := strings.Split(view, "\n")
	foundPromptBottom := false
	for index, line := range lines {
		if strings.Contains(line, "▀▀▀") {
			foundPromptBottom = true
			require.Less(t, index+1, len(lines))
			assert.Contains(t, lines[index+2], "Type to filter changes")
			break
		}
	}
	require.True(t, foundPromptBottom)

	got, _ := sendKey(m, tea.KeyDown)
	got, _ = sendKey(got, tea.KeyDown)
	got, _ = sendKey(got, tea.KeyDown)

	assert.Equal(t, 3, got.changeList.Selected)
	assert.Equal(t, 1, got.changeList.Offset)
	view = stripANSI(got.View())
	assert.Contains(t, view, "Change Two")
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
	m := newChangeTestModel(client)
	m.state = ChangesListState
	m.currentProject = dto.Option{}

	got, cmd := sendCommand(m, "/new-change")

	require.Nil(t, cmd)
	assert.Equal(t, ChangesListState, got.state)
	assert.Equal(t, "current project must be numeric", got.err)
	assert.Zero(t, client.changeCreateCalls)
}

func TestChangeUpdateStructuralValidationDoesNotFetchReferences(t *testing.T) {
	client := &fakeClient{err: errors.New("reference backend unavailable")}
	m := newChangeTestModel(client)
	m.currentProject = dto.Option{ID: "7", Label: "Project Seven"}
	m.state = ChangeUpdateState
	m.changeList.Detail = dto.ChangeView{
		ID:          "12",
		Title:       "Existing Change",
		Spec:        "# Existing Change\n\nTypes: feature\n\n## Problem Statement\nSpec.",
		ChangeTypes: []string{"feature"},
	}
	m.input.SetValue("   ")

	updated, cmd := m.executeCommandFrom(ChangeUpdateState, "/save")
	got := updated.(Model)
	require.NotNil(t, cmd)
	got = applyMsg(got, cmd())

	assert.Equal(t, ChangeUpdateState, got.state)
	assert.Equal(t, "spec is required", got.err)
	assert.Zero(t, client.typeCalls)
	assert.Zero(t, client.epicCalls)
	assert.Zero(t, client.changeTitleUpdateCalls)
	assert.Zero(t, client.changeSpecUpdateCalls)
	assert.Zero(t, client.changeTypesUpdateCalls)
	assert.Zero(t, client.changeEpicUpdateCalls)
}

func TestChangeUpdateSaveUpdatesChangedExtractedFieldsAndReloads(t *testing.T) {
	original := dto.ChangeView{
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
		gotChange: dto.ChangeView{ID: "12", Title: "New Change", Spec: spec, ChangeTypes: []string{"test"}, EpicID: "5", EpicName: "Epic Five"},
	}
	m := newChangeTestModel(client)
	m.currentProject = dto.Option{ID: "7", Label: "Project Seven"}
	m.state = ChangeUpdateState
	m.changeList.Detail = original
	m.input.SetValue(spec)

	updated, cmd := m.executeCommandFrom(ChangeUpdateState, "/save")
	got := updated.(Model)
	require.NotNil(t, cmd)
	got = applyMsg(got, cmd())

	assert.Empty(t, client.changeTitleUpdates) // Document edits do not rename the change.
	assert.Equal(t, []string{spec}, client.changeSpecUpdates)
	assert.Equal(t, [][]string{{"test"}}, client.changeTypesUpdates)
	assert.Zero(t, client.epicCalls)
	assert.Zero(t, client.changeEpicUpdateCalls)
	assert.Equal(t, []int{12}, client.changeGetIDs)
	assert.Equal(t, ChangeDetailsState, got.state)
}

func TestChangeUpdateSaveLeavesTypesUnchangedWhenMetadataIsOmitted(t *testing.T) {
	original := dto.ChangeView{
		ID:          "12",
		Title:       "Old Change",
		ChangeTypes: []string{},
	}
	spec := "# New Change\n\n## Problem Statement\nNew spec."
	client := &fakeClient{
		types:     []dto.Option{{ID: "feature", Label: "feature"}},
		gotChange: dto.ChangeView{ID: "12", Title: "New Change", Spec: spec, ChangeTypes: []string{}},
	}
	m := newChangeTestModel(client)
	m.currentProject = dto.Option{ID: "7", Label: "Project Seven"}
	m.state = ChangeUpdateState
	m.changeList.Detail = original
	m.input.SetValue(spec)

	updated, cmd := m.executeCommandFrom(ChangeUpdateState, "/save")
	got := updated.(Model)
	require.NotNil(t, cmd)
	got = applyMsg(got, cmd())

	assert.Empty(t, got.err)
	assert.Empty(t, client.changeTitleUpdates) // Document edits do not rename the change.
	assert.Equal(t, []string{spec}, client.changeSpecUpdates)
	assert.Zero(t, client.changeTypesUpdateCalls)
	assert.Equal(t, []int{12}, client.changeGetIDs)
	assert.Equal(t, ChangeDetailsState, got.state)
}

func TestChangeUpdateSaveTreatsBlankTypesAsEmpty(t *testing.T) {
	original := dto.ChangeView{
		ID:          "12",
		Title:       "Existing Change",
		Spec:        "# Existing Change\n\nTypes: feature\n\n## Problem Statement\nOld spec.",
		ChangeTypes: []string{"feature"},
	}
	spec := "# Existing Change\n\nTypes:\n\n## Problem Statement\nOld spec."
	client := &fakeClient{
		types:     []dto.Option{{ID: "feature", Label: "feature"}},
		gotChange: dto.ChangeView{ID: "12", Title: "Existing Change", Spec: spec, ChangeTypes: []string{}},
	}
	m := newChangeTestModel(client)
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
	original := dto.ChangeView{
		ID:          "12",
		Title:       "Old Change",
		Spec:        "# Old Change\n\nTypes: feature\n\n## Problem Statement\nOld spec.",
		ChangeTypes: []string{"feature"},
	}
	spec := "# Old Change\n\nTypes: feature\n\n## Problem Statement\nNew spec."
	client := &fakeClient{
		types:     []dto.Option{{ID: "feature", Label: "feature"}},
		gotChange: dto.ChangeView{ID: "12", Title: "Old Change", Spec: spec, ChangeTypes: []string{"feature"}},
	}
	m := newChangeTestModel(client)
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
	change := dto.ChangeView{
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
	assert.Equal(t, change.Spec, got.input.Value())
}

func TestChangeSpecEditDoesNotInjectBackendEpic(t *testing.T) {
	change := dto.ChangeView{
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
	change := dto.ChangeView{
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

	change := dto.ChangeView{
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

func beginSpecArtifactEditor(t *testing.T, change dto.ChangeView) Model {
	t.Helper()
	client := &fakeClient{gotChange: change}
	m := newChangeTestModel(client)
	m.state = ChangeDetailsState
	m = loadedChangeForTest(m, change, nil)
	loading, loadCmd := sendCommand(m, "/edit-spec")
	require.NotNil(t, loadCmd)
	return loading
}

func TestChangeUpdateDoesNotChangeBackendEpic(t *testing.T) {
	original := dto.ChangeView{
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
	m := newChangeTestModel(client)
	m.currentProject = dto.Option{ID: "7", Label: "Project Seven"}
	m.state = ChangeUpdateState
	m.changeList.Detail = original

	updated, cmd := m.saveChangeUpdateValue(changes.SpecMarkdown(original))
	got := updated.(Model)
	require.Nil(t, cmd)

	assert.Zero(t, client.changeTitleUpdateCalls)
	assert.Zero(t, client.changeSpecUpdateCalls)
	assert.Empty(t, client.changeTypesUpdates)
	assert.Zero(t, client.epicCalls)
	assert.Zero(t, client.changeEpicUpdateCalls)
	assert.Empty(t, client.changeGetIDs)
	assert.Equal(t, ChangeDetailsState, got.state)
}

func TestChangeFindFilterNarrowsVisibleRowsAndClearRestoresList(t *testing.T) {
	m := NewModelWithClient(&fakeClient{})
	m.state = ChangesListState
	m.changeList = m.changeList.WithRows([]dto.ChangeView{
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
	m.changeList = m.changeList.WithRows([]dto.ChangeView{
		{ID: "1", Ref: "3", Title: "Alpha", ChangePhase: "backlog", ChangeTypes: []string{"feature"}},
		{ID: "2", Ref: "4", Title: "Beta", ChangePhase: "done", ChangeTypes: []string{"test"}},
	})

	got, _ := sendCommand(m, "/find-filter")
	got.input.SetValue("3")
	got, _ = sendKey(got, tea.KeyEnter)

	assert.Equal(t, ChangesListState, got.state)
	view := stripANSI(got.View())
	assert.Contains(t, view, "3")
	assert.Contains(t, view, "Alpha")
	assert.NotContains(t, view, "Beta")
}

func TestChangeFindFilterClampsSelectedRow(t *testing.T) {
	client := &fakeClient{gotChange: dto.ChangeView{ID: "2", Title: "Beta"}}
	m := newChangeTestModel(client)
	m.state = ChangesListState
	m.changeList = m.changeList.WithRows([]dto.ChangeView{
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
		ID:          777777,
		Name:        "Very Long Project Name That Must Be Truncated",
		ChangeCount: 123,
		CreatedAt:   projectTime("2026-06-29T08:15:00Z"),
		UpdatedAt:   projectTime("2026-06-29T10:45:00Z"),
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
	m.changeDetailLoaded = true
	m.changeList = m.changeList.WithDetail(dto.ChangeView{ID: "11", Title: "Change", DocumentTypes: []string{"brief", "spec"}, TestCases: []dto.TestCase{{ID: 31}}, Comments: []dto.Document{{ID: 7}}})
	require.Equal(t, -1, m.changeList.DetailSelected)
	got, _ := sendKey(m, tea.KeyUp)
	require.Equal(t, -1, got.changeList.DetailSelected)
	for _, row := range changes.DetailRows(m.changeList.Detail) {
		if !row.Selectable {
			continue
		}
		got, _ = sendKey(got, tea.KeyDown)
		selected, ok := changes.DetailRowAtSelection(got.changeList.Detail, got.changeList.DetailSelected)
		require.True(t, ok)
		require.Equal(t, row, selected)
	}
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
	m.changeList = m.changeList.WithDetail(dto.ChangeView{
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

	got = selectDetailLabel(got, "Ref UUID")
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
	m.changeList = m.changeList.WithDetail(dto.ChangeView{ID: "11", Title: "Backend Change"})

	got, cmd := sendKeyMsg(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("ctrl+insert")})
	require.NotNil(t, cmd)
	got = applyMsg(got, cmd())

	assert.Equal(t, "copy failed", got.status)
	assert.Equal(t, "clipboard unavailable", got.err)
}

func TestChangeDetailsPhaseSelectionSavesAndReloads(t *testing.T) {
	client := &fakeClient{
		phases: []dto.Option{{ID: "stage", Label: "stage"}, {ID: "backlog", Label: "backlog"}},
		gotChange: dto.ChangeView{
			ID:          "12",
			Ref:         "3",
			Title:       "Backend Change",
			ChangePhase: "stage",
		},
	}
	m := newChangeTestModel(client)
	m.state = ChangeDetailsState
	m.changeList = m.changeList.WithDetail(dto.ChangeView{
		ID:          "12",
		Ref:         "3",
		Title:       "Backend Change",
		ChangePhase: "backlog",
	})
	m = selectDetailLabel(m, "Phase")

	got, cmd := sendKey(m, tea.KeyEnter)
	require.NotNil(t, cmd)
	assert.Equal(t, ChangeDetailsState, got.state)
	got = applyMsg(got, cmd())
	assert.Equal(t, 1, got.dropdown.highlighted)
	assert.Contains(t, stripANSI(got.dropdownView(80)), "    [ ] stage")
	assert.Contains(t, stripANSI(got.dropdownView(80)), "    [●] backlog")

	got, _ = sendKey(got, tea.KeyUp)
	got, cmd = sendKey(got, tea.KeyEnter)
	require.NotNil(t, cmd)
	assert.Equal(t, ChangeDetailsState, got.state)
	got = applyMsg(got, cmd())

	assert.Equal(t, []string{"stage"}, client.changePhaseUpdates)
	assert.Equal(t, []int{12}, client.changeGetIDs)
	assert.Equal(t, "stage", got.changeList.Detail.ChangePhase)
	assert.Equal(t, ChangeDetailsState, got.state)
	row, ok := changes.DetailRowAtSelection(got.changeList.Detail, got.changeList.DetailSelected)
	require.True(t, ok)
	assert.Equal(t, "Phase", row.Label)
}

func TestChangeDetailsFieldSelectionEscapeCancelsWithoutSaving(t *testing.T) {
	client := &fakeClient{
		phases: []dto.Option{{ID: "stage", Label: "stage"}},
	}
	m := newChangeTestModel(client)
	m.state = ChangeDetailsState
	m.changeList = m.changeList.WithDetail(dto.ChangeView{
		ID:          "12",
		Ref:         "3",
		Title:       "Backend Change",
		ChangePhase: "backlog",
	})
	m = selectDetailLabel(m, "Phase")

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
		gotChange: dto.ChangeView{
			ID:    "12",
			Ref:   "3",
			Title: "Backend Change",
		},
	}
	m := newChangeTestModel(client)
	m.currentProject = dto.Option{ID: "7", Label: "Project Seven"}
	m.state = ChangeDetailsState
	m.changeList = m.changeList.WithDetail(dto.ChangeView{
		ID:       "12",
		Ref:      "3",
		Title:    "Backend Change",
		EpicID:   "5",
		EpicName: "Epic Five",
	})
	m = selectDetailLabel(m, "Epic")

	got, cmd := sendKey(m, tea.KeyEnter)
	require.NotNil(t, cmd)
	assert.Equal(t, ChangeDetailsState, got.state)
	got = applyMsg(got, cmd())
	assert.Equal(t, 1, got.dropdown.highlighted)
	assert.Contains(t, stripANSI(got.dropdownView(80)), "    [ ] @none")
	assert.Contains(t, stripANSI(got.dropdownView(80)), "    [●] Epic Five #5")

	got, _ = sendKey(got, tea.KeyDown)
	got, cmd = sendKey(got, tea.KeyEnter)
	require.NotNil(t, cmd)
	got = applyMsg(got, cmd())

	require.Len(t, client.changeEpicUpdates, 1)
	assert.Nil(t, client.changeEpicUpdates[0])
	assert.Equal(t, []int{12}, client.changeGetIDs)
	assert.Equal(t, "null", got.changeList.Detail.EpicID)
	assert.Equal(t, ChangeDetailsState, got.state)
	row, ok := changes.DetailRowAtSelection(got.changeList.Detail, got.changeList.DetailSelected)
	require.True(t, ok)
	assert.Equal(t, "Epic", row.Label)
}

func TestChangeDetailsTitleSelectionOpensPromptAndSaves(t *testing.T) {
	client := &fakeClient{
		gotChange: dto.ChangeView{
			ID:    "12",
			Ref:   "3",
			Title: "New Title",
		},
	}
	m := newChangeTestModel(client)
	m.state = ChangeDetailsState
	m.changeList = m.changeList.WithDetail(dto.ChangeView{
		ID:    "12",
		Ref:   "3",
		Title: "Old Title",
	})
	m = selectDetailLabel(m, "Title")

	got, cmd := sendKey(m, tea.KeyEnter)
	require.Nil(t, cmd)
	assert.Equal(t, ChangeDetailsState, got.state)
	assert.Equal(t, detailEditTitle, got.detailEditField)
	assert.Equal(t, "Old Title", got.input.Value())
	assert.Equal(t, "Enter value (Ctrl+C/Esc cancel)", got.input.Placeholder)
	assert.Contains(t, got.View(), "ChangeDetailsScreen")
	assert.Contains(t, stripANSI(got.View()), "Title > Old Title")

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
	client := &fakeClient{gotChange: dto.ChangeView{ID: "12", Ref: "3", Title: "Old Title"}}
	m := newChangeTestModel(client)
	m.state = ChangeDetailsState
	m.changeList = m.changeList.WithDetail(dto.ChangeView{
		ID:    "12",
		Ref:   "3",
		Title: "Old Title",
	})
	m = selectDetailLabel(m, "Title")

	got, cmd := sendKey(m, tea.KeyEnter)
	require.Nil(t, cmd)
	require.Equal(t, ChangeDetailsState, got.state)
	assert.Equal(t, detailEditTitle, got.detailEditField)

	got = got.setPromptValue("/cancel")
	got, cmd = sendKey(got, tea.KeyEnter)

	require.Nil(t, cmd)
	assert.Equal(t, ChangeDetailsState, got.state)
	assert.Empty(t, got.detailEditField)
	assert.Empty(t, got.input.Value())
	assert.Zero(t, client.changeTitleUpdateCalls)
	assert.Empty(t, client.changeGetIDs)
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
			wantError: "pr is required",
		},
		{
			name:      "empty pr url",
			field:     detailEditPRUrl,
			value:     "   ",
			wantError: "PR URL requires a nonblank HTTP(S) URL; clearing is unsupported",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &fakeClient{}
			cmd := changeDetailTextUpdateCommand(client, ChangeDetailsState, dto.ChangeView{ID: "12"}, tt.field, tt.value)
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

func TestBriefSpecAndPRSavesApplyPresentTypesMetadata(t *testing.T) {
	fields := []detailEditField{detailEditBrief, detailEditSpec, detailEditPullRequest}
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
				client := &fakeClient{gotChange: dto.ChangeView{ID: "12"}}
				value := "# Artifact\n\n" + tt.line + "\n\nBody"
				msg := changeDetailTextUpdateCommand(client, ChangeDetailsState, dto.ChangeView{ID: "12"}, field, value)()
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
		gotChange: dto.ChangeView{
			ID:          "12",
			Ref:         "3",
			Title:       "Backend Change",
			ChangeTypes: []string{"docs", "feature"},
		},
	}
	m := newChangeTestModel(client)
	m.state = ChangeDetailsState
	m.changeList = m.changeList.WithDetail(dto.ChangeView{
		ID:          "12",
		Ref:         "3",
		Title:       "Backend Change",
		ChangeTypes: []string{"feature"},
	})
	m = selectDetailLabel(m, "Types")

	got, cmd := sendKey(m, tea.KeyEnter)
	require.NotNil(t, cmd)
	assert.Equal(t, ChangeDetailsState, got.state)
	got = applyMsg(got, cmd())
	assert.Equal(t, 1, got.dropdown.highlighted)
	view := stripANSI(got.dropdownView(80))
	assert.Less(t, strings.Index(view, "    [ ] docs"), strings.Index(view, "    [✓] feature"))
	assert.Less(t, strings.Index(view, "    [✓] feature"), strings.Index(view, "    [ ] test"))
	assert.Contains(t, view, "press <space> to change")
	selectedLine := ""
	for _, line := range strings.Split(got.dropdownView(80), "\n") {
		if strings.Contains(stripANSI(line), "    [✓] feature") {
			selectedLine = line
			break
		}
	}
	require.Equal(t, styles.Default.MenuSelected.Width(80).Render("    [✓] feature"), selectedLine)

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
	assert.Equal(t, 2, got.changeList.DetailSelected)
}

func TestChangeDetailsTypesSelectionRemovesSelectedType(t *testing.T) {
	client := &fakeClient{
		types: []dto.Option{
			{ID: "docs", Label: "docs"},
			{ID: "feature", Label: "feature"},
			{ID: "test", Label: "test"},
		},
		gotChange: dto.ChangeView{
			ID:          "12",
			Ref:         "3",
			Title:       "Backend Change",
			ChangeTypes: []string{"test"},
		},
	}
	m := newChangeTestModel(client)
	m.state = ChangeDetailsState
	m.changeList = m.changeList.WithDetail(dto.ChangeView{
		ID:          "12",
		Ref:         "3",
		Title:       "Backend Change",
		ChangeTypes: []string{"feature", "test"},
	})
	m = selectDetailLabel(m, "Types")

	got, cmd := sendKey(m, tea.KeyEnter)
	require.NotNil(t, cmd)
	got = applyMsg(got, cmd())
	assert.Equal(t, 1, got.dropdown.highlighted)
	assert.Contains(t, stripANSI(got.dropdownView(80)), "    [✓] feature")

	got, cmd = sendKey(got, tea.KeySpace)
	require.Nil(t, cmd)
	got, cmd = sendKey(got, tea.KeyEnter)
	require.NotNil(t, cmd)
	got = applyMsg(got, cmd())

	assert.Equal(t, [][]string{{"test"}}, client.changeTypesUpdates)
	assert.Equal(t, []int{12}, client.changeGetIDs)
	assert.Equal(t, []string{"test"}, got.changeList.Detail.ChangeTypes)
	assert.Equal(t, ChangeDetailsState, got.state)
	assert.Equal(t, 2, got.changeList.DetailSelected)
}

func TestChangeDetailsTypesSelectionEnterWithoutToggleReturnsWithoutSaving(t *testing.T) {
	client := &fakeClient{
		types: []dto.Option{
			{ID: "feature", Label: "feature"},
			{ID: "test", Label: "test"},
		},
	}
	m := newChangeTestModel(client)
	m.state = ChangeDetailsState
	m.changeList = m.changeList.WithDetail(dto.ChangeView{
		ID:          "12",
		Ref:         "3",
		Title:       "Backend Change",
		ChangeTypes: []string{"feature"},
	})
	m = selectDetailLabel(m, "Types")

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
		gotChange: dto.ChangeView{
			ID:     "12",
			Ref:    "3",
			Title:  "Backend Change",
			Active: false,
		},
	}
	m := newChangeTestModel(client)
	m.state = ChangeDetailsState
	m.changeList = m.changeList.WithDetail(dto.ChangeView{
		ID:     "12",
		Ref:    "3",
		Title:  "Backend Change",
		Active: true,
	})
	m = selectDetailLabel(m, "Active")

	got, cmd := sendRune(m, ' ')
	require.NotNil(t, cmd)
	assert.Equal(t, "saving", got.status)
	got = applyMsg(got, cmd())

	assert.Equal(t, []bool{false}, client.changeOpenUpdates)
	assert.Equal(t, []int{12}, client.changeGetIDs)
	assert.False(t, got.changeList.Detail.Active)
	assert.Equal(t, ChangeDetailsState, got.state)
	row, ok := changes.DetailRowAtSelection(got.changeList.Detail, got.changeList.DetailSelected)
	require.True(t, ok)
	assert.Equal(t, "Active", row.Label)
}

func TestChangeDetailsTestCaseSpaceTogglesAndReloads(t *testing.T) {
	client := &fakeClient{
		gotChange: dto.ChangeView{
			ID:    "12",
			Ref:   "3",
			Title: "Backend Change",
			TestCases: []dto.TestCase{
				{ID: 31, Scenario: "first", Done: true},
				{ID: 32, Scenario: "second", Done: true},
			},
		},
	}
	m := newChangeTestModel(client)
	m.state = ChangeDetailsState
	m.changeList = m.changeList.WithDetail(dto.ChangeView{
		ID:    "12",
		Ref:   "3",
		Title: "Backend Change",
		TestCases: []dto.TestCase{
			{ID: 31, Scenario: "first", Done: false},
			{ID: 32, Scenario: "second", Done: true},
		},
	})
	m = selectTestcase(m, "31")

	printer := &itemPrinter{}
	m.historyPrinter = printer
	got, cmd := sendRune(m, ' ')
	require.NotNil(t, cmd)
	assert.Equal(t, "saving test case", got.status)
	got = applyMsg(got, cmd())

	assert.Equal(t, []int{31}, client.testCaseDoneIDs)
	assert.Equal(t, []bool{true}, client.testCaseDoneUpdates)
	assert.Equal(t, []int{12}, client.changeGetIDs)
	assert.True(t, got.changeList.Detail.TestCases[0].Done)
	assert.Equal(t, ChangeDetailsState, got.state)
	row, ok := changes.DetailRowAtSelection(got.changeList.Detail, got.changeList.DetailSelected)
	require.True(t, ok)
	assert.Equal(t, "31", row.TestCaseID)
	assert.False(t, got.historyOpen)
	assert.Empty(t, printer.body)
	client.gotChange.TestCases = append([]dto.TestCase(nil), client.gotChange.TestCases...)
	client.gotChange.TestCases[0].Done = false
	got, cmd = sendKey(got, tea.KeySpace)
	require.NotNil(t, cmd)
	got = applyMsg(got, cmd())
	assert.Equal(t, []int{31, 31}, client.testCaseDoneIDs)
	assert.Equal(t, []bool{true, false}, client.testCaseDoneUpdates)
	assert.False(t, got.changeList.Detail.TestCases[0].Done)
	assert.True(t, got.changeList.Detail.TestCases[1].Done)
	assert.Equal(t, ChangeDetailsState, got.state)
	assert.False(t, got.historyOpen)
	assert.Empty(t, printer.body)
}

func TestChangeDetailsNewTestcaseCreatesAndRefreshes(t *testing.T) {
	client := &fakeClient{
		gotChange: dto.ChangeView{
			ID:    "12",
			Ref:   "3",
			Title: "Backend Change",
			TestCases: []dto.TestCase{
				{ID: 31, Scenario: "new scenario", Done: false, ChangeID: 12},
			},
		},
	}
	m := newChangeTestModel(client)
	m.state = ChangeDetailsState
	m.changeList = m.changeList.WithDetail(dto.ChangeView{ID: "12", Ref: "3", Title: "Backend Change"})

	got, cmd := sendCommand(m, "/new-testcase")
	require.Nil(t, cmd)
	assert.Equal(t, TestCaseCreateState, got.state)
	assert.Equal(t, "Write a Scenario", got.input.Placeholder)

	got.input.SetValue("new scenario")
	got, cmd = sendKey(got, tea.KeyEnter)
	require.NotNil(t, cmd)
	got = applyMsg(got, cmd())

	assert.Equal(t, ChangeDetailsState, got.state)
	assert.Equal(t, []dto.TestCase{{ChangeID: 12, Scenario: "new scenario"}}, client.testCaseCreateInputs)
	require.Len(t, got.changeList.Detail.TestCases, 1)
	assert.Equal(t, "new scenario", got.changeList.Detail.TestCases[0].Scenario)
}

func TestChangeDetailsTestcaseEnterEditsScenarioAndRefreshes(t *testing.T) {
	client := &fakeClient{
		gotChange: dto.ChangeView{
			ID:    "12",
			Ref:   "3",
			Title: "Backend Change",
			TestCases: []dto.TestCase{
				{ID: 31, Scenario: "updated scenario", Done: false, ChangeID: 12},
			},
		},
	}
	m := newChangeTestModel(client)
	m.state = ChangeDetailsState
	m.changeList = m.changeList.WithDetail(dto.ChangeView{
		ID:    "12",
		Ref:   "3",
		Title: "Backend Change",
		TestCases: []dto.TestCase{
			{ID: 31, Scenario: "old scenario", Done: false, ChangeID: 12},
		},
	})
	m = selectTestcase(m, "31")

	got, cmd := sendKey(m, tea.KeyEnter)
	require.NotNil(t, cmd)
	assert.Equal(t, TestCaseUpdateState, got.state)
	assert.Equal(t, "old scenario", got.input.Value())

	got.input.SetValue("updated scenario")
	got, cmd = sendKey(got, tea.KeyEnter)
	require.NotNil(t, cmd)
	got = applyMsg(got, cmd())

	assert.Equal(t, ChangeDetailsState, got.state)
	assert.Equal(t, []dto.TestCase{{ID: 31, Scenario: "updated scenario"}}, client.testCaseUpdateInputs)
	require.Len(t, got.changeList.Detail.TestCases, 1)
	assert.Equal(t, "updated scenario", got.changeList.Detail.TestCases[0].Scenario)
}

func TestChangeDetailsTestcaseDeleteConfirmsAndRefreshes(t *testing.T) {
	client := &fakeClient{
		gotChange: dto.ChangeView{ID: "12", Ref: "3", Title: "Backend Change"},
	}
	m := newChangeTestModel(client)
	m.state = ChangeDetailsState
	m.changeList = m.changeList.WithDetail(dto.ChangeView{
		ID:    "12",
		Ref:   "3",
		Title: "Backend Change",
		TestCases: []dto.TestCase{
			{ID: 31, Scenario: "old scenario", Done: false, ChangeID: 12},
		},
	})
	m = selectTestcase(m, "31")

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
	got, cmd := sendKey(changeList, tea.KeyCtrlN)
	require.NotNil(t, cmd)
	assert.Equal(t, ChangesListState, got.state)

	detail := NewModelWithClient(&fakeClient{})
	detail.state = ChangeDetailsState
	detail.changeList = detail.changeList.WithDetail(dto.ChangeView{ID: "12", Ref: "3", Title: "Backend Change"})
	detail.changeDetailLoaded = true
	got, cmd = sendKey(detail, tea.KeyCtrlN)
	require.Nil(t, cmd)
	assert.Equal(t, TestCaseCreateState, got.state)
	assert.Equal(t, "Write a Scenario", got.input.Placeholder)
}

func TestShortcutHelpRendersInFooter(t *testing.T) {
	m := NewModelWithClient(&fakeClient{})
	m.state = ChangeDetailsState
	m.width = 120
	m.changeList = m.changeList.WithDetail(dto.ChangeView{ID: "12", Ref: "3", Title: "Backend Change"})

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
	m.changeList = m.changeList.WithRows([]dto.ChangeView{{
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
	assert.Contains(t, lines[1], "/phase-filter")
	assert.Contains(t, lines[1], "backlog")
	assert.Contains(t, lines[1], "/types-filter")
	assert.Contains(t, lines[1], "feature")
	assert.Contains(t, lines[1], "/epic-filter")
	assert.Contains(t, lines[1], "Epic Five")
	assert.Contains(t, lines[1], "/find-filter")
	assert.Contains(t, lines[1], "backend")
	assert.Contains(t, lines[2], "┌")
	assert.Equal(t, lipgloss.Width(lines[2]), lipgloss.Width(lines[1]))
	footer := strings.Join(lines[max(0, len(lines)-4):], "\n")
	assert.Contains(t, footer, "<ctrl+n> new change")
	assert.Contains(t, footer, "</> command")
}

func TestChangesListFiltersRenderValuesPureWhite(t *testing.T) {
	m := NewModelWithClient(&fakeClient{})
	m.state = ChangesListState
	m.width = 160
	m.changeList = m.changeList.WithRows([]dto.ChangeView{{
		ID:          "12",
		Ref:         "3",
		Title:       "Backend Change",
		ChangePhase: "backlog",
	}})
	m.changesFilters.typ = dto.Option{ID: "feature", Label: "feature"}

	view := m.View()
	assert.Contains(t, view, styles.Default.Muted.Render("/types-filter "))
	assert.Contains(t, view, lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Render("feature"))
}

func TestChangeDetailsTableTruncatesLongSpecAndPullRequestRows(t *testing.T) {
	m := newChangeTestModel(&fakeClient{})
	m.state = ChangeDetailsState
	m.width, m.height = 120, 40
	m.changeList.Detail = dto.ChangeView{ID: "11", Title: "Change", RefSlug: "003-change", Brief: strings.Repeat("brief content\n", 40), DocumentTypes: []string{"brief", "spec", "pr"}, Documents: []dto.Document{{ID: 9, DocType: "spec", Body: strings.Repeat("spec content ", 180)}, {ID: 8, DocType: "pr", Body: "pull request start\npull request end"}}}
	var rendered strings.Builder
	for i := 0; i < 5; i++ {
		rendered.WriteString(stripANSI(m.View()))
		m, _ = sendKey(m, tea.KeyPgDown)
	}
	assert.Contains(t, rendered.String(), "[✓] spec")
	assert.Contains(t, rendered.String(), "[✓] pr")
	assert.NotContains(t, rendered.String(), "brief content")
	assert.NotContains(t, rendered.String(), "spec content")
	assert.NotContains(t, rendered.String(), "pull request start")
}

func TestP302EpicActionsRequireRealSelection(t *testing.T) {
	m := NewModel()
	m.state = EpicsListState

	next, cmd := sendKey(m, tea.KeyEnter)
	require.Nil(t, cmd)
	assert.Equal(t, EpicsListState, next.state)
	assert.Contains(t, next.err, "no epics selectable")
	for _, state := range []State{EpicCreateState, EpicUpdateState} {
		next, cmd := m.executeCommandFrom(state, "/save")
		require.Nil(t, cmd)
		assert.NotEqual(t, "save", next.(Model).status)
		assert.NotEmpty(t, next.(Model).err)
	}
	assert.Contains(t, commandsByState[EpicsListState], "/new-epic")
	assert.Contains(t, commandsByState[ProjectDetailsState], "/delete")
}

func TestCreateUpdateSaveCancelTransitions(t *testing.T) {
	tests := []struct {
		start   State
		command string
		want    State
	}{
		{start: ChangeDetailsState, command: "/edit-spec", want: ChangeDetailsState},
		{start: ChangeUpdateState, command: "/cancel", want: ChangeDetailsState},
		{start: ChangeDetailsState, command: "/new-testcase", want: TestCaseCreateState},
		{start: TestCaseUpdateState, command: "/cancel", want: ChangeDetailsState},
		{start: TestCaseDetailsState, command: "/edit", want: TestCaseUpdateState},
		{start: ProjectsListState, command: "/new-project", want: ProjectCreateState},
		{start: ProjectDetailsState, command: "/edit", want: ProjectUpdateState},
	}

	for _, tt := range tests {
		t.Run(string(tt.start)+tt.command, func(t *testing.T) {
			m := NewModel()
			m.state = tt.start
			if tt.start == ChangeDetailsState {
				m.changeDetailLoaded = true
			}

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
		"/new-comment",
		"/find",
		"/document", "/title", "/brief", "/pr-url", "/after-change", "/active", "/retry", "/help",
		"/new-testcase",
		"/phase",
		"/epic",
		"/types",
		"/edit-spec",
		"/delete",
		"/documents",
		"/return",
	}, commandsByState[ChangeDetailsState])
}

func TestChangesListCommandsAreExact(t *testing.T) {
	assert.Equal(t, []string{
		"/new-change",
		"/phase-filter",
		"/types-filter",
		"/epic-filter",
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
}

func TestSelectorDropdownsLoadAndReturn(t *testing.T) {
	client := &fakeClient{
		projects:  []dto.Option{{ID: "7", Label: "Project Seven"}},
		phases:    []dto.Option{{ID: "backlog", Label: "backlog"}},
		types:     []dto.Option{{ID: "feature", Label: "feature"}},
		epics:     []dto.Option{{ID: "3", Label: "Epic Three"}},
		gotChange: dto.ChangeView{ID: "12", Ref: "3", Title: "Backend Change"},
	}

	m := newModelWithOptionCatalog(client)
	got, cmd := sendCommand(m, "/select-project")
	require.Equal(t, SelectProjectDropDown, got.state)
	require.NotNil(t, cmd)
	got = applyMsg(got, cmd())
	got, cmd = sendKey(got, tea.KeyEnter)
	got = applyCommand(got, cmd)
	assert.Equal(t, MainState, got.state)
	assert.Equal(t, "7", got.currentProject.ID)

	got.state = ChangeDetailsState
	got.changeDetailLoaded = true
	got.changeList = got.changeList.WithDetail(dto.ChangeView{ID: "12", Ref: "3", Title: "Backend Change"})
	got, cmd = sendCommand(got, "/phase")
	got = applyMsg(got, cmd())
	got, cmd = sendKey(got, tea.KeyEnter)
	require.NotNil(t, cmd)
	got = applyMsg(got, cmd())
	assert.Equal(t, ChangeDetailsState, got.state)
	assert.Equal(t, 1, client.phaseCalls)
	assert.Equal(t, []string{"backlog"}, client.changePhaseUpdates)

	got, cmd = sendCommand(got, "/types")
	got = applyMsg(got, cmd())
	got, cmd = sendKey(got, tea.KeySpace)
	require.Nil(t, cmd)
	got, cmd = sendKey(got, tea.KeyEnter)
	require.NotNil(t, cmd)
	got = applyMsg(got, cmd())
	assert.Equal(t, ChangeDetailsState, got.state)
	assert.Equal(t, 1, client.typeCalls)
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
	got, saveCmd := sendKey(got, tea.KeyEnter)
	got = applyCommand(got, saveCmd)

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
	m.changeDetailLoaded = true

	got, cmd := sendCommand(m, "/phase")
	got = applyMsg(got, cmd())
	assert.Equal(t, ChangeDetailsState, got.state)
	assert.NotEmpty(t, got.err)

	got, _ = sendKey(got, tea.KeyEsc)
	assert.Equal(t, ChangeDetailsState, got.state)
}

func TestFilterSelectorsReturnToChangesList(t *testing.T) {
	client := &fakeClient{
		phases: []dto.Option{{ID: "done", Label: "done"}},
		epics:  []dto.Option{{ID: "1", Label: "Epic One"}},
		types:  []dto.Option{{ID: "test", Label: "test"}},
	}
	m := newChangeTestModel(client)
	m.state = ChangesListState
	m.currentProject = dto.Option{ID: "7", Label: "Project One"}

	got, cmd := sendCommand(m, "/phase-filter")
	require.NotNil(t, cmd)
	assert.Equal(t, ChangesListState, got.state)
	assert.Contains(t, got.View(), "ChangesListScreen")
	got = applyMsg(got, cmd())
	phaseDropdown := strings.Split(got.dropdownView(80), "\n")
	require.GreaterOrEqual(t, len(phaseDropdown), 3)
	assert.True(t, strings.HasPrefix(stripANSI(phaseDropdown[3]), "    [ ] done"))
	assert.Contains(t, stripANSI(got.dropdownView(80)), "[●] @clear")
	got, _ = sendKey(got, tea.KeyEnter)
	assert.Equal(t, ChangesListState, got.state)
	assert.Equal(t, "done", got.changesFilters.phase.ID)
	assert.Equal(t, "done", got.changesFilters.phase.Label)

	got, cmd = sendCommand(got, "/epic-filter")
	require.NotNil(t, cmd)
	assert.Equal(t, ChangesListState, got.state)
	assert.Contains(t, got.View(), "ChangesListScreen")
	got = applyMsg(got, cmd())
	assert.Contains(t, stripANSI(got.dropdownView(80)), "[●] @clear")
	assert.NotContains(t, stripANSI(got.dropdownView(80)), "@clear #")
	got, _ = sendKey(got, tea.KeyEnter)
	assert.Equal(t, ChangesListState, got.state)
	assert.Equal(t, "1", got.changesFilters.epic.ID)

	got, cmd = sendCommand(got, "/types-filter")
	require.NotNil(t, cmd)
	assert.Equal(t, ChangesListState, got.state)
	assert.Contains(t, got.View(), "ChangesListScreen")
	got = applyMsg(got, cmd())
	assert.Contains(t, stripANSI(got.dropdownView(80)), "Types Filter >")
	assert.Contains(t, stripANSI(got.dropdownView(80)), "[●] @clear")
	got, _ = sendKey(got, tea.KeyEnter)
	assert.Equal(t, ChangesListState, got.state)
	assert.Equal(t, "test", got.changesFilters.typ.ID)

	got, _ = sendCommand(got, "/find-filter")
	assert.Equal(t, FindInputState, got.state)
	got.input.SetValue("needle")
	got, _ = sendKey(got, tea.KeyEnter)
	assert.Equal(t, ChangesListState, got.state)
	assert.Equal(t, "needle", got.changesFilters.find)

	for _, tc := range []struct {
		command string
		field   filterField
	}{
		{command: "/phase-filter", field: filterPhase},
		{command: "/types-filter", field: filterTypes},
		{command: "/epic-filter", field: filterEpic},
	} {
		got, cmd = sendCommand(got, tc.command)
		require.NotNil(t, cmd)
		got = applyMsg(got, cmd())
		assert.Contains(t, stripANSI(got.dropdownView(80)), "[ ] @clear")
		got.dropdown.filter = "@clear"
		got, _ = sendKey(got, tea.KeyEnter)
		assert.Equal(t, ChangesListState, got.state)
		assert.Contains(t, got.status, "cleared "+string(tc.field)+" filter")
		assert.Equal(t, "needle", got.changesFilters.find)
		switch tc.field {
		case filterPhase:
			assert.Empty(t, got.changesFilters.phase.ID)
			assert.Equal(t, "test", got.changesFilters.typ.ID)
			assert.Equal(t, "1", got.changesFilters.epic.ID)
		case filterTypes:
			assert.Empty(t, got.changesFilters.typ.ID)
			assert.Equal(t, "1", got.changesFilters.epic.ID)
		case filterEpic:
			assert.Empty(t, got.changesFilters.epic.ID)
		}
	}

	got, _ = sendKey(got, tea.KeyCtrlF)
	assert.Equal(t, FindInputState, got.state)
	assert.Equal(t, ChangesListState, got.previousState)
	assert.Equal(t, "needle", got.input.Value())
	got, _ = sendKey(got, tea.KeyEnter)
	assert.Equal(t, ChangesListState, got.state)
	assert.Equal(t, "needle", got.changesFilters.find)

	got, _ = sendCommand(got, "/clear-filters")
	assert.Empty(t, got.changesFilters.phase.ID)
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
	m.changeList = m.changeList.WithDetail(dto.ChangeView{ID: "12", Title: "Backend Change"})

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
		changeRows: []dto.ChangeView{{ID: "13", Ref: "4", Title: "Remaining Change"}},
	}
	m := newChangeTestModel(client)
	m.currentProject = dto.Option{ID: "7", Label: "Project Seven"}
	m.state = ChangeDetailsState
	m.changeList = m.changeList.WithDetail(dto.ChangeView{ID: "12", Ref: "3", Title: "Backend Change"})

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
	assert.False(t, got.changeList.Loading)
	assert.Equal(t, []int{12}, client.changeDeleteIDs)

	require.Nil(t, reload)

	assert.Equal(t, ChangesListState, got.state)
	assert.Equal(t, []string{"7"}, client.changeListProjectIDs)
	require.Len(t, got.changeList.Rows, 1)
	assert.Equal(t, "13", got.changeList.Rows[0].ID)
	assert.Equal(t, "Remaining Change", got.changeList.Rows[0].Title)
}

func TestChangeDeleteRefreshFailureOffersListReloadPath(t *testing.T) {
	m := newChangeTestModel(&fakeClient{})
	m.state = ChangeDetailsState
	m.changeList = m.changeList.WithDetail(dto.ChangeView{ID: "12", Title: "Backend Change"})
	m.changeList.Operation = changes.Delete
	m.changeList.EntityID = 12
	m.changeList.ProjectID = 7
	next, cmd := m.applyChangeResult(changes.Result{ProjectID: 7, ID: 12, Operation: changes.Delete, Steps: []string{"deleted change"}, RefreshErr: errors.New("list unavailable")})
	require.Nil(t, cmd)
	m = next.(Model)
	require.Equal(t, ChangesListState, m.state)
	require.Contains(t, m.status, "return to Main and reopen /changes")
	require.NotContains(t, m.status, "/retry")
	m, _ = sendCommand(m, "/retry")
	require.Contains(t, m.err, "unknown command")
}

func TestChangeDeleteFailurePreservesDetail(t *testing.T) {
	client := &fakeClient{changeDeleteErr: errors.New("delete failed")}
	m := newChangeTestModel(client)
	m.currentProject = dto.Option{ID: "7", Label: "Project Seven"}
	m.state = ChangeDetailsState
	m.changeList = m.changeList.WithDetail(dto.ChangeView{ID: "12", Ref: "3", Title: "Backend Change"})

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
	assert.Equal(t, " > /"+strings.Repeat(" ", 76), stripANSI(lines[1]))
	assert.True(t, strings.HasPrefix(stripANSI(lines[3]), "    changes"))
	assert.Contains(t, stripANSI(lines[3]), "Browse changes")
	assert.NotContains(t, dropdown, "(1/10)")
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
	assert.NotContains(t, view, "flow_dir: /repo/.mch/default")
	assert.NotContains(t, view, "slug: brief")
	assert.NotContains(t, view, "prompt: prompts/change-brief.md")
	assert.NotContains(t, view, "entry: make brief-entry")
	assert.NotContains(t, view, "exec: make brief-exec")
	assert.NotContains(t, view, "exit: make brief-exit")
	assert.NotContains(t, view, "stage_modes:")
	assert.NotContains(t, view, "task_statuses:")
	assert.NotContains(t, view, "task_steps:")
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

			var got Model
			if state == BackendConfigFormState {
				got, _ = sendKey(m, tea.KeyCtrlG)
			} else {
				got, _ = sendRune(m, '/')
			}

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
	m.projectList.Rows = []dto.Project{{ID: 7, Name: "Project Seven"}}

	got, _ := sendRune(m, '/')

	assert.Equal(t, ProjectsListState, got.state)
	assert.Equal(t, dropdownCommand, got.dropdown.kind)
	view := stripANSI(got.View())
	assert.Contains(t, view, "ProjectsListScreen")
	assert.Contains(t, view, "    new-project")
	assert.Contains(t, view, "    help")
	assert.Contains(t, view, "    find")
	assert.Contains(t, view, "    return")
}

func TestCreateStatesUseContextSpecificNewCommandVocabulary(t *testing.T) {
	createCommands := map[State]string{
		ChangesListState:     "/new-change",
		ChangeDetailsState:   "/new-testcase",
		TestCaseDetailsState: "/new-testcase",
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
				assert.Contains(t, view, "/phase-filter")
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
	if cmd == nil {
		return m
	}
	msg := cmd()
	// Bubble Tea's ordered sequence message is private but has a slice of commands.
	value := reflect.ValueOf(msg)
	if value.Kind() == reflect.Slice && value.Type().Elem() == reflect.TypeOf(tea.Cmd(nil)) {
		for i := 0; i < value.Len(); i++ {
			m = applyCommand(m, value.Index(i).Interface().(tea.Cmd))
		}
		return m
	}
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

func projectTime(value string) time.Time {
	result, _ := time.Parse(time.RFC3339Nano, value)
	return result
}
func (f *fakeClient) DeleteProject(context.Context, int) error { return f.err }
func (f *fakeClient) GetProjectConfig(context.Context, int) (dto.ProjectConfig, error) {
	f.phaseCalls++
	f.typeCalls++
	cfg := dto.ProjectConfig{Slug: "test", ProjectDocs: []string{}, EpicDocs: []string{}, ChangeDocs: []string{}, ChangePhases: []string{}, ChangeColors: []string{}, ChangeTypes: []string{}}
	for _, o := range f.phases {
		cfg.ChangePhases = append(cfg.ChangePhases, o.ID)
		cfg.ChangeColors = append(cfg.ChangeColors, o.Color)
	}
	for _, o := range f.types {
		cfg.ChangeTypes = append(cfg.ChangeTypes, o.ID)
	}
	return cfg, f.err
}

func (f *fakeClient) GetEpic(_ context.Context, id int) (dto.Epic, error) {
	return dto.Epic{ID: id, ProjectID: 7, Name: "Epic", Active: true}, f.getErr
}

func (f *fakeClient) CreateEpic(_ context.Context, _ int, _ string) (int, error) {
	return 3, f.createErr
}
func (f *fakeClient) UpdateEpic(_ context.Context, _ int, _ string) error { return f.updateErr }
func (f *fakeClient) DeleteEpic(_ context.Context, _ int) error           { return f.err }

func fakeWire(v dto.ChangeView) dto.Change {
	id, _ := strconv.Atoi(v.ID)
	project, _ := strconv.Atoi(v.ProjectID)
	if project == 0 {
		project = 7
	}
	w := dto.Change{ID: id, ProjectID: project, RefUUID: v.RefUUID, Title: v.Title, ChangePhase: v.ChangePhase, ChangeTypes: v.ChangeTypes, Active: v.Active, DoneTC: v.Done, TotalTC: v.Total, Completed: v.Completed, PRUrl: v.PRUrl}
	w.CreatedAt, _ = time.Parse(time.RFC3339, v.Created)
	w.UpdatedAt, _ = time.Parse(time.RFC3339, v.Modified)
	if v.RefSlug != "" {
		w.RefSlug = &v.RefSlug
	} else if v.Ref != "" && v.Ref != "null" {
		refSlug := v.Ref + "-fixture"
		w.RefSlug = &refSlug
	}
	if v.EpicID != "" {
		n, _ := strconv.Atoi(v.EpicID)
		w.EpicID = &n
	}
	if v.EpicName != "" {
		w.EpicName = &v.EpicName
	}
	return w
}

func (f *fakeClient) UpdateChangeAfterChange(_ context.Context, _ int, _ *int) error {
	return f.changeUpdateErr
}

func (f *fakeClient) ActiveDocuments(_ context.Context, id int, _ string) ([]dto.Document, error) {
	return []dto.Document{{ID: 1, RefID: id, RefTable: "change", DocType: "brief", Body: f.gotChange.Brief}, {ID: 2, RefID: id, RefTable: "change", DocType: "spec", Body: f.gotChange.Spec}, {ID: 3, RefID: id, RefTable: "change", DocType: "pr", Body: f.gotChange.PR}}, nil
}

func (f *fakeClient) ListDocuments(_ context.Context, _ int, _ string) ([]dto.Document, error) {
	return []dto.Document{}, f.err
}

func (f *fakeClient) DocumentDetails(_ context.Context, id int) (dto.Document, error) {
	return dto.Document{ID: id, RefID: 12, RefTable: "change", DocType: "spec"}, f.err
}

func (f *fakeClient) InsertDocument(_ context.Context, in dto.DocumentInput) (int, error) {
	f.requestOrder = append(f.requestOrder, "doc/insert")
	f.changeArtifactAgentEdits = append(f.changeArtifactAgentEdits, in.AgentEdit)
	switch in.DocType {
	case "brief":
		f.changeBriefUpdateCalls++
		f.changeBriefUpdates = append(f.changeBriefUpdates, in.Body)
	case "spec":
		f.changeSpecUpdateCalls++
		f.changeSpecUpdates = append(f.changeSpecUpdates, in.Body)
	case "pr":
		f.changePRUpdateCalls++
		f.changePRUpdates = append(f.changePRUpdates, in.Body)
	}
	return 91, f.changeUpdateErr
}

func (f *fakeClient) ListTestCases(context.Context, int) ([]dto.TestCase, error) {
	return f.gotChange.TestCases, nil
}

// Drive the replacement feature operation in retained editor follow-up assertions.
func changeDetailTextUpdateCommand(client appClient, source State, view dto.ChangeView, field detailEditField, value string) tea.Cmd {
	m := NewModelWithClient(client)
	m.currentProject = dto.Option{ID: "7"}
	m.changeList.Detail = view
	m.state = source
	m.detailEditField = field
	m.optionCatalog.config = dto.ProjectConfig{ChangeDocs: []string{"brief", "spec", "pr"}, ChangeTypes: []string{"feature", "fix", "unsupported"}}
	next, cmd := m.saveChangeDetailTextValue(value)
	return func() tea.Msg {
		if cmd == nil {
			return changeSavedMsg{source: source, err: next.(Model).changeList.Err}
		}
		r := cmd().(changes.Result)
		return changeSavedMsg{source: source, change: r.Detail, err: r.Err, reloadErr: r.RefreshErr}
	}
}

func newChangeTestModel(client *fakeClient) Model {
	m := newModelWithOptionCatalog(client)
	m.currentProject = dto.Option{ID: "7"}
	m.changeList.ProjectID = 7
	m.changeDetailLoaded = true
	m.optionCatalog.config = dto.ProjectConfig{ChangePhases: []string{"backlog", "progress", "done", "stage"}, ChangeTypes: []string{"feature", "fix", "test", "docs", "unsupported"}, ChangeDocs: []string{"brief", "spec", "pr"}}
	return m
}

func loadedChangeForTest(m Model, view dto.ChangeView, err error) Model {
	id, _ := strconv.Atoi(view.ID)
	project, _ := strconv.Atoi(m.currentProject.ID)
	if project == 0 {
		project = 7
		m.currentProject.ID = "7"
	}
	if m.changeList.Operation != changes.Details {
		m.changeList.ProjectID = project
		m.changeList.EntityID = id
		m.changeList.Operation = changes.Details
	}
	return applyMsg(m, changes.Result{Generation: m.changeList.Generation, ProjectID: project, ID: id, Operation: changes.Details, Detail: view, Err: err})
}

func (f *fakeClient) ListComments(context.Context, int, string) ([]dto.Document, error) {
	return []dto.Document{}, nil
}

func (f *fakeClient) InsertComment(context.Context, int, string, string) (int, error) {
	return 99, f.err
}
func (f *fakeClient) UpdateComment(context.Context, int, string) error { return f.err }
func (f *fakeClient) DeleteDocument(context.Context, int) error        { return f.err }
func (f *fakeClient) ActivateDocument(context.Context, int) error      { return f.err }
func (f *fakeClient) UndeleteComment(context.Context, int) error       { return f.err }
func (f *fakeClient) ListInactiveChanges(ctx context.Context, id int) ([]dto.Change, error) {
	return f.ListChangeRows(ctx, id)
}
