package documents

import (
	"cli/internal/dto"
	"cli/internal/styles"
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
	"github.com/stretchr/testify/require"
)

type refactorAPI struct {
	*docAPI
	rows, active, comments []dto.Document
	calls                  []string
	ids                    []int
	bodies                 []string
	writeErr, readErr      error
}

func refactorRow(id int, kind, body string) dto.Document {
	created := time.Date(2026, 3, 29, 0, 30, 0, 0, time.UTC)
	return dto.Document{ID: id, RefID: 12, RefTable: "change", DocType: kind, Body: body, HTML: "<p>body</p>", CreatedAt: created, UpdatedAt: created.Add(time.Hour)}
}

func (a *refactorAPI) ActiveDocuments(context.Context, int, string) ([]dto.Document, error) {
	a.calls = append(a.calls, "active")
	return slices.Clone(a.active), a.readErr
}

func (a *refactorAPI) ListDocuments(context.Context, int, string) ([]dto.Document, error) {
	a.calls = append(a.calls, "history")
	return slices.Clone(a.rows), a.readErr
}

func (a *refactorAPI) ListComments(context.Context, int, string) ([]dto.Document, error) {
	a.calls = append(a.calls, "comments")
	return slices.Clone(a.comments), a.readErr
}

func (a *refactorAPI) InsertComment(_ context.Context, _ int, _ string, body string) (int, error) {
	a.calls = append(a.calls, "insert-comment")
	a.bodies = append(a.bodies, body)
	if a.writeErr == nil {
		a.comments = append([]dto.Document{refactorRow(99, "comment", body)}, a.comments...)
	}
	return 99, a.writeErr
}

func (a *refactorAPI) UpdateComment(_ context.Context, id int, body string) error {
	a.calls = append(a.calls, "update-comment")
	a.ids = append(a.ids, id)
	a.bodies = append(a.bodies, body)
	if a.writeErr == nil {
		for i := range a.comments {
			if a.comments[i].ID == id {
				a.comments[i].Body = body
			}
		}
	}
	return a.writeErr
}

func (a *refactorAPI) DeleteDocument(_ context.Context, id int) error {
	a.calls = append(a.calls, "delete")
	a.ids = append(a.ids, id)
	if a.writeErr == nil {
		a.active = slices.DeleteFunc(a.active, func(d dto.Document) bool { return d.ID == id })
		deleted := time.Now()
		for i := range a.comments {
			if a.comments[i].ID == id {
				a.comments[i].DeletedAt = &deleted
			}
		}
	}
	return a.writeErr
}

func (a *refactorAPI) ActivateDocument(_ context.Context, id int) error {
	a.calls = append(a.calls, "activate")
	a.ids = append(a.ids, id)
	if a.writeErr == nil {
		for i := range a.rows {
			if a.rows[i].ID == id {
				a.rows[i].DeletedAt = nil
				a.active = []dto.Document{a.rows[i]}
			}
		}
	}
	return a.writeErr
}

func (a *refactorAPI) UndeleteComment(_ context.Context, id int) error {
	a.calls = append(a.calls, "undelete")
	a.ids = append(a.ids, id)
	if a.writeErr == nil {
		for i := range a.comments {
			if a.comments[i].ID == id {
				a.comments[i].DeletedAt = nil
			}
		}
	}
	return a.writeErr
}

type coloredPrinter struct {
	bodies []string
	err    error
}

func (p *coloredPrinter) Print(ctx context.Context, body string) (string, error) {
	p.bodies = append(p.bodies, body)
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	if p.err != nil {
		return "", p.err
	}
	return "\x1b[38;2;11;22;33m" + body + "\x1b[0m", nil
}

func finishHistory(t *testing.T, h History, cmd tea.Cmd) History {
	t.Helper()
	require.NotNil(t, cmd)
	h, ok := h.Apply(cmd().(HistoryResult))
	require.True(t, ok)
	return h
}

func Test031ActiveDocumentSelectionAndDuplicateTypes(t *testing.T) {
	a := &refactorAPI{docAPI: &docAPI{}, active: []dto.Document{refactorRow(9, "spec", "older active")}, rows: []dto.Document{refactorRow(10, "spec", "newest"), refactorRow(9, "spec", "older active")}}
	rows, err := (Access{API: a}).Load(context.Background(), 12)
	require.NoError(t, err)
	require.Equal(t, 9, rows[0].ID)
	a.active = append(a.active, refactorRow(10, "spec", "duplicate"))
	_, err = (Access{API: a}).Load(context.Background(), 12)
	require.ErrorContains(t, err, "duplicate")
	a.active = []dto.Document{refactorRow(9, "spec", "bad owner")}
	a.active[0].RefID = 13
	_, err = (Access{API: a}).Load(context.Background(), 12)
	require.Error(t, err)
}

