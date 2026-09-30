package changes

import (
	"cli/internal/dto"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilteredRowsMatchesAllNormalizedWordPrefixesInVisibleFields(t *testing.T) {
	rows := []dto.ChangeView{
		{Ref: "AC-12", Title: "Add DB Migrations", ChangePhase: "in-progress", ChangeTypes: []string{"maintenance"}, EpicName: "Storage"},
		{Ref: "AC-13", Title: "---===Feature===---", ChangePhase: "backlog"},
		{Ref: "AC-14", Title: "Other", ChangePhase: "done", ChangeTypes: []string{"feature"}},
		{Ref: "AC-15", Title: "Unrelated", ChangePhase: "done", Brief: "Add DB Migrations", Spec: "Feature", RefSlug: "015-feature"},
	}
	for _, tt := range []struct {
		query string
		want  []dto.ChangeView
	}{
		{query: "ad mig", want: rows[:1]},
		{query: "AD mig", want: rows[:1]},
		{query: "fea", want: rows[1:3]},
		{query: "AC-12 stor", want: rows[:1]},
		{query: "prog mai", want: []dto.ChangeView{}},
		{query: "in-prog mai", want: rows[:1]},
		{query: "igr", want: []dto.ChangeView{}},
		{query: "", want: rows},
	} {
		t.Run(tt.query, func(t *testing.T) {
			assert.Equal(t, tt.want, FilteredRows(rows, Filters{Find: tt.query}))
		})
	}
}

func TestParseSpecStructureTracksOptionalTypesMetadata(t *testing.T) {
	tests := []struct {
		name    string
		spec    string
		present bool
		values  []string
	}{
		{name: "omitted", spec: "# Change\n\nBody"},
		{name: "empty", spec: "# Change\n\nTypes:\n\nBody", present: true, values: []string{}},
		{name: "pipe-delimited and processed locally", spec: "# Change\n\nTypes: fix|feature|unsupported!\n\nBody", present: true, values: []string{"fix", "feature", "unsupported"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := ParseSpecStructure(tt.spec)
			require.NoError(t, err)
			assert.Equal(t, tt.present, parsed.ChangeTypesPresent)
			assert.Equal(t, tt.values, parsed.ChangeTypes)
			assert.Equal(t, tt.spec, parsed.Spec)
		})
	}
}

func TestParseBriefStructureTracksOptionalTypesMetadata(t *testing.T) {
	parsed, err := ParseBriefStructure("# Change\n\nBody")
	require.NoError(t, err)
	assert.False(t, parsed.ChangeTypesPresent)

	parsed, err = ParseBriefStructure("# Change\n\nTypes:\n\nBody")
	require.NoError(t, err)
	assert.True(t, parsed.ChangeTypesPresent)
	assert.Empty(t, parsed.ChangeTypes)

	parsed, err = ParseBriefStructure("# Change\nTypes: fix|feature\n\nBody")
	require.NoError(t, err)
	assert.True(t, parsed.ChangeTypesPresent)
	assert.Equal(t, []string{"fix", "feature"}, parsed.ChangeTypes)
}

func TestParseBriefStructureRequiresNonMetadataBody(t *testing.T) {
	for _, brief := range []string{
		"# Change\n\n",
		"# Change\n\nTypes:",
		"# Change\n\nTypes: feature",
	} {
		_, err := ParseBriefStructure(brief)
		require.EqualError(t, err, "brief body is required")
	}
}
