"""Run the complete APIHydra suite against an owned instrumented server and DB."""
import os
import json
import contextlib
from pathlib import Path
import shlex
import shutil
import signal
import socket
import subprocess
import sys
import tempfile
import time
import urllib.request
from urllib.parse import quote, urlencode

import coverage

BACKEND = coverage.BACKEND


class Runner:
    def __init__(self, directory):
        self.directory = directory
        self.env = {key: value for key, value in os.environ.items() if not key.startswith('PG')}
        self.env['XDG_CACHE_HOME'] = str(directory / 'cache')
        self.commands = []
        self.server = None
        self.database = None
        self.log = (directory / 'runner.log').open('w')

    def execute(self, args, timeout=120):
        # DB URLs are deliberately not included in provenance/logged commands.
        self.commands.append([str(arg) for arg in args])
        process = subprocess.Popen(args, cwd=BACKEND, env=self.env, stdout=self.log,
                                   stderr=subprocess.STDOUT, start_new_session=True)
        try:
            code = process.wait(timeout=timeout)
        except BaseException:
            os.killpg(process.pid, signal.SIGKILL)
            process.wait()
            raise
        if code:
            raise subprocess.CalledProcessError(code, args)

    def stop_server(self):
        if self.server is None:
            return
        # Retain ownership if a signal interrupts this wait: final cleanup must
        # still be able to stop and reap the process.
        server = self.server
        if server.poll() is None:
            server.send_signal(signal.SIGTERM)
        try:
            code = server.wait(timeout=15)
        except subprocess.TimeoutExpired:
            server.kill()
            server.wait()
            self.server = None
            raise RuntimeError('owned server did not shut down normally') from None
        self.server = None
        if code != 0:
            raise RuntimeError(f'owned server exited {code}')

    def cleanup(self):
        handlers = {sig: signal.signal(sig, signal.SIG_IGN) for sig in (signal.SIGINT, signal.SIGTERM)}
        try:
            self._cleanup()
        finally:
            for sig, handler in handlers.items():
                signal.signal(sig, handler)

    def _cleanup(self):
        failure = None
        try:
            self.stop_server()
        except Exception as error:
            failure = error
        if self.database is not None:
            # pg_ctl only addresses our newly created private data directory.
            try:
                self.execute(['pg_ctl', '-D', str(self.database), '-m', 'fast', '-w', '-t', '10', 'stop'], 15)
            except Exception as error:
                failure = failure or error
                if (self.database / 'postmaster.pid').exists():
                    try:
                        self.execute(['pg_ctl', '-D', str(self.database), '-m', 'immediate', '-w', '-t', '10', 'stop'], 15)
                    except Exception as fallback:
                        print(f'owned database emergency cleanup failed: {fallback}', file=sys.stderr)
        if failure:
            raise failure


@contextlib.contextmanager
def private_cluster():
    scratch = tempfile.mkdtemp(prefix='mch-pg-')
    try:
        yield scratch
    finally:
        if (Path(scratch) / 'data/postmaster.pid').exists():
            # Never unlink a cluster whose shutdown failed; retain diagnostics.
            print(f'owned cluster still has a PID file; retained at {scratch}', file=sys.stderr)
        else:
            shutil.rmtree(scratch)


def free_port(port):
    # The application listens on all interfaces; reject any occupied TCP port,
    # including a service that accepts connections without speaking HTTP.
    with socket.socket(socket.AF_INET6, socket.SOCK_STREAM) as sock:
        sock.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
        sock.setsockopt(socket.IPPROTO_IPV6, socket.IPV6_V6ONLY, 0)
        sock.bind(('::', port))


def prerequisites():
    for name in [coverage.GO, 'apih', 'initdb', 'pg_ctl', 'postgres', 'psql', 'curl', 'jq', 'git']:
        if not shutil.which(name):
            raise RuntimeError('missing prerequisite: ' + name)


def copy_suite(directory, port):
    source = BACKEND / 'apih-tests'
    suite = directory / 'suite'
    shutil.copytree(source, suite)
    # Parse YAML, including quoted/escaped keys, flow mappings and aliases.
    # Reuse the backend's existing YAML module; no Python dependency is needed.
    coverage.command([coverage.GO, 'run', './scripts/validate-apih-suite', str(suite)])
    root = suite / 'root.yaml'
    text = root.read_text()
    expected = 'base_url: http://127.0.0.1:19080'
    if text.count(expected) != 1:
        raise ValueError('suite root must declare the canonical local base_url')
    root.write_text(text.replace(expected, f'base_url: http://127.0.0.1:{port}'))
    return suite


