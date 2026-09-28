package testcases

import (
	"cli/internal/dto"
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// Operation identifies a testcase write or a read-only refresh.
type Operation string

const (
	// Create inserts a testcase for the selected change.
	Create Operation = "create"
	// Edit changes its scenario.
	Edit Operation = "edit"
	// SetDone sets its explicit completion state.
	SetDone Operation = "set-done"
	// Delete removes one testcase.
	Delete Operation = "delete"
	// Refresh retries reads without replaying a committed write.
	Refresh Operation = "refresh"
)

// Result identifies a completed operation and preserves the committed outcome.
type Result struct {
	ProjectID, ChangeID int
	Revision            uint64
	Operation           Operation
	ID                  int
	Committed           bool
	Rows                []dto.TestCase
	Change              dto.Change
	Err                 error
	RefreshErr          error
	RowsErr             error
	ChangeErr           error
}

// Model owns testcase operation state and its cancellation lifetime.
type Model struct {
	ProjectID, ChangeID int
	Revision            uint64
	Busy                bool
	Loaded              bool
	Rows                []dto.TestCase
	CommittedID         int
	committedOperation  Operation
	TargetID            string
	Draft               string
	Status              string
	Err                 error
	cancel              context.CancelFunc
}

// NeedsRefresh reports whether a committed write still needs a successful read.
func (m Model) NeedsRefresh() bool {
	return m.committedOperation != ""
}

// OpenCreate starts a new scenario form.
func (m Model) OpenCreate() Model {
	m.TargetID, m.Draft = "", ""
	return m
}

// OpenEdit retains the selected row identity and scenario for a retryable form.
func (m Model) OpenEdit(id, scenario string) Model {
	m.TargetID, m.Draft = id, scenario
	return m
}

// OpenDelete retains the row identity until confirmation.
func (m Model) OpenDelete(id string) Model {
	m.TargetID = id
	return m
}

// ClearForm discards a finished or canceled testcase form.
func (m Model) ClearForm() Model {
	m.TargetID, m.Draft = "", ""
	return m
}

// BeginTarget validates the selected form row before starting a write.
func (m Model) BeginTarget(ctx context.Context, api API, projectID, changeID int, op Operation, scenario string, done bool) (Model, tea.Cmd) {
	return m.BeginRow(ctx, api, projectID, changeID, op, m.TargetID, scenario, done)
}

// BeginRow validates a visible row identity before starting a write.
func (m Model) BeginRow(ctx context.Context, api API, projectID, changeID int, op Operation, rawID, scenario string, done bool) (Model, tea.Cmd) {
	id, err := ParseID(rawID)
	if err != nil {
		m.Err, m.Status = err, "validation failed"
		return m, nil
	}
	return m.Begin(ctx, api, projectID, changeID, op, id, scenario, done)
}

// Invalidate cancels owned work and rejects every pending result.
func (m Model) Invalidate() Model {
	if m.cancel != nil {
		m.cancel()
	}
	m.cancel = nil
	m.Revision++
	m.Busy = false
	m.CommittedID = 0
	m.committedOperation = ""
	return m
}

// Begin validates one action and starts exactly one operation.
func (m Model) Begin(ctx context.Context, api API, projectID, changeID int, op Operation, id int, scenario string, done bool) (Model, tea.Cmd) {
	if m.Busy {
		return m, nil
	}
	if projectID <= 0 || changeID <= 0 {
		m.Err = errors.New("valid project and change required")
		return m, nil
	}
	if op == Create || op == Edit {
		m.Draft = scenario
		if strings.TrimSpace(scenario) == "" {
			m.Err = errors.New("test case scenario is required")
			m.Status = "validation failed"
			return m, nil
		}
	}
	if op == Edit || op == SetDone || op == Delete {
		if id <= 0 {
			m.Err = errors.New("test case ID must be a valid positive number")
			m.Status = "validation failed"
			return m, nil
		}
	}
	committedOperation, committedID := m.committedOperation, m.CommittedID
	retainCommit := op == Refresh && m.ProjectID == projectID && m.ChangeID == changeID
	m = m.Invalidate()
	if retainCommit {
		m.committedOperation, m.CommittedID = committedOperation, committedID
	}
	m.ProjectID, m.ChangeID = projectID, changeID
	m.Busy = true
	m.Err = nil
	m.Status = "loading test cases"
	if op != Refresh {
		m.Status = "saving test case"
	}
	work, cancel := context.WithCancel(ctx)
	m.cancel = cancel
	revision := m.Revision
	return m, func() tea.Msg {
		r := Result{ProjectID: projectID, ChangeID: changeID, Revision: revision, Operation: op, ID: id}
		switch op {
		case Create:
			r.ID, r.Err = api.CreateTestCase(work, changeID, scenario)
		case Edit:
			r.Err = api.UpdateTestCase(work, id, scenario)
		case SetDone:
			r.Err = api.UpdateTestCaseDone(work, id, done)
		case Delete:
			r.Err = api.DeleteTestCase(work, id)
		case Refresh:
		default:
			r.Err = fmt.Errorf("unsupported testcase operation %q", op)
		}
		if r.Err != nil {
			return r
		}
		r.Committed = op != Refresh
		r.Rows, r.RowsErr = api.ListTestCases(work, changeID)
		r.Change, r.ChangeErr = api.GetChange(work, changeID)
		if r.ChangeErr == nil && (r.Change.ID != changeID || r.Change.ProjectID != projectID) {
			r.ChangeErr = errors.New("change does not belong to selected project/entity")
		}
		r.RefreshErr = errors.Join(r.RowsErr, r.ChangeErr)
		return r
	}
}

// Apply accepts only the current scoped result and records commit before refresh feedback.
func (m Model) Apply(r Result) (Model, bool) {
	if !m.Busy || r.Revision != m.Revision || r.ProjectID != m.ProjectID || r.ChangeID != m.ChangeID {
		return m, false
	}
	m.Busy = false
	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}
	if r.Err != nil {
		m.Err = r.Err
		m.Status = "save failed"
		return m, true
	}
	if r.Committed {
		m.committedOperation = r.Operation
		m.Status = m.committedStatus()
		m.Draft = ""
		if r.Operation == Create {
			m.CommittedID = r.ID
		}
	}
	if r.RefreshErr != nil {
		m.Err = r.RefreshErr
		m.Loaded = false
		if m.committedOperation != "" {
			m.Status = m.committedStatus() + "; refresh failed"
		} else {
			m.Status = "refresh failed"
		}
		if m.committedOperation == Create && m.CommittedID > 0 {
			m.Status += fmt.Sprintf(" (#%d)", m.CommittedID)
		}
		return m, true
	}
	m.Rows = r.Rows
	m.Loaded = true
	m.Err = nil
	m.Draft = ""
	if r.Committed {
		m.Status = m.committedStatus()
	} else if m.committedOperation != "" {
		m.Status = m.committedStatus() + "; refreshed test cases"
	} else {
		m.Status = "refreshed test cases"
	}
	if m.committedOperation == Create && m.CommittedID > 0 {
		m.Status += fmt.Sprintf(" (#%d)", m.CommittedID)
	}
	m.committedOperation = ""
	return m, true
}

func (m Model) committedStatus() string {
	if m.committedOperation == Delete {
		return "deleted test case"
	}
	return "saved test case"
}

// ParseID rejects empty, negative and out-of-range row identities.
func ParseID(value string) (int, error) {
	id, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || id <= 0 {
		return 0, errors.New("test case ID must be a valid positive number")
	}
	return id, nil
}
