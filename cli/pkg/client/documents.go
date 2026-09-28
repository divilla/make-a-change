package client

import (
	"cli/internal/dto"
	"context"
	"errors"
	"strings"
	"time"
)

type documentWire struct {
	ID        *int       `json:"id"`
	RefID     *int       `json:"ref_id"`
	RefTable  *string    `json:"ref_table"`
	DocType   *string    `json:"doc_type"`
	Body      *string    `json:"body"`
	AgentEdit *bool      `json:"agent_edit"`
	Current   *bool      `json:"current"`
	CreatedAt *time.Time `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`
	HTML      *string    `json:"html"`
}

func validDocumentOwner(id int, table string) error {
	if id <= 0 || (table != "project" && table != "epic" && table != "change") {
		return errors.New("document owner must have a positive ID and project, epic, or change table")
	}
	return nil
}

func (w documentWire) value(path string, id int, table string) (dto.Document, error) {
	if w.ID == nil || *w.ID <= 0 || w.RefID == nil || *w.RefID <= 0 || w.RefTable == nil || (*w.RefTable != "project" && *w.RefTable != "epic" && *w.RefTable != "change") || w.DocType == nil || strings.TrimSpace(*w.DocType) == "" || w.Body == nil || w.AgentEdit == nil || w.Current == nil || w.CreatedAt == nil || w.UpdatedAt == nil || w.CreatedAt.IsZero() || w.UpdatedAt.IsZero() || w.UpdatedAt.Before(*w.CreatedAt) || w.HTML == nil {
		return dto.Document{}, contractStatus(path, &ContractError{errors.New("missing or invalid document fields")})
	}
	if (id > 0 && *w.ID != id) || (table != "" && *w.RefTable != table) {
		return dto.Document{}, contractStatus(path, &ContractError{errors.New("document identity or owner differs from request")})
	}
	return dto.Document{ID: *w.ID, RefID: *w.RefID, RefTable: *w.RefTable, DocType: *w.DocType, Body: *w.Body, AgentEdit: *w.AgentEdit, Current: *w.Current, CreatedAt: *w.CreatedAt, UpdatedAt: *w.UpdatedAt, HTML: *w.HTML}, nil
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
	for i, w := range wire {
		row, err := w.value(path, 0, table)
		if err != nil {
			return nil, err
		}
		if row.RefID != id || (i > 0 && row.ID >= previous) || (current && !row.Current) {
			return nil, contractStatus(path, &ContractError{errors.New("invalid document owner, order, or current flag")})
		}
		previous = row.ID
		rows = append(rows, row)
	}
	return rows, nil
}

// ListDocuments reads every retained version for one owner.
func (c HTTPClient) ListDocuments(ctx context.Context, id int, table string) ([]dto.Document, error) {
	return c.documentRows(ctx, "/api/v1/doc/list", id, table, false)
}

// CurrentDocuments reads every current type for an owner in one operation.
func (c HTTPClient) CurrentDocuments(ctx context.Context, id int, table string) ([]dto.Document, error) {
	return c.documentRows(ctx, "/api/v1/doc/current", id, table, true)
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
	if strings.TrimSpace(input.DocType) == "" || strings.TrimSpace(input.Body) == "" {
		return 0, errors.New("document type and body are required")
	}
	return c.insertID(ctx, "/api/v1/doc/insert", input)
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
