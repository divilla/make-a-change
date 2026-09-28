package app

import (
	"cli/internal/agent"
	"cli/internal/dto"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/require"
)

func TestP801NewAndExistingWorkflowEntryAndIdentity(t *testing.T) {
	m := NewModelWithClient(&fakeClient{})
	m.currentProject.ID = "7"
	next, cmd := m.executeCommandFrom(MainState, "/brief-new")
	got := next.(Model)
	require.Equal(t, BriefState, got.state)
	require.Equal(t, 7, got.brief.ProjectID)
	require.Zero(t, got.brief.ChangeID)
	require.NotNil(t, cmd)
	got.brief = got.brief.Invalidate()
	got.state = ChangeDetailsState
	got.changeDetailLoaded = true
	got.changeList.Detail.ID = "12"
	next, cmd = got.executeCommandFrom(ChangeDetailsState, "/brief-clarify")
	got = next.(Model)
	require.Equal(t, BriefState, got.state)
	require.Equal(t, 12, got.brief.ChangeID)
	require.NotNil(t, cmd)
	got.brief = got.brief.Invalidate()
}

func TestP801ShellRoutesTypedWorkflowMessages(t *testing.T) {
	m := NewModelWithClient(&fakeClient{})
	m.currentProject.ID = "7"
	m.state = BriefState
	m.brief = agent.New(7)
	next, run := m.brief.Begin(m.ctx, agent.Preflight)
	m.brief = next
	require.NotNil(t, run)
	r := agent.Result{Generation: m.brief.Generation, Revision: m.brief.Revision, ProjectID: 7, Step: agent.Preflight, Config: validBriefConfig()}
	updated, _ := m.Update(r)
	got := updated.(Model)
	require.Equal(t, agent.Draft, got.brief.Phase)
	stale := r
	stale.Generation--
	updated, _ = got.Update(stale)
	require.Equal(t, got.brief.Revision, updated.(Model).brief.Revision)
	got.brief = got.brief.Invalidate()
}

func TestP805ReentryRejectsLateCommittedResultForSameScope(t *testing.T) {
	for _, existing := range []bool{false, true} {
		name := "new"
		if existing {
			name = "existing"
		}
		t.Run(name, func(t *testing.T) {
			m := NewModelWithClient(&fakeClient{})
			m.currentProject.ID = "7"
			m.state = MainState
			if existing {
				m.state = ChangeDetailsState
				m.changeDetailLoaded = true
				m.changeList.Detail.ID = "21"
			}
			enter := func(m Model) Model {
				next, cmd := m.openBrief(!existing)
				require.NotNil(t, cmd)
				m = next.(Model)
				preflight := agent.Result{Generation: m.brief.Generation, Revision: m.brief.Revision, ProjectID: 7, ChangeID: m.brief.ChangeID, Step: agent.Preflight, Config: validBriefConfig()}
				if existing {
					preflight.Detail = dto.Change{ID: 21, ProjectID: 7}
					preflight.Documents = []dto.Document{{ID: 31, RefID: 21, RefTable: "change", DocType: "brief", Body: "Before", Current: true}}
				}
				var ok bool
				m.brief, _, ok = m.brief.Apply(preflight)
				require.True(t, ok)
				if !existing {
					m.brief = m.brief.EditIdentity("Title", "")
				}
				m.brief = m.brief.EditBrief("After")
				m.brief, _ = m.brief.Begin(m.ctx, agent.Write)
				require.Equal(t, agent.Write, m.brief.Pending)
				return m
			}
			m = enter(m)
			late := agent.Result{Generation: m.brief.Generation, Revision: m.brief.Revision, ProjectID: 7, ChangeID: m.brief.ChangeID, Step: agent.Write, ID: 99}
			next, _ := m.briefCommand("/return")
			m = next.(Model)
			if existing {
				m.changeDetailLoaded = true
			}
			m = enter(m)
			require.Greater(t, m.brief.Generation, late.Generation)
			next, _ = m.Update(late)
			m = next.(Model)
			require.True(t, m.brief.Busy)
			require.Equal(t, agent.Write, m.brief.Pending)
			require.Zero(t, m.brief.CommittedID)
			if existing {
				require.Equal(t, 31, m.brief.DocumentID)
			} else {
				require.Zero(t, m.brief.ChangeID)
			}
			m.brief = m.brief.Invalidate()
		})
	}
}

func TestP803ShellShowsLiveProgressAndDropsStaleProgress(t *testing.T) {
	m := NewModelWithClient(&fakeClient{})
	m.state = BriefState
	m.currentProject.ID = "7"
	m.brief = agent.Existing(7, 21)
	m.brief.Busy = true
	m.brief.Pending = agent.Run
	progress := make(chan string, 1)
	release := make(chan struct{})
	updated, cmd := m.launchBriefOperation(func(agent.API, agent.Runner, string) agent.Result {
		progress <- "agent stderr: checking\x1b[31m\ncontext"
		<-release
		return agent.Result{ProjectID: 7, ChangeID: 21, Generation: m.brief.Generation, Revision: m.brief.Revision, Step: agent.Run}
	}, progress)
	m = updated.(Model)
	messages := make(chan tea.Msg, 1)
	go func() { messages <- cmd() }()
	var msg tea.Msg
	select {
	case msg = <-messages:
	case <-time.After(time.Second):
		close(release)
		t.Fatal("progress was not delivered while runner was active")
	}
	updated, next := m.Update(msg)
	m = updated.(Model)
	require.Contains(t, m.status, `checking\x1b[31m\ncontext`)
	require.NotContains(t, m.status, "\x1b")
	require.NotNil(t, next)
	close(release)
	require.IsType(t, agent.Result{}, next())
	stale := briefProgressMsg{operation: m.briefOperation, text: "late"}
	m.brief = m.brief.Invalidate()
	updated, next = m.Update(stale)
	require.Nil(t, next)
	require.NotContains(t, updated.(Model).status, "late")
}

