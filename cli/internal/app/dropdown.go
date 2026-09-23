package app

import (
    "regexp"
    "slices"
    "strings"

    "cli/internal/agent"
    "cli/internal/dto"
    "cli/internal/styles"

    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
)

func (m Model) handleDropdownKey(key string, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
    switch key {
    case "ctrl+c":
        if m.dropdown.kind == dropdownConfirm {
            return m.cancelDropdown()
        }
        return m, nil
    case "esc":
        return m.cancelDropdown()
    case "up":
        m.moveHighlight(-1)
        return m, nil
    case "down":
        m.moveHighlight(1)
        return m, nil
    case "backspace":
        if len(m.dropdown.filter) > 0 {
            m.dropdown.filter = m.dropdown.filter[:len(m.dropdown.filter)-1]
            m.dropdown.highlighted = 0
        }
        return m, nil
    case "enter":
        if m.dropdown.loading {
            return m, nil
        }
        return m.confirmDropdown()
    case " ", "space":
        if m.dropdown.loading {
            return m, nil
        }
        if m.dropdown.editField == detailEditTypes {
            return m.togglePendingChangeType()
        }
        return m, nil
    }
    if len(msg.Runes) > 0 {
        m.dropdown.filter += string(msg.Runes)
        m.dropdown.highlighted = 0
    }
    return m, nil
}

func (m *Model) openCommandDropdown() {
    options := commandOptions(m.state)
    m.previousState = m.state
    m.dropdown = dropdownModel{
        kind:     dropdownCommand,
        state:    CommandDropDownState,
        previous: m.state,
        onSelect: m.state,
        label:    "Commands",
        options:  options,
    }
    m.status = string(CommandDropDownState)
}

func (m *Model) openDropdown(state State, kind dropdownKind, previous State, onSelect State, label string, options []dto.Option, loading bool) {
    m.previousState = previous
    m.state = state
    m.dropdown = dropdownModel{
        kind:     kind,
        state:    state,
        previous: previous,
        onSelect: onSelect,
        label:    label,
        options:  options,
        loading:  loading,
    }
    m.status = string(state)
}

func (m *Model) openSelectorDropdown(state State, previous State, onSelect State, label string, source selectorSource) {
    m.previousState = previous
    m.state = state
    m.dropdown = dropdownModel{
        kind:     dropdownSelect,
        state:    state,
        previous: previous,
        onSelect: onSelect,
        source:   source,
        label:    label,
        loading:  true,
    }
    m.status = label
}

func (m *Model) openFilterDropdown(label string, source selectorSource, field filterField) {
    m.previousState = ChangesListState
    m.state = ChangesListState
    m.dropdown = dropdownModel{
        kind:        dropdownSelect,
        previous:    ChangesListState,
        onSelect:    ChangesListState,
        source:      source,
        filterField: field,
        label:       label,
        loading:     true,
    }
    m.status = label
}

func (m Model) cancelDropdown() (tea.Model, tea.Cmd) {
    m.state = m.dropdown.previous
    m.status = "cancel"
    m.dropdown = dropdownModel{}
    return m, nil
}

