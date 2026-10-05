package documentprocess

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

func Test031HistoryBatArgumentsAndExactBody(t *testing.T) {
	root := t.TempDir()
	executable := filepath.Join(root, "fake-bat")
	// The executable checks argv and copies the full file; shell metacharacters are data.
	require.NoError(t, os.WriteFile(executable, []byte("#!/bin/sh\n[ \"$#\" = 3 ] || exit 8\n[ \"$1\" = '-pp' ] || exit 9\n[ \"$2\" = '--color=always' ] || exit 10\ncase \"$3\" in *.md) ;; *) exit 11;; esac\nprintf '\\033[38;2;1;2;3m'\ncat \"$3\"\nprintf '\\033[0m'\n"), 0o700))
	body := "\t# Exact\n$(touch do-not-create) `echo data`\n\n" + strings.Repeat("long body\n", 50)
	output, err := (Bat{Executable: executable, TempRoot: root}).Print(context.Background(), body)
	require.NoError(t, err)
	require.Equal(t, "\x1b[38;2;1;2;3m"+body+"\x1b[0m", output)
	entries, err := os.ReadDir(root)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.Equal(t, "fake-bat", entries[0].Name())
}

func Test031HistoryBatFailureCancellationAndCleanup(t *testing.T) {
	for _, mode := range []string{"missing", "failure", "cancellation", "file"} {
		t.Run(mode, func(t *testing.T) {
			root := t.TempDir()
			b := Bat{Executable: filepath.Join(root, "bat"), TempRoot: root}
			ctx := context.Background()
			switch mode {
			case "failure":
				require.NoError(t, os.WriteFile(b.Executable, []byte("#!/bin/sh\nexit 23\n"), 0o700))
			case "cancellation":
				require.NoError(t, os.WriteFile(b.Executable, []byte("#!/bin/sh\nexec sleep 30\n"), 0o700))
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, 30*time.Millisecond)
				defer cancel()
			case "file":
				b.TempRoot = filepath.Join(root, "nonexistent")
			}
			_, err := b.Print(ctx, "full body")
			require.Error(t, err)
			if mode == "cancellation" {
				require.True(t, errors.Is(err, context.DeadlineExceeded))
			}
			entries, readErr := os.ReadDir(root)
			require.NoError(t, readErr)
			for _, entry := range entries {
				require.NotContains(t, entry.Name(), "mch-history-")
			}
		})
	}
}

func Test031RealBatForcesMarkdownColorsInCapturedStdout(t *testing.T) {
	output, err := (Bat{TempRoot: t.TempDir()}).Print(context.Background(), "# Heading\n\n```go\nfunc main() { println(\"colored\") }\n```\n")
	require.NoError(t, err)
	require.Contains(t, output, "\x1b[")
	require.Contains(t, output, "func")
	require.Contains(t, output, "colored")
}
