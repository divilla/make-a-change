package epics

import (
	"cli/internal/dto"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
)

func TestDetailsViewportScrollResizeAndReset(t *testing.T) {
	m := Model{DetailLoaded: true, Detail: dto.Epic{ID: 3, ProjectID: 7, Name: strings.Repeat("line\n", 40) + "last name line"}}
	assert.Contains(t, DetailsViewport(m, 80, 10), "ID: 3\nProject ID: 7\nName:")
	assert.NotContains(t, DetailsViewport(m, 80, 10), "Modified:")
	m = m.ScrollDetails(1000, 80, 10)
	assert.Contains(t, DetailsViewport(m, 80, 10), "last name line")
	assert.Contains(t, DetailsViewport(m, 80, 10), "Modified:")
	assert.Contains(t, DetailsViewport(m, 80, 100), "ID: 3")
	m = m.ScrollDetails(-1, 80, 100)
	assert.Zero(t, m.DetailOffset)
	m = m.ScrollDetails(-1000, 80, 1)
	assert.Equal(t, "ID: 3", DetailsViewport(m, 80, 1))
	assert.Empty(t, DetailsViewport(m, 80, 0))
	assert.Empty(t, DetailsViewport(Model{}, 80, 10))
	assert.Contains(t, DetailsViewport(Model{Loading: true, DetailOffset: 40}, 80, 10), "Loading")
	m.DetailOffset = 30
	m.Rows = []dto.Epic{{ID: 4, ProjectID: 7, Name: "next"}}
	m, _, _ = m.SelectDetail()
	assert.Zero(t, m.DetailOffset)
	for _, op := range []Operation{Details, Create, Edit} {
		m.Operation = op
		m.DetailOffset = 30
		var accepted bool
		m, accepted = m.Apply(Result{Generation: m.Generation, Operation: op, Epic: m.Rows[0]})
		assert.True(t, accepted)
		assert.Zero(t, m.DetailOffset)
	}
}

func TestLongEpicRenderingRemainsResponsive(t *testing.T) {
	m := Model{DetailLoaded: true, Detail: dto.Epic{ID: 3, ProjectID: 7, Name: strings.Repeat("x", 50000)}}
	m.Rows = []dto.Epic{m.Detail}
	for _, render := range []struct {
		name string
		view func() string
	}{
		{"table", func() string { return TableView(m, 80, 24) }},
		{"details", func() string { return DetailsViewport(m, 80, 12) }},
	} {
		t.Run(render.name, func(t *testing.T) {
			started := time.Now()
			view := render.view()
			// A broad ceiling catches the previous multi-second quadratic stall.
			assert.Less(t, time.Since(started), 2*time.Second)
			assert.Contains(t, view, strings.Repeat("x", 60))
			for _, line := range strings.Split(view, "\n") {
				assert.LessOrEqual(t, ansi.StringWidth(line), 80)
			}
		})
	}
}

func BenchmarkEpicLongNames(b *testing.B) {
	for _, size := range []int{5000, 50000, 100000} {
		m := Model{DetailLoaded: true, Detail: dto.Epic{ID: 3, ProjectID: 7, Name: strings.Repeat("x", size)}}
		m.Rows = []dto.Epic{m.Detail}
		b.Run(fmt.Sprintf("table/%d", size), func(b *testing.B) {
			for b.Loop() {
				TableView(m, 80, 24)
			}
		})
		b.Run(fmt.Sprintf("details/%d", size), func(b *testing.B) {
			for b.Loop() {
				DetailsViewport(m, 80, 12)
			}
		})
	}
}
