package app

import (
	"cli/internal/documents"
	"cli/internal/dto"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/require"
)

type appDocs struct {
	fakeClient
	config             dto.ProjectConfig
	configErr          error
	configIDs          []int
	listOwner          []string
	rows               []dto.Document
	current            []dto.Document
	detail             dto.Document
	insert             []dto.DocumentInput
	listErr, insertErr error
}

func (a *appDocs) GetProjectConfig(_ context.Context, id int) (dto.ProjectConfig, error) {
	a.configIDs = append(a.configIDs, id)
	return a.config, a.configErr
}

func (a *appDocs) ListDocuments(_ context.Context, id int, table string) ([]dto.Document, error) {
	a.listOwner = append(a.listOwner, table+":"+stringID(id))
	return a.rows, a.listErr
}

func (a *appDocs) CurrentDocuments(context.Context, int, string) ([]dto.Document, error) {
	return a.current, nil
}

func (a *appDocs) DocumentDetails(context.Context, int) (dto.Document, error) { return a.detail, nil }

func (a *appDocs) InsertDocument(_ context.Context, in dto.DocumentInput) (int, error) {
	a.insert = append(a.insert, in)
	return 91, a.insertErr
}
func stringID(v int) string { return strconv.Itoa(v) }
func TestP602ProjectEpicChangeNavigationAndScope(t *testing.T) {
	for _, tt := range []struct {
		source         State
		project, owner int
		table          string
		types          []string
	}{
		{ProjectDetailsState, 8, 8, "project", []string{"project-note"}},
		{EpicDetailsState, 7, 12, "epic", []string{"epic-note"}},
		{ChangeDetailsState, 7, 31, "change", []string{"change-note"}},
	} {
		t.Run(tt.table, func(t *testing.T) {
			a := &appDocs{config: dto.ProjectConfig{ProjectDocs: []string{"project-note"}, EpicDocs: []string{"epic-note"}, ChangeDocs: []string{"change-note"}}, rows: []dto.Document{}, current: []dto.Document{}}
			m := NewModelWithClient(a)
			m.state = tt.source
			m.currentProject.ID = "7"
			m.projectList.Detail = dto.Project{ID: 8}
			m.projectList.DetailLoaded = true
			m.epicList.Detail = dto.Epic{ID: 12, ProjectID: 7}
			m.epicList.DetailLoaded = true
			m.changeList.Detail = dto.ChangeView{ID: "31", ProjectID: "7"}
			m.changeDetailLoaded = true
			got, cmd := sendCommand(m, "/documents")
			require.Equal(t, DocumentState, got.state)
			require.NotNil(t, cmd)
			got = applyMsg(got, cmd())
			require.Equal(t, tt.project, got.document.ProjectID)
			require.Equal(t, tt.owner, got.document.OwnerID)
			require.Equal(t, tt.table, got.document.OwnerTable)
			require.Equal(t, tt.types, got.document.Types)
			require.Equal(t, []int{tt.project}, a.configIDs)
			require.Equal(t, []string{tt.table + ":" + stringID(tt.owner)}, a.listOwner)
			require.Contains(t, got.View(), "Documents: "+tt.table)
			got, _ = sendCommand(got, "/return")
			require.Equal(t, tt.source, got.state)
		})
	}
}

func TestP602StaleOwnerSelectionCannotAct(t *testing.T) {
	a := &appDocs{config: dto.ProjectConfig{EpicDocs: []string{"notes"}}, rows: []dto.Document{}, current: []dto.Document{}}
	m := NewModelWithClient(a)
	m.state = EpicDetailsState
	m.currentProject.ID = "7"
	m.epicList.Detail = dto.Epic{ID: 12, ProjectID: 7}
	m.epicList.DetailLoaded = true
	got, cmd := sendCommand(m, "/documents")
	got.epicList.Detail.ID = 13
	got = applyMsg(got, cmd())
	require.False(t, got.document.Loaded)
	got, _ = sendCommand(got, "/new-document")
	require.ErrorContains(t, errors.New(got.err), "owner selection changed")
	require.Empty(t, a.insert)
}

func TestP602DocumentsRequireLoadedOwnerDetails(t *testing.T) {
	for _, source := range []State{ProjectDetailsState, EpicDetailsState} {
		t.Run(string(source), func(t *testing.T) {
			a := &appDocs{}
			m := NewModelWithClient(a)
			m.state = source
			m.currentProject.ID = "7"
			m.projectList.Detail = dto.Project{ID: 7}
			m.epicList.Detail = dto.Epic{ID: 12, ProjectID: 7}
			got, cmd := sendCommand(m, "/documents")
			require.Nil(t, cmd)
			require.Equal(t, source, got.state)
			require.Contains(t, got.err, "load ")
			require.Empty(t, a.configIDs)

			m.projectList.DetailLoaded = true
			m.epicList.DetailLoaded = true
			got, cmd = sendCommand(m, "/documents")
			require.NotNil(t, cmd)
			require.Equal(t, DocumentState, got.state)
		})
	}
}

