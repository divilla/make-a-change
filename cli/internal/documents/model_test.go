package documents

import (
	"cli/internal/dto"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/require"
)

type docAPI struct {
	cfg                                               dto.ProjectConfig
	rows, current                                     []dto.Document
	detail                                            dto.Document
	listErr, currentErr, insertErr, detailErr, cfgErr error
	inserted                                          []dto.DocumentInput
	listCalls, currentCalls, detailCalls              int
	ctx                                               context.Context
}

func (a *docAPI) GetProjectConfig(context.Context, int) (dto.ProjectConfig, error) {
	return a.cfg, a.cfgErr
}

func (a *docAPI) ListDocuments(ctx context.Context, _ int, _ string) ([]dto.Document, error) {
	a.ctx = ctx
	a.listCalls++
	return a.rows, a.listErr
}

func (a *docAPI) CurrentDocuments(context.Context, int, string) ([]dto.Document, error) {
	a.currentCalls++
	return a.current, a.currentErr
}

func (a *docAPI) DocumentDetails(context.Context, int) (dto.Document, error) {
	a.detailCalls++
	return a.detail, a.detailErr
}

func (a *docAPI) InsertDocument(_ context.Context, in dto.DocumentInput) (int, error) {
	a.inserted = append(a.inserted, in)
	return 91, a.insertErr
}

func scoped(t *testing.T, a *docAPI, project, owner int, table string) Model {
	t.Helper()
	m, cmd := Model{}.BeginOpen(context.Background(), a, project, owner, table)
	require.NotNil(t, cmd)
	var ok bool
	m, ok = m.Apply(cmd().(Result))
	require.True(t, ok)
	return m
}

func doc(id, owner int, table string, current bool) dto.Document {
	when := time.Date(2026, 9, 28, 10, 0, 0, 0, time.UTC)
	return dto.Document{ID: id, RefID: owner, RefTable: table, DocType: "spec", Body: "raw", HTML: "<p>rendered</p>", Current: current, CreatedAt: when, UpdatedAt: when}
}

func TestP602OwnerCatalogsAndEmptyReadAccess(t *testing.T) {
	for _, tt := range []struct {
		table          string
		project, owner int
		types          []string
	}{{"project", 7, 7, []string{"roadmap"}}, {"epic", 7, 12, []string{"story"}}, {"change", 7, 31, []string{"brief", "spec"}}} {
		a := &docAPI{cfg: dto.ProjectConfig{ProjectDocs: []string{"roadmap"}, EpicDocs: []string{"story"}, ChangeDocs: []string{"brief", "spec"}}, rows: []dto.Document{}, current: []dto.Document{}}
		m := scoped(t, a, tt.project, tt.owner, tt.table)
		require.Equal(t, tt.types, m.Types)
		require.Equal(t, tt.types[0], m.DraftType)
		require.Contains(t, View(m, 80, 12), "No document versions")
		require.Equal(t, 1, a.listCalls)
		require.Equal(t, 1, a.currentCalls)
	}
	a := &docAPI{rows: []dto.Document{doc(9, 12, "epic", false)}, current: []dto.Document{}, cfg: dto.ProjectConfig{EpicDocs: []string{}}}
	m := scoped(t, a, 7, 12, "epic")
	require.True(t, m.Loaded)
	require.Contains(t, View(m, 80, 12), "#9")
	m, cmd := m.BeginInsert(context.Background(), a, "body", false)
	require.Nil(t, cmd)
	require.ErrorContains(t, m.Err, "configured")
}

func TestP602HistorySelectionDetailsAndViewport(t *testing.T) {
	a := &docAPI{cfg: dto.ProjectConfig{ChangeDocs: []string{"spec"}}, rows: []dto.Document{doc(91, 12, "change", true), doc(90, 12, "change", false)}, current: []dto.Document{doc(91, 12, "change", true)}}
	m := scoped(t, a, 7, 12, "change")
	require.Contains(t, View(m, 80, 10), "Current: 1 | History: 2")
	m = m.Move(1)
	require.Equal(t, 1, m.Selected)
	a.detail = doc(90, 12, "change", false)
	a.detail.Body = strings.Repeat("long ", 120) + "\x1b[2J\nsecond"
	a.detail.HTML = "<p>server &amp; safe</p>"
	m, cmd := m.BeginDetails(context.Background(), a)
	require.NotNil(t, cmd)
	m, _ = m.Apply(cmd().(Result))
	require.True(t, m.ShowingDetail)
	view := View(m, 30, 8)
	require.Contains(t, view, "historical")
	require.NotContains(t, view, "\x1b[2J")
	require.Contains(t, view, "Raw body:")
	m = m.Scroll(50, 30, 8)
	require.Greater(t, m.Offset, 0)
	require.Contains(t, View(m, 30, 8), "Rendered HTML:")
	m = m.Back()
	require.False(t, m.ShowingDetail)
}

