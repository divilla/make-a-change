package app

import (
	"cli/internal/changes"
	"cli/internal/documents"
	"cli/internal/dto"
	"context"
	"errors"
	"strconv"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/require"
)

type review031Client struct {
	*docs031Client
	activeErr, commentsErr error
}

func (a *review031Client) ActiveDocuments(ctx context.Context, id int, table string) ([]dto.Document, error) {
	if a.activeErr != nil {
		return nil, a.activeErr
	}
	return a.docs031Client.ActiveDocuments(ctx, id, table)
}

func (a *review031Client) ListComments(ctx context.Context, id int, table string) ([]dto.Document, error) {
	if a.commentsErr != nil {
		return nil, a.commentsErr
	}
	return a.docs031Client.ListComments(ctx, id, table)
}

func Test031CommentCommitClosesEditorBeforeRefreshRecovery(t *testing.T) {
	for _, id := range []int{0, 7} {
		for _, stage := range []string{"active", "comments"} {
			t.Run(strconv.Itoa(id)+"/"+stage, func(t *testing.T) {
				m, base := app031Model(t)
				a := &review031Client{docs031Client: base}
				m.client = a
				if stage == "active" {
					a.activeErr = errors.New("refresh offline")
				} else {
					a.commentsErr = errors.New("refresh offline")
				}
				next, _ := m.beginComment(id)
				m = next.(Model)
				next, cmd := m.Update(editorFinishedMsg{source: ChangeDetailsState, content: "exact\ncomment draft"})
				m = applyCommand(next.(Model), cmd)
				require.Contains(t, m.status, "committed")
				require.Contains(t, m.status, "/retry reads only")
				require.Contains(t, m.err, "refresh offline")
				require.Empty(t, m.detailEditField)
				require.Nil(t, m.editorDraft)
				require.Empty(t, m.promptValue())
				require.Empty(t, m.changeDocuments.Draft)
				pressed, _ := sendKey(m, tea.KeyEnter)
				require.False(t, pressed.changeDocuments.Busy, "Enter cannot replay the committed write")
				// Exercise typed command routing, which an editor draft previously captured.
				m, _ = sendKeyMsg(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/retry")})
				m, cmd = sendKey(m, tea.KeyEnter)
				require.NotNil(t, cmd)
				r := cmd().(documents.ChangeResult)
				require.Equal(t, documents.Read, r.Operation)
				m = applyMsg(m, r)
				require.Contains(t, m.err, "refresh offline")
				a.activeErr, a.commentsErr = nil, nil
				m, cmd = sendCommand(m, "/retry")
				m = applyCommand(m, cmd)
				require.Empty(t, m.err)
				require.Equal(t, []string{"exact\ncomment draft"}, a.commentBodies)
				if id == 0 {
					require.Equal(t, []int{99}, a.commentInserts)
					require.Empty(t, a.commentUpdates)
				} else {
					require.Equal(t, []int{7}, a.commentUpdates)
					require.Empty(t, a.commentInserts)
				}
			})
		}
	}
}

func Test031FailedCommentWriteRetainsExactEditorDraft(t *testing.T) {
	for _, id := range []int{0, 7} {
		t.Run(strconv.Itoa(id), func(t *testing.T) {
			m, a := app031Model(t)
			a.insertErr = errors.New("write offline")
			next, _ := m.beginComment(id)
			draft := "exact\ncomment draft\n"
			next, cmd := next.(Model).Update(editorFinishedMsg{source: ChangeDetailsState, content: draft})
			m = applyCommand(next.(Model), cmd)
			require.Contains(t, m.err, "write offline")
			require.Equal(t, detailEditComment, m.detailEditField)
			require.Equal(t, draft, m.promptValue())
			require.Equal(t, draft, m.changeDocuments.Draft)
			require.NotNil(t, m.editorDraft)
			a.insertErr = nil
			m, cmd = sendKey(m, tea.KeyEnter)
			m = applyCommand(m, cmd)
			require.Empty(t, m.err)
			require.Empty(t, m.detailEditField)
			require.Nil(t, m.editorDraft)
			require.Equal(t, []string{draft, draft}, a.commentBodies)
		})
	}
}

func Test031InactiveReturnReloadsActiveRowsAndPreservesSelection(t *testing.T) {
	for _, key := range []tea.KeyType{tea.KeyEsc, tea.KeyCtrlC} {
		t.Run(key.String(), func(t *testing.T) {
			m, a := app031Model(t)
			m.state = ChangesListState
			m.changesFilters.find = "match"
			m.changeList.Rows = []dto.ChangeView{{ID: "2", Title: "match first"}, {ID: "3", Title: "match selected"}}
			m.changeList.Selected = 1
			a.inactive = []dto.Change{{ID: 4, ProjectID: 7, Title: "match activated"}}
			m, cmd := sendKey(m, tea.KeyCtrlH)
			m = applyCommand(m, cmd)
			m, cmd = sendRune(m, ' ')
			m = applyCommand(m, cmd)
			require.Equal(t, []int{4}, a.activationIDs)
			a.changeRows = []dto.ChangeView{{ID: "4", Title: "match activated"}, {ID: "2", Title: "match first"}, {ID: "3", Title: "match selected"}, {ID: "5", Title: "excluded"}}
			m, cmd = sendKey(m, key)
			m = applyCommand(m, cmd)
			require.False(t, m.changeList.Inactive)
			require.False(t, m.changeList.Loading)
			require.Equal(t, 1, a.changeListCalls)
			require.Equal(t, "match", m.changesFilters.find)
			require.Equal(t, "4", m.changeList.Rows[0].ID)
			require.Equal(t, "3", changes.FilteredRows(m.changeList.Rows, m.changeFilters())[m.changeList.Selected].ID)
		})
	}
}

func Test031InactiveReturnRestartsCanceledActiveLoad(t *testing.T) {
	m, a := app031Model(t)
	m.state = ChangesListState
	a.changeRows = []dto.ChangeView{{ID: "2", Title: "fresh active"}}
	next, canceled := m.beginChange(changes.List, 0, changes.Input{})
	m, cmd := sendKey(next.(Model), tea.KeyCtrlH)
	m = applyCommand(m, cmd)
	m, cmd = sendKey(m, tea.KeyEsc)
	require.NotNil(t, cmd, "return must restart the canceled active request")
	require.True(t, m.changeList.Loading)
	m = applyCommand(m, cmd)
	require.False(t, m.changeList.Loading)
	require.Equal(t, "2", m.changeList.Rows[0].ID)
	m = applyCommand(m, canceled)
	require.False(t, m.changeList.Loading)
	require.Empty(t, m.err, "late canceled result must not replace the reload")
}

func Test031OwnerHistoryReturnKeepsScrolledSelectionVisible(t *testing.T) {
	for _, key := range []tea.KeyType{tea.KeyEsc, tea.KeyCtrlC} {
		t.Run(key.String(), func(t *testing.T) {
			m, a := app031Model(t)
			m.width, m.height = 80, 15
			for id := 40; id > 0; id-- {
				a.rows = append(a.rows, app031Row(id, "notes", "body"))
			}
			m, cmd := sendCommand(m, "/documents")
			m = applyCommand(m, cmd)
			for range 25 {
				m, _ = sendKey(m, tea.KeyDown)
			}
			selected := m.document.Rows[m.document.Selected]
			require.Positive(t, m.document.Offset)
			a.detail = selected
			m, cmd = sendKey(m, tea.KeyEnter)
			next, render := m.Update(cmd())
			m = applyCommand(next.(Model), render)
			require.True(t, m.historyOpen)
			require.False(t, m.history.Busy)
			require.Zero(t, m.document.Offset)
			m, _ = sendKey(m, key)
			require.False(t, m.historyOpen)
			require.False(t, m.document.ShowingDetail)
			require.Equal(t, selected, m.document.Rows[m.document.Selected])
			require.Contains(t, documents.View(m.document, 80, m.documentViewportHeight()), "> #"+strconv.Itoa(selected.ID))
		})
	}
}
