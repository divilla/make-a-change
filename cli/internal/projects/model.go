package projects

import "cli/internal/dto"

// NoSelectableError is shown when enter is pressed without a selectable project.
const NoSelectableError = "no projects selectable"

// Model stores projects list and detail state.
type Model struct {
    Rows     []dto.Project
    Selected int
    Detail   dto.Project
    Loading  bool
}

// StartLoading returns a projects model in loading state.
func StartLoading() Model {
    return Model{Loading: true}
}

// WithRows returns a projects model populated with loaded rows.
func (m Model) WithRows(rows []dto.Project) Model {
    m.Rows = rows
    m.Selected = 0
    m.Loading = false
    return m
}

// WithError returns a projects model reset after load failure.
func (m Model) WithError() Model {
    m.Rows = nil
    m.Selected = 0
    m.Loading = false
    return m
}
