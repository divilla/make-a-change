package app

import (
	"cli/internal/configurations"
	"cli/internal/health"
	"context"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func isConfigurationState(state State) bool {
	return state == BackendConfigListState || state == BackendConfigDetailsState || state == BackendConfigFormState || state == BackendConfigDeleteState
}

func (m Model) selectedConfigurationMayBe(slug string) bool {
	return m.appConfig.ProjectID > 0 && (m.selectedConfigSlug == "" || m.selectedConfigSlug == slug)
}

func (m Model) cancelConfigurationCatalogRefresh() Model {
	if m.configCatalogCancel != nil {
		m.configCatalogCancel()
		m.configCatalogCancel = nil
		m.catalogGeneration++
	}
	return m
}

func (m Model) beginConfigurationCatalogRefresh() (Model, tea.Cmd) {
	m = m.cancelConfigurationCatalogRefresh()
	m.catalogGeneration++
	m.optionCatalog = optionCatalog{}
	ctx, cancel := context.WithCancel(m.ctx)
	m.configCatalogCancel = cancel
	cmd := optionCatalogCommand(ctx, m.client, m.appConfig.ProjectID, m.catalogGeneration)
	return m, func() tea.Msg {
		defer cancel()
		return cmd()
	}
}

func (m Model) beginConfiguration(op configurations.Operation, slug string) (tea.Model, tea.Cmd) {
	api, ok := m.client.(configurations.API)
	if !ok {
		m.err = "backend configuration client unavailable"
		return m, nil
	}
	row := m.configurations.Detail
	if op == configurations.Insert || op == configurations.Update {
		var err error
		row, err = m.configurations.ParseDraft()
		if err != nil {
			m.err = err.Error()
			m.status = "configuration draft retained"
			return m, nil
		}
		slug = row.Slug
	}
	var cmd tea.Cmd
	m.configurations, cmd = m.configurations.Begin(m.ctx, api, op, slug, row)
	m.status = m.configurations.Status
	return m, cmd
}

func (m Model) applyConfigurationResult(r configurations.Result) (tea.Model, tea.Cmd) {
	var accepted bool
	m.configurations, accepted = m.configurations.Apply(r)
	if !accepted {
		return m, nil
	}
	m.status = m.configurations.Status
	if r.Err != nil {
		m.err = r.Err.Error()
		return m, nil
	}
	if !strings.Contains(m.err, "project catalog refresh failed") {
		m.err = ""
	}
	if r.Operation == configurations.Insert || r.Operation == configurations.Update || r.Operation == configurations.Delete {
		m = m.setPromptValue("")
		if r.Operation == configurations.Delete {
			m.state = BackendConfigListState
		} else {
			m.state = BackendConfigDetailsState
		}
		var catalog tea.Cmd
		if r.Operation == configurations.Update && m.selectedConfigurationMayBe(r.Slug) {
			m, catalog = m.beginConfigurationCatalogRefresh()
		}
		var list tea.Cmd
		var next tea.Model
		next, list = m.beginConfiguration(configurations.List, "")
		m = next.(Model)
		return m, tea.Batch(list, catalog)
	}
	if r.Operation == configurations.List && m.configurations.Refresh == configurations.List {
		if m.configurations.CommittedOperation == configurations.Delete {
			m.configurations.Refresh = ""
			m = m.keepConfigurationSelectionVisible()
			return m, nil
		}
		m.configurations.Refresh = configurations.Details
		return m.beginConfiguration(configurations.Details, m.configurations.CommittedSlug)
	}
	if r.Operation == configurations.List && m.state == BackendConfigListState {
		m = m.keepConfigurationSelectionVisible()
	}
	if r.Operation == configurations.Details {
		m.configurations.Refresh = ""
	}
	return m, nil
}

func (m Model) configurationCommand(source State, command string) (tea.Model, tea.Cmd) {
	switch command {
	case "/return":
		m.configurations = m.configurations.Invalidate()
		if source == BackendConfigDetailsState {
			m.state = BackendConfigListState
			m.configurations.DetailLoaded = false
			m.configurations.Refresh = ""
			m = m.keepConfigurationSelectionVisible()
			return m, nil
		}
		return m.arrive(MainState, "return")
	case "/new-config":
		m.configurations = m.configurations.OpenForm(false)
		m.state = BackendConfigFormState
		m = m.setConfigurationPrompt(m.configurations.Raw[0])
		m = m.keepConfigurationFieldVisible()
		m.status = "new backend configuration"
	case "/edit":
		if !m.configurations.DetailLoaded {
			m.err = "load configuration details before editing"
			return m, nil
		}
		m.configurations = m.configurations.OpenForm(true)
		m.state = BackendConfigFormState
		m = m.setConfigurationPrompt(m.configurations.Raw[m.configurations.Field])
		m = m.keepConfigurationFieldVisible()
		m.status = "edit backend configuration"
	case "/delete":
		if !m.configurations.DetailLoaded {
			m.err = "load configuration details before deleting"
			return m, nil
		}
		m.configurations.Confirm = true
		m.state = BackendConfigDeleteState
		m.status = "confirm delete " + m.configurations.Detail.Slug
	case "/confirm":
		if !m.configurations.Confirm {
			return m, nil
		}
		return m.beginConfiguration(configurations.Delete, m.configurations.Detail.Slug)
	case "/cancel":
		switch source {
		case BackendConfigFormState:
			m.configurations.Form = false
			if m.configurations.Editing {
				m.state = BackendConfigDetailsState
			} else {
				m.state = BackendConfigListState
			}
		case BackendConfigDeleteState:
			m.configurations.Confirm = false
			m.state = BackendConfigDetailsState
		}
		m = m.setPromptValue("")
		m.status = "cancel"
	case "/save":
		if source != BackendConfigFormState {
			return m, nil
		}
		m.configurations.Raw[m.configurations.Field] = m.promptValue()
		if m.configurations.Editing {
			return m.beginConfiguration(configurations.Update, m.configurations.Detail.Slug)
		}
		return m.beginConfiguration(configurations.Insert, "")
	case "/retry":
		var catalog tea.Cmd
		if m.optionCatalog.err != nil && m.configurations.Committed != "" && m.configurations.CommittedOperation == configurations.Update && m.selectedConfigurationMayBe(m.configurations.CommittedSlug) {
			m, catalog = m.beginConfigurationCatalogRefresh()
		}
		var read tea.Cmd
		var next tea.Model
		if m.configurations.Refresh != "" {
			next, read = m.beginConfiguration(m.configurations.Refresh, m.configurations.CommittedSlug)
		} else if source == BackendConfigDetailsState {
			next, read = m.beginConfiguration(configurations.Details, m.configurations.RequestedSlug)
		} else {
			next, read = m.beginConfiguration(configurations.List, "")
		}
		return next, tea.Batch(read, catalog)
	}
	return m, nil
}

func (m Model) configurationKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if m.configurations.Busy {
		if key == "esc" || key == "ctrl+c" {
			m.configurations = m.configurations.Invalidate()
			return m.arrive(MainState, "return")
		}
		return m, nil
	}
	if m.state == BackendConfigFormState {
		if key == "ctrl+g" {
			m.openCommandDropdown()
			return m, nil
		}
		switch key {
		case "esc":
			return m.configurationCommand(m.state, "/cancel")
		case "pgup", "pgdown":
			step := max(1, m.height/2)
			if key == "pgup" {
				step = -step
			}
			return m.scrollConfiguration(step), nil
		case "ctrl+e":
			return m.openTextEditor(BackendConfigFormState, m.promptValue())
		case "ctrl+s":
			return m.configurationCommand(m.state, "/save")
		case "tab", "enter", "shift+tab":
			m.configurations.Raw[m.configurations.Field] = m.promptValue()
			first := 0
			if m.configurations.Editing {
				first = 1
			}
			switch {
			case key == "shift+tab" && m.configurations.Field == first:
				m.configurations.Field = len(configurations.FieldNames) - 1
			case key == "shift+tab":
				m.configurations.Field--
			case m.configurations.Field == len(configurations.FieldNames)-1:
				m.configurations.Field = first
			default:
				m.configurations.Field++
			}
			m = m.setConfigurationPrompt(m.configurations.Raw[m.configurations.Field])
			m = m.keepConfigurationFieldVisible()
			m.status = "editing " + configurations.FieldNames[m.configurations.Field]
			return m, nil
		}
		return m.updatePromptInput(msg)
	}
	if m.state == BackendConfigDeleteState {
		switch key {
		case "/":
			m.openCommandDropdown()
			return m, nil
		case "esc", "ctrl+c":
			return m.configurationCommand(m.state, "/cancel")
		case "enter":
			return m.configurationCommand(m.state, "/confirm")
		}
		return m, nil
	}
	if m.input.Value() == "" {
		switch key {
		case "esc", "ctrl+c":
			return m.configurationCommand(m.state, "/return")
		case "up", "down":
			if m.state == BackendConfigListState && len(m.configurations.Rows) > 0 {
				if key == "up" {
					m.configurations.Selected = max(0, m.configurations.Selected-1)
				} else {
					m.configurations.Selected = min(len(m.configurations.Rows)-1, m.configurations.Selected+1)
				}
				m = m.keepConfigurationSelectionVisible()
			}
			return m, nil
		case "pgup", "pgdown":
			if m.state == BackendConfigListState && len(m.configurations.Rows) > 0 {
				step := max(1, m.height/2)
				if key == "pgup" {
					m.configurations.Selected = max(0, m.configurations.Selected-step)
				} else {
					m.configurations.Selected = min(len(m.configurations.Rows)-1, m.configurations.Selected+step)
				}
				return m.keepConfigurationSelectionVisible(), nil
			}
			step := max(1, m.height/2)
			if key == "pgup" {
				step = -step
			}
			return m.scrollConfiguration(step), nil
		case "ctrl+n":
			if m.state == BackendConfigListState {
				return m.configurationCommand(m.state, "/new-config")
			}
		case "enter":
			if m.state == BackendConfigListState && len(m.configurations.Rows) > 0 {
				slug := m.configurations.Rows[m.configurations.Selected].Slug
				m.configurations.Committed = ""
				m.configurations.CommittedSlug = ""
				m.state = BackendConfigDetailsState
				m.configurations.Offset = 0
				return m.beginConfiguration(configurations.Details, slug)
			}
			return m, nil
		case "/":
			m.openCommandDropdown()
			return m, nil
		}
	}
	if key == "enter" {
		return m.submitPrompt()
	}
	return m.updatePromptInput(msg)
}

