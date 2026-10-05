package briefprocess

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const testUUID = "0198a86f-9b8a-7d89-ae5b-6f25b528b04c"

func Test032WorkspaceOwnershipAndExactFiles(t *testing.T) {
	root := t.TempDir()
	w := &Workspace{TempDir: root}
	path, err := w.Prepare(testUUID, "")
	require.NoError(t, err)
	require.Equal(t, filepath.Join(root, "mch", testUUID, "brief.md"), path)
	body, err := w.Read(path)
	require.NoError(t, err)
	require.Empty(t, body)
	stamp, err := w.Stamp(path)
	require.NoError(t, err)
	require.True(t, stamp.Exists)
	spec := filepath.Join(filepath.Dir(path), "spec.md")
	stamp, err = w.Stamp(spec)
	require.NoError(t, err)
	require.False(t, stamp.Exists)
	require.NoError(t, os.WriteFile(path, []byte("# Draft\t\n"), 0o600))
	body, err = w.Read(path)
	require.NoError(t, err)
	require.Equal(t, "# Draft\t\n", body)
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".mch/default/prompts"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".mch/default/prompts/brief-rewrite.md"), []byte("Read [brief-file-path.md] then [brief-file-path.md]"), 0o600))
	prompt, err := w.Prompt(root, "brief-rewrite", path)
	require.NoError(t, err)
	require.Equal(t, "Read "+path+" then "+path, prompt)
	_, err = w.Prompt(root, "unknown", path)
	require.Error(t, err)
	_, err = w.Prompt(root, "spec-write", path)
	require.Error(t, err)
	require.NoError(t, os.WriteFile(spec, []byte("spec"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(filepath.Dir(path), "final.txt"), []byte("Done."), 0o600))
	require.NoError(t, w.Cleanup(path))
	require.NoDirExists(t, filepath.Dir(path))
}

func Test032WorkspaceRefusesUnownedAndNonRegularPaths(t *testing.T) {
	for _, tc := range []string{"parent file", "parent symlink", "unowned dir", "uuid file", "invalid uuid"} {
		t.Run(tc, func(t *testing.T) {
			root := t.TempDir()
			parent := filepath.Join(root, "mch")
			dir := filepath.Join(parent, testUUID)
			w := &Workspace{TempDir: root}
			ref := testUUID
			sentinel := ""
			switch tc {
			case "parent file":
				sentinel = parent
				require.NoError(t, os.WriteFile(parent, []byte("keep"), 0o600))
			case "parent symlink":
				require.NoError(t, os.Symlink(t.TempDir(), parent))
			case "unowned dir":
				require.NoError(t, os.MkdirAll(dir, 0o700))
				sentinel = filepath.Join(dir, "brief.md")
				require.NoError(t, os.WriteFile(sentinel, []byte("keep"), 0o600))
			case "uuid file":
				require.NoError(t, os.Mkdir(parent, 0o700))
				sentinel = dir
				require.NoError(t, os.WriteFile(dir, []byte("keep"), 0o600))
			case "invalid uuid":
				ref = "../../escape"
			}
			_, err := w.Prepare(ref, "overwrite")
			require.Error(t, err)
			if sentinel != "" {
				data, err := os.ReadFile(sentinel)
				require.NoError(t, err)
				require.Equal(t, "keep", string(data))
			}
		})
	}
	for _, tc := range []string{"symlink draft", "unexpected file", "replaced directory", "outside path", "missing ownership"} {
		t.Run(tc, func(t *testing.T) {
			w := &Workspace{TempDir: t.TempDir()}
			path, err := w.Prepare(testUUID, "keep")
			require.NoError(t, err)
			switch tc {
			case "symlink draft":
				require.NoError(t, os.Symlink(path, filepath.Join(filepath.Dir(path), "spec.md")))
			case "unexpected file":
				require.NoError(t, os.WriteFile(filepath.Join(filepath.Dir(path), "unrelated"), []byte("keep"), 0o600))
			case "replaced directory":
				require.NoError(t, os.Rename(filepath.Dir(path), filepath.Dir(path)+"-old"))
				require.NoError(t, os.Mkdir(filepath.Dir(path), 0o700))
			case "outside path":
				path = filepath.Join(t.TempDir(), "brief.md")
			case "missing ownership":
				w = &Workspace{}
			}
			require.Error(t, w.Cleanup(path))
			if tc == "symlink draft" || tc == "unexpected file" {
				body, err := os.ReadFile(path)
				require.NoError(t, err)
				require.Equal(t, "keep", string(body))
			}
		})
	}
}

