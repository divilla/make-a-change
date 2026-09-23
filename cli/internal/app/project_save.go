package app

import (
    "fmt"
    "strconv"
    "strings"

    "cli/internal/dto"

    tea "github.com/charmbracelet/bubbletea"
)

func (m Model) saveProjectCreate() (tea.Model, tea.Cmd) {
    return m.saveProjectCreateValue(m.input.Value())
}

func (m Model) saveProjectCreateValue(name string) (tea.Model, tea.Cmd) {
    if strings.TrimSpace(name) == "" {
        m.err = "project name is required"
        m.status = "validation failed"
        return m, nil
    }
    m.status = "saving"
    return m, projectCreateCommand(m.client, name)
}

func (m Model) saveProjectUpdate() (tea.Model, tea.Cmd) {
    return m.saveProjectUpdateValue(m.input.Value())
}

func (m Model) saveProjectUpdateValue(name string) (tea.Model, tea.Cmd) {
    if strings.TrimSpace(name) == "" {
        m.err = "project name is required"
        m.status = "validation failed"
        return m, nil
    }
    id, err := projectNumericID(m.projectList.Detail)
    if err != nil {
        m.err = err.Error()
        m.status = "validation failed"
        return m, nil
    }
    m.status = "saving"
    return m, projectUpdateCommand(m.client, id, name)
}

func projectCreateCommand(client appClient, name string) tea.Cmd {
    return func() tea.Msg {
        created, err := client.CreateProject(name)
        if err != nil {
            return projectSavedMsg{source: ProjectCreateState, err: err}
        }
        id, err := projectNumericID(created)
        if err != nil {
            return projectSavedMsg{source: ProjectCreateState, err: err}
        }
        project, err := client.GetProject(id)
        return projectSavedMsg{source: ProjectCreateState, project: project, err: err}
    }
}

func projectUpdateCommand(client appClient, id int, name string) tea.Cmd {
    return func() tea.Msg {
        updated, err := client.UpdateProject(id, name)
        if err != nil {
            return projectSavedMsg{source: ProjectUpdateState, err: err}
        }
        updatedID, err := projectNumericIDWithFallback(updated, id)
        if err != nil {
            return projectSavedMsg{source: ProjectUpdateState, err: err}
        }
        project, err := client.GetProject(updatedID)
        return projectSavedMsg{source: ProjectUpdateState, project: project, err: err}
    }
}

func projectGetCommand(client appClient, id int) tea.Cmd {
    return func() tea.Msg {
        project, err := client.GetProject(id)
        return projectLoadedMsg{id: id, project: project, err: err}
    }
}

func currentProjectCommand(client appClient, id int) tea.Cmd {
    return func() tea.Msg {
        project, err := client.GetProject(id)
        return currentProjectLoadedMsg{id: id, project: project, err: err}
    }
}

func projectNumericID(project dto.Project) (int, error) {
    return projectNumericIDWithFallback(project, 0)
}

func projectNumericIDWithFallback(project dto.Project, fallback int) (int, error) {
    value := strings.TrimSpace(project.ID)
    if value == "" && fallback > 0 {
        return fallback, nil
    }
    id, err := strconv.Atoi(value)
    if err != nil || id <= 0 {
        return 0, fmt.Errorf("project ID must be a valid positive number")
    }
    return id, nil
}
