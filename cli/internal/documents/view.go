package documents

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"github.com/charmbracelet/x/ansi"
)

// Commands exposes document actions to the shell command picker.
func Commands() []string { return []string{"/new-document", "/type", "/retry", "/cancel", "/return"} }

// DetailTitle names the shared owner document screen.
func DetailTitle() string { return "DocumentScreen - Title: Documents" }

// Help describes history selection and the ordinary append form.
func Help(m Model) string {
	if m.ShowingDetail {
		return "<up/down> scroll  |  <pgup/pgdown> page  |  /return history"
	}
	return "<up/down> select version  |  <return> details  |  /type choose  |  /new-document append  |  /retry reload  |  /return"
}

// View renders selected-owner history or a single fetched version in a bounded viewport.
func View(m Model, width, height int) string {
	if height <= 0 {
		return ""
	}
	lines := viewLines(m, width)
	for i := range lines {
		lines[i] = ansi.Truncate(lines[i], max(1, width), "")
	}
	start := min(max(0, m.Offset), max(0, len(lines)-height))
	end := min(len(lines), start+height)
	return strings.Join(lines[start:end], "\n")
}

// SelectedRowLine returns the selected history row's position in the rendered lines.
func (m Model) SelectedRowLine(width int) int {
	return len(viewLines(m, width)) - len(m.Rows) + m.Selected
}

func viewLines(m Model, width int) []string {
	width = max(1, width)
	lines := []string{fmt.Sprintf("Documents: %s #%d (project #%d)", m.OwnerTable, m.OwnerID, m.ProjectID)}
	if len(m.Types) == 0 {
		lines = append(lines, "Types: none configured; history remains readable")
	} else {
		lines = append(lines, "Types: "+SafeLine(strings.Join(m.Types, ", "))+" | selected: "+SafeLine(m.DraftType))
	}
	if m.CatalogErr != nil {
		lines = append(lines, "Catalog error: "+SafeLine(m.CatalogErr.Error()))
	}
	if m.Busy {
		return append(lines, "Loading documents…")
	}
	if m.Err != nil && !m.Loaded {
		return append(lines, "Error: "+SafeLine(m.Err.Error()), "Use /retry to read again")
	}
	if !m.Loaded {
		return append(lines, "Documents not loaded")
	}
	if m.ShowingDetail {
		if !m.DetailLoaded {
			return append(lines, "Version details not loaded")
		}
		d := m.Detail
		provenance := "human"
		if d.AgentEdit {
			provenance = "agent"
		}
		status := "historical"
		if m.IsActive(d.ID) {
			status = "current"
		}
		lines = append(lines, fmt.Sprintf("Version #%d | %s | %s | %s #%d | type %s", d.ID, status, provenance, d.RefTable, d.RefID, SafeLine(d.DocType)),
			"Created: "+d.CreatedAt.Local().Format("2006-01-02 15:04"), "Updated: "+d.UpdatedAt.Local().Format("2006-01-02 15:04"), "Raw body:")
		lines = appendWrapped(lines, d.Body, width)
		lines = append(lines, "Rendered HTML:")
		lines = appendWrapped(lines, d.HTML, width)
		return lines
	}
	lines = append(lines, fmt.Sprintf("Current: %d | History: %d", len(m.Current), len(m.Rows)))
	if len(m.Rows) == 0 {
		return append(lines, "No document versions for this owner")
	}
	for i, d := range m.Rows {
		marker := " "
		if i == m.Selected {
			marker = ">"
		}
		status := "history"
		if m.IsActive(d.ID) {
			status = "current"
		}
		provenance := "human"
		if d.AgentEdit {
			provenance = "agent"
		}
		line := fmt.Sprintf("%s #%d %s %s %s created %s", marker, d.ID, SafeLine(d.DocType), status, provenance, d.CreatedAt.Local().Format("2006-01-02 15:04"))
		lines = append(lines, ansi.Truncate(line, width, ""))
	}
	return lines
}

func appendWrapped(lines []string, raw string, width int) []string {
	for _, line := range strings.Split(safeText(raw), "\n") {
		lines = append(lines, strings.Split(ansi.Hardwrap(line, width, true), "\n")...)
	}
	return lines
}

// SafeText escapes raw control bytes for display without changing stored data.
func SafeText(raw string) string { return safeText(raw) }

// SafeLine renders untrusted text within one terminal line.
func SafeLine(raw string) string { return strings.ReplaceAll(safeText(raw), "\n", `\n`) }

func safeText(raw string) string {
	var b strings.Builder
	for _, r := range raw {
		if r == '\n' {
			b.WriteRune(r)
			continue
		}
		if r == '\t' {
			b.WriteString("    ")
			continue
		}
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			b.WriteString(strconv.QuoteRune(r)[1 : len(strconv.QuoteRune(r))-1])
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