func TestP805ConfirmAfterCommittedBriefRetriesRefresh(t *testing.T) {
	m := NewModelWithClient(&fakeClient{})
	m.state = BriefState
	m.brief = agent.Existing(7, 21).EditBrief("After")
	m.brief.BackendBrief = "Before"
	m.brief.CommittedID = 32
	m.brief.Phase = agent.Failed

	next, cmd := m.briefCommand("/confirm")
	got := next.(Model)
	require.NotNil(t, cmd)
	require.Equal(t, agent.Refresh, got.brief.Pending)
	require.Equal(t, 32, got.brief.CommittedID)
	got.brief = got.brief.Invalidate()
}

func TestBriefCommandDropdownRoutesSelectionAndEscape(t *testing.T) {
	m := NewModel()
	m.state = BriefState
	m.brief = agent.New(7)
	m.briefField = "brief"

	got, _ := sendRune(m, '/')
	require.Equal(t, dropdownCommand, got.dropdown.kind)
	got, _ = sendKey(got, tea.KeyDown)
	require.Equal(t, 1, got.dropdown.highlighted)
	got, _ = sendKey(got, tea.KeyEnter)
	require.Equal(t, BriefState, got.state)
	require.Empty(t, got.dropdown.kind)
	require.Equal(t, "uuid", got.briefField)

	got, _ = sendRune(got, '/')
	got, _ = sendKey(got, tea.KeyEsc)
	require.Equal(t, BriefState, got.state)
	require.Empty(t, got.dropdown.kind)
	require.Equal(t, agent.Draft, got.brief.Phase)
}

func TestBriefDirectInputPreservesMoreThanDefaultLimit(t *testing.T) {
	m := NewModel()
	m.state = BriefState
	m.brief = agent.New(7)
	m.briefField = "brief"
	brief := strings.Repeat("a", defaultPromptCharLimit+25)

	for _, r := range brief {
		m, _ = sendRune(m, r)
	}
	require.Zero(t, m.input.CharLimit)
	require.Equal(t, brief, m.input.Value())
	m, _ = sendKey(m, tea.KeyEnter)
	require.Equal(t, brief, m.brief.Draft)
	require.Equal(t, brief, m.brief.Original)
}

func TestP802PendingBriefSeedsEditorAndBecomesOriginal(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("TMPDIR", dir)
	m := NewModel()
	m.state = BriefState
	m.brief = agent.New(7)
	m.briefField = "brief"
	pending := "# Brief\n```sh\nmake test\n```\n"
	m = m.setPromptValue(pending)

	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlE})
	require.NotNil(t, cmd)
	m = next.(Model)
	files, err := filepath.Glob(filepath.Join(dir, "mch-project-*.md"))
	require.NoError(t, err)
	require.Len(t, files, 1)
	seed, err := os.ReadFile(files[0])
	require.NoError(t, err)
	require.Equal(t, pending, string(seed))

	edited := "\t" + pending
	updated, _ := m.Update(editorFinishedMsg{source: BriefState, original: pending, content: edited})
	m = updated.(Model)
	require.Equal(t, edited, m.brief.Draft)
	require.Equal(t, edited, m.brief.Original)
}

func TestBriefEditorUsesSavedDraftWhenInputIsEmpty(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("TMPDIR", dir)
	m := NewModel()
	m.state = BriefState
	m.brief = agent.New(7).EditBrief("saved draft")
	m.briefField = "brief"

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlE})
	require.NotNil(t, cmd)
	files, err := filepath.Glob(filepath.Join(dir, "mch-project-*.md"))
	require.NoError(t, err)
	require.Len(t, files, 1)
	seed, err := os.ReadFile(files[0])
	require.NoError(t, err)
	require.Equal(t, "saved draft", string(seed))
}

func validBriefConfig() dto.ProjectConfig {
	return dto.ProjectConfig{ChangeDocs: []string{"brief", "spec"}, ChangePhases: []string{"backlog"}}
}

func TestP807ScenarioManifestIncludesBriefAndPTY(t *testing.T) {
	data, err := os.ReadFile("../../scripts/terminal-scenarios.json")
	require.NoError(t, err)
	var manifest struct {
		Program struct {
			Tests []string `json:"tests"`
		} `json:"program"`
		PTY struct {
			Tests []string `json:"tests"`
		} `json:"pty"`
	}
	require.NoError(t, json.Unmarshal(data, &manifest))
	require.Contains(t, manifest.Program.Tests, "TestCLIProgramBriefNewAndExistingPersistence")
	require.Contains(t, manifest.Program.Tests, "TestCLIProgramBriefFailuresAndStaleCancellation")
	require.Contains(t, manifest.Program.Tests, "TestCLIProgramBriefStaleCancellation")
	require.Contains(t, manifest.PTY.Tests, "TestShellNavigationEditorAndScrolling")
}
