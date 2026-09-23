package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseRequirementMarkdown(t *testing.T) {
	req, err := ParseRequirementMarkdown(`# Test Requirement

Types: feature|test

Epic: Platform Foundation

## Problem

Body text.
`)
	require.NoError(t, err)
	assert.Equal(t, "Test Requirement", req.Title)
	assert.Equal(t, []string{"feature", "test"}, req.Types)
	require.NotNil(t, req.EpicName)
	assert.Equal(t, "Platform Foundation", *req.EpicName)
	assert.NotContains(t, req.Body, "Types:")
	assert.NotContains(t, req.Body, "Epic:")
	assert.Contains(t, req.Body, "## Problem")
}

func TestParseRequirementMarkdownRejectsInvalidOutput(t *testing.T) {
	_, err := ParseRequirementMarkdown("Types: feature\n\nBody")
	require.Error(t, err)

	_, err = ParseRequirementMarkdown("# One\n\nTypes: feature\n\n# Two")
	require.Error(t, err)

	_, err = ParseRequirementMarkdown("# One\n\nTypes: feature | test\n\nBody")
	require.Error(t, err)
}

func TestValidateRequirementReferences(t *testing.T) {
	epicName := "Platform"
	req := FinalRequirement{Title: "T", Types: []string{"feature"}, EpicName: &epicName, Body: "Body"}
	validated, err := ValidateRequirementReferences(req, ChangeReferences{
		Types: []ReferenceOption{{Slug: "feature"}},
	}, []Epic{{ID: 4, Name: "Platform"}})
	require.NoError(t, err)
	require.NotNil(t, validated.EpicID)
	assert.Equal(t, 4, *validated.EpicID)

	_, err = ValidateRequirementReferences(req, ChangeReferences{
		Types: []ReferenceOption{{Slug: "fix"}},
	}, []Epic{{ID: 4, Name: "Platform"}})
	require.Error(t, err)
}
