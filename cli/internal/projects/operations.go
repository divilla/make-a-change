package projects

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

// Supported project operations. Retries use only List, Details or Config.
const (
	List    Operation = "list"
	Details Operation = "details"
	Config  Operation = "config"
	Create  Operation = "create"
	Edit    Operation = "edit"
	Delete  Operation = "delete"
)

// Result binds work to its generation, entity and operation.
type Result struct {
	Generation uint64
	Operation  Operation
	ID         int
	Project    dto.Project
	Rows       []dto.Project
	Config     dto.ProjectConfig
	Err        error
	RefreshErr error
	Committed  bool
}

// Begin validates forms and schedules effects outside the event loop.
func (m Model) Begin(ctx context.Context, api API, op Operation, id int, name string) (Model, tea.Cmd) {
	if m.Busy {
		return m, nil
	}
	if op == Create || op == Edit {
		m.Draft = name
	}
	if (op == Create || op == Edit) && strings.TrimSpace(name) == "" {
		m.Err = errors.New("project name is required")
		m.Status = "validation failed"
		return m, nil
	}
	if op != List && op != Create && id <= 0 {
		m.Err = errors.New("project ID must be a valid positive number")
		m.Status = "validation failed"
		return m, nil
	}
	m = m.Invalidate()
	m.Operation = op
	m.EntityID = id
	m.Err = nil
	m.Status = "loading project"
	m.Loading = true
	if op == Details || op == List {
		m.DetailLoaded = false
	}
	if op == List {
		m.Rows = nil
		m.Selected = 0
		m.Status = "loading projects"
	}
	if op == Config {
		m.Catalog = dto.ProjectConfig{}
		m.ShowConfig = true
	}
	if op == Create || op == Edit || op == Delete {
		m.Busy = true
		m.Status = "saving"
	}
	if op == Delete {
		m.Status = "deleting project"
	}
	ctx, cancel := context.WithCancel(ctx)
	m.cancel = cancel
	generation := m.Generation
	prior := m.Detail
	return m, func() tea.Msg {
		defer cancel()
		r := Result{Generation: generation, Operation: op, ID: id}
		switch op {
		case List:
			r.Rows, r.Err = api.ListProjectRows(ctx)
		case Details:
			r.Project, r.Err = api.GetProject(ctx, id)
		case Config:
			r.Config, r.Err = api.GetProjectConfig(ctx, id)
		case Create:
			r.ID, r.Err = api.CreateProject(ctx, name)
			if r.Err == nil {
				r.Committed = true
				r.Project = dto.Project{ID: r.ID, Name: name}
				var p dto.Project
				p, r.RefreshErr = api.GetProject(ctx, r.ID)
				if r.RefreshErr == nil {
					r.Project = p
				}
			}
		case Edit:
			r.Err = api.UpdateProject(ctx, id, name)
			if r.Err == nil {
				r.Committed = true
				r.Project = prior
				r.Project.ID = id
				r.Project.Name = name
				var p dto.Project
				p, r.RefreshErr = api.GetProject(ctx, id)
				if r.RefreshErr == nil {
					r.Project = p
				}
			}
		case Delete:
			r.Err = api.DeleteProject(ctx, id)
			if r.Err == nil {
				r.Committed = true
				r.Rows, r.RefreshErr = api.ListProjectRows(ctx)
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
	m.ShowConfig = false
	return m
}

// Apply accepts only the active entity/operation and retains committed mutations.
func (m Model) Apply(r Result) (Model, bool) {
	if r.Generation != m.Generation || r.Operation != m.Operation || (r.Operation != Create && r.ID != m.EntityID) {
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
		return m, true
	}
	switch r.Operation {
	case List:
		m.Rows = r.Rows
		m.Selected = 0
		m.Status = "loaded projects"
		if len(r.Rows) == 0 {
			m.Status = "no projects"
		}
	case Details:
		m.Detail = r.Project
		m.DetailLoaded = true
		m.Status = "loaded project"
	case Config:
		m.Catalog = r.Config
		m.Status = "loaded project configuration"
	case Create, Edit:
		m.Detail = r.Project
		m.DetailLoaded = r.RefreshErr == nil
		m.Draft = ""
		m.Status = "saved project"
	case Delete:
		m.Detail = dto.Project{}
		m.DetailLoaded = false
		m.Rows = r.Rows
		m.Selected = 0
		m.Status = "deleted project"
	}
	if r.RefreshErr != nil {
		m.Err = r.RefreshErr
		m.Status += "; refresh failed — /retry reads only"
	}
	return m, true
}

// Options converts typed projects at the presentation boundary.
func Options(rows []dto.Project) []dto.Option {
	options := make([]dto.Option, 0, len(rows))
	for _, p := range rows {
		options = append(options, dto.Option{ID: strconv.Itoa(p.ID), Label: p.Name})
	}
	return options
}

// CatalogOptions preserves backend order and only supplied colors.
func CatalogOptions(cfg dto.ProjectConfig) ([]dto.Option, []dto.Option) {
	phases := make([]dto.Option, 0, len(cfg.ChangePhases))
	types := make([]dto.Option, 0, len(cfg.ChangeTypes))
	for i, s := range cfg.ChangePhases {
		o := dto.Option{ID: s, Label: s}
		if i < len(cfg.ChangeColors) {
			o.Color = cfg.ChangeColors[i]
		}
		phases = append(phases, o)
	}
	for _, s := range cfg.ChangeTypes {
		types = append(types, dto.Option{ID: s, Label: s})
	}
	return phases, types
}
