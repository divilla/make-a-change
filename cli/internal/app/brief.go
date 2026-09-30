package app

import (
	"cli/internal/agent"
	"cli/internal/documents"
	"cli/pkg/briefprocess"
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type briefCleanupMsg struct{ err error }

type briefOperation struct {
	ctx      context.Context
	progress chan string
	result   chan agent.Result
}

type briefProgressMsg struct {
	operation *briefOperation
	text      string
}

func waitBriefOperation(operation *briefOperation) tea.Cmd {
	return func() tea.Msg {
		select {
		case <-operation.ctx.Done():
			return nil
		case result := <-operation.result:
			return result
		case progress := <-operation.progress:
			return briefProgressMsg{operation: operation, text: progress}
		}
	}
}

func (m Model) launchBriefOperation(run func(agent.API, agent.Runner, string) agent.Result, progress chan string) (tea.Model, tea.Cmd) {
	client, runner, root := m.client, m.briefRunner, m.appConfig.RepositoryRoot
	if progress == nil {
		return m, func() tea.Msg { return run(client, runner, root) }
	}
	operation := &briefOperation{ctx: m.ctx, progress: progress, result: make(chan agent.Result, 1)}
	m.briefOperation = operation
	return m, func() tea.Msg {
		go func() { operation.result <- run(client, runner, root) }()
		return waitBriefOperation(operation)()
	}
}

func (m Model) openBrief(newChange bool) (tea.Model, tea.Cmd) {
	project, err := currentProjectNumericID(m.currentProject.ID)
	if err != nil {
		m.err = err.Error()
		return m, nil
	}
	change := 0
	if !newChange {
		if m.state != ChangeDetailsState || !m.changeDetailLoaded {
			m.err = "load change details before clarifying its brief"
			return m, nil
		}
		change, err = changeNumericID(m.changeList.Detail)
		if err != nil {
			m.err = err.Error()
			return m, nil
		}
	}
	m.briefReturn = m.state
	m.state = BriefState
	m.briefField = "brief"
	m.briefOffset = 0
	previous := m.brief.Invalidate()
	if newChange {
		m.brief = agent.New(project)
	} else {
		m.brief = agent.Existing(project, change)
	}
	m.brief.Generation = previous.Generation
	m = m.setPromptValue("")
	return m.beginBrief(agent.Preflight)
}

func (m Model) beginBrief(step agent.Step) (tea.Model, tea.Cmd) {
	var run func(agent.API, agent.Runner, string) agent.Result
	var progress chan string
	if step == agent.Run {
		progress = make(chan string, 1)
	}
	m.brief, run = m.brief.Begin(m.ctx, step, progress)
	m.status = m.brief.Status
	if m.brief.Err != nil {
		m.err = m.brief.Err.Error()
	}
	if run == nil {
		return m, nil
	}
	return m.launchBriefOperation(run, progress)
}

func (m Model) applyBriefProgress(msg briefProgressMsg) (tea.Model, tea.Cmd) {
	if m.state != BriefState || m.briefOperation != msg.operation || !m.brief.Busy || m.brief.Pending != agent.Run {
		return m, nil
	}
	m.status = "clarifying: " + documents.SafeLine(msg.text)
	return m, waitBriefOperation(msg.operation)
}

func (m Model) applyBriefResult(r agent.Result) (tea.Model, tea.Cmd) {
	if m.state != BriefState || m.currentProject.ID != strconv.Itoa(r.ProjectID) {
		return m, nil
	}
	var next agent.Step
	var ok bool
	m.brief, next, ok = m.brief.Apply(r)
	if !ok {
		return m, nil
	}
	m.briefOperation = nil
	m.status, m.err = m.brief.Status, ""
	if m.brief.Err != nil {
		m.err = m.brief.Err.Error()
	}
	if next != "" {
		return m.beginBrief(next)
	}
	return m, nil
}

func (m Model) briefCommand(command string) (tea.Model, tea.Cmd) {
	switch command {
	case "/title":
		if !m.brief.New || m.brief.ChangeID != 0 {
			m.err = "title applies only before new-change creation"
			return m, nil
		}
		m.briefField = "title"
		m = m.setPromptValue(m.brief.Title)
	case "/uuid":
		if !m.brief.New || m.brief.ChangeID != 0 {
			m.err = "UUID applies only before new-change creation"
			return m, nil
		}
		m.briefField = "uuid"
		m = m.setPromptValue(m.brief.UUID)
	case "/brief":
		m.briefField = "brief"
		m = m.setPromptValue("")
	case "/confirm":
		if m.brief.Busy {
			return m, nil
		}
		if m.brief.CommittedID != 0 {
			return m.beginBrief(agent.Refresh)
		}
		if m.brief.Phase == agent.Review || m.brief.Phase == agent.Answering {
			m.err = "approve the rewrite or resolve answers before another save"
			return m, nil
		}
		if m.brief.Phase == agent.Ready {
			m.err = "brief is already ready"
			return m, nil
		}
		if strings.TrimSpace(m.brief.Draft) == "" {
			m.err = "brief is required"
			return m, nil
		}
		if m.brief.New && strings.TrimSpace(m.brief.Title) == "" {
			m.err = "title is required"
			return m, nil
		}
		return m.beginBrief(agent.Write)
	case "/approve":
		var run func(agent.API, agent.Runner, string) agent.Result
		m.brief, run = m.brief.Approve(m.ctx)
		if run == nil {
			m.status = m.brief.Status
			return m, nil
		}
		client, runner, root := m.client, m.briefRunner, m.appConfig.RepositoryRoot
		return m, func() tea.Msg { return run(client, runner, root) }
	case "/resolve":
		var run func(agent.API, agent.Runner, string) agent.Result
		progress := make(chan string, 1)
		m.brief, run = m.brief.Resolve(m.ctx, progress)
		if m.brief.Err != nil {
			m.err = m.brief.Err.Error()
		}
		if run == nil {
			return m, nil
		}
		return m.launchBriefOperation(run, progress)
	case "/retry":
		if m.brief.Busy {
			return m, nil
		}
		if m.brief.CommittedID != 0 {
			return m.beginBrief(agent.Refresh)
		}
		if m.brief.FailedStep != "" {
			return m.beginBrief(m.brief.FailedStep)
		}
		return m.beginBrief(agent.Preflight)
	case "/reload":
		if m.brief.Busy {
			return m, nil
		}
		return m.beginBrief(agent.Preflight)
	case "/return":
		paths := append([]string(nil), m.brief.ScratchPaths...)
		retain := m.brief.Phase == agent.Failed
		m.brief = m.brief.Invalidate()
		m.briefOperation = nil
		returnState := m.briefReturn
		m.briefField = ""
		m = m.setPromptValue("")
		next, cmd := m.arrive(returnState, "returned from brief workflow")
		if retain || len(paths) == 0 {
			return next, cmd
		}
		root := m.appConfig.RepositoryRoot
		cleanup := func() tea.Msg {
			for _, path := range paths {
				rel, err := filepath.Rel(filepath.Join(root, ".mch", "tmp"), path)
				if err != nil || filepath.Dir(rel) != "." {
					continue
				}
				if err := briefprocess.CleanupOwned(root, path); err != nil {
					return briefCleanupMsg{err: err}
				}
			}
			return briefCleanupMsg{}
		}
		return next, tea.Batch(cmd, cleanup)
	}
	return m, nil
}

func (m Model) briefKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.Type == tea.KeyPgDown {
		m.briefOffset += max(1, m.height-8)
		return m, nil
	}
	if msg.Type == tea.KeyPgUp {
		m.briefOffset = max(0, m.briefOffset-max(1, m.height-8))
		return m, nil
	}
	if msg.String() == "/" && m.input.Value() == "" {
		m.openCommandDropdown()
		return m, nil
	}
	if msg.Type == tea.KeyEsc || msg.Type == tea.KeyCtrlC {
		if m.input.Value() != "" {
			m = m.setPromptValue("")
			m.status = "prompt cleared"
			return m, nil
		}
		return m.briefCommand("/return")
	}
	if m.brief.Busy {
		return m, nil
	}
	if msg.Type == tea.KeyCtrlE {
		brief := m.brief.Draft
		if m.briefField == "brief" && m.promptValue() != "" {
			brief = m.promptValue()
		}
		return m.openTextEditor(BriefState, brief)
	}
	if msg.Type == tea.KeyEnter {
		value := m.promptValue()
		if commandAllowed(BriefState, strings.TrimSpace(value)) {
			m = m.setPromptValue("")
			return m.briefCommand(strings.TrimSpace(value))
		}
		switch m.briefField {
		case "title":
			m.brief = m.brief.EditIdentity(value, m.brief.UUID)
		case "uuid":
			m.brief = m.brief.EditIdentity(m.brief.Title, value)
		case "brief":
			m.brief = m.brief.EditBrief(value)
		case "answer":
			id, answer, ok := strings.Cut(value, ":")
			if !ok {
				m.err = "answer format: question ID: answer"
				return m, nil
			}
			m.brief = m.brief.Answer(strings.TrimSpace(id), strings.TrimSpace(answer))
		}
		m = m.setPromptValue("")
		m.status = "draft retained; use /confirm, /approve or /resolve"
		if m.brief.Err != nil {
			m.err = m.brief.Err.Error()
		}
		return m, nil
	}
	if msg.Type == tea.KeyCtrlA && m.brief.Phase == agent.Answering {
		m.briefField = "answer"
		m = m.setPromptValue("")
		return m, nil
	}
	return m.updatePromptInput(msg)
}

