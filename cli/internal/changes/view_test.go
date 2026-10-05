package changes

import (
	"cli/internal/dto"
	"cli/internal/styles"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func TestNoMatchingChangesKeepBoxHeaderAndRedMessage(t *testing.T) {
	red := lipgloss.NewStyle().Foreground(styles.AccentRed).Render("x")
	redPrefix := strings.SplitN(red, "x", 2)[0]
	for _, rows := range [][]dto.ChangeView{
		{{Title: "Existing change"}},
		{},
	} {
		view := TableView(Model{Rows: rows}, Filters{Find: "missing"}, 80, 3)
		lines := strings.Split(stripANSI(view), "\n")
		require.GreaterOrEqual(t, len(lines), 6)
		assert.Contains(t, lines[0], "┌")
		assert.Contains(t, lines[1], "#Ref")
		assert.Contains(t, lines[1], "Title")
		assert.Contains(t, lines[2], "No changes match filters.")
		assert.Equal(t, strings.Index(lines[1], "Title"), strings.Index(lines[2], "No changes match filters."))
		assert.Contains(t, view, redPrefix+"No changes match filters.")
		assert.Contains(t, view, "Rows 0-0 of 0")
		assert.Contains(t, lines[len(lines)-1], "┘")
		for _, line := range strings.Split(view, "\n") {
			assert.LessOrEqual(t, lipgloss.Width(line), 80)
		}
	}
	assert.Contains(t, TableView(Model{}, Filters{}, 80, 3), "No changes.")
}

func TestFindHighlightsOverrideSelectedRowBackground(t *testing.T) {
	assert.Equal(t, lipgloss.Color("#5F5F87"), styles.MutedPurple)
	assert.Equal(t, lipgloss.Color("#875F5F"), styles.MutedRed)
	assert.Equal(t, lipgloss.Color("#5F875F"), styles.MutedGreen)
	assert.Equal(t, fmt.Sprint(styles.MutedPurple), fmt.Sprint(styles.Default.Selection.GetBackground()))

	const typesWidth, epicWidth, titleWidth = 12, 14, 32
	queryWords := normalizedFindWords("FEA")
	matchStyle := lipgloss.NewStyle().Foreground(styles.Foreground).Background(styles.MutedGreen)
	for _, selected := range []bool{false, true} {
		line := changeTableRowLine("FEA-9", "Feature", "feature", "Feature Epic", "---===Feature===--- Feature", "1", "2", "50", "2026-09-29", typesWidth, epicWidth, titleWidth, selected, nil, queryWords)
		assert.Equal(t, 1, strings.Count(line, matchStyle.Render("FEA")))
		assert.Equal(t, 1, strings.Count(line, matchStyle.Render("fea")))
		assert.Equal(t, 4, strings.Count(line, matchStyle.Render("Fea")))
		assert.Equal(t, changeTableContentWidth(typesWidth, epicWidth, titleWidth), lipgloss.Width(line))
		assert.Contains(t, stripANSI(line), "---===Feature===--- Feature")
	}
	view := TableView(Model{Rows: []dto.ChangeView{{Ref: "FEA-9", ChangePhase: "Feature", ChangeTypes: []string{"feature"}, EpicName: "Feature Epic", Title: "---===Feature===--- Feature"}}}, Filters{Find: "fea"}, 160, 1)
	assert.Contains(t, view, matchStyle.Render("fea"))
	assert.Contains(t, view, matchStyle.Render("Fea"))
}

func TestTypesAndCompletionUseRequestedColors(t *testing.T) {
	const typesWidth, epicWidth, titleWidth = 12, 10, 20
	header := changeTableHeaderLine(typesWidth, epicWidth, titleWidth)
	assert.Contains(t, header, lipgloss.NewStyle().Foreground(styles.AccentPurple).Render("Types"))
	assert.Contains(t, header, lipgloss.NewStyle().Foreground(styles.AccentBlue).Render("%"))
	for _, selected := range []bool{false, true} {
		line := changeTableRowLine("006", "backlog", "feature", "Epic", "Title", "1", "2", "50", "2026-09-29", typesWidth, epicWidth, titleWidth, selected, nil, nil)
		typesStyle := lipgloss.NewStyle().Foreground(styles.AccentPurple)
		completeStyle := lipgloss.NewStyle().Foreground(styles.AccentBlue)
		if selected {
			typesStyle = typesStyle.Background(styles.MutedPurple)
			completeStyle = completeStyle.Background(styles.MutedPurple)
		}
		assert.Contains(t, line, typesStyle.Render(fmt.Sprintf("%-*s", typesWidth, "feature")))
		assert.Contains(t, line, completeStyle.Render(" 50"))
	}
	for _, selected := range []bool{false, true} {
		style := detailValueStyle
		if selected {
			style = detailSelectedStyle
		}
		assert.Equal(t, styles.AccentPurple, style(DetailRow{Label: "Types"}, nil).GetForeground())
		assert.Equal(t, styles.AccentBlue, style(DetailRow{Label: "Complete"}, nil).GetForeground())
	}
}

func TestDetailsViewSeparatesSpecAndTestCases(t *testing.T) {
	model := Model{}.WithDetail(dto.ChangeView{ID: "12", RefUUID: "uuid", Title: "Change", DocumentTypes: []string{"brief", "spec"}, Documents: []dto.Document{{ID: 9, DocType: "spec", Body: "Spec text"}}, TestCases: []dto.TestCase{{ID: 31, Scenario: "first scenario", Done: true}}})
	view := stripANSI(DetailsView(model, 120, 30))
	assert.Contains(t, view, "[✓] spec")
	assert.NotContains(t, view, "Spec text")
	assert.Contains(t, view, "first scenario (#31)")
	assert.Contains(t, view, "Docs")
	assert.Contains(t, view, "Comments")
	assert.Greater(t, strings.Index(view, "first scenario (#31)"), strings.Index(view, "Docs"))
}

func TestDetailsViewEmojiRowsDoNotOverflowSelectionWidth(t *testing.T) {
	change := dto.ChangeView{
		ID:      "12",
		Ref:     "3",
		Title:   "Backend Change",
		Spec:    "Spec text",
		Active:  true,
		Created: "2026-06-29T08:15:00Z",
		TestCases: []dto.TestCase{
			{ID: 31, Scenario: "first scenario", Done: true},
			{ID: 32, Scenario: "second scenario", Done: false},
		},
	}

	for _, selected := range []int{7, 13} {
		view := DetailsView(Model{Detail: change, DetailSelected: selected}, 120, 20)
		for _, line := range strings.Split(view, "\n") {
			assert.LessOrEqual(t, lipgloss.Width(line), 120)
		}
	}
}

func TestDetailsViewRendersUnassignedRefAsBlank(t *testing.T) {
	model := Model{}.WithDetail(dto.ChangeView{
		ID:    "201",
		Title: "Unreferenced Change",
	})

	view := stripANSI(DetailsView(model, 120, 20))

	assert.Contains(t, view, "ID │ 201")
	assert.Contains(t, view, "Ref UUID │")
	assert.Contains(t, view, "Slug │")
	assert.NotContains(t, view, "id:201")
	assert.NotContains(t, view, "Ref │")
}

func TestDetailIdentityOrderColorsAndTitleDivider(t *testing.T) {
	change := dto.ChangeView{ID: "12", RefUUID: "uuid", RefSlug: "006-some-slug", EpicName: "Epic", ChangePhase: "backlog", ChangeTypes: []string{"feature"}, AfterChangeName: "First change #2", Title: "Title"}
	rows := append(fixedDetailRows(change), DetailRows(change)...)
	var labels []string
	for _, row := range rows {
		if row.Label != "" {
			labels = append(labels, row.Label)
		}
	}
	require.Equal(t, []string{"ID", "Epic", "Phase", "Types", "Active", "Timestamps", "Title", "Docs", "Testcases", "Completed", "Comments", "After Change", "Ref UUID", "Slug", "PR URL"}, labels)
	view := stripANSI(DetailsView(Model{Detail: change}, 120, 30))
	require.Contains(t, view, "After Change │ First change #2")
	require.NotContains(t, view, "┌")
	require.Contains(t, view, "─────────────┼")
	require.Contains(t, view, "Completed │ ---===")
}

func TestNullableChangeFieldsRenderAsDashes(t *testing.T) {
	change := dto.ChangeView{ID: "12", Ref: "null", RefSlug: "null", EpicID: "null", EpicName: "null", AfterChangeID: "null", Title: "Missing associations"}
	list := stripANSI(TableView(Model{Rows: []dto.ChangeView{change}}, Filters{}, 120, 1))
	assert.NotContains(t, list, "null")
	assert.Contains(t, list, "Missing associations")
	rows := DetailRows(change)
	for _, label := range []string{"Slug", "Epic", "After Change"} {
		found := false
		for _, row := range rows {
			if row.Label == label {
				assert.Equal(t, "-", row.Text, label)
				found = true
			}
		}
		assert.True(t, found, label)
	}
	assert.NotContains(t, stripANSI(DetailsView(Model{Detail: change}, 120, 25)), "null")
	assert.Equal(t, "#3", epicLabel(dto.ChangeView{EpicID: "3", EpicName: "null"}))
	assert.Equal(t, "Planning", epicLabel(dto.ChangeView{EpicID: "3", EpicName: "Planning"}))
	list = stripANSI(TableView(Model{Rows: []dto.ChangeView{{EpicID: "3", EpicName: "Planning", Title: "Example"}}}, Filters{}, 120, 2))
	assert.Contains(t, list, "Planning")
	assert.NotContains(t, list, "Planning #3")
}

func TestInProgressPhaseKeepsTypesAndFollowingColumnsAligned(t *testing.T) {
	const typesWidth, epicWidth, titleWidth = 12, 10, 20
	rows := []string{
		stripANSI(changeTableRowLine("1", "todo", "feature", "Epic", "Title", "111", "222", "333", "2026-09-29", typesWidth, epicWidth, titleWidth, false, nil, nil)),
		stripANSI(changeTableRowLine("2", "in-progress", "feature", "Epic", "Title", "111", "222", "333", "2026-09-29", typesWidth, epicWidth, titleWidth, false, nil, nil)),
	}
	header := stripANSI(changeTableLine("#Ref", "Phase", "Types", "Epic", "Title", "Don", "Tot", "%", "Modified", typesWidth, epicWidth, titleWidth))
	for _, column := range []string{"feature", "Epic", "Title", "111", "222", "333", "2026-09-29"} {
		assert.Equal(t, strings.Index(rows[0], column), strings.Index(rows[1], column), column)
	}
	assert.Equal(t, strings.Index(header, "Types"), strings.Index(rows[0], "feature"))
	assert.Equal(t, strings.Index(header, "Types"), strings.Index(rows[1], "feature"))
	assert.Equal(t, changeTableContentWidth(typesWidth, epicWidth, titleWidth), lipgloss.Width(rows[1]))
}

func TestDetailsViewCountsFixedRowsInsidePageSize(t *testing.T) {
	model := Model{}.WithDetail(dto.ChangeView{
		ID:      "12",
		RefUUID: "11111111-2222-4333-8444-555555555555",
		Ref:     "3",
		Title:   "Backend Change",
		Spec:    "Spec text",
	})

	view := stripANSI(DetailsView(model, 120, 4))
	lines := strings.Split(view, "\n")

	require.Len(t, lines, 4)
	assert.Contains(t, view, "ID │ 12")
	assert.Contains(t, view, "Epic │ -")
}

func TestMoveDetailSelectionKeepsVisibleRowsAnchored(t *testing.T) {
	model := Model{}.WithDetail(dto.ChangeView{
		ID:          "12",
		Ref:         "201",
		RefSlug:     "201-change",
		ChangePhase: "backlog",
		Title:       "Backend Change",
		Spec:        "Spec text",
	})
	model.DetailSelected = 0

	model = model.MoveDetailSelection(2, 4, 120)

	assert.Equal(t, 2, model.DetailSelected)
	assert.Equal(t, 0, model.DetailOffset)
}

func TestMoveDetailSelectionScrollsOnlyEnoughToRevealBottom(t *testing.T) {
	model := Model{}.WithDetail(dto.ChangeView{
		ID:          "12",
		Ref:         "201",
		RefSlug:     "201-change",
		ChangePhase: "backlog",
		EpicName:    "CLI",
		Title:       "Backend Change",
		Spec:        "Spec text",
	})
	model.DetailSelected = 0

	model = model.MoveDetailSelection(3, 3, 120)

	assert.Equal(t, 3, model.DetailSelected)
	assert.Equal(t, 2, model.DetailOffset)
}

func TestP205PhaseStyleUsesOnlyProjectColors(t *testing.T) {
	assert.Equal(t, "12", fmt.Sprint(phaseStyle("custom", PhaseColors{"custom": "12"}).GetForeground()))
	for _, phase := range []string{"custom", "backlog", "production"} {
		assert.Equal(t, lipgloss.NewStyle().GetForeground(), phaseStyle(phase, PhaseColors{}).GetForeground())
	}
}

func stripANSI(value string) string {
	return ansiPattern.ReplaceAllString(value, "")
}

func BenchmarkChangeLongRendering(b *testing.B) {
	for _, n := range []int{5000, 50000, 100000} {
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			m := Model{}.WithDetail(dto.ChangeView{ID: "12", Title: strings.Repeat("x", n)})
			b.ReportAllocs()
			for b.Loop() {
				_ = DetailsView(m, 80, 12)
			}
		})
	}
}

