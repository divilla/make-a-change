package app

import (
	"cli/internal/changes"
	"cli/internal/configurations"
	"cli/internal/documents"
	"cli/internal/epics"
	"cli/internal/health"
	"cli/internal/help"
	"cli/internal/projects"
	"cli/internal/styles"
	"cli/internal/testcases"
	"cli/internal/ui"
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// View renders the root application shell and active screen.
func (m Model) View() string {
	lines, epicIndex := m.viewLines()
	width := terminalWidth(m.width)
	if epicIndex != 0 {
		height := m.epicViewportHeight(lines)
		switch {
		case m.historyOpen:
			lines[epicIndex] = m.history.View(width, height)
		case m.state == ChangesListState:
			lines[epicIndex] = changes.TableViewport(m.changeList, m.changeFilters(), width, height, phaseColorMap(m.optionCatalog.phases))
		case m.state == ChangeDetailsState:
			lines[epicIndex] = changes.DetailsViewport(m.changeList, width, height, phaseColorMap(m.optionCatalog.phases))
		case m.state == DocumentState:
			lines[epicIndex] = documents.View(m.document, width, height)
		case m.state == EpicDetailsState:
			lines[epicIndex] = epics.DetailsViewport(m.epicList, width, height)
		case isConfigurationState(m.state):
			lines[epicIndex] = configurations.View(m.configurations, width, height)
		default:
			lines[epicIndex] = epics.TableView(m.epicList, width, height)
		}
	}
	return styles.Default.Surface.Width(width).Render(strings.Join(lines, "\n"))
}

// viewLines reserves one placeholder line for the active epic viewport.
func (m Model) viewLines() ([]string, int) {
	width := terminalWidth(m.width)
	lines := []string{m.headerLine(width)}
	epicIndex := 0
	if m.historyOpen {
		lines = append(lines, ansi.Truncate(m.history.Metadata(), width, ""), "")
		epicIndex = len(lines) - 1
		if m.hasDropdown() {
			lines = append(lines, m.dropdownView(width))
		}
		if m.err != "" {
			lines = append(lines, styles.Default.Error.Render(m.err))
		}
		lines = append(lines, styles.Default.Footer.Width(width).Render(m.footerText()))
		return lines, epicIndex
	}
	if m.state == MainHelpState {
		lines = append(lines, "Selected project: /brief-new starts brief clarification for a new change.\nUse /changes to browse and select an existing change, then /brief-clarify.")
	}
	if m.state == ChangesHelpState {
		lines = append(lines, changes.HelpView())
	}
	if m.state == ChangeCreateState {
		lines = append(lines, "Title: "+m.changeList.Draft.Title, "Optional UUID: "+m.changeList.Draft.UUID)
	}
	if m.state == BriefState {
		lines = append(lines, m.briefView(width))
	}
	if m.state == EpicsHelpState {
		lines = append(lines, epics.HelpView())
	}
	if m.state == EpicsListState && !m.hasDropdown() {
		lines = append(lines, "", "")
		epicIndex = len(lines) - 1
	}
	if m.state == DocumentState {
		if m.documentForm {
			lines = append(lines, "Document draft: "+m.document.OwnerTable+" #"+fmt.Sprint(m.document.OwnerID)+" | type "+documents.SafeLine(m.document.DraftType)+" | agent_edit=false")
		}
		lines = append(lines, "", "")
		epicIndex = len(lines) - 1
	}
	if m.state == EpicDetailsState {
		lines = append(lines, "", "")
		epicIndex = len(lines) - 1
	}
	if m.state == ProjectsListState && !m.hasDropdown() {
		lines = append(lines, "")
		lines = append(lines, projects.TableView(m.projectList, width))
	}
	if m.state == ChangesListState {
		lines = append(lines, m.changeFiltersLine(""), "")
		epicIndex = len(lines) - 1
	}
	if m.state == ChangeDetailsState {
		lines = append(lines, "", "")
		epicIndex = len(lines) - 1
	}
	if m.state == ProjectDetailsState {
		details := projects.DetailsView(m.projectList.Detail, width)
		if m.projectList.Loading {
			details += "\nLoading project…"
		}
		if m.projectList.ShowConfig {
			details += "\n" + projects.ConfigView(m.projectList.Catalog)
		}
		if details != "" {
			lines = append(lines, "")
			lines = append(lines, details)
		}
	}
	if m.state == ConfigState {
		lines = append(lines, "")
		lines = append(lines, m.configView(width))
	}
	if m.state == BackendConfigListState || m.state == BackendConfigDetailsState || m.state == BackendConfigFormState || m.state == BackendConfigDeleteState {
		lines = append(lines, "", "")
		epicIndex = len(lines) - 1
	}
	if m.state == HealthState {
		lines = append(lines, "", health.View(m.health))
	}
	if m.state == FindInputState {
		lines = append(lines, "")
		lines = append(lines, m.inputBand(width))
	} else if m.hasDropdown() {
		if m.state != ChangesListState && m.state != ChangeDetailsState {
			lines = append(lines, "")
		}
		lines = append(lines, m.dropdownView(width))

	} else if m.state == BackendConfigFormState {
		lines = append(lines, "", m.configurationInputBand(width))
	} else {
		if m.state != ChangesListState && m.state != ChangeDetailsState {
			lines = append(lines, "")
		}
		lines = append(lines, m.inputBand(width))
	}
	if m.err != "" {
		lines = append(lines, styles.Default.Error.Render("Error: "+m.visibleDocumentText(m.err)))
	}
	if m.helpQuery != "" {
		lines = append(lines, styles.Default.Success.Render("Highlight: "+m.helpQuery))
	}
	if m.state != ChangesListState && m.state != ChangeDetailsState {
		lines = append(lines, "")
	}
	lines = append(lines, styles.Default.Footer.Width(width).Render(m.footerText()))
	if m.quitting {
		lines = append(lines, styles.Default.Success.Render("done"))
	}
	return lines, epicIndex
}

func (m Model) epicViewportHeight(lines []string) int {
	// Include wrapped feedback and footer, excluding the placeholder line.
	shellHeight := lipgloss.Height(styles.Default.Surface.Width(terminalWidth(m.width)).Render(strings.Join(lines, "\n"))) - 1
	return max(0, m.height-shellHeight)
}

func (m Model) headerLine(width int) string {
	left := appTitle()
	right := m.headerRight()
	padding := width - lipgloss.Width(left) - lipgloss.Width(right)
	if padding < 1 {
		padding = 1
	}
	return left + strings.Repeat(" ", padding) + right
}

func (m Model) headerRight() string {
	if m.historyOpen {
		return styles.Default.Foreground.Render("DocumentHistoryScreen")
	}
	title := screenTitle(m.state)
	if before, _, ok := strings.Cut(title, " - "); ok {
		title = before
	}
	return styles.Default.Foreground.Render(title)
}

func (m Model) configView(width int) string {
	return styles.Default.InputBand.Width(width).Render(renderResolvedConfig(m.appConfig))
}

func (m Model) configurationInputBand(width int) string {
	lines := promptValueLines(m.input.Value())
	row := min(max(0, m.promptCursorRow), len(lines)-1)
	col := min(max(0, m.promptCursorCol), runeCount(lines[row]))
	before := strings.Join(lines[:row], "\n")
	if row > 0 {
		before += "\n"
	}
	runes := []rune(lines[row])
	before += string(runes[:col])
	after := string(runes[col:])
	if row+1 < len(lines) {
		after += "\n" + strings.Join(lines[row+1:], "\n")
	}
	label := " " + strings.ReplaceAll(configurations.FieldNames[m.configurations.Field], "_", " ") + " > "
	caption := styles.Default.InputBand.Foreground(styles.AccentPurple).Render(label)
	entry := styles.Default.InputBand.Foreground(styles.AccentGreen).Render(configurationCursorWindow(documents.SafeLine(before), documents.SafeLine(after), max(3, width-lipgloss.Width(label))))
	return styles.Default.InputBand.Width(width).Render(caption + entry)
}

func configurationCursorWindow(before, after string, width int) string {
	afterRoom := min(ansi.StringWidth(after), width/2)
	start := max(0, ansi.StringWidth(before)-(width-2-afterRoom))
	left := ""
	if start > 0 {
		left = "…"
	}
	visibleBefore := ansi.Cut(before, start, ansi.StringWidth(before))
	remaining := max(0, width-ansi.StringWidth(left+visibleBefore)-1)
	return left + visibleBefore + "▏" + ansi.Truncate(after, remaining, "…")
}

func (m Model) changeFiltersLine(table string) string {
	line := changeFilterLabel("/phase-filter ") + changeFilterValue(m.changesFilters.phase.Label) +
		"   " + changeFilterLabel("/types-filter ") + changeFilterValue(m.changesFilters.typ.Label) +
		"   " + changeFilterLabel("/epic-filter ") + changeFilterValue(m.changesFilters.epic.Label) +
		"   " + changeFilterLabel("/find-filter ") + changeFilterValue(m.changesFilters.find)
	tableWidth := firstLineWidth(table)
	padding := tableWidth - lipgloss.Width(line)
	if padding < 0 {
		padding = 0
	}
	return strings.Repeat(" ", padding) + line
}

func changeFilterLabel(value string) string {
	return styles.Default.Muted.Render(value)
}

func changeFilterValue(value string) string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Render(value)
}

