package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractSessionID(t *testing.T) {
	id, ok := ExtractSessionID("session_id: abc-123\n")
	require.True(t, ok)
	assert.Equal(t, "abc-123", id)

	id, ok = ExtractSessionID(`{"session_id":"session.json"}`)
	require.True(t, ok)
	assert.Equal(t, "session.json", id)
}

func TestBuildInitialPromptInjectsIdea(t *testing.T) {
	prompt := BuildInitialPrompt("Initial idea:\n[Describe the feature/change in 2-5 sentences.]\n", "Build a CLI")
	assert.Contains(t, prompt, "Build a CLI")
	assert.NotContains(t, prompt, "[Describe")
}
