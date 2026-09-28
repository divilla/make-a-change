package client

import (
	"cli/internal/dto"
	"context"
	"errors"
	"strconv"
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

// CurrentDocuments reads every current type for an owner in one operation.
func (c HTTPClient) CurrentDocuments(ctx context.Context, id int, table string) ([]dto.Document, error) {
	const path = "/api/v1/doc/current"
	if id <= 0 {
		return nil, errors.New("document owner ID must be positive")
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
	docs := make([]dto.Document, 0, len(wire))
	for _, w := range wire {
		if w.ID == nil || *w.ID <= 0 || w.RefID == nil || *w.RefID != id || w.RefTable == nil || *w.RefTable != table || w.DocType == nil || w.Body == nil || w.AgentEdit == nil || w.Current == nil || !*w.Current || w.CreatedAt == nil || w.UpdatedAt == nil || w.HTML == nil {
			return nil, contractStatus(path, &ContractError{errors.New("missing or invalid current document fields")})
		}
		docs = append(docs, dto.Document{ID: *w.ID, RefID: *w.RefID, RefTable: *w.RefTable, DocType: *w.DocType, Body: *w.Body, AgentEdit: *w.AgentEdit, Current: *w.Current, CreatedAt: *w.CreatedAt, UpdatedAt: *w.UpdatedAt, HTML: *w.HTML})
	}
	return docs, nil
}

// InsertDocument appends exactly one version, returning the committed ID.
func (c HTTPClient) InsertDocument(ctx context.Context, input dto.DocumentInput) (int, error) {
	if input.RefID <= 0 {
		return 0, errors.New("document owner ID must be positive")
	}
	return c.insertID(ctx, "/api/v1/doc/insert", input)
}

// ListTestCases is the separate read needed by change details; mutation migration is P5.
func (c HTTPClient) ListTestCases(ctx context.Context, id int) ([]dto.TestCase, error) {
	const path = "/api/v1/test-case/list"
	if id <= 0 {
		return nil, errors.New("change ID must be positive")
	}
	var wire []struct {
		ID       *int    `json:"id"`
		ChangeID *int    `json:"change_id"`
		Scenario *string `json:"scenario"`
		Done     *bool   `json:"done"`
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
	for _, w := range wire {
		if w.ID == nil || *w.ID <= 0 || w.ChangeID == nil || *w.ChangeID != id || w.Scenario == nil || w.Done == nil {
			return nil, contractStatus(path, &ContractError{errors.New("invalid testcase fields")})
		}
		rows = append(rows, dto.TestCase{ID: strconv.Itoa(*w.ID), ChangeID: strconv.Itoa(*w.ChangeID), Scenario: *w.Scenario, Done: *w.Done})
	}
	return rows, nil
}