func firstLineWidth(value string) int {
	first, _, _ := strings.Cut(value, "\n")
	return lipgloss.Width(first)
}

func (m Model) helpText() string {
	if m.historyOpen {
		return m.history.Help()
	}
	if m.hasDropdown() {
		if m.dropdown.kind == dropdownConfirm {
			return "<return> select  |  <esc> or <ctrl+c> cancel"
		}
		return "<return> select  |  <esc> or <ctrl+c> cancel"
	}
	if m.state == ChangeDetailsState && m.detailEditField != "" {
		return "<return> save  |  <esc> or <ctrl+c> cancel"
	}
	switch m.state {
	case BriefState:
		return "Enter edit/answer | /confirm | /approve | /resolve | Ctrl+E editor | PgUp/PgDn scroll | Esc return"
	case DocumentState:
		if m.documentForm {
			return "<return> append selected type  |  <ctrl+e> editor  |  <esc> cancel draft"
		}
		return documents.Help(m.document)
	case ChangesListState:
		if m.changeList.Inactive {
			return "Inactive changes | Space activate | /retry reload | Esc/Ctrl+C return | Up/Down select | Type to filter"
		}
		return "Ctrl+H inactive changes | Type to filter changes  |  <ctrl+n> new change  |  <return> view  |  </> command"
	case ChangeDetailsState:
		return "Ctrl+H document history | <ctrl+n> new testcase  |  <return> edit  |  <space> toggle  |  <del> delete  |  <ctrl+ins> copy  |  </> command"
	case TestCaseCreateState:
		return testcases.CreateForm().Help
	case TestCaseUpdateState:
		return testcases.EditForm().Help
	case ChangeCreateState:
		return "<ctrl+t> title | <ctrl+u> optional UUID | <ctrl+e> brief editor | <return> save | <ctrl+c> cancel"
	case ChangeUpdateState, EpicCreateState, EpicUpdateState, ProjectCreateState, ProjectUpdateState:
		return "<return> save  |  <ctrl+c> delete prompt  |  <esc> cancel"
	case FindInputState:
		return "<return> search  |  <ctrl+c> delete prompt  |  <esc> cancel"
	case ConfigState:
		return "/return  |  <esc> or <ctrl+c> return"
	case BackendConfigListState:
		return "<up/down> select  |  <return> details  |  <ctrl+n> create  |  /retry  |  /return"
	case BackendConfigDetailsState:
		return "/edit  |  /delete  |  /retry  |  /return"
	case BackendConfigFormState:
		return "<tab/return> next field  |  <shift+tab> previous field  |  <pgup/pgdown> scroll  |  <ctrl+s> save  |  <ctrl+e> editor  |  <ctrl+g> commands  |  <esc> cancel"
	case BackendConfigDeleteState:
		return "<return> confirm delete  |  <esc> cancel"
	case HealthState:
		return "/health-v1  |  /health-legacy  |  /retry  |  /return"
	case ProjectsListState, EpicsListState:
		return "<return> view  |  </> command"
	case EpicDetailsState:
		return "<up/down> scroll  |  <pgup/pgdown> page  |  <return> edit  |  </> command"
	case ProjectDetailsState, TestCaseDetailsState:
		return "<return> edit  |  </> command"
	default:
		return "</> command  |  <esc> cancel"
	}
}

