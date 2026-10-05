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
	assert.Equal(t, []string{"brief-rewrite.md", "spec-fix.md", "spec-review.md", "spec-write.md"}, names)
	responsibilities := map[string][]string{
		"brief-rewrite.md": {"[brief-file-path.md]", "overwrite the supplied input file", "questions one at a time", "level-one heading"},
		"spec-write.md":    {"[brief-file-path.md]", "spec.md", "Done.", "spec-template.md", "do not implement code"},
	}
	for name, markers := range responsibilities {
		body := readTestFile(t, filepath.Join(root, "prompts", name))
		flat := strings.Join(strings.Fields(body), " ")
		for _, marker := range markers {
			assert.Contains(t, strings.ToLower(flat), strings.ToLower(marker), name)
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
		for _, command := range []string{"/brief-new", "/brief-clarify", "/approve", "/resolve", "/brief-write", "/brief-review", "/spec-write", "/spec-review", "/spec-review-chat", "/pr-write", "/artifact-chat", "/chat", "/resume", "/reference"} {
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