func (m Model) beginHealth() (tea.Model, tea.Cmd) {
	api, ok := m.client.(health.API)
	if !ok {
		m.err = "backend health client unavailable"
		return m, nil
	}
	var cmd tea.Cmd
	m.health, cmd = m.health.Begin(m.ctx, api)
	m.status = "checking backend health"
	return m, cmd
}

func (m Model) healthCommand(command string) (tea.Model, tea.Cmd) {
	switch command {
	case "/return":
		m.health = m.health.Invalidate()
		m.err = ""
		return m.arrive(MainState, "return")
	case "/health-v1":
		m.health = m.health.SelectRoute("/api/v1/health")
		m.err = ""
		return m.beginHealth()
	case "/health-legacy":
		m.health = m.health.SelectRoute("/api/health")
		m.err = ""
		return m.beginHealth()
	case "/retry":
		return m.beginHealth()
	}
	return m, nil
}

func (m Model) healthKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if key == "esc" || key == "ctrl+c" {
		return m.healthCommand("/return")
	}
	if m.input.Value() == "" {
		if key == "/" {
			m.openCommandDropdown()
			return m, nil
		}
		if key == "r" && !m.health.Busy {
			return m.healthCommand("/retry")
		}
	}
	if key == "enter" {
		return m.submitPrompt()
	}
	if m.health.Busy {
		return m, nil
	}
	return m.updatePromptInput(msg)
}

