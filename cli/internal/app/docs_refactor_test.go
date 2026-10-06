package app

import (
	"cli/internal/changes"
	"cli/internal/documents"
	"cli/internal/dto"
	"cli/internal/epics"
	"cli/internal/projects"
	"cli/internal/styles"
	"cli/internal/testcases"
	"context"
	"encoding/json"
	"errors"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"github.com/stretchr/testify/require"
)

type docs031Client struct {
	appDocs
	comments                                []dto.Document
	deletes, commentUpdates, commentInserts []int
	commentBodies                           []string
	inactive                                []dto.Change
	inactiveProjects                        []int
	activationIDs                           []int
	epicRows                                []dto.Epic
}

func (a *docs031Client) ListComments(context.Context, int, string) ([]dto.Document, error) {
	return slices.Clone(a.comments), nil
}

func (a *docs031Client) InsertComment(_ context.Context, _ int, _ string, body string) (int, error) {
	a.commentInserts = append(a.commentInserts, 99)
	a.commentBodies = append(a.commentBodies, body)
	a.comments = append([]dto.Document{app031Row(99, "comment", body)}, a.comments...)
	return 99, a.insertErr
}

func (a *docs031Client) UpdateComment(_ context.Context, id int, body string) error {
	a.commentUpdates = append(a.commentUpdates, id)
	a.commentBodies = append(a.commentBodies, body)
	for i := range a.comments {
		if a.comments[i].ID == id {
			a.comments[i].Body = body
		}
	}
	return a.insertErr
}

func (a *docs031Client) DeleteDocument(_ context.Context, id int) error {
	a.deletes = append(a.deletes, id)
	a.current = slices.DeleteFunc(a.current, func(d dto.Document) bool { return d.ID == id })
	deleted := time.Now()
	for i := range a.comments {
		if a.comments[i].ID == id {
			a.comments[i].DeletedAt = &deleted
		}
	}
	return a.insertErr
}

func (a *docs031Client) ListInactiveChanges(_ context.Context, project int) ([]dto.Change, error) {
	a.inactiveProjects = append(a.inactiveProjects, project)
	return slices.Clone(a.inactive), a.err
}

func (a *docs031Client) UpdateChangeActive(_ context.Context, id int, active bool) error {
	requireActive := active
	if !requireActive {
		return errors.New("expected activation")
	}
	a.activationIDs = append(a.activationIDs, id)
	a.inactive = slices.DeleteFunc(a.inactive, func(c dto.Change) bool { return c.ID == id })
	return a.err
}

func (a *docs031Client) ListEpics(context.Context, int) ([]dto.Epic, error) { return a.epicRows, a.err }

type app031Printer struct{}

func (app031Printer) Print(_ context.Context, body string) (string, error) {
	return "\x1b[38;2;11;22;33m" + body + "\x1b[0m", nil
}

func app031Row(id int, kind, body string) dto.Document {
	when := time.Date(2026, 9, 28, 10, 0, 0, 0, time.UTC)
	return dto.Document{ID: id, RefID: 12, RefTable: "change", DocType: kind, Body: body, CreatedAt: when, UpdatedAt: when.Add(time.Hour)}
}

func app031Model(t *testing.T) (Model, *docs031Client) {
	t.Helper()
	t.Setenv("TMPDIR", t.TempDir())
	profile := lipgloss.ColorProfile()
	t.Cleanup(func() { lipgloss.SetColorProfile(profile) })
	lipgloss.SetColorProfile(termenv.TrueColor)
	a := &docs031Client{appDocs: appDocs{config: dto.ProjectConfig{ProjectDocs: []string{"readme"}, EpicDocs: []string{"epic-notes"}, ChangeDocs: []string{"brief", "spec", "notes", "pr"}}, current: []dto.Document{app031Row(9, "spec", "# full spec\n"+strings.Repeat("long body\n", 40)), app031Row(8, "brief", strings.Repeat("brief line\n", 20))}}, comments: []dto.Document{app031Row(7, "comment", "one\ntwo\nthree\nfull comment tail"), app031Row(6, "comment", "independent")}}
	m := newModelWithConfig(a, appConfig{ProjectID: 7})
	m.currentProject = dto.Option{ID: "7", Label: "Project"}
	m.state = ChangeDetailsState
	m.changeDetailLoaded = true
	m.width, m.height = 140, 40
	m.historyPrinter = app031Printer{}
	m.optionCatalog = optionCatalog{loaded: true, config: a.config}
	m.changeList.Detail = dto.ChangeView{ID: "12", ProjectID: "7", Title: "Change", DocumentTypes: a.config.ChangeDocs, Documents: slices.Clone(a.current), Comments: slices.Clone(a.comments), Brief: a.current[1].Body}
	return m, a
}