func TestP602HistorySelectionVisibleWithCatalogError(t *testing.T) {
	a := &appDocs{}
	m := NewModelWithClient(a)
	m.state = DocumentState
	m.documentReturn = ProjectDetailsState
	m.projectList.Detail = dto.Project{ID: 7}
	m.projectList.DetailLoaded = true
	m.document = documents.Model{ProjectID: 7, OwnerID: 7, OwnerTable: "project", Loaded: true, CatalogErr: errors.New("catalog offline")}
	for id := 30; id >= 1; id-- {
		m.document.Rows = append(m.document.Rows, dto.Document{ID: id, RefID: 7, RefTable: "project"})
	}
	m.width, m.height = 80, 15
	for i := 0; i < 20; i++ {
		var handled bool
		m, _, handled = m.documentKey("down", tea.KeyMsg{Type: tea.KeyDown})
		require.True(t, handled)
	}
	height := m.documentViewportHeight()
	require.Positive(t, height)
	selected := m.document.Rows[m.document.Selected]
	require.Contains(t, documents.View(m.document, 80, height), "> #"+strconv.Itoa(selected.ID))
	a.detail = selected
	m, cmd, handled := m.documentKey("enter", tea.KeyMsg{Type: tea.KeyEnter})
	require.True(t, handled)
	require.NotNil(t, cmd)
	m = applyMsg(m, cmd())
	require.Equal(t, selected.ID, m.document.Detail.ID)
}

func TestP602SelectedVersionVisibleAfterDetailsAndRefresh(t *testing.T) {
	a := &appDocs{config: dto.ProjectConfig{ProjectDocs: []string{"notes"}}}
	m := NewModelWithClient(a)
	m.state = DocumentState
	m.documentReturn = ProjectDetailsState
	m.projectList.Detail = dto.Project{ID: 7}
	m.projectList.DetailLoaded = true
	m.document = documents.Model{ProjectID: 7, OwnerID: 7, OwnerTable: "project", Loaded: true}
	for id := 30; id >= 1; id-- {
		m.document.Rows = append(m.document.Rows, dto.Document{ID: id, RefID: 7, RefTable: "project"})
	}
	m.width, m.height = 80, 15
	for i := 0; i < 20; i++ {
		m, _, _ = m.documentKey("down", tea.KeyMsg{Type: tea.KeyDown})
	}
	selected := m.document.Rows[m.document.Selected]
	height := m.documentViewportHeight()
	require.Contains(t, documents.View(m.document, 80, height), "> #"+strconv.Itoa(selected.ID))
	rowLine := m.document.SelectedRowLine(80)
	a.detail = selected
	a.detail.Body = strings.Repeat("detail line\n", 60)
	m, cmd, handled := m.documentKey("enter", tea.KeyMsg{Type: tea.KeyEnter})
	require.True(t, handled)
	m = applyMsg(m, cmd())
	m.document = m.document.Scroll(50, 80, height)
	require.Greater(t, m.document.Offset, rowLine)
	m, cmd = sendCommand(m, "/return")
	require.Nil(t, cmd)
	require.Contains(t, documents.View(m.document, 80, m.documentViewportHeight()), "> #"+strconv.Itoa(selected.ID))

	a.rows = append([]dto.Document{{ID: 31, RefID: 7, RefTable: "project"}}, m.document.Rows...)
	m, cmd = sendCommand(m, "/retry")
	require.NotNil(t, cmd)
	m = applyMsg(m, cmd())
	require.Equal(t, selected.ID, m.document.Rows[m.document.Selected].ID)
	require.Contains(t, documents.View(m.document, 80, m.documentViewportHeight()), "> #"+strconv.Itoa(selected.ID))
}

func TestP602CatalogRetryRestoresDocumentInsertion(t *testing.T) {
	a := &appDocs{configErr: errors.New("catalog offline"), config: dto.ProjectConfig{ProjectDocs: []string{"notes"}}, rows: []dto.Document{}, current: []dto.Document{}}
	m := NewModelWithClient(a)
	m.state = ProjectDetailsState
	m.projectList.Detail = dto.Project{ID: 7}
	m.projectList.DetailLoaded = true
	got, cmd := sendCommand(m, "/documents")
	got = applyMsg(got, cmd())
	require.True(t, got.document.Loaded)
	require.Empty(t, got.document.Types)
	require.Contains(t, got.View(), "Catalog error")
	a.configErr = nil
	got, cmd = sendCommand(got, "/retry")
	require.NotNil(t, cmd)
	got = applyMsg(got, cmd())
	require.Equal(t, []string{"notes"}, got.document.Types)
	require.NoError(t, got.document.CatalogErr)
	got, _ = sendCommand(got, "/new-document")
	require.True(t, got.documentForm)
	require.Equal(t, []int{7, 7}, a.configIDs)
	require.Empty(t, a.insert)
}

