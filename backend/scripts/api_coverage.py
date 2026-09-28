"""Run the complete APIHydra suite against an owned instrumented server and DB."""
import os
import json
import hashlib
from datetime import datetime, timezone
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
        self.database_state = 'unowned'
        self.socket_directory = None
        self.server_pid = None
        self.cleaning_up = False
        self.cleanup_evidence_error = None
        self.evidence = {'status': 'incomplete', 'phases': [], 'postconditions': [], 'events': [],
                         'readiness': 'initial HTTP health probes synchronize startup and can contribute counters'}
        self.log = (directory / 'runner.log').open('w')

    def save_evidence(self):
        try:
            (self.directory / 'campaign.json').write_text(json.dumps(self.evidence, indent=2))
        except OSError as error:
            print(f'campaign evidence persistence failed: {error}', file=sys.stderr)
            if not self.cleaning_up:
                raise
            # Even nested stop commands must run before persistence can fail cleanup.
            self.cleanup_evidence_error = self.cleanup_evidence_error or error

    def action(self, name, function, record=None):
        if record is None:
            record = {'name': name}
            self.evidence['events'].append(record)
        record.update(status='running', start=datetime.now(timezone.utc).isoformat())
        self.save_evidence()
        try:
            result = function()
        except BaseException as error:
            record.update(status='failed', error=str(error), exit=getattr(error, 'returncode',
                          128 + error.errno if isinstance(error, InterruptedError) else None))
            raise
        else:
            record.update(status='passed', exit=0)
            return result
        finally:
            record['end'] = datetime.now(timezone.utc).isoformat()
            active_error = sys.exc_info()[0] is not None
            try:
                self.save_evidence()
            except OSError:
                if not active_error:
                    raise

    def execute(self, args, timeout=120):
        record = {'name': 'command', 'command': [str(arg) for arg in args], 'timeout': timeout}
        self.evidence['events'].append(record)
        return self.action('command', lambda: self._execute(args, timeout), record)

    def _execute(self, args, timeout):
        # DB URLs are deliberately not included in provenance/logged commands.
        self.commands.append([str(arg) for arg in args])
        process = None
        try:
            with deferred_signals() as previous:
                process = subprocess.Popen(args, cwd=BACKEND, env=self.env, stdout=self.log,
                                           stderr=subprocess.STDOUT, start_new_session=True,
                                           preexec_fn=lambda: signal.pthread_sigmask(signal.SIG_SETMASK, previous))
            code = process.wait(timeout=timeout)
        except BaseException:
            if process is not None:
                handlers = {sig: signal.signal(sig, signal.SIG_IGN)
                            for sig in (signal.SIGINT, signal.SIGTERM)}
                try:
                    # Preserve the original interruption/timeout even if emergency
                    # termination or the bounded reap also fails. Still try both.
                    for name, finish in (
                            ('kill', lambda: os.killpg(process.pid, signal.SIGKILL)),
                            ('reap', lambda: process.wait(timeout=5))):
                        try:
                            finish()
                        except ProcessLookupError:
                            pass
                        except BaseException as secondary:
                            print(f'owned command emergency {name} failed: {secondary}', file=sys.stderr)
                            self.evidence['events'].append({
                                'name': 'command-emergency-' + name, 'status': 'failed',
                                'error': str(secondary), 'pid': process.pid})
                finally:
                    for sig, handler in handlers.items():
                        signal.signal(sig, handler)
            raise
        if code:
            raise subprocess.CalledProcessError(code, args)

    def assert_server(self):
        if self.server is None or self.server.pid != self.server_pid or self.server.poll() is not None:
            raise RuntimeError('original server is no longer live')

    def start_database(self):
        self.database_state = 'uncertain'
        def start():
            self.execute(['pg_ctl', '-D', str(self.database), '-l', str(self.directory / 'postgres.log'),
                          '-o', shlex.join(['-k', self.socket_directory, '-h', '', '-p', '15432']),
                          '-w', '-t', '10', 'start'], 15)
            if not (self.database / 'postmaster.pid').is_file():
                raise RuntimeError('database start succeeded without PID file')
            self.database_state = 'running'
        self.action('database-start', start)

    def stop_database(self, mode='fast'):
        self.database_state = 'uncertain'
        def stop():
            self.execute(['pg_ctl', '-D', str(self.database), '-m', mode, '-w', '-t', '10', 'stop'], 15)
            if (self.database / 'postmaster.pid').exists():
                raise RuntimeError('database stop left PID file')
            self.database_state = 'stopped'
        self.action('database-stop-' + mode, stop)

    def sql(self, filename):
        self.execute(['psql', '-X', '-h', self.socket_directory, '-p', '15432', '-U', 'postgres',
                      '-d', 'postgres', '-v', 'ON_ERROR_STOP=1', '-f', str(filename)])

    def phase(self, name, suite):
        record = next(item for item in self.evidence['phases'] if item['name'] == name)
        record['command'] = ['apih', '--parallelism', '0', str(suite / name)]
        record['server_pid'] = self.server_pid
        record['binary_sha256'] = self.evidence['binary_sha256']
        def run():
            self.assert_server()
            self.execute(record['command'], 300)
            self.assert_server()
        self.action(name, run, record)

    def postcondition(self, name, suite):
        record = next(item for item in self.evidence['postconditions'] if item['name'] == name)
        def run():
            self.assert_server()
            self.sql(suite / (name + '-postconditions.sql'))
            self.assert_server()
        self.action(name + '-sql-postconditions', run, record)

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
            server.wait(timeout=5)
            self.server = None
            raise RuntimeError('owned server did not shut down normally') from None
        self.server = None
        if code != 0:
            raise RuntimeError(f'owned server exited {code}')

    def cleanup(self):
        handlers = {sig: signal.signal(sig, signal.SIG_IGN) for sig in (signal.SIGINT, signal.SIGTERM)}
        self.cleaning_up = True
        self.cleanup_evidence_error = None
        try:
            self.action('cleanup', self._cleanup)
            if self.cleanup_evidence_error is not None:
                raise self.cleanup_evidence_error
        finally:
            self.cleaning_up = False
            for sig, handler in handlers.items():
                signal.signal(sig, handler)

    def _cleanup(self):
        failure = None
        try:
            self.stop_server()
        except Exception as error:
            failure = error
        if self.database is not None:
            # Ownership survives intentional outage and partial restarts.
            if self.database_state != 'stopped' or (self.database / 'postmaster.pid').exists():
                try:
                    self.stop_database()
                except Exception as error:
                    failure = failure or error
                    try:
                        self.stop_database('immediate')
                    except Exception as fallback:
                        print(f'owned database emergency cleanup failed: {fallback}', file=sys.stderr)
            if self.database_state == 'stopped' and not (self.database / 'postmaster.pid').exists():
                self.database = None
        if failure:
            raise failure


