package documents

import (
	"cli/internal/dto"
	"context"
	"errors"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// MutationAPI is the document feature's independent comment and selection capability.
type MutationAPI interface {
	ScreenAPI
	InsertComment(context.Context, int, string, string) (int, error)
	UpdateComment(context.Context, int, string) error
	DeleteDocument(context.Context, int) error
	ActivateDocument(context.Context, int) error
	UndeleteComment(context.Context, int) error
}

// Mutation is one ID-preserving operation, or an independent comment insert.
type Mutation string

// Document mutations retain IDs; Read never repeats a committed write.
const (
	Read           Mutation = "read"
	NewComment     Mutation = "comment-insert"
	EditComment    Mutation = "comment-update"
	DeleteDocument Mutation = "delete"
	Activate       Mutation = "active-set"
	Undelete       Mutation = "comment-undelete"
)

// ChangeModel owns change document/comment mutations and read-only recovery.
type ChangeModel struct {
	ProjectID, OwnerID int
	Revision           uint64
	Busy               bool
	Draft              string
	Committed          string
	Err                error
	cancel             context.CancelFunc
}

// ChangeResult binds exact write outcomes and refreshed data to a scope and revision.
type ChangeResult struct {
	ProjectID, OwnerID int
	Revision           uint64
	Operation          Mutation
	Active, Comments   []dto.Document
	ID                 int
	Committed          string
	Err, RefreshErr    error
}

// Invalidate cancels pending work and makes its results obsolete.
func (m ChangeModel) Invalidate() ChangeModel {
	if m.cancel != nil {
		m.cancel()
	}
	m.cancel = nil
	m.Revision++
	m.Busy = false
	return m
}

// Begin performs at most one mutation, followed by independent current reads.
func (m ChangeModel) Begin(ctx context.Context, api MutationAPI, project, owner int, op Mutation, id int, body string) (ChangeModel, tea.Cmd) {
	if m.Busy {
		return m, nil
	}
	if project <= 0 || owner <= 0 || (op != NewComment && op != Read && id <= 0) {
		m.Err = errors.New("invalid document mutation scope")
		return m, nil
	}
	if op == NewComment && strings.TrimSpace(body) == "" {
		m.Err = errors.New("comment body is required")
		m.Draft = body
		return m, nil
	}
	if project != m.ProjectID || owner != m.OwnerID {
		m = m.Invalidate()
		m.Committed = ""
	}
	m = m.Invalidate()
	m.ProjectID, m.OwnerID = project, owner
	m.Busy, m.Err = true, nil
	if op != Read {
		m.Draft = body
		m.Committed = ""
	}
	work, cancel := context.WithCancel(ctx)
	m.cancel = cancel
	revision, committed := m.Revision, m.Committed
	return m, func() tea.Msg {
		defer cancel()
		r := ChangeResult{ProjectID: project, OwnerID: owner, Revision: revision, Operation: op, ID: id, Committed: committed}
		if err := work.Err(); err != nil {
			r.Err = err
			return r
		}
		switch op {
		case NewComment:
			r.ID, r.Err = api.InsertComment(work, owner, "change", body)
		case EditComment:
			r.Err = api.UpdateComment(work, id, body)
		case DeleteDocument:
			r.Err = api.DeleteDocument(work, id)
		case Activate:
			r.Err = api.ActivateDocument(work, id)
		case Undelete:
			r.Err = api.UndeleteComment(work, id)
		case Read:
		default:
			r.Err = errors.New("unsupported document mutation")
		}
		if r.Err != nil {
			return r
		}
		if op != Read {
			r.Committed = fmt.Sprintf("committed %s document #%d", op, r.ID)
		}
		r.Active, r.RefreshErr = api.ActiveDocuments(work, owner, "change")
		if r.RefreshErr == nil {
			r.RefreshErr = ValidateActive(r.Active, owner, "change")
		}
		if r.RefreshErr == nil {
			r.Comments, r.RefreshErr = api.ListComments(work, owner, "change")
		}
		return r
	}
}

// Apply rejects late or duplicate results and retains committed feedback.
func (m ChangeModel) Apply(r ChangeResult) (ChangeModel, bool) {
	if !m.Busy || r.Revision != m.Revision || r.ProjectID != m.ProjectID || r.OwnerID != m.OwnerID {
		return m, false
	}
	m = m.Invalidate()
	m.Err = r.Err
	if r.Err == nil {
		m.Committed = r.Committed
		if r.Operation != Read {
			m.Draft = ""
		}
		m.Err = r.RefreshErr
	}
	return m, true
}

// ValidateActive enforces owner scope and a unique live non-comment per type.
func ValidateActive(rows []dto.Document, owner int, table string) error {
	seen := map[string]bool{}
	for _, d := range rows {
		if d.ID <= 0 || d.RefID != owner || d.RefTable != table || d.DeletedAt != nil || strings.TrimSpace(d.DocType) == "" || d.DocType == "comment" || seen[d.DocType] {
			return errors.New("invalid owner or duplicate active document type")
		}
		seen[d.DocType] = true
	}
	return nil
}
