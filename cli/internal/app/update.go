package app

import (
	"cli/internal/agent"
	"cli/internal/changes"
	"cli/internal/configurations"
	"cli/internal/documents"
	"cli/internal/dto"
	"cli/internal/epics"
	"cli/internal/health"
	"cli/internal/navigation"
	"cli/internal/projects"
	"cli/internal/testcases"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// Init starts any initial asynchronous command required by the model.
func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{tea.ClearScreen}
	if m.needsProjectSelection() {
		selectProject := func() tea.Msg {
			return startupProjectSelectionMsg{}
		}
		cmds = append(cmds, selectProject)
		return tea.Batch(cmds...)
	}
	if m.appConfig.ProjectID > 0 {
		cmds = append(cmds, currentProjectCommand(m.ctx, m.client, m.appConfig.ProjectID, m.selectionGeneration), optionCatalogCommand(m.ctx, m.client, m.appConfig.ProjectID, m.catalogGeneration))
		return tea.Batch(cmds...)
	}
	return tea.Batch(cmds...)
}

// Update applies Bubble Tea messages to the root model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case briefCleanupMsg:
		if msg.err != nil {
			m.err = "brief scratch cleanup failed: " + msg.err.Error()
		}
		return m, nil
	case agent.Result:
		return m.applyBriefResult(msg)
	case briefProgressMsg:
		return m.applyBriefProgress(msg)
	case configSavedMsg:
		m.configSaveInFlight = false
		if msg.err != nil {
			// Keep the UI open so a failed save is visible before the user exits.
			m.quitRequested = false
			if msg.projectID == 0 {
				// Clearing a deleted selection is a separate local write. Keep the
				// committed deletion and any refresh error/read-only retry guidance.
				m.err = strings.TrimPrefix(m.err+"; project selection cleared in memory; failed to save project_id: "+msg.err.Error(), "; ")
				m.status += "; config save failed"
			} else {
				m.err = "project selected in memory; failed to save project_id: " + msg.err.Error()
				m.status = "config save failed"
			}
		}
		if m.configSavePending {
			m.configSavePending = false
			return m.persistCurrentProject()
		}
		if m.quitRequested {
			return m.requestQuit()
		}
		if msg.err == nil && !strings.HasPrefix(m.status, "deleted project") && !strings.HasPrefix(m.status, "project delete committed") && !strings.HasPrefix(m.status, "project deactivated") {
			m.status = "project selection saved"
		}
		return m, nil
	case startupProjectSelectionMsg:
		if !m.needsProjectSelection() {
			return m, nil
		}
		return m.beginSelector(SelectProjectDropDown)
	case selectorLoadedMsg:
		if m.dropdown.source != msg.source || msg.generation != m.selectorGeneration || (msg.projectID != "" && msg.projectID != m.currentProject.ID) {
			return m, nil
		}
		m.dropdown.loading = false
		if msg.err != nil {
			m.err = msg.err.Error()
			m.dropdown.options = nil
			return m, nil
		}
		options := msg.options
		if m.dropdown.filterField != "" {
			options = filterOptions(options)
		}
		if m.dropdown.editField == detailEditEpic {
			options = append(options, dto.Option{ID: "@none", Label: "@none"})
		}
		m.dropdown.options = options
		m.dropdown.highlighted = m.dropdownCurrentValueIndex(options)
		if len(options) == 0 {
			if m.dropdown.source == selectorProjects && m.needsProjectSelection() {
				m.state = MainState
				m.dropdown = dropdownModel{}
				m.err = noProjectsToSelectError
				return m, nil
			}
			m.err = "no options available"
		}
		return m, nil
	case changes.Result:
		return m.applyChangeResult(msg)
	case configurations.Result:
		return m.applyConfigurationResult(msg)
	case health.Result:
		return m.applyHealthResult(msg)
	case testcases.Result:
		return m.applyTestCaseResult(msg)
	case documents.ChangeResult:
		return m.applyChangeDocumentResult(msg)
	case documents.HistoryResult:
		return m.applyHistoryResult(msg)
	case documents.Result:
		return m.applyDocumentResult(msg)
	case epics.Result:
		return m.applyEpicResult(msg)
	case projects.Result:
		return m.applyProjectResult(msg)
	case detailCopiedMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
			m.status = "copy failed"
			return m, nil
		}
		m.status = detailCopyStatus(msg.label)
		return m, nil
	case changeSavedMsg:
		if m.state != msg.source {
			return m, nil
		}
		if msg.err != nil {
			if msg.source == ChangeDetailsState && msg.change.ID != "" {
				detailSelected := m.changeList.DetailSelected
				detailOffset := m.changeList.DetailOffset
				m.changeList = m.changeList.WithDetail(msg.change)
				m.changeList.DetailSelected = detailSelected
				m.changeList.DetailOffset = detailOffset
				m.changeList = m.changeList.ClampDetailSelection(m.changeTableRows(), terminalWidth(m.width))
			}
			m.err = msg.err.Error()
			m.status = "save failed"
			return m, nil
		}
		detailSelected := m.changeList.DetailSelected
		detailOffset := m.changeList.DetailOffset
		preserveDetailSelection := msg.source == ChangeDetailsState || m.detailEditField != ""
		m.changeList = m.changeList.WithDetail(msg.change)
		if preserveDetailSelection {
			m.changeList.DetailSelected = detailSelected
			m.changeList.DetailOffset = detailOffset
			m.changeList = m.changeList.ClampDetailSelection(m.changeTableRows(), terminalWidth(m.width))
		}
		m.state = ChangeDetailsState
		m.changeDetailLoaded = msg.reloadErr == nil
		m.status = "save"
		if msg.reloadErr != nil {
			m.err = msg.reloadErr.Error()
			m.status = "load failed"
		}
		m.detailEditField = ""
		m.testCase = m.testCase.ClearForm()
		m = m.setPromptValue("")
		return m, nil
	case optionCatalogLoadedMsg:
		if msg.id != m.appConfig.ProjectID || msg.generation != m.catalogGeneration {
			return m, nil
		}
		m.configCatalogCancel = nil
		if msg.err != nil {
			m.changeList.Detail.DocumentTypes = nil
			m.optionCatalog = optionCatalog{err: msg.err}
			if m.status == "config save failed" {
				m.err += "; project configuration unavailable: " + msg.err.Error()
			} else if m.configurations.Committed != "" && m.configurations.CommittedOperation == configurations.Update && m.selectedConfigurationMayBe(m.configurations.CommittedSlug) {
				m.err = "committed configuration; project catalog refresh failed: " + msg.err.Error()
				m.status = m.configurations.Committed + "; project catalog refresh failed — /retry reads only"
			} else {
				m.err = msg.err.Error()
				m.status = "option catalog failed"
			}
			return m, nil
		}
		m.optionCatalog = optionCatalog{config: msg.config, phases: msg.phases, types: msg.types, loaded: true}
		m.selectedConfigSlug = msg.config.Slug
		m.changeList.Detail.DocumentTypes = append([]string(nil), msg.config.ChangeDocs...)
		if strings.Contains(m.err, "project catalog refresh failed") {
			m.err = ""
			m.status = m.configurations.Committed + "; project catalog refreshed"
		}
		return m, nil
	case currentProjectLoadedMsg:
		currentID, err := strconv.Atoi(m.currentProject.ID)
		if err != nil || currentID != msg.id || msg.generation != m.selectionGeneration {
			return m, nil
		}
		if msg.err != nil {
			if m.status == "config save failed" {
				m.err += "; project details unavailable: " + msg.err.Error()
			} else {
				m.err = msg.err.Error()
			}
			return m, nil
		}
		m.currentProject = dto.Option{ID: m.currentProject.ID, Label: strings.TrimSpace(msg.project.Name)}
		m.selectedConfigSlug = msg.project.ConfigSlug
		return m, nil
	case editorFinishedMsg:
		if msg.generation > 0 && (msg.generation != m.editorGeneration || msg.source != m.state || msg.projectID != m.currentProject.ID || msg.ownerID != m.changeList.Detail.ID || msg.field != m.detailEditField) {
			return m, nil
		}
		if msg.generation > 0 {
			if msg.source == DocumentState && (msg.documentRevision != m.document.Revision || msg.documentOwner != m.document.OwnerID || msg.documentTable != m.document.OwnerTable || msg.documentType != m.document.DraftType) {
				return m, nil
			}
			if msg.source == ChangeDetailsState && ((msg.field == detailEditDocument && msg.documentType != m.changeList.Draft.DocumentType) || (msg.field == detailEditComment && msg.commentID != m.commentID)) {
				return m, nil
			}
		}
		if m.state == BriefState && msg.source == BriefState {
			if msg.err != nil {
				m.err = msg.err.Error()
				return m, tea.ClearScreen
			}
			m.brief = m.brief.EditBrief(msg.content)
			m.briefField = "brief"
			m = m.setPromptValue("")
			m.status = "brief edited; /confirm saves"
			return m, tea.ClearScreen
		}
		if m.state == BackendConfigFormState && msg.source == BackendConfigFormState {
			return m.applyConfigurationEditor(msg)
		}
		if m.state == DocumentState && m.documentForm && msg.source == DocumentState {
			if msg.err != nil {
				m.err = msg.err.Error()
				m.status = "editor failed; draft retained"
				return m, tea.ClearScreen
			}
			m = m.setPromptValue(documents.SafeText(msg.content))
			m.editorDraft = &msg.content
			m.document.DraftBody = msg.content
			m.status = "document draft ready; Enter saves"
			return m, tea.ClearScreen
		}
		if m.state != msg.source {
			return m, nil
		}
		if msg.err != nil {
			m.err = msg.err.Error()
			m.status = "editor failed"
			return m, tea.ClearScreen
		}
		if msg.source == ChangeDetailsState && m.editorDraft == nil && msg.content == msg.original {
			switch m.detailEditField {
			case detailEditBrief, detailEditSpec, detailEditPullRequest, detailEditDocument:
				m.detailEditField = ""
				m = m.setPromptValue("")
				m.status = "unchanged"
				return m, tea.ClearScreen
			}
		}
		if msg.source == EpicCreateState || msg.source == EpicUpdateState || msg.source == ProjectCreateState || msg.source == ProjectUpdateState || msg.source == ChangeCreateState || msg.source == ChangeUpdateState ||
			msg.source == TestCaseCreateState || msg.source == TestCaseUpdateState ||
			(msg.source == ChangeDetailsState && m.detailEditField != "") {
			m = m.setPromptValue(msg.content)
			m.editorDraft = &msg.content
		}
		if msg.source == ChangeCreateState && m.detailEditField == "" {
			m.changeList = m.changeList.PrepareCreate(msg.content)
			m.err = ""
			m.status = "review creation fields; Enter saves, Ctrl+T title, Ctrl+U UUID"
			return m, tea.ClearScreen
		}
		next, cmd := m.submitPromptValue(msg.content)
		m = next.(Model)
		if cmd == nil {
			return m, tea.ClearScreen
		}
		if msg.source == ChangeDetailsState && m.detailEditField != "" {
			return m, tea.Batch(tea.ClearScreen, cmd)
		}
		return m, tea.Sequence(tea.ClearScreen, cmd)
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.state == BackendConfigFormState {
			m = m.keepConfigurationFieldVisible()
		}
		return m, nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	// Clipboard results arrive asynchronously, outside key handling. Only a
	// lossless preview may replace the raw editor draft.
	canSyncDraft := m.editorDraft != nil && *m.editorDraft == m.input.Value()
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	if canSyncDraft {
		m.syncEditorDraft()
	}
	return m, cmd
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.quitRequested {
		return m, nil
	}
	if m.hasDropdown() {
		m.err = ""
		return m.handleDropdownKey(msg.String(), msg)
	}
	if m.historyOpen {
		return m.historyKey(msg)
	}
	if m.changeDocuments.Busy {
		return m, nil
	}
	if m.state == BriefState {
		return m.briefKey(msg)
	}
	if m.document.Busy && m.state == DocumentState {
		if msg.Type == tea.KeyEsc || msg.Type == tea.KeyCtrlC {
			return m.leaveDocuments()
		}
		return m, nil
	}
	if m.testCase.Busy {
		if msg.Type == tea.KeyEsc || msg.Type == tea.KeyCtrlC {
			m.testCase = m.testCase.Invalidate()
			return m.handleEsc()
		}
		return m, nil
	}
	if m.changeList.Busy || m.projectList.Busy || m.epicList.Busy {
		return m, nil
	}
	key := msg.String()
	if m.state == ChangesListState && m.changeList.Inactive && (key == "esc" || key == "ctrl+c") && m.input.Value() == "" {
		return m.leaveInactiveChanges()
	}
	if m.isDropdownState() {
		m.err = ""
		return m.handleDropdownKey(key, msg)
	}
	if isConfigurationState(m.state) {
		return m.configurationKey(msg)
	}
	if m.state == HealthState {
		return m.healthKey(msg)
	}
	m.err = ""
	if m.state == FindInputState {
		return m.handleFindKey(key, msg)
	}

	if updated, cmd, ok := m.handleListNavigationKey(key, msg); ok {
		return updated, cmd
	}
	if m.detailEditField == detailEditSlug && isPromptNewlineKey(msg) {
		return m, nil
	}

	if isPromptNewlineKey(msg) {
		return m.insertPromptNewline(), nil
	}
	var handled bool
	m, handled = m.handlePendingShiftEnter(msg)
	if handled {
		return m, nil
	}
	if isShiftEnterPrefix(msg) {
		m.pendingAltO = true
		return m, nil
	}

	switch key {
	case "ctrl+t", "ctrl+u":
		if m.state == ChangeCreateState {
			if m.detailEditField == "" {
				m.changeList.Draft.Value = m.promptValue()
			}
			field, value := detailCreateTitle, m.changeList.Draft.Title
			if key == "ctrl+u" {
				field, value = detailCreateUUID, m.changeList.Draft.UUID
			}
			m.detailEditField = field
			m = m.setPromptValue(value)
			m.editorDraft = &value
			m.status = "editing " + string(field)
			return m, nil
		}
	case "ctrl+c":
		return m.handlePromptCancel()
	case "ctrl+e":
		if m.detailEditField == detailEditSlug {
			return m, nil
		}
		return m.openPromptEditor(m.state)
	case "esc":
		if m.detailEditField != "" || m.editorDraft != nil || m.input.Value() != "" {
			return m.handlePromptCancel()
		}
		return m.handleEsc()
	case "up":
	case "down":
	case "/":
		if m.editorDraft == nil && m.input.Value() == "" {
			m.openCommandDropdown()
			return m, nil
		}
	case "enter":
		return m.submitPrompt()
	}

	return m.updatePromptInput(msg)
}

