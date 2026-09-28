package epics

import (
	"cli/internal/dto"
	"context"
	"errors"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// Operation identifies a feature action, including read-only retries.
type Operation string

// Supported epic operations. Retries use only List or Details.
const (
	List    Operation = "list"
	Details Operation = "details"
	Create  Operation = "create"
	Edit    Operation = "edit"
	Delete  Operation = "delete"
)

// Result binds work to its generation, entity and operation.
type Result struct {
	Generation uint64
	Operation  Operation
	ID         int
	ProjectID  int
	Epic       dto.Epic
	Rows       []dto.Epic
	Err        error
	RefreshErr error
	Committed  bool
}

// Begin validates forms and schedules effects outside the event loop.
func (m Model) Begin(ctx context.Context, api API, op Operation, projectID, id int, name string) (Model, tea.Cmd) {
	if m.Busy {
		return m, nil
	}
	if projectID <= 0 {
		m.Err = errors.New("select a valid project with /select-project first")
		m.Status = "validation failed"
		return m, nil
	}
	if m.ProjectID != projectID {
		m = m.Scope(projectID)
	}
	if op != m.refreshOp || id != m.refreshID {
		m.Outcome = ""
	}
	if op == Create || op == Edit {
		m.Draft = name
	}
	if (op == Create || op == Edit) && strings.TrimSpace(name) == "" {
		m.Err = errors.New("epic name is required")
		m.Status = "validation failed"
		return m, nil
	}
	if op != List && op != Create && id <= 0 {
		m.Err = errors.New("epic ID must be a valid positive number")
		m.Status = "validation failed"
		return m, nil
	}
	if op == Edit && m.DetailLoaded && name == m.Detail.Name {
		m.Err = nil
		m.Status = "unchanged"
		return m, nil
	}
	// Only the pending refresh's read retry inherits the mutation outcome.
	outcome := m.Outcome
	m = m.Invalidate()
	m.Outcome = outcome
	m.Operation = op
	m.EntityID = id
	m.Err = nil
	if op == Details {
		m.DetailLoaded = false
	}
	m.Status = "loading epic"
	m.Loading = true
	if op == List {
		m.Rows = nil
		m.Selected = 0
		m.Status = "loading epics"
	}
	if op == Create || op == Edit || op == Delete {
		m.Outcome = ""
		m.Busy = true
		m.Status = "saving"
	}
	if op == Delete {
		m.Status = "deleting epic"
	}
	ctx, cancel := context.WithCancel(ctx)
	m.cancel = cancel
	generation := m.Generation
	prior := m.Detail
	priorRows := append([]dto.Epic(nil), m.Rows...)
	return m, func() tea.Msg {
		defer cancel()
		r := Result{Generation: generation, Operation: op, ID: id, ProjectID: projectID}
		switch op {
		case List:
			r.Rows, r.Err = api.ListEpics(ctx, projectID)
		case Details:
			r.Epic, r.Err = readEpic(ctx, api, projectID, id)
		case Create:
			r.ID, r.Err = api.CreateEpic(ctx, projectID, name)
			if r.Err == nil {
				r.Committed = true
				r.Epic = dto.Epic{ID: r.ID, ProjectID: projectID, Name: name}
				var p dto.Epic
				p, r.RefreshErr = readEpic(ctx, api, projectID, r.ID)
				if r.RefreshErr == nil {
					r.Epic = p
				}
			}
		case Edit:
			r.Err = api.UpdateEpic(ctx, id, name)
			if r.Err == nil {
				r.Committed = true
				r.Epic = prior
				r.Epic.ID = id
				r.Epic.Name = name
				var p dto.Epic
				p, r.RefreshErr = readEpic(ctx, api, projectID, id)
				if r.RefreshErr == nil {
					r.Epic = p
				}
			}
		case Delete:
			r.Err = api.DeleteEpic(ctx, id)
			if r.Err == nil {
				r.Committed = true
				r.Rows, r.RefreshErr = api.ListEpics(ctx, projectID)
				if r.RefreshErr != nil {
					for _, e := range priorRows {
						if e.ID != id {
							r.Rows = append(r.Rows, e)
						}
					}
				}
			}
		}
		return r
	}
}

// Invalidate cancels obsolete reads and prevents late results from changing state.
func (m Model) Invalidate() Model {
	if m.cancel != nil {
		m.cancel()
	}
	m.cancel = nil
	m.Generation++
	m.Loading = false
	m.Outcome = ""
	return m
}

// Apply accepts only the active entity/operation and retains committed mutations.
func (m Model) Apply(r Result) (Model, bool) {
	if r.ProjectID != m.ProjectID || r.Generation != m.Generation || r.Operation != m.Operation || (r.Operation != Create && r.ID != m.EntityID) {
		return m, false
	}
	m.Generation++
	m.Busy = false
	m.Loading = false
	m.cancel = nil
	m.Err = r.Err
	if r.Err != nil {
		m.Status = "load failed"
		if r.Operation == Create || r.Operation == Edit {
			m.Status = "save failed"
		}
		if r.Operation == Delete {
			m.Status = "delete failed"
		}
		if m.Outcome != "" {
			m.Status = m.Outcome + "; refresh failed — /retry reads only"
		}
		return m, true
	}
	switch r.Operation {
	case List:
		m.Rows = r.Rows
		m.Selected = 0
		m.Status = "loaded epics"
		if len(r.Rows) == 0 {
			m.Status = "no epics"
		}
	case Details:
		m.DetailOffset = 0
		m.Detail = r.Epic
		m.DetailLoaded = true
		m.Status = "loaded epic"
	case Create, Edit:
		m.DetailOffset = 0
		m.Detail = r.Epic
		m.DetailLoaded = r.RefreshErr == nil
		m.Draft = ""
		m.Status = "saved epic"
	case Delete:
		m.Detail = dto.Epic{}
		m.DetailLoaded = false
		m.Rows = r.Rows
		m.Selected = 0
		m.Status = "deleted epic"
	}
	m.Outcome = ""
	if r.Committed && r.RefreshErr != nil {
		m.Outcome = m.Status
		m.refreshOp, m.refreshID = Details, r.ID
		if r.Operation == Delete {
			m.refreshOp, m.refreshID = List, 0
		}
	}
	if r.RefreshErr != nil {
		m.Err = r.RefreshErr
		m.Status += "; refresh failed — /retry reads only"
	}
	return m, true
}

// Options converts typed epics at the presentation boundary.
func Options(rows []dto.Epic) []dto.Option {
	options := make([]dto.Option, 0, len(rows))
	for _, p := range rows {
		options = append(options, dto.Option{ID: strconv.Itoa(p.ID), Label: p.Name})
	}
	return options
}

// Scope clears values belonging to a previous project and invalidates its work.
func (m Model) Scope(projectID int) Model {
	m = m.Invalidate()
	return Model{Generation: m.Generation, ProjectID: projectID}
}

func readEpic(ctx context.Context, api API, projectID, id int) (dto.Epic, error) {
	e, err := api.GetEpic(ctx, id)
	if err == nil && (e.ProjectID != projectID || e.ID != id) {
		err = errors.New("epic does not belong to the selected project or entity")
	}
	return e, err
}

// Form prepares a feature-owned name draft only after successful detail loading.
func (m Model) Form(edit bool) (Model, bool) {
	if m.ProjectID <= 0 {
		m.Err = errors.New("select a valid project with /select-project first")
		return m, false
	}
	if edit && (!m.DetailLoaded || m.Loading) {
		m.Err = errors.New("load epic details with /retry before editing")
		return m, false
	}
	m = m.Invalidate()
	m.Draft = ""
	if edit {
		m.Draft = m.Detail.Name
	}
	m.Err = nil
	return m, true
}