func (m Model) inputBand(width int) string {
	return m.promptBand(width, styles.Default.InputBand, styles.Gray)
}

func (m Model) promptBand(width int, base lipgloss.Style, placeholderColor lipgloss.TerminalColor) string {
	width = ui.NormalizeWidth(width)
	content := m.promptLines(width, base, placeholderColor)
	lines := []string{styles.Default.PromptEdge.Render(strings.Repeat("▄", width))}
	lines = append(lines, content...)
	lines = append(lines, styles.Default.PromptEdge.Render(strings.Repeat("▀", width)))
	return strings.Join(lines, "\n")
}

func (m Model) promptLines(width int, base lipgloss.Style, placeholderColor lipgloss.TerminalColor) []string {
	lines := promptValueLines(m.input.Value())
	padded := make([]string, 0, len(lines))
	for index, value := range lines {
		showCursor := m.input.Focused() && (m.input.Value() != "" || m.detailEditField != "") && index == m.promptCursorRow
		line := m.renderPromptLineWithStyle(value, showCursor, base, placeholderColor)
		if visible := lipgloss.Width(line); visible < width {
			line += base.Render(strings.Repeat(" ", width-visible))
		}
		padded = append(padded, line)
	}
	return padded
}

func (m Model) renderPromptLineWithStyle(value string, showCursor bool, base lipgloss.Style, placeholderColor lipgloss.TerminalColor) string {
	label := m.promptLabel()
	caption := " > "
	if label != "" {
		caption = " " + label + " > "
	}
	prompt := base.Foreground(styles.AccentPurple).Render(caption)
	if m.detailEditField == detailEditSlug {
		prompt += base.Foreground(styles.AccentPurple).Render(m.slugPrefix)
	}
	if m.input.Value() == "" {
		if showCursor {
			return prompt + promptCursorWithStyle(base)
		}
		placeholder := base.Foreground(placeholderColor).Render(m.input.Placeholder)
		return prompt + placeholder
	}
	if showCursor {
		runes := []rune(value)
		col := m.promptCursorCol
		if col < 0 {
			col = 0
		}
		if col > len(runes) {
			col = len(runes)
		}
		before := base.Foreground(styles.AccentGreen).Render(string(runes[:col]))
		after := base.Foreground(styles.AccentGreen).Render(string(runes[col:]))
		return prompt + before + promptCursorWithStyle(base) + after
	}
	return prompt + base.Foreground(styles.AccentGreen).Render(value)
}