func (m Model) applyHealthResult(r health.Result) (tea.Model, tea.Cmd) {
	var accepted bool
	m.health, accepted = m.health.Apply(r)
	if !accepted {
		return m, nil
	}
	if r.Err != nil {
		m.err = r.Err.Error()
		m.status = "health refresh failed"
	} else {
		m.err = ""
		m.status = "backend health " + r.Value.Status
	}
	return m, nil
}

func (m Model) applyConfigurationEditor(msg editorFinishedMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.err = msg.err.Error()
		m.status = "editor failed; configuration draft retained"
		return m, tea.ClearScreen
	}
	m.configurations.Raw[m.configurations.Field] = msg.content
	m = m.setConfigurationPrompt(msg.content)
	m.status = "configuration draft ready"
	return m, tea.ClearScreen
}

func (m Model) setConfigurationPrompt(value string) Model {
	m = m.setPromptValue(value)
	m.editorDraft = &value
	return m
}

func (m Model) keepConfigurationSelectionVisible() Model {
	lines, _ := m.viewLines()
	m.configurations = m.configurations.KeepSelectionVisible(terminalWidth(m.width), m.epicViewportHeight(lines))
	return m
}

func (m Model) keepConfigurationFieldVisible() Model {
	lines, _ := m.viewLines()
	m.configurations = m.configurations.KeepFieldVisible(terminalWidth(m.width), m.epicViewportHeight(lines))
	return m
}

func (m Model) scrollConfiguration(delta int) Model {
	lines, _ := m.viewLines()
	m.configurations = m.configurations.Scroll(delta, terminalWidth(m.width), m.epicViewportHeight(lines))
	return m
}