func TestP602DocumentHeadersStayWithinViewportWidth(t *testing.T) {
	longType := strings.Repeat("planning-", 12)
	m := Model{
		ProjectID: 7, OwnerID: 7, OwnerTable: "project",
		Types: []string{longType, "brief", "spec"}, DraftType: longType,
		CatalogErr: errors.New(strings.Repeat("catalog unavailable ", 8)),
		Loaded:     true, Rows: []dto.Document{doc(9, 7, "project", true)},
		Offset: 1,
	}
	const width, height = 24, 4
	view := View(m, width, height)
	lines := strings.Split(view, "\n")
	require.Len(t, lines, height)
	require.Contains(t, view, "> #9")
	for _, line := range lines {
		require.LessOrEqual(t, ansi.StringWidth(line), width, "wrapped viewport line: %q", line)
	}

	m.ShowingDetail, m.DetailLoaded, m.Offset = true, true, 0
	m.Detail = doc(9, 7, "project", true)
	m.Detail.DocType = longType
	for _, line := range strings.Split(View(m, width, height), "\n") {
		require.LessOrEqual(t, ansi.StringWidth(line), width, "wrapped detail header: %q", line)
	}
}

func TestP603DocumentFeatureOwnsTransitionsAndForms(t *testing.T) {
	a := &docAPI{cfg: dto.ProjectConfig{ChangeDocs: []string{"brief", "spec"}}, rows: []dto.Document{}, current: []dto.Document{}}
	m := scoped(t, a, 7, 12, "change")
	require.Equal(t, "brief", m.DraftType)
	m = m.NextType()
	require.Equal(t, "spec", m.DraftType)
	m = m.SetType("missing")
	require.Error(t, m.Err)
	m, cmd := m.BeginInsert(context.Background(), a, " \n ", false)
	require.Nil(t, cmd)
	require.ErrorContains(t, m.Err, "body")
	require.Empty(t, a.inserted)
	m, cmd = m.BeginInsert(context.Background(), a, " /quit\nraw ", false)
	require.NotNil(t, cmd)
	require.True(t, m.Busy)
	require.Contains(t, View(m, 80, 10), "Loading")
	r := cmd().(Result)
	require.Equal(t, 91, r.ID)
	m, _ = m.Apply(r)
	require.Equal(t, "", m.DraftBody)
	require.False(t, m.Busy)
	require.Equal(t, dto.DocumentInput{RefID: 12, RefTable: "change", DocType: "spec", Body: " /quit\nraw ", AgentEdit: false}, a.inserted[0])
}

func TestP603ProvenanceAndLiteralEditorBody(t *testing.T) {
	a := &docAPI{cfg: dto.ProjectConfig{ProjectDocs: []string{"notes"}}, rows: []dto.Document{}, current: []dto.Document{}}
	m := scoped(t, a, 7, 7, "project")
	body := " /cancel\t\x1b[0m\n"
	m, cmd := m.BeginInsert(context.Background(), a, body, true)
	require.NotNil(t, cmd)
	_, _ = m.Apply(cmd().(Result))
	require.True(t, a.inserted[0].AgentEdit)
	require.Equal(t, body, a.inserted[0].Body)
	require.NotContains(t, SafeText(body), "\x1b")
}

