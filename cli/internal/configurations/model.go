// Package configurations owns backend configuration management state and presentation.
package configurations

import (
	"cli/internal/dto"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

// API is the backend configuration contract used by this screen.
type API interface {
	ListConfigurations(context.Context) ([]dto.BackendConfig, error)
	GetConfiguration(context.Context, string) (dto.BackendConfig, error)
	InsertConfiguration(context.Context, dto.BackendConfig) (string, error)
	UpdateConfiguration(context.Context, dto.BackendConfig) error
	DeleteConfiguration(context.Context, string) error
}

// Operation identifies a single backend request.
type Operation string

// Configuration operations each correspond to one backend request.
const (
	List    Operation = "list"
	Details Operation = "details"
	Insert  Operation = "insert"
	Update  Operation = "update"
	Delete  Operation = "delete"
)

// Result belongs to one generation and one request.
type Result struct {
	Generation uint64
	Operation  Operation
	Slug       string
	Rows       []dto.BackendConfig
	Row        dto.BackendConfig
	Err        error
}

// Model owns rows, drafts, selection, and committed write feedback.
type Model struct {
	Generation         uint64
	cancel             context.CancelFunc
	Busy               bool
	Rows               []dto.BackendConfig
	Selected           int
	Detail             dto.BackendConfig
	DetailLoaded       bool
	RequestedSlug      string
	Form               bool
	Editing            bool
	Field              int
	Raw                [7]string
	Confirm            bool
	Committed          string
	CommittedSlug      string
	CommittedOperation Operation
	Refresh            Operation
	Err                error
	Status             string
	Offset             int
}

// Invalidate cancels owned work and prevents obsolete results from applying.
func (m Model) Invalidate() Model {
	if m.cancel != nil {
		m.cancel()
	}
	m.cancel = nil
	m.Generation++
	m.Busy = false
	return m
}

// Begin starts exactly one request. Reads remove cached actionable data.
func (m Model) Begin(parent context.Context, api API, op Operation, slug string, row dto.BackendConfig) (Model, tea.Cmd) {
	if m.Busy {
		return m, nil
	}
	m = m.Invalidate()
	ctx, cancel := context.WithCancel(parent)
	m.cancel = cancel
	m.Busy = true
	m.Err = nil
	switch op {
	case List:
		m.Rows = nil
		m.DetailLoaded = false
	case Details:
		m.DetailLoaded = false
		m.RequestedSlug = slug
	}
	m.Status = "loading configuration " + string(op)
	generation := m.Generation
	return m, func() tea.Msg {
		defer cancel()
		r := Result{Generation: generation, Operation: op, Slug: slug}
		switch op {
		case List:
			r.Rows, r.Err = api.ListConfigurations(ctx)
		case Details:
			r.Row, r.Err = api.GetConfiguration(ctx, slug)
		case Insert:
			r.Slug, r.Err = api.InsertConfiguration(ctx, row)
		case Update:
			r.Err = api.UpdateConfiguration(ctx, row)
		case Delete:
			r.Err = api.DeleteConfiguration(ctx, slug)
		}
		return r
	}
}

// Apply accepts only the active generation and records write commitment before refresh.
func (m Model) Apply(r Result) (Model, bool) {
	if r.Generation != m.Generation || !m.Busy {
		return m, false
	}
	m.Busy = false
	m.cancel = nil
	m.Generation++
	m.Err = r.Err
	if r.Err != nil {
		if m.Committed != "" && (r.Operation == List || r.Operation == Details) {
			m.Status = m.Committed + "; refresh failed — /retry reads only"
		} else {
			m.Status = string(r.Operation) + " failed"
		}
		return m, true
	}
	switch r.Operation {
	case List:
		m.Rows = r.Rows
		if m.Selected >= len(m.Rows) {
			m.Selected = max(0, len(m.Rows)-1)
		}
		m.Status = "loaded configurations"
		if len(m.Rows) == 0 {
			m.Status = "no backend configurations"
		}
		if m.Committed != "" {
			m.Status = m.Committed
		}
	case Details:
		m.Detail = r.Row
		m.DetailLoaded = true
		m.Status = "loaded configuration " + r.Slug
		if m.Committed != "" {
			m.Status = m.Committed
		}
	case Insert, Update, Delete:
		m.Committed = "created configuration " + r.Slug
		if r.Operation == Update {
			m.Committed = "updated configuration " + r.Slug
		}
		if r.Operation == Delete {
			m.Committed = "deleted configuration " + r.Slug
		}
		m.Status = m.Committed
		m.CommittedSlug = r.Slug
		m.CommittedOperation = r.Operation
		m.Form = false
		m.Confirm = false
		m.DetailLoaded = false
		m.Rows = nil
		m.Refresh = List
	}
	return m, true
}

// OpenForm initializes a complete editable seven-field draft.
func (m Model) OpenForm(edit bool) Model {
	m.Form, m.Editing, m.Field = true, edit, 0
	m.Confirm = false
	m.Offset = 0
	m.Committed = ""
	m.Refresh = ""
	m.Raw = [7]string{}
	if edit {
		m.Raw = Raw(m.Detail)
		m.Field = 1
	} else {
		for i := 1; i < len(m.Raw); i++ {
			m.Raw[i] = "[]"
		}
	}
	return m
}

// Raw serializes every ordered catalog without normalization.
func Raw(row dto.BackendConfig) [7]string {
	fields := [7]string{row.Slug}
	for i, values := range [][]string{row.ProjectDocs, row.EpicDocs, row.ChangeDocs, row.ChangePhases, row.ChangeColors, row.ChangeTypes} {
		data, _ := json.Marshal(values)
		fields[i+1] = string(data)
	}
	return fields
}

// ParseDraft rejects blank identities/members and omitted, null, or malformed arrays.
func (m Model) ParseDraft() (dto.BackendConfig, error) {
	row := dto.BackendConfig{Slug: m.Raw[0]}
	if strings.TrimSpace(row.Slug) == "" {
		return row, errors.New("slug is required")
	}
	fields := []*[]string{&row.ProjectDocs, &row.EpicDocs, &row.ChangeDocs, &row.ChangePhases, &row.ChangeColors, &row.ChangeTypes}
	for i, field := range fields {
		if err := json.Unmarshal([]byte(m.Raw[i+1]), field); err != nil || *field == nil {
			return row, fmt.Errorf("%s must be a JSON string array", FieldNames[i+1])
		}
		var raw []any
		if err := json.Unmarshal([]byte(m.Raw[i+1]), &raw); err != nil {
			return row, err
		}
		for _, item := range raw {
			value, ok := item.(string)
			if !ok || strings.TrimSpace(value) == "" {
				return row, fmt.Errorf("%s contains a blank or non-string member", FieldNames[i+1])
			}
		}
	}
	if m.Editing && row.Slug != m.Detail.Slug {
		return row, errors.New("configuration slug is immutable")
	}
	return row, nil
}

// FieldNames are displayed next to the editable values.
var FieldNames = [7]string{"slug", "project_docs", "epic_docs", "change_docs", "change_phases", "change_colors", "change_types"}

// KeepSelectionVisible moves the list viewport so the selected row's marker is visible.
func (m Model) KeepSelectionVisible(width, height int) Model {
	if len(m.Rows) == 0 {
		return m
	}
	if width < 20 {
		width = 80
	}
	height = max(1, height)
	selectedStart := len(strings.Split(ansi.Hardwrap("Backend configurations", width, true), "\n"))
	for i := 0; i < m.Selected; i++ {
		selectedStart += len(strings.Split(ansi.Hardwrap("  "+safeLine(m.Rows[i].Slug), width, true), "\n"))
	}
	if selectedStart < m.Offset {
		m.Offset = selectedStart
	} else if selectedStart >= m.Offset+height {
		m.Offset = selectedStart - height + 1
	}
	return m
}

// KeepFieldVisible moves the form viewport to the active field's label.
func (m Model) KeepFieldVisible(width, height int) Model {
	if !m.Form {
		return m
	}
	if width < 20 {
		width = 80
	}
	height = max(1, height)
	fieldStart := len(strings.Split(ansi.Hardwrap("Backend configurations", width, true), "\n"))
	for i := 0; i < m.Field; i++ {
		value := m.Raw[i]
		if i == 0 && m.Editing {
			value += " (fixed)"
		}
		fieldStart += len(strings.Split(ansi.Hardwrap("  "+FieldNames[i]+": "+safeLine(value), width, true), "\n"))
	}
	if fieldStart < m.Offset {
		m.Offset = fieldStart
	} else if fieldStart >= m.Offset+height {
		m.Offset = fieldStart - height + 1
	}
	return m
}

// Scroll moves the stored viewport offset within the rendered page bounds.
func (m Model) Scroll(delta, width, height int) Model {
	height = max(1, height)
	lastPage := max(0, len(wrappedLines(m, width))-height)
	current := min(max(0, m.Offset), lastPage)
	m.Offset = min(max(0, current+delta), lastPage)
	return m
}

// View renders safe, scrollable rows and the complete form.
func View(m Model, width, height int) string {
	height = max(1, height)
	wrapped := wrappedLines(m, width)
	start := min(max(0, m.Offset), max(0, len(wrapped)-height))
	end := min(len(wrapped), start+height)
	return strings.Join(wrapped[start:end], "\n")
}

func wrappedLines(m Model, width int) []string {
	if width < 20 {
		width = 80
	}
	lines := []string{"Backend configurations"}
	if m.Form {
		for i, name := range FieldNames {
			marker := "  "
			if i == m.Field {
				marker = "> "
			}
			value := m.Raw[i]
			if i == 0 && m.Editing {
				value += " (fixed)"
			}
			lines = append(lines, marker+name+": "+safeLine(value))
		}
	} else if m.Confirm {
		lines = append(lines, "Delete configuration "+safeLine(m.Detail.Slug)+"? Enter confirms; Esc cancels")
	} else {
		if m.Busy {
			lines = append(lines, "Loading…")
		}
		if len(m.Rows) == 0 && !m.Busy {
			lines = append(lines, "No configurations loaded.")
		}
		if !m.DetailLoaded {
			for i, row := range m.Rows {
				marker := "  "
				if i == m.Selected {
					marker = "> "
				}
				lines = append(lines, marker+safeLine(row.Slug))
			}
		}
		if m.DetailLoaded {
			lines = append(lines, "", "Configuration: "+safeLine(m.Detail.Slug))
			values := [][]string{m.Detail.ProjectDocs, m.Detail.EpicDocs, m.Detail.ChangeDocs, m.Detail.ChangePhases, m.Detail.ChangeColors, m.Detail.ChangeTypes}
			for i, field := range values {
				data, _ := json.Marshal(field)
				lines = append(lines, FieldNames[i+1]+": "+safeLine(string(data)))
			}
		}
	}
	wrapped := []string{}
	for _, line := range lines {
		wrapped = append(wrapped, strings.Split(ansi.Hardwrap(line, width, true), "\n")...)
	}
	return wrapped
}

func safeLine(raw string) string {
	var b strings.Builder
	for _, r := range raw {
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			quoted := strconv.QuoteRune(r)
			b.WriteString(quoted[1 : len(quoted)-1])
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}