func TestShortDetailsViewportScrollsIdentityAndBody(t *testing.T) {
	for _, height := range []int{3, 4, 5} {
		t.Run(fmt.Sprint(height), func(t *testing.T) {
			const width = 100
			page := height
			original := Model{}.WithDetail(dto.ChangeView{ID: "12", RefUUID: "11111111-2222-4333-8444-555555555555", Ref: "3", Title: "first title line\nlast title line", PRUrl: "https://example.test/pr"})
			model := original
			var seen strings.Builder
			for i := 0; i < 100; i++ {
				view := DetailsViewport(model, width, height, nil)
				require.LessOrEqual(t, lipgloss.Height(view), height)
				seen.WriteString(stripANSI(view))
				model = model.ScrollDetailViewport(page, page, width)
			}
			for _, value := range []string{"ID │ 12", "Ref UUID", original.Detail.RefUUID, "first title line", "last title line", "https://example.test/pr", "Timestamps"} {
				require.Contains(t, seen.String(), value)
			}
			for i := 0; i < 100; i++ {
				model = model.ScrollDetailViewport(-page, page, width)
			}
			require.Contains(t, stripANSI(DetailsViewport(model, width, height, nil)), "ID │ 12")
			model = original
			for i := 0; i < len(DetailRows(model.Detail))+2; i++ {
				var row DetailRow
				var ok bool
				model, row, ok = model.SelectDetailRow(page, width)
				require.True(t, ok)
				view := stripANSI(DetailsViewport(model, width, height, nil))
				if row.Label == "Title" {
					require.Contains(t, view, "title line")
				} else {
					require.Contains(t, view, row.Label)
				}
				model = model.MoveDetailSelection(1, page, width)
			}
		})
	}
}