func (m Model) confirmDropdown() (tea.Model, tea.Cmd) {
    if m.dropdown.kind == dropdownConfirm {
        selected := m.selectedOption()
        if selected.Label == "" {
            m.err = "confirmation requires /yes or /no"
            return m, nil
        }
        switch selected.ID {
        case "/yes":
            target := m.dropdown.onSelect
            previous := m.dropdown.previous
            m.dropdown = dropdownModel{}
            if previous == ChangeDetailsState && target == ChangesListState {
                m.state = ChangeDetailsState
                m.status = "deleting change"
                return m, changeDeleteCommand(m.client, m.changeList.Detail, target)
            }
            if previous == ChangeDetailsState && target == ChangeDetailsState {
                m.state = ChangeDetailsState
                m.status = "deleting test case"
                return m, testCaseDeleteCommand(m.client, m.activeTestCase)
            }
            return m.arrive(target, "confirmed")
        case "/no", "/cancel":
            return m.cancelDropdown()
        default:
            m.err = "confirmation requires /yes or /no"
            return m, nil
        }
    }

    if m.dropdown.kind == dropdownCommand {
        selected := m.selectedOption()
        if selected.ID == "" {
            m.err = "unknown command"
            return m, nil
        }
        return m.executeCommandFrom(m.dropdown.previous, selected.ID)
    }
    if m.dropdown.kind == dropdownAgent {
        selected := m.selectedOption()
        m.dropdown = dropdownModel{}
        switch selected.ID {
        case "/resume":
            content, err := m.agentFlow.Workspace.ReadDef()
            if err != nil {
                m.err = err.Error()
                m.status = "agent failed"
                return m, nil
            }
            return m.openAgentDefEditor(false, content)
        case "/clear":
            return m.openAgentDefEditor(true, "")
        case "/cancel":
            return m.discardAgentDef(ChangesListState, "cancel", true)
        default:
            m.err = "unknown command"
            return m, nil
        }
    }
    if m.dropdown.kind == dropdownPersistedDef {
        selected := m.selectedOption()
        m.dropdown = dropdownModel{}
        switch selected.ID {
        case "/fix":
            if err := m.agentFlow.Workspace.PromoteOutput(); err != nil {
                m = m.showPersistedChangeError(err)
                return m, nil
            }
            content, err := m.agentFlow.Workspace.ReadDef()
            if err != nil {
                m = m.showPersistedChangeError(err)
                return m, nil
            }
            m.agentFlow.DefEntryContent = content
            m.agentFlow.Stage = agent.StagePersistedDefEntry
            m.state = UpdateDefState
            m.status = "agent def"
            return m.openPersistentEditor(UpdateDefState, m.agentFlow.Workspace.DefPath())
        case "/cancel":
            m.agentFlow.Stage = agent.StageIdle
            m.state = ChangeDetailsState
            m.status = "cancel"
            return m, nil
        default:
            m.err = "unknown command"
            return m, nil
        }
    }
    if m.dropdown.kind == dropdownDef {
        selected := m.selectedOption()
        previous := m.dropdown.previous
        m.dropdown = dropdownModel{}
        switch selected.ID {
        case "/fix":
            if err := m.agentFlow.Workspace.PromoteOutput(); err != nil {
                m.err = err.Error()
                m.status = "agent failed"
                return m, nil
            }
            return m.openAgentDefEditor(false, m.agentFlow.DefEntryContent)
        case "/edit":
            if previous == UpdateDefState {
                m.state = UpdateDefState
                if err := m.agentFlow.Workspace.WriteDef(m.agentFlow.DefEntryContent); err != nil {
                    m.err = err.Error()
                    m.status = "agent failed"
                    return m, nil
                }
                return m.openPersistentEditor(UpdateDefState, m.agentFlow.Workspace.DefPath())
            }
            content, err := m.agentFlow.Workspace.ReadDef()
            if err != nil {
                m.err = err.Error()
                m.status = "agent failed"
                return m, nil
            }
            return m.openAgentDefEditor(false, content)
        case "/cancel", "/no":
            return m.discardAgentDef(ChangesListState, "cancel", true)
        case "/yes":
            projectID, err := currentProjectNumericID(m.currentProject.ID)
            if err != nil {
                m.err = err.Error()
                m.status = "validation failed"
                return m, nil
            }
            m.state = CreateDefState
            m.status = "creating change"
            return m, agentChangeCreateForRewriteCommand(m.client, projectID, m.agentFlow.Workspace)
        default:
            m.err = "unknown command"
            return m, nil
        }
    }

    selected := m.selectedOption()
    if selected.Label == "" {
        m.err = "no matching option"
        return m, nil
    }
    if m.dropdown.editField == detailEditTypes {
        change := m.changeList.Detail
        m.state = m.dropdown.onSelect
        m.status = "selected Types"
        if !m.dropdown.typesChanged {
            m.dropdown = dropdownModel{}
            return m, nil
        }
        pending := append([]string(nil), m.dropdown.pendingTypes...)
        m.dropdown = dropdownModel{}
        return m, changeDetailTypesUpdateCommand(m.client, change, pending)
    }
    if m.dropdown.editField != "" {
        field := m.dropdown.editField
        change := m.changeList.Detail
        m.state = m.dropdown.onSelect
        m.status = "saving " + string(field)
        m.dropdown = dropdownModel{}
        return m, changeDetailFieldUpdateCommand(m.client, change, field, selected)
    }
    if m.dropdown.filterField != "" {
        if selected.ID == "/clear" {
            m.clearChangesFilter(m.dropdown.filterField)
            m.state = m.dropdown.onSelect
            m.status = "cleared " + string(m.dropdown.filterField) + " filter"
            m.dropdown = dropdownModel{}
            return m, nil
        }
        m.setChangesFilter(m.dropdown.filterField, selected)
    }
    if m.state == SelectProjectDropDown {
        m.currentProject = selected
        if err := m.saveCurrentProject(selected); err != nil {
            m.err = err.Error()
        }
    }
    m.state = m.dropdown.onSelect
    m.status = "selected " + selected.Label
    m.dropdown = dropdownModel{}
    return m, nil
}

