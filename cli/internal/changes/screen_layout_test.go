package changes

import (
	"cli/internal/dto"
	"cli/internal/styles"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"github.com/stretchr/testify/require"
)

func TestDetailLayoutColorsAndSectionDividers(t *testing.T) {
	profile := lipgloss.ColorProfile()
	t.Cleanup(func() { lipgloss.SetColorProfile(profile) })
	lipgloss.SetColorProfile(termenv.TrueColor)
	when := time.Date(2026, 10, 4, 10, 0, 0, 0, time.UTC)
	change := dto.ChangeView{ID: "1", Title: "Respect q=0", Active: true, Created: when.Format(time.RFC3339), Modified: when.Format(time.RFC3339), DocumentTypes: []string{"brief", "spec", "pr", "review"}, Documents: []dto.Document{{ID: 8, DocType: "brief", UpdatedAt: when}, {ID: 9, DocType: "spec", UpdatedAt: when}}, TestCases: []dto.TestCase{{ID: 1, Scenario: "first", Done: true}, {ID: 2, Scenario: "second"}, {ID: 3, Scenario: "third"}}, Done: 1, Total: 3, Completed: 33, Comments: []dto.Document{{ID: 7, Body: "Comment 1\ncontinued"}, {ID: 6, Body: "Comment 2"}}}
	view := DetailsView(Model{}.WithDetail(change), 160, 35)
	require.Contains(t, stripANSI(view), "Docs │ "+when.Local().Format("2006-01-02 15:04")+" - [✓] brief")
	require.Contains(t, stripANSI(view), "Testcases │ [✓] first (#1)")
	require.Contains(t, stripANSI(view), "Completed │ ---=== 1/3 - 33% ===---")
	require.Contains(t, stripANSI(view), "Comments │ - Comment 1 continued")
	require.Equal(t, 6, strings.Count(stripANSI(view), "─┼─"))
	require.Contains(t, view, lipgloss.NewStyle().Foreground(styles.AccentGreen).Render("✓"))
	for _, row := range DetailRows(change) {
		for _, selected := range []bool{false, true} {
			style := detailValueStyle(row, nil)
			if selected {
				style = detailSelectedStyle(row, nil)
			}
			if row.TestCaseID != "" {
				require.Equal(t, styles.AccentWhite, style.GetForeground())
			}
			if row.DocumentType != "" {
				require.Equal(t, styles.Foreground, style.GetForeground())
			}
			if row.Label == "Timestamps" {
				require.Equal(t, styles.AccentCyan, style.GetForeground())
			}
			if row.Label == "Slug" {
				require.Equal(t, styles.AccentGreen, style.GetForeground())
			}
			if row.Label == "Completed" {
				require.Equal(t, styles.AccentBlue, style.GetForeground())
				require.True(t, style.GetBold())
			}
			if row.Timestamp != "" {
				stampStyle := lipgloss.NewStyle().Foreground(styles.AccentCyan)
				if selected {
					stampStyle = stampStyle.Background(styles.MutedPurple)
				}
				require.Contains(t, strings.Join(detailTableRowLines(row, 12, 146, selected, nil), "\n"), stampStyle.Render(row.Timestamp))
			}
		}
	}
	require.Contains(t, view, lipgloss.NewStyle().Foreground(styles.AccentCyan).Render(when.Local().Format("2006-01-02 15:04")))
}
