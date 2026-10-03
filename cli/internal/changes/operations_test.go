package changes

import (
	"cli/internal/dto"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/require"
)

func TestP405LongSavedValuesKeepBoundedFeedbackAndFullDetail(t *testing.T) {
	for _, op := range []Operation{Title, PRURL, Phase, Types} {
		t.Run(string(op), func(t *testing.T) {
			m, api := changeSetup()
			api.fail = "details"
			value := "https://example.test/" + strings.Repeat("界\n\t", 1000)
			in := Input{Value: value, Types: []string{value}}
			catalog := changeCatalog()
			catalog.ChangePhases = append(catalog.ChangePhases, value)
			catalog.ChangeTypes = append(catalog.ChangeTypes, value)
			if op == PRURL {
				value = "https://example.test/" + strings.Repeat("path", 1000)
				in.Value = value
			}
			m, cmd := m.Begin(context.Background(), api, api, op, 7, 12, in, catalog)
			m = finish(t, m, cmd)
			switch op {
			case Title:
				require.Equal(t, value, m.Detail.Title)
			case PRURL:
				require.Equal(t, value, m.Detail.PRUrl)
			case Phase:
				require.Equal(t, value, m.Detail.ChangePhase)
			case Types:
				require.Equal(t, []string{value}, m.Detail.ChangeTypes)
			}
			require.Contains(t, m.Status, "saved "+string(op))
			require.Contains(t, m.Status, "…\" on change #12")
			require.Contains(t, m.Status, "/retry reads only")
			require.NotContains(t, m.Status, "\n")
			require.Less(t, ansi.StringWidth(m.Status), 120)
			m, cmd = m.Begin(context.Background(), api, api, Details, 7, 12, Input{}, catalog)
			m = finish(t, m, cmd)
			require.Contains(t, m.Status, "saved "+string(op))
			require.Contains(t, m.Status, "refresh failed")
			require.Less(t, ansi.StringWidth(m.Status), 120)
			require.Equal(t, []string{string(op), "details", "details"}, api.calls)
		})
	}
}

type changeAPI struct {
	cancelAfter string
	cancel      context.CancelFunc
	calls       []string
	fail        string
	value       dto.Change
	context     context.Context
}

func (a *changeAPI) call(ctx context.Context, s string) error {
	a.context = ctx
	a.calls = append(a.calls, s)
	if s == a.cancelAfter {
		a.cancel()
		return nil
	}
	if a.fail == s {
		return errors.New(s + " failed")
	}
	return ctx.Err()
}

func (a *changeAPI) ListChangeRows(ctx context.Context, _ int) ([]dto.Change, error) {
	return []dto.Change{a.value}, a.call(ctx, "list")
}

func (a *changeAPI) GetChange(ctx context.Context, _ int) (dto.Change, error) {
	return a.value, a.call(ctx, "details")
}

func (a *changeAPI) CreateChange(ctx context.Context, _ dto.ChangeCreateInput) (int, error) {
	return 12, a.call(ctx, "create")
}
func (a *changeAPI) DeleteChange(ctx context.Context, _ int) error { return a.call(ctx, "delete") }
func (a *changeAPI) UpdateChangeTitle(ctx context.Context, _ int, v string) error {
	a.value.Title = v
	return a.call(ctx, "title")
}

func (a *changeAPI) UpdateChangeSlug(ctx context.Context, _ int, v string) error {
	refSlug := "123-" + v
	a.value.RefSlug = &refSlug
	return a.call(ctx, "slug")
}

func (a *changeAPI) UpdateChangePhase(ctx context.Context, _ int, v string) error {
	a.value.ChangePhase = v
	return a.call(ctx, "phase")
}

func (a *changeAPI) UpdateChangeTypes(ctx context.Context, _ int, v []string) error {
	a.value.ChangeTypes = v
	return a.call(ctx, "types")
}

func (a *changeAPI) UpdateChangeEpic(ctx context.Context, _ int, v *int) error {
	a.value.EpicID = v
	return a.call(ctx, "epic")
}

func (a *changeAPI) UpdateChangeAfterChange(ctx context.Context, _ int, v *int) error {
	a.value.AfterChangeID = v
	return a.call(ctx, "after-change")
}

func (a *changeAPI) UpdateChangeActive(ctx context.Context, _ int, v bool) error {
	a.value.Active = v
	return a.call(ctx, "active")
}

func (a *changeAPI) UpdateChangePRUrl(ctx context.Context, _ int, v string) error {
	a.value.PRUrl = v
	return a.call(ctx, "pr-url")
}

