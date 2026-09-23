package changes

import (
    "fmt"
    "regexp"
    "strings"
    "testing"

    "cli/internal/dto"

    "github.com/charmbracelet/lipgloss"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func TestDetailsViewSeparatesSpecAndTestCases(t *testing.T) {
    model := Model{}.WithDetail(dto.Change{
        ID:      "12",
        RefUUID: "11111111-2222-4333-8444-555555555555",
        Ref:     "3",
        Title:   "Backend Change",
        Spec:    "Spec text",
        TestCases: []dto.TestCase{
            {ID: "31", Scenario: "first scenario", Done: true},
        },
    })

    view := stripANSI(DetailsView(model, 120, 20))

    assert.Contains(t, view, "ID │ 12")
    assert.Contains(t, view, "Ref UUID │ 11111111-2222-4333-8444-555555555555")
    specIndex := strings.Index(view, "Spec │ Spec text")
    dividerIndex := strings.Index(view[specIndex:], "───────────┼")
    testCaseIndex := strings.Index(view, "✅ │ first scenario (#31)")
    require.NotEqual(t, -1, specIndex)
    require.NotEqual(t, -1, dividerIndex)
    require.NotEqual(t, -1, testCaseIndex)
    assert.Less(t, specIndex, specIndex+dividerIndex)
    assert.Less(t, specIndex+dividerIndex, testCaseIndex)
}

func TestDetailsViewEmojiRowsDoNotOverflowSelectionWidth(t *testing.T) {
    change := dto.Change{
        ID:      "12",
        Ref:     "3",
        Title:   "Backend Change",
        Spec:    "Spec text",
        Open:    true,
        Created: "2026-06-29T08:15:00Z",
        TestCases: []dto.TestCase{
            {ID: "31", Scenario: "first scenario", Done: true},
            {ID: "32", Scenario: "second scenario", Done: false},
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
    model := Model{}.WithDetail(dto.Change{
        ID:    "201",
        Title: "Unreferenced Change",
    })

    view := stripANSI(DetailsView(model, 120, 20))

    assert.Contains(t, view, "ID │ 201")
    assert.Contains(t, view, "Ref UUID │")
    assert.Contains(t, view, "Ref │")
    assert.NotContains(t, view, "id:201")
    assert.NotContains(t, view, "Ref │ ?")
}

func TestDetailsViewCountsFixedRowsInsidePageSize(t *testing.T) {
    model := Model{}.WithDetail(dto.Change{
        ID:      "12",
        RefUUID: "11111111-2222-4333-8444-555555555555",
        Ref:     "3",
        Title:   "Backend Change",
        Spec:    "Spec text",
    })

    view := stripANSI(DetailsView(model, 120, 4))
    lines := strings.Split(view, "\n")

    require.GreaterOrEqual(t, len(lines), 6)
    contentLines := lines[1 : len(lines)-1]
    assert.Len(t, contentLines, 4)
    assert.Contains(t, view, "ID │ 12")
    assert.Contains(t, view, "Ref UUID │ 11111111-2222-4333-8444-555555555555")
}

func TestMoveDetailSelectionKeepsVisibleRowsAnchored(t *testing.T) {
    model := Model{}.WithDetail(dto.Change{
        ID:          "12",
        Ref:         "201",
        Slug:        "201-change",
        ChangePhase: "backlog",
        Title:       "Backend Change",
        Spec:        "Spec text",
    })
    model.DetailSelected = 0

    model = model.MoveDetailSelection(2, 4, 120)

    assert.Equal(t, 2, model.DetailSelected)
    assert.Equal(t, 1, model.DetailOffset)
}

func TestMoveDetailSelectionScrollsOnlyEnoughToRevealBottom(t *testing.T) {
    model := Model{}.WithDetail(dto.Change{
        ID:          "12",
        Ref:         "201",
        Slug:        "201-change",
        ChangePhase: "backlog",
        EpicName:    "CLI",
        Title:       "Backend Change",
        Spec:        "Spec text",
    })
    model.DetailSelected = 0

    model = model.MoveDetailSelection(3, 3, 120)

    assert.Equal(t, 3, model.DetailSelected)
    assert.Equal(t, 3, model.DetailOffset)
}

func TestPhaseStyleUsesOptionColorOrGreyFallback(t *testing.T) {
    assert.Equal(t, "12", fmt.Sprint(phaseStyle("custom", PhaseColors{"custom": "12"}).GetForeground()))
    assert.Equal(t, "240", fmt.Sprint(phaseStyle("custom", PhaseColors{}).GetForeground()))
}

func stripANSI(value string) string {
    return ansiPattern.ReplaceAllString(value, "")
}
