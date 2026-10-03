package changes

import (
	"cli/internal/dto"
	"cli/internal/styles"
	"cli/internal/ui"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// ListTitle returns the changes list screen title.
func ListTitle() string {
	return "ChangesListScreen - Title: Changes List"
}

// DetailTitle returns the change details screen title.
func DetailTitle() string {
	return "ChangeDetailsScreen - Title: Change Details"
}

// TableView renders the selectable changes list.
func TableView(m Model, filters Filters, width int, pageSize int, phaseColors ...PhaseColors) string {
	colors := activePhaseColors(phaseColors)
	width = ui.NormalizeWidth(width)
	if m.Loading {
		return styles.Default.InputBand.Width(width).Render("Changes: loading")
	}
	rows := FilteredRows(m.Rows, filters)
	queryWords := normalizedFindWords(filters.Find)
	filtersActive := filters.Phase.ID != "" || filters.Type.ID != "" || filters.Epic.ID != "" || strings.TrimSpace(filters.Find) != ""
	if len(rows) == 0 && len(m.Rows) == 0 && !filtersActive {
		return styles.Default.Muted.Render("No changes.")
	}
	if pageSize < 1 {
		pageSize = 1
	}
	selected, offset, end := 0, 0, 0
	if len(rows) > 0 {
		selected = m.ClampSelection(filters, pageSize).Selected
		offset = clampOffset(m.Offset, selected, len(rows), pageSize)
		end = min(offset+pageSize, len(rows))
	}
	terminalTableWidth := innerTableWidth(width)
	typesWidth, epicWidth, titleWidth := changeTableColumnWidths(terminalTableWidth)
	tableWidth := changeTableContentWidth(typesWidth, epicWidth, titleWidth)
	lines := []string{changeTableHeaderLine(typesWidth, epicWidth, titleWidth)}
	if len(rows) == 0 {
		lines = append(lines, changeTableEmptyLine("No changes match filters.", typesWidth, epicWidth, titleWidth))
	}
	for i, change := range rows[offset:end] {
		rowIndex := offset + i
		line := changeTableRowLine(
			displayRef(change),
			change.ChangePhase,
			strings.Join(change.ChangeTypes, "|"),
			epicLabel(change),
			change.Title,
			strconv.FormatInt(change.Done, 10),
			strconv.FormatInt(change.Total, 10),
			strconv.FormatInt(change.Completed, 10),
			formatListTimestamp(change.Modified),
			typesWidth,
			epicWidth,
			titleWidth,
			rowIndex == selected,
			colors,
			queryWords,
		)
		lines = append(lines, line)
	}
	for len(lines) < pageSize+1 {
		lines = append(lines, "")
	}
	first := 0
	if len(rows) > 0 {
		first = offset + 1
	}
	lines = append(lines, styles.Default.Foreground.Render(fmt.Sprintf("Rows %d-%d of %d", first, end, len(rows))))
	content := ui.TruncateBlock(strings.Join(lines, "\n"), tableWidth)
	return boxedTable(content, tableWidth)
}

func changeTableEmptyLine(message string, typesWidth, epicWidth, titleWidth int) string {
	prefix := fmt.Sprintf("%6s %-11s %-*s %-*s ", "", "", typesWidth, "", epicWidth, "")
	remaining := changeTableContentWidth(typesWidth, epicWidth, titleWidth) - lipgloss.Width(prefix)
	return styles.Default.Muted.Render(prefix) +
		lipgloss.NewStyle().Foreground(styles.AccentRed).Render(padRightDisplay(message, remaining))
}

func changeTableHeaderLine(typesWidth, epicWidth, titleWidth int) string {
	line := changeTableLine("#Ref", "Phase", "Types", "Epic", "Title", "Don", "Tot", "%", "Modified", typesWidth, epicWidth, titleWidth)
	types := strings.Index(line, "Types")
	percent := strings.Index(line, "%")
	if types < 0 || percent < 0 {
		return styles.Default.Muted.Render(line)
	}
	return styles.Default.Muted.Render(line[:types]) +
		lipgloss.NewStyle().Foreground(styles.AccentPurple).Render("Types") +
		styles.Default.Muted.Render(line[types+len("Types"):percent]) +
		lipgloss.NewStyle().Foreground(styles.AccentBlue).Render("%") +
		styles.Default.Muted.Render(line[percent+1:])
}

func changeTableLine(ref, phase, types, epic, title, done, total, completed, modified string, typesWidth, epicWidth, titleWidth int) string {
	return fmt.Sprintf(
		"%6s %-11s %-*s %-*s %-*s %3s %3s %3s %-16s",
		tableText(ref, 6),
		tableText(phase, 11),
		typesWidth,
		tableText(types, typesWidth),
		epicWidth,
		tableText(epic, epicWidth),
		titleWidth,
		tableText(title, titleWidth),
		tableText(done, 3),
		tableText(total, 3),
		tableText(completed, 3),
		tableText(modified, 16),
	)
}

func changeTableRowLine(ref, phase, types, epic, title, done, total, completed, modified string, typesWidth, epicWidth, titleWidth int, selected bool, phaseColors PhaseColors, queryWords []string) string {
	prefix := fmt.Sprintf("%6s ", tableText(ref, 6))
	phaseValue := fmt.Sprintf("%-11s", tableText(phase, 11))
	typesValue := fmt.Sprintf("%-*s", typesWidth, tableText(types, typesWidth))
	epicValue := fmt.Sprintf(" %-*s ", epicWidth, tableText(epic, epicWidth))
	titleValue := fmt.Sprintf("%-*s", titleWidth, tableText(title, titleWidth))
	beforeCompleted := fmt.Sprintf(
		" %3s %3s ",
		tableText(done, 3),
		tableText(total, 3),
	)
	completedValue := fmt.Sprintf("%3s", tableText(completed, 3))
	afterCompleted := fmt.Sprintf(" %-16s", tableText(modified, 16))

	if selected {
		base := styles.Default.Selection
		return renderFindHighlights(prefix, base, queryWords) +
			renderFindHighlights(phaseValue, phaseStyle(phase, phaseColors).Background(styles.MutedPurple), queryWords) +
			base.Render(" ") +
			renderFindHighlights(typesValue, base.Foreground(styles.AccentPurple), queryWords) +
			renderFindHighlights(epicValue, base, queryWords) +
			renderFindHighlights(titleValue, base.Foreground(lipgloss.Color("15")), queryWords) +
			base.Render(beforeCompleted) +
			base.Foreground(styles.AccentBlue).Render(completedValue) +
			base.Render(afterCompleted)
	}

	return renderFindHighlights(prefix, styles.Default.Muted, queryWords) +
		renderFindHighlights(phaseValue, phaseStyle(phase, phaseColors), queryWords) +
		styles.Default.Muted.Render(" ") +
		renderFindHighlights(typesValue, lipgloss.NewStyle().Foreground(styles.AccentPurple), queryWords) +
		renderFindHighlights(epicValue, styles.Default.Muted, queryWords) +
		renderFindHighlights(titleValue, lipgloss.NewStyle().Foreground(lipgloss.Color("15")), queryWords) +
		styles.Default.Muted.Render(beforeCompleted) +
		lipgloss.NewStyle().Foreground(styles.AccentBlue).Render(completedValue) +
		styles.Default.Muted.Render(afterCompleted)
}

func renderFindHighlights(value string, normal lipgloss.Style, queryWords []string) string {
	if len(queryWords) == 0 {
		return normal.Render(value)
	}
	matchStyle := lipgloss.NewStyle().Foreground(styles.Foreground).Background(styles.MutedGreen)
	var rendered strings.Builder
	plainStart := 0
	for position := 0; position < len(value); {
		first, size := utf8.DecodeRuneInString(value[position:])
		if !findWordCharacter(first) {
			position += size
			continue
		}
		wordStart := position
		wordOffsets := []int{wordStart}
		var word strings.Builder
		for position < len(value) {
			letter, width := utf8.DecodeRuneInString(value[position:])
			if !findWordCharacter(letter) {
				break
			}
			word.WriteRune(unicode.ToLower(letter))
			position += width
			wordOffsets = append(wordOffsets, position)
		}
		matchLength := 0
		lowerWord := word.String()
		for _, queryWord := range queryWords {
			if strings.HasPrefix(lowerWord, queryWord) && len(queryWord) > matchLength {
				matchLength = len(queryWord)
			}
		}
		if matchLength == 0 {
			continue
		}
		if plainStart < wordStart {
			rendered.WriteString(normal.Render(value[plainStart:wordStart]))
		}
		rendered.WriteString(matchStyle.Render(value[wordStart:wordOffsets[matchLength]]))
		plainStart = wordOffsets[matchLength]
	}
	if plainStart < len(value) {
		rendered.WriteString(normal.Render(value[plainStart:]))
	}
	return rendered.String()
}

func findWordCharacter(letter rune) bool {
	letter = unicode.ToLower(letter)
	return letter >= 'a' && letter <= 'z' || letter >= '0' && letter <= '9' || letter == '-' || letter == '_'
}

func phaseStyle(phase string, phaseColors PhaseColors) lipgloss.Style {
	key := strings.TrimSpace(phase)
	color := strings.TrimSpace(phaseColors[key])
	if color == "" {
		return lipgloss.NewStyle()
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color(color))
}

func changeTableColumnWidths(width int) (int, int, int) {
	const targetTypesWidth = 30
	const maxEpicWidth = 20
	const maxTitleWidth = 80
	const minTypesWidth = 5
	const minEpicWidth = 4
	const minTitleWidth = 5

	available := width - changeTableFixedWidth
	if available >= targetTypesWidth+maxEpicWidth+maxTitleWidth {
		return targetTypesWidth, maxEpicWidth, maxTitleWidth
	}
	if available <= minTypesWidth+minEpicWidth+minTitleWidth {
		return shrinkColumns(available, minTypesWidth, minEpicWidth, minTitleWidth)
	}

	typesWidth := targetTypesWidth
	epicWidth := maxEpicWidth
	titleWidth := available - typesWidth - epicWidth

	if titleWidth < minTitleWidth {
		deficit := minTitleWidth - titleWidth
		epicReduction := min(deficit, epicWidth-minEpicWidth)
		epicWidth -= epicReduction
		deficit -= epicReduction
		typesWidth -= min(deficit, typesWidth-minTypesWidth)
		titleWidth = available - typesWidth - epicWidth
	}
	if epicWidth < minEpicWidth {
		deficit := minEpicWidth - epicWidth
		epicWidth = minEpicWidth
		titleWidth -= deficit
	}
	if titleWidth > maxTitleWidth {
		extra := titleWidth - maxTitleWidth
		titleWidth = maxTitleWidth
		typesWidth += extra
	}
	return typesWidth, epicWidth, titleWidth
}

const changeTableFixedWidth = 6 + 1 + 11 + 1 + 1 + 1 + 1 + 3 + 1 + 3 + 1 + 3 + 1 + 16

func changeTableContentWidth(typesWidth, epicWidth, titleWidth int) int {
	return changeTableFixedWidth + typesWidth + epicWidth + titleWidth
}

func shrinkColumns(available, typesWidth, epicWidth, titleWidth int) (int, int, int) {
	if available <= 0 {
		return 1, 1, 1
	}
	for typesWidth+epicWidth+titleWidth > available {
		switch {
		case titleWidth > 1:
			titleWidth--
		case typesWidth > 1:
			typesWidth--
		case epicWidth > 1:
			epicWidth--
		default:
			return typesWidth, epicWidth, titleWidth
		}
	}
	return typesWidth, epicWidth, titleWidth
}

// DetailsView renders selected change details as a two-column selectable table.
func DetailsView(m Model, width int, pageSize int, phaseColors ...PhaseColors) string {
	colors := activePhaseColors(phaseColors)
	if m.Detail.ID == "" && m.Detail.Title == "" {
		return ""
	}
	if pageSize < 1 {
		pageSize = 1
	}
	m = m.ClampDetailSelection(pageSize, width)
	rows, prefix := detailViewportRows(m.Detail, pageSize, width)
	if len(rows) == 0 {
		return ""
	}
	tableWidth := innerTableWidth(width)
	contentWidth := max(20, tableWidth)
	labelWidth, textWidth := DetailColumnWidths(m.Detail, width)

	allLines := make([]string, 0, len(rows))
	for rowIndex, row := range rows {
		allLines = append(allLines, detailTableRowLines(row, labelWidth, textWidth, rowIndex-prefix == m.DetailSelected, colors)...)
		if detailDividerAfter(row) {
			allLines = append(allLines, detailDividerLine(labelWidth, textWidth))
		}
	}
	fixedLines := make([]string, 0, len(fixedDetailRows(m.Detail)))
	fixedRows := fixedDetailRows(m.Detail)
	if prefix > 0 {
		fixedRows = nil
	}
	for rowIndex, row := range fixedRows {
		selected := rowIndex-len(fixedRows) == m.DetailSelected
		fixedLines = append(fixedLines, detailTableRowLines(row, labelWidth, textWidth, selected, colors)...)
	}
	lines := append([]string(nil), fixedLines...)
	if len(lines) > pageSize {
		lines = lines[:pageSize]
	}
	scrollPageSize := pageSize - len(lines)
	offset := clampLineOffset(m.DetailOffset, len(allLines), scrollPageSize)
	end := offset + scrollPageSize
	if end > len(allLines) {
		end = len(allLines)
	}
	lines = append(lines, allLines[offset:end]...)
	for len(lines) < pageSize {
		lines = append(lines, detailBlankLine(labelWidth, textWidth))
	}
	content := ui.TruncateBlock(strings.Join(lines, "\n"), contentWidth)
	return boxedTable(content, contentWidth)
}

func activePhaseColors(values []PhaseColors) PhaseColors {
	if len(values) == 0 || values[0] == nil {
		return PhaseColors{}
	}
	return values[0]
}

func tableText(value string, limit int) string {
	value = strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
	return truncateDisplay(value, limit)
}

func truncateDisplay(value string, limit int) string {
	if limit <= 0 {
		return ""
	}
	return ui.TruncateBlock(value, limit)
}

func padLeftDisplay(value string, width int) string {
	value = truncateDisplay(value, width)
	padding := width - lipgloss.Width(value)
	if padding < 0 {
		padding = 0
	}
	return strings.Repeat(" ", padding) + value
}

func padRightDisplay(value string, width int) string {
	value = truncateDisplay(value, width)
	padding := width - lipgloss.Width(value)
	if padding < 0 {
		padding = 0
	}
	return value + strings.Repeat(" ", padding)
}

func boxedTable(content string, width int) string {
	if width < 1 {
		width = 1
	}
	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		Width(width).
		Render(content)
}

func innerTableWidth(width int) int {
	width = ui.NormalizeWidth(width)
	if width <= 4 {
		return 20
	}
	return width - 2
}

func detailTableRowLines(row DetailRow, labelWidth int, textWidth int, selected bool, phaseColors PhaseColors) []string {
	textLines := detailRowTextLines(row, textWidth)
	lines := make([]string, 0, len(textLines))
	for i, text := range textLines {
		label := ""
		if i == 0 {
			label = row.Label
		}
		labelText := padLeftDisplay(tableText(label, labelWidth), labelWidth)
		valueText := padRightDisplay(text, textWidth)
		if row.DocumentID > 0 && !row.Comment && row.DocumentType != "brief" {
			valueText = strings.Replace(valueText, "✓", lipgloss.NewStyle().Foreground(styles.AccentGreen).Render("✓"), 1)
		}
		line := labelText + " │ " + valueText
		if selected && row.Selectable {
			lines = append(lines, detailSelectedStyle(row, phaseColors).Render(line))
			continue
		}
		lines = append(lines, styles.Default.Muted.Render(labelText+" │ ")+detailValueStyle(row, phaseColors).Render(valueText))
	}
	return lines
}

func detailDividerLine(labelWidth int, textWidth int) string {
	return styles.Default.Muted.Render(strings.Repeat("─", labelWidth) + "─┼─" + strings.Repeat("─", textWidth))
}

func detailBlankLine(labelWidth int, textWidth int) string {
	return styles.Default.Muted.Render(padLeftDisplay("", labelWidth) + " │ " + padRightDisplay("", textWidth))
}

func detailValueStyle(row DetailRow, phaseColors PhaseColors) lipgloss.Style {
	switch row.Label {
	case "Slug":
		return styles.Default.AccentCyan
	case "Phase":
		return phaseStyle(row.Text, phaseColors)
	case "Title":
		return lipgloss.NewStyle().Foreground(styles.Foreground)
	case "Types":
		return lipgloss.NewStyle().Foreground(styles.AccentPurple)
	case "Agent Edit":
		return booleanIconStyle(row.Text)
	case "Complete":
		return lipgloss.NewStyle().Foreground(styles.AccentBlue)
	default:
		return styles.Default.Foreground
	}
}

func detailSelectedStyle(row DetailRow, phaseColors PhaseColors) lipgloss.Style {
	switch row.Label {
	case "Slug":
		return styles.Default.AccentCyan.Background(styles.MutedPurple)
	case "Phase":
		return phaseStyle(row.Text, phaseColors).Background(styles.MutedPurple)
	case "Title":
		return lipgloss.NewStyle().Foreground(styles.Foreground).Background(styles.MutedPurple)
	case "Types":
		return lipgloss.NewStyle().Foreground(styles.AccentPurple).Background(styles.MutedPurple)
	case "Agent Edit":
		return booleanIconStyle(row.Text).Background(styles.MutedPurple)
	case "Complete":
		return lipgloss.NewStyle().Foreground(styles.AccentBlue).Background(styles.MutedPurple)
	default:
		return styles.Default.Selection
	}
}

func booleanIconStyle(value string) lipgloss.Style {
	switch value {
	case "\u2714":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	case "\u2718":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	default:
		return styles.Default.Foreground
	}
}

func wrapWords(value string, limit int) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return strings.Split(ansi.Wrap(value, max(1, limit), ""), "\n")
}