func (m Model) handleListNavigationKey(key string, msg tea.KeyMsg) (Model, tea.Cmd, bool) {
	if next, cmd, ok := m.documentKey(key, msg); ok {
		return next, cmd, true
	}
	if m.editorDraft != nil || m.detailEditField != "" || (m.input.Value() != "" && m.state != ChangesListState) {
		return m, nil, false
	}
	switch m.state {
	case ChangesListState:
		switch {
		case key == "ctrl+h" && m.input.Value() == "":
			next, cmd := m.openInactiveChanges()
			return next.(Model), cmd, true
		case (key == " " || key == "space") && m.changeList.Inactive && m.input.Value() == "":
			rows := changes.FilteredRows(m.changeList.Rows, m.changeFilters())
			if m.changeList.Selected >= 0 && m.changeList.Selected < len(rows) {
				id, _ := strconv.Atoi(rows[m.changeList.Selected].ID)
				next, cmd := m.beginChange(changes.Reactivate, id, changes.Input{})
				return next.(Model), cmd, true
			}
			return m, nil, true
		case (key == "enter" || msg.Type == tea.KeyCtrlJ) && strings.HasPrefix(strings.TrimSpace(m.input.Value()), "/"):
			return m, nil, false
		case key == "up":
			m.changeList = m.changeList.MoveSelection(-1, m.changeFilters(), m.changeTableRows())
			return m, nil, true
		case key == "down":
			m.changeList = m.changeList.MoveSelection(1, m.changeFilters(), m.changeTableRows())
			return m, nil, true
		case key == "pgup":
			m.changeList = m.changeList.MoveSelection(-m.changeTableRows(), m.changeFilters(), m.changeTableRows())
			return m, nil, true
		case key == "pgdown":
			m.changeList = m.changeList.MoveSelection(m.changeTableRows(), m.changeFilters(), m.changeTableRows())
			return m, nil, true
		case key == "enter" || msg.Type == tea.KeyCtrlJ:
			updated, cmd := m.handleListSelection()
			return updated.(Model), cmd, true
		case key == "ctrl+n":
			updated, cmd := m.executeCommandFrom(ChangesListState, "/new-change")
			return updated.(Model), cmd, true
		case key == "ctrl+f" || msg.Type == tea.KeyCtrlF:
			updated, cmd := m.executeCommandFrom(ChangesListState, "/find-filter")
			return updated.(Model), cmd, true
		}
	case ChangeDetailsState:
		switch {
		case key == "ctrl+h":
			next, cmd := m.selectedDocumentHistory()
			return next.(Model), cmd, true
		case key == "up":
			m.changeList = m.changeList.MoveDetailSelection(-1, m.changeTableRows(), terminalWidth(m.width))
			return m, nil, true
		case key == "down":
			m.changeList = m.changeList.MoveDetailSelection(1, m.changeTableRows(), terminalWidth(m.width))
			return m, nil, true
		case key == "pgup":
			m.changeList = m.changeList.ScrollDetailViewport(-m.changeTableRows(), m.changeTableRows(), terminalWidth(m.width))
			return m, nil, true
		case key == "pgdown":
			m.changeList = m.changeList.ScrollDetailViewport(m.changeTableRows(), m.changeTableRows(), terminalWidth(m.width))
			return m, nil, true
		case key == "enter" || msg.Type == tea.KeyCtrlJ:
			updated, cmd := m.handleListSelection()
			return updated.(Model), cmd, true
		case key == "ctrl+n":
			updated, cmd := m.executeCommandFrom(ChangeDetailsState, "/new-testcase")
			return updated.(Model), cmd, true
		case key == " " || key == "space":
			updated, cmd := m.handleDetailSpaceToggle()
			return updated.(Model), cmd, true
		case key == "delete" || key == "del":
			updated, cmd := m.handleDetailDelete()
			return updated.(Model), cmd, true
		case key == "ctrl+shift+c" || key == "ctrl+insert":
			updated, cmd := m.handleDetailCopy()
			return updated.(Model), cmd, true
		}
	case EpicDetailsState:
		if key == "up" || key == "down" || key == "pgup" || key == "pgdown" {
			lines, _ := m.viewLines()
			height := m.epicViewportHeight(lines)
			delta := 1
			if key == "pgup" || key == "pgdown" {
				delta = max(1, height)
			}
			if key == "up" || key == "pgup" {
				delta = -delta
			}
			m.epicList = m.epicList.ScrollDetails(delta, terminalWidth(m.width), height)
			return m, nil, true
		}
	case EpicsListState:
		switch {
		case key == "up":
			m.epicList = m.epicList.MoveSelection(-1)
			return m, nil, true
		case key == "down":
			m.epicList = m.epicList.MoveSelection(1)
			return m, nil, true
		case key == "enter" || msg.Type == tea.KeyCtrlJ:
			updated, cmd := m.handleListSelection()
			return updated.(Model), cmd, true
		}
	case ProjectsListState:
		switch {
		case key == "up":
			m.projectList = m.projectList.MoveSelection(-1)
			return m, nil, true
		case key == "down":
			m.projectList = m.projectList.MoveSelection(1)
			return m, nil, true
		case key == "enter" || msg.Type == tea.KeyCtrlJ:
			updated, cmd := m.handleListSelection()
			return updated.(Model), cmd, true
		}
	}
	return m, nil, false
}

