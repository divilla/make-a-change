package app

import (
	"cli/internal/changes"
	"cli/internal/dto"
	"cli/internal/epics"
	"cli/internal/projects"
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func filterOptions(options []dto.Option) []dto.Option {
	withClear := append([]dto.Option(nil), options...)
	return append(withClear, dto.Option{ID: "@clear", Label: "@clear"})
}

func (m Model) dropdownCurrentValueIndex(options []dto.Option) int {
	if len(options) == 0 {
		return 0
	}
	if m.dropdown.editField != "" {
		switch m.dropdown.editField {
		case detailEditPhase:
			return optionIndex(options, m.changeList.Detail.ChangePhase, m.changeList.Detail.ChangePhase)
		case detailEditEpic:
			if (m.changeList.Detail.EpicID == "" && m.changeList.Detail.EpicName == "") || m.changeList.Detail.EpicID == "null" {
				return optionIndex(options, "@none", "@none")
			}
			return optionIndex(options, m.changeList.Detail.EpicID, m.changeList.Detail.EpicName)
		case detailEditTypes:
			for i, option := range options {
				if selectedChangeType(m.changeList.Detail.ChangeTypes, option) {
					return i
				}
			}
		}
	}
	if m.dropdown.filterField != "" {
		switch m.dropdown.filterField {
		case filterPhase:
			return optionIndex(options, m.changesFilters.phase.ID, m.changesFilters.phase.Label)
		case filterEpic:
			return optionIndex(options, m.changesFilters.epic.ID, m.changesFilters.epic.Label)
		case filterTypes:
			return optionIndex(options, m.changesFilters.typ.ID, m.changesFilters.typ.Label)
		}
	}
	if m.state == SelectProjectDropDown {
		return optionIndex(options, m.currentProject.ID, m.currentProject.Label)
	}
	return 0
}

func optionIndex(options []dto.Option, id string, label string) int {
	for i, option := range options {
		if id != "" && option.ID == id {
			return i
		}
		if label != "" && option.Label == label {
			return i
		}
	}
	return 0
}

func phaseColorMap(options []dto.Option) changes.PhaseColors {
	colors := make(changes.PhaseColors, len(options))
	for _, option := range options {
		id := strings.TrimSpace(option.ID)
		if id == "" {
			id = strings.TrimSpace(option.Label)
		}
		color := strings.TrimSpace(option.Color)
		if id != "" && color != "" {
			colors[id] = color
		}
	}
	return colors
}

func (m *Model) setChangesFilter(field filterField, option dto.Option) {
	switch field {
	case filterPhase:
		m.changesFilters.phase = option
	case filterEpic:
		m.changesFilters.epic = option
	case filterTypes:
		m.changesFilters.typ = option
	}
	m.clampChangeListSelection()
}

func (m *Model) clampChangeListSelection() {
	m.changeList = m.changeList.ClampSelection(m.changeFilters(), m.changeTableRows())
}

func (m *Model) rememberSelectedChange() {
	rows := changes.FilteredRows(m.changeList.Rows, m.changeFilters())
	if m.changeList.Selected >= 0 && m.changeList.Selected < len(rows) {
		m.changeSelectionID = rows[m.changeList.Selected].ID
	}
}

func (m *Model) restoreSelectedChange() {
	rows := changes.FilteredRows(m.changeList.Rows, m.changeFilters())
	for i, row := range rows {
		if row.ID == m.changeSelectionID {
			m.changeList.Selected = i
			break
		}
	}
	m.clampChangeListSelection()
	if len(rows) == 0 {
		m.changeSelectionID = ""
	} else {
		m.changeSelectionID = rows[m.changeList.Selected].ID
	}
}

func (m Model) changeFilters() changes.Filters {
	find := m.changesFilters.find
	if m.state == ChangesListState && !m.hasDropdown() {
		prompt := strings.TrimSpace(m.input.Value())
		if prompt != "" && !strings.HasPrefix(prompt, "/") {
			find = strings.TrimSpace(find + " " + prompt)
		}
	}
	return changes.Filters{
		Phase: m.changesFilters.phase,
		Epic:  m.changesFilters.epic,
		Type:  m.changesFilters.typ,
		Find:  find,
	}
}

func selectorSourceForState(state State) selectorSource {
	switch state {
	case SelectProjectDropDown:
		return selectorProjects
	case SelectPhaseDropDown:
		return selectorPhases
	case SelectEpicDropDown:
		return selectorEpics
	case SelectTypesDropDown:
		return selectorTypes
	default:
		return ""
	}
}

func optionCatalogCommand(ctx context.Context, client appClient, id int, generation uint64) tea.Cmd {
	return func() tea.Msg {
		cfg, err := client.GetProjectConfig(ctx, id)
		phases, types := projects.CatalogOptions(cfg)
		return optionCatalogLoadedMsg{id: id, generation: generation, config: cfg, phases: phases, types: types, err: err}
	}
}

func (m Model) selectorCommand(source selectorSource) tea.Cmd {
	if source == selectorDocuments {
		options := []dto.Option{}
		for _, kind := range m.optionCatalog.config.ChangeDocs {
			options = append(options, dto.Option{ID: kind, Label: kind})
		}
		generation := m.selectorGeneration
		return func() tea.Msg { return selectorLoadedMsg{source: source, generation: generation, options: options} }
	}
	var cmd tea.Cmd
	switch source {
	case selectorPhases:
		cmd = cachedSelectorCommand(source, m.optionCatalog.phases, m.optionCatalog.loaded)
	case selectorTypes:
		cmd = cachedSelectorCommand(source, m.optionCatalog.types, m.optionCatalog.loaded)
	default:
		cmd = selectorCommand(m.ctx, m.client, source, m.currentProject.ID)
	}
	generation, projectID := m.selectorGeneration, m.currentProject.ID
	return func() tea.Msg {
		r := cmd().(selectorLoadedMsg)
		r.generation = generation
		r.projectID = projectID
		return r
	}
}

func cachedSelectorCommand(source selectorSource, options []dto.Option, loaded bool) tea.Cmd {
	return func() tea.Msg {
		if !loaded {
			return selectorLoadedMsg{source: source, err: fmt.Errorf("backend option catalog is not loaded")}
		}
		return selectorLoadedMsg{source: source, options: options}
	}
}

func selectorCommand(ctx context.Context, client appClient, source selectorSource, projectID string) tea.Cmd {
	return func() tea.Msg {
		var (
			options []dto.Option
			err     error
		)
		switch source {
		case selectorProjects:
			var rows []dto.Project
			rows, err = client.ListProjectRows(ctx)
			options = projects.Options(rows)
		case selectorEpics:
			id, parseErr := currentProjectNumericID(projectID)
			if parseErr != nil {
				err = parseErr
				break
			}
			var rows []dto.Epic
			rows, err = client.ListEpics(ctx, id)
			active := make([]dto.Epic, 0, len(rows))
			for _, row := range rows {
				if row.Active {
					active = append(active, row)
				}
			}
			options = epics.Options(active)
		}
		return selectorLoadedMsg{source: source, options: options, err: err}
	}
}

type configSavedMsg struct {
	projectID int
	err       error
}

func (m Model) persistCurrentProject() (Model, tea.Cmd) {
	if m.configSaveInFlight {
		m.configSavePending = true
		return m, nil
	}
	m.configSaveInFlight = true
	cfg, path := m.appConfig, m.configPath
	return m, func() tea.Msg {
		if path == "" {
			return configSavedMsg{projectID: cfg.ProjectID}
		}
		return configSavedMsg{projectID: cfg.ProjectID, err: saveAppConfig(path, cfg)}
	}
}
