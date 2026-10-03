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

func Test031BriefPreviewAndFullEditorSeed(t *testing.T) {
	body := strings.Repeat("line\n", 20) + "full tail"
	v := dto.ChangeView{ID: "12", Brief: body, DocumentTypes: []string{"brief", "spec", "comment"}}
	rows := DetailRows(v)
	var brief DetailRow
	for _, r := range rows {
		if r.Label == "Brief" {
			brief = r
		}
	}
	lines := detailRowTextLines(brief, 80)
	require.Len(t, lines, 16)
	require.Equal(t, "...", lines[15])
	require.Equal(t, body, v.Brief)
	for _, r := range rows {
		if r.Label != "Brief" {
			require.NotEqual(t, "brief", r.DocumentType)
		}
	}
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
				require.Equal(t, "spec [✓] "+updated.Local().Format("2006-01-02 15:04"), r.Text)
			case "notes", "pr":
				require.Equal(t, r.DocumentType+" [ ]", r.Text)
				require.Zero(t, r.DocumentID)
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

func Test031CommentPreviewNoWrapAndFullBody(t *testing.T) {
	body := strings.Repeat("long", 50) + "\nsecond\nthird\nfourth\nfifth"
	v := dto.ChangeView{ID: "12", Comments: []dto.Document{{ID: 9, DocType: "comment", Body: body, UpdatedAt: time.Now()}}}
	var preview DetailRow
	for _, r := range DetailRows(v) {
		if r.DocumentID == 9 {
			preview = r
		}
	}
	lines := detailRowTextLines(preview, 10)
	require.Len(t, lines, 4)
	require.Equal(t, strings.Repeat("long", 50), lines[0])
	require.Equal(t, "third", lines[2])
	require.NotContains(t, preview.Text, "fourth")
	require.Equal(t, body, v.Comments[0].Body)
}

func Test031CommentTimestampLayout(t *testing.T) {
	when := time.Date(2026, 9, 28, 11, 30, 0, 0, time.UTC)
	for _, n := range []int{1, 2, 3, 6} {
		body := strings.TrimSuffix(strings.Repeat("body\n", n), "\n")
		v := dto.ChangeView{ID: "12", Comments: []dto.Document{{ID: 9, DocType: "comment", Body: body, UpdatedAt: when}}}
		for _, r := range DetailRows(v) {
			if r.DocumentID == 9 {
				lines := detailRowTextLines(r, 80)
				require.Len(t, lines, min(n, 3)+1)
				require.Equal(t, when.Local().Format("2006-01-02 15:04"), lines[len(lines)-1])
			}
		}
	}
	deleted := when
	v := dto.ChangeView{ID: "12", Comments: []dto.Document{{ID: 9, DocType: "comment", Body: "deleted", DeletedAt: &deleted}}}
	for _, r := range DetailRows(v) {
		require.NotEqual(t, 9, r.DocumentID)
	}
}

func Test031ChangeDetailsShowsOnlyUpdatedAt(t *testing.T) {
	created := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	updated := created.Add(48 * time.Hour)
	deleted := created.Add(72 * time.Hour)
	v := dto.ChangeView{ID: "12", Created: created.Format(time.RFC3339), Modified: updated.Format(time.RFC3339), DocumentTypes: []string{"spec"}, Documents: []dto.Document{{ID: 9, DocType: "spec", CreatedAt: created, UpdatedAt: updated}}, Comments: []dto.Document{{ID: 8, DocType: "comment", Body: "live", CreatedAt: created, UpdatedAt: updated}, {ID: 7, DocType: "comment", Body: "deleted", CreatedAt: created, UpdatedAt: updated, DeletedAt: &deleted}}, TestCases: []dto.TestCase{{ID: 31, Scenario: "case", CreatedAt: created, UpdatedAt: updated}}}
	view := stripANSI(DetailsView(Model{Detail: v}, 160, 40))
	require.Contains(t, view, updated.Local().Format("2006-01-02 15:04"))
	require.NotContains(t, view, created.Local().Format("2006-01-02 15:04"))
	require.NotContains(t, view, deleted.Local().Format("2006-01-02 15:04"))
	require.NotContains(t, view, "Created")
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
