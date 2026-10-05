package app

import (
	"cli/internal/agent"
	"cli/internal/changes"
	"cli/internal/documents"
	"cli/internal/dto"
	"cli/internal/testcases"
	"cli/pkg/briefprocess"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

type agentOperation struct {
	ctx           context.Context
	parent        context.Context
	cancel        context.CancelFunc
	projectID     string
	events        chan tea.Msg
	progress      chan string
	files         *briefprocess.Workspace
	path, uuid    string
	title, brief  string
	savedDocument *dto.Document
}

type newBriefEditedMsg struct {
	operation *agentOperation
	err       error
}
type agentInteractiveMsg struct {
	operation *agentOperation
	command   *exec.Cmd
	reply     chan error
}
type agentInteractiveFinishedMsg struct {
	request agentInteractiveMsg
	err     error
}
type agentFinishedMsg struct {
	operation *agentOperation
	result    agent.Result
}
type agentProgressMsg struct {
	operation *agentOperation
	text      string
}
type savedDocumentSyncMsg struct {
	project       string
	change        int
	generation    uint64
	err           error
	status        string
	savedDocument dto.Document
}
type agentDetailMsg struct {
	result               tea.Msg
	project, status, err string
}

func waitAgentOperation(op *agentOperation) tea.Cmd {
	return func() tea.Msg {
		var parentDone <-chan struct{}
		if op.parent != nil {
			parentDone = op.parent.Done()
		}
		select {
		case text := <-op.progress:
			return agentProgressMsg{operation: op, text: text}
		default:
		}
		select {
		case <-parentDone:
			return nil
		case event := <-op.events:
			return event
		case text := <-op.progress:
			return agentProgressMsg{operation: op, text: text}
		}
	}
}

func (m Model) newChangeBrief() (tea.Model, tea.Cmd) {
	if m.agentOperation != nil {
		return m, nil
	}
	if _, err := currentProjectNumericID(m.currentProject.ID); err != nil {
		m.err = err.Error()
		return m, nil
	}
	ref := make([]byte, 16)
	if _, err := rand.Read(ref); err != nil {
		m.err = err.Error()
		return m, nil
	}
	ref[6] = (ref[6] & 15) | 64
	ref[8] = (ref[8] & 63) | 128
	uuid := fmt.Sprintf("%x-%x-%x-%x-%x", ref[0:4], ref[4:6], ref[6:8], ref[8:10], ref[10:16])
	files := &briefprocess.Workspace{}
	path, err := files.Prepare(uuid, "")
	m.state = ChangesListState
	if err != nil {
		m.err = err.Error()
		return m, nil
	}
	ctx, cancel := context.WithCancel(m.ctx)
	op := &agentOperation{ctx: ctx, parent: m.ctx, cancel: cancel, projectID: m.currentProject.ID, files: files, path: path, uuid: uuid}
	m.agentOperation = op
	m.status = "editing new brief"
	return m, tea.ExecProcess(editorProcess(ctx, m.appConfig.Editor, path), func(err error) tea.Msg { return newBriefEditedMsg{operation: op, err: err} })
}

func (m Model) applyNewBrief(msg newBriefEditedMsg) (tea.Model, tea.Cmd) {
	op := msg.operation
	if m.agentOperation != op || op.projectID != m.currentProject.ID {
		return m, nil
	}
	if msg.err != nil {
		op.cancel()
		m.agentOperation = nil
		m.err = fmt.Sprintf("editor failed; draft retained at %s: %v", op.path, msg.err)
		return m, nil
	}
	brief, err := op.files.Read(op.path)
	if err != nil {
		op.cancel()
		m.agentOperation = nil
		m.err = err.Error()
		return m, nil
	}
	parsed, err := changes.ParseBriefStructure(brief)
	if err != nil {
		op.cancel()
		m.agentOperation = nil
		m.err = fmt.Sprintf("%v; draft retained at %s", err, op.path)
		return m, nil
	}
	project, _ := currentProjectNumericID(m.currentProject.ID)
	return m.launchAgent(op, agent.Request{Root: m.appConfig.RepositoryRoot, ProjectID: project, UUID: op.uuid, Path: op.path, Title: parsed.Title, Brief: brief, ChangeTypes: parsed.ChangeTypes, ChangeTypesPresent: parsed.ChangeTypesPresent})
}

func (m Model) startSavedBrief(change int, saved dto.Document) (tea.Model, tea.Cmd) {
	if m.agentOperation != nil {
		return m, nil
	}
	project, _ := currentProjectNumericID(m.currentProject.ID)
	ctx, cancel := context.WithCancel(m.ctx)
	op := &agentOperation{ctx: ctx, parent: m.ctx, cancel: cancel, projectID: m.currentProject.ID, files: &briefprocess.Workspace{}, savedDocument: &saved}
	return m.launchAgent(op, agent.Request{Root: m.appConfig.RepositoryRoot, ProjectID: project, ChangeID: change})
}

func (m Model) launchAgent(op *agentOperation, req agent.Request) (tea.Model, tea.Cmd) {
	op.events, op.progress = make(chan tea.Msg, 1), make(chan string, 32)
	m.agentOperation = op
	m.agentOutput = ""
	m.agentOffset = 0
	op.title, op.brief = req.Title, req.Brief
	m.state = AgentExecState
	m.err, m.status = "", "brief saved; rewrite / spec writing"
	if req.ChangeID == 0 {
		m.status = "creating change from brief"
	}
	api, runner := m.client, m.briefRunner
	return m, func() tea.Msg {
		go func() {
			interactive := func(command *exec.Cmd) error {
				reply := make(chan error, 1)
				select {
				case op.events <- agentInteractiveMsg{operation: op, command: command, reply: reply}:
				case <-op.ctx.Done():
					return op.ctx.Err()
				}
				select {
				case err := <-reply:
					return err
				case <-op.ctx.Done():
					return op.ctx.Err()
				}
			}
			syncCases := func(ctx context.Context, id int, body string) error { return testcases.Synchronize(ctx, api, id, body) }
			result := agent.Run(op.ctx, api, op.files, runner, req, interactive, op.progress, syncCases)
			select {
			case op.events <- agentFinishedMsg{operation: op, result: result}:
			case <-m.ctx.Done():
			}
		}()
		return waitAgentOperation(op)()
	}
}

func (m Model) applyAgentFinished(msg agentFinishedMsg) (tea.Model, tea.Cmd) {
	op, r := msg.operation, msg.result
	if m.agentOperation != op || op.projectID != m.currentProject.ID {
		return m, nil
	}
	op.cancel()
	m.agentOperation = nil
	m.status, m.err = r.Status, ""
	if r.Err != nil {
		m.err = r.Err.Error()
		if strings.Contains(m.err, "error generating `spec`") {
			m.err = "Error generating `spec`"
		}
	}
	m.state = ChangesListState
	if r.ChangeID == 0 {
		return m, nil
	}
	m.state = ChangeDetailsState
	if m.changeList.Detail.ID != strconv.Itoa(r.ChangeID) {
		m.changeList = m.changeList.Invalidate()
		m.changeList.DetailLoaded, m.changeDetailLoaded = false, false
		m.testCase = m.testCase.Invalidate().ClearForm()
		m.testCase.Loaded = false
		m.changeList.Detail = dto.ChangeView{ID: strconv.Itoa(r.ChangeID), ProjectID: m.currentProject.ID, RefUUID: op.uuid, Title: op.title, Brief: strings.TrimSpace(op.brief), DocumentTypes: m.optionCatalog.config.ChangeDocs}
	}
	if r.ChangeTypesSaved {
		m.changeList.Detail.ChangeTypes = r.ChangeTypes
	}
	for _, doc := range []*dto.Document{r.Brief, r.Spec} {
		if doc == nil {
			continue
		}
		m = m.retainSavedChangeDocument(*doc)
	}
	m.detailEditField = ""
	m = m.setPromptValue("")
	if r.Err != nil {
		return m.invalidateFailedSync(r.Err), nil
	}
	if op.savedDocument != nil {
		project, _ := strconv.Atoi(op.projectID)
		m = m.cleanupEditorDraft(project, *op.savedDocument)
	}
	return m.refreshSavedDetails(m.status, m.err)
}

func (m Model) invalidateFailedSync(err error) Model {
	var syncErr *testcases.SyncError
	if errors.As(err, &syncErr) {
		m.changeList = m.changeList.Invalidate()
		m.changeList.DetailLoaded = false
		m.changeDetailLoaded = false
		m.testCase = m.testCase.Invalidate().ClearForm()
		m.testCase.Loaded = false
		m.err += "; /retry reads current details and testcases"
	}
	return m
}

func (m Model) refreshSavedDetails(status, errText string) (tea.Model, tea.Cmd) {
	id, _ := changeNumericID(m.changeList.Detail)
	next, cmd := m.beginChange(changes.Details, id, changes.Input{})
	m = next.(Model)
	m.status, m.err = status, errText
	if cmd == nil {
		return m, nil
	}
	project := m.currentProject.ID
	return m, func() tea.Msg { return agentDetailMsg{result: cmd(), project: project, status: status, err: errText} }
}

func (m Model) syncSavedSpec(change int, body string) (tea.Model, tea.Cmd) {
	m.state = ChangeDetailsState
	m.detailEditField = ""
	m = m.setPromptValue("")
	m.changeList.Detail.ID = strconv.Itoa(change)
	ctx, cancel := context.WithCancel(m.ctx)
	m.specSyncCancel = cancel
	project, generation, api, status := m.currentProject.ID, m.selectionGeneration, m.client, m.status
	return m, func() tea.Msg {
		return savedDocumentSyncMsg{project: project, change: change, generation: generation, status: status, savedDocument: dto.Document{RefID: change, RefTable: "change", DocType: "spec", Body: body}, err: testcases.Synchronize(ctx, api, change, body)}
	}
}

func (m Model) agentView(width int) string {
	lines := strings.Split(documents.SafeANSI(m.agentOutput), "\n")
	page := max(1, m.height-10)
	start := max(0, len(lines)-page-m.agentOffset)
	end := min(len(lines), start+page)
	for i := start; i < end; i++ {
		lines[i] = ansi.Truncate(lines[i], width, "")
	}
	return strings.Join(lines[start:end], "\n")
}