func TestP604CommittedInsertSurvivesFailedRefreshAndReadOnlyRetry(t *testing.T) {
	a := &docAPI{cfg: dto.ProjectConfig{ChangeDocs: []string{"spec"}}, rows: []dto.Document{}, current: []dto.Document{}}
	m := scoped(t, a, 7, 12, "change")
	a.listErr = errors.New("offline")
	m, cmd := m.BeginInsert(context.Background(), a, "saved", false)
	m, _ = m.Apply(cmd().(Result))
	require.Equal(t, 91, m.CommittedID)
	require.False(t, m.Loaded)
	require.Contains(t, m.Status, "saved document #91; refreshing")
	require.Len(t, a.inserted, 1)
	m, cmd = m.BeginRefresh(context.Background(), a)
	m, _ = m.Apply(cmd().(Result))
	require.Contains(t, m.Status, "saved document #91; refresh failed")
	m, cmd = m.BeginRefresh(context.Background(), a)
	m, _ = m.Apply(cmd().(Result))
	require.Len(t, a.inserted, 1)
	require.Contains(t, m.Status, "saved document #91")
	a.listErr = nil
	a.rows = []dto.Document{doc(91, 12, "change", true), doc(90, 12, "change", false)}
	a.current = []dto.Document{a.rows[0]}
	a.detail = a.rows[0]
	m, cmd = m.BeginRefresh(context.Background(), a)
	m, _ = m.Apply(cmd().(Result))
	require.True(t, m.Loaded)
	require.Equal(t, 91, m.Rows[m.Selected].ID)
	require.Equal(t, 90, m.Rows[1].ID)
	require.Zero(t, m.CommittedID)
	require.Len(t, a.inserted, 1)
}

func TestP604FailedInsertDraftAndBusyDeduplication(t *testing.T) {
	a := &docAPI{cfg: dto.ProjectConfig{EpicDocs: []string{"story"}}, rows: []dto.Document{}, current: []dto.Document{}, insertErr: errors.New("rejected")}
	m := scoped(t, a, 7, 12, "epic")
	m, cmd := m.BeginInsert(context.Background(), a, "draft", false)
	busy, duplicate := m.BeginInsert(context.Background(), a, "again", false)
	require.Nil(t, duplicate)
	require.Equal(t, "draft", busy.DraftBody)
	m, _ = m.Apply(cmd().(Result))
	require.Equal(t, "draft", m.DraftBody)
	require.ErrorIs(t, m.Err, a.insertErr)
	a.insertErr = nil
	m, cmd = m.BeginInsert(context.Background(), a, m.DraftBody, false)
	m, _ = m.Apply(cmd().(Result))
	require.Len(t, a.inserted, 2)
	require.Equal(t, "draft", a.inserted[1].Body)
}

func TestP604StaleOwnerRevisionAndCancellation(t *testing.T) {
	a := &docAPI{cfg: dto.ProjectConfig{ProjectDocs: []string{"notes"}}, rows: []dto.Document{}, current: []dto.Document{}}
	m, cmd := Model{}.BeginOpen(context.Background(), a, 7, 7, "project")
	old := cmd().(Result)
	m, cmd = m.BeginOpen(context.Background(), a, 8, 8, "project")
	_, ok := m.Apply(old)
	require.False(t, ok)
	m, _ = m.Apply(cmd().(Result))
	require.Equal(t, 8, m.OwnerID)
	m, cmd = m.BeginRefresh(context.Background(), a)
	r := cmd().(Result)
	m = m.Invalidate()
	require.ErrorIs(t, a.ctx.Err(), context.Canceled)
	_, ok = m.Apply(r)
	require.False(t, ok)
	require.False(t, m.Loaded)
}

func TestP604HistorySelectionAfterAppendAndReloadSafety(t *testing.T) {
	a := &docAPI{cfg: dto.ProjectConfig{ChangeDocs: []string{"spec"}}, rows: []dto.Document{doc(90, 12, "change", true)}, current: []dto.Document{doc(90, 12, "change", true)}}
	m := scoped(t, a, 7, 12, "change")
	m, cmd := m.BeginRefresh(context.Background(), a)
	require.False(t, m.Loaded)
	m, detail := m.BeginDetails(context.Background(), a)
	require.Nil(t, detail)
	require.Error(t, m.Err)
	m, _ = m.Apply(cmd().(Result))
	a.rows = []dto.Document{doc(91, 12, "change", true), doc(90, 12, "change", false)}
	a.current = []dto.Document{a.rows[0]}
	a.detail = a.rows[0]
	m, cmd = m.BeginInsert(context.Background(), a, "next", false)
	m, _ = m.Apply(cmd().(Result))
	require.Equal(t, 91, m.CommittedID)
	m, cmd = m.BeginRefresh(context.Background(), a)
	m, _ = m.Apply(cmd().(Result))
	require.Equal(t, 91, m.Rows[m.Selected].ID)
	m = m.Move(1)
	require.Equal(t, 90, m.Rows[m.Selected].ID)
	m, cmd = m.BeginRefresh(context.Background(), a)
	m, _ = m.Apply(cmd().(Result))
	require.Equal(t, 90, m.Rows[m.Selected].ID)
}