func (m Model) handleFindKey(key string, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if isPromptNewlineKey(msg) {
		return m.insertPromptNewline(), nil
	}
	var handled bool
	m, handled = m.handlePendingShiftEnter(msg)
	if handled {
		return m, nil
	}
	if isShiftEnterPrefix(msg) {
		m.pendingAltO = true
		return m, nil
	}
	switch key {
	case "ctrl+c":
		if m.input.Value() != "" {
			m = m.setPromptValue("")
			m.status = "prompt cleared"
			return m, nil
		}
		return m.arrive(m.previousState, "cancel")
	case "ctrl+e":
		return m.openPromptEditor(m.state)
	case "esc":
		if m.input.Value() != "" {
			m = m.setPromptValue("")
			m.status = "prompt cleared"
			return m, nil
		}
		return m.arrive(m.previousState, "cancel")
	case "enter":
		return m.submitFindValue(m.input.Value())
	}
	return m.updatePromptInput(msg)
}

func isPromptNewlineKey(msg tea.KeyMsg) bool {
	key := msg.String()
	return key == "shift+enter" || (msg.Type == tea.KeyEnter && msg.Alt) || msg.Type == tea.KeyCtrlJ
}

func (m Model) submitPrompt() (tea.Model, tea.Cmd) {
	value := m.promptValue()
	if m.editorDraft != nil {
		return m.submitPromptValue(value)
	}
	if m.detailEditField != "" && strings.TrimSpace(value) == "/cancel" {
		return m.handlePromptCancel()
	}
	trimmed := strings.TrimSpace(value)
	if m.state == DocumentState && m.documentForm {
		return m.submitPromptValue(value)
	}
	if (commandAllowed(m.state, "/save") || m.detailEditField != "") && !commandAllowed(m.state, trimmed) {
		return m.submitPromptValue(value)
	}
	if m.detailEditField == "" && strings.HasPrefix(trimmed, "/") {
		m = m.setPromptValue("")
		return m.executeCommand(trimmed)
	}
	if m.editorDraft == nil && m.detailEditField == detailEditTitle && strings.HasPrefix(trimmed, "/") {
		return m.executeCommand(trimmed)
	}
	return m.submitPromptValue(value)
}

