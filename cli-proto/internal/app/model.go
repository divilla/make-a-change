package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type screen int

const (
	screenProjectSelect screen = iota
	screenReady
	screenIdea
	screenRunning
	screenPlanning
	screenConfirm
	screenSaving
	screenDone
)

type model struct {
	repoRoot   string
	promptPath string
	configPath string
	cfg        Config
	api        *APIClient
	codex      CodexRunner
	editor     Editor

	projects       []Project
	projectCursor  int
	currentProject *Project

	input        textinput.Model
	spinner      spinner.Model
	width        int
	menuOpen     bool
	menuIndex    int
	screen       screen
	elapsed      int
	output       string
	sessionID    string
	errText      string
	finalReq     FinalRequirement
	editedBody   string
	createdTitle string
}

type codexMsg struct {
	result CodexResult
	err    error
}

type savePreparedMsg struct {
	req  FinalRequirement
	body string
	err  error
}

type createMsg struct {
	change Change
	err    error
}

type elapsedMsg time.Time

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("86"))
	metaStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))
	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("203"))
	loadingStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))
	outputStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1)
	promptBarStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("240")).
			Foreground(lipgloss.Color("252"))
	menuStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")).
			PaddingLeft(1)
	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("86")).
			Bold(true)
)

var slashCommands = []string{"change-new", "cancel"}

func newModel(repoRoot, configPath string, cfg Config, api *APIClient, projects []Project, projectErr error, codex CodexRunner, editor Editor) model {
	input := textinput.New()
	input.Focus()
	input.CharLimit = 4000
	input.Width = 0
	input.Placeholder = "/change-new or /cancel"
	input.PromptStyle = promptBarStyle.Copy().Foreground(lipgloss.Color("183"))
	input.TextStyle = promptBarStyle.Copy().Foreground(lipgloss.Color("252"))
	input.PlaceholderStyle = promptBarStyle.Copy().Foreground(lipgloss.Color("238"))
	input.Cursor.Style = promptBarStyle.Copy().Foreground(lipgloss.Color("252"))
	input.Cursor.TextStyle = input.TextStyle

	spin := spinner.New(
		spinner.WithSpinner(spinner.MiniDot),
		spinner.WithStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("86"))),
	)

	m := model{
		repoRoot:   repoRoot,
		promptPath: filepath.Join(repoRoot, "agent", "prompts", "build-requirement-with-agent.md"),
		configPath: configPath,
		cfg:        cfg,
		api:        api,
		codex:      codex,
		editor:     editor,
		projects:   projects,
		input:      input,
		spinner:    spin,
		width:      100,
		screen:     screenReady,
	}
	if projectErr != nil {
		m.errText = "backend unavailable: " + projectErr.Error()
		return m
	}
	if len(projects) == 0 {
		m.errText = "no projects exist"
		return m
	}
	if cfg.CurrentProjectID != nil {
		for i, project := range projects {
			if project.ID == *cfg.CurrentProjectID {
				m.currentProject = &m.projects[i]
				return m
			}
		}
	}
	m.screen = screenProjectSelect
	return m
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil
	case spinner.TickMsg:
		if m.screen != screenRunning {
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case elapsedMsg:
		if m.screen != screenRunning {
			return m, nil
		}
		m.elapsed++
		return m, elapsedTick()
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		switch m.screen {
		case screenProjectSelect:
			return m.updateProjectSelect(msg)
		case screenConfirm:
			return m.updateConfirm(msg)
		case screenReady, screenIdea, screenPlanning:
			if m.menuOpen {
				return m.updateSlashMenu(msg)
			}
			if msg.String() == "/" && m.input.Value() == "" && m.input.Position() == 0 {
				m.input.SetValue("/")
				m.input.CursorEnd()
				m.menuOpen = true
				m.menuIndex = 0
				return m, nil
			}
			if msg.String() == "enter" {
				return m.submitInput()
			}
		}
	case codexMsg:
		if msg.err != nil {
			m.screen = screenPlanning
			if m.sessionID == "" {
				m.screen = screenReady
			}
			m.errText = msg.err.Error()
			m.output = strings.TrimSpace(msg.result.Output)
			return m, nil
		}
		m.screen = screenPlanning
		m.elapsed = 0
		m.errText = ""
		m.sessionID = msg.result.SessionID
		m.output = strings.TrimSpace(msg.result.Output)
		m.input.Placeholder = "Refine, /save, or /cancel"
		return m, nil
	case savePreparedMsg:
		if msg.err != nil {
			m.screen = screenPlanning
			m.errText = msg.err.Error()
			return m, nil
		}
		m.finalReq = msg.req
		m.editedBody = msg.body
		m.screen = screenConfirm
		m.errText = ""
		return m, nil
	case createMsg:
		if msg.err != nil {
			m.screen = screenPlanning
			m.errText = msg.err.Error()
			return m, nil
		}
		m.screen = screenDone
		m.createdTitle = msg.change.Title
		return m, nil
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m model) View() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("mch"))
	b.WriteString("\n\n")
	if m.errText != "" {
		b.WriteString(errorStyle.Render("Error: " + m.errText))
		b.WriteString("\n\n")
	}
	if m.currentProject != nil {
		b.WriteString(metaStyle.Render(fmt.Sprintf("Project: %s", m.currentProject.Name)))
		b.WriteString("\n")
	}
	if m.sessionID != "" {
		b.WriteString(metaStyle.Render(fmt.Sprintf("Codex session: %s", m.sessionID)))
		b.WriteString("\n")
	}
	if m.output != "" {
		b.WriteString("\n")
		b.WriteString(outputStyle.Render(m.output))
		b.WriteString("\n")
	}
	switch m.screen {
	case screenProjectSelect:
		b.WriteString("Select project:\n")
		for i, project := range m.projects {
			cursor := " "
			if i == m.projectCursor {
				cursor = selectedStyle.Render(">")
			}
			b.WriteString(fmt.Sprintf("%s %s\n", cursor, project.Name))
		}
	case screenReady, screenIdea, screenPlanning:
		b.WriteString("\n" + m.promptView())
		if m.menuOpen {
			b.WriteString("\n")
			b.WriteString(m.slashMenuView())
		}
		b.WriteString("\n")
	case screenRunning:
		b.WriteString("\n" + m.loadingView() + "\n")
	case screenConfirm:
		b.WriteString(fmt.Sprintf("\nSave %q? y/n\n", m.finalReq.Title))
	case screenSaving:
		b.WriteString("\nSaving...\n")
	case screenDone:
		b.WriteString(fmt.Sprintf("\nSaved %q\n", m.createdTitle))
	}
	return b.String()
}