func select031Row(t *testing.T, m Model, kind string, id int) Model {
	t.Helper()
	for i, r := range changes.DetailRows(m.changeList.Detail) {
		if r.DocumentType == kind && (id < 0 || r.DocumentID == id) {
			m.changeList.DetailSelected = i
			return m
		}
	}
	t.Fatalf("row %s #%d missing", kind, id)
	return m
}

func Test031DocumentDeleteConfirmationLayoutAndColors(t *testing.T) {
	m, a := app031Model(t)
	m = select031Row(t, m, "spec", 9)
	m, cmd := sendKey(m, tea.KeyDelete)
	require.Nil(t, cmd)
	require.Empty(t, a.deletes)
	require.Equal(t, ChangeDetailsState, m.state)
	view := m.View()
	require.Contains(t, view, styles.Default.MenuPromptIndicator.Render(" Are you sure? > "))
	plain := stripANSI(view)
	require.Less(t, strings.Index(plain, "Docs"), strings.Index(plain, "Are you sure?"))
	require.Contains(t, plain, "yes")
	require.Contains(t, plain, "no")
	require.Equal(t, 9, m.deleteDocumentID)
	// Moving the confirmation menu cannot change its captured document target.
	m.changeList.DetailSelected = 0
	m, cmd = sendKey(m, tea.KeyEnter)
	require.NotNil(t, cmd)
	m = applyCommand(m, cmd)
	require.Equal(t, []int{9}, a.deletes)
	require.Zero(t, m.deleteDocumentID)
}

func Test031DocumentDeleteConfirmCancelAndMissingSlotNoOp(t *testing.T) {
	for _, key := range []tea.KeyType{tea.KeyEsc, tea.KeyCtrlC, tea.KeyEnter} {
		t.Run(key.String(), func(t *testing.T) {
			m, a := app031Model(t)
			m = select031Row(t, m, "brief", 8)
			m, _ = sendKey(m, tea.KeyDelete)
			if key == tea.KeyEnter {
				m, _ = sendKey(m, tea.KeyDown)
			}
			m, cmd := sendKey(m, key)
			require.Nil(t, cmd)
			require.Empty(t, a.deletes)
			require.False(t, m.hasDropdown())
			require.Zero(t, m.deleteDocumentID)
		})
	}
	m, a := app031Model(t)
	m = select031Row(t, m, "notes", 0)
	m, cmd := sendKey(m, tea.KeyDelete)
	require.Nil(t, cmd)
	require.False(t, m.hasDropdown())
	require.Empty(t, a.deletes)
	m = select031Row(t, m, "spec", 9)
	m, _ = sendKey(m, tea.KeyDelete)
	m, cmd = sendKey(m, tea.KeyEnter)
	m = applyCommand(m, cmd)
	require.Len(t, a.current, 1)
	for _, d := range m.changeList.Detail.Documents {
		require.NotEqual(t, 9, d.ID)
	}
	m = select031Row(t, m, "spec", 0)
	require.NotContains(t, m.View(), "full spec")
}

