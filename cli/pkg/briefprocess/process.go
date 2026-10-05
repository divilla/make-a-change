package briefprocess

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// InteractiveCommand keeps terminal access in the foreground group while
// cancelling only the owned command and its descendants.
func InteractiveCommand(ctx context.Context, executable string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, executable, args...)
	cmd.WaitDelay = 2 * time.Second
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return os.ErrProcessDone
		}
		return killInteractiveTree(cmd.Process.Pid)
	}
	return cmd
}

// killInteractiveTree leaves the shared foreground group (including the UI)
// alone. Stop each parent before discovering its children so it cannot fork
// new children during traversal, then kill descendants before their parent.
func killInteractiveTree(pid int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return killStoppedTree(ctx, pid)
}

func killStoppedTree(ctx context.Context, pid int) (err error) {
	if err := syscall.Kill(pid, syscall.SIGSTOP); err != nil {
		if errors.Is(err, syscall.ESRCH) {
			return nil
		}
		return err
	}
	// Always terminate stopped processes, including when enumeration fails.
	defer func() {
		killErr := syscall.Kill(pid, syscall.SIGKILL)
		if !errors.Is(killErr, syscall.ESRCH) {
			err = errors.Join(err, killErr)
		}
	}()
	output, err := exec.CommandContext(ctx, "ps", "-A", "-o", "pid=", "-o", "ppid=").Output()
	if err != nil {
		return fmt.Errorf("enumerate interactive descendants: %w", err)
	}
	for _, line := range strings.Split(string(output), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}
		parent, _ := strconv.Atoi(fields[1])
		if parent != pid {
			continue
		}
		child, parseErr := strconv.Atoi(fields[0])
		if parseErr != nil || child <= 0 {
			return fmt.Errorf("invalid interactive descendant PID: %q", fields[0])
		}
		err = errors.Join(err, killStoppedTree(ctx, child))
	}
	return err
}
