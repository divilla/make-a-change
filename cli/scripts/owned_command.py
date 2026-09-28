"""Linux command supervisor: reap every descendant, including detached PTYs."""
import ctypes
import os
from pathlib import Path
import signal
import subprocess
import sys
import time


class Interrupted(Exception):
    pass


def interrupt(signum, frame):
    raise Interrupted(signum)


def cleanup():
    # As a subreaper, this dedicated process adopts orphaned grandchildren even
    # across setsid(). Kill only its direct children, then repeat as descendants
    # are adopted. Never select processes by name or signal a borrowed group.
    deadline = time.monotonic() + 5
    timely = True
    while True:
        try:
            while os.waitpid(-1, os.WNOHANG)[0]:
                pass
        except ChildProcessError:
            return timely
        children = Path(f'/proc/self/task/{os.getpid()}/children').read_text()
        for child in children.split():
            try:
                os.kill(int(child), signal.SIGKILL)
            except ProcessLookupError:
                pass
        if time.monotonic() >= deadline:
            # Keep ownership (and the waiting caller's lock) while the kernel
            # finishes a killed child, but make the cleanup failure visible.
            print('owned command cleanup exceeded 5 seconds', file=sys.stderr, flush=True)
            timely = False
            deadline = float('inf')
        time.sleep(0.01)


def main(args):
    # A separate supervisor avoids adopting or reaping the caller's other work.
    # Fail before spawning on hosts without Linux subreaper/procfs support.
    if sys.platform != 'linux' or not Path('/proc/self/task').is_dir():
        raise RuntimeError('coverage command ownership requires Linux procfs')
    libc = ctypes.CDLL(None, use_errno=True)
    if libc.prctl(36, 1, 0, 0, 0) != 0:  # PR_SET_CHILD_SUBREAPER
        raise OSError(ctypes.get_errno(), 'PR_SET_CHILD_SUBREAPER')
    signal.signal(signal.SIGINT, interrupt)
    signal.signal(signal.SIGTERM, interrupt)
    result = 1
    try:
        process = subprocess.Popen(args)
        result = process.wait()
    except Interrupted as error:
        result = -error.args[0]
    finally:
        signal.signal(signal.SIGINT, signal.SIG_IGN)
        signal.signal(signal.SIGTERM, signal.SIG_IGN)
        if not cleanup() and result == 0:
            result = 1
    if result < 0:
        if -result != signal.SIGKILL:
            signal.signal(-result, signal.SIG_DFL)
        os.kill(os.getpid(), -result)
    return result


if __name__ == '__main__':
    sys.exit(main(sys.argv[1:]))