func TestP603ShellRoutesDocumentMessagesWithoutMutationLogic(t *testing.T) {
	a := &appDocs{config: dto.ProjectConfig{ProjectDocs: []string{"notes"}}, rows: []dto.Document{}, current: []dto.Document{}}
	m := NewModelWithClient(a)
	m.state = ProjectDetailsState
	m.projectList.Detail = dto.Project{ID: 7}
	m.projectList.DetailLoaded = true
	got, cmd := sendCommand(m, "/documents")
	got = applyMsg(got, cmd())
	got, _ = sendCommand(got, "/new-document")
	require.True(t, got.documentForm)
	require.Contains(t, got.View(), "agent_edit=false")
	require.Contains(t, got.View(), "<esc> cancel draft")
	raw := " /cancel\nbytes\t\x1b[0m\n"
	got = applyMsg(got, editorFinishedMsg{source: DocumentState, content: raw})
	require.Equal(t, raw, got.promptValue())
	require.NotContains(t, got.View(), "\x1b[0m")
	got, cmd = sendKey(got, tea.KeyEnter)
	require.NotNil(t, cmd)
	require.Len(t, a.insert, 0)
	got = applyMsg(got, cmd())
	require.Len(t, a.insert, 1)
	require.Equal(t, dto.DocumentInput{RefID: 7, RefTable: "project", DocType: "notes", Body: raw, AgentEdit: false}, a.insert[0])
	require.False(t, got.documentForm)
	require.Contains(t, got.status, "saved document #91")
}

func TestP604ShellCommittedDocumentRetryAndDraftPreservation(t *testing.T) {
	a := &appDocs{config: dto.ProjectConfig{ProjectDocs: []string{"notes"}}, rows: []dto.Document{}, current: []dto.Document{}, insertErr: errors.New("write refused")}
	m := NewModelWithClient(a)
	m.state = ProjectDetailsState
	m.projectList.Detail = dto.Project{ID: 7}
	m.projectList.DetailLoaded = true
	got, cmd := sendCommand(m, "/documents")
	got = applyMsg(got, cmd())
	got, _ = sendCommand(got, "/new-document")
	got = got.setPromptValue("draft")
	got, cmd = sendKey(got, tea.KeyEnter)
	got = applyMsg(got, cmd())
	require.True(t, got.documentForm)
	require.Equal(t, "draft", got.promptValue())
	require.Equal(t, "draft", got.document.DraftBody)
	a.insertErr = nil
	a.listErr = errors.New("offline")
	got, cmd = sendKey(got, tea.KeyEnter)
	updated, follow := got.Update(cmd())
	got = updated.(Model)
	require.False(t, got.documentForm)
	require.Contains(t, got.status, "saved document #91; refreshing")
	require.NotNil(t, follow)
	got = applyMsg(got, follow())
	require.Contains(t, got.status, "saved document #91; refresh failed")
	require.Len(t, a.insert, 2)
	got, cmd = sendCommand(got, "/retry")
	got = applyMsg(got, cmd())
	require.Len(t, a.insert, 2)
	require.Contains(t, got.status, "saved document #91")
}

func TestP605ExistingChangeEditorsAndPartialSuccess(t *testing.T) {
	a := &appDocs{config: dto.ProjectConfig{ChangeDocs: []string{"spec"}}, rows: []dto.Document{}, current: []dto.Document{}}
	m := NewModelWithClient(a)
	m.state = ChangeDetailsState
	m.currentProject.ID = "7"
	m.changeList.Detail = dto.ChangeView{ID: "31", ProjectID: "7", Spec: "unchanged"}
	m.changeDetailLoaded = true
	got, cmd := sendCommand(m, "/documents")
	got = applyMsg(got, cmd())
	require.Equal(t, "spec", got.document.DraftType)
	got, _ = sendCommand(got, "/return")
	require.Equal(t, ChangeDetailsState, got.state)
	require.Empty(t, a.insert)
	// The old change editor keeps its own no-op behavior when its bytes are unchanged.
	got.changeDetailLoaded = true
	got.detailEditField = detailEditSpec
	got = applyMsg(got, editorFinishedMsg{source: ChangeDetailsState, original: "unchanged", content: "unchanged"})
	require.Empty(t, a.insert)
	require.True(t, strings.Contains(got.status, "unchanged"))
}

func TestP606ScenarioManifestIncludesDocumentsAndPTY(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "scripts", "terminal-scenarios.json"))
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
	for _, name := range []string{"TestCLIProgramDocumentOwnersAndHistory", "TestCLIProgramDocumentAppendAndRecovery", "TestCLIProgramDocumentMalformedAndStaleScope", "TestCLIProgramDocumentShutdownCancelsRequest"} {
		require.Contains(t, manifest.Program.Tests, name)
	}
	require.Contains(t, manifest.PTY.Tests, "TestShellNavigationEditorAndScrolling")
}
