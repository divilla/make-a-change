package app

import (
	"cli/internal/changes"
	"cli/internal/dto"
	"cli/internal/epics"
	"cli/internal/projects"
	"cli/internal/styles"
	"cli/internal/testcases"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
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
			if previous == EpicDetailsState {
				m.state = EpicDetailsState
				return m.beginEpic(epics.Delete, m.epicList.Detail.ID, "")
			}
			if previous == ProjectDetailsState {
				m.state = ProjectDetailsState
				return m.beginProject(projects.Delete, m.projectList.Detail.ID, "")
			}
			if previous == ChangeDetailsState && target == ChangesListState {
				m.state = ChangeDetailsState
				m.status = "deleting change"
				id, _ := changeNumericID(m.changeList.Detail)
				return m.beginChange(changes.Delete, id, changes.Input{})
			}
			if previous == ChangeDetailsState && target == ChangeDetailsState {
				m.state = ChangeDetailsState
				return m.beginTestCase(testcases.Delete, "", "", false)
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
	selected := m.selectedOption()
	if selected.Label == "" {
		m.err = "no matching option"
		return m, nil
	}
	if m.dropdown.editField == detailEditDocument {
		m.changeList.Draft.DocumentType = selected.ID
		m.state = ChangeDetailsState
		m.dropdown = dropdownModel{}
		return m.beginDetailTextEditor(detailEditDocument)
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
		id, _ := changeNumericID(change)
		return m.beginChange(changes.Types, id, changes.Input{Types: pending})
	}
	if m.dropdown.editField != "" {
		field := m.dropdown.editField
		change := m.changeList.Detail
		m.state = m.dropdown.onSelect
		m.status = "saving " + string(field)
		m.dropdown = dropdownModel{}
		id, _ := changeNumericID(change)
		in := changes.Input{Value: selected.ID}
		op := changes.Phase
		if field == detailEditEpic {
			op = changes.Epic
			var err error
			in.Association, err = selectedEpicID(selected)
			if err != nil {
				m.err = err.Error()
				return m, nil
			}
		}
		return m.beginChange(op, id, in)
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
		id, err := strconv.Atoi(selected.ID)
		if err != nil || id <= 0 {
			m.err = "failed to save project_id: current project ID must be a positive number"
			return m, nil
		}
		m.appConfig.ProjectID = id
		m.epicList = m.epicList.Scope(id)
		m.state = m.dropdown.onSelect
		m.status = "selected " + selected.Label + "; saving config"
		m.dropdown = dropdownModel{}
		m.selectionGeneration++
		m.catalogGeneration++
		m.optionCatalog = optionCatalog{}
		m.selectedConfigSlug = ""
		m.changesFilters = changesFilters{}
		m.changeList = m.changeList.Scope(id)
		m, save := m.persistCurrentProject()
		return m, tea.Batch(save, optionCatalogCommand(m.ctx, m.client, id, m.catalogGeneration), currentProjectCommand(m.ctx, m.client, id, m.selectionGeneration))
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
	if m.dropdown.kind == dropdownCommand {
		rendered += "\n" + styles.Default.Background.Width(width).Render("")
	}
	return rendered
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
