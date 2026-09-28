package epics

import (
	"cli/internal/styles"
	"cli/internal/ui"
	"fmt"
	"strconv"
	"strings"
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
	var lines []string
	if height > 1 {
		lines = append(lines, "ID   Name   Done/Total TC   Completed   Changes")
		height--
	}
	count := min(height, len(m.Rows))
	start := max(0, min(m.Selected-count/2, len(m.Rows)-count))
	for i := start; i < start+count; i++ {
		e := m.Rows[i]
		line := fmt.Sprintf("%d  %s  %d/%d  %d  %d", e.ID, strings.Join(strings.Fields(e.Name), " "), e.DoneTC, e.TotalTC, e.Completed, e.ChangeCount)
		if i == m.Selected {
			line = styles.Default.Selection.Render(line)
		}
		lines = append(lines, line)
	}
	return ui.TruncateBlock(strings.Join(lines, "\n"), width)
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
		lines = append(lines, fmt.Sprintf("Done TC: %d\nTotal TC: %d\nCompleted: %d\nChanges: %d\nCreated: %s\nModified: %s", e.DoneTC, e.TotalTC, e.Completed, e.ChangeCount, e.CreatedAt.Format("2006-01-02 15:04:05Z07:00"), e.UpdatedAt.Format("2006-01-02 15:04:05Z07:00")))
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
	return "List: /new-epic creates an epic; Enter opens details.\nScroll details: Up/Down or PgUp/PgDown.\nDetails: /edit changes the name; /delete asks for confirmation.\nForms: Enter or /save; Ctrl+E editor; /cancel discards.\n/retry reloads without repeating a write; /return goes back."
}