func (a *changeAPI) ListTestCases(ctx context.Context, _ int) ([]dto.TestCase, error) {
	return []dto.TestCase{{ID: 31, Scenario: "Separate"}}, a.call(ctx, "testcases")
}

func (a *changeAPI) Load(ctx context.Context, _ int) ([]dto.Document, error) {
	return []dto.Document{{DocType: "brief", Body: "Original"}}, a.call(ctx, "documents")
}

func (a *changeAPI) Save(ctx context.Context, _ int, _, _ string) (int, error) {
	return 91, a.call(ctx, "document")
}

func changeCatalog() dto.ProjectConfig {
	return dto.ProjectConfig{ChangePhases: []string{"review", "backlog"}, ChangeTypes: []string{"fix", "feature"}, ChangeDocs: []string{"brief", "spec", "pr"}}
}

func changeSetup() (Model, *changeAPI) {
	a := &changeAPI{value: dto.Change{ID: 12, ProjectID: 7, Title: "Original", Active: true, Completed: 73, DoneTC: 2, TotalTC: 9}}
	return Model{ProjectID: 7, Detail: Present(a.value), DetailLoaded: true}, a
}

func finish(t *testing.T, m Model, cmd tea.Cmd) Model {
	t.Helper()
	require.NotNil(t, cmd)
	m, ok := m.Apply(cmd().(Result))
	require.True(t, ok)
	return m
}

func TestP403EachChangeActionAndValidation(t *testing.T) {
	assoc := 4
	for _, tt := range []struct {
		op Operation
		in Input
	}{
		{List, Input{}}, {Details, Input{}}, {Create, Input{Title: "Explicit", Value: "plain brief"}}, {Delete, Input{}}, {Title, Input{Value: "/save"}}, {Phase, Input{Value: "review"}}, {Types, Input{Types: []string{}}}, {Epic, Input{Association: &assoc}}, {Epic, Input{}}, {AfterChange, Input{Association: &assoc}}, {AfterChange, Input{}}, {Active, Input{Active: false}}, {PRURL, Input{Value: "https://host/path"}}, {Document, Input{DocumentType: "spec", Value: "new\tbytes\n"}},
	} {
		t.Run(string(tt.op), func(t *testing.T) {
			m, a := changeSetup()
			if tt.op == AfterChange && tt.in.Association == nil {
				m.Detail.AfterChangeID = "4"
			}
			m, cmd := m.Begin(context.Background(), a, a, tt.op, 7, 12, tt.in, changeCatalog())
			m = finish(t, m, cmd)
			require.NoError(t, m.Err)
			require.False(t, m.Busy)
			require.Contains(t, a.calls, string(tt.op))
			if tt.op != List && tt.op != Details && tt.op != Create {
				require.Contains(t, m.Status, map[bool]string{true: "deleted", false: "saved"}[tt.op == Delete])
				if tt.op == Create {
					require.Contains(t, m.Status, "created change #12")
				}
			}
		})
	}
}

func TestP403InvalidFormsAndAbsentCatalogsNeverWrite(t *testing.T) {
	invalid := -1
	for _, tt := range []struct {
		op      Operation
		in      Input
		catalog dto.ProjectConfig
	}{
		{Create, Input{Value: "brief"}, changeCatalog()},
		{Create, Input{Title: "Title"}, changeCatalog()},
		{Create, Input{Title: "Title", Value: "plain", UUID: "bad"}, changeCatalog()},
		{Create, Input{Title: "Title", Value: "plain"}, dto.ProjectConfig{}},
		{Title, Input{Value: " "}, changeCatalog()},
		{Phase, Input{Value: "invented"}, changeCatalog()},
		{Types, Input{Types: []string{"missing"}}, changeCatalog()},
		{Epic, Input{Association: &invalid}, changeCatalog()},
		{AfterChange, Input{Association: &invalid}, changeCatalog()},
		{PRURL, Input{Value: ""}, changeCatalog()},
		{PRURL, Input{Value: "ftp://host/path"}, changeCatalog()},
		{Document, Input{DocumentType: "missing"}, changeCatalog()},
	} {
		m, a := changeSetup()
		m, cmd := m.Begin(context.Background(), a, a, tt.op, 7, 12, tt.in, tt.catalog)
		require.Nil(t, cmd)
		require.Error(t, m.Err)
		require.Empty(t, a.calls)
	}
	for _, in := range []Input{{Title: "Title", Value: "plain"}, {Title: "Title", Value: "plain", UUID: "0198a86f-9b8a-7d89-ae5b-6f25b528b04c"}} {
		require.NoError(t, validate(Create, 7, 0, in, changeCatalog()))
	}
	for _, value := range []string{"HTTP://host/path", "https://host/path"} {
		require.NoError(t, validate(PRURL, 7, 12, Input{Value: value}, changeCatalog()))
	}
}