func TestP605DocumentCapabilitySupportsAgentProvenanceWithoutRunningAgent(t *testing.T) {
	a := &docAPI{cfg: dto.ProjectConfig{ChangeDocs: []string{"brief"}}, rows: []dto.Document{}, current: []dto.Document{}}
	m := scoped(t, a, 7, 12, "change")
	m, cmd := m.BeginInsert(context.Background(), a, "agent produced text", true)
	r := cmd().(Result)
	require.Nil(t, r.Err)
	require.True(t, a.inserted[0].AgentEdit)
	require.Equal(t, "brief", a.inserted[0].DocType)
	m, _ = m.Apply(r)
	require.Contains(t, m.Status, fmt.Sprint(r.ID))
}

func TestP602CatalogFailureStillReadsHistoryAndShowsReadErrors(t *testing.T) {
	a := &docAPI{cfgErr: errors.New("catalog offline"), rows: []dto.Document{doc(9, 12, "epic", false)}, current: []dto.Document{}}
	m := scoped(t, a, 7, 12, "epic")
	require.True(t, m.Loaded)
	require.ErrorIs(t, m.CatalogErr, a.cfgErr)
	require.Contains(t, View(m, 80, 10), "Catalog error: catalog offline")
	require.Contains(t, View(m, 80, 10), "#9")
	m = m.NextType()
	require.ErrorContains(t, m.Err, "no document types")
	m, cmd := m.BeginInsert(context.Background(), a, "body", false)
	require.Nil(t, cmd)
	require.Empty(t, a.inserted)
	a.currentErr = errors.New("current offline")
	m, cmd = m.BeginRefresh(context.Background(), a)
	require.NotNil(t, cmd)
	m, _ = m.Apply(cmd().(Result))
	require.False(t, m.Loaded)
	require.Contains(t, View(m, 80, 10), "Use /retry")
}

func TestP603InvalidScopeAndWrongOwnerDetailsNeverAct(t *testing.T) {
	a := &docAPI{cfg: dto.ProjectConfig{ChangeDocs: []string{"spec"}}, rows: []dto.Document{doc(9, 12, "change", true)}, current: []dto.Document{doc(9, 12, "change", true)}}
	for _, scope := range []struct {
		project, owner int
		table          string
	}{{0, 12, "change"}, {7, 0, "change"}, {7, 12, "task"}, {7, 8, "project"}} {
		m, cmd := Model{}.BeginOpen(context.Background(), a, scope.project, scope.owner, scope.table)
		require.Nil(t, cmd)
		require.Error(t, m.Err)
	}
	m := scoped(t, a, 7, 12, "change")
	a.detail = doc(9, 13, "change", true)
	m, cmd := m.BeginDetails(context.Background(), a)
	m, _ = m.Apply(cmd().(Result))
	require.ErrorContains(t, m.Err, "different owner")
	require.False(t, m.Loaded)
	require.False(t, m.DetailLoaded)
	m, cmd = m.BeginInsert(context.Background(), a, "body", false)
	require.NotNil(t, cmd)
	m = m.Invalidate()
	_, ok := m.Apply(cmd().(Result))
	require.False(t, ok)
	require.Empty(t, a.inserted)
}

