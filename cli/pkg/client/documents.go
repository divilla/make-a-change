package client

import (
	"cli/internal/dto"
	"context"
	"errors"
	"strings"
	"time"
)

type documentWire struct {
	ID        *int                    `json:"id"`
	RefID     *int                    `json:"ref_id"`
	RefTable  *string                 `json:"ref_table"`
	DocType   *string                 `json:"doc_type"`
	Body      *string                 `json:"body"`
	AgentEdit *bool                   `json:"agent_edit"`
	DeletedAt nullableWire[time.Time] `json:"deleted_at"`
	CreatedAt *time.Time              `json:"created_at"`
	UpdatedAt *time.Time              `json:"updated_at"`
	HTML      *string                 `json:"html"`
}

func validDocumentOwner(id int, table string) error {
	if id <= 0 || (table != "project" && table != "epic" && table != "change") {
		return errors.New("document owner must have a positive ID and project, epic, or change table")
	}
	return nil
}

func (w documentWire) value(path string, id int, table string) (dto.Document, error) {
	if w.ID == nil || *w.ID <= 0 || w.RefID == nil || *w.RefID <= 0 || w.RefTable == nil || (*w.RefTable != "project" && *w.RefTable != "epic" && *w.RefTable != "change") || w.DocType == nil || strings.TrimSpace(*w.DocType) == "" || w.Body == nil || w.AgentEdit == nil || !w.DeletedAt.Present || w.CreatedAt == nil || w.UpdatedAt == nil || w.CreatedAt.IsZero() || w.UpdatedAt.IsZero() || w.UpdatedAt.Before(*w.CreatedAt) || w.HTML == nil {
		return dto.Document{}, contractStatus(path, &ContractError{errors.New("missing or invalid document fields")})
	}
	if (id > 0 && *w.ID != id) || (table != "" && *w.RefTable != table) {
		return dto.Document{}, contractStatus(path, &ContractError{errors.New("document identity or owner differs from request")})
	}
	return dto.Document{ID: *w.ID, RefID: *w.RefID, RefTable: *w.RefTable, DocType: *w.DocType, Body: *w.Body, AgentEdit: *w.AgentEdit, DeletedAt: w.DeletedAt.Value, CreatedAt: *w.CreatedAt, UpdatedAt: *w.UpdatedAt, HTML: *w.HTML}, nil
}

func (c HTTPClient) documentRows(ctx context.Context, path string, id int, table string, current bool) ([]dto.Document, error) {
	if err := validDocumentOwner(id, table); err != nil {
		return nil, err
	}
	var wire []documentWire
	if err := c.projectRequest(ctx, path, struct {
		RefID    int    `json:"ref_id"`
		RefTable string `json:"ref_table"`
	}{id, table}, 200, &wire); err != nil {
		return nil, err
	}
	if wire == nil {
		return nil, contractStatus(path, &ContractError{errors.New("expected document array")})
	}
	rows := make([]dto.Document, 0, len(wire))
	previous := 0
	seen := map[string]bool{}
	for i, w := range wire {
		row, err := w.value(path, 0, table)
		if err != nil {
			return nil, err
		}
		if row.RefID != id || (i > 0 && row.ID >= previous) || (current && (row.DeletedAt != nil || row.DocType == "comment")) {
			return nil, contractStatus(path, &ContractError{errors.New("invalid document owner, order, or active row")})
		}
		if current && seen[row.DocType] {
			return nil, contractStatus(path, &ContractError{errors.New("duplicate active document type")})
		}
		seen[row.DocType] = true
		previous = row.ID
		rows = append(rows, row)
	}
	return rows, nil
}

// ListDocuments reads every retained version for one owner.
func (c HTTPClient) ListDocuments(ctx context.Context, id int, table string) ([]dto.Document, error) {
	return c.documentRows(ctx, "/api/v1/doc/list", id, table, false)
}

// ActiveDocuments reads every current type for an owner in one operation.
func (c HTTPClient) ActiveDocuments(ctx context.Context, id int, table string) ([]dto.Document, error) {
	return c.documentRows(ctx, "/api/v1/doc/list-active", id, table, true)
}

// DocumentDetails reads a single version by document ID.
func (c HTTPClient) DocumentDetails(ctx context.Context, id int) (dto.Document, error) {
	const path = "/api/v1/doc/details"
	if id <= 0 {
		return dto.Document{}, errors.New("document ID must be positive")
	}
	var wire documentWire
	if err := c.projectRequest(ctx, path, struct {
		ID int `json:"id"`
	}{id}, 200, &wire); err != nil {
		return dto.Document{}, err
	}
	return wire.value(path, id, "")
}

