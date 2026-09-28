package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFixedPromptInventoryAndResponsibilities(t *testing.T) {
	root := filepath.Join("..", "..", "..", ".mch", "default")
	entries, err := os.ReadDir(filepath.Join(root, "prompts"))
	require.NoError(t, err)
	names := []string{}
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	assert.Equal(t, []string{"brief-resolve.md", "brief-rewrite.md", "spec-fix.md", "spec-review.md", "spec-write.md"}, names)
	responsibilities := map[string][]string{
		"brief-rewrite.md": {"preserving", "intent", "level of detail", "fenced", "questions", "not ready"},
		"brief-resolve.md": {"supplied answers", "unanswered", "unresolved blockers", "follow-up", "readiness"},
		"spec-write.md":    {"clarified brief", "Goal, Scope, Requirements", "Verification", "QA Test Cases", "testable", "spec review"},
		"spec-review.md":   {"ambiguity, contradictions, omissions", "actionable findings", "affected spec", "No findings.", "revision", "empty or malformed"},
		"spec-fix.md":      {"each finding", "supplied answers", "unanswered blockers", "return the revised spec to spec review", "does not\ncomplete"},
	}
	for name, markers := range responsibilities {
		body := readTestFile(t, filepath.Join(root, "prompts", name))
		flat := strings.Join(strings.Fields(body), " ")
		for _, marker := range markers {
			assert.Contains(t, flat, strings.Join(strings.Fields(marker), " "), name)
		}
		for _, marker := range []string{"input", "context paths", "output path", "distinct", "Preserve original inputs", "Ask material questions", "Do not implement code", "no branch changes, commits, pushes"} {
			assert.Contains(t, flat, marker, name)
		}
		for _, obsolete := range []string{"/stg-tmp-dir", "/def-dir", "MCH_", "session-id", "change-types.md", "flow.yaml"} {
			assert.NotContains(t, body, obsolete, name)
		}
	}
	require.NoError(t, filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		assert.NotContains(t, []string{".yaml", ".yml", ".sh"}, filepath.Ext(path))
		assert.NotEqual(t, "Makefile", entry.Name())
		return nil
	}))
}

func TestRemovedCommandsCannotDispatchProcesses(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	for state := range commandsByState {
		for _, command := range []string{"/def-write", "/def-review", "/spec-write", "/spec-review", "/spec-review-chat", "/pr-write", "/artifact-chat", "/chat", "/resume", "/reference"} {
			m := NewModelWithClient(&fakeClient{})
			m.state = state
			next, cmd := m.executeCommand(command)
			require.Nil(t, cmd, "%s %s", state, command)
			assert.Equal(t, state, next.(Model).state)
			assert.Contains(t, next.(Model).err, "unknown command")
			assert.NotContains(t, commandsByState[state], command)
		}
	}
}
