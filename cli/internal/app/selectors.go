package app

import (
    "fmt"
    "strconv"
    "strings"

    "cli/internal/changes"
    "cli/internal/dto"

    tea "github.com/charmbracelet/bubbletea"
)

func filterOptions(options []dto.Option) []dto.Option {
    filtered := make([]dto.Option, 0, len(options)+1)
    filtered = append(filtered, options...)
    filtered = append(filtered, dto.Option{ID: "/clear", Label: "/clear"})
    return filtered
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
            if m.changeList.Detail.EpicID == "" && m.changeList.Detail.EpicName == "" {
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
        case filterType:
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
    case filterType:
        m.changesFilters.typ = option
    }
    m.clampChangeListSelection()
}

func (m *Model) clearChangesFilter(field filterField) {
    switch field {
    case filterPhase:
        m.changesFilters.phase = dto.Option{}
    case filterEpic:
        m.changesFilters.epic = dto.Option{}
    case filterType:
        m.changesFilters.typ = dto.Option{}
    }
    m.clampChangeListSelection()
}

func (m *Model) clampChangeListSelection() {
    m.changeList = m.changeList.ClampSelection(m.changeFilters(), m.changeTableRows())
}

func (m Model) changeFilters() changes.Filters {
    return changes.Filters{
        Phase: m.changesFilters.phase,
        Epic:  m.changesFilters.epic,
        Type:  m.changesFilters.typ,
        Find:  m.changesFilters.find,
    }
}

func (m *Model) saveCurrentProject(project dto.Option) error {
    if m.configPath == "" {
        return nil
    }
    projectID, err := strconv.Atoi(project.ID)
    if err != nil {
        return fmt.Errorf("failed to save project_id: current project ID must be numeric")
    }
    m.appConfig.ProjectID = projectID
    if m.appConfig.BackendURL == "" {
        m.appConfig.BackendURL = defaultBackendURL
    }
    if err := saveAppConfig(m.configPath, m.appConfig); err != nil {
        return fmt.Errorf("failed to save project_id: %w", err)
    }
    return nil
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

func optionCatalogCommand(client appClient, flowDir string) tea.Cmd {
    return func() tea.Msg {
        phases, err := client.ListPhases()
        if err != nil {
            return optionCatalogLoadedMsg{err: err}
        }
        types, err := client.ListTypes()
        if err != nil {
            return optionCatalogLoadedMsg{err: err}
        }
        if flowDir != "" {
            if err := rebuildChangeTypeSlugsFile(flowDir, types); err != nil {
                return optionCatalogLoadedMsg{err: err}
            }
        }
        cacheChangeTypes(types)
        return optionCatalogLoadedMsg{phases: phases, types: types}
    }
}

func (m Model) selectorCommand(source selectorSource) tea.Cmd {
    switch source {
    case selectorPhases:
        return cachedSelectorCommand(source, m.optionCatalog.phases, m.optionCatalog.loaded)
    case selectorTypes:
        return cachedSelectorCommand(source, m.optionCatalog.types, m.optionCatalog.loaded)
    default:
        return selectorCommand(m.client, source, m.currentProject.ID)
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

func selectorCommand(client appClient, source selectorSource, projectID string) tea.Cmd {
    return func() tea.Msg {
        var (
            options []dto.Option
            err     error
        )
        switch source {
        case selectorProjects:
            options, err = client.ListProjects()
        case selectorEpics:
            options, err = client.ListEpics(projectID)
        }
        return selectorLoadedMsg{source: source, options: options, err: err}
    }
}

func projectListCommand(client appClient) tea.Cmd {
    return func() tea.Msg {
        projects, err := client.ListProjectRows()
        return projectListLoadedMsg{projects: projects, err: err}
    }
}

func changeListCommand(client appClient, projectID string) tea.Cmd {
    return func() tea.Msg {
        changes, err := client.ListChangeRows(projectID)
        return changeListLoadedMsg{changes: changes, err: err}
    }
}