func (m *Model) openConfirmation(state, previous, onYes State) {
    m.openDropdown(state, dropdownConfirm, previous, onYes, "Are you sure?", []dto.Option{
        {ID: "/yes", Label: "/yes"},
        {ID: "/no", Label: "/no"},
    }, false)
}

func (m Model) dropdownView(width int) string {
    if m.dropdown.loading {
        return styles.Default.InputBand.Width(width).Render(m.dropdown.label + ": loading")
    }
    options := m.filteredOptions()
    if len(options) == 0 {
        return styles.Default.InputBand.Width(width).Render(m.dropdown.label + ": no options")
    }
    promptLines := make([]string, 0, len(options)+2)
    promptLines = append(promptLines, m.dropdown.label+" "+m.dropdown.filter)
    if m.dropdown.editField == detailEditTypes {
        promptLines = append(promptLines, "press <space> to change")
    }
    for i, option := range options {
        line := m.dropdownLine(option)
        if i == m.dropdown.highlighted {
            line = styles.Default.Selection.Render(line)
        }
        promptLines = append(promptLines, line)
    }
    rendered := styles.Default.InputBand.Width(width).Render(strings.Join(promptLines, "\n"))
    if m.dropdownShowsDefPreview() {
        if def := m.renderDefPreview(width); strings.TrimSpace(def) != "" {
            rendered = strings.TrimRight(def, "\n") + "\n\n" + rendered
        }
    }
    if m.dropdown.kind == dropdownCommand {
        rendered += "\n" + styles.Default.Background.Width(width).Render("")
    }
    return rendered
}

func (m Model) dropdownShowsDefPreview() bool {
    return m.state == CreateDefState || m.state == UpdateDefState
}

func (m Model) renderDefPreview(_ int) string {
    def := strings.TrimSpace(m.defPreviewContent())
    if def == "" {
        return ""
    }
    return strings.TrimRight(highlightMarkdownPreview(def), "\n")
}

func (m Model) defPreviewContent() string {
    if m.dropdownShowsDefPreview() && strings.TrimSpace(m.agentFlow.Workspace.Dir) != "" {
        if content, err := m.agentFlow.Workspace.ReadDef(); err == nil {
            return content
        }
    }
    return m.agentFlow.DefEntryContent
}

var (
    markdownHeadingLine     = regexp.MustCompile(`^#.*`)
    markdownHeadingRuleLine = regexp.MustCompile(`^(=+|-+)$`)
    markdownQuoteLine       = regexp.MustCompile(`^[ \t]*>.*`)
    markdownListMarker      = regexp.MustCompile(`^(    |\t)* ? ? ?(\*|\+|-|[0-9]+\.)( +|\t)`)
    markdownIndentedCode    = regexp.MustCompile(`^(    |\t)+ *([^*+0-9> \t-]|[*+-]\S|[0-9][^.]).*`)
    markdownLink            = regexp.MustCompile(`\[[^]]+\]\([^)]+\)`)
    markdownLinkLabel       = regexp.MustCompile(`!?\[[^]]+\]`)
    markdownStrong          = regexp.MustCompile(`\*\*[^*]+\*\*|__[^_]+__`)
    markdownEmphasis        = regexp.MustCompile(`\*[^* \t][^*]*\*|_[^_ \t][^_]*_`)
    markdownStrike          = regexp.MustCompile(`~~[^~]+~~`)
    markdownInlineCode      = regexp.MustCompile("`[^`]+`")
    markdownHTML            = regexp.MustCompile(`<!--.*-->|<[^>]+>`)
)

