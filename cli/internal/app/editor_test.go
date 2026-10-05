package app

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func editorProcessRunning(pid int) bool {
	output, err := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "stat=").Output()
	state := strings.TrimSpace(string(output))
	return err == nil && state != "" && !strings.HasPrefix(state, "Z")
}

func Test032EditorCancellationKillsDescendantsAndRetainsForegroundGroupAndDraft(t *testing.T) {
	for _, configured := range []bool{true, false} {
		t.Run(fmt.Sprintf("configured=%t", configured), func(t *testing.T) {
			dir := t.TempDir()
			path, script := filepath.Join(dir, "brief.md"), filepath.Join(dir, "editor")
			require.NoError(t, os.WriteFile(path, []byte("# Retained draft"), 0o600))
			require.NoError(t, os.WriteFile(script, []byte(`#!/bin/sh
if [ "$1" = child ]; then
 sleep 30 &
 echo $! > "$0.leaf"
else
 sh "$0" child &
 echo $! > "$0.child"
fi
wait
`), 0o700))
			editor := script
			if !configured {
				t.Setenv("EDITOR", script)
				editor = ""
			}
			sibling := exec.Command("sleep", "30")
			require.NoError(t, sibling.Start())
			t.Cleanup(func() { _ = sibling.Process.Kill(); _ = sibling.Wait() })
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			cmd := editorProcess(ctx, editor, path)
			require.NoError(t, cmd.Start())
			t.Cleanup(func() { _ = cmd.Process.Kill() })
			pgid, err := syscall.Getpgid(cmd.Process.Pid)
			require.NoError(t, err)
			require.Equal(t, syscall.Getpgrp(), pgid)
			var descendants []int
			for _, suffix := range []string{"child", "leaf"} {
				var data []byte
				require.Eventually(t, func() bool {
					data, err = os.ReadFile(script + "." + suffix)
					return err == nil && len(data) > 0
				}, 3*time.Second, 10*time.Millisecond)
				pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
				require.NoError(t, err)
				descendants = append(descendants, pid)
				t.Cleanup(func() { _ = syscall.Kill(pid, syscall.SIGKILL) })
				require.True(t, editorProcessRunning(pid))
			}
			cancel()
			finished := make(chan error, 1)
			go func() { finished <- cmd.Wait() }()
			select {
			case err := <-finished:
				require.Error(t, err)
			case <-time.After(4 * time.Second):
				t.Fatal("editor cancellation did not finish")
			}
			for _, pid := range descendants {
				require.Eventually(t, func() bool { return !editorProcessRunning(pid) }, time.Second, 10*time.Millisecond)
			}
			require.True(t, editorProcessRunning(sibling.Process.Pid))
			require.Equal(t, "# Retained draft", readTestFile(t, path))
		})
	}
}