func (m model) loadingView() string {
	unit := "seconds"
	if m.elapsed == 1 {
		unit = "second"
	}
	return loadingStyle.Render(fmt.Sprintf("%s Waiting for Codex... %d %s", m.spinner.View(), m.elapsed, unit))
}

func (m model) promptView() string {
	width := m.width
	if width < 20 {
		width = 100
	}
	content := m.promptLine(width)
	blank := strings.Repeat(" ", width)
	return strings.Join([]string{
		promptBarStyle.Render(blank),
		promptBarStyle.Render(content),
		promptBarStyle.Render(blank),
	}, "\n")
}

func (m model) promptLine(width int) string {
	content := m.input.View()
	if visible := lipgloss.Width(content); visible < width {
		content += promptBarStyle.Render(strings.Repeat(" ", width-visible))
	}
	return content
}

func (m model) slashMenuView() string {
	var b strings.Builder
	for i, command := range slashCommands {
		prefix := "  "
		lineStyle := menuStyle
		if i == m.menuIndex {
			prefix = "> "
			lineStyle = selectedStyle.Copy().PaddingLeft(1)
		}
		b.WriteString(lineStyle.Render(prefix + command))
		if i < len(slashCommands)-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

func (m model) updateSlashMenu(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.menuIndex > 0 {
			m.menuIndex--
		}
	case "down", "j":
		if m.menuIndex < len(slashCommands)-1 {
			m.menuIndex++
		}
	case "enter":
		command := "/" + slashCommands[m.menuIndex]
		m.menuOpen = false
		m.input.SetValue("")
		return m.submitValue(command)
	case "esc", "backspace":
		m.menuOpen = false
		m.input.SetValue("")
	default:
		m.menuOpen = false
	}
	return m, nil
}

func (m model) updateProjectSelect(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.projectCursor > 0 {
			m.projectCursor--
		}
	case "down", "j":
		if m.projectCursor < len(m.projects)-1 {
			m.projectCursor++
		}
	case "enter":
		if len(m.projects) == 0 {
			return m, nil
		}
		m.currentProject = &m.projects[m.projectCursor]
		id := m.currentProject.ID
		m.cfg.CurrentProjectID = &id
		if err := saveConfig(m.configPath, m.cfg); err != nil {
			m.errText = err.Error()
			return m, nil
		}
		m.screen = screenReady
		m.input.Placeholder = "/change-new or /cancel"
	case "q":
		return m, tea.Quit
	}
	return m, nil
}