func detailLabelWidth(rows []DetailRow) int {
	width := 5
	for _, row := range rows {
		if rowWidth := lipgloss.Width(row.Label); rowWidth > width {
			width = rowWidth
		}
	}
	return width
}

func normalizeNewlines(value string) string {
	return strings.ReplaceAll(strings.ReplaceAll(value, "\r\n", "\n"), "\r", "\n")
}

func displayRef(change dto.ChangeView) string { return displayNullable(change.Ref) }

func displayNullable(value string) string {
	if strings.TrimSpace(value) == "" || strings.EqualFold(strings.TrimSpace(value), "null") {
		return "-"
	}
	return value
}

func epicLabel(change dto.ChangeView) string {
	if name := strings.TrimSpace(change.EpicName); name != "" && !strings.EqualFold(name, "null") {
		return name
	}
	if id := strings.TrimSpace(change.EpicID); id != "" && !strings.EqualFold(id, "null") {
		return "#" + strings.TrimPrefix(strings.TrimSpace(change.EpicID), "#")
	}
	if strings.EqualFold(strings.TrimSpace(change.EpicName), "null") || strings.EqualFold(strings.TrimSpace(change.EpicID), "null") {
		return "-"
	}
	return ""
}

func formatListTimestamp(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "not a date"
	}
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
	}
	for _, layout := range layouts {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return parsed.Local().Format("2006-01-02 15:04")
		}
	}
	return "not a date"
}

// TableViewport fits the list into the shell's remaining measured height.
func TableViewport(m Model, f Filters, width, height int, colors PhaseColors) string {
	if height <= 0 {
		return ""
	}
	v := TableView(m, f, width, max(1, height-4), colors)
	lines := strings.Split(v, "\n")
	if len(lines) > height {
		return strings.Join(lines[:height], "\n")
	}
	return v
}

// DetailsViewport keeps all fields accessible through the scrollable table.
func DetailsViewport(m Model, width, height int, colors PhaseColors) string {
	if height <= 0 {
		return ""
	}
	v := DetailsView(m, width, max(1, height-2), colors)
	lines := strings.Split(v, "\n")
	if len(lines) > height {
		return strings.Join(lines[:height], "\n")
	}
	return v
}
