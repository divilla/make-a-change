"""Measure the standalone HTTP suite using an owned server and an existing DB.

Only the server process is owned here. No database clients, SQL setup, fixture
loading, outage phases or database lifecycle commands are used.
"""
import contextlib
import hashlib
import json
import os
from pathlib import Path
import re
import shutil
import signal
import socket
import subprocess
import sys
import time

import coverage


@contextlib.contextmanager
def deferred_signals():
    previous = signal.pthread_sigmask(signal.SIG_BLOCK, {signal.SIGINT, signal.SIGTERM})
    try:
        yield previous
    finally:
        signal.pthread_sigmask(signal.SIG_SETMASK, previous)


@contextlib.contextmanager
def cleanup_signals():
    handlers = {sig: signal.signal(sig, signal.SIG_IGN) for sig in (signal.SIGINT, signal.SIGTERM)}
    try:
        yield
    finally:
        for sig, handler in handlers.items():
            signal.signal(sig, handler)


def spawn(args, env, output, previous_signals):
    return subprocess.Popen(args, cwd=coverage.BACKEND, env=env, stdout=output,
                            stderr=None if output is None else subprocess.STDOUT,
                            start_new_session=True,
                            preexec_fn=lambda: signal.pthread_sigmask(signal.SIG_SETMASK, previous_signals))


def execute(args, env, output, timeout=120):
    process = None
    try:
        with deferred_signals() as previous:
            process = spawn(args, env, output, previous)
        code = process.wait(timeout=timeout)
    except BaseException:
        if process is not None:
            with cleanup_signals():
                for cleanup in (lambda: os.killpg(process.pid, signal.SIGKILL),
                                lambda: process.wait(timeout=5)):
                    try:
                        cleanup()
                    except ProcessLookupError:
                        pass
                    except BaseException as error:
                        print(f'command cleanup failed: {error}', file=sys.stderr)
        raise
    if code:
        raise subprocess.CalledProcessError(code, args)


def free_port(port):
    if not 1 <= port <= 65535:
        raise ValueError('invalid API_TEST_PORT')
    with socket.socket(socket.AF_INET6, socket.SOCK_STREAM) as sock:
        sock.setsockopt(socket.IPPROTO_IPV6, socket.IPV6_V6ONLY, 0)
        sock.bind(('::', port))


def await_server(server, port, timeout=15):
    deadline = time.monotonic() + timeout
    while time.monotonic() < deadline:
        if server.poll() is not None:
            raise RuntimeError('instrumented server exited before readiness; see .coverage/api/server.log')
        try:
            with socket.create_connection(('127.0.0.1', port), timeout=.2):
                if server.poll() is not None:
                    raise RuntimeError('instrumented server exited during readiness')
                return
        except OSError:
            time.sleep(.05)
    raise TimeoutError('instrumented server readiness timed out')


def stop_server(server):
    with cleanup_signals():
        if server.poll() is not None:
            raise RuntimeError(f'instrumented server exited unexpectedly ({server.returncode})')
        server.send_signal(signal.SIGTERM)
        try:
            code = server.wait(timeout=15)
        except subprocess.TimeoutExpired:
            server.kill()
            server.wait(timeout=5)
            raise RuntimeError('instrumented server did not stop gracefully; coverage invalid') from None
        if code:
            raise RuntimeError(f'instrumented server shutdown failed ({code})')


def copy_suite(directory, port):
    source = coverage.BACKEND / 'apih-tests'
    # Validate before copying: no symlinks, malformed YAML or debug breakpoints.
    manifest = coverage.command([coverage.GO, 'run', './scripts/validate-apih-suite', str(source)],
                                capture_output=True).stdout
    json.loads(manifest)
    (directory / 'suite-manifest.json').write_text(manifest)
    suite = directory / 'suite'
    shutil.copytree(source, suite)
    root = suite / 'root.yaml'
    text, count = re.subn(r'(?m)^([ \t]*)base_url:[^\n]*$',
                          rf'\1base_url: http://127.0.0.1:{port}', root.read_text())
    if count != 1:
        raise ValueError('suite root must declare exactly one base_url')
    root.write_text(text)
    (directory / 'suite-inputs.json').write_text(json.dumps({
        str(path.relative_to(suite)): hashlib.sha256(path.read_bytes()).hexdigest()
        for path in suite.rglob('*.yaml')}, indent=2))
    return suite