func Test031NewCommentFirstCommandAndIndependentInsert(t *testing.T) {
	m, a := app031Model(t)
	require.Equal(t, "/new-comment", commandOptions(ChangeDetailsState)[0].ID)
	m, cmd := sendCommand(m, "/new-comment")
	require.NotNil(t, cmd)
	require.Equal(t, detailEditComment, m.detailEditField)
	require.Empty(t, m.promptValue())
	// Simulate the terminal editor callback with exact multiline bytes.
	next, save := m.Update(editorFinishedMsg{source: ChangeDetailsState, content: "independent\nnew comment"})
	m = next.(Model)
	require.NotNil(t, save)
	m = applyCommand(m, save)
	require.Equal(t, []int{99}, a.commentInserts)
	require.Empty(t, a.insert)
	require.Equal(t, "independent\nnew comment", a.comments[0].Body)
	require.Len(t, a.current, 2)
	m, cmd = sendCommand(m, "/new-comment")
	require.NotNil(t, cmd)
	m, cmd = sendKey(m, tea.KeyEsc)
	require.Nil(t, cmd)
	require.Len(t, a.commentInserts, 1)
	m.detailEditField = detailEditComment
	m.commentID = 0
	next, cmd = m.saveChangeDetailTextValue(" \n")
	require.Nil(t, cmd)
	require.Contains(t, next.(Model).err, "required")
	require.Len(t, a.commentInserts, 1)
}

func Test031CommentEditingUsesFullBodyAndEmptyUpdate(t *testing.T) {
	m, a := app031Model(t)
	m = select031Row(t, m, "comment", 7)
	m, cmd := sendKey(m, tea.KeyEnter)
	require.NotNil(t, cmd)
	require.Equal(t, a.comments[0].Body, m.promptValue())
	require.Contains(t, m.promptValue(), "full comment tail")
	next, save := m.Update(editorFinishedMsg{source: ChangeDetailsState, original: a.comments[0].Body, content: ""})
	m = applyCommand(next.(Model), save)
	require.Equal(t, []int{7}, a.commentUpdates)
	require.Equal(t, "", a.comments[0].Body)
	require.Equal(t, "independent", a.comments[1].Body)
	require.Empty(t, a.commentInserts)
	m = select031Row(t, m, "comment", 7)
	m, _ = sendKey(m, tea.KeyDelete)
	m, cmd = sendKey(m, tea.KeyEnter)
	m = applyCommand(m, cmd)
	require.Equal(t, []int{7}, a.deletes)
	for _, r := range changes.DetailRows(m.changeList.Detail) {
		require.NotEqual(t, 7, r.DocumentID)
	}
}

func Test031DocumentSlotCreateEditCancelAndNoOp(t *testing.T) {
	m, a := app031Model(t)
	m = select031Row(t, m, "spec", 9)
	m, cmd := sendKey(m, tea.KeyEnter)
	require.NotNil(t, cmd)
	require.Equal(t, a.current[0].Body, m.promptValue())
	require.Equal(t, "spec", m.changeList.Draft.DocumentType)
	next, save := m.Update(editorFinishedMsg{source: ChangeDetailsState, original: a.current[0].Body, content: a.current[0].Body})
	m = next.(Model)
	require.Empty(t, a.insert)
	require.NotNil(t, save)
	require.Equal(t, "unchanged", m.status)
	m = select031Row(t, m, "notes", 0)
	m, cmd = sendKey(m, tea.KeyEnter)
	require.NotNil(t, cmd)
	require.Empty(t, m.promptValue())
	require.Equal(t, "notes", m.changeList.Draft.DocumentType)
	m, cmd = sendKey(m, tea.KeyEsc)
	require.Nil(t, cmd)
	require.Empty(t, a.insert)
	m.detailEditField = detailEditDocument
	m.changeList.Draft.DocumentType = "notes"
	_, save = m.saveChangeDetailTextValue("exact new\nbody")
	require.NotNil(t, save)
	r := save().(changes.Result)
	require.NoError(t, r.Err)
	require.Len(t, a.insert, 1)
	require.Equal(t, "notes", a.insert[0].DocType)
	require.False(t, a.insert[0].AgentEdit)
}

