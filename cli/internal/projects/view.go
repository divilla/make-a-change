package projects

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

var (
    projectDetailValueStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
    projectDetailIDStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("218"))
    projectDetailTimestampStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
)

// ListTitle returns the projects list screen title.
func ListTitle() string {
    return "ProjectsListScreen - Title: Projects List"
}

// DetailTitle returns the project details screen title.
func DetailTitle() string {
    return "ProjectDetailsScreen - Title: Project Details"
}

// CreateTitle returns the new project screen title.
func CreateTitle() string {
    return "ProjectCreateScreen - Title: New Project"
}

// UpdateTitle returns the edit project screen title.
func UpdateTitle() string {
    return "ProjectUpdateScreen - Title: Edit Project"
}

// TableView renders the selectable projects table.
func TableView(m Model, width int) string {
    if m.Loading {
        return styles.Default.InputBand.Width(width).Render("Projects: loading")
    }
    if len(m.Rows) == 0 {
        return styles.Default.Muted.Render("No projects.")
    }
    nameWidth := projectTableNameWidth(m.Rows)
    rows := []string{
        tableLine("id", "Name", "Changes", "Created", "Modified", nameWidth),
    }
    for i, project := range m.Rows {
        line := tableLine(
            FormatID(project.ID),
            ProjectTableName(project.Name),
            strconv.Itoa(project.ChangeCount),
            FormatTimestamp(project.Created),
            FormatTimestamp(project.Modified),
            nameWidth,
        )
        if i == m.Selected {
            line = styles.Default.Selection.Render(line)
        }
        rows = append(rows, line)
    }
    return ui.TruncateBlock(strings.Join(rows, "\n"), width)
}

func projectTableNameWidth(projects []dto.Project) int {
    width := lipgloss.Width("Name")
    for _, project := range projects {
        if nameWidth := lipgloss.Width(ProjectTableName(project.Name)); nameWidth > width {
            width = nameWidth
        }
    }
    return width
}

// ProjectTableName trims very long names by whole words for table rendering.
func ProjectTableName(name string) string {
    const maxTrimmedNameLength = 77
    const trimSuffix = "..."

    name = strings.Join(strings.Fields(strings.TrimSpace(name)), " ")
    if len([]rune(name)) <= 80 {
        return name
    }
    words := strings.Fields(name)
    for len(words) > 0 && len([]rune(strings.Join(words, " ")+trimSuffix)) > maxTrimmedNameLength {
        words = words[:len(words)-1]
    }
    trimmed := strings.Join(words, " ")
    if trimmed != "" {
        return trimmed + trimSuffix
    }
    runes := []rune(name)
    return string(runes[:maxTrimmedNameLength-len(trimSuffix)]) + trimSuffix
}

// DetailsView renders selected project details.
func DetailsView(project dto.Project, width int) string {
    if project.ID == "" && project.Name == "" {
        return ""
    }
    lines := []string{
        detailLine("#ID", FormatID(project.ID), projectDetailIDStyle),
    }
    lines = append(lines, detailWrappedLines("Name", project.Name, styles.Default.AccentCyan, 80)...)
    lines = append(lines,
        detailLine("Changes", strconv.Itoa(project.ChangeCount), projectDetailValueStyle),
        detailLine("Created", FormatTimestamp(project.Created), projectDetailTimestampStyle),
        detailLine("Modified", FormatTimestamp(project.Modified), projectDetailTimestampStyle),
    )
    return ui.TruncateBlock(strings.Join(lines, "\n"), width)
}

func tableLine(id, name, changeCount, created, modified string, nameWidth int) string {
    return fmt.Sprintf("%6s  %-*s  %7s  %-16s  %-16s", id, nameWidth, name, changeCount, created, modified)
}

func detailLine(label, value string, valueStyle lipgloss.Style) string {
    return styles.Default.Muted.Render(fmt.Sprintf("    %8s: ", label)) + valueStyle.Render(value)
}

func detailWrappedLines(label, value string, valueStyle lipgloss.Style, limit int) []string {
    wrapped := wrapExplicitLines(value, limit)
    if len(wrapped) == 0 {
        return []string{detailLine(label, "", valueStyle)}
    }
    lines := []string{detailLine(label, wrapped[0], valueStyle)}
    indent := strings.Repeat(" ", 14)
    for _, line := range wrapped[1:] {
        lines = append(lines, styles.Default.Muted.Render(indent)+valueStyle.Render(line))
    }
    return lines
}

func wrapExplicitLines(value string, limit int) []string {
    value = strings.ReplaceAll(value, "\r\n", "\n")
    value = strings.ReplaceAll(value, "\r", "\n")
    parts := strings.Split(value, "\n")
    lines := make([]string, 0, len(parts))
    for _, part := range parts {
        wrapped := wrapWords(part, limit)
        if len(wrapped) == 0 {
            lines = append(lines, "")
            continue
        }
        lines = append(lines, wrapped...)
    }
    return lines
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

// FormatID renders a project ID without a leading number prefix.
func FormatID(id string) string {
    id = strings.TrimSpace(id)
    if id == "" {
        return "?"
    }
    return strings.TrimPrefix(id, "#")
}

// FormatTimestamp renders a project timestamp in local display format.
func FormatTimestamp(value string) string {
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

// DisplayName returns a compact project label for status and details.
func DisplayName(project dto.Project) string {
    name := strings.TrimSpace(project.Name)
    id := FormatID(project.ID)
    if name == "" {
        return id
    }
    return id + " " + name
}
