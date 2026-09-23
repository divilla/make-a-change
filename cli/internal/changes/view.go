package changes

import (
    "fmt"
    "strconv"
    "strings"
    "time"

    "cli/internal/dto"
    "cli/internal/styles"
    "cli/internal/ui"

    "github.com/charmbracelet/lipgloss"
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
    if len(rows) == 0 {
        if len(m.Rows) == 0 {
            return styles.Default.Muted.Render("No changes.")
        }
        return styles.Default.Muted.Render("No changes match filters.")
    }
    if pageSize < 1 {
        pageSize = 1
    }
    selected := m.ClampSelection(filters, pageSize).Selected
    offset := clampOffset(m.Offset, selected, len(rows), pageSize)
    end := offset + pageSize
    if end > len(rows) {
        end = len(rows)
    }
    terminalTableWidth := innerTableWidth(width)
    typesWidth, epicWidth, titleWidth := changeTableColumnWidths(terminalTableWidth)
    tableWidth := changeTableContentWidth(typesWidth, epicWidth, titleWidth)
    lines := []string{styles.Default.Muted.Render(changeTableLine("#Ref", "Phase", "Types", "Epic", "Title", "Don", "Tot", "%", "Modified", typesWidth, epicWidth, titleWidth))}
    for i, change := range rows[offset:end] {
        rowIndex := offset + i
        line := changeTableRowLine(
            displayRef(change),
            change.ChangePhase,
            strings.Join(change.ChangeTypes, "|"),
            epicLabel(change),
            change.Title,
            strconv.Itoa(change.Done),
            strconv.Itoa(change.Total),
            strconv.Itoa(change.Completed),
            formatListTimestamp(change.Modified),
            typesWidth,
            epicWidth,
            titleWidth,
            rowIndex == selected,
            colors,
        )
        lines = append(lines, line)
    }
    for len(lines) < pageSize+1 {
        lines = append(lines, "")
    }
    lines = append(lines, styles.Default.Foreground.Render(fmt.Sprintf("Rows %d-%d of %d", offset+1, end, len(rows))))
    content := ui.TruncateBlock(strings.Join(lines, "\n"), tableWidth)
    return boxedTable(content, tableWidth)
}

func changeTableLine(ref, phase, types, epic, title, done, total, completed, modified string, typesWidth, epicWidth, titleWidth int) string {
    return fmt.Sprintf(
        "%6s %-10s %-*s %-*s %-*s %3s %3s %3s %-16s",
        tableText(ref, 6),
        tableText(phase, 10),
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

func changeTableRowLine(ref, phase, types, epic, title, done, total, completed, modified string, typesWidth, epicWidth, titleWidth int, selected bool, phaseColors PhaseColors) string {
    prefix := fmt.Sprintf("%6s ", tableText(ref, 6))
    phaseValue := fmt.Sprintf("%-10s", tableText(phase, 10))
    beforeTitle := fmt.Sprintf(
        " %-*s %-*s ",
        typesWidth,
        tableText(types, typesWidth),
        epicWidth,
        tableText(epic, epicWidth),
    )
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
        return base.Render(prefix) +
            phaseStyle(phase, phaseColors).Background(lipgloss.Color("60")).Render(phaseValue) +
            base.Render(beforeTitle) +
            base.Foreground(lipgloss.Color("15")).Render(titleValue) +
            base.Render(beforeCompleted) +
            base.Foreground(lipgloss.Color("14")).Render(completedValue) +
            base.Render(afterCompleted)
    }

    return styles.Default.Muted.Render(prefix) +
        phaseStyle(phase, phaseColors).Render(phaseValue) +
        styles.Default.Muted.Render(beforeTitle) +
        lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Render(titleValue) +
        styles.Default.Muted.Render(beforeCompleted) +
        lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Render(completedValue) +
        styles.Default.Muted.Render(afterCompleted)
}

func phaseStyle(phase string, phaseColors PhaseColors) lipgloss.Style {
    key := strings.TrimSpace(phase)
    color := strings.TrimSpace(phaseColors[key])
    if color == "" {
        color = defaultPhaseColors[key]
    }
    if color == "" {
        color = "240"
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

const changeTableFixedWidth = 6 + 1 + 10 + 1 + 1 + 1 + 1 + 3 + 1 + 3 + 1 + 3 + 1 + 16

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
    rows := DetailRows(m.Detail)
    if len(rows) == 0 {
        return ""
    }
    tableWidth := innerTableWidth(width)
    contentWidth := max(20, tableWidth)
    labelWidth, textWidth := DetailColumnWidths(m.Detail, width)

    allLines := make([]string, 0, len(rows))
    for rowIndex, row := range rows {
        allLines = append(allLines, detailTableRowLines(row, labelWidth, textWidth, rowIndex == m.DetailSelected, colors)...)
        if detailDividerAfter(row) {
            allLines = append(allLines, detailDividerLine(labelWidth, textWidth))
        }
    }
    fixedLines := make([]string, 0, len(fixedDetailRows(m.Detail)))
    fixedRows := fixedDetailRows(m.Detail)
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
    if lipgloss.Width(value) <= limit {
        return value
    }
    var b strings.Builder
    for _, r := range value {
        next := b.String() + string(r)
        if lipgloss.Width(next) > limit {
            break
        }
        b.WriteRune(r)
    }
    return b.String()
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
        valueText := padRightDisplay(tableText(text, textWidth), textWidth)
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
    case "Phase":
        return phaseStyle(row.Text, phaseColors)
    case "Title":
        return lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
    case "Agent Edit":
        return booleanIconStyle(row.Text)
    case "Complete":
        return lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
    default:
        return styles.Default.Foreground
    }
}

func detailSelectedStyle(row DetailRow, phaseColors PhaseColors) lipgloss.Style {
    switch row.Label {
    case "Phase":
        return phaseStyle(row.Text, phaseColors).Background(lipgloss.Color("60"))
    case "Title":
        return lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Background(lipgloss.Color("60"))
    case "Agent Edit":
        return booleanIconStyle(row.Text).Background(lipgloss.Color("60"))
    case "Complete":
        return lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Background(lipgloss.Color("60"))
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
    words := strings.Fields(strings.TrimSpace(value))
    if len(words) == 0 {
        return nil
    }
    lines := make([]string, 0, len(words))
    current := words[0]
    for _, word := range words[1:] {
        if len([]rune(current))+1+len([]rune(word)) > limit {
            lines = append(lines, current)
            current = word
            continue
        }
        current += " " + word
    }
    lines = append(lines, current)
    return lines
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

func displayRef(change dto.Change) string {
    ref := strings.TrimPrefix(strings.TrimSpace(change.Ref), "#")
    if ref != "" {
        if value, err := strconv.Atoi(ref); err == nil {
            return fmt.Sprintf("%06d", value)
        }
        return ref
    }
    return ""
}

func epicLabel(change dto.Change) string {
    if strings.TrimSpace(change.EpicName) != "" {
        return strings.TrimSpace(change.EpicName)
    }
    if strings.TrimSpace(change.EpicID) != "" {
        return "#" + strings.TrimPrefix(strings.TrimSpace(change.EpicID), "#")
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
            return parsed.Format("2006-01-02 15.04")
        }
    }
    return "not a date"
}