var markdownPreviewStyles = struct {
    text       lipgloss.Style
    quote      lipgloss.Style
    listMarker lipgloss.Style
    emphasis   lipgloss.Style
    strong     lipgloss.Style
    strike     lipgloss.Style
    link       lipgloss.Style
    linkLabel  lipgloss.Style
    code       lipgloss.Style
    heading    lipgloss.Style
    html       lipgloss.Style
}{
    text:       lipgloss.NewStyle().Foreground(lipgloss.Color("15")),
    quote:      lipgloss.NewStyle().Foreground(lipgloss.Color("5")),
    listMarker: lipgloss.NewStyle().Foreground(lipgloss.Color("13")),
    emphasis:   lipgloss.NewStyle().Foreground(lipgloss.Color("2")),
    strong:     lipgloss.NewStyle().Foreground(lipgloss.Color("10")),
    strike:     lipgloss.NewStyle().Foreground(lipgloss.Color("1")),
    link:       lipgloss.NewStyle().Foreground(lipgloss.Color("12")),
    linkLabel:  lipgloss.NewStyle().Foreground(lipgloss.Color("13")),
    code:       lipgloss.NewStyle().Foreground(lipgloss.Color("14")),
    heading:    lipgloss.NewStyle().Foreground(lipgloss.Color("11")),
    html:       lipgloss.NewStyle().Foreground(lipgloss.Color("6")),
}

func highlightMarkdownPreview(markdown string) string {
    lines := strings.Split(markdown, "\n")
    highlighted := make([]string, 0, len(lines))
    inFence := false
    for _, line := range lines {
        if strings.HasPrefix(strings.TrimSpace(line), "```") {
            highlighted = append(highlighted, markdownPreviewStyles.code.Render(line))
            inFence = !inFence
            continue
        }
        if inFence || markdownIndentedCode.MatchString(line) {
            highlighted = append(highlighted, markdownPreviewStyles.code.Render(line))
            continue
        }
        switch {
        case markdownHeadingLine.MatchString(line), markdownHeadingRuleLine.MatchString(line):
            highlighted = append(highlighted, markdownPreviewStyles.heading.Render(line))
        case markdownQuoteLine.MatchString(line):
            highlighted = append(highlighted, markdownPreviewStyles.quote.Render(line))
        default:
            highlighted = append(highlighted, highlightMarkdownInline(line))
        }
    }
    return strings.Join(highlighted, "\n")
}

func highlightMarkdownInline(line string) string {
    spans := make([]markdownSpan, 0, 8)
    spans = appendMarkdownSpans(spans, line, markdownListMarker, markdownPreviewStyles.listMarker)
    spans = appendMarkdownSpans(spans, line, markdownLink, markdownPreviewStyles.link)
    spans = appendMarkdownSpans(spans, line, markdownLinkLabel, markdownPreviewStyles.linkLabel)
    spans = appendMarkdownSpans(spans, line, markdownStrong, markdownPreviewStyles.strong)
    spans = appendMarkdownSpans(spans, line, markdownEmphasis, markdownPreviewStyles.emphasis)
    spans = appendMarkdownSpans(spans, line, markdownStrike, markdownPreviewStyles.strike)
    spans = appendMarkdownSpans(spans, line, markdownInlineCode, markdownPreviewStyles.code)
    spans = appendMarkdownSpans(spans, line, markdownHTML, markdownPreviewStyles.html)
    if len(spans) == 0 {
        return markdownPreviewStyles.text.Render(line)
    }
    slices.SortFunc(spans, func(a, b markdownSpan) int {
        if a.start != b.start {
            return a.start - b.start
        }
        return b.end - a.end
    })
    var b strings.Builder
    pos := 0
    for _, span := range spans {
        if span.start < pos {
            continue
        }
        b.WriteString(markdownPreviewStyles.text.Render(line[pos:span.start]))
        b.WriteString(span.style.Render(line[span.start:span.end]))
        pos = span.end
    }
    b.WriteString(markdownPreviewStyles.text.Render(line[pos:]))
    return b.String()
}

