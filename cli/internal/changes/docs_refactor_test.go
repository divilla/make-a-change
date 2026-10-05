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

func TestBriefIsADocumentSlotWithFullBodyRetained(t *testing.T) {
	body := strings.Repeat("line\n", 20) + "full tail"
	v := dto.ChangeView{ID: "12", Brief: body, DocumentTypes: []string{"brief", "spec", "comment"}, Documents: []dto.Document{{ID: 8, DocType: "brief", Body: body}}}
	for _, row := range DetailRows(v) {
		if row.DocumentType == "brief" {
			require.Equal(t, "[✓] brief", row.Text)
			require.Equal(t, 8, row.DocumentID)
			require.Len(t, detailRowTextLines(row, 80), 1)
		}
	}
	require.Equal(t, body, v.Brief)
	require.NotContains(t, DetailsView(Model{Detail: v}, 120, 40), "full tail")
}

func Test031ConfiguredDocumentSlotsAndCheckColor(t *testing.T) {
	profile := lipgloss.ColorProfile()
	t.Cleanup(func() { lipgloss.SetColorProfile(profile) })
	lipgloss.SetColorProfile(termenv.TrueColor)
	updated := time.Date(2026, 9, 28, 11, 30, 0, 0, time.UTC)
	v := dto.ChangeView{ID: "12", DocumentTypes: []string{"brief", "notes", "spec", "comment", "pr"}, Documents: []dto.Document{{ID: 9, DocType: "spec", Body: "full spec hidden", UpdatedAt: updated}}, PRUrl: "https://example.test/pr"}
	rows := DetailRows(v)
	var types []string
	for _, r := range rows {
		if r.DocumentType != "" && r.DocumentType != "brief" && !r.Comment {
			types = append(types, r.DocumentType)
			switch r.DocumentType {
			case "spec":
				require.Equal(t, "[✓] spec", r.Text)
				require.Equal(t, updated.Local().Format("2006-01-02 15:04"), r.Timestamp)
				require.Equal(t, []string{r.Timestamp + " - [✓] spec"}, detailRowTextLines(r, 80))
				for _, selected := range []bool{false, true} {
					stampStyle := lipgloss.NewStyle().Foreground(styles.AccentCyan)
					checkStyle := lipgloss.NewStyle().Foreground(styles.AccentGreen)
					bodyStyle := lipgloss.NewStyle().Foreground(styles.Foreground)
					if selected {
						stampStyle = stampStyle.Background(styles.MutedPurple)
						checkStyle = checkStyle.Background(styles.MutedPurple)
						bodyStyle = bodyStyle.Background(styles.MutedPurple)
					}
					view := strings.Join(detailTableRowLines(r, 12, 80, selected, nil), "\n")
					require.Contains(t, view, stampStyle.Render(r.Timestamp)+bodyStyle.Render(" - [")+checkStyle.Render("✓")+bodyStyle.Render(padRightDisplay("] spec", 80-lipgloss.Width(r.Timestamp)-5)))
				}
			case "notes", "pr":
				require.Equal(t, "[ ] "+r.DocumentType, r.Text)
				require.Zero(t, r.DocumentID)
				require.Equal(t, []string{strings.Repeat(" ", 16) + " - [ ] " + r.DocumentType}, detailRowTextLines(r, 80))
			}
		}
	}
	require.Equal(t, []string{"notes", "spec", "pr"}, types)
	view := DetailsView(Model{Detail: v, DetailSelected: -2}, 140, 30)
	require.Contains(t, view, lipgloss.NewStyle().Foreground(styles.AccentGreen).Render("✓"))
	require.NotContains(t, view, "full spec hidden")
	require.Contains(t, view, v.PRUrl)
	v.DocumentTypes = []string{}
	for _, r := range DetailRows(v) {
		require.False(t, r.DocumentType != "" && r.DocumentType != "brief" && !r.Comment)
	}
}

func TestDocCheckboxesAlignWithMissingTimestamps(t *testing.T) {
	change := dto.ChangeView{ID: "12", DocumentTypes: []string{"brief", "spec", "review"}, Documents: []dto.Document{{ID: 8, DocType: "brief", UpdatedAt: time.Date(2026, 9, 28, 11, 30, 0, 0, time.UTC)}, {ID: 9, DocType: "spec"}}}
	for _, selected := range []bool{false, true} {
		var count int
		for _, row := range DetailRows(change) {
			if row.DocumentType == "" || row.Comment {
				continue
			}
			count++
			line := stripANSI(strings.Join(detailTableRowLines(row, 12, 80, selected, nil), "\n"))
			require.Equal(t, 34, lipgloss.Width(line[:strings.Index(line, "[")])) // label/separator + timestamp/separator
			if row.Timestamp == "" {
				require.Contains(t, line, strings.Repeat(" ", 16)+" - "+row.Text)
			}
		}
		require.Equal(t, 3, count)
	}
}