func Test031CommentEditEmptyBodyAndConfirmedDelete(t *testing.T) {
	a := &refactorAPI{docAPI: &docAPI{}, comments: []dto.Document{refactorRow(10, "comment", "one\ntwo\nthree\nfour"), refactorRow(9, "comment", "independent")}}
	m, cmd := (ChangeModel{}).Begin(context.Background(), a, 7, 12, EditComment, 10, "")
	m, ok := m.Apply(cmd().(ChangeResult))
	require.True(t, ok)
	require.NoError(t, m.Err)
	require.Equal(t, "", a.comments[0].Body)
	require.Equal(t, "independent", a.comments[1].Body)
	require.Equal(t, []int{10}, a.ids)
	m, cmd = m.Begin(context.Background(), a, 7, 12, DeleteDocument, 10, "")
	m, ok = m.Apply(cmd().(ChangeResult))
	require.True(t, ok)
	require.NoError(t, m.Err)
	require.NotNil(t, a.comments[0].DeletedAt)
	require.Equal(t, 9, a.comments[1].ID)
	require.NotContains(t, a.calls, "insert-comment")
}

func Test031MutationPartialSuccessAndReadOnlyRetry(t *testing.T) {
	for _, op := range []Mutation{NewComment, EditComment, DeleteDocument, Activate, Undelete} {
		t.Run(string(op), func(t *testing.T) {
			a := &refactorAPI{docAPI: &docAPI{}, readErr: errors.New("read unavailable")}
			m, cmd := (ChangeModel{}).Begin(context.Background(), a, 7, 12, op, 9, "draft")
			_, duplicate := m.Begin(context.Background(), a, 7, 12, op, 9, "draft")
			require.Nil(t, duplicate)
			result := cmd().(ChangeResult)
			require.NotEmpty(t, result.Committed)
			require.NoError(t, result.Err)
			require.Error(t, result.RefreshErr)
			m, ok := m.Apply(result)
			require.True(t, ok)
			require.NotEmpty(t, m.Committed)
			before := len(a.calls)
			m, cmd = m.Begin(context.Background(), a, 7, 12, Read, 0, "")
			m, ok = m.Apply(cmd().(ChangeResult))
			require.True(t, ok)
			require.NotEmpty(t, m.Committed)
			require.Equal(t, []string{"active"}, a.calls[before:])
			a.readErr = nil
			m, cmd = m.Begin(context.Background(), a, 7, 12, Read, 0, "")
			m, ok = m.Apply(cmd().(ChangeResult))
			require.True(t, ok)
			require.NoError(t, m.Err)
		})
	}
	a := &refactorAPI{docAPI: &docAPI{}, writeErr: errors.New("write refused")}
	m, cmd := (ChangeModel{}).Begin(context.Background(), a, 7, 12, NewComment, 0, "useful\ndraft")
	m, ok := m.Apply(cmd().(ChangeResult))
	require.True(t, ok)
	require.Equal(t, "useful\ndraft", m.Draft)
	require.Empty(t, m.Committed)
	_, cmd = m.Begin(context.Background(), a, 7, 12, NewComment, 0, " \n")
	require.Nil(t, cmd)
}

func Test031LateResultIsolationAndBusyDeduplication(t *testing.T) {
	a := &refactorAPI{docAPI: &docAPI{}}
	m, cmd := (ChangeModel{}).Begin(context.Background(), a, 7, 12, NewComment, 0, "draft")
	_, duplicate := m.Begin(context.Background(), a, 7, 12, NewComment, 0, "draft")
	require.Nil(t, duplicate)
	stale := ChangeResult{ProjectID: 7, OwnerID: 12, Revision: m.Revision}
	m = m.Invalidate()
	m, ok := m.Apply(stale)
	require.False(t, ok)
	result := cmd().(ChangeResult)
	require.ErrorIs(t, result.Err, context.Canceled)
	require.Empty(t, a.calls)
	h, cmd := (History{}).Open(context.Background(), a, &coloredPrinter{}, 7, 12, "change", "spec")
	old := HistoryResult{ProjectID: 7, OwnerID: 12, Table: "change", Type: "spec", Revision: h.Revision}
	h = h.Invalidate()
	h, ok = h.Apply(old)
	require.False(t, ok)
	require.ErrorIs(t, cmd().(HistoryResult).Err, context.Canceled)
}