// submitPromptValue submits data, including editor output, without command dispatch.
func (m Model) submitPromptValue(value string) (tea.Model, tea.Cmd) {
	if m.state == DocumentState && m.documentForm {
		return m.beginDocumentInsert(value)
	}
	if m.detailEditField != "" && (m.state == ChangeDetailsState || m.state == ChangeUpdateState || m.state == ChangeCreateState) {
		return m.saveChangeDetailTextValue(value)
	}
	if commandAllowed(m.state, "/save") {
		if m.state == EpicCreateState {
			return m.beginEpic(epics.Create, 0, value)
		}
		if m.state == EpicUpdateState {
			return m.beginEpic(epics.Edit, m.epicList.Detail.ID, value)
		}
		if m.state == ChangeCreateState {
			return m.saveChangeCreateValue(value)
		}
		if m.state == ChangeUpdateState {
			return m.saveChangeUpdateValue(value)
		}
		if m.state == TestCaseCreateState {
			return m.saveTestCaseCreateValue(value)
		}
		if m.state == TestCaseUpdateState {
			return m.saveTestCaseUpdateValue(value)
		}
		if m.state == ProjectCreateState {
			return m.saveProjectCreateValue(value)
		}
		if m.state == ProjectUpdateState {
			return m.saveProjectUpdateValue(value)
		}
		return m.executeCommandFrom(m.state, "/save")
	}
	if m.state == FindInputState {
		return m.submitFindValue(value)
	}
	m = m.setPromptValue("")
	if strings.TrimSpace(value) == "" {
		return m.handleListSelection()
	}
	return m, nil
}

func (m Model) submitFindValue(value string) (tea.Model, tea.Cmd) {
	query := strings.TrimSpace(value)
	if query == "" {
		if m.previousState == ChangesListState {
			if len(m.changeList.Rows) == 0 {
				next, cmd := m.arrive(ChangesListState, "find filter")
				m = next.(Model)
				m.err = "find text is required"
				return m, cmd
			}
			m = m.setPromptValue("")
			m.state = ChangesListState
			m.input.Placeholder = defaultInputPlaceholder
			m.err = "find text is required"
			return m, nil
		}
		next, cmd := m.arrive(m.previousState, m.status)
		m = next.(Model)
		m.err = "find text is required"
		return m, cmd
	}
	if m.previousState == ChangesListState {
		m.rememberSelectedChange()
		m = m.setPromptValue("")
		m.changesFilters.find = query
		m.state = ChangesListState
		m.restoreSelectedChange()
		m.input.Placeholder = defaultInputPlaceholder
		m.status = "find filter"
		if len(m.changeList.Rows) == 0 {
			return m.arrive(ChangesListState, "find filter")
		}
		return m, nil
	}
	m = m.setPromptValue("")
	m.helpQuery = query
	return m.arrive(m.previousState, "highlight "+query)
}

