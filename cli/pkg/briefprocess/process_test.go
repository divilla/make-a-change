package briefprocess

import (
	"context"
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

func processRunning(pid int) bool {
	output, err := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "stat=").Output()
	state := strings.TrimSpace(string(output))
	return err == nil && state != "" && !strings.HasPrefix(state, "Z")
}

func Test032InteractiveCancellationKillsDescendantsWithoutSignalingForegroundGroup(t *testing.T) {
	for _, args := range [][]string{{"-C", "repo", "prompt"}, {"resume", testUUID}} {
		t.Run(args[0], func(t *testing.T) {
			dir := t.TempDir()
			script := filepath.Join(dir, "codex")
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
			// A sibling shares the foreground group but must survive cancellation.
			sibling := exec.Command("sleep", "30")
			require.NoError(t, sibling.Start())
			t.Cleanup(func() { _ = sibling.Process.Kill(); _ = sibling.Wait() })
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			cmd := (Runner{Executable: script}).Interactive(ctx, args...)
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
				require.True(t, processRunning(pid))
			}
			cancel()
			finished := make(chan error, 1)
			go func() { finished <- cmd.Wait() }()
			select {
			case err := <-finished:
				require.Error(t, err)
			case <-time.After(4 * time.Second):
				t.Fatal("interactive cancellation did not finish")
			}
			for _, pid := range descendants {
				require.Eventually(t, func() bool { return !processRunning(pid) }, time.Second, 10*time.Millisecond)
			}
			require.True(t, processRunning(sibling.Process.Pid))
		})
	}
}

func Test032InteractiveCancellationEnumerationFailureStillKillsStoppedParent(t *testing.T) {
	cmd := exec.Command("sleep", "30")
	require.NoError(t, cmd.Start())
	t.Cleanup(func() { _ = cmd.Process.Kill() })
	t.Setenv("PATH", t.TempDir())
	require.ErrorContains(t, killInteractiveTree(cmd.Process.Pid), "enumerate interactive descendants")
	require.Error(t, cmd.Wait())
	require.NoError(t, killInteractiveTree(cmd.Process.Pid))
	interactive := (Runner{}).Interactive(context.Background(), "resume", testUUID)
	require.ErrorIs(t, interactive.Cancel(), os.ErrProcessDone)
}