func Test031CommentPreviewNoWrapAndFullBody(t *testing.T) {
	body := strings.Repeat("long", 50) + "\nsecond\nthird\nfourth\nfifth"
	updated := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
	v := dto.ChangeView{ID: "12", Comments: []dto.Document{{ID: 9, DocType: "comment", Body: body, UpdatedAt: updated}}}
	var preview DetailRow
	for _, r := range DetailRows(v) {
		if r.DocumentID == 9 {
			preview = r
		}
	}
	lines := detailRowTextLines(preview, 10)
	require.Len(t, lines, 1)
	require.Equal(t, updated.Local().Format("2006-01-02 15:04")+" - "+strings.ReplaceAll(body, "\n", " "), lines[0])
	require.Contains(t, lines[0], "third")
	require.Contains(t, preview.Text, "fourth")
	require.Equal(t, body, v.Comments[0].Body)
}

func Test031CommentPreviewEscapesTerminalControls(t *testing.T) {
	for _, tc := range []struct{ name, raw, escaped string }{
		{"clipboard", "\x1b]52;c;cGF5bG9hZA==\a", `\x1b]52;c;cGF5bG9hZA==\a`},
		{"display", "\x1b[2J\x1b[31m", `\x1b[2J\x1b[31m`},
		{"C1", "\u009d52;c;payload\u009c", `\u009d52;c;payload\u009c`},
		{"controls", "\b\x00\u202e", `\b\x00\u202e`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := "before " + tc.raw + " after\nsecond\nthird\nraw editor tail"
			v := dto.ChangeView{ID: "12", Comments: []dto.Document{{ID: 9, DocType: "comment", Body: body}}}
			var preview DetailRow
			for _, row := range DetailRows(v) {
				if row.DocumentID == 9 {
					preview = row
				}
			}
			require.Contains(t, preview.Text, "before "+tc.escaped+" after")
			require.NotContains(t, preview.Text, tc.raw)
			require.Len(t, detailRowTextLines(preview, 10), 1)
			for _, width := range []int{40, 160} {
				view := DetailsView(Model{Detail: v}, width, 40)
				require.NotContains(t, view, tc.raw)
				if width == 160 {
					require.Contains(t, stripANSI(view), tc.escaped)
				}
			}
			require.Equal(t, body, v.Comments[0].Body)
		})
	}
}

func Test031CommentTimestampLayout(t *testing.T) {
	profile := lipgloss.ColorProfile()
	t.Cleanup(func() { lipgloss.SetColorProfile(profile) })
	lipgloss.SetColorProfile(termenv.TrueColor)
	when := time.Date(2026, 9, 28, 11, 30, 0, 0, time.UTC)
	for _, n := range []int{1, 2, 3, 6} {
		body := strings.TrimSuffix(strings.Repeat("body ✓\n", n), "\n")
		v := dto.ChangeView{ID: "12", Comments: []dto.Document{{ID: 9, DocType: "comment", Body: body, CreatedAt: when.Add(-24 * time.Hour), UpdatedAt: when}}}
		for _, r := range DetailRows(v) {
			if r.DocumentID == 9 {
				lines := detailRowTextLines(r, 80)
				require.Len(t, lines, 1)
				require.Equal(t, when.Local().Format("2006-01-02 15:04")+" - "+strings.ReplaceAll(body, "\n", " "), lines[0])
				for _, selected := range []bool{false, true} {
					stampStyle := lipgloss.NewStyle().Foreground(styles.AccentCyan)
					bodyStyle := lipgloss.NewStyle().Foreground(styles.Foreground)
					if selected {
						stampStyle = stampStyle.Background(styles.MutedPurple)
						bodyStyle = bodyStyle.Background(styles.MutedPurple)
					}
					view := strings.Join(detailTableRowLines(r, 12, 146, selected, nil), "\n")
					require.Contains(t, view, stampStyle.Render(r.Timestamp))
					require.Contains(t, view, bodyStyle.Render(padRightDisplay(" - "+strings.ReplaceAll(body, "\n", " "), 146-lipgloss.Width(r.Timestamp))))
					require.NotContains(t, view, lipgloss.NewStyle().Foreground(styles.AccentGreen).Render("✓"))
				}
			}
		}
	}
	deleted := when
	v := dto.ChangeView{ID: "12", Comments: []dto.Document{{ID: 9, DocType: "comment", Body: "deleted", DeletedAt: &deleted}}}
	for _, r := range DetailRows(v) {
		require.NotEqual(t, 9, r.DocumentID)
	}
}