func Test032CodexExecArgumentsStreamingFinalAndSession(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "repo with spaces")
	scratch := filepath.Join(base, "scratch with spaces")
	require.NoError(t, os.Mkdir(root, 0o700))
	require.NoError(t, os.Mkdir(scratch, 0o700))
	w := &Workspace{TempDir: scratch}
	brief, err := w.Prepare(testUUID, "# Brief")
	require.NoError(t, err)
	dir := filepath.Dir(brief)
	prompt := "literal `prompt` $(untouched)\nsecond line"
	script := filepath.Join(root, "codex")
	body := `#!/bin/sh
set -eu
[ "$#" = 12 ] && [ "$1" = exec ] && [ "$2" = -C ] && [ "$4" = --color ] && [ "$5" = always ] && [ "$6" = --output-last-message ] || exit 8
[ "$8" = --sandbox ] && [ "$9" = workspace-write ] && [ "${10}" = --add-dir ] && [ "${11}" = "$(dirname "$7")" ] || exit 9
printf '%s' "$3" > "$7.root"
printf '%s' "${12}" > "$7.prompt"
printf '# Generated spec\n' > "${11}/spec.md"
printf '\033[32mstreaming progress\033[0m\n'
printf 'session id: ` + testUUID + `\n' >&2
printf 'Done.\n' > "$7"
`
	require.NoError(t, os.WriteFile(script, []byte(body), 0o700))
	progress := make(chan string, 10)
	result, err := (Runner{Executable: script}).Exec(context.Background(), root, brief, prompt, progress)
	require.NoError(t, err)
	require.Equal(t, "Done.", result.Final)
	require.Equal(t, testUUID, result.SessionID)
	spec, err := w.Read(filepath.Join(dir, "spec.md"))
	require.NoError(t, err)
	require.Equal(t, "# Generated spec\n", spec)
	gotRoot, err := os.ReadFile(filepath.Join(dir, "final.txt.root"))
	require.NoError(t, err)
	require.Equal(t, root, string(gotRoot))
	gotPrompt, err := os.ReadFile(filepath.Join(dir, "final.txt.prompt"))
	require.NoError(t, err)
	require.Equal(t, prompt, string(gotPrompt))
	output := ""
	for len(progress) > 0 {
		output += <-progress
	}
	require.Contains(t, output, "\x1b[32mstreaming progress\x1b[0m")
	cmd := (Runner{Executable: script}).Interactive(context.Background(), "resume", testUUID)
	require.Equal(t, []string{script, "resume", testUUID}, cmd.Args)
	_, err = (Runner{Executable: script}).Exec(context.Background(), root, brief, prompt, nil)
	require.ErrorContains(t, err, "file exists")
}

func Test032CodexFailuresCancellationAndFinalFileValidation(t *testing.T) {
	for _, tc := range []string{"failed with ID", "cancel", "symlink final", "large final", "missing final"} {
		t.Run(tc, func(t *testing.T) {
			dir := t.TempDir()
			script := filepath.Join(dir, "codex")
			body := "#!/bin/sh\n"
			switch tc {
			case "failed with ID":
				body += "printf 'session id: " + testUUID + "\\n'; exit 3\n"
			case "cancel":
				body += "sleep 10\n"
			case "symlink final":
				body += "rm \"$7\"; ln -s /dev/null \"$7\"\n"
			case "large final":
				body += "head -c 1048577 /dev/zero > \"$7\"\n"
			case "missing final":
				body += "rm \"$7\"\n"
			}
			require.NoError(t, os.WriteFile(script, []byte(body), 0o700))
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			result, err := (Runner{Executable: script}).Exec(ctx, dir, filepath.Join(dir, "brief.md"), "prompt", nil)
			require.Error(t, err)
			if tc == "failed with ID" {
				require.Equal(t, testUUID, result.SessionID)
			}
			if tc == "cancel" {
				require.True(t, errors.Is(ctx.Err(), context.DeadlineExceeded))
			}
		})
	}
	writer := &streamWriter{ctx: context.Background()}
	_, err := writer.Write([]byte(strings.Repeat("x", maxOutput+100)))
	require.NoError(t, err)
	require.Equal(t, maxOutput, writer.buffer.Len())
}