def campaign(runner, port, legacy=False):
    directory = runner.directory
    metadata = coverage.inventory(directory) if not legacy else None
    if not legacy:
        metadata['linked'] = coverage.command(
            [coverage.GO, 'list', '-deps', '-f', '{{.ImportPath}}', './cmd/server'],
            capture_output=True).stdout.splitlines()
        (directory / 'denominator.json').write_text(json.dumps(metadata, indent=2))
    suite = copy_suite(directory, port) if not legacy else None
    binary = directory / 'mch-server'
    runner.execute([coverage.GO, 'build', '-cover', '-covermode=atomic',
                    '-coverpkg=' + ','.join(coverage.PATTERNS), '-o', str(binary), './cmd/server'])
    counters = directory / 'counters'
    counters.mkdir()
    runner.env['GOCOVERDIR'] = str(counters)
    # Private Unix socket only: never connect to or stop the user's port 5432.
    with private_cluster() as scratch:
        data = Path(scratch) / 'data'
        runner.execute(['initdb', '-D', str(data), '-U', 'postgres', '--auth=trust', '--no-locale'])
        try:
            runner.database = data
            runner.execute(['pg_ctl', '-D', str(data), '-l', str(directory / 'postgres.log'),
                            '-o', shlex.join(['-k', scratch, '-h', '', '-p', '15432']),
                            '-w', '-t', '10', 'start'], 15)
            for filename in ['init.sql', 'seed.sql']:
                runner.execute(['psql', '-X', '-h', scratch, '-p', '15432', '-U', 'postgres', '-d', 'postgres',
                                '-v', 'ON_ERROR_STOP=1', '-f', str(BACKEND.parent / 'db' / filename)])
            db_url = 'postgresql://postgres@/postgres?' + urlencode(
                {'host': scratch, 'port': '15432', 'sslmode': 'disable'}, quote_via=quote)
            with (directory / 'server.log').open('w') as server_log:
                runner.server = subprocess.Popen([str(binary), '-port', str(port), '-db', db_url],
                    cwd=BACKEND, env=runner.env, stdout=server_log, stderr=subprocess.STDOUT)
                deadline = time.monotonic() + 15
                while True:
                    if runner.server.poll() is not None:
                        raise RuntimeError('server startup failed; see server.log')
                    try:
                        with urllib.request.urlopen(f'http://127.0.0.1:{port}/api/v1/health', timeout=0.5) as response:
                            if response.status == 200:
                                break
                    except OSError:
                        pass
                    if time.monotonic() >= deadline:
                        raise RuntimeError('server readiness timed out')
                    time.sleep(0.1)
                if legacy:
                    runner.env['API_TEST_BASE_URL'] = f'http://127.0.0.1:{port}'
                    runner.env['API_TEST_DB_URL'] = db_url
                    runner.execute([coverage.GO, 'test', '-count=1', './api-tests/...'], 300)
                else:
                    runner.execute(['apih', '--parallelism', '0', str(suite)], 300)
                runner.stop_server()
            if legacy:
                return True
            if not list(counters.glob('covmeta.*')) or not list(counters.glob('covcounters.*')):
                raise ValueError('missing instrumented server metadata/counters')
            profile = directory / 'coverage.out'
            runner.execute([coverage.GO, 'tool', 'covdata', 'textfmt', '-i=' + str(counters), '-o=' + str(profile)])
        finally:
            # Preserve the original failure; still expose any cleanup failure.
            active_error = sys.exc_info()[0] is not None
            try:
                runner.cleanup()
            except Exception as error:
                print(f'cleanup failed: {error}', file=sys.stderr)
                if not active_error:
                    raise
            finally:
                runner.database = None
    return coverage.report(metadata, profile, directory, integration=True)


def interrupted(signum, _frame):
    raise InterruptedError(signum, 'integration run interrupted')


def main():
    legacy = '--legacy' in sys.argv[1:]
    for sig in (signal.SIGINT, signal.SIGTERM):
        signal.signal(sig, interrupted)
    with coverage.fresh_run('legacy' if legacy else 'api') as directory:
        prerequisites()
        port = int(os.environ.get('API_TEST_PORT', '19080'))
        if not 1 <= port <= 65535:
            raise ValueError('invalid API_TEST_PORT')
        free_port(port)
        runner = Runner(directory)
        try:
            return 0 if campaign(runner, port, legacy=legacy) else 1
        except BaseException:
            for name in ('coverage.out', 'report.txt', 'result.json'):
                (directory / name).unlink(missing_ok=True)
            raise
        finally:
            runner.log.close()
            coverage.provenance(directory, {
                'commands': runner.commands,
                'apih': coverage.command([coverage.GO, 'version', '-m', shutil.which('apih')], capture_output=True).stdout,
                'server': 'mch-server -port <owned-port> -db <private-socket-URL>',
            })


if __name__ == '__main__':
    try:
        sys.exit(main())
    except subprocess.CalledProcessError as error:
        print(f'API campaign command failed (exit {error.returncode}); see .coverage/api/runner.log', file=sys.stderr)
        sys.exit(error.returncode if error.returncode > 0 else 1)
    except InterruptedError as error:
        sys.exit(128 + error.errno)
    except (OSError, RuntimeError, ValueError, subprocess.TimeoutExpired) as error:
        print(f'API campaign failed: {error}', file=sys.stderr)
        sys.exit(1)