func (m Model) handlePromptCancel() (tea.Model, tea.Cmd) {
	m.editorGeneration++
	if m.editorDraft != nil || m.input.Value() != "" || m.detailEditField != "" {
		m.detailEditField = ""
		m.editorDraft = nil
		m = m.setPromptValue("")
		m.input.Placeholder = defaultInputPlaceholder
		if m.state == ChangeUpdateState {
			m.state = ChangeDetailsState
		}
		if m.state == ChangesListState {
			m.clampChangeListSelection()
		}
		m.status = "prompt cleared"
		return m, nil
	}
	return m.handleEsc()
}

func (m Model) requestQuit() (tea.Model, tea.Cmd) {
	m.configurations = m.configurations.Invalidate()
	m = m.cancelConfigurationCatalogRefresh()
	m.health = m.health.Invalidate()
	m.testCase = m.testCase.Invalidate()
	m.history = m.history.Invalidate()
	m.historyOpen, m.historyReturning = false, false
	m.changeDocuments = m.changeDocuments.Invalidate()
	m.document = m.document.Invalidate()
	m.changeList = m.changeList.Invalidate()
	m.epicList = m.epicList.Invalidate()
	if m.configSaveInFlight {
		m.quitRequested = true
		m.status = "saving project selection before exit"
		return m, nil
	}
	m.state = DoneState
	m.quitting = true
	return m, tea.Quit
}

func (m Model) handleEsc() (tea.Model, tea.Cmd) {
	switch m.state {
	case MainState:
		return m.requestQuit()
	case DocumentState:
		if m.documentForm {
			m.documentForm = false
			m.document.DraftBody = ""
			m = m.setPromptValue("")
			m.status = "document draft canceled"
			return m, nil
		}
		return m.documentCommand("/return")
	case ChangeUpdateState:
		if m.detailEditField != "" {
			m.detailEditField = ""
			m = m.setPromptValue("")
			return m.arrive(ChangeDetailsState, "cancel")
		}
		return m.arrive(navigation.CancelTarget(m.state), "cancel")
	case ChangeCreateState, TestCaseCreateState, TestCaseUpdateState,
		EpicCreateState, EpicUpdateState, ProjectCreateState, ProjectUpdateState:
		m = m.setPromptValue("")
		if m.state == TestCaseCreateState || m.state == TestCaseUpdateState {
			m.detailEditField = ""
			m.testCase = m.testCase.ClearForm()
			m = m.setPromptValue("")
		}
		return m.arrive(navigation.CancelTarget(m.state), "cancel")
	default:
		if target, ok := navigation.ReturnTargets()[m.state]; ok {
			return m.arrive(target, "return")
		}
		m.err = "cannot cancel from this state"
		return m, nil
	}
}

func (m Model) handleListSelection() (tea.Model, tea.Cmd) {
	switch m.state {
	case EpicDetailsState:
		return m.epicForm(true)
	case ChangesListState:
		next, selected, ok := m.changeList.SelectDetail(m.changeFilters())
		m.changeList = next
		if !ok {
			m.err = "no changes selectable"
			return m, nil
		}
		m = m.setPromptValue("")
		m.changeSelectionID = selected.ID
		m.state = ChangeDetailsState
		m.changeDetailLoaded = false
		m.status = "selected " + selected.Title
		id, err := changeNumericID(selected)
		if err != nil {
			m.err = err.Error()
			return m, nil
		}
		return m.beginChange(changes.Details, id, changes.Input{})
	case ChangeDetailsState:
		if !m.changeDetailLoaded {
			m.err = "load change details with /retry before editing"
			return m, nil
		}
		next, row, ok := m.changeList.SelectDetailRow(m.changeTableRows(), terminalWidth(m.width))
		m.changeList = next
		if !ok {
			m.err = "no change details selectable"
			return m, nil
		}
		if row.Comment && row.DocumentID > 0 {
			return m.beginComment(row.DocumentID)
		}
		if row.DocumentType != "" && row.DocumentType != "brief" && !row.Comment {
			m.changeList.Draft.DocumentType = row.DocumentType
			return m.beginDetailTextEditor(detailEditDocument)
		}
		if row.TestCaseID != "" {
			return m.beginTestCaseScenarioEdit(row)
		}
		switch row.Label {
		case "Slug":
			return m.beginChangeField(detailEditSlug)
		case "Phase":
			return m.beginDetailFieldSelector(detailEditPhase)
		case "Epic":
			return m.beginDetailFieldSelector(detailEditEpic)
		case "Types":
			return m.beginDetailFieldSelector(detailEditTypes)
		case "After Change":
			return m.beginChangeField(detailEditAfterChange)
		case "Title":
			return m.beginDetailTitleEdit()
		case "Brief":
			return m.beginDetailTextEditor(detailEditBrief)
		case "Spec":
			return m.beginDetailTextEditor(detailEditSpec)
		case "Pull Request", "PR":
			return m.beginDetailTextEditor(detailEditPullRequest)
		case "PR URL":
			return m.beginChangeField(detailEditPRUrl)
		}
		m.status = "selected " + row.Label
	case EpicsListState:
		next, selected, ok := m.epicList.SelectDetail()
		m.epicList = next
		if !ok {
			m.err = epics.NoSelectableError
			return m, nil
		}
		m.state = EpicDetailsState
		return m.beginEpic(epics.Details, selected.ID, "")
	case ProjectsListState:
		next, selected, ok := m.projectList.SelectDetail()
		m.projectList = next
		if !ok {
			m.err = projects.NoSelectableError
			return m, nil
		}
		m.state = ProjectDetailsState
		m.status = "selected " + projects.DisplayName(selected)
		return m.beginProject(projects.Details, selected.ID, "")
	default:
		m.err = "nothing selectable in current state"
	}
	return m, nil
}

func (m Model) executeCommand(command string) (tea.Model, tea.Cmd) {
	return m.executeCommandFrom(m.state, command)
}