type markdownSpan struct {
    start int
    end   int
    style lipgloss.Style
}

func appendMarkdownSpans(spans []markdownSpan, line string, expression *regexp.Regexp, style lipgloss.Style) []markdownSpan {
    for _, index := range expression.FindAllStringIndex(line, -1) {
        spans = append(spans, markdownSpan{start: index[0], end: index[1], style: style})
    }
    return spans
}

func (m Model) dropdownLine(option dto.Option) string {
    label := option.Label
    if m.dropdown.editField == detailEditTypes {
        prefix := "+"
        if selectedChangeType(m.dropdown.pendingTypes, option) {
            prefix = "-"
        }
        return "    " + prefix + strings.TrimLeft(label, "+-")
    }
    if m.dropdown.editField == detailEditPhase {
        return "    -" + strings.TrimPrefix(label, "-")
    }
    if option.ID == "/clear" {
        return "    " + label
    }
    if m.dropdown.filterField != "" {
        return "    -" + strings.TrimPrefix(label, "-")
    }
    return "    " + label
}

func (m Model) togglePendingChangeType() (tea.Model, tea.Cmd) {
    selected := m.selectedOption()
    if selected.Label == "" {
        m.err = "no matching option"
        return m, nil
    }
    m.dropdown.pendingTypes = toggleChangeType(m.dropdown.pendingTypes, selected)
    m.dropdown.typesChanged = !sameTypeSet(m.dropdown.pendingTypes, m.changeList.Detail.ChangeTypes)
    return m, nil
}

func sameTypeSet(a, b []string) bool {
    normalizedA := normalizeTypeSet(a)
    normalizedB := normalizeTypeSet(b)
    if len(normalizedA) != len(normalizedB) {
        return false
    }
    for i := range normalizedA {
        if normalizedA[i] != normalizedB[i] {
            return false
        }
    }
    return true
}

func selectedChangeType(current []string, option dto.Option) bool {
    for _, changeType := range current {
        if changeType != "" && (changeType == option.ID || changeType == option.Label) {
            return true
        }
    }
    return false
}

func (m Model) isDropdownState() bool {
    if m.hasDropdown() {
        return true
    }
    switch m.state {
    case CommandDropDownState, ListSelectionDropDownState, SelectProjectDropDown, SelectPhaseDropDown,
        SelectEpicDropDown, SelectTypesDropDown, ChangeDeleteConfirmation, TestCaseDeleteConfirmation,
        EpicDeleteConfirmation, ProjectDeleteConfirmation:
        return true
    default:
        return false
    }
}

func (m Model) hasDropdown() bool {
    return m.dropdown.kind != ""
}

func (m Model) selectedOption() dto.Option {
    options := m.filteredOptions()
    if len(options) == 0 {
        return dto.Option{}
    }
    if m.dropdown.highlighted >= len(options) {
        m.dropdown.highlighted = len(options) - 1
    }
    return options[m.dropdown.highlighted]
}

func (m *Model) moveHighlight(delta int) {
    options := m.filteredOptions()
    if len(options) == 0 {
        m.dropdown.highlighted = 0
        return
    }
    m.dropdown.highlighted = (m.dropdown.highlighted + delta + len(options)) % len(options)
}

func (m Model) filteredOptions() []dto.Option {
    filter := strings.ToLower(strings.TrimSpace(m.dropdown.filter))
    if filter == "" {
        return m.dropdown.options
    }
    var options []dto.Option
    for _, option := range m.dropdown.options {
        label := strings.ToLower(option.Label)
        id := strings.ToLower(option.ID)
        if !strings.HasPrefix(filter, "/") {
            label = strings.TrimPrefix(label, "/")
            id = strings.TrimPrefix(id, "/")
        }
        if strings.Contains(label, filter) || strings.Contains(id, filter) {
            options = append(options, option)
        }
    }
    return options
}
