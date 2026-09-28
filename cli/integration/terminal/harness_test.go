package terminal_test

import (
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

func terminalBinary(root, scratch, supplied, counters string) (string, error) {
	if supplied != "" {
		info, err := os.Stat(supplied)
		if err != nil {
			return "", err
		}
		if !filepath.IsAbs(supplied) || info.IsDir() || info.Mode()&0o111 == 0 {
			return "", fmt.Errorf("invalid supplied executable")
		}
		info, err = os.Stat(counters)
		if err != nil {
			return "", err
		}
		if !filepath.IsAbs(counters) || !info.IsDir() {
			return "", fmt.Errorf("private counters directory required")
		}
		return supplied, nil
	}
	if counters != "" {
		return "", fmt.Errorf("counters require supplied covered executable")
	}
	binary := filepath.Join(scratch, "mch")
	cmd := exec.Command("go", "build", "-o", binary, "./cmd/mch")
	cmd.Dir = root
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("build terminal child: %w: %s", err, output)
	}
	return binary, nil
}

func waitTerminal(done <-chan error, timeout time.Duration) error {
	select {
	case err := <-done:
		return err
	case <-time.After(timeout):
		return fmt.Errorf("terminal process did not exit within %s", timeout)
	}
}

// socat's EXEC uses setsid. Only the process group recorded by our private
// wrapper is signalled; never search by executable name or kill a borrowed PID.
func cleanupTerminal(cmd *exec.Cmd, done <-chan error, pidPath string, timeout time.Duration) error {
	var cleanupErr error
	if content, err := os.ReadFile(pidPath); err == nil {
		pid, err := strconv.Atoi(strings.TrimSpace(string(content)))
		if err != nil || pid <= 1 || pid == os.Getpid() {
			cleanupErr = fmt.Errorf("invalid owned terminal PID")
		} else if group, err := syscall.Getpgid(pid); err == nil {
			if group != pid {
				cleanupErr = fmt.Errorf("terminal child is not its owned group leader")
			} else if err := syscall.Kill(-pid, syscall.SIGKILL); err != nil && err != syscall.ESRCH {
				cleanupErr = err
			}
		} else if err != syscall.ESRCH {
			cleanupErr = err
		}
	} else if !os.IsNotExist(err) {
		cleanupErr = err
	}
	if err := cmd.Process.Kill(); err != nil && err != os.ErrProcessDone && cleanupErr == nil {
		cleanupErr = err
	}
	select {
	case <-done: // A forced cleanup cannot erase the original assertion failure.
		return cleanupErr
	case <-time.After(timeout):
		return fmt.Errorf("terminal cleanup timed out; prior error: %v", cleanupErr)
	}
}

func TestHarnessBinarySelection(t *testing.T) {
	root := t.TempDir()
	binary := filepath.Join(root, "covered")
	require.NoError(t, os.WriteFile(binary, []byte("#!/bin/sh\nexit 0\n"), 0o700))
	got, err := terminalBinary("nonexistent", "unused", binary, root)
	require.NoError(t, err)
	require.Equal(t, binary, got)
	for _, pair := range [][2]string{{binary, ""}, {"missing", root}, {root, root}, {"", root}} {
		_, err := terminalBinary(root, root, pair[0], pair[1])
		require.Error(t, err)
	}
	_, err = terminalBinary(root, root, "", "")
	require.ErrorContains(t, err, "build terminal child")
}

func TestHarnessTimeoutAndOriginalExit(t *testing.T) {
	done := make(chan error, 1)
	require.ErrorContains(t, waitTerminal(done, time.Millisecond), "did not exit")
	original := fmt.Errorf("child crashed")
	done <- original
	require.ErrorIs(t, waitTerminal(done, time.Second), original)
	done <- nil
	require.NoError(t, waitTerminal(done, time.Second))
}

func TestHarnessOwnedCleanup(t *testing.T) {
	cmd := exec.Command("sleep", "30")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	require.NoError(t, cmd.Start())
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	pidFile := filepath.Join(t.TempDir(), "owned.pid")
	require.NoError(t, os.WriteFile(pidFile, []byte(strconv.Itoa(cmd.Process.Pid)), 0o600))
	require.NoError(t, cleanupTerminal(cmd, done, pidFile, time.Second))
	require.Error(t, syscall.Kill(cmd.Process.Pid, 0))
}

func TestHarnessCleanupRejectsUnownedPIDAndTimeout(t *testing.T) {
	cmd := exec.Command("sleep", "30")
	require.NoError(t, cmd.Start())
	t.Cleanup(func() { _ = cmd.Process.Kill(); _ = cmd.Wait() })
	pidFile := filepath.Join(t.TempDir(), "invalid.pid")
	require.NoError(t, os.WriteFile(pidFile, []byte(strconv.Itoa(os.Getpid())), 0o600))
	done := make(chan error, 1)
	done <- nil
	require.ErrorContains(t, cleanupTerminal(cmd, done, pidFile, time.Second), "invalid owned")
	require.ErrorContains(t, cleanupTerminal(cmd, done, pidFile, time.Millisecond), "cleanup timed out")
	require.NoError(t, syscall.Kill(os.Getpid(), 0))
}
