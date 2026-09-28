import contextlib
import io
import json
import os
from pathlib import Path
import signal
import socket
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import Mock, patch

import api_coverage as api
import coverage


class APICoverageTest(unittest.TestCase):
    def setUp(self):
        temporary = tempfile.TemporaryDirectory()
        self.addCleanup(temporary.cleanup)
        self.root = Path(temporary.name)
        self.server = Mock(pid=123456)
        self.server.poll.return_value = None
        self.server.wait.return_value = 0
        self.commands = []
        self.envs = []

    def campaign(self, failure=None, code=101, counters=True, stop_error=None):
        metadata = dict(packages=['m/p'], blocks={'m/p/a.go:1.1,2.1':1, 'm/p/a.go:3.1,4.1':9}, sources={})
        def execute(args, env, output, timeout=120):
            self.commands.append(args)
            self.envs.append(env)
            if args[0] == 'apih':
                self.assertIsNone(output)
                print('APIHydra standard output')
                if counters:
                    (self.root/'counters/covmeta.fake').touch()
                    (self.root/'counters/covcounters.fake').touch()
            if failure in args:
                raise subprocess.CalledProcessError(code, args)
            if 'build' in args:
                (self.root/'mch-server').write_bytes(b'covered binary')
            if 'textfmt' in args:
                self.server.wait.assert_called_once_with(timeout=15)
                (self.root/'coverage.out').write_text(
                    'mode: atomic\nm/p/a.go:1.1,2.1 1 1\nm/p/a.go:3.1,4.1 9 0\n')
        with patch.object(api, 'free_port'), patch.object(api, 'copy_suite', return_value=self.root/'suite'), \
             patch.object(coverage, 'inventory', return_value=metadata), \
             patch.object(coverage, 'command', return_value=subprocess.CompletedProcess([], 0, 'm/p\n')), \
             patch.object(coverage, 'provenance') as provenance, \
             patch.object(api, 'execute', side_effect=execute), \
             patch.object(api, 'spawn', return_value=self.server), \
             patch.object(api, 'await_server'), \
             patch.dict(os.environ, APIH='apih', DATABASE_URL='postgres://test-secret@existing/database'), \
             contextlib.redirect_stdout(io.StringIO()) as output:
            if stop_error:
                self.server.wait.side_effect = stop_error
            api.campaign(self.root, 19080)
            self.provenance = provenance.call_args.args[1]
            return output.getvalue()

    def test_reports_real_counts_without_threshold_and_only_owns_server(self):
        output = self.campaign()
        self.assertIn('APIHydra standard output\n\npackage covered/total percent', output)
        self.assertIn('TOTAL 1/10 10.0000%', output)
        self.server.send_signal.assert_called_once_with(signal.SIGTERM)
        self.assertEqual(len(self.commands), 3)
        build, suite, convert = self.commands
        self.assertIn('-cover', build)
        self.assertIn('-covermode=atomic', build)
        self.assertIn('-coverpkg=' + ','.join(coverage.PATTERNS), build)
        self.assertEqual(suite, ['apih', str(self.root/'suite')])
        self.assertIn('textfmt', convert)
        self.assertEqual(self.envs[0]['DATABASE_URL'], 'postgres://test-secret@existing/database')
        self.assertEqual(self.envs[0]['GOCOVERDIR'], str(self.root/'counters'))
        self.assertNotIn('test-secret', json.dumps(self.provenance))
        result = json.loads((self.root/'result.json').read_text())
        self.assertFalse(result['threshold_enforced'])
        self.assertEqual((result['covered'], result['total']), (1, 10))

    def test_apih_failures_preserve_codes_cleanup_and_do_not_report(self):
        for code in [101, 102, 103, 17]:
            with self.subTest(code=code), tempfile.TemporaryDirectory() as directory:
                self.root = Path(directory)
                self.server.reset_mock()
                self.commands = []
                with self.assertRaises(subprocess.CalledProcessError) as raised:
                    self.campaign('apih', code)
                self.assertEqual(raised.exception.returncode, code)
                self.server.send_signal.assert_called_once_with(signal.SIGTERM)
                self.assertFalse(any('textfmt' in args for args in self.commands))
                self.assertFalse((self.root/'result.json').exists())

    def test_build_failure_does_not_spawn_or_report(self):
        with self.assertRaises(subprocess.CalledProcessError):
            self.campaign('build', 23)
        self.server.send_signal.assert_not_called()
        self.assertEqual(len(self.commands), 1)
        self.assertFalse((self.root/'result.json').exists())

    def test_missing_counters_and_non_graceful_shutdown_block_reporting(self):
        with self.assertRaisesRegex(ValueError, 'missing instrumented'):
            self.campaign(counters=False)
        self.assertFalse((self.root/'result.json').exists())
        with tempfile.TemporaryDirectory() as directory:
            self.root = Path(directory)
            timeout = subprocess.TimeoutExpired('server', 15)
            with self.assertRaisesRegex(RuntimeError, 'gracefully'):
                self.campaign(stop_error=[timeout, -9])
            self.server.kill.assert_called_once()
            self.assertFalse((self.root/'result.json').exists())

    def test_failed_run_removes_all_report_artifacts(self):
        def fail(directory, port):
            for name in ['coverage.out', 'report.txt', 'result.json']:
                (directory/name).write_text('invalid')
            raise subprocess.CalledProcessError(101, ['apih'])
        with patch.object(coverage, 'BACKEND', self.root), patch.object(api, 'campaign', side_effect=fail):
            with self.assertRaises(subprocess.CalledProcessError):
                api.main()
        for name in ['coverage.out', 'report.txt', 'result.json']:
            self.assertFalse((self.root/'.coverage/api'/name).exists())

    def test_copy_changes_only_private_suite_url_and_records_inputs(self):
        source = self.root/'backend/apih-tests'
        source.mkdir(parents=True)
        root = source/'root.yaml'
        original = 'app: apihydra\nkind: root\nspec:\n  base_url: http://127.0.0.1:8080\n  timeout: 5\n'
        root.write_text(original)
        output = self.root/'measurement'
        output.mkdir()
        with patch.object(coverage, 'BACKEND', source.parent), \
             patch.object(coverage, 'command', return_value=subprocess.CompletedProcess([], 0, '{}')):
            suite = api.copy_suite(output, 19876)
        self.assertEqual(root.read_text(), original)
        self.assertIn('http://127.0.0.1:19876', (suite/'root.yaml').read_text())
        self.assertIn('root.yaml', json.loads((output/'suite-inputs.json').read_text()))

    def test_occupied_port_and_crashed_server_rejected(self):
        with socket.socket(socket.AF_INET6, socket.SOCK_STREAM) as sock:
            sock.bind(('::', 0))
            sock.listen()
            with self.assertRaises(OSError):
                api.free_port(sock.getsockname()[1])
        self.server.poll.return_value = 7
        with self.assertRaisesRegex(RuntimeError, 'exited before readiness'):
            api.await_server(self.server, 19080)
        with self.assertRaises(TimeoutError):
            api.await_server(self.server, 19080, timeout=0)
        for port in [0, -1, 65536]:
            with self.assertRaises(ValueError):
                api.free_port(port)

    def test_subprocess_signal_mask_and_exit_codes(self):
        with (self.root/'command.log').open('w') as output:
            api.execute([sys.executable, '-c',
                'import signal; raise SystemExit(int(signal.SIGTERM in signal.pthread_sigmask(signal.SIG_BLOCK, set())))'],
                os.environ.copy(), output)
            with self.assertRaises(subprocess.CalledProcessError) as raised:
                api.execute([sys.executable, '-c', 'raise SystemExit(101)'], os.environ.copy(), output)
            self.assertEqual(raised.exception.returncode, 101)
            with self.assertRaises(subprocess.TimeoutExpired):
                api.execute([sys.executable, '-c', 'import time; time.sleep(30)'], os.environ.copy(), output, timeout=.05)