func TestP405CommittedStepsSurviveLaterFailureAndRetryOnlyReads(t *testing.T) {
	for _, tt := range []struct {
		op   Operation
		in   Input
		fail string
		step string
	}{
		{Create, Input{Title: "New", Value: "brief"}, "details", "created change #12"},
		{Create, Input{Title: "New", Value: "Types: feature\n\nbrief"}, "types", "created change #12"},
		{Title, Input{Value: "/save"}, "details", "saved title"},
		{Delete, Input{}, "list", "deleted change #12"},
		{Document, Input{DocumentType: "brief", Value: "new\tbytes"}, "documents", "saved brief document #91"},
		{Document, Input{DocumentType: "brief", Value: "Types: feature\nnew\tbytes"}, "types", "saved brief document #91"},
	} {
		t.Run(string(tt.op)+tt.fail, func(t *testing.T) {
			m, a := changeSetup()
			a.fail = tt.fail
			m, cmd := m.Begin(context.Background(), a, a, tt.op, 7, 12, tt.in, changeCatalog())
			_, duplicate := m.Begin(context.Background(), a, a, tt.op, 7, 12, tt.in, changeCatalog())
			require.Nil(t, duplicate)
			m = finish(t, m, cmd)
			require.Error(t, m.Err)
			require.Contains(t, m.Status, tt.step)
			require.Contains(t, m.Status, "/retry reads only")
			if tt.op == Title {
				require.Equal(t, "/save", m.Detail.Title)
			}
			if tt.op == Document {
				require.Equal(t, tt.in.Value, m.Detail.Brief)
			}
			before := len(a.calls)
			op, id := Details, 12
			if tt.op == Delete {
				op, id = List, 0
			}
			a.fail = map[bool]string{true: "list", false: "details"}[op == List]
			m, cmd = m.Begin(context.Background(), a, a, op, 7, id, Input{}, changeCatalog())
			m = finish(t, m, cmd)
			require.Contains(t, m.Status, tt.step)
			a.fail = ""
			m, cmd = m.Begin(context.Background(), a, a, op, 7, id, Input{}, changeCatalog())
			m = finish(t, m, cmd)
			require.NoError(t, m.Err)
			require.Empty(t, m.Outcome)
			for _, call := range a.calls[before:] {
				require.NotContains(t, []string{"create", "title", "delete", "document", "types"}, call)
			}
			a.fail = "details"
			m, cmd = m.Begin(context.Background(), a, a, Details, 7, 99, Input{}, changeCatalog())
			m = finish(t, m, cmd)
			require.NotContains(t, m.Status, tt.step)
		})
	}
}

type savedSlotDocuments struct {
	api *changeAPI
	row dto.Document
}

func (d *savedSlotDocuments) Save(ctx context.Context, id int, kind, body string) (int, error) {
	d.row.ID, d.row.RefID, d.row.RefTable = 91, id, "change"
	d.row.DocType, d.row.Body = kind, body
	return d.row.ID, d.api.call(ctx, "document")
}

func (d *savedSlotDocuments) Load(ctx context.Context, _ int) ([]dto.Document, error) {
	return []dto.Document{d.row}, d.api.call(ctx, "documents")
}