func (m Model) executeCommandFrom(source State, command string) (tea.Model, tea.Cmd) {
	m.state = source
	m.dropdown = dropdownModel{}
	if command != "/quit" && !commandAllowed(source, command) {
		m.err = "unknown command: " + command
		return m, nil
	}

	if source == DocumentState {
		return m.documentCommand(command)
	}
	if source == BriefState {
		return m.briefCommand(command)
	}
	if isConfigurationState(source) {
		return m.configurationCommand(source, command)
	}
	if source == HealthState {
		return m.healthCommand(command)
	}
	switch command {
	case "/new-comment":
		return m.beginComment(0)
	case "/brief-new":
		return m.openBrief(true)
	case "/brief-clarify":
		return m.openBrief(false)
	case "/documents":
		return m.openDocuments(source)
	case "/quit":
		if source != MainState {
			m.err = "/quit is only available from MainState"
			return m, nil
		}
		return m.requestQuit()
	case "/changes":
		return m.arrive(ChangesListState, string(ChangesListState))
	case "/epics":
		return m.arrive(EpicsListState, string(EpicsListState))
	case "/projects":
		return m.arrive(ProjectsListState, string(ProjectsListState))
	case "/backend-configs":
		m.state = BackendConfigListState
		m.configurations = m.configurations.Invalidate()
		m.configurations.Form = false
		m.configurations.Confirm = false
		m.configurations.Editing = false
		m.configurations.Committed = ""
		m.configurations.Refresh = ""
		return m.beginConfiguration(configurations.List, "")
	case "/health":
		m.state = HealthState
		m.health = m.health.SelectRoute("/api/v1/health")
		return m.beginHealth()
	case "/select-project":
		return m.beginSelector(SelectProjectDropDown)
	case "/project-config":
		return m.beginProject(projects.Config, m.projectList.Detail.ID, "")
	case "/retry":
		if m.historyOpen {
			var cmd tea.Cmd
			m.history, cmd = m.history.Refresh(m.ctx, m.client, m.historyPrinter)
			return m, cmd
		}
		if source == ChangeDetailsState && m.changeDocuments.Committed != "" && m.changeDocuments.Err != nil && m.changeDocuments.ProjectID == m.appConfig.ProjectID && strconv.Itoa(m.changeDocuments.OwnerID) == m.changeList.Detail.ID {
			return m.beginChangeDocumentMutation(documents.Read, 0, "")
		}
		if source == ChangesListState {
			m.rememberSelectedChange()
			return m.beginChange(changes.List, 0, changes.Input{})
		}
		if source == ChangeDetailsState {
			if m.testCase.NeedsRefresh() {
				return m.beginTestCase(testcases.Refresh, "", "", false)
			}
			id, _ := changeNumericID(m.changeList.Detail)
			return m.beginChange(changes.Details, id, changes.Input{})
		}
		if source == EpicsListState {
			return m.beginEpic(epics.List, 0, "")
		}
		if source == EpicDetailsState {
			return m.beginEpic(epics.Details, m.epicList.Detail.ID, "")
		}
		if source == ProjectsListState {
			return m.beginProject(projects.List, 0, "")
		}
		if m.projectList.ShowConfig {
			return m.beginProject(projects.Config, m.projectList.Detail.ID, "")
		}
		return m.beginProject(projects.Details, m.projectList.Detail.ID, "")
	case "/config":
		return m.arrive(ConfigState, string(ConfigState))
	case "/help":
		m.changeList = m.changeList.Invalidate()
		m.epicList = m.epicList.Invalidate()
		m.state = helpStateFor(source)
	case "/find":
		m.changeList = m.changeList.Invalidate()
		m.epicList = m.epicList.Invalidate()
		m.previousState = source
		m.state = FindInputState
		m = m.setPromptValue("")
	case "/find-filter":
		m.rememberSelectedChange()
		m.changeList = m.changeList.Invalidate()
		m.previousState = ChangesListState
		m.state = FindInputState
		m = m.setPromptValue(m.changesFilters.find)
		m.input.Placeholder = "Find changes"
	case "/clear-filters":
		m.rememberSelectedChange()
		m.changesFilters = changesFilters{}
		m.restoreSelectedChange()
		m.status = "filters cleared"
	case "/return":
		if source == ChangesListState && m.changeList.Inactive {
			return m.leaveInactiveChanges()
		}
		return m.arrive(navigation.ReturnTargets()[source], "return")
	case "/new-change", "/new-testcase", "/new-test-case", "/new-epic", "/new-project":
		if source == ChangeDetailsState && (command == "/new-testcase" || command == "/new-test-case") && !m.changeDetailLoaded {
			m.err = "load change details with /retry before editing"
			return m, nil
		}
		if command == "/new-epic" {
			return m.epicForm(false)
		}
		if command == "/new-change" {
			if _, err := currentProjectNumericID(m.currentProject.ID); err != nil {
				m.err = err.Error()
				return m, nil
			}
		}
		m.projectList = m.projectList.Invalidate()
		m.state = navigation.CreateTarget(source)
		if m.state == ChangeCreateState {
			m.changeList = m.changeList.Invalidate()
			m.changeList.Draft = changes.Input{}
			m.detailEditField = ""
			m = m.setPromptValue("")
			m.input.Placeholder = defaultInputPlaceholder
			return m.openPromptEditor(ChangeCreateState)
		}
		if m.state == TestCaseCreateState {
			m.testCase = m.testCase.OpenCreate()
			form := testcases.CreateForm()
			m.input.Placeholder = form.Placeholder
			m = m.setPromptValue("")
			m.status = form.Status
			return m, nil
		}
		if m.state == ProjectCreateState {
			m = m.setPromptValue("")
			m.input.Placeholder = "Write a Name"
		} else {
			m.input.Placeholder = defaultInputPlaceholder
		}
	case "/edit", "/edit-spec":
		if source == EpicDetailsState {
			return m.epicForm(true)
		}
		if command == "/edit" && source == ChangeDetailsState {
			m.err = "/edit is not available from ChangeDetailsState; use /edit-spec"
			return m, nil
		}
		if command == "/edit-spec" && source != ChangeDetailsState {
			m.err = "/edit-spec is only available from ChangeDetailsState"
			return m, nil
		}
		if command == "/edit-spec" {
			return m.beginDetailTextEditor(detailEditSpec)
		}
		m.projectList = m.projectList.Invalidate()
		m.state = navigation.UpdateTarget(source)
		if m.state == ChangeUpdateState {
			m = m.setPromptValue(m.changeList.Detail.Spec)
			m.input.Placeholder = defaultInputPlaceholder
			return m.openPromptEditor(ChangeUpdateState)
		}
		if m.state == ProjectUpdateState {
			m = m.setPromptValue(m.projectList.Detail.Name)
		}
		m.input.Placeholder = defaultInputPlaceholder
	case "/save":
		if source == EpicCreateState {
			return m.beginEpic(epics.Create, 0, m.promptValue())
		}
		if source == EpicUpdateState {
			return m.beginEpic(epics.Edit, m.epicList.Detail.ID, m.promptValue())
		}
		if source == ChangeCreateState {
			return m.saveChangeCreate()
		}
		if source == ChangeUpdateState {
			return m.saveChangeUpdate()
		}
		if source == TestCaseCreateState {
			return m.saveTestCaseCreateValue(m.input.Value())
		}
		if source == TestCaseUpdateState {
			return m.saveTestCaseUpdateValue(m.input.Value())
		}
		if source == ProjectCreateState {
			return m.saveProjectCreate()
		}
		if source == ProjectUpdateState {
			return m.saveProjectUpdate()
		}
		m.err = "saving is not available for this screen"
	case "/editor":
		return m.openPromptEditor(source)
	case "/cancel":
		if (source == ChangeUpdateState || source == TestCaseUpdateState) && m.detailEditField != "" {
			m.detailEditField = ""
			m = m.setPromptValue("")
		}
		if source == TestCaseCreateState || source == TestCaseUpdateState {
			m.testCase = m.testCase.ClearForm()
		}
		return m.arrive(navigation.CancelTarget(source), "cancel")
	case "/delete":
		if source == EpicDetailsState {
			m.epicList = m.epicList.Invalidate()
		}
		m.openConfirmation(navigation.DeleteConfirmationState(source), source, navigation.DeleteReturnState(source))
	case "/title":
		if source == ChangeCreateState {
			m.changeList.Draft.Value = m.promptValue()
			m.detailEditField = detailCreateTitle
			m = m.setPromptValue(m.changeList.Draft.Title)
			return m, nil
		}
		return m.beginChangeField(detailEditTitle)
	case "/uuid":
		m.changeList.Draft.Value = m.promptValue()
		m.detailEditField = detailCreateUUID
		m = m.setPromptValue(m.changeList.Draft.UUID)
		return m, nil
	case "/document":
		m.changeList = m.changeList.Invalidate()
		m.selectorGeneration++
		m.openSelectorDropdown(SelectTypesDropDown, ChangeDetailsState, ChangeDetailsState, "Document type", selectorDocuments)
		m.dropdown.editField = detailEditDocument
		return m, m.selectorCommand(selectorDocuments)
	case "/brief":
		return m.beginDetailTextEditor(detailEditBrief)
	case "/pr-url":
		return m.beginChangeField(detailEditPRUrl)
	case "/after-change":
		return m.beginChangeField(detailEditAfterChange)
	case "/active":
		if !m.changeDetailLoaded {
			m.err = "load change details with /retry before editing"
			return m, nil
		}
		id, _ := changeNumericID(m.changeList.Detail)
		return m.beginChange(changes.Active, id, changes.Input{Active: !m.changeList.Detail.Active})
	case "/phase":
		if source == ChangeDetailsState {
			return m.beginDetailFieldSelector(detailEditPhase)
		}
		return m.beginSelector(SelectPhaseDropDown)
	case "/epic":
		if source == ChangeDetailsState {
			return m.beginDetailFieldSelector(detailEditEpic)
		}
		return m.beginSelector(SelectEpicDropDown)
	case "/types":
		if source == ChangeDetailsState {
			return m.beginDetailFieldSelector(detailEditTypes)
		}
		return m.beginSelector(SelectTypesDropDown)
	case "/phase-filter":
		return m.beginFilter("Phase Filter", selectorPhases, filterPhase)
	case "/epic-filter":
		return m.beginFilter("Epic Filter", selectorEpics, filterEpic)
	case "/types-filter":
		return m.beginFilter("Types Filter", selectorTypes, filterTypes)
	}
	if m.state == "" {
		m.state = source
		m.err = "unknown command: " + command
	}
	return m, nil
}