func Test031DocumentHistoryTypeScopeOrderAndArrowBounds(t *testing.T) {
	deleted := time.Now()
	older := refactorRow(8, "spec", "old")
	older.DeletedAt = &deleted
	a := &refactorAPI{docAPI: &docAPI{}, rows: []dto.Document{older, refactorRow(11, "brief", "unrelated"), refactorRow(10, "spec", "new")}, active: []dto.Document{refactorRow(8, "spec", "old")}}
	a.active[0].DeletedAt = nil
	p := &coloredPrinter{}
	h, cmd := (History{}).Open(context.Background(), a, p, 7, 12, "change", "spec")
	h = finishHistory(t, h, cmd)
	require.NoError(t, h.Err)
	require.Equal(t, 10, h.Rows[h.Selected].ID)
	require.Len(t, h.Rows, 2)
	h, cmd = h.Move(context.Background(), a, p, -1)
	require.Nil(t, cmd)
	h, cmd = h.Move(context.Background(), a, p, 1)
	h = finishHistory(t, h, cmd)
	require.Equal(t, 8, h.Rows[h.Selected].ID)
	require.NotNil(t, h.Rows[h.Selected].DeletedAt)
	h, cmd = h.Move(context.Background(), a, p, 1)
	require.Nil(t, cmd)
	h, cmd = h.Move(context.Background(), a, p, -1)
	h = finishHistory(t, h, cmd)
	require.Equal(t, 10, h.Rows[h.Selected].ID)
	require.Equal(t, []string{"new", "old", "new"}, p.bodies)
	a.rows[0].RefID = 13
	h, cmd = h.Refresh(context.Background(), a, p)
	h = finishHistory(t, h, cmd)
	require.ErrorContains(t, h.Err, "owner")
}

func Test031HistoryShowsCreatedUpdatedAndDeletedTimestamps(t *testing.T) {
	profile := lipgloss.ColorProfile()
	t.Cleanup(func() { lipgloss.SetColorProfile(profile) })
	lipgloss.SetColorProfile(termenv.TrueColor)
	old := time.Local
	local, err := time.LoadLocation("Europe/Zagreb")
	require.NoError(t, err)
	time.Local = local
	t.Cleanup(func() { time.Local = old })
	d := refactorRow(9, "spec", "body")
	deleted := d.UpdatedAt.Add(time.Hour)
	d.DeletedAt = &deleted
	h := History{Rows: []dto.Document{d}}
	metadata := h.Metadata()
	require.Contains(t, metadata, "created_at: 2026-03-29 01:30")
	require.Contains(t, metadata, "updated_at: 2026-03-29 03:30")
	require.Contains(t, metadata, "deleted_at: 2026-03-29 04:30")
	require.Contains(t, metadata, lipgloss.NewStyle().Foreground(styles.AccentRed).Render("deleted_at: 2026-03-29 04:30"))
	require.NotContains(t, metadata, "\n")
	h.Rows[0].DeletedAt = nil
	require.NotContains(t, h.Metadata(), "deleted_at")
	require.Equal(t, d.CreatedAt, h.Rows[0].CreatedAt)
}

func Test031HistoryPreservesBatANSIColorsAndVisibleWidth(t *testing.T) {
	a := &refactorAPI{docAPI: &docAPI{}, rows: []dto.Document{refactorRow(9, "spec", strings.Repeat("# highlighted **text** and 漢字\n", 30)), refactorRow(8, "spec", "older **text**")}}
	p := &coloredPrinter{}
	h, cmd := (History{}).Open(context.Background(), a, p, 7, 12, "change", "spec")
	h = finishHistory(t, h, cmd)
	for _, size := range [][2]int{{12, 3}, {30, 5}, {80, 7}} {
		view := h.View(size[0], size[1])
		require.Contains(t, view, "\x1b[38;2;11;22;33m")
		for _, line := range strings.Split(view, "\n") {
			require.LessOrEqual(t, ansi.StringWidth(line), size[0])
		}
		h = h.Scroll(3, size[1])
		view = h.View(size[0], size[1])
		require.Contains(t, view, "# high")
		require.Contains(t, view, "\x1b[38;2;11;22;33m")
		h = h.Scroll(-999, size[1])
	}
	h, cmd = h.Move(context.Background(), a, p, 1)
	h = finishHistory(t, h, cmd)
	require.Contains(t, h.View(80, 4), "\x1b[38;2;11;22;33m")
	require.Contains(t, h.Output, "older")
}