func Test031DocumentSlotTimestampUnavailableUntilRefresh(t *testing.T) {
	oldTime := time.Date(2026, 9, 28, 10, 0, 0, 0, time.UTC)
	newTime := oldTime.Add(48 * time.Hour)
	for _, existing := range []bool{false, true} {
		for _, failure := range []string{"types", "details", "documents", "testcases"} {
			t.Run(fmt.Sprintf("existing=%t/failure=%s", existing, failure), func(t *testing.T) {
				m, api := changeSetup()
				m.Detail.DocumentTypes = []string{"spec"}
				if existing {
					m.Detail.Spec = "old body"
					m.Detail.Documents = []dto.Document{{ID: 9, RefID: 12, RefTable: "change", DocType: "spec", Body: "old body", UpdatedAt: oldTime}}
				}
				prior := m.Detail.Documents
				docs := &savedSlotDocuments{api: api, row: dto.Document{UpdatedAt: newTime}}
				api.fail = failure
				input := Input{DocumentType: "spec", Value: "Types: feature\nnew body"}
				m, cmd := m.Begin(context.Background(), api, docs, Document, 7, 12, input, changeCatalog())
				m = finish(t, m, cmd)
				require.Error(t, m.Err)
				require.Contains(t, m.Status, "saved spec document #91")
				require.Contains(t, m.Status, "/retry reads only")
				require.Len(t, m.Detail.Documents, 1)
				require.Equal(t, 91, m.Detail.Documents[0].ID)
				require.Equal(t, input.Value, m.Detail.Documents[0].Body)
				require.True(t, m.Detail.Documents[0].UpdatedAt.IsZero())
				if existing {
					require.Equal(t, 9, prior[0].ID)
					require.Equal(t, oldTime, prior[0].UpdatedAt)
				}
				for _, retryFails := range []bool{true, false} {
					m.Detail.DocumentTypes = []string{"spec"} // The shell restores the project catalog after reads.
					var slot DetailRow
					for _, row := range DetailRows(m.Detail) {
						if row.DocumentType == "spec" {
							slot = row
						}
					}
					require.Equal(t, 91, slot.DocumentID)
					require.Equal(t, "spec [✓]", slot.Text)
					api.fail = ""
					if retryFails {
						api.fail = "details"
					}
					before := len(api.calls)
					m, cmd = m.Begin(context.Background(), api, docs, Details, 7, 12, Input{}, changeCatalog())
					m = finish(t, m, cmd)
					for _, call := range api.calls[before:] {
						require.Contains(t, []string{"details", "documents", "testcases"}, call)
					}
				}
				require.NoError(t, m.Err)
				require.Equal(t, newTime, m.Detail.Documents[0].UpdatedAt)
				m.Detail.DocumentTypes = []string{"spec"}
				for _, row := range DetailRows(m.Detail) {
					if row.DocumentType == "spec" {
						require.Equal(t, "spec [✓] "+newTime.Local().Format("2006-01-02 15:04"), row.Text)
					}
				}
			})
		}
	}
}

func TestP404StaleResultsCanceledWorkAndInvisibleRows(t *testing.T) {
	m, a := changeSetup()
	m.Rows = []dto.ChangeView{m.Detail}
	m, cmd := m.Begin(context.Background(), a, a, List, 7, 0, Input{}, changeCatalog())
	require.Empty(t, m.Rows)
	_, _, ok := m.SelectDetail(Filters{})
	require.False(t, ok)
	pending := cmd()
	old := m
	m = m.Invalidate()
	_, ok = m.Apply(pending.(Result))
	require.False(t, ok)
	m = m.Scope(8)
	_, ok = m.Apply(pending.(Result))
	require.False(t, ok)
	m = old
	m, cmd = m.Begin(context.Background(), a, a, Details, 7, 12, Input{}, changeCatalog())
	m = m.Invalidate()
	r := cmd().(Result)
	require.ErrorIs(t, r.Err, context.Canceled)
	_, ok = m.Apply(r)
	require.False(t, ok)
	ctx, cancel := context.WithCancel(context.Background())
	m, a = changeSetup()
	m, cmd = m.Begin(ctx, a, a, Title, 7, 12, Input{Value: "changed"}, changeCatalog())
	cancel()
	m = finish(t, m, cmd)
	require.ErrorIs(t, m.Err, context.Canceled)
	require.NotContains(t, m.Status, "saved")
}

func TestP402EveryReturnedFieldAndLiteralNoOp(t *testing.T) {
	m, a := changeSetup()
	refSlug := "123-server-identity"
	epic := 3
	name := "Epic"
	after := 4
	a.value.RefSlug = &refSlug
	a.value.EpicID = &epic
	a.value.EpicName = &name
	a.value.AfterChangeID = &after
	a.value.RefUUID = "uuid"
	a.value.PRUrl = "https://host/pr"
	a.value.Title = strings.Repeat("line\n", 40)
	view := Present(a.value)
	require.Equal(t, "123", view.Ref)
	require.Equal(t, "123-server-identity", view.RefSlug)
	require.Equal(t, "3", view.EpicID)
	require.Equal(t, "4", view.AfterChangeID)
	require.Equal(t, int64(73), view.Completed)
	labels := map[string]bool{}
	for _, row := range append(fixedDetailRows(view), DetailRows(view)...) {
		labels[row.Label] = true
	}
	for _, label := range []string{"ID", "Ref UUID", "Slug", "Epic", "After Change", "Phase", "Types", "Title", "Active", "Complete", "Comments", "Modified", "PR URL"} {
		require.True(t, labels[label], label)
	}
	for _, value := range []string{"/save", "/cancel", "/editor", "/return"} {
		m.Detail.Title = value
		m, cmd := m.Begin(context.Background(), a, a, Title, 7, 12, Input{Value: value}, changeCatalog())
		require.Nil(t, cmd)
		require.Equal(t, "unchanged", m.Status)
	}
}

