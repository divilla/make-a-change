package app

import (
	"cli/internal/dto"
	"cli/internal/projects"
	"context"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) saveProjectCreate() (tea.Model, tea.Cmd) {
	return m.saveProjectCreateValue(m.input.Value())
}

func (m Model) saveProjectCreateValue(name string) (tea.Model, tea.Cmd) {
	return m.beginProject(projects.Create, 0, name)
}

func (m Model) saveProjectUpdate() (tea.Model, tea.Cmd) {
	return m.saveProjectUpdateValue(m.input.Value())
}

func (m Model) saveProjectUpdateValue(name string) (tea.Model, tea.Cmd) {
	return m.beginProject(projects.Edit, m.projectList.Detail.ID, name)
}

func (m Model) beginProject(op projects.Operation, id int, name string) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.projectList, cmd = m.projectList.Begin(m.ctx, m.client, op, id, name)
	m.status = m.projectList.Status
	m.err = ""
	if m.projectList.Err != nil {
		m.err = m.projectList.Err.Error()
	}
	return m, cmd
}

func currentProjectCommand(ctx context.Context, client appClient, id int, generation uint64) tea.Cmd {
	return func() tea.Msg {
		project, err := client.GetProject(ctx, id)
		return currentProjectLoadedMsg{id: id, generation: generation, project: project, err: err}
	}
}

func (m Model) applyProjectResult(r projects.Result) (tea.Model, tea.Cmd) {
	next, accepted := m.projectList.Apply(r)
	if !accepted {
		return m, nil
	}
	m.projectList = next
	m.status = next.Status
	m.err = ""
	if next.Err != nil {
		m.err = next.Err.Error()
	}
	if r.Err == nil {
		switch r.Operation {
		case projects.Config:
			if r.ID == m.appConfig.ProjectID {
				// Keep the shared load valid until its replacement succeeds; screen
				// navigation can cancel or invalidate the manual read at any time.
				m.catalogGeneration++
				phases, types := projects.CatalogOptions(r.Config)
				m.optionCatalog = optionCatalog{phases: phases, types: types, loaded: true}
			}
		case projects.Create, projects.Edit:
			m.state = ProjectDetailsState
			if m.currentProject.ID == strconv.Itoa(r.Project.ID) {
				m.currentProject.Label = r.Project.Name
			}
			m = m.setPromptValue("")
		case projects.Delete:
			m.state = ProjectsListState
			if m.appConfig.ProjectID == r.ID {
				m.currentProject = dto.Option{}
				m.appConfig.ProjectID = 0
				m.selectionGeneration++
				m.catalogGeneration++
				m.optionCatalog = optionCatalog{}
				return m.persistCurrentProject()
			}
		}
	}
	return m, nil
}