func Test031HistoryTimestampTopLineAndDeletedAccentRed(t *testing.T) {
	m, a := app031Model(t)
	deleted := a.current[0].UpdatedAt.Add(time.Hour)
	d := a.current[0]
	d.DeletedAt = &deleted
	a.rows = []dto.Document{d}
	m = select031Row(t, m, "spec", 9)
	m, cmd := sendKey(m, tea.KeyCtrlH)
	m = applyCommand(m, cmd)
	require.True(t, m.historyOpen)
	lines := strings.Split(m.View(), "\n")
	require.Contains(t, lines[1], "created_at:")
	require.Contains(t, lines[1], "updated_at:")
	require.Contains(t, lines[1], lipgloss.NewStyle().Foreground(styles.AccentRed).Render("deleted_at: "+deleted.Local().Format("2006-01-02 15:04")))
	metadata := lines[1]
	m, _ = sendKey(m, tea.KeyPgDown)
	require.Equal(t, metadata, strings.Split(m.View(), "\n")[1])
	require.NotZero(t, m.history.Offset)
}

func Test031HistoryEscapeCtrlCAndSelectionRestoration(t *testing.T) {
	for _, key := range []tea.KeyType{tea.KeyEsc, tea.KeyCtrlC} {
		t.Run(key.String(), func(t *testing.T) {
			m, a := app031Model(t)
			a.rows = slices.Clone(a.current)
			m = select031Row(t, m, "spec", 9)
			selected := m.changeList.DetailSelected
			m, cmd := sendKey(m, tea.KeyCtrlH)
			m = applyCommand(m, cmd)
			require.True(t, m.historyOpen)
			m, _ = sendKey(m, key)
			require.False(t, m.historyOpen)
			require.Equal(t, ChangeDetailsState, m.state)
			require.Equal(t, selected, m.changeList.DetailSelected)
		})
	}
}

func Test031HistoryRootViewFiltersDocumentAndCommentControls(t *testing.T) {
	for _, kind := range []string{"spec", "comment"} {
		t.Run(kind, func(t *testing.T) {
			m, a := app031Model(t)
			body := "before\x1b]52;c;YXR0YWNr\a\x1b[2Jafter\n" + strings.Repeat("safe line\n", 40)
			row := app031Row(9, kind, body)
			if kind == "comment" {
				a.comments = []dto.Document{row}
				m.changeList.Detail.Comments = slices.Clone(a.comments)
			} else {
				a.rows = []dto.Document{row}
			}
			m = select031Row(t, m, kind, 9)
			m, cmd := sendKey(m, tea.KeyCtrlH)
			m = applyCommand(m, cmd)
			require.True(t, m.historyOpen)
			view := m.View()
			require.Contains(t, view, "beforeafter")
			require.Contains(t, view, "\x1b[38;2;11;22;33m")
			require.NotContains(t, view, "\x1b]52")
			require.NotContains(t, view, "\x1b[2J")
			require.NotContains(t, view, "YXR0YWNr")
			require.Equal(t, body, m.history.Rows[0].Body)
			m, _ = sendKey(m, tea.KeyPgDown)
			view = m.View()
			require.Contains(t, view, "\x1b[38;2;11;22;33m")
			require.NotContains(t, view, "\x1b]52")
			require.NotContains(t, view, "\x1b[2J")
		})
	}
}

func Test031HistoryTabsKeepFooterWithinTerminal(t *testing.T) {
	for _, kind := range []string{"spec", "comment"} {
		t.Run(kind, func(t *testing.T) {
			m, a := app031Model(t)
			body := strings.Repeat("123456789012345\t12345\n", 40)
			row := app031Row(9, kind, body)
			if kind == "comment" {
				a.comments = []dto.Document{row}
				m.changeList.Detail.Comments = slices.Clone(a.comments)
			} else {
				a.rows = []dto.Document{row}
			}
			m = select031Row(t, m, kind, 9)
			m, cmd := sendKey(m, tea.KeyCtrlH)
			m = applyCommand(m, cmd)
			require.True(t, m.historyOpen)
			output := m.history.Output
			for _, width := range []int{20, 40, 80} {
				updated, _ := m.Update(tea.WindowSizeMsg{Width: width, Height: 24})
				m = updated.(Model)
				for _, key := range []tea.KeyType{tea.KeyPgDown, tea.KeyPgUp} {
					m, _ = sendKey(m, key)
					view := m.View()
					require.LessOrEqual(t, lipgloss.Height(view), m.height, "history and footer fit at width %d", width)
					require.Contains(t, view, "\x1b[38;2;11;22;33m")
					require.True(t, strings.HasSuffix(view, styles.Default.Footer.Width(width).Render(m.footerText())))
				}
			}
			require.Equal(t, body, m.history.Rows[0].Body)
			require.Equal(t, output, m.history.Output)
		})
	}
}