@contextlib.contextmanager
# The runner is single-threaded. Children restore the original mask before exec;
# only the parent defers delivery until the owned handle has been assigned.
def deferred_signals():
    previous = signal.pthread_sigmask(signal.SIG_BLOCK, {signal.SIGINT, signal.SIGTERM})
    try:
        yield previous
    finally:
        signal.pthread_sigmask(signal.SIG_SETMASK, previous)


@contextlib.contextmanager
def private_cluster(runner):
    scratch = tempfile.mkdtemp(prefix='mch-pg-')
    try:
        yield scratch
    finally:
        if runner.server is not None or runner.database is not None or (Path(scratch) / 'data/postmaster.pid').exists():
            # Never unlink a cluster whose shutdown failed; retain diagnostics.
            print(f'owned cluster shutdown unconfirmed; retained at {scratch}', file=sys.stderr)
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
    shutil.copytree(source, suite, symlinks=True)
    # Parse YAML, including quoted/escaped keys, flow mappings and aliases.
    # Reuse the backend's existing YAML module; no Python dependency is needed.
    manifest = coverage.command([coverage.GO, 'run', './scripts/validate-apih-suite', str(suite)],
                                capture_output=True).stdout
    (directory / 'suite-manifest.json').write_text(manifest)
    json.loads(manifest)  # Fail before setup if the helper did not produce JSON.
    root = suite / 'root.yaml'
    text = root.read_text()
    expected = 'base_url: http://127.0.0.1:19080'
    if text.count(expected) != 1:
        raise ValueError('suite root must declare the canonical local base_url')
    root.write_text(text.replace(expected, f'base_url: http://127.0.0.1:{port}'))
    # Source hashes above describe original YAML. Record the actual copied inputs
    # too, including the port-substituted root, fixtures and SQL postconditions.
    (directory / 'suite-inputs.json').write_text(json.dumps({
        str(path.relative_to(suite)): hashlib.sha256(path.read_bytes()).hexdigest()
        for path in suite.rglob('*') if path.is_file()}, indent=2))
    return suite