def campaign(directory, port):
    free_port(port)
    suite = copy_suite(directory, port)
    metadata = coverage.inventory(directory)
    metadata['linked'] = coverage.command([coverage.GO, 'list', '-deps', '-f', '{{.ImportPath}}',
                                          './cmd/server'], capture_output=True).stdout.splitlines()
    (directory / 'denominator.json').write_text(json.dumps(metadata, indent=2))
    binary = directory / 'mch-server'
    counters = directory / 'counters'
    counters.mkdir()
    env = dict(os.environ, GOCOVERDIR=str(counters), XDG_CACHE_HOME=str(directory / 'cache'))
    apih = os.environ.get('APIH', 'apih')
    build = [coverage.GO, 'build', '-cover', '-covermode=atomic',
             '-coverpkg=' + ','.join(coverage.PATTERNS), '-o', str(binary), './cmd/server']
    suite_command = [apih, str(suite)]
    with (directory / 'runner.log').open('w') as log, (directory / 'server.log').open('w') as server_log:
        execute(build, env, log)
        coverage.provenance(directory, {'build': build, 'suite_command': suite_command,
            'server': [str(binary), '-port', str(port)],
            'server_sha256': hashlib.sha256(binary.read_bytes()).hexdigest(),
            'database': 'existing configured database; credentials not recorded',
            'readiness': 'TCP only; all HTTP requests come from APIHydra',
            'coverage_threshold_enforced': True})
        server = None
        try:
            with deferred_signals() as previous:
                server = spawn([str(binary), '-port', str(port)], env, server_log, previous)
            await_server(server, port)
            sys.stdout.flush()
            sys.stderr.flush()
            execute(suite_command, env, None, timeout=300)
        finally:
            primary_error = sys.exc_info()[0] is not None
            if server is not None:
                try:
                    stop_server(server)
                except Exception as error:
                    print(f'server cleanup failed: {error}', file=sys.stderr)
                    if not primary_error:
                        raise
        if not list(counters.glob('covmeta.*')) or not list(counters.glob('covcounters.*')):
            raise ValueError('missing instrumented server coverage counters')
        profile = directory / 'coverage.out'
        execute([coverage.GO, 'tool', 'covdata', 'textfmt', '-i=' + str(counters), '-o=' + str(profile)], env, log)
    print(flush=True)
    return coverage.report(metadata, profile, directory, integration=True)


def interrupted(signum, _frame):
    raise InterruptedError(signum, 'API coverage interrupted')


def main():
    handlers = {sig: signal.signal(sig, interrupted) for sig in (signal.SIGINT, signal.SIGTERM)}
    try:
        with coverage.fresh_run('api') as directory:
            try:
                passed = campaign(directory, int(os.environ.get('API_TEST_PORT', '19080')))
            except BaseException:
                for name in ('coverage.out', 'report.txt', 'result.json'):
                    (directory / name).unlink(missing_ok=True)
                raise
        return 0 if passed else 1
    finally:
        for sig, handler in handlers.items():
            signal.signal(sig, handler)


if __name__ == '__main__':
    try:
        sys.exit(main())
    except subprocess.CalledProcessError as error:
        print(f'API coverage command failed (exit {error.returncode}); see output above and .coverage/api/runner.log', file=sys.stderr)
        sys.exit(error.returncode if error.returncode > 0 else 1)
    except InterruptedError as error:
        sys.exit(128 + error.errno)
    except (OSError, RuntimeError, ValueError, subprocess.TimeoutExpired) as error:
        print(f'API coverage failed: {error}', file=sys.stderr)
        sys.exit(1)