// InsertDocument appends exactly one version, returning the committed ID.
func (c HTTPClient) InsertDocument(ctx context.Context, input dto.DocumentInput) (int, error) {
	if err := validDocumentOwner(input.RefID, input.RefTable); err != nil {
		return 0, err
	}
	if strings.TrimSpace(input.DocType) == "" || input.DocType == "comment" || strings.TrimSpace(input.Body) == "" {
		return 0, errors.New("document type and body are required")
	}
	return c.insertID(ctx, "/api/v1/doc/insert", input)
}

// ListComments reads retained independent comments.
func (c HTTPClient) ListComments(ctx context.Context, id int, table string) ([]dto.Document, error) {
	rows, err := c.documentRows(ctx, "/api/v1/doc/comment-list", id, table, false)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		if row.DocType != "comment" {
			return nil, contractStatus("/api/v1/doc/comment-list", &ContractError{errors.New("expected comment")})
		}
	}
	return rows, nil
}

// InsertComment creates one independent human comment.
func (c HTTPClient) InsertComment(ctx context.Context, id int, table, body string) (int, error) {
	if err := validDocumentOwner(id, table); err != nil {
		return 0, err
	}
	if strings.TrimSpace(body) == "" {
		return 0, errors.New("comment body is required")
	}
	return c.insertID(ctx, "/api/v1/doc/comment-insert", struct {
		RefID     int    `json:"ref_id"`
		RefTable  string `json:"ref_table"`
		Body      string `json:"body"`
		AgentEdit bool   `json:"agent_edit"`
	}{id, table, body, false})
}

// UpdateComment edits the same ID, preserving an explicit empty body.
func (c HTTPClient) UpdateComment(ctx context.Context, id int, body string) error {
	if id <= 0 {
		return errors.New("comment ID must be positive")
	}
	return c.projectRequest(ctx, "/api/v1/doc/comment-update", struct {
		ID   int    `json:"id"`
		Body string `json:"body"`
	}{id, body}, 204, nil)
}

// DeleteDocument soft-deletes a document or comment by ID.
func (c HTTPClient) DeleteDocument(ctx context.Context, id int) error {
	return c.documentMutation(ctx, "/api/v1/doc/delete", id)
}

// ActivateDocument selects the same historical non-comment ID.
func (c HTTPClient) ActivateDocument(ctx context.Context, id int) error {
	return c.documentMutation(ctx, "/api/v1/doc/active-set", id)
}

// UndeleteComment restores the same retained comment ID.
func (c HTTPClient) UndeleteComment(ctx context.Context, id int) error {
	return c.documentMutation(ctx, "/api/v1/doc/comment-undelete", id)
}

func (c HTTPClient) documentMutation(ctx context.Context, path string, id int) error {
	if id <= 0 {
		return errors.New("document ID must be positive")
	}
	return c.projectRequest(ctx, path, projectID{id}, 204, nil)
}

// ListTestCases is the separate read needed by change details.
func (c HTTPClient) ListTestCases(ctx context.Context, id int) ([]dto.TestCase, error) {
	const path = "/api/v1/test-case/list"
	if id <= 0 {
		return nil, errors.New("change ID must be positive")
	}
	var wire []struct {
		ID        *int       `json:"id"`
		ChangeID  *int       `json:"change_id"`
		Scenario  *string    `json:"scenario"`
		Done      *bool      `json:"done"`
		CreatedAt *time.Time `json:"created_at"`
		UpdatedAt *time.Time `json:"updated_at"`
	}
	if err := c.projectRequest(ctx, path, struct {
		ChangeID int `json:"change_id"`
	}{id}, 200, &wire); err != nil {
		return nil, err
	}
	if wire == nil {
		return nil, contractStatus(path, &ContractError{errors.New("expected testcase array")})
	}
	rows := make([]dto.TestCase, 0, len(wire))
	previous := 0
	for _, w := range wire {
		if w.ID == nil || *w.ID <= previous || w.ChangeID == nil || *w.ChangeID != id || w.Scenario == nil || strings.TrimSpace(*w.Scenario) == "" || w.Done == nil || w.CreatedAt == nil || w.UpdatedAt == nil || w.CreatedAt.IsZero() || w.UpdatedAt.IsZero() || w.UpdatedAt.Before(*w.CreatedAt) {
			return nil, contractStatus(path, &ContractError{errors.New("invalid testcase fields")})
		}
		previous = *w.ID
		rows = append(rows, dto.TestCase{ID: *w.ID, ChangeID: *w.ChangeID, Scenario: *w.Scenario, Done: *w.Done, CreatedAt: *w.CreatedAt, UpdatedAt: *w.UpdatedAt})
	}
	return rows, nil
}