func Test031DeletedCommentsHistoryAndEmptySectionAccess(t *testing.T) {
	m, a := app031Model(t)
	deleted := time.Now()
	for i := range a.comments {
		a.comments[i].DeletedAt = &deleted
	}
	m.changeList.Detail.Comments = slices.Clone(a.comments)
	m = select031Row(t, m, "comment", 0)
	m, cmd := sendKey(m, tea.KeyCtrlH)
	m = applyCommand(m, cmd)
	require.True(t, m.historyOpen)
	require.Len(t, m.history.Rows, 2)
	require.Equal(t, 7, m.history.Rows[0].ID)
	require.NotNil(t, m.history.Rows[0].DeletedAt)
}

func Test031DocumentHistoryAndDeleteKeyFocus(t *testing.T) {
	m, a := app031Model(t)
	m = select031Row(t, m, "spec", 9)
	m = m.setPromptValue("text")
	m, _ = sendKey(m, tea.KeyDelete)
	require.False(t, m.hasDropdown())
	require.Empty(t, a.deletes)
	m, _ = sendKey(m, tea.KeyCtrlH)
	require.False(t, m.historyOpen)
	m = m.setPromptValue("")
	m.detailEditField = detailEditDocument
	m, _ = sendKey(m, tea.KeyDelete)
	require.False(t, m.hasDropdown())
	m.detailEditField = ""
	m.openCommandDropdown()
	m, _ = sendKey(m, tea.KeyCtrlH)
	require.False(t, m.historyOpen)
	require.Empty(t, a.deletes)
}

func Test031InactiveListRoutesKeyboardAndScope(t *testing.T) {
	m, a := app031Model(t)
	m.state = ChangesListState
	m.changeList.Rows = []dto.ChangeView{{ID: "2", ProjectID: "7", Title: "active"}}
	a.changeRows = slices.Clone(m.changeList.Rows)
	a.inactive = []dto.Change{{ID: 4, ProjectID: 7, Title: "inactive"}, {ID: 3, ProjectID: 7, Title: "other"}}
	m, cmd := sendKey(m, tea.KeyCtrlH)
	require.NotNil(t, cmd)
	require.True(t, m.changeList.Loading)
	stale := cmd().(changes.Result)
	m = applyMsg(m, stale)
	require.Equal(t, []int{7}, a.inactiveProjects)
	require.True(t, m.changeList.Inactive)
	require.Equal(t, "4", m.changeList.Rows[0].ID)
	m, _ = sendKey(m, tea.KeyDown)
	require.Equal(t, 1, m.changeList.Selected)
	m, cmd = sendCommand(m, "/inactive-filter")
	m = applyCommand(m, cmd)
	require.False(t, m.changeList.Inactive)
	require.Equal(t, "2", m.changeList.Rows[0].ID)
	m = applyMsg(m, stale)
	require.Equal(t, "2", m.changeList.Rows[0].ID)
	m, cmd = sendKey(m, tea.KeyCtrlH)
	m.currentProject.ID = "8"
	m.changeList = m.changeList.Scope(8)
	m = applyCommand(m, cmd)
	require.Empty(t, m.changeList.Rows)
}

