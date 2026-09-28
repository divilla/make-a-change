import contextlib
import errno
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
        self.pid=12345
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

    def campaign(self, fail=None, code=17, server_code=None, counters=True, report=True, legacy=False, ready=True, hook=None):
        self.calls=[]
        self.server=Server(server_code)
        self.meta=dict(packages=[],blocks={},sources={})
        def execute(args, timeout=120):
            self.calls.append(args)
            if hook: hook(args)
            if 'build' in args: (self.directory/'mch-server').write_bytes(b'covered binary')
            if args[0]=='pg_ctl':
                data=Path(args[args.index('-D')+1]); data.mkdir(exist_ok=True)
                if args[-1]=='start': (data/'postmaster.pid').write_text('owned fake pid')
                if args[-1]=='stop': (data/'postmaster.pid').unlink(missing_ok=True)

            if args[0]==fail or (fail=='conversion' and 'textfmt' in args) or (fail=='fixtures' and str(args[-1]).endswith('/fixtures.sql')):
                raise subprocess.CalledProcessError(code,args)
            if args[0]=='apih' and counters:
                (self.directory/'counters/covmeta.fake').touch()
                (self.directory/'counters/covcounters.fake').touch()
        with patch.object(self.runner,'_execute',side_effect=execute), \
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
        self.assertEqual([Path(args[-1]).name for args in psql], ['init.sql','seed.sql','fixtures.sql','normal-postconditions.sql','outage-postconditions.sql','recovery-postconditions.sql'])
        for args in psql:
            self.assertIn('ON_ERROR_STOP=1',args)
            self.assertNotIn('5432',args)
        self.assertTrue(Path(self.runner.env['GOCOVERDIR']).is_absolute())
        self.assertTrue(self.reporting.call_args.kwargs['integration'])
        self.assertIn('textfmt', self.calls[-1])

    def test_fixture_failure_stops_before_server_and_suite(self):
        with self.assertRaises(subprocess.CalledProcessError) as raised:
            self.campaign('fixtures',23)
        self.assertEqual(raised.exception.returncode,23)
        self.assertFalse(any(args[0]=='apih' for args in self.calls))
        self.assertEqual(self.server.signals,[])
        self.assertEqual(self.calls[-1][-1],'stop')
        self.assertIsNone(self.runner.database)

    def test_sql_startup_and_conversion_failures_remain_visible(self):
        for failed in ['psql','pg_ctl','conversion']:
            with self.subTest(failed=failed):
                import shutil
                shutil.rmtree(self.directory/'counters',ignore_errors=True)
                with self.assertRaises(subprocess.CalledProcessError): self.campaign(failed)
                if failed!='conversion':
                    self.assertFalse(any(args[0]=='apih' for args in self.calls))
                self.assertEqual(self.calls[-1][-1] if failed!='conversion' else 'stop','stop')

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

    def test_emergency_command_errors_preserve_primary_and_restore_signals(self):
        for secondary in [subprocess.TimeoutExpired('owned', 5),
                          InterruptedError(signal.SIGINT, 'second interrupt'),
                          PermissionError('cannot signal group')]:
            with self.subTest(secondary=type(secondary).__name__):
                primary = InterruptedError(signal.SIGTERM, 'original interrupt')
                process = Mock(pid=987654)
                handlers = {sig: signal.getsignal(sig) for sig in (signal.SIGINT, signal.SIGTERM)}
                calls = []
                def wait(timeout):
                    calls.append(timeout)
                    if len(calls) == 1:
                        raise primary
                    for sig in handlers:
                        self.assertEqual(signal.getsignal(sig), signal.SIG_IGN)
                    if not isinstance(secondary, PermissionError):
                        raise secondary
                    return -9
                process.wait.side_effect = wait
                errors = io.StringIO()
                with patch.object(api.subprocess, 'Popen', return_value=process), \
                     patch.object(api.os, 'killpg', side_effect=secondary if isinstance(secondary, PermissionError) else None), \
                     contextlib.redirect_stderr(errors):
                    with self.assertRaises(InterruptedError) as raised:
                        self.runner.execute(['owned'], timeout=30)
                self.assertIs(raised.exception, primary)
                self.assertEqual(calls, [30, 5])
                self.assertIn(str(secondary), errors.getvalue())
                self.assertTrue(any(event.get('error') == str(secondary)
                                    for event in self.runner.evidence['events']))
                for sig, handler in handlers.items():
                    self.assertEqual(signal.getsignal(sig), handler)

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
        with patch.object(api,'BACKEND',backend), patch.object(coverage,'command',return_value=subprocess.CompletedProcess([],0,'{}')) as validate:
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

    def test_three_phase_lifecycle_order_and_identity(self):
        self.assertTrue(self.campaign())
        actions=[]
        for args in self.calls:
            if args[0]=='apih': actions.append(Path(args[-1]).name)
            elif args[0]=='pg_ctl': actions.append(args[-1])
            elif 'textfmt' in args: actions.append('convert')
        self.assertEqual(actions,['start','normal','stop','outage','start','recovery','stop','convert'])
        self.assertEqual(sum('build' in a for a in self.calls),1)
        self.assertEqual(sum(a[0]=='initdb' for a in self.calls),1)
        starts=[a for a in self.calls if a[0]=='pg_ctl' and a[-1]=='start']
        self.assertEqual(starts[0],starts[1])
        self.assertEqual([x['server_pid'] for x in self.runner.evidence['phases']],[12345]*3)
        self.assertEqual([x['status'] for x in self.runner.evidence['phases']],['passed']*3)
        events=[e['name'] for e in self.runner.evidence['events']]
        self.assertLess(events.index('server-stop'),events.index('cleanup'))
        self.assertEqual(self.runner.evidence['status'],'complete')
        self.assertEqual(self.server.signals,[signal.SIGTERM])

    def test_every_phase_failure_preserves_exit_and_blocks_later_phases(self):
        import shutil
        for phase in ['normal','outage','recovery']:
            for code in [101,102,103,9]:
                with self.subTest(phase=phase,code=code):
                    shutil.rmtree(self.directory/'counters',ignore_errors=True)
                    def fail(args):
                        if args[0]=='apih' and Path(args[-1]).name==phase:
                            raise subprocess.CalledProcessError(code,args)
                    with self.assertRaises(subprocess.CalledProcessError) as raised:
                        self.campaign(hook=fail)
                    self.assertEqual(raised.exception.returncode,code)
                    self.assertFalse(any('textfmt' in a for a in self.calls))
                    phases=self.runner.evidence['phases']; index=['normal','outage','recovery'].index(phase)
                    self.assertEqual([p['status'] for p in phases],['passed']*index+['failed']+['not-reached']*(2-index))
                    self.assertEqual(self.server.signals,[signal.SIGTERM])
                    stops=[a for a in self.calls if a[0]=='pg_ctl' and a[-1]=='stop']
                    self.assertEqual(len(stops),2 if phase=='recovery' else 1)
                    self.assertIsNone(self.runner.database)

    def test_postcondition_failures_block_reporting(self):
        import shutil
        for name in ['normal-postconditions.sql','outage-postconditions.sql','recovery-postconditions.sql']:
            shutil.rmtree(self.directory/'counters',ignore_errors=True)
            def fail(args):
                if args[0]=='psql' and Path(args[-1]).name==name:
                    raise subprocess.CalledProcessError(3,args)
            with self.assertRaises(subprocess.CalledProcessError): self.campaign(hook=fail)
            self.assertFalse(any('textfmt' in a for a in self.calls))
            if name.startswith('normal'):
                self.assertEqual([p['status'] for p in self.runner.evidence['phases']],['passed','not-reached','not-reached'])

    def test_server_crash_in_each_phase_rejects_successful_apih_exit(self):
        import shutil
        for phase in ['normal','outage','recovery']:
            shutil.rmtree(self.directory/'counters',ignore_errors=True)
            def crash(args):
                if args[0]=='apih' and Path(args[-1]).name==phase: self.server.code=4
            with self.assertRaisesRegex(RuntimeError,'original server'):
                self.campaign(hook=crash)
            self.assertFalse(any('textfmt' in a for a in self.calls))
            self.assertIsNone(self.runner.database)

    def test_failed_partial_restart_keeps_ownership_until_cleanup(self):
        starts=0
        def fail(args):
            nonlocal starts
            if args[0]=='pg_ctl' and args[-1]=='start':
                starts+=1
                if starts==2:
                    (self.runner.database/'postmaster.pid').write_text('partially started')
                    raise subprocess.TimeoutExpired(args,15)
        with self.assertRaises(subprocess.TimeoutExpired): self.campaign(hook=fail)
        self.assertEqual(self.calls[-1][-1],'stop')
        self.assertIsNone(self.runner.database)
        self.assertEqual([p['status'] for p in self.runner.evidence['phases']],['passed','passed','not-reached'])

    def test_interrupt_at_each_transition_and_phase_cleans_resources(self):
        import shutil
        for boundary in ['normal','stop','outage','restart','recovery']:
            for sig in [signal.SIGINT,signal.SIGTERM]:
                with self.subTest(boundary=boundary,sig=sig):
                    shutil.rmtree(self.directory/'counters',ignore_errors=True)
                    fired=False; starts=0
                    def interrupt(args):
                        nonlocal fired,starts
                        if args[0]=='pg_ctl' and args[-1]=='start': starts+=1
                        hit=(args[0]=='apih' and Path(args[-1]).name==boundary or
                             args[0]=='pg_ctl' and args[-1]=='stop' and boundary=='stop' or
                             args[0]=='pg_ctl' and args[-1]=='start' and starts==2 and boundary=='restart')
                        if hit and not fired:
                            fired=True
                            raise InterruptedError(sig,'test boundary')
                    with self.assertRaises(InterruptedError) as raised: self.campaign(hook=interrupt)
                    self.assertEqual(raised.exception.errno,sig)
                    self.assertIsNone(self.runner.database)
                    self.assertIsNone(self.runner.server)
                    self.assertFalse(any('textfmt' in a for a in self.calls))

    def test_false_success_pid_and_timeout_states_require_cleanup(self):
        self.runner.database=self.directory/'data'; self.runner.database.mkdir()
        self.runner.socket_directory=str(self.directory)
        pid=self.runner.database/'postmaster.pid'
        with patch.object(self.runner,'execute'):
            with self.assertRaisesRegex(RuntimeError,'without PID'): self.runner.start_database()
            self.assertEqual(self.runner.database_state,'uncertain')
            pid.touch()
            with self.assertRaisesRegex(RuntimeError,'left PID'): self.runner.stop_database()
            self.assertEqual(self.runner.database_state,'uncertain')
            with self.assertRaisesRegex(RuntimeError,'left PID'): self.runner.cleanup()
            self.assertIsNotNone(self.runner.database)
        pid.unlink()
        for operation in [self.runner.start_database,self.runner.stop_database]:
            with patch.object(self.runner,'execute',side_effect=subprocess.TimeoutExpired('pg_ctl',15)):
                with self.assertRaises(subprocess.TimeoutExpired): operation()
                self.assertEqual(self.runner.database_state,'uncertain')
        with patch.object(self.runner,'execute') as execute:
            self.runner.cleanup()
            self.assertEqual(execute.call_count,1)
            self.assertIsNone(self.runner.database)

    def test_cleanup_failure_invalidates_success_and_preserves_primary_error(self):
        import shutil
        for primary in [False,True]:
            shutil.rmtree(self.directory/'counters',ignore_errors=True)
            stops=0
            def fail(args):
                nonlocal stops
                if primary and args[0]=='apih' and Path(args[-1]).name=='recovery':
                    raise subprocess.CalledProcessError(101,args)
                if args[0]=='pg_ctl' and args[-1]=='stop':
                    stops+=1
                    if stops==2: raise subprocess.CalledProcessError(29,args)
            with self.assertRaises(subprocess.CalledProcessError) as raised:
                self.campaign(hook=fail)
            self.assertEqual(raised.exception.returncode,101 if primary else 29)
            self.assertFalse(any('textfmt' in a for a in self.calls))
            self.assertTrue(any(e['name']=='cleanup' and e['status']=='failed' for e in self.runner.evidence['events']))
            self.assertIsNone(self.runner.database) # Emergency stop still reaps it.

    def test_uncertain_cluster_without_pid_retains_diagnostics(self):
        import shutil
        with contextlib.redirect_stderr(io.StringIO()) as errors:
            with api.private_cluster(self.runner) as scratch:
                self.runner.database=Path(scratch)/'data'
                self.runner.database_state='uncertain'
        self.assertTrue(Path(scratch).exists())
        self.assertIn('unconfirmed',errors.getvalue())
        shutil.rmtree(scratch)

    def test_evidence_write_failure_cannot_block_nested_cleanup_or_emergency_stop(self):
        original_write = Path.write_text
        for emergency in [False, True]:
            # Fail each before/after write in cleanup, database-stop and command.
            for fail_at in range(1, 7):
                with self.subTest(emergency=emergency, fail_at=fail_at):
                    data = self.directory / 'data'
                    data.mkdir(exist_ok=True)
                    pid = data / 'postmaster.pid'
                    pid.write_text('owned fake pid')
                    self.runner.database = data
                    self.runner.database_state = 'uncertain' if emergency else 'running'
                    server = self.runner.server = Server()
                    calls = []
                    writes = 0
                    def write(path, *args, **kwargs):
                        nonlocal writes
                        if path.name == 'campaign.json':
                            writes += 1
                            if writes >= fail_at:
                                raise OSError(errno.ENOSPC, 'disk full')
                        return original_write(path, *args, **kwargs)
                    def execute(args, timeout):
                        self.assertIsNone(self.runner.server)
                        self.assertEqual(server.signals, [signal.SIGTERM])
                        self.assertEqual(timeout, 15)
                        self.assertEqual(args[args.index('-D') + 1], str(data))
                        mode = args[args.index('-m') + 1]
                        calls.append(mode)
                        if emergency and mode == 'fast':
                            raise subprocess.CalledProcessError(29, args)
                        pid.unlink()
                    handlers = {sig: signal.getsignal(sig) for sig in (signal.SIGINT, signal.SIGTERM)}
                    with patch.object(Path, 'write_text', write), \
                         patch.object(self.runner, '_execute', side_effect=execute), \
                         contextlib.redirect_stderr(io.StringIO()) as errors:
                        with self.assertRaises(subprocess.CalledProcessError if emergency else OSError) as raised:
                            self.runner.cleanup()
                    if emergency:
                        self.assertEqual(raised.exception.returncode, 29)
                    else:
                        self.assertEqual(raised.exception.errno, errno.ENOSPC)
                    self.assertEqual(calls, ['fast', 'immediate'] if emergency else ['fast'])
                    self.assertIsNone(self.runner.server)
                    self.assertIsNone(self.runner.database)
                    self.assertFalse(pid.exists())
                    self.assertIn('campaign evidence persistence failed', errors.getvalue())
                    self.assertFalse(self.runner.cleaning_up)
                    for sig, handler in handlers.items():
                        self.assertEqual(signal.getsignal(sig), handler)

    def test_campaign_evidence_failure_after_startup_cleans_up_and_blocks_coverage(self):
        import shutil
        original_write = Path.write_text
        for primary in [None, 101, 102, 103, signal.SIGTERM]:
            with self.subTest(primary=primary):
                shutil.rmtree(self.directory / 'counters', ignore_errors=True)
                failed = False
                def write(path, *args, **kwargs):
                    if failed and path.name == 'campaign.json':
                        raise OSError(errno.ENOSPC, 'disk full')
                    return original_write(path, *args, **kwargs)
                def fail(args):
                    nonlocal failed
                    if args[0] == 'apih':
                        failed = True
                        if primary == signal.SIGTERM:
                            raise InterruptedError(primary, 'interrupted')
                        if primary is not None:
                            raise subprocess.CalledProcessError(primary, args)
                with patch.object(Path, 'write_text', write), \
                     contextlib.redirect_stderr(io.StringIO()) as errors:
                    expected = InterruptedError if primary == signal.SIGTERM else subprocess.CalledProcessError if primary else OSError
                    with self.assertRaises(expected) as raised:
                        self.campaign(hook=fail)
                if primary == signal.SIGTERM:
                    self.assertEqual(raised.exception.errno, primary)
                elif primary:
                    self.assertEqual(raised.exception.returncode, primary)
                self.assertEqual(self.server.signals, [signal.SIGTERM])
                self.assertIsNone(self.runner.server)
                self.assertIsNone(self.runner.database)
                self.assertFalse(any('textfmt' in args for args in self.calls))
                self.assertIn('campaign evidence persistence failed', errors.getvalue())

    def test_final_evidence_failure_preserves_primary_and_removes_success_artifacts(self):
        original_write = Path.write_text
        def write(path, *args, **kwargs):
            if path.name == 'campaign.json':
                raise OSError(errno.ENOSPC, 'disk full')
            return original_write(path, *args, **kwargs)
        def fail(runner, port, legacy=False):
            for name in ('coverage.out', 'report.txt', 'result.json'):
                (runner.directory / name).write_text('invalid')
            raise subprocess.CalledProcessError(101, ['apih'])
        with patch.object(coverage, 'BACKEND', self.directory), patch.object(api, 'prerequisites'), \
             patch.object(api, 'free_port'), patch.object(api, 'campaign', side_effect=fail), \
             patch.object(coverage, 'provenance'), \
             patch.object(coverage, 'command', return_value=subprocess.CompletedProcess([], 0, '')), \
             patch.object(Path, 'write_text', write), contextlib.redirect_stderr(io.StringIO()) as errors:
            with self.assertRaises(subprocess.CalledProcessError) as raised:
                api.main()
        self.assertEqual(raised.exception.returncode, 101)
        self.assertIn('campaign evidence persistence failed', errors.getvalue())
        for name in ('coverage.out', 'report.txt', 'result.json'):
            self.assertFalse((self.directory / '.coverage/api' / name).exists())

    def test_deferred_spawn_interrupt_retains_server_handle(self):
        @contextlib.contextmanager
        def interrupt_after_assignment():
            yield set()
            raise InterruptedError(signal.SIGTERM,'after spawn')
        with patch.object(api,'deferred_signals',interrupt_after_assignment):
            with self.assertRaises(InterruptedError): self.campaign()
        self.assertEqual(self.server.signals,[signal.SIGTERM])
        self.assertIsNone(self.runner.server)
        self.assertIsNone(self.runner.database)

    def test_executed_child_does_not_inherit_blocked_termination_signal(self):
        self.runner.execute([sys.executable,'-c',
            'import signal; assert not ({signal.SIGTERM,signal.SIGINT} & signal.pthread_sigmask(signal.SIG_BLOCK,set()))'])

    def test_provenance_failure_cannot_leave_success_or_replace_primary(self):
        for primary in [False,True]:
            def campaign(runner, port, legacy=False):
                for name in ('coverage.out','report.txt','result.json'):
                    (runner.directory/name).write_text('invalid')
                if primary: raise subprocess.CalledProcessError(101,['apih'])
                return True
            with patch.object(coverage,'BACKEND',self.directory), patch.object(api,'prerequisites'), \
                 patch.object(api,'free_port'), patch.object(api,'campaign',side_effect=campaign), \
                 patch.object(coverage,'provenance',side_effect=RuntimeError('evidence failed')), \
                 patch.object(coverage,'command',return_value=subprocess.CompletedProcess([],0,'')):
                with self.assertRaises(subprocess.CalledProcessError if primary else RuntimeError) as raised:
                    api.main()
                if primary: self.assertEqual(raised.exception.returncode,101)
            for name in ('coverage.out','report.txt','result.json'):
                self.assertFalse((self.directory/'.coverage/api'/name).exists())

    def test_cleanup_evidence_and_shutdown_ignore_interrupts(self):
        self.runner.server=Server()
        original=self.runner.save_evidence
        def evidence():
            self.assertEqual(signal.getsignal(signal.SIGINT),signal.SIG_IGN)
            self.assertEqual(signal.getsignal(signal.SIGTERM),signal.SIG_IGN)
            original()
        with patch.object(self.runner,'save_evidence',side_effect=evidence):
            self.runner.cleanup()
        self.assertIsNone(self.runner.server)
        self.assertEqual(self.runner.evidence['events'][-1]['status'],'passed')
