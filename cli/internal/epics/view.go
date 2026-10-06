package epics

import (
	"cli/internal/dto"
	"cli/internal/styles"
	"cli/internal/ui"
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// ListTitle names the epic list screen.
func ListTitle() string { return "EpicsListScreen - Title: Epics List" }

// DetailTitle names the epic details screen.
func DetailTitle() string { return "EpicDetailsScreen - Title: Epic Details" }

// TableView renders actual server rows and explicit loading/empty states.
func TableView(m Model, width, height int) string {
	if height <= 0 {
		return ""
	}
	if m.Loading {
		return "Epics: loading"
	}
	if len(m.Rows) == 0 {
		return "No epics."
	}
	width = ui.NormalizeWidth(width)
	tableWidth := min(63, max(1, width-2))
	header := epicTableLine("ID", "Name", "DoneTC", "Compl", "Chngs", "Active")
	lines := []string{}
	if height > 1 {
		lines = append(lines, styles.Default.Muted.Render(header))
	}
	rowHeight := height - 4
	if height < 5 {
		rowHeight = height - len(lines)
	}
	count := min(max(1, rowHeight), len(m.Rows))
	start := max(0, min(m.Selected-count/2, len(m.Rows)-count))
	for i := start; i < start+count; i++ {
		e := m.Rows[i]
		active := ""
		if !e.Active {
			active = "inactive"
		}
		line := epicTableLine(strconv.Itoa(e.ID), e.Name, fmt.Sprintf("%d/%d", e.DoneTC, e.TotalTC), fmt.Sprintf("%d%%", e.Completed), strconv.Itoa(e.ChangeCount), active)
		lines = append(lines, epicRowStyle(e, i == m.Selected).Render(line))
	}
	lines = append(lines, styles.Default.Foreground.Render(fmt.Sprintf("Rows %d-%d of %d", start+1, start+count, len(m.Rows))))
	for i := range lines {
		lines[i] = ansi.Truncate(lines[i], tableWidth, "")
	}
	content := strings.Join(lines, "\n")
	if height < 5 || width < 3 {
		return ui.TruncateBlock(strings.Join(strings.Split(content, "\n")[:min(height, len(lines))], "\n"), width)
	}
	return lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("240")).Width(tableWidth).Render(content)
}

func epicTableLine(id, name, done, completed, count, active string) string {
	return epicCell(id, 4, true) + " " + epicCell(name, 30, false) + " " + epicCell(done, 6, true) + " " + epicCell(completed, 5, true) + " " + epicCell(count, 5, true) + " " + epicCell(active, 8, false)
}

func epicCell(value string, width int, right bool) string {
	value = ansi.Truncate(strings.Join(strings.Fields(ui.SafeText(value)), " "), width, "")
	padding := strings.Repeat(" ", max(0, width-lipgloss.Width(value)))
	if right {
		return padding + value
	}
	return value + padding
}

func epicRowStyle(e dto.Epic, selected bool) lipgloss.Style {
	color := styles.Foreground
	if !e.Active {
		color = styles.AccentRed
	}
	style := lipgloss.NewStyle().Foreground(color)
	if selected {
		style = style.Background(styles.MutedPurple)
	}
	return style
}

// DetailsView shows server completion values without calculating business state.
func DetailsView(m Model, width int) string {
	if m.Loading {
		return "Loading epic…"
	}
	e := m.Detail
	if e.ID <= 0 {
		return ""
	}
	lines := []string{"ID: " + strconv.Itoa(e.ID), "Project ID: " + strconv.Itoa(e.ProjectID), "Name: " + e.Name}
	if m.DetailLoaded {
		lines = append(lines, fmt.Sprintf("Done TC: %d\nTotal TC: %d\nCompleted: %d\nChanges: %d\nCreated: %s\nModified: %s", e.DoneTC, e.TotalTC, e.Completed, e.ChangeCount, e.CreatedAt.Local().Format("2006-01-02 15:04"), e.UpdatedAt.Local().Format("2006-01-02 15:04")))
	} else {
		lines = append(lines, "Details unavailable; /retry to load")
	}
	return ui.TruncateBlock(strings.Join(lines, "\n"), width)
}

// DetailsViewport bounds the detail body while retaining access to every line.
func DetailsViewport(m Model, width, height int) string {
	if height <= 0 {
		return ""
	}
	lines := strings.Split(DetailsView(m, width), "\n")
	start := detailOffset(m.DetailOffset, len(lines), height)
	return strings.Join(lines[start:min(start+height, len(lines))], "\n")
}

// ScrollDetails moves within the current viewport, clamping after a resize.
func (m Model) ScrollDetails(delta, width, height int) Model {
	lines := strings.Count(DetailsView(m, width), "\n") + 1
	m.DetailOffset = detailOffset(detailOffset(m.DetailOffset, lines, height)+delta, lines, height)
	return m
}

func detailOffset(offset, lines, height int) int {
	return max(0, min(offset, lines-max(1, height)))
}

// HelpView describes the available operations and literal form/editor behavior.
func HelpView() string {
	return "List: /new-epic creates an epic; Enter opens details; Delete deletes or deactivates; Space toggles activity.\nScroll details: Up/Down or PgUp/PgDown.\nDetails: /edit changes the name; /delete asks for confirmation.\nForms: Enter or /save; Ctrl+E editor; /cancel discards.\n/retry reloads without repeating a write; /return goes back."
}
