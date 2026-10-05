// Package documents owns configured document access independently of agent workflows.
package documents

import (
	"cli/internal/dto"
	"context"
	"fmt"
	"slices"
	"strings"
)

// API exposes individual document operations.
type API interface {
	ActiveDocuments(context.Context, int, string) ([]dto.Document, error)
	InsertDocument(context.Context, dto.DocumentInput) (int, error)
	ListComments(context.Context, int, string) ([]dto.Document, error)
}

// ScreenAPI adds history, details and configuration to ordinary document access.
type ScreenAPI interface {
	API
	ListDocuments(context.Context, int, string) ([]dto.Document, error)
	DocumentDetails(context.Context, int) (dto.Document, error)
	GetProjectConfig(context.Context, int) (dto.ProjectConfig, error)
}

// Access binds ordinary change editors to the selected project's document catalog.
type Access struct {
	API   API
	Types []string
}

// Load reads all current versions without manufacturing absent documents.
func (a Access) Load(ctx context.Context, id int) ([]dto.Document, error) {
	rows, err := a.API.ActiveDocuments(ctx, id, "change")
	if err != nil {
		return nil, err
	}
	if err = ValidateActive(rows, id, "change"); err != nil {
		return nil, err
	}
	comments, err := a.API.ListComments(ctx, id, "change")
	if err != nil {
		return nil, err
	}
	return append(rows, comments...), nil
}

// Save validates a configured type and appends one human-edited version.
func (a Access) Save(ctx context.Context, id int, kind, body string) (int, error) {
	if !slices.Contains(a.Types, kind) {
		return 0, fmt.Errorf("change document type %q is not configured for this project", kind)
	}
	if strings.TrimSpace(body) == "" {
		return 0, fmt.Errorf("%s is required", kind)
	}
	return a.API.InsertDocument(ctx, dto.DocumentInput{RefID: id, RefTable: "change", DocType: kind, Body: body, AgentEdit: false})
}