func (m Model) promptLabel() string {
	switch m.detailEditField {
	case detailEditPRUrl:
		return "PR URL"
	case detailCreateUUID:
		return "UUID"
	case detailCreateTitle:
		return "Title"
	case detailEditTestCase:
		return "Scenario"
	case detailEditAfterChange:
		return "After Change"
	}
	if m.detailEditField != "" {
		label := strings.ReplaceAll(string(m.detailEditField), "-", " ")
		return strings.ToUpper(label[:1]) + label[1:]
	}
	switch m.state {
	case ProjectCreateState, ProjectUpdateState, EpicCreateState, EpicUpdateState:
		return "Name"
	case TestCaseCreateState, TestCaseUpdateState:
		return "Scenario"
	case ChangeCreateState:
		return "Brief"
	case ChangeUpdateState:
		return "Title"
	case FindInputState:
		return "Find"
	case DocumentState:
		if m.documentForm {
			return "Document"
		}
	case BriefState:
		if m.briefField != "" {
			return strings.ToUpper(m.briefField[:1]) + m.briefField[1:]
		}
	}
	return ""
}

func promptCursorWithStyle(base lipgloss.Style) string {
	return base.
		Background(styles.Foreground).
		Foreground(styles.Background).
		Render(" ")
}

func promptValueLines(value string) []string {
	if value == "" {
		return []string{""}
	}
	return strings.Split(value, "\n")
}

func (m Model) footerText() string {
	currentProject := "Current Project: " + m.currentProjectFooter()
	if m.status != "" {
		return fmt.Sprintf("status %s  |  %s  |  %s  |  %s", m.visibleDocumentText(m.status), m.helpText(), currentProject, footerColorStrip())
	}
	return m.helpText() + "  |  " + currentProject + "  |  " + footerColorStrip()
}

func footerColorStrip() string {
	cells := make([]string, 0, 17)
	for color := 0; color <= 16; color++ {
		label := fmt.Sprintf("%d", color)
		foreground := lipgloss.Color("15")
		switch color {
		case 7, 10, 11, 12, 14, 15, 16:
			foreground = lipgloss.Color("0")
		}
		cells = append(cells, lipgloss.NewStyle().
			Background(lipgloss.Color(label)).
			Foreground(foreground).
			Render(label))
	}
	return strings.Join(cells, " ")
}