func Test031InactiveChangeSpaceActivationAndFooter(t *testing.T) {
	m, a := app031Model(t)
	m.state = ChangesListState
	a.inactive = []dto.Change{{ID: 4, ProjectID: 7, Title: "inactive"}}
	m, cmd := sendKey(m, tea.KeyCtrlH)
	m = applyCommand(m, cmd)
	require.Contains(t, m.View(), "Space activate")
	m, cmd = sendRune(m, ' ')
	require.NotNil(t, cmd)
	_, duplicate := sendRune(m, ' ')
	require.Nil(t, duplicate)
	m = applyCommand(m, cmd)
	require.Equal(t, []int{4}, a.activationIDs)
	require.Empty(t, m.changeList.Rows)
	require.True(t, m.changeList.Inactive)
	m, _ = sendKey(m, tea.KeyCtrlC)
	require.True(t, m.changeList.Inactive)
	require.Equal(t, MainState, m.state)
}

func Test031ChangeDetailsEpicSelectionExcludesInactiveEpics(t *testing.T) {
	for _, rows := range [][]dto.Epic{{{ID: 3, ProjectID: 7, Name: "active", Active: true}, {ID: 4, ProjectID: 7, Name: "inactive", Active: false}}, {{ID: 4, ProjectID: 7, Name: "inactive", Active: false}}, {}} {
		m, a := app031Model(t)
		a.epicRows = rows
		m, cmd := sendCommand(m, "/epic")
		m = applyCommand(m, cmd)
		for _, o := range m.dropdown.options {
			require.NotEqual(t, "4", o.ID)
		}
		require.Equal(t, len(rows) > 1, slices.ContainsFunc(m.dropdown.options, func(o dto.Option) bool { return o.ID == "3" }))
		require.Contains(t, m.dropdown.options, dto.Option{ID: "@none", Label: "@none"})
	}
}

func Test031ProjectEpicDeleteDeactivationOutcomes(t *testing.T) {
	m, a := app031Model(t)
	a.projectRows = []dto.Project{{ID: 8, Name: "first", Active: true}, {ID: 7, Name: "retained", Active: false}}
	m.state = ProjectDetailsState
	m.projectList.Detail = dto.Project{ID: 7, Active: true}
	next, cmd := m.beginProject(projects.Delete, 7, "")
	m = applyCommand(next.(Model), cmd)
	require.Equal(t, a.projectRows, m.projectList.Rows)
	require.Contains(t, m.status, "deactivated")
	require.Equal(t, "7", m.currentProject.ID)
	e := epics.Model{ProjectID: 7, Detail: dto.Epic{ID: 3, Active: true}}
	a.epicRows = []dto.Epic{{ID: 3, ProjectID: 7, Active: false}}
	e, cmd = e.Begin(context.Background(), a, epics.Delete, 7, 3, "")
	e, ok := e.Apply(cmd().(epics.Result))
	require.True(t, ok)
	require.Contains(t, e.Status, "deactivated")
	require.Equal(t, a.epicRows, e.Rows)
}

func Test031LateEditorResultCannotChangeAnotherDocument(t *testing.T) {
	m, _ := app031Model(t)
	m.editorGeneration = 2
	m.detailEditField = detailEditDocument
	m.changeList.Draft.DocumentType = "notes"
	next, cmd := m.Update(editorFinishedMsg{source: ChangeDetailsState, generation: 1, projectID: "7", ownerID: "12", field: detailEditDocument, content: "late"})
	require.Nil(t, cmd)
	require.Empty(t, next.(Model).promptValue())
	next, cmd = m.Update(editorFinishedMsg{source: ChangeDetailsState, generation: 2, projectID: "7", ownerID: "13", field: detailEditDocument, content: "late"})
	require.Nil(t, cmd)
	require.Empty(t, next.(Model).promptValue())
}