func TestP602DocumentPresentationModesAndSafeViewport(t *testing.T) {
	a := &docAPI{cfg: dto.ProjectConfig{ProjectDocs: []string{"notes"}}, rows: []dto.Document{doc(9, 7, "project", true)}, current: []dto.Document{doc(9, 7, "project", true)}}
	m := scoped(t, a, 7, 7, "project")
	require.Empty(t, View(m, 0, 0))
	require.Contains(t, View(m, 80, 10), "current human")
	a.detail = doc(9, 7, "project", true)
	a.detail.AgentEdit = true
	a.detail.Body = "body\x1b[2J\u202e"
	a.detail.HTML = "<p>rendered</p>"
	m, cmd := m.BeginDetails(context.Background(), a)
	m, _ = m.Apply(cmd().(Result))
	require.Contains(t, Help(m), "history")
	view := View(m, 80, 15)
	require.Contains(t, view, "current | agent")
	require.Contains(t, view, "Rendered HTML:")
	require.Contains(t, view, "<p>rendered</p>")
	require.NotContains(t, view, "\x1b[2J")
	require.NotContains(t, view, "\u202e")
	require.Equal(t, `line\n\x1b[2J`, SafeLine("line\n\x1b[2J"))
	m.DetailLoaded = false
	require.Contains(t, View(m, 80, 10), "details not loaded")
	m = m.Back()
	require.Contains(t, Help(m), "select version")
}

func TestP604CommittedDetailsRefreshRequiresVisibleMatchingVersion(t *testing.T) {
	a := &docAPI{cfg: dto.ProjectConfig{ProjectDocs: []string{"notes"}}, rows: []dto.Document{}, current: []dto.Document{}}
	m := scoped(t, a, 7, 7, "project")
	m, cmd := m.BeginInsert(context.Background(), a, "committed", false)
	m, _ = m.Apply(cmd().(Result))
	require.Equal(t, 91, m.CommittedID)
	m, cmd = m.BeginRefresh(context.Background(), a)
	m, _ = m.Apply(cmd().(Result))
	require.ErrorContains(t, m.Err, "missing from refreshed history")
	require.Zero(t, a.detailCalls)
	require.Equal(t, 91, m.CommittedID)
	a.rows = []dto.Document{doc(91, 7, "project", true)}
	a.current = a.rows
	a.detail = doc(91, 8, "project", true)
	m, cmd = m.BeginRefresh(context.Background(), a)
	m, _ = m.Apply(cmd().(Result))
	require.ErrorContains(t, m.Err, "different owner")
	require.Equal(t, 1, a.detailCalls)
	require.Equal(t, 91, m.CommittedID)
	a.detail = a.rows[0]
	m, cmd = m.BeginRefresh(context.Background(), a)
	m, _ = m.Apply(cmd().(Result))
	require.True(t, m.Loaded)
	require.Equal(t, 91, m.Detail.ID)
	require.Zero(t, m.CommittedID)
}

func TestP603InvalidOwnerCancellationAndUnconfirmedIDCannotAppend(t *testing.T) {
	a := &docAPI{cfg: dto.ProjectConfig{EpicDocs: []string{"notes"}}, rows: []dto.Document{}, current: []dto.Document{}}
	invalid := Model{ProjectID: 7, OwnerID: 0, OwnerTable: "epic", Types: []string{"notes"}, DraftType: "notes"}
	invalid, cmd := invalid.BeginInsert(context.Background(), a, "body", false)
	require.Nil(t, cmd)
	require.ErrorContains(t, invalid.Err, "owner")
	invalid, cmd = invalid.BeginRefresh(context.Background(), a)
	require.Nil(t, cmd)
	require.ErrorContains(t, invalid.Err, "owner")
	require.Empty(t, a.inserted)
	m := scoped(t, a, 7, 12, "epic")
	m, cmd = m.BeginInsert(context.Background(), a, "canceled", false)
	_ = m.Invalidate()
	r := cmd().(Result)
	require.ErrorIs(t, r.Err, context.Canceled)
	require.Empty(t, a.inserted)
	m = scoped(t, a, 7, 12, "epic")
	m, cmd = m.BeginInsert(context.Background(), a, "unknown", false)
	r = cmd().(Result)
	r.ID = 0
	m, _ = m.Apply(r)
	require.ErrorContains(t, m.Err, "no committed ID")
	require.Zero(t, m.CommittedID)
}

func TestP602AgentHistoryRowDisplaysProvenance(t *testing.T) {
	row := doc(9, 7, "project", false)
	row.AgentEdit = true
	a := &docAPI{cfg: dto.ProjectConfig{ProjectDocs: []string{"notes"}}, rows: []dto.Document{row}, current: []dto.Document{}}
	m := scoped(t, a, 7, 7, "project")
	require.Contains(t, View(m, 80, 10), "history agent")
}