func (m model) updateConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		m.screen = screenSaving
		return m, m.createChangeCmd()
	case "n", "N", "esc":
		m.screen = screenPlanning
	}
	return m, nil
}

func (m model) submitInput() (tea.Model, tea.Cmd) {
	value := strings.TrimSpace(m.input.Value())
	m.input.SetValue("")
	m.errText = ""
	return m.submitValue(value)
}

func (m model) submitValue(value string) (tea.Model, tea.Cmd) {
	m.errText = ""
	if value == "/cancel" {
		return m, tea.Quit
	}
	if value == "/change-new" && m.screen != screenReady {
		m.errText = "/change-new is only available before a planning flow starts"
		return m, nil
	}
	switch m.screen {
	case screenReady:
		if value == "/change-new" {
			if m.currentProject == nil {
				m.errText = "select a project before starting a change"
				return m, nil
			}
			m.screen = screenIdea
			m.input.Placeholder = "Initial change idea"
			return m, nil
		}
		m.errText = "unknown slash command"
	case screenIdea:
		if value == "" {
			m.errText = "initial idea cannot be empty"
			return m, nil
		}
		template, err := os.ReadFile(m.promptPath)
		if err != nil {
			m.screen = screenReady
			m.errText = err.Error()
			return m, nil
		}
		m.screen = screenRunning
		return m.startCodex(BuildInitialPrompt(string(template), value), "")
	case screenPlanning:
		if value == "/save" {
			m.screen = screenSaving
			return m, m.prepareSaveCmd()
		}
		if strings.HasPrefix(value, "/") {
			m.errText = "unknown slash command"
			return m, nil
		}
		if value == "" {
			m.errText = "refinement prompt cannot be empty"
			return m, nil
		}
		m.screen = screenRunning
		return m.startCodex(value, m.sessionID)
	}
	return m, nil
}

func (m model) startCodex(prompt, sessionID string) (tea.Model, tea.Cmd) {
	m.elapsed = 0
	return m, tea.Batch(m.codexCmd(prompt, sessionID), m.spinner.Tick, elapsedTick())
}

func (m model) codexCmd(prompt, sessionID string) tea.Cmd {
	return func() tea.Msg {
		result, err := m.codex.Run(context.Background(), CodexRequest{
			RepoRoot:  m.repoRoot,
			Prompt:    prompt,
			SessionID: sessionID,
		})
		return codexMsg{result: result, err: err}
	}
}

func elapsedTick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return elapsedMsg(t)
	})
}

func (m model) prepareSaveCmd() tea.Cmd {
	return func() tea.Msg {
		if m.currentProject == nil {
			return savePreparedMsg{err: fmt.Errorf("select a project before saving")}
		}
		req, err := ParseRequirementMarkdown(m.output)
		if err != nil {
			return savePreparedMsg{err: err}
		}
		refs, err := m.api.ChangeReferences(context.Background())
		if err != nil {
			return savePreparedMsg{err: err}
		}
		epics, err := m.api.ListEpics(context.Background(), m.currentProject.ID)
		if err != nil {
			return savePreparedMsg{err: err}
		}
		req, err = ValidateRequirementReferences(req, refs, epics)
		if err != nil {
			return savePreparedMsg{err: err}
		}
		body, err := m.editor.Edit(req.Body)
		if err != nil {
			return savePreparedMsg{err: err}
		}
		return savePreparedMsg{req: req, body: body}
	}
}

func (m model) createChangeCmd() tea.Cmd {
	return func() tea.Msg {
		change, err := m.api.CreateChange(context.Background(), ChangeCreateInput{
			ProjectID:   m.currentProject.ID,
			EpicID:      m.finalReq.EpicID,
			Title:       m.finalReq.Title,
			Body:        m.editedBody,
			ChangePhase: "backlog",
			ChangeTypes: m.finalReq.Types,
		})
		return createMsg{change: change, err: err}
	}
}