func Test031OwnerCatalogsAndProjectSwitch(t *testing.T) {
	m, a := app031Model(t)
	m.catalogGeneration = 10
	m.appConfig.ProjectID = 7
	cfg := dto.ProjectConfig{ProjectDocs: []string{"project-only"}, EpicDocs: []string{"epic-only"}, ChangeDocs: []string{"notes", "spec", "brief"}}
	next, _ := m.Update(optionCatalogLoadedMsg{id: 7, generation: 10, config: cfg})
	m = next.(Model)
	require.Equal(t, cfg, m.optionCatalog.config)
	require.Equal(t, cfg.ChangeDocs, m.changeList.Detail.DocumentTypes)
	require.NotContains(t, m.View(), "project-only")
	require.NotContains(t, m.View(), "epic-only")
	m.appConfig.ProjectID, m.currentProject.ID, m.catalogGeneration = 8, "8", 11
	next, _ = m.Update(optionCatalogLoadedMsg{id: 7, generation: 10, config: a.config})
	m = next.(Model)
	require.Equal(t, cfg, m.optionCatalog.config, "late previous-project configuration rejected")
	next, _ = m.Update(optionCatalogLoadedMsg{id: 8, generation: 11, err: errors.New("catalog unavailable")})
	m = next.(Model)
	require.Empty(t, m.changeList.Detail.DocumentTypes)
	require.False(t, m.optionCatalog.loaded)
	require.Contains(t, m.View(), "catalog unavailable")
	m.detailEditField = detailEditDocument
	m.changeList.Draft.DocumentType = "spec"
	next, cmd := m.saveChangeDetailTextValue("draft")
	require.Nil(t, cmd)
	require.Empty(t, a.insert)
	require.NotEmpty(t, next.(Model).err)
	next, _ = m.Update(optionCatalogLoadedMsg{id: 8, generation: 11, config: dto.ProjectConfig{ChangeDocs: []string{}}})
	require.Empty(t, next.(Model).changeList.Detail.DocumentTypes)
}

func Test031RetainedOwnerHistory(t *testing.T) {
	for _, table := range []string{"project", "epic", "change"} {
		t.Run(table, func(t *testing.T) {
			m, a := app031Model(t)
			owner := 12
			origin := ChangeDetailsState
			if table == "project" {
				owner, origin = 7, ProjectDetailsState
				m.projectList.Detail = dto.Project{ID: 7}
				m.projectList.DetailLoaded = true
			}
			if table == "epic" {
				origin = EpicDetailsState
				m.epicList.Detail = dto.Epic{ID: 12, ProjectID: 7}
				m.epicList.DetailLoaded = true
			}
			row := app031Row(20, "notes", "# exact retained body\n")
			row.RefTable, row.RefID = table, owner
			a.rows = []dto.Document{row}
			a.current = []dto.Document{}
			a.detail = row
			m.state = origin
			m, cmd := sendCommand(m, "/documents")
			m = applyCommand(m, cmd)
			m, cmd = sendKey(m, tea.KeyEnter)
			next, render := m.Update(cmd())
			m = applyCommand(next.(Model), render)
			require.True(t, m.historyOpen)
			d, ok := m.history.Current()
			require.True(t, ok)
			require.Equal(t, row, d)
			require.Contains(t, m.View(), "exact retained body")
			require.Contains(t, m.View(), "created_at:")
			m, _ = sendKey(m, tea.KeyEsc)
			require.False(t, m.historyOpen)
			require.False(t, m.document.ShowingDetail)
			require.Equal(t, 20, m.document.Rows[m.document.Selected].ID)
			m, _ = sendCommand(m, "/return")
			require.Equal(t, origin, m.state)
		})
	}
}

func Test031ScenarioManifestIncludesDocsAndInactiveLists(t *testing.T) {
	body, err := os.ReadFile("../../scripts/terminal-scenarios.json")
	require.NoError(t, err)
	var manifest map[string]struct {
		Tests []string `json:"tests"`
	}
	require.NoError(t, json.Unmarshal(body, &manifest))
	for _, name := range []string{"TestCLIProgram031DocumentCommentsAndHistory", "TestCLIProgram031InactiveChangesAndEpicSelection", "TestCLIProgram031BatFailureKeepsHistoryAndReturns"} {
		require.Contains(t, manifest["program"].Tests, name)
	}
	require.Contains(t, manifest["pty"].Tests, "TestShellNavigationEditorAndScrolling")
	require.NotContains(t, manifest["program"].Tests, "TestCLIPackageBoundaries")
}

func Test031EmptyCommentHistoryIsAValidBlankViewport(t *testing.T) {
	h := documents.History{Rows: []dto.Document{app031Row(7, "comment", "")}}
	require.Empty(t, h.View(80, 10))
	require.Contains(t, h.Metadata(), "created_at:")
}

