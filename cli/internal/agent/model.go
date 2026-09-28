// Package agent owns the fixed brief clarification workflow.
package agent

import (
	"cli/internal/dto"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"unicode"
)

// API is the small backend capability required by brief clarification.
type API interface {
	GetProjectConfig(context.Context, int) (dto.ProjectConfig, error)
	GetChange(context.Context, int) (dto.Change, error)
	CreateChange(context.Context, dto.ChangeCreateInput) (int, error)
	CurrentDocuments(context.Context, int, string) ([]dto.Document, error)
	InsertDocument(context.Context, dto.DocumentInput) (int, error)
}

// Runner produces one structured clarification result for explicit input paths.
type Runner interface {
	Run(context.Context, Request) (Output, string, error)
}

// Question carries the affected brief context and the user's answer.
type Question = dto.BriefQuestion

// Output is the only accepted agent result for one input revision.
type Output = dto.BriefOutput

// Request fixes all runner inputs and the output location to one operation.
type Request = dto.BriefRequest

// Phase is the controller-owned workflow phase.
type Phase string

// Workflow phases are controller-owned and never backend change phases.
const (
	Draft      Phase = "draft"
	Loading    Phase = "loading"
	Saving     Phase = "saving"
	Clarifying Phase = "clarifying"
	Review     Phase = "review"
	Answering  Phase = "answering"
	Ready      Phase = "brief ready for spec writing"
	Failed     Phase = "failed"
)

// Step identifies one asynchronous backend or runner effect.
type Step string

// Steps each perform one cancelable workflow operation.
const (
	Preflight Step = "preflight"
	Write     Step = "write"
	Refresh   Step = "refresh"
	Run       Step = "run"
	Verify    Step = "verify"
)

// Result is a typed, scope-bound asynchronous response.
type Result struct {
	Generation uint64
	Revision   uint64
	ProjectID  int
	ChangeID   int
	Step       Step
	Config     dto.ProjectConfig
	Detail     dto.Change
	Documents  []dto.Document
	ID         int
	Output     Output
	Path       string
	Err        error
}

// Model owns all mutable session data; backend calls never mutate it directly.
type Model struct {
	ProjectID         int
	ChangeID          int
	DocumentID        int
	CommittedID       int
	Title             string
	UUID              string
	Original          string
	Draft             string
	BackendBrief      string
	Revision          uint64
	Generation        uint64
	Phase             Phase
	Pending           Step
	FailedStep        Step
	Questions         []Question
	Unresolved        []string
	NeedsAnswer       map[string]bool
	Candidate         Output
	Status            string
	Err               error
	ScratchPath       string
	ScratchPaths      []string
	New               bool
	Busy              bool
	cancel            context.CancelFunc
	writeAgent        bool
	committedChange   bool
	committedRevision uint64

	originalFromBackend bool
}

// New starts an unsaved new-change brief in a selected project.
func New(projectID int) Model {
	return Model{ProjectID: projectID, New: true, Phase: Draft, Revision: 1}
}

// Existing starts with a change identity that must be verified by Details.
func Existing(projectID, changeID int) Model {
	return Model{ProjectID: projectID, ChangeID: changeID, Phase: Draft, Revision: 1}
}

// Invalidate cancels owned work and prevents late results from applying.
func (m Model) Invalidate() Model {
	if m.cancel != nil {
		m.cancel()
	}
	m.cancel = nil
	m.Generation++
	m.Busy = false
	m.Pending = ""
	return m
}

// EditBrief retains exact bytes and invalidates earlier readiness.
func (m Model) EditBrief(brief string) Model {
	if brief == m.Draft {
		return m
	}
	m = m.Invalidate()
	m.Draft = brief
	if m.Original == "" || m.originalFromBackend {
		m.Original = brief
		m.originalFromBackend = false
	}
	m.writeAgent = false
	m.Revision++
	m.Candidate = Output{}
	m.Phase = Draft
	m.Err = nil
	m.FailedStep = ""
	return m
}

// EditIdentity changes the new-change title and optional UUID before creation.
func (m Model) EditIdentity(title, uuid string) Model {
	if m.ChangeID != 0 {
		return m
	}
	m.Title, m.UUID = title, uuid
	return m
}

