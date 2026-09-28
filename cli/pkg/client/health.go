package client

import (
	"cli/internal/dto"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// CheckHealth performs one GET on the selected supported route.
func (c HTTPClient) CheckHealth(ctx context.Context, path string) (result dto.Health, err error) {
	if path != "/api/v1/health" && path != "/api/health" {
		return dto.Health{}, errors.New("unsupported health route")
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+path, nil)
	if err != nil {
		return dto.Health{}, &HTTPError{Path: path, Cause: err}
	}
	client := c.Client
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	bounded := *client
	bounded.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	resp, err := bounded.Do(req)
	if err != nil {
		return dto.Health{}, &HTTPError{Path: path, Cause: err}
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			err = errors.Join(err, &HTTPError{Path: path, Status: resp.StatusCode, Cause: closeErr})
		}
	}()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusServiceUnavailable {
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return dto.Health{}, &HTTPError{Path: path, Status: resp.StatusCode, Cause: errors.Join(fmt.Errorf("unexpected status: %s", body), readErr)}
	}
	decoder := json.NewDecoder(io.LimitReader(resp.Body, 1<<20))
	decoder.DisallowUnknownFields()
	var wire struct {
		Status   *string `json:"status"`
		API      *string `json:"api"`
		Database *string `json:"database"`
		Error    *string `json:"error"`
	}
	if err := decoder.Decode(&wire); err != nil {
		return dto.Health{}, &HTTPError{Path: path, Status: resp.StatusCode, Cause: &ContractError{err}}
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			err = errors.New("multiple JSON values")
		}
		return dto.Health{}, &HTTPError{Path: path, Status: resp.StatusCode, Cause: &ContractError{err}}
	}
	if wire.Status == nil || wire.API == nil || wire.Database == nil {
		return dto.Health{}, &HTTPError{Path: path, Status: resp.StatusCode, Cause: &ContractError{errors.New("missing health fields")}}
	}
	result = dto.Health{Route: path, HTTPStatus: resp.StatusCode, Status: *wire.Status, API: *wire.API, Database: *wire.Database}
	if wire.Error != nil {
		result.Error = *wire.Error
	}
	valid := resp.StatusCode == http.StatusOK && result.Status == "ok" && result.API == "ok" && result.Database == "ok" && result.Error == "" ||
		resp.StatusCode == http.StatusServiceUnavailable && result.Status == "degraded" && result.API == "ok" && result.Database == "error" && result.Error == "database unavailable"
	if !valid {
		return dto.Health{}, &HTTPError{Path: path, Status: resp.StatusCode, Cause: &ContractError{errors.New("inconsistent health fields and status")}}
	}
	return result, nil
}
