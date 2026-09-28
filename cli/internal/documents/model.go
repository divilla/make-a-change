package documents

import (
	"cli/internal/dto"
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// Operation identifies a read or append in the document feature.
type Operation string

const (
	// Open loads an owner and its configured document types.
	Open Operation = "open"
	// Refresh reads history and current rows without writing.
	Refresh Operation = "refresh"
	// Details reads one document version by ID.
	Details Operation = "details"
	// Insert appends one version before refreshing reads.
	Insert Operation = "insert"
)

// Result carries one scoped, ordered document operation back to the shell.
type Result struct {
	Revision                    uint64
	ProjectID, OwnerID          int
	OwnerTable                  string
	Operation                   Operation
	Rows, Current               []dto.Document
	Detail                      dto.Document
	Types                       []string
	ID                          int
	Err, CatalogErr, RefreshErr error
}

// Model owns the document history, form, revision and operation lifetime.
type Model struct {
	ProjectID, OwnerID   int
	OwnerTable           string
	Types                []string
	Rows, Current        []dto.Document
	Selected             int
	Offset               int
	Detail               dto.Document
	DetailLoaded         bool
	ShowingDetail        bool
	Loaded, Busy         bool
	Revision             uint64
	DraftType, DraftBody string
	DraftAgentEdit       bool
	CommittedID          int
	Status               string
	Err, CatalogErr      error
	cancel               context.CancelFunc
}

func validScope(projectID, ownerID int, table string) error {
	if projectID <= 0 || ownerID <= 0 || !slices.Contains([]string{"project", "epic", "change"}, table) {
		return errors.New("document project, owner and table must be valid")
	}
	if table == "project" && ownerID != projectID {
		return errors.New("project document owner differs from selected project")
	}
	return nil
}

// Invalidate cancels an owned request and rejects all pending results.
func (m Model) Invalidate() Model {
	if m.cancel != nil {
		m.cancel()
	}
	m.cancel = nil
	m.Revision++
	m.Busy = false
	m.Loaded = false
	m.DetailLoaded = false
	m.Rows = nil
	m.Current = nil
	return m
}

// BeginOpen sets the exact owner scope and loads its catalog and history.
func (m Model) BeginOpen(ctx context.Context, api ScreenAPI, projectID, ownerID int, table string) (Model, tea.Cmd) {
	if err := validScope(projectID, ownerID, table); err != nil {
		m.Err, m.Status = err, "invalid document owner"
		return m, nil
	}
	m = m.Invalidate()
	m.ProjectID, m.OwnerID, m.OwnerTable = projectID, ownerID, table
	m.Types, m.DraftType, m.DraftBody = nil, "", ""
	m.DraftAgentEdit = false
	m.Selected, m.Offset, m.Detail, m.ShowingDetail = 0, 0, dto.Document{}, false
	m.CommittedID = 0
	return m.begin(ctx, api, Open, dto.DocumentInput{}, 0)
}

// BeginRefresh performs reads only, including after a committed insert.
func (m Model) BeginRefresh(ctx context.Context, api ScreenAPI) (Model, tea.Cmd) {
	return m.begin(ctx, api, Refresh, dto.DocumentInput{}, 0)
}

// BeginDetails fetches the selected version by document ID.
func (m Model) BeginDetails(ctx context.Context, api ScreenAPI) (Model, tea.Cmd) {
	if !m.Loaded || m.Busy || m.Selected < 0 || m.Selected >= len(m.Rows) {
		m.Err = errors.New("load document history before selecting a version")
		return m, nil
	}
	return m.begin(ctx, api, Details, dto.DocumentInput{}, m.Rows[m.Selected].ID)
}

// SetType chooses a configured type without modifying the stored value.
func (m Model) SetType(kind string) Model {
	if !slices.Contains(m.Types, kind) || strings.TrimSpace(kind) == "" {
		m.Err = fmt.Errorf("document type %q is not configured", kind)
		return m
	}
	m.DraftType, m.Err = kind, nil
	return m
}

// NextType cycles through the selected owner's configured catalog.
func (m Model) NextType() Model {
	if len(m.Types) == 0 {
		m.Err = errors.New("no document types configured for this owner")
		return m
	}
	i := slices.Index(m.Types, m.DraftType)
	return m.SetType(m.Types[(i+1)%len(m.Types)])
}

// BeginInsert validates the draft and sends exact editor bytes and explicit provenance.
func (m Model) BeginInsert(ctx context.Context, api ScreenAPI, body string, agentEdit bool) (Model, tea.Cmd) {
	if m.Busy {
		return m, nil
	}
	m.DraftBody, m.DraftAgentEdit = body, agentEdit
	if err := validScope(m.ProjectID, m.OwnerID, m.OwnerTable); err != nil {
		m.Err = err
		return m, nil
	}
	if !slices.Contains(m.Types, m.DraftType) || strings.TrimSpace(m.DraftType) == "" {
		m.Err = errors.New("select a configured document type")
		return m, nil
	}
	if strings.TrimSpace(body) == "" {
		m.Err = errors.New("document body is required")
		return m, nil
	}
	input := dto.DocumentInput{RefID: m.OwnerID, RefTable: m.OwnerTable, DocType: m.DraftType, Body: body, AgentEdit: agentEdit}
	return m.begin(ctx, api, Insert, input, 0)
}

func (m Model) begin(ctx context.Context, api ScreenAPI, op Operation, input dto.DocumentInput, id int) (Model, tea.Cmd) {
	if m.Busy {
		return m, nil
	}
	if err := validScope(m.ProjectID, m.OwnerID, m.OwnerTable); err != nil {
		m.Err = err
		return m, nil
	}
	if m.cancel != nil {
		m.cancel()
	}
	work, cancel := context.WithCancel(ctx)
	m.cancel = cancel
	m.Revision++
	m.Busy, m.Loaded, m.DetailLoaded = true, false, false
	m.Err = nil
	m.Status = "loading documents"
	if op == Refresh && m.CommittedID > 0 {
		m.Status = fmt.Sprintf("saved document #%d; refreshing documents", m.CommittedID)
	}
	revision, projectID, ownerID, table, committedID := m.Revision, m.ProjectID, m.OwnerID, m.OwnerTable, m.CommittedID
	return m, func() tea.Msg {
		r := Result{Revision: revision, ProjectID: projectID, OwnerID: ownerID, OwnerTable: table, Operation: op}
		if err := work.Err(); err != nil {
			r.Err = err
			return r
		}
		if op == Open || op == Refresh {
			cfg, err := api.GetProjectConfig(work, projectID)
			r.CatalogErr = err
			if err == nil {
				switch table {
				case "project":
					r.Types = cfg.ProjectDocs
				case "epic":
					r.Types = cfg.EpicDocs
				case "change":
					r.Types = cfg.ChangeDocs
				}
			}
		}
		if op == Details {
			r.Detail, r.Err = api.DocumentDetails(work, id)
			if r.Err == nil && (r.Detail.ID != id || r.Detail.RefID != ownerID || r.Detail.RefTable != table) {
				r.Err = errors.New("document details belong to a different owner or version")
			}
			return r
		}
		if op == Insert {
			if err := work.Err(); err != nil {
				r.Err = err
				return r
			}
			r.ID, r.Err = api.InsertDocument(work, input)
			if r.Err != nil {
				return r
			}
			return r
		}
		r.Rows, r.RefreshErr = api.ListDocuments(work, ownerID, table)
		if r.RefreshErr != nil {
			return r
		}
		r.Current, r.RefreshErr = api.CurrentDocuments(work, ownerID, table)
		if r.RefreshErr != nil {
			return r
		}
		if op == Refresh && committedID > 0 {
			found := false
			for _, row := range r.Rows {
				if row.ID == committedID {
					found = true
					break
				}
			}
			if !found {
				r.RefreshErr = errors.New("committed document missing from refreshed history")
				return r
			}
			r.Detail, r.RefreshErr = api.DocumentDetails(work, committedID)
			if r.RefreshErr == nil && (r.Detail.ID != committedID || r.Detail.RefID != ownerID || r.Detail.RefTable != table) {
				r.RefreshErr = errors.New("refreshed document details belong to a different owner or version")
			}
		}
		return r
	}
}

// Apply rejects stale scope/revisions and records a commit before refresh feedback.
func (m Model) Apply(r Result) (Model, bool) {
	if !m.Busy || m.Revision != r.Revision || m.ProjectID != r.ProjectID || m.OwnerID != r.OwnerID || m.OwnerTable != r.OwnerTable {
		return m, false
	}
	m.Busy = false
	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}
	if r.Operation == Open || r.Operation == Refresh {
		m.Types, m.CatalogErr = r.Types, r.CatalogErr
		if !slices.Contains(m.Types, m.DraftType) {
			m.DraftType = ""
		}
		if m.DraftType == "" && len(m.Types) > 0 {
			m.DraftType = m.Types[0]
		}
	}
	if r.Err != nil {
		m.Err, m.Status = r.Err, "document operation failed"
		return m, true
	}
	if r.Operation == Details {
		m.Detail, m.ShowingDetail, m.DetailLoaded, m.Loaded = r.Detail, true, true, true
		m.Offset = 0
		m.Status = fmt.Sprintf("document #%d", r.Detail.ID)
		return m, true
	}
	if r.Operation == Insert {
		if r.ID <= 0 {
			m.Err = errors.New("document insert returned no committed ID")
			m.Status = "document operation failed"
			return m, true
		}
		m.CommittedID = r.ID
		m.DraftBody = ""
		m.ShowingDetail = false
		m.Status = fmt.Sprintf("saved document #%d; refreshing documents", r.ID)
		return m, true
	}
	if r.RefreshErr != nil {
		m.Err = r.RefreshErr
		if m.CommittedID > 0 {
			m.Status = fmt.Sprintf("saved document #%d; refresh failed; /retry reads only", m.CommittedID)
		} else {
			m.Status = "document refresh failed; /retry reads only"
		}
		return m, true
	}
	selectedID := m.CommittedID
	if selectedID == 0 && m.Selected >= 0 && m.Selected < len(m.Rows) {
		selectedID = m.Rows[m.Selected].ID
	}
	m.Rows, m.Current = r.Rows, r.Current
	if r.Detail.ID > 0 {
		m.Detail = r.Detail
		m.DetailLoaded = true
	}
	m.ShowingDetail = false
	m.Offset = 0
	m.Selected = 0
	for i, row := range m.Rows {
		if row.ID == selectedID {
			m.Selected = i
			break
		}
	}
	m.Loaded, m.Err = true, nil
	if r.Operation != Insert && m.CommittedID == 0 {
		m.Status = "loaded documents"
	}
	if m.CommittedID > 0 {
		m.Status = fmt.Sprintf("saved document #%d; refreshed documents", m.CommittedID)
		m.CommittedID = 0
	}
	return m, true
}

// Move changes history selection within visible loaded rows.
func (m Model) Move(delta int) Model {
	if !m.Loaded || m.Busy || m.ShowingDetail {
		return m
	}
	if len(m.Rows) > 0 {
		m.Selected = max(0, min(len(m.Rows)-1, m.Selected+delta))
	}
	return m
}

// KeepSelectedVisible positions the history viewport around its selected row.
func (m Model) KeepSelectedVisible(height int) Model {
	if height <= 0 || !m.Loaded || m.ShowingDetail || len(m.Rows) == 0 {
		return m
	}
	rowLine := m.SelectedRowLine(1)
	if rowLine < m.Offset {
		m.Offset = rowLine
	}
	if rowLine >= m.Offset+height {
		m.Offset = rowLine - height + 1
	}
	return m
}

// Back returns from version detail to history.
func (m Model) Back() Model { m.ShowingDetail = false; m.DetailLoaded = false; return m }

// Scroll moves the details viewport while keeping the document bytes immutable.
func (m Model) Scroll(delta, width, height int) Model {
	lines := viewLines(m, width)
	m.Offset = max(0, min(max(0, len(lines)-height), m.Offset+delta))
	return m
}