func (m Model) arrive(state State, status string) (tea.Model, tea.Cmd) {
	if state != ChangeDetailsState {
		m.historyDetailReload = false
	}
	if m.state == ChangesListState {
		m.rememberSelectedChange()
		if state != ChangesListState {
			m = m.setPromptValue("")
		}
	}
	if m.state == ChangeDetailsState && state == ChangesListState && m.changeList.Detail.ID != "" {
		m.changeSelectionID = m.changeList.Detail.ID
	}
	if m.state == FindInputState && state != FindInputState {
		m = m.setPromptValue("")
	}
	if m.state == BriefState && state != BriefState {
		m.brief = m.brief.Invalidate()
		m.briefOperation = nil
	}
	var catalog tea.Cmd
	if isConfigurationState(m.state) && !isConfigurationState(state) {
		m.configurations = m.configurations.Invalidate()
		refreshCanceled := m.configCatalogCancel != nil
		m = m.cancelConfigurationCatalogRefresh()
		if refreshCanceled && m.appConfig.ProjectID > 0 {
			catalog = optionCatalogCommand(m.ctx, m.client, m.appConfig.ProjectID, m.catalogGeneration)
		}
	}
	withCatalog := func(next tea.Model, cmd tea.Cmd) (tea.Model, tea.Cmd) {
		if catalog == nil {
			return next, cmd
		}
		return next, tea.Batch(cmd, catalog)
	}
	if m.state == HealthState && state != HealthState {
		m.health = m.health.Invalidate()
	}
	if state != ChangeDetailsState && state != TestCaseCreateState && state != TestCaseUpdateState {
		m.testCase = m.testCase.Invalidate()
	}
	m.changeList = m.changeList.Invalidate()
	m.epicList = m.epicList.Invalidate()
	if m.state == ChangeCreateState {
		m.detailEditField = ""
	}
	if m.state == EpicCreateState || m.state == EpicUpdateState || m.state == ChangeCreateState {
		m = m.setPromptValue("")
	}
	m.projectList = m.projectList.Invalidate()
	m.state = state
	m.status = status
	m.applyPromptLimit()
	if state != ProjectCreateState {
		m.input.Placeholder = defaultInputPlaceholder
	}
	switch state {
	case EpicsListState:
		return withCatalog(m.beginEpic(epics.List, 0, ""))
	case EpicDetailsState:
		return withCatalog(m.beginEpic(epics.Details, m.epicList.Detail.ID, ""))
	case ChangesListState:
		return withCatalog(m.beginChange(changes.List, 0, changes.Input{}))
	case ChangeDetailsState:
		m.changeDetailLoaded = false
		id, err := changeNumericID(m.changeList.Detail)
		if err != nil {
			m.err = err.Error()
			return withCatalog(m, nil)
		}
		m.status = "loading change"
		return withCatalog(m.beginChange(changes.Details, id, changes.Input{}))
	case ProjectsListState:
		return withCatalog(m.beginProject(projects.List, 0, ""))
	case ProjectDetailsState:
		return withCatalog(m.beginProject(projects.Details, m.projectList.Detail.ID, ""))
	default:
		return withCatalog(m, nil)
	}
}