func appTitle() string {
	return styles.Default.Title.Render("Make a change") + styles.Default.Muted.Render(" v"+Version)
}

func (m Model) currentProjectFooter() string {
	id := strings.TrimSpace(m.currentProject.ID)
	label := strings.TrimSpace(m.currentProject.Label)
	if id == "" {
		return "none"
	}
	if label == "" || label == id || label == "Project #"+id {
		return "#" + id
	}
	return "#" + id + " " + label
}

func screenTitle(state State) string {
	titles := map[State]string{
		BriefState:                 "BriefScreen - Title: Clarify Brief",
		DocumentState:              documents.DetailTitle(),
		MainState:                  "MainScreen - Title: Main",
		ChangesListState:           changes.ListTitle(),
		ChangeDetailsState:         changes.DetailTitle(),
		TestCaseDetailsState:       testcases.DetailTitle(),
		ChangeCreateState:          "ChangeCreateScreen - Title: New Change",
		ChangeUpdateState:          "ChangeUpdateScreen - Title: Edit Change",
		TestCaseCreateState:        testcases.CreateTitle(),
		TestCaseUpdateState:        testcases.UpdateTitle(),
		EpicsListState:             epics.ListTitle(),
		EpicDetailsState:           epics.DetailTitle(),
		EpicCreateState:            "EpicCreateScreen - Title: New Epic",
		EpicUpdateState:            "EpicUpdateScreen - Title: Edit Epic",
		ProjectsListState:          projects.ListTitle(),
		ProjectDetailsState:        projects.DetailTitle(),
		ProjectCreateState:         projects.CreateTitle(),
		ProjectUpdateState:         projects.UpdateTitle(),
		MainHelpState:              help.MainTitle(),
		ChangesHelpState:           help.ChangesTitle(),
		EpicsHelpState:             help.EpicsTitle(),
		ProjectsHelpState:          help.ProjectsTitle(),
		ConfigState:                "ConfigScreen - Title: Config",
		BackendConfigListState:     "BackendConfigListScreen - Title: Backend Configurations",
		BackendConfigDetailsState:  "BackendConfigDetailsScreen - Title: Configuration Details",
		BackendConfigFormState:     "BackendConfigFormScreen - Title: Configuration Editor",
		BackendConfigDeleteState:   "BackendConfigDeleteScreen - Title: Confirm Delete",
		HealthState:                "HealthScreen - Title: Backend Health",
		FindInputState:             help.FindInputTitle(),
		CommandDropDownState:       "CommandDropDownScreen - Title: Commands",
		ListSelectionDropDownState: "ListSelectionDropDownScreen - Title: Select Item",
		SelectProjectDropDown:      "SelectProjectDropDownScreen - Title: Select Project",
		SelectPhaseDropDown:        "SelectChangePhasesDropDownScreen - Title: Select Change Phases",
		SelectEpicDropDown:         "SelectEpicDropDownScreen - Title: Select Epic",
		SelectTypesDropDown:        "SelectChangeTypesDropDownScreen - Title: Select Change Types",
		ChangeDeleteConfirmation:   "ChangeDeleteConfirmationScreen - Title: Are you sure?",
		TestCaseDeleteConfirmation: "TestCaseDeleteConfirmationScreen - Title: Are you sure?",
		EpicDeleteConfirmation:     "EpicDeleteConfirmationScreen - Title: Are you sure?",
		ProjectDeleteConfirmation:  "ProjectDeleteConfirmationScreen - Title: Are you sure?",
		DoneState:                  "DoneScreen - Title: Done",
	}
	if title, ok := titles[state]; ok {
		return title
	}
	return "UnknownScreen - Title: Unknown"
}

func terminalWidth(width int) int {
	return ui.NormalizeWidth(width)
}

func (m Model) changeTableRows() int {
	lines, _ := m.viewLines()
	extra := 4
	if m.state == ChangeDetailsState {
		extra = 2
	}
	return max(1, m.epicViewportHeight(lines)-extra)
}

func (m Model) visibleDocumentText(value string) string {
	if m.state == DocumentState || isConfigurationState(m.state) || m.state == HealthState {
		return documents.SafeLine(value)
	}
	return value
}