func (m Model) briefView(width int) string {
	line := func(label, value string) string {
		value = documents.SafeText(value)
		value = strings.ReplaceAll(value, "\n", " ↵ ")
		return label + ": " + value
	}
	parts := []string{fmt.Sprintf("Project #%d  Change #%d  Phase: %s", m.brief.ProjectID, m.brief.ChangeID, m.brief.Phase), line("Title", m.brief.Title), line("Optional UUID", m.brief.UUID), line("Original user brief", m.brief.Original), line("Current draft", m.brief.Draft), line("Backend current brief", m.brief.BackendBrief), fmt.Sprintf("Revision %d  document #%d  committed #%d", m.brief.Revision, m.brief.DocumentID, m.brief.CommittedID)}
	if m.brief.Candidate.RewrittenBrief != "" {
		parts = append(parts, line("Agent proposed rewrite", m.brief.Candidate.RewrittenBrief))
	}
	for _, q := range m.brief.Questions {
		parts = append(parts, line("Question "+q.ID+" context", q.Context), line("Question "+q.ID, q.Text), line("Answer "+q.ID, q.Answer))
		if m.brief.NeedsAnswer[q.ID] {
			parts = append(parts, "Answer needed for "+q.ID)
		}
	}
	for _, q := range m.brief.Candidate.Questions {
		parts = append(parts, line("Proposed question "+q.ID+" context", q.Context), line("Proposed question "+q.ID, q.Text))
	}
	if len(m.brief.Candidate.Unresolved) > 0 {
		parts = append(parts, "Proposed unresolved: "+strings.Join(m.brief.Candidate.Unresolved, ", "))
	}
	if len(m.brief.Unresolved) > 0 {
		parts = append(parts, "Unresolved: "+strings.Join(m.brief.Unresolved, ", "))
	}
	if m.brief.ScratchPath != "" {
		parts = append(parts, line("Operation files", m.brief.ScratchPath))
	}
	var rows []string
	for _, part := range parts {
		runes := []rune(part)
		for len(runes) > width {
			rows = append(rows, string(runes[:width]))
			runes = runes[width:]
		}
		rows = append(rows, string(runes))
	}
	page := max(1, m.height-9)
	start := min(m.briefOffset, max(0, len(rows)-page))
	return strings.Join(rows[start:min(len(rows), start+page)], "\n") + "\nInput: " + m.briefField + "  (Ctrl+A to answer as ID: text)"
}