// Begin checks preconditions and schedules the next independent effect.
func (m Model) Begin(ctx context.Context, step Step, progress ...chan<- string) (Model, func(API, Runner, string) Result) {
	if m.Busy {
		return m, nil
	}
	if m.ProjectID <= 0 || (!m.New && m.ChangeID <= 0) {
		m.Err = errors.New("selected project or change identity is unavailable")
		m.Phase = Failed
		return m, nil
	}
	if step == Write && m.CommittedID != 0 {
		step = Refresh
	}
	if step == Write && strings.TrimSpace(m.Draft) == "" {
		m.Err = errors.New("brief is required")
		m.Phase = Draft
		return m, nil
	}
	if step == Write && m.New && m.ChangeID == 0 && strings.TrimSpace(m.Title) == "" {
		m.Err = errors.New("title is required")
		m.Phase = Draft
		return m, nil
	}
	if step == Write && m.ChangeID != 0 && strings.TrimSpace(m.Draft) == strings.TrimSpace(m.BackendBrief) {
		step = Run
	}
	if step == Run && m.ChangeID == 0 {
		m.Err = errors.New("save the brief before agent clarification")
		m.Phase = Draft
		return m, nil
	}
	m = m.Invalidate()
	ctx, cancel := context.WithCancel(ctx)
	m.cancel = cancel
	m.Pending, m.Busy, m.Err = step, true, nil
	m.FailedStep = ""
	switch step {
	case Preflight, Refresh, Verify:
		m.Phase = Loading
	case Write:
		m.Phase = Saving
	case Run:
		m.Phase = Clarifying
	}
	generation, revision, project, change := m.Generation, m.Revision, m.ProjectID, m.ChangeID
	input := dto.ChangeCreateInput{ProjectID: project, RefUUID: m.UUID, Title: m.Title, Brief: m.Draft}
	brief, agentEdit := m.Draft, m.writeAgent
	previousBrief, previousID := m.BackendBrief, m.DocumentID
	original := m.Original
	committedID, committedChange := m.CommittedID, m.committedChange
	questions := append([]Question(nil), m.Questions...)
	prompt := "brief-rewrite"
	if len(questions) > 0 {
		prompt = "brief-resolve"
	}
	return m, func(api API, runner Runner, root string) Result {
		defer cancel()
		r := Result{Generation: generation, Revision: revision, ProjectID: project, ChangeID: change, Step: step}
		switch step {
		case Preflight:
			r.Config, r.Err = api.GetProjectConfig(ctx, project)
			if r.Err != nil {
				return r
			}
			if !slices.Contains(r.Config.ChangeDocs, "brief") || !slices.Contains(r.Config.ChangeDocs, "spec") {
				r.Err = errors.New("selected project requires configured brief and spec document types")
				return r
			}
			if change == 0 {
				if !slices.Contains(r.Config.ChangePhases, "backlog") {
					r.Err = errors.New("selected project requires backlog phase for new change")
				}
				return r
			}
			r.Detail, r.Err = api.GetChange(ctx, change)
			if r.Err != nil {
				return r
			}
			if r.Detail.ID != change || r.Detail.ProjectID != project {
				r.Err = errors.New("change details do not match selected project")
				return r
			}
			r.Documents, r.Err = api.CurrentDocuments(ctx, change, "change")
			if r.Err == nil {
				_, _, r.Err = currentBrief(r.Documents, change)
			}
		case Write:
			config, err := api.GetProjectConfig(ctx, project)
			if err != nil {
				r.Err = err
				return r
			}
			if !slices.Contains(config.ChangeDocs, "brief") || !slices.Contains(config.ChangeDocs, "spec") {
				r.Err = errors.New("selected project requires configured brief and spec document types")
				return r
			}
			if change == 0 {
				if !slices.Contains(config.ChangePhases, "backlog") {
					r.Err = errors.New("selected project requires backlog phase for new change")
					return r
				}
			} else {
				detail, err := api.GetChange(ctx, change)
				if err != nil {
					r.Err = err
					return r
				}
				if detail.ID != change || detail.ProjectID != project {
					r.Err = errors.New("change details do not match selected project")
					return r
				}
				rows, err := api.CurrentDocuments(ctx, change, "change")
				if err != nil {
					r.Err = err
					return r
				}
				current, id, err := currentBrief(rows, change)
				if err != nil {
					r.Err = err
					return r
				}
				if id != previousID || current != previousBrief {
					r.Err = errors.New("current brief changed; reload before writing")
					return r
				}
			}
			if change == 0 {
				r.ID, r.Err = api.CreateChange(ctx, input)
			} else {
				r.ID, r.Err = api.InsertDocument(ctx, dto.DocumentInput{RefID: change, RefTable: "change", DocType: "brief", Body: brief, AgentEdit: agentEdit})
			}
			if r.Err == nil && r.ID <= 0 {
				r.Err = errors.New("backend returned no committed ID")
			}
		case Refresh:
			r.Documents, r.Err = api.CurrentDocuments(ctx, change, "change")
			if r.Err == nil {
				current, id, err := currentBrief(r.Documents, change)
				r.Err = err
				if r.Err == nil && id == 0 {
					r.Err = errors.New("committed brief missing from current documents")
				}
				if r.Err == nil && committedChange && current != strings.TrimSpace(brief) {
					r.Err = errors.New("created brief is not the current document; reload before agent clarification")
				}
				if r.Err == nil && committedID != 0 && !committedChange && id != committedID {
					r.Err = errors.New("committed brief is not the current document")
				}
			}
		case Run, Verify:
			config, err := api.GetProjectConfig(ctx, project)
			if err != nil {
				r.Err = err
				return r
			}
			if !slices.Contains(config.ChangeDocs, "brief") || !slices.Contains(config.ChangeDocs, "spec") {
				r.Err = errors.New("selected project requires configured brief and spec document types")
				return r
			}
			detail, err := api.GetChange(ctx, change)
			if err != nil {
				r.Err = err
				return r
			}
			if detail.ID != change || detail.ProjectID != project {
				r.Err = errors.New("change details do not match selected project")
				return r
			}
			rows, err := api.CurrentDocuments(ctx, change, "change")
			if err != nil {
				r.Err = err
				return r
			}
			current, id, err := currentBrief(rows, change)
			if err != nil {
				r.Err = err
				return r
			}
			if id != previousID || current != previousBrief {
				r.Err = errors.New("current brief changed; reload before agent clarification")
				return r
			}
			if step == Verify {
				return r
			}
			if runner == nil {
				r.Err = errors.New("brief agent runner is unavailable")
				return r
			}
			request := Request{Root: root, Prompt: prompt, Revision: revision, ProjectID: project, ChangeID: change, DocumentID: previousID, Original: original, Brief: brief, Questions: questions}
			var dir string
			dir, r.Err = allocateBriefPaths(&request)
			if r.Err != nil {
				r.Path = dir
				return r
			}
			if len(progress) > 0 {
				request.Progress = progress[0]
			}
			var returnedPath string
			r.Output, returnedPath, r.Err = runner.Run(ctx, request)
			r.Path = dir
			if r.Err == nil && returnedPath != dir {
				r.Err = errors.New("agent returned an output location outside its operation directory")
			}
			if r.Err == nil {
				r.Err = validateOutput(r.Output, revision, questions)
			}
		}
		return r
	}
}