func Test031HistoricalDocumentActivation(t *testing.T) {
	deleted := time.Now()
	d := refactorRow(8, "spec", "older exact\nbody")
	d.DeletedAt = &deleted
	a := &refactorAPI{docAPI: &docAPI{}, rows: []dto.Document{refactorRow(9, "spec", "new"), d}, active: []dto.Document{refactorRow(9, "spec", "new")}}
	p := &coloredPrinter{}
	h, cmd := (History{}).Open(context.Background(), a, p, 7, 12, "change", "spec")
	h = finishHistory(t, h, cmd)
	h, cmd = h.Activate(context.Background(), a, p)
	require.Nil(t, cmd)
	h, cmd = h.Move(context.Background(), a, p, 1)
	h = finishHistory(t, h, cmd)
	h, cmd = h.Activate(context.Background(), a, p)
	h = finishHistory(t, h, cmd)
	require.NoError(t, h.Err)
	require.Equal(t, 8, h.Rows[h.Selected].ID)
	require.Equal(t, 8, h.Active[0].ID)
	require.Equal(t, d.Body, h.Active[0].Body)
	require.Nil(t, h.Rows[h.Selected].DeletedAt)
	require.Equal(t, []int{8}, a.ids)
	require.Contains(t, h.Help(), "Space active selection")
	h, cmd = h.Activate(context.Background(), a, p)
	require.Nil(t, cmd)
}

func Test031DeletedCommentRestoration(t *testing.T) {
	deleted := time.Now()
	d := refactorRow(9, "comment", "same body")
	d.DeletedAt = &deleted
	a := &refactorAPI{docAPI: &docAPI{}, comments: []dto.Document{d}}
	p := &coloredPrinter{}
	h, cmd := (History{}).Open(context.Background(), a, p, 7, 12, "change", "comment")
	h = finishHistory(t, h, cmd)
	h, cmd = h.Activate(context.Background(), a, p)
	h = finishHistory(t, h, cmd)
	require.NoError(t, h.Err)
	require.Equal(t, 9, h.Rows[0].ID)
	require.Nil(t, h.Rows[0].DeletedAt)
	require.Equal(t, d.Body, h.Rows[0].Body)
	require.Equal(t, d.UpdatedAt, h.Rows[0].UpdatedAt)
	require.Equal(t, []int{9}, a.ids)
	require.Equal(t, []string{"comments", "active", "undelete", "comments", "active"}, a.calls)
	require.Contains(t, h.Help(), "Space undelete comment")
	h, cmd = h.Activate(context.Background(), a, p)
	require.Nil(t, cmd)
}

func Test031HistoryPrintingFailureAndReadOnlyRetry(t *testing.T) {
	a := &refactorAPI{docAPI: &docAPI{}, rows: []dto.Document{refactorRow(9, "spec", "exact body")}}
	p := &coloredPrinter{err: errors.New("bat unavailable")}
	h, cmd := (History{}).Open(context.Background(), a, p, 7, 12, "change", "spec")
	h = finishHistory(t, h, cmd)
	require.Equal(t, 9, h.Rows[h.Selected].ID)
	require.ErrorContains(t, h.Err, "bat unavailable")
	p.err = nil
	h, cmd = h.Refresh(context.Background(), a, p)
	h = finishHistory(t, h, cmd)
	require.NoError(t, h.Err)
	a.readErr = errors.New("read failed")
	h, cmd = h.Activate(context.Background(), a, p)
	h = finishHistory(t, h, cmd)
	require.NotEmpty(t, h.Committed)
	require.Error(t, h.Err)
	before := len(a.ids)
	h, cmd = h.Activate(context.Background(), a, p)
	h = finishHistory(t, h, cmd)
	require.Len(t, a.ids, before)
}

func Test031HistoryRefreshClearsAnEmptyRetainedList(t *testing.T) {
	a := &refactorAPI{rows: []dto.Document{refactorRow(9, "spec", "body")}}
	p := &coloredPrinter{}
	h, cmd := (History{}).Open(context.Background(), a, p, 7, 12, "change", "spec")
	h, ok := h.Apply(cmd().(HistoryResult))
	require.True(t, ok)
	a.rows = []dto.Document{}
	h, cmd = h.Refresh(context.Background(), a, p)
	h, ok = h.Apply(cmd().(HistoryResult))
	require.True(t, ok)
	require.Empty(t, h.Rows)
	require.Contains(t, h.View(80, 10), "No retained documents")
	require.Len(t, p.bodies, 1)
}