func Test031FullBriefReturnAndDocumentEditorScope(t *testing.T) {
	m, a := app031Model(t)
	m = select031Row(t, m, "brief", 8)
	m, cmd := sendKey(m, tea.KeyEnter)
	require.NotNil(t, cmd)
	require.Equal(t, a.current[1].Body, m.promptValue())
	require.Greater(t, len(strings.Split(m.promptValue(), "\n")), 16)
	m.editorGeneration = 4
	m.detailEditField = detailEditDocument
	m.changeList.Draft.DocumentType = "notes"
	next, cmd := m.Update(editorFinishedMsg{generation: 4, source: ChangeDetailsState, projectID: "7", ownerID: "12", field: detailEditDocument, documentType: "spec", content: "late"})
	require.Nil(t, cmd)
	require.Empty(t, a.insert)
	require.Equal(t, "notes", next.(Model).changeList.Draft.DocumentType)
	m.state, m.documentForm = DocumentState, true
	m.document = documents.Model{OwnerID: 22, OwnerTable: "epic", Revision: 6, DraftType: "notes", DraftBody: "keep"}
	msg := editorFinishedMsg{generation: 4, source: DocumentState, projectID: "7", ownerID: "12", field: detailEditDocument, documentOwner: 21, documentTable: "epic", documentRevision: 5, documentType: "notes", content: "late"}
	next, cmd = m.Update(msg)
	require.Nil(t, cmd)
	require.Equal(t, "keep", next.(Model).document.DraftBody)
}

func Test031RetryBelongsToTheLatestCommittedOperation(t *testing.T) {
	m, _ := app031Model(t)
	m.changeDocuments.ProjectID, m.changeDocuments.OwnerID = 7, 12
	m.changeDocuments.Committed = "committed comment-insert document #99"
	// A completed comment operation must not capture a later ordinary reload.
	m, cmd := sendCommand(m, "/retry")
	require.NotNil(t, cmd)
	require.IsType(t, changes.Result{}, cmd())
	// Start a different field operation after a failed comment refresh. Its
	// read-only recovery must now belong to that field, retaining no old outcome.
	m, a := app031Model(t)
	a.changeGetErr = errors.New("refresh unavailable")
	m.changeDocuments.ProjectID, m.changeDocuments.OwnerID = 7, 12
	m.changeDocuments.Committed = "committed comment-insert document #99"
	m.changeDocuments.Err = errors.New("comment refresh failed")
	next, cmd := m.beginChange(changes.Title, 12, changes.Input{Value: "new title"})
	m = next.(Model)
	require.NotNil(t, cmd)
	require.Empty(t, m.changeDocuments.Committed)
	require.NoError(t, m.changeDocuments.Err)
	result := cmd().(changes.Result)
	require.NoError(t, result.Err)
	require.Len(t, result.Steps, 1)
	require.ErrorContains(t, result.RefreshErr, "refresh unavailable")
	// The committed title refresh fails; /retry must still only read.
	m = applyMsg(m, result)
	m, cmd = sendCommand(m, "/retry")
	require.NotNil(t, cmd)
	require.IsType(t, changes.Result{}, cmd())
	require.Empty(t, a.commentInserts)
	m, a = app031Model(t)
	a.changeGetErr = errors.New("refresh unavailable")
	m.changeDocuments.ProjectID, m.changeDocuments.OwnerID = 7, 12
	m.changeDocuments.Committed = "committed comment-insert document #99"
	m.changeDocuments.Err = errors.New("comment refresh failed")
	next, cmd = m.beginTestCase(testcases.Create, "", "case", false)
	m = next.(Model)
	require.NotNil(t, cmd)
	require.Empty(t, m.changeDocuments.Committed)
	m = applyMsg(m, cmd())
	m, cmd = sendCommand(m, "/retry")
	require.NotNil(t, cmd)
	require.IsType(t, testcases.Result{}, cmd())
	require.Equal(t, 1, a.testCaseCreateCalls)
}
