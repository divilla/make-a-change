package client

import (
	"cli/internal/dto"
	"context"
	"errors"
	"net/http"
	"strings"
)

func validConfig(c dto.BackendConfig) error {
	if strings.TrimSpace(c.Slug) == "" {
		return errors.New("configuration slug is required")
	}
	for _, values := range [][]string{c.ProjectDocs, c.EpicDocs, c.ChangeDocs, c.ChangePhases, c.ChangeColors, c.ChangeTypes} {
		if values == nil {
			return errors.New("all six configuration arrays are required")
		}
		for _, value := range values {
			if strings.TrimSpace(value) == "" {
				return errors.New("configuration array members must not be blank")
			}
		}
	}
	return nil
}

func (w configWire) backendValue() (dto.BackendConfig, error) {
	c := dto.BackendConfig{Slug: w.Slug, ProjectDocs: []string(w.ProjectDocs), EpicDocs: []string(w.EpicDocs), ChangeDocs: []string(w.ChangeDocs), ChangePhases: []string(w.ChangePhases), ChangeColors: []string(w.ChangeColors), ChangeTypes: []string(w.ChangeTypes)}
	if err := validConfig(c); err != nil {
		return dto.BackendConfig{}, &ContractError{err}
	}
	return c, nil
}

// ListConfigurations reads full rows in backend slug order.
func (c HTTPClient) ListConfigurations(ctx context.Context) ([]dto.BackendConfig, error) {
	const path = "/api/v1/config/list"
	var wire []configWire
	if err := c.projectRequest(ctx, path, struct{}{}, http.StatusOK, &wire); err != nil {
		return nil, err
	}
	if wire == nil {
		return nil, contractStatus(path, &ContractError{errors.New("expected configuration array")})
	}
	rows := make([]dto.BackendConfig, 0, len(wire))
	seen := make(map[string]struct{}, len(wire))
	for _, item := range wire {
		row, err := item.backendValue()
		if err != nil {
			return nil, contractStatus(path, err)
		}
		if _, duplicate := seen[row.Slug]; duplicate {
			return nil, contractStatus(path, &ContractError{errors.New("configuration slugs must be unique")})
		}
		seen[row.Slug] = struct{}{}
		rows = append(rows, row)
	}
	return rows, nil
}

// GetConfiguration reads the exact immutable slug.
func (c HTTPClient) GetConfiguration(ctx context.Context, slug string) (dto.BackendConfig, error) {
	const path = "/api/v1/config/details"
	if strings.TrimSpace(slug) == "" {
		return dto.BackendConfig{}, errors.New("configuration slug is required")
	}
	var wire configWire
	if err := c.projectRequest(ctx, path, struct {
		Slug string `json:"slug"`
	}{slug}, http.StatusOK, &wire); err != nil {
		return dto.BackendConfig{}, err
	}
	row, err := wire.backendValue()
	if err == nil && row.Slug != slug {
		err = &ContractError{errors.New("configuration slug differs from request")}
	}
	return row, contractStatus(path, err)
}

// InsertConfiguration creates a new row and returns only its committed slug.
func (c HTTPClient) InsertConfiguration(ctx context.Context, row dto.BackendConfig) (string, error) {
	const path = "/api/v1/config/insert"
	if err := validConfig(row); err != nil {
		return "", err
	}
	var result struct {
		Slug *string `json:"slug"`
	}
	if err := c.projectRequest(ctx, path, row, http.StatusCreated, &result); err != nil {
		return "", err
	}
	if result.Slug == nil || *result.Slug != row.Slug {
		return "", &HTTPError{Path: path, Status: http.StatusCreated, Cause: &ContractError{errors.New("created slug differs from request")}}
	}
	return *result.Slug, nil
}

// UpdateConfiguration replaces all six arrays for the existing slug.
func (c HTTPClient) UpdateConfiguration(ctx context.Context, row dto.BackendConfig) error {
	if err := validConfig(row); err != nil {
		return err
	}
	return c.projectRequest(ctx, "/api/v1/config/update", row, http.StatusNoContent, nil)
}

// DeleteConfiguration removes the exact slug when the backend permits it.
func (c HTTPClient) DeleteConfiguration(ctx context.Context, slug string) error {
	if strings.TrimSpace(slug) == "" {
		return errors.New("configuration slug is required")
	}
	return c.projectRequest(ctx, "/api/v1/config/delete", struct {
		Slug string `json:"slug"`
	}{slug}, http.StatusNoContent, nil)
}
