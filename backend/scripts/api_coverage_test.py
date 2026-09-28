import contextlib
import io
import json
import os
from pathlib import Path
import signal
import shlex
import socket
import subprocess
import sys
import tempfile
import unittest
from urllib.parse import parse_qs, urlsplit
from unittest.mock import Mock, patch

import api_coverage as api
import coverage


class Server:
    def __init__(self, code=None):
        self.code=code
        self.signals=[]
    def poll(self): return self.code
    def send_signal(self, sig):
        self.signals.append(sig)
        self.code=0
    def wait(self, timeout): return self.code
    def kill(self): self.code=-9


class APICoverageTest(unittest.TestCase):
    def setUp(self):
        self.temp=tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.directory=Path(self.temp.name)
        self.runner=api.Runner(self.directory)
        self.addCleanup(self.runner.log.close)

    def campaign(self, fail=None, code=17, server_code=None, counters=True, report=True, legacy=False, ready=True):
        self.calls=[]
        self.server=Server(server_code)
        self.meta=dict(packages=[],blocks={},sources={})
        def execute(args, timeout=120):
            self.calls.append(args)
            if args[0]==fail or (fail=='conversion' and 'textfmt' in args):
                raise subprocess.CalledProcessError(code,args)
            if args[0]=='apih' and counters:
                (self.directory/'counters/covmeta.fake').touch()
                (self.directory/'counters/covcounters.fake').touch()
        with patch.object(self.runner,'execute',side_effect=execute), \
             patch.object(coverage,'inventory',return_value=self.meta), \
             patch.object(coverage,'command',return_value=subprocess.CompletedProcess([],0,'')), \
             patch.object(api,'copy_suite',return_value=self.directory/'suite'), \
             patch.object(api.subprocess,'Popen',return_value=self.server), \
             patch.object(api.urllib.request,'urlopen') as request, \
             patch.object(coverage,'report',return_value=report) as reporting:
            request.return_value.__enter__.return_value.status=200 if ready else 503
            result=api.campaign(self.runner,19080,legacy=legacy)
            self.reporting=reporting
            return result

    def test_suite_exit_codes_preserved_and_owned_resources_stop(self):
        for code in [101,102,103,7]:
            with self.subTest(code=code):
                if (self.directory/'counters').exists():
                    import shutil
                    shutil.rmtree(self.directory/'counters')
                with self.assertRaises(subprocess.CalledProcessError) as raised:
                    self.campaign('apih',code)
                self.assertEqual(raised.exception.returncode,code)
                self.assertEqual(self.server.signals,[signal.SIGTERM])
                self.assertEqual(self.calls[-1][-1],'stop')
                self.assertFalse(any('textfmt' in args for args in self.calls))
                self.assertIsNone(self.runner.database)

    def test_success_stops_before_conversion_and_uses_private_cluster(self):
        self.assertTrue(self.campaign())
        self.assertEqual(self.server.signals,[signal.SIGTERM])
        apih=next(args for args in self.calls if args[0]=='apih')
        self.assertEqual(apih[1:3],['--parallelism','0'])
        build=self.calls[0]
        self.assertIn('-cover',build)
        self.assertIn('-covermode=atomic',build)
        self.assertIn('-coverpkg='+','.join(coverage.PATTERNS),build)
        psql=[args for args in self.calls if args[0]=='psql']
        self.assertEqual(len(psql),2)
        for args in psql:
            self.assertIn('ON_ERROR_STOP=1',args)
            self.assertNotIn('5432',args)
        self.assertTrue(Path(self.runner.env['GOCOVERDIR']).is_absolute())
        self.assertTrue(self.reporting.call_args.kwargs['integration'])
        self.assertEqual(self.calls[-1][-1],'stop')

    def test_sql_startup_and_conversion_failures_remain_visible(self):
        for failed in ['psql','pg_ctl','conversion']:
            with self.subTest(failed=failed):
                import shutil
                shutil.rmtree(self.directory/'counters',ignore_errors=True)
                with self.assertRaises(subprocess.CalledProcessError): self.campaign(failed)
                if failed!='conversion':
                    self.assertFalse(any(args[0]=='apih' for args in self.calls))
                self.assertEqual(self.calls[-1][-1],'stop')

    def test_server_crash_and_missing_counters_rejected(self):
        with self.assertRaisesRegex(RuntimeError,'startup failed'):
            self.campaign(server_code=1)
        self.assertFalse(any(args[0]=='apih' for args in self.calls))
        import shutil
        shutil.rmtree(self.directory/'counters')
        with self.assertRaisesRegex(ValueError,'metadata/counters'):
            self.campaign(counters=False)
        self.assertEqual(self.calls[-1][-1],'stop')

    def test_legacy_output_never_enters_api_profile(self):
        self.assertTrue(self.campaign(legacy=True))
        self.assertFalse(any(args[0]=='apih' or 'textfmt' in args for args in self.calls))
        self.assertTrue(any('./api-tests/...' in args for args in self.calls))
        self.reporting.assert_not_called()
        host = parse_qs(urlsplit(self.runner.env['API_TEST_DB_URL']).query)['host'][0]
        self.assertIn('/mch-pg-', host)

    def test_private_socket_handles_spaces_and_query_characters(self):
        with tempfile.TemporaryDirectory(prefix='mch space & # ') as scratch:
            with patch.object(api, 'private_cluster', return_value=contextlib.nullcontext(scratch)):
                self.assertTrue(self.campaign(legacy=True))
            startup = next(args for args in self.calls if args[0] == 'pg_ctl' and args[-1] == 'start')
            options = shlex.split(startup[startup.index('-o') + 1])
            self.assertEqual(options, ['-k', scratch, '-h', '', '-p', '15432'])
            params = parse_qs(urlsplit(self.runner.env['API_TEST_DB_URL']).query)
            self.assertEqual(params['host'], [scratch])
            # pgx parses URL parameters with PathUnescape, which retains '+'.
            self.assertNotIn('+', self.runner.env['API_TEST_DB_URL'])
            self.assertIn('%20', self.runner.env['API_TEST_DB_URL'])

    def test_readiness_timeout_cleans_up_without_running_suite(self):
        with patch.object(api.time,'monotonic',side_effect=[0,100]):
            with self.assertRaisesRegex(RuntimeError,'readiness timed out'):
                self.campaign(ready=False)
        self.assertFalse(any(args[0]=='apih' for args in self.calls))
        self.assertEqual(self.server.signals,[signal.SIGTERM])
        self.assertEqual(self.calls[-1][-1],'stop')

    def test_unsuccessful_server_exit_cannot_supply_coverage(self):
        server=Mock()
        server.poll.return_value=None
        server.wait.return_value=1
        self.runner.server=server
        with self.assertRaisesRegex(RuntimeError,'exited 1'): self.runner.stop_server()

    def test_interrupted_wait_preserves_server_for_final_cleanup(self):
        server = Mock()
        server.poll.return_value = None
        server.wait.side_effect = [InterruptedError(signal.SIGTERM, 'interrupt'), 0]
        self.runner.server = server
        with self.assertRaises(InterruptedError):
            self.runner.stop_server()
        self.assertIs(self.runner.server, server)
        self.runner.cleanup()
        self.assertIsNone(self.runner.server)
        self.assertEqual(server.wait.call_count, 2)

    def test_valid_below_threshold_result_is_failure(self):
        self.assertFalse(self.campaign(report=False))

    def test_occupied_non_http_service_survives(self):
        with socket.socket(socket.AF_INET,socket.SOCK_STREAM) as owned:
            owned.bind(('127.0.0.1',0)); owned.listen()
            port=owned.getsockname()[1]
            with patch.dict(os.environ,{'API_TEST_PORT':str(port)}), \
                 patch.object(coverage,'BACKEND',self.directory), patch.object(api,'prerequisites'), \
                 patch.object(api,'campaign') as campaign:
                with self.assertRaises(OSError): api.main()
                campaign.assert_not_called()
            with socket.create_connection(('127.0.0.1',port),timeout=1): pass

    def test_recently_closed_owned_listener_can_be_reused(self):
        with socket.socket(socket.AF_INET6,socket.SOCK_STREAM) as listener:
            listener.setsockopt(socket.SOL_SOCKET,socket.SO_REUSEADDR,1)
            listener.bind(('::',0)); listener.listen()
            port=listener.getsockname()[1]
            client=socket.create_connection(('127.0.0.1',port))
            accepted,_=listener.accept()
            accepted.close()
            client.close()
        api.free_port(port)

    def test_missing_prerequisite_prevents_setup(self):
        with patch.object(api.shutil,'which',side_effect=lambda name: None if name=='apih' else '/fake/'+name):
            with self.assertRaisesRegex(RuntimeError,'missing prerequisite: apih'):
                api.prerequisites()

    def test_failed_campaign_removes_partial_or_stale_success_reports(self):
        def fail(runner, port, legacy=False):
            for name in ('coverage.out','report.txt','result.json'):
                (runner.directory/name).write_text('partial or invalid')
            raise subprocess.CalledProcessError(101,['apih'])
        with patch.object(coverage,'BACKEND',self.directory), patch.object(api,'prerequisites'), \
             patch.object(api,'free_port'), patch.object(api,'campaign',side_effect=fail), \
             patch.object(coverage,'provenance'), patch.object(coverage,'command',return_value=subprocess.CompletedProcess([],0,'')):
            with self.assertRaises(subprocess.CalledProcessError) as raised: api.main()
            self.assertEqual(raised.exception.returncode,101)
        for name in ('coverage.out','report.txt','result.json'):
            self.assertFalse((self.directory/'.coverage/api'/name).exists())

    def test_execute_preserves_actual_external_exit_codes(self):
        for code in [101,102,103,23]:
            with self.assertRaises(subprocess.CalledProcessError) as raised:
                self.runner.execute([sys.executable,'-c',f'raise SystemExit({code})'])
            self.assertEqual(raised.exception.returncode,code)

    def test_timeout_kills_owned_child_and_shutdown_is_bounded(self):
        with self.assertRaises(subprocess.TimeoutExpired):
            self.runner.execute([sys.executable,'-c','import time; time.sleep(30)'],timeout=.02)
        server=Mock()
        server.poll.return_value=None
        server.wait.side_effect=[subprocess.TimeoutExpired('server',15),-9]
        self.runner.server=server
        with self.assertRaisesRegex(RuntimeError,'normally'): self.runner.stop_server()
        server.kill.assert_called_once()
        self.assertIsNone(self.runner.server)

    def test_interrupt_cleans_owned_command_group(self):
        process=Mock(pid=987654)
        process.wait.side_effect=[InterruptedError(signal.SIGTERM,'test'),-9]
        with patch.object(api.subprocess,'Popen',return_value=process), patch.object(api.os,'killpg') as kill:
            with self.assertRaises(InterruptedError): self.runner.execute(['fake'])
            kill.assert_called_once_with(987654,signal.SIGKILL)
        for sig in [signal.SIGINT,signal.SIGTERM]:
            with self.assertRaises(InterruptedError) as raised: api.interrupted(sig,None)
            self.assertEqual(raised.exception.errno,sig)

    def test_suite_rejects_debug_and_copies_complete_tree(self):
        backend=self.directory/'backend'; suite=backend/'apih-tests'; suite.mkdir(parents=True)
        (suite/'root.yaml').write_text('base_url: http://127.0.0.1:19080')
        (suite/'steps.yaml').write_text('debug: true')
        with patch.object(api,'BACKEND',backend), patch.object(coverage,'command',
                side_effect=subprocess.CalledProcessError(1,['validator'])):
            with self.assertRaises(subprocess.CalledProcessError): api.copy_suite(self.directory,19876)
        import shutil
        shutil.rmtree(self.directory/'suite')
        (suite/'steps.yaml').write_text('expected_status: 200')
        with patch.object(api,'BACKEND',backend), patch.object(coverage,'command') as validate:
            copied=api.copy_suite(self.directory,19876)
        self.assertEqual(validate.call_args.args[0],
                         [coverage.GO,'run','./scripts/validate-apih-suite',str(copied)])
        self.assertIn('19876',(copied/'root.yaml').read_text())
        self.assertEqual((copied/'steps.yaml').read_text(),'expected_status: 200')

    def test_environment_ignores_external_pg_settings_and_uses_private_cache(self):
        with patch.dict(os.environ,{'PGHOST':'unrelated','PGDATABASE':'production'}):
            runner=api.Runner(self.directory)
        runner.log.close()
        self.assertNotIn('PGHOST',runner.env)
        self.assertNotIn('PGDATABASE',runner.env)
        self.assertEqual(runner.env['XDG_CACHE_HOME'],str(self.directory/'cache'))