def campaign(runner, port, legacy=False):
    directory = runner.directory
    runner.evidence['status'] = 'incomplete'
    runner.evidence['phases'] = [] if legacy else [
        {'name': name, 'selection': name, 'status': 'not-reached',
         'start': None, 'end': None, 'exit': None, 'error': None} for name in ('normal', 'outage', 'recovery')]
    runner.evidence['postconditions'] = [] if legacy else [
        {'name': name, 'status': 'not-reached', 'start': None, 'end': None, 'exit': None, 'error': None}
        for name in ('normal', 'outage', 'recovery')]
    runner.save_evidence()
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
    runner.evidence['binary_sha256'] = hashlib.sha256(binary.read_bytes()).hexdigest()
    counters = directory / 'counters'
    counters.mkdir()
    runner.env['GOCOVERDIR'] = str(counters)
    # Private Unix socket only: never connect to or stop the user's port 5432.
    with private_cluster(runner) as scratch:
        data = Path(scratch) / 'data'
        runner.execute(['initdb', '-D', str(data), '-U', 'postgres', '--auth=trust', '--no-locale'])
        try:
            runner.database = data
            runner.socket_directory = scratch
            runner.database_state = 'stopped'
            runner.start_database()
            for sql_file in [BACKEND.parent / 'db/init.sql', BACKEND.parent / 'db/seed.sql',
                             BACKEND / 'apih-tests/fixtures.sql']:
                runner.evidence.setdefault('database_inputs', {})[str(sql_file)] = hashlib.sha256(sql_file.read_bytes()).hexdigest()
                runner.sql(sql_file)
            db_url = 'postgresql://postgres@/postgres?' + urlencode(
                {'host': scratch, 'port': '15432', 'sslmode': 'disable'}, quote_via=quote)
            with (directory / 'server.log').open('w') as server_log:
                with deferred_signals() as previous:
                    runner.server = subprocess.Popen([str(binary), '-port', str(port), '-db', db_url],
                        cwd=BACKEND, env=runner.env, stdout=server_log, stderr=subprocess.STDOUT,
                        preexec_fn=lambda: signal.pthread_sigmask(signal.SIG_SETMASK, previous))
                runner.server_pid = runner.server.pid
                runner.evidence['server_pid'] = runner.server_pid
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
                    runner.phase('normal', suite)
                    runner.postcondition('normal', suite)
                    runner.assert_server()
                    runner.stop_database()
                    runner.phase('outage', suite)
                    runner.start_database()
                    runner.postcondition('outage', suite)
                    runner.phase('recovery', suite)
                    runner.postcondition('recovery', suite)
                runner.assert_server()
                runner.action('server-stop', runner.stop_server)
        finally:
            # Preserve the original failure; still expose any cleanup failure.
            active_error = sys.exc_info()[0] is not None
            try:
                runner.cleanup()
            except Exception as error:
                print(f'cleanup failed: {error}', file=sys.stderr)
                if not active_error:
                    raise
    runner.evidence['status'] = 'complete'
    runner.save_evidence()
    if legacy:
        return True
    if not list(counters.glob('covmeta.*')) or not list(counters.glob('covcounters.*')):
        raise ValueError('missing instrumented server metadata/counters')
    profile = directory / 'coverage.out'
    runner.execute([coverage.GO, 'tool', 'covdata', 'textfmt', '-i=' + str(counters), '-o=' + str(profile)])
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
            try:
                passed = campaign(runner, port, legacy=legacy)
            finally:
                active_error = sys.exc_info()[0] is not None
                if active_error:
                    runner.evidence['status'] = 'incomplete'
                try:
                    coverage.provenance(directory, {
                        'commands': runner.commands,
                        'campaign': runner.evidence,
                        'apih_sha256': hashlib.sha256(Path(shutil.which('apih')).read_bytes()).hexdigest(),
                        'apih': coverage.command([coverage.GO, 'version', '-m', shutil.which('apih')], capture_output=True).stdout,
                        'server': 'mch-server -port <owned-port> -db <private-socket-URL>',
                    })
                except Exception as error:
                    print(f'provenance failed: {error}', file=sys.stderr)
                    if not active_error:
                        raise
            return 0 if passed else 1
        except BaseException:
            runner.evidence['status'] = 'incomplete'
            for name in ('coverage.out', 'report.txt', 'result.json'):
                (directory / name).unlink(missing_ok=True)
            # Persistence was already reported; retain the primary command/signal error.
            with contextlib.suppress(OSError):
                runner.save_evidence()
            raise
        finally:
            runner.log.close()


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