func currentBrief(rows []dto.Document, change int) (string, int, error) {
	brief, id := "", 0
	seen := map[string]bool{}
	for _, row := range rows {
		if row.RefID != change || row.RefTable != "change" || !row.Current || row.ID <= 0 || strings.TrimSpace(row.DocType) == "" {
			return "", 0, errors.New("current document owner or status conflicts with selected change")
		}
		if seen[row.DocType] {
			return "", 0, fmt.Errorf("conflicting current %s versions", row.DocType)
		}
		seen[row.DocType] = true
		if row.DocType == "brief" {
			if strings.TrimSpace(row.Body) == "" {
				return "", 0, errors.New("current brief body is empty")
			}
			brief, id = row.Body, row.ID
		}
	}
	return brief, id, nil
}

// allocateBriefPaths gives every runner, including an injected one, the same
// controller-owned operation directory and fixed file names.
func allocateBriefPaths(req *Request) (string, error) {
	owner := filepath.Join(req.Root, ".mch")
	info, err := os.Lstat(owner)
	if err != nil {
		return "", err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("workflow resource path %s is not a directory", owner)
	}
	base := filepath.Join(owner, "tmp")
	if info, err := os.Lstat(base); err == nil {
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("scratch path %s is not an owned directory", base)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	if err := os.MkdirAll(base, 0o700); err != nil {
		return "", err
	}
	dir, err := os.MkdirTemp(base, "brief-")
	if err != nil {
		return "", err
	}
	req.OriginalPath = filepath.Join(dir, "original.md")
	req.InputPath = filepath.Join(dir, "brief.md")
	req.ContextPath = filepath.Join(dir, "context.json")
	req.QuestionsPath = filepath.Join(dir, "questions.json")
	req.AnswersPath = filepath.Join(dir, "answers.json")
	req.OutputPath = filepath.Join(dir, "result.json")
	return dir, nil
}

func validateOutput(o Output, revision uint64, previous []Question) error {
	if o.InputRevision != revision || strings.TrimSpace(o.RewrittenBrief) == "" {
		return errors.New("agent output is empty or has a mismatched input revision")
	}
	if o.Questions == nil || o.Unresolved == nil {
		return errors.New("agent questions and unresolved must be arrays")
	}
	seen := map[string]bool{}
	for _, q := range o.Questions {
		if q.ID == "" || strings.TrimSpace(q.ID) != q.ID || strings.ContainsRune(q.ID, ':') || strings.IndexFunc(q.ID, unicode.IsControl) >= 0 || strings.TrimSpace(q.Text) == "" || strings.TrimSpace(q.Context) == "" || q.Answer != "" || seen[q.ID] {
			return errors.New("agent output has duplicate or incomplete question")
		}
		seen[q.ID] = true
	}
	unresolved := map[string]bool{}
	for _, id := range o.Unresolved {
		if unresolved[id] {
			return fmt.Errorf("duplicate unresolved question %q", id)
		}
		unresolved[id] = true
		if !seen[id] {
			return fmt.Errorf("unresolved question %q is missing from output", id)
		}
	}
	for _, q := range o.Questions {
		answered := false
		for _, old := range previous {
			if old.ID == q.ID && strings.TrimSpace(old.Answer) != "" {
				answered = true
			}
		}
		if !answered && !slices.Contains(o.Unresolved, q.ID) {
			return fmt.Errorf("unanswered question %q is not marked unresolved", q.ID)
		}
	}
	if o.ReadyForSpec && len(o.Unresolved) != 0 {
		return errors.New("agent output contradicts readiness with unresolved blockers")
	}
	if !o.ReadyForSpec && len(o.Unresolved) == 0 {
		return errors.New("agent output is not ready but has no unresolved blockers")
	}
	for _, q := range previous {
		if strings.TrimSpace(q.Answer) == "" && !seen[q.ID] {
			return fmt.Errorf("unanswered question %q cannot be silently discarded", q.ID)
		}
		if strings.TrimSpace(q.Answer) == "" && o.ReadyForSpec {
			return fmt.Errorf("unanswered question %q cannot be silently resolved", q.ID)
		}
	}
	return nil
}

// Apply consumes only the result for this selected scope and generation.
func (m Model) Apply(r Result) (Model, Step, bool) {
	if !m.Busy || r.Generation != m.Generation || r.Revision != m.Revision || r.ProjectID != m.ProjectID || r.ChangeID != m.ChangeID || r.Step != m.Pending {
		return m, "", false
	}
	m.Busy, m.Pending, m.cancel = false, "", nil
	if r.Path != "" {
		m.ScratchPath = r.Path
		if !slices.Contains(m.ScratchPaths, r.Path) {
			m.ScratchPaths = append(m.ScratchPaths, r.Path)
		}
	}
	if r.Step == Write && r.Err == nil {
		m.CommittedID = r.ID
		m.committedRevision = r.Revision
		if m.ChangeID == 0 {
			m.ChangeID = r.ID
			m.New = false
			m.committedChange = true
		} else {
			m.DocumentID = r.ID
			m.committedChange = false
		}
		m.Status = fmt.Sprintf("committed ID %d", r.ID)
	}
	if r.Err != nil {
		m.Err = r.Err
		m.Phase = Failed
		m.FailedStep = r.Step
		if m.CommittedID != 0 {
			m.Status = fmt.Sprintf("committed ID %d; retry read or agent step", m.CommittedID)
		}
		return m, "", true
	}
	switch r.Step {
	case Preflight:
		if m.ChangeID != 0 {
			brief, id, _ := currentBrief(r.Documents, m.ChangeID)
			m.BackendBrief, m.DocumentID = brief, id
			if m.Draft == "" {
				m.Draft, m.Original = brief, brief
				m.originalFromBackend = brief != ""
			}
		}
		m.Phase = Draft
		m.Status = "brief ready for editing; confirm to continue"
	case Write:
		return m, Refresh, true
	case Refresh:
		m.BackendBrief, m.DocumentID, _ = currentBrief(r.Documents, m.ChangeID)
		m.CommittedID = 0
		m.committedChange = false
		if m.committedRevision != m.Revision {
			m.committedRevision = 0
			m.Phase = Draft
			m.Status = "committed brief refreshed; edited draft needs confirmation"
			return m, "", true
		}
		m.committedRevision = 0
		if m.Candidate.RewrittenBrief != "" && m.Draft == m.Candidate.RewrittenBrief {
			return m.finishCandidate(), "", true
		}
		return m, Run, true
	case Run:
		m.Candidate = r.Output
		m.Phase = Review
		m.Status = "review rewrite and questions; approve or edit"
	case Verify:
		return m.finishCandidate(), "", true
	}
	return m, "", true
}

func (m Model) finishCandidate() Model {
	previous := m.Questions
	m.Questions = append([]Question(nil), m.Candidate.Questions...)
	for i := range m.Questions {
		m.Questions[i].Answer = ""
		for _, old := range previous {
			if old.ID == m.Questions[i].ID {
				m.Questions[i].Answer = old.Answer
				break
			}
		}
	}
	for _, old := range previous {
		found := false
		for _, current := range m.Questions {
			if current.ID == old.ID {
				found = true
				break
			}
		}
		if !found && strings.TrimSpace(old.Answer) != "" {
			m.Questions = append(m.Questions, old)
		}
	}
	m.Unresolved = append([]string(nil), m.Candidate.Unresolved...)
	m.NeedsAnswer = make(map[string]bool, len(m.Unresolved))
	for _, id := range m.Unresolved {
		m.NeedsAnswer[id] = true
	}
	if m.Candidate.ReadyForSpec && len(m.Unresolved) == 0 {
		m.Phase = Ready
	} else {
		m.Phase = Answering
	}
	m.Status = string(m.Phase)
	m.Candidate = Output{}
	return m
}

// Approve accepts the current agent draft, appending a new agent version if changed.
func (m Model) Approve(ctx context.Context) (Model, func(API, Runner, string) Result) {
	if m.Phase != Review || m.Candidate.InputRevision != m.Revision {
		return m, nil
	}
	if strings.TrimSpace(m.Candidate.RewrittenBrief) == strings.TrimSpace(m.BackendBrief) {
		return m.Begin(ctx, Verify)
	}
	m.Draft = m.Candidate.RewrittenBrief
	m.writeAgent = true
	return m.Begin(ctx, Write)
}

// Answer records a nonblank answer for a known question ID.
func (m Model) Answer(id, answer string) Model {
	if strings.TrimSpace(answer) == "" {
		m.Err = errors.New("answer cannot be empty")
		return m
	}
	for i := range m.Questions {
		if m.Questions[i].ID == id {
			m = m.Invalidate()
			m.Questions[i].Answer = answer
			if m.NeedsAnswer != nil {
				m.NeedsAnswer[id] = false
			}
			m.Revision++
			m.Phase = Answering
			m.Err = nil
			return m
		}
	}
	m.Err = fmt.Errorf("unknown question ID %q", id)
	return m
}

// Resolve invokes the answer-aware prompt while preserving all earlier answers.
func (m Model) Resolve(ctx context.Context, progress ...chan<- string) (Model, func(API, Runner, string) Result) {
	if m.Phase != Answering {
		return m, nil
	}
	for _, id := range m.Unresolved {
		if m.NeedsAnswer[id] {
			m.Err = fmt.Errorf("unanswered blocker %q", id)
			return m, nil
		}
	}
	return m.Begin(ctx, Run, progress...)
}