func TestPresentUsesPaddedRefSlugForReferenceAndSlugRow(t *testing.T) {
	refSlug := "006-some-slug"
	view := Present(dto.Change{ID: 12, RefSlug: &refSlug})
	require.Equal(t, "006", view.Ref)
	require.Equal(t, refSlug, view.RefSlug)
	rows := DetailRows(view)
	require.Equal(t, refSlug, rows[0].Text)
	require.Equal(t, "-", rows[4].Text)
}

func TestP403NullableAssociationForm(t *testing.T) {
	for _, value := range []string{"", "null", " null "} {
		id, err := AssociationInput(value)
		require.NoError(t, err)
		require.Nil(t, id)
	}
	id, err := AssociationInput(" 4 ")
	require.NoError(t, err)
	require.Equal(t, 4, *id)
	for _, value := range []string{"/save", "-1", "0", "1.5"} {
		_, err := AssociationInput(value)
		require.Error(t, err)
	}
}

func TestP405CancellationAfterCommitRetainsStepAndLiteralValue(t *testing.T) {
	for _, op := range []Operation{Create, Title, Document} {
		t.Run(string(op), func(t *testing.T) {
			m, a := changeSetup()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			a.cancelAfter = string(op)
			a.cancel = cancel
			in := Input{Title: "Comments", Value: "literal\tbytes\n", DocumentType: "brief"}
			m, cmd := m.Begin(ctx, a, a, op, 7, 12, in, changeCatalog())
			m = finish(t, m, cmd)
			require.ErrorIs(t, m.Err, context.Canceled)
			require.NotContains(t, m.Status, "save failed")
			require.Contains(t, m.Status, "/retry reads only")
			require.Equal(t, "12", m.Detail.ID)
			switch op {
			case Title:
				require.Equal(t, in.Value, m.Detail.Title)
			case Create, Document:
				require.Equal(t, in.Value, m.Detail.Brief)
			}
			before := len(a.calls)
			m, cmd = m.Begin(context.Background(), a, a, Details, 7, 12, Input{}, changeCatalog())
			m = finish(t, m, cmd)
			require.NoError(t, m.Err)
			require.Equal(t, []string{"details", "documents", "testcases"}, a.calls[before:])
		})
	}
}

func (a *changeAPI) ListInactiveChanges(ctx context.Context, id int) ([]dto.Change, error) {
	return a.ListChangeRows(ctx, id)
}

type inactiveDeleteAPI struct{ *changeAPI }

func (a inactiveDeleteAPI) ListInactiveChanges(ctx context.Context, _ int) ([]dto.Change, error) {
	return []dto.Change{{ID: 13, ProjectID: 7, Active: false}}, a.call(ctx, "list-inactive")
}

func Test031DeleteRefreshPreservesListModeAndReadOnlyRecovery(t *testing.T) {
	for _, inactive := range []bool{false, true} {
		for _, refreshFails := range []bool{false, true} {
			t.Run(fmt.Sprintf("inactive=%t/failure=%t", inactive, refreshFails), func(t *testing.T) {
				m, base := changeSetup()
				api := inactiveDeleteAPI{base}
				m.Inactive = inactive
				m.Rows = []dto.ChangeView{{ID: "12"}, {ID: "13", Active: false}}
				endpoint := "list"
				if inactive {
					endpoint = "list-inactive"
				}
				if refreshFails {
					base.fail = endpoint
				}
				m, cmd := m.Begin(context.Background(), api, base, Delete, 7, 12, Input{}, changeCatalog())
				m = finish(t, m, cmd)
				require.Equal(t, []string{"delete", endpoint}, base.calls)
				require.Equal(t, inactive, m.Inactive)
				require.Contains(t, m.Status, "deleted change #12")
				if refreshFails {
					require.ErrorContains(t, m.Err, endpoint+" failed")
					require.Contains(t, m.Status, "/retry reads only")
					require.Equal(t, []dto.ChangeView{{ID: "13", Active: false}}, m.Rows)
					base.fail = ""
					m, cmd = m.Begin(context.Background(), api, base, List, 7, 0, Input{}, changeCatalog())
					m = finish(t, m, cmd)
					require.Equal(t, []string{"delete", endpoint, endpoint}, base.calls)
				}
				require.NoError(t, m.Err)
				if inactive {
					require.Equal(t, "13", m.Rows[0].ID)
					require.False(t, m.Rows[0].Active)
				}
			})
		}
	}
}