func TestChangeDetailsShowsCreatedAndModifiedTimestamps(t *testing.T) {
	created := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	updated := created.Add(48 * time.Hour)
	deleted := created.Add(72 * time.Hour)
	v := dto.ChangeView{ID: "12", Created: created.Format(time.RFC3339), Modified: updated.Format(time.RFC3339), DocumentTypes: []string{"spec"}, Documents: []dto.Document{{ID: 9, DocType: "spec", CreatedAt: created, UpdatedAt: updated}}, Comments: []dto.Document{{ID: 8, DocType: "comment", Body: "live", CreatedAt: created, UpdatedAt: updated}, {ID: 7, DocType: "comment", Body: "deleted", CreatedAt: created, UpdatedAt: updated, DeletedAt: &deleted}}, TestCases: []dto.TestCase{{ID: 31, Scenario: "case", CreatedAt: created, UpdatedAt: updated}}}
	view := stripANSI(DetailsView(Model{Detail: v}, 160, 40))
	require.Contains(t, view, updated.Local().Format("2006-01-02 15:04"))
	require.Contains(t, view, created.Local().Format("2006-01-02 15:04"))
	require.NotContains(t, view, deleted.Local().Format("2006-01-02 15:04"))
	require.Contains(t, view, "Created")
	require.NotContains(t, view, "deleted_at")
}

func Test031AllDisplayedTimestampsUseLocalTime(t *testing.T) {
	prior := time.Local
	local, err := time.LoadLocation("Europe/Zagreb")
	require.NoError(t, err)
	time.Local = local
	t.Cleanup(func() { time.Local = prior })
	for _, tt := range []struct{ input, want string }{{"2026-03-29T00:30:00Z", "2026-03-29 01:30"}, {"2026-03-29T01:30:00Z", "2026-03-29 03:30"}, {"2026-03-29T03:30:00+02:00", "2026-03-29 03:30"}, {"2026-10-25T00:30:00Z", "2026-10-25 02:30"}, {"2026-10-25T01:30:00Z", "2026-10-25 02:30"}} {
		require.Equal(t, tt.want, formatListTimestamp(tt.input))
	}
}

func TestDetailItemTimestampTodayUsesAccentPurple(t *testing.T) {
	profile, local := lipgloss.ColorProfile(), time.Local
	t.Cleanup(func() {
		lipgloss.SetColorProfile(profile)
		time.Local = local
	})
	lipgloss.SetColorProfile(termenv.TrueColor)
	time.Local = time.FixedZone("local-test", 14*60*60)
	now := time.Now().Local()
	for _, offset := range []int{-1, 0, 1} {
		updated := now.AddDate(0, 0, offset).UTC()
		change := dto.ChangeView{ID: "12", DocumentTypes: []string{"brief"}, Documents: []dto.Document{{ID: 8, DocType: "brief", UpdatedAt: updated}}, Comments: []dto.Document{{ID: 9, DocType: "comment", Body: "body", UpdatedAt: updated}}}
		color := styles.AccentCyan
		if offset == 0 {
			color = styles.AccentPurple
		}
		var checked int
		for _, row := range DetailRows(change) {
			if row.Timestamp == "" {
				continue
			}
			checked++
			require.Equal(t, updated.Local().Format("2006-01-02 15:04"), row.Timestamp)
			for _, selected := range []bool{false, true} {
				style := lipgloss.NewStyle().Foreground(color)
				if selected {
					style = style.Background(styles.MutedPurple)
				}
				require.Contains(t, strings.Join(detailTableRowLines(row, 12, 80, selected, nil), "\n"), style.Render(row.Timestamp))
			}
		}
		require.Equal(t, 2, checked)
	}
}

func Test031ActiveCommandLabelAndHelp(t *testing.T) {
	require.Equal(t, "/new-comment", DetailCommands()[0])
	require.Contains(t, DetailCommands(), "/active")
	require.NotContains(t, DetailCommands(), "/open")
	require.Contains(t, HelpView(), "/active")
	require.NotContains(t, HelpView(), "/open")
	for _, r := range DetailRows(dto.ChangeView{ID: "12", Active: true}) {
		require.NotEqual(t, "Open", r.Label)
	}
}
