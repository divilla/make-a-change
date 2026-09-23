package changes

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func TestParseDefStructureTracksOptionalTypesMetadata(t *testing.T) {
	parsed, err := ParseDefStructure("# Change\n\nBody")
	require.NoError(t, err)
	assert.False(t, parsed.ChangeTypesPresent)

	parsed, err = ParseDefStructure("# Change\n\nTypes:\n\nBody")
	require.NoError(t, err)
	assert.True(t, parsed.ChangeTypesPresent)
	assert.Empty(t, parsed.ChangeTypes)

	parsed, err = ParseDefStructure("# Change\nTypes: fix|feature\n\nBody")
	require.NoError(t, err)
	assert.True(t, parsed.ChangeTypesPresent)
	assert.Equal(t, []string{"fix", "feature"}, parsed.ChangeTypes)
}

func TestParseDefStructureRequiresNonMetadataBody(t *testing.T) {
	for _, def := range []string{
		"# Change\n\n",
		"# Change\n\nTypes:",
		"# Change\n\nTypes: feature",
	} {
		_, err := ParseDefStructure(def)
		require.EqualError(t, err, "definition body is required")
	}
}
