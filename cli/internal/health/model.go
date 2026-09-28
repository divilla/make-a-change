// Package health owns backend health diagnostic state and presentation.
package health

import (
	"cli/internal/dto"
	"context"
	"fmt"
	"strconv"
	"strings"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
)

// API is the backend health contract used by this screen.
type API interface {
	CheckHealth(context.Context, string) (dto.Health, error)
}

// Result belongs to one route and one generation.
type Result struct {
	Generation uint64
	Route      string
	Value      dto.Health
	Err        error
}

// Model preserves a valid diagnostic when a later refresh fails.
type Model struct {
	Route      string
	Generation uint64
	cancel     context.CancelFunc
	Busy       bool
	Last       *dto.Health
	Err        error
}

// Invalidate cancels a pending check and rejects late messages.
func (m Model) Invalidate() Model {
	if m.cancel != nil {
		m.cancel()
	}
	m.cancel = nil
	m.Generation++
	m.Busy = false
	return m
}

// SelectRoute switches to one of the two supported routes and clears prior-route data.
func (m Model) SelectRoute(route string) Model {
	m = m.Invalidate()
	m.Route = route
	m.Last = nil
	m.Err = nil
	return m
}

// Begin performs one health check for the selected route.
func (m Model) Begin(parent context.Context, api API) (Model, tea.Cmd) {
	if m.Busy {
		return m, nil
	}
	m = m.Invalidate()
	ctx, cancel := context.WithCancel(parent)
	m.cancel = cancel
	m.Busy = true
	generation, route := m.Generation, m.Route
	return m, func() tea.Msg {
		defer cancel()
		value, err := api.CheckHealth(ctx, route)
		return Result{Generation: generation, Route: route, Value: value, Err: err}
	}
}

// Apply ignores obsolete route/generation results.
func (m Model) Apply(r Result) (Model, bool) {
	if r.Generation != m.Generation || r.Route != m.Route || !m.Busy {
		return m, false
	}
	m.Busy = false
	m.cancel = nil
	m.Generation++
	m.Err = r.Err
	if r.Err == nil {
		value := r.Value
		m.Last = &value
	}
	return m, true
}

// View renders a safe public diagnostic and the selected route.
func View(m Model) string {
	lines := []string{"Backend health", "Route: " + m.Route}
	if m.Busy {
		lines = append(lines, "Loading health…")
	}
	if m.Last != nil {
		v := m.Last
		lines = append(lines, fmt.Sprintf("HTTP %d | status: %s | api: %s | database: %s", v.HTTPStatus, safeLine(v.Status), safeLine(v.API), safeLine(v.Database)))
		if v.Error != "" {
			lines = append(lines, "Error: "+safeLine(v.Error))
		}
	}
	if m.Err != nil {
		lines = append(lines, "Refresh failed: "+safeLine(m.Err.Error()))
	}
	return strings.Join(lines, "\n")
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
