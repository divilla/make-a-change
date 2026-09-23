package projects

import "cli/internal/dto"

// MoveSelection returns the model with bounded row selection moved by delta.
func (m Model) MoveSelection(delta int) Model {
    if len(m.Rows) == 0 {
        m.Selected = 0
        return m
    }
    m.Selected += delta
    if m.Selected < 0 {
        m.Selected = 0
    }
    if m.Selected >= len(m.Rows) {
        m.Selected = len(m.Rows) - 1
    }
    return m
}

// SelectDetail stores and returns the currently selected project detail.
func (m Model) SelectDetail() (Model, dto.Project, bool) {
    if len(m.Rows) == 0 {
        return m, dto.Project{}, false
    }
    m.Detail = m.Rows[m.Selected]
    return m, m.Detail, true
}