func (m Model) beginSelector(state State) (tea.Model, tea.Cmd) {
	previous := m.state
	if previous == CommandDropDownState {
		previous = m.dropdown.previous
	}
	onSelect := previous
	if state == SelectProjectDropDown {
		m.changeList = m.changeList.Invalidate()
		onSelect = MainState
	}
	source := selectorSourceForState(state)
	m.selectorGeneration++
	m.openSelectorDropdown(state, previous, onSelect, string(state), source)
	return m, m.selectorCommand(source)
}

func (m Model) beginFilter(label string, source selectorSource, field filterField) (tea.Model, tea.Cmd) {
	m.selectorGeneration++
	m.openFilterDropdown(label, source, field)
	return m, m.selectorCommand(source)
}

func (m Model) beginDetailTitleEdit() (tea.Model, tea.Cmd) {
	return m.beginChangeField(detailEditTitle)
}

func (m Model) beginDetailTextEditor(field detailEditField) (tea.Model, tea.Cmd) {
	if !m.changeDetailLoaded {
		m.err = "Load change details before editing; wait for loading or return to the list and select the change again"
		return m, nil
	}
	m.changeList = m.changeList.Invalidate()
	m.detailEditField = field
	var original string
	switch field {
	case detailEditDocument:
		for _, d := range m.changeList.Detail.Documents {
			if d.DocType == m.changeList.Draft.DocumentType {
				original = d.Body
			}
		}
	case detailEditBrief:
		original = m.changeList.Detail.Brief
	case detailEditSpec:
		original = m.changeList.Detail.Spec
	case detailEditPullRequest:
		original = m.changeList.Detail.PR
	case detailEditPRUrl:
		original = m.changeList.Detail.PRUrl
	default:
		m.err = "unsupported editable detail text field"
		return m, nil
	}
	m = m.setPromptValue(original)
	return m.openTextEditor(ChangeDetailsState, original)
}

func (m Model) beginTestCaseScenarioEdit(row changes.DetailRow) (tea.Model, tea.Cmd) {
	m.previousState = ChangeDetailsState
	m.state = TestCaseUpdateState
	m.detailEditField = detailEditTestCase
	m.testCase = m.testCase.OpenEdit(row.TestCaseID, row.TestCaseText)
	form := testcases.EditForm()
	m.input.Placeholder = form.Placeholder
	m = m.setPromptValue(row.TestCaseText)
	m.status = form.Status
	return m, nil
}

func (m Model) beginDetailFieldSelector(field detailEditField) (tea.Model, tea.Cmd) {
	if !m.changeDetailLoaded {
		m.err = "load change details with /retry before editing"
		return m, nil
	}
	m.changeList = m.changeList.Invalidate()
	m.detailEditField = ""
	switch field {
	case detailEditPhase:
		m.openSelectorDropdown(SelectPhaseDropDown, ChangeDetailsState, ChangeDetailsState, "Phase", selectorPhases)
	case detailEditEpic:
		m.openSelectorDropdown(SelectEpicDropDown, ChangeDetailsState, ChangeDetailsState, "Epic", selectorEpics)
	case detailEditTypes:
		m.openSelectorDropdown(SelectTypesDropDown, ChangeDetailsState, ChangeDetailsState, "Types", selectorTypes)
	default:
		m.err = "unsupported editable detail field"
		return m, nil
	}
	m.selectorGeneration++
	m.dropdown.editField = field
	if field == detailEditTypes {
		m.dropdown.pendingTypes = normalizeTypeSet(m.changeList.Detail.ChangeTypes)
	}
	m.state = ChangeDetailsState
	return m, m.selectorCommand(m.dropdown.source)
}

func (m Model) handleDetailSpaceToggle() (tea.Model, tea.Cmd) {
	if !m.changeDetailLoaded {
		m.err = "load change details with /retry before editing"
		return m, nil
	}
	next, row, ok := m.changeList.SelectDetailRow(m.changeTableRows(), terminalWidth(m.width))
	m.changeList = next
	if !ok {
		m.err = "no change details selectable"
		return m, nil
	}
	switch {
	case row.Label == "Active":
		m.status = "saving active"
		id, _ := changeNumericID(m.changeList.Detail)
		return m.beginChange(changes.Active, id, changes.Input{Active: !m.changeList.Detail.Active})
	case row.TestCaseID != "":
		return m.beginTestCase(testcases.SetDone, row.TestCaseID, "", !row.TestCaseDone)
	default:
		return m, nil
	}
}

func (m Model) handleDetailDelete() (tea.Model, tea.Cmd) {
	if !m.changeDetailLoaded {
		m.err = "load change details with /retry before editing"
		return m, nil
	}
	next, row, ok := m.changeList.SelectDetailRow(m.changeTableRows(), terminalWidth(m.width))
	m.changeList = next
	if !ok {
		m.err = "no change details selectable"
		return m, nil
	}
	if row.DocumentID > 0 {
		return m.openDocumentConfirmation(row.DocumentID)
	}
	if row.TestCaseID == "" {
		return m, nil
	}
	m.testCase = m.testCase.OpenDelete(row.TestCaseID)
	m.openConfirmation(TestCaseDeleteConfirmation, ChangeDetailsState, ChangeDetailsState)
	return m, nil
}

func (m Model) needsProjectSelection() bool {
	return m.currentProject.ID == "" && m.appConfig.ProjectID <= 0
}
