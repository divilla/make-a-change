import contextlib
import io
import json
import hashlib
import os
from pathlib import Path
import signal
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import patch

import coverage


class CoverageTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.profile = self.root / 'coverage.out'

    def measure(self, covered, total, integration=False):
        blocks = {'m/p/a.go:1.1,2.1': covered, 'm/p/a.go:3.1,4.1': total-covered}
        meta = dict(packages=['m/p', 'm/empty'], blocks=blocks, sources={})
        self.profile.write_text('mode: atomic\n' + ''.join(f'{k} {v} {int(i == 0)}\n' for i, (k, v) in enumerate(blocks.items())))
        with contextlib.redirect_stdout(io.StringIO()):
            return coverage.report(meta, self.profile, self.root, integration)

    def test_exact_boundaries_not_rounded(self):
        for covered, total, api, expected in [(95,100,False,False), (95001,100000,False,True),
                (94999,100000,False,False), (90,100,True,False), (90001,100000,True,True), (89999,100000,True,False)]:
            with self.subTest(covered=covered, api=api):
                self.assertEqual(self.measure(covered,total,api), expected)
        self.assertIn('no executable statements', (self.root/'report.txt').read_text())

    def test_diagnostic_report_below_threshold_retains_actual_counts(self):
        meta = dict(packages=['m/p'], blocks={'m/p/a.go:1.1,2.1':1, 'm/p/a.go:3.1,4.1':9}, sources={})
        self.profile.write_text('mode: atomic\nm/p/a.go:1.1,2.1 1 1\nm/p/a.go:3.1,4.1 9 0\n')
        with contextlib.redirect_stdout(io.StringIO()) as output:
            self.assertTrue(coverage.report(meta, self.profile, self.root, integration=True, enforce=False))
        self.assertIn('TOTAL 1/10 10.0000% — diagnostic (no coverage gate)', output.getvalue())
        result = json.loads((self.root/'result.json').read_text())
        self.assertEqual((result['covered'], result['total']), (1, 10))
        self.assertIsNone(result['passed'])
        self.assertFalse(result['threshold_enforced'])

    def test_denominator_missing_unlinked_and_partial_packages(self):
        meta = dict(packages=['m/p','m/q'], blocks={'m/p/a.go:1.1,2.1':9, 'm/q/a.go:1.1,2.1':1}, sources={})
        self.profile.write_text('mode: atomic\nm/p/a.go:1.1,2.1 9 1\n')
        with self.assertRaisesRegex(ValueError,'incomplete unit'):
            coverage.report(meta,self.profile,self.root)
        with contextlib.redirect_stdout(io.StringIO()):
            self.assertFalse(coverage.report(meta,self.profile,self.root,True))
        self.assertEqual(json.loads((self.root/'result.json').read_text())['total'],10)
        meta['blocks']['m/p/b.go:1.1,2.1'] = 1
        with self.assertRaisesRegex(ValueError,'partial integration'):
            coverage.report(meta,self.profile,self.root,True)

    def test_linked_package_metadata_cannot_be_silently_zeroed(self):
        meta=dict(packages=['m/p','m/q'], blocks={'m/p/a.go:1.1,2.1':99,'m/q/a.go:1.1,2.1':1}, sources={}, linked=['m/p','m/q'])
        self.profile.write_text('mode: atomic\nm/p/a.go:1.1,2.1 99 1\n')
        with self.assertRaisesRegex(ValueError,'missing linked'):
            coverage.report(meta,self.profile,self.root,True)

    def test_changed_production_file_set_rejected(self):
        meta=dict(sources={},layout={'m/p':['a.go']})
        result=subprocess.CompletedProcess([],0,json.dumps(dict(ImportPath='m/p',Dir=str(coverage.CLI/'pkg/p'),GoFiles=['a.go','new.go'])))
        with patch.object(coverage,'command',return_value=result):
            with self.assertRaisesRegex(ValueError,'package/file set changed'):
                coverage.unchanged(meta)

    def test_malformed_and_mismatched_profiles(self):
        meta = dict(packages=['m/p'], blocks={'m/p/a.go:1.1,2.1':1}, sources={})
        for text in ['', 'mode: atomic\n', 'mode: set\n', 'mode: atomic\ngarbage', 'mode: atomic\nm/p/a.go:1.1,2.1 2 1\n',
                     'mode: atomic\nm/p/a.go:1.1,2.1 1 -1\n']:
            self.profile.write_text(text)
            with self.subTest(text=text), self.assertRaises(ValueError):
                coverage.report(meta,self.profile,self.root)

    def test_source_change_rejected(self):
        path = self.root/'source.go'; path.write_text('changed')
        with self.assertRaisesRegex(ValueError,'source changed'):
            coverage.unchanged({'sources':{str(path):'old digest'}})

    def test_fresh_run_removes_stale_data_and_locks_concurrent_runs(self):
        with patch.object(coverage,'CLI',self.root):
            directory=self.root/'.coverage/unit'; directory.mkdir(parents=True)
            (directory/'result.json').write_text('stale PASS')
            with coverage.fresh_run('unit') as fresh:
                self.assertEqual(list(fresh.iterdir()),[])
                with self.assertRaises(BlockingIOError), coverage.fresh_run('unit'):
                    pass

    def test_inventory_uses_all_go_packages_and_structural_metadata(self):
        source=self.root/'file.go'; source.write_text('package p\nfunc F() {}\n')
        def command(args,**kwargs):
            if args[1]=='list':
                self.assertEqual(args[3:],coverage.PATTERNS)
                return subprocess.CompletedProcess(args,0,json.dumps(dict(ImportPath='m/p',Dir=str(self.root),GoFiles=['file.go'])))
            output=Path(args[args.index('-o')+1])
            output.write_text('Pos: [3 * 1]uint32{\n1, 2, 0x20001,\n}\nNumStmt: [1]uint16{\n1,\n}')
        with patch.object(coverage,'CLI',self.root), patch.object(coverage,'command',side_effect=command):
            meta=coverage.inventory(self.root)
        self.assertEqual(meta['blocks'],{'m/p/file.go:1.1,2.2':1})
        with patch.object(coverage,'CLI',self.root), patch.object(coverage,'command',side_effect=command):
            coverage.unchanged(meta)

    def test_provenance_records_staged_unstaged_and_untracked_backend_inputs(self):
        def git(*args):
            return subprocess.run(['git', *args], cwd=self.root, check=True, text=True,
                                  capture_output=True)
        git('init')
        git('config', 'user.name', 'Coverage Fixture')
        git('config', 'user.email', 'fixture@example.invalid')
        backend = self.root / 'backend'
        backend.mkdir()
        tracked = backend / 'source.go'
        tracked.write_text('initial\n')
        git('add', 'backend/source.go')
        git('commit', '-m', 'initial fixture')
        tracked.write_text('staged edit\n')
        git('add', 'backend/source.go')
        tracked.write_text('unstaged edit\n')
        (backend / 'new.go').write_text('new source\n')
        output = self.root / 'report'
        output.mkdir()
        with patch.object(coverage, 'CLI', backend):
            coverage.provenance(output)
        provenance = json.loads((output / 'provenance.json').read_text())
        self.assertIn('+unstaged edit', provenance['diff'])
        self.assertIn('-initial', provenance['diff'])
        self.assertEqual(provenance['inputs_sha256']['new.go'],
                         hashlib.sha256(b'new source\n').hexdigest())
        self.assertEqual(provenance['inputs_sha256']['source.go'],
                         hashlib.sha256(b'unstaged edit\n').hexdigest())

    def test_overlapping_profiles_union_hits_never_sum_statements(self):
        other = self.root/'child.out'
        self.profile.write_text('mode: atomic\nm/p/a.go:1.1,2.1 9 1\nm/p/a.go:1.1,2.1 9 2\nm/q/a.go:1.1,2.1 1 0\n')
        other.write_text('mode: atomic\nm/p/a.go:1.1,2.1 9 0\nm/q/a.go:1.1,2.1 1 3\n')
        merged = self.root/'merged.out'
        coverage.union_profiles([self.profile,other], merged)
        self.assertEqual(coverage.parse_profile(merged), {'m/p/a.go:1.1,2.1':(9,2),'m/q/a.go:1.1,2.1':(1,3)})
        other.write_text('mode: atomic\nm/p/a.go:1.1,2.1 8 1\n')
        with self.assertRaisesRegex(ValueError,'mismatched'):
            coverage.union_profiles([self.profile,other],merged)
        with self.assertRaisesRegex(ValueError,'another campaign'):
            coverage.union_profiles([self.root/'unit'/'coverage.out'],merged)

    def test_discovery_empty_failures_and_unusual_production_locations(self):
        packages = [dict(ImportPath='cli/'+p,Dir=str(self.root/p),GoFiles=['a.go'])
                    for p in ['cmd/mch','internal/a','pkg/b','extra','integration','scripts/tool']]
        with patch.object(coverage,'CLI',self.root), patch.object(coverage,'command') as command:
            command.return_value.stdout = '\n'.join(json.dumps(p) for p in packages)
            found,_ = coverage.discover()
            self.assertEqual([p['ImportPath'] for p in found],['cli/cmd/mch','cli/internal/a','cli/pkg/b','cli/extra'])
            command.return_value.stdout = ''
            with self.assertRaisesRegex(ValueError,'no production'): coverage.discover()
            command.side_effect = subprocess.CalledProcessError(7,['go','list'])
            with self.assertRaises(subprocess.CalledProcessError): coverage.discover()

    def test_events_fail_closed_and_require_every_scenario(self):
        valid = [{'Action':'run','Test':'TestProgram'}, {'Action':'pass','Test':'TestProgram/sub'},
                 {'Action':'pass','Test':'TestProgram'}]
        encode = lambda events: '\n'.join(json.dumps(e) for e in events)
        coverage.validate_events(encode(valid),['TestProgram'])
        for events, names in [(valid,[]),(valid,['TestMissing']),
                              (valid+[{'Action':'skip','Test':'TestProgram/sub'}],['TestProgram']),
                              (valid+[{'Action':'fail'}],['TestProgram']),
                              (valid+[{'Action':'run','Test':'TestHTTPAdapter'}],['TestProgram'])]:
            with self.assertRaises(ValueError): coverage.validate_events(encode(events),names)

    def test_scenario_manifest_rejects_scripts_empty_and_unmatched(self):
        manifest=json.loads((coverage.CLI/'scripts/terminal-scenarios.json').read_text())
        self.assertNotIn('TestCLIProgramDefReviewUsesDefinitionPromptAndSharedArtifactSession',manifest['program']['tests'])
        self.assertNotIn('TestCLIProgramArtifactChatResumesSharedArtifactSession',manifest['program']['tests'])
        self.assertEqual(manifest['pty']['tests'], ['TestShellNavigationEditorAndScrolling'])
        self.assertEqual(set(manifest['program']['tests']), {
            'TestCLIProgramStartupNavigationAndSelection', 'TestCLIProgramEditorSaveAndFailure',
            'TestCLIProgramOrdinaryDocumentEditor', 'TestCLIStartupWithoutFlowResources'})
        available='\n'.join(n for suite in manifest.values() for n in suite['tests'])
        with patch.object(coverage,'command',return_value=subprocess.CompletedProcess([],0,available)):
            self.assertEqual(coverage.scenarios(),manifest)
            for name in ['', 'TestHTTPClient', 'TestCLIProgramMissing',
                         'TestCLIProgramArtifactChatResumesSharedArtifactSession', 'TestRewriteScreenUsesColoredBlackScrollableViewport']:
                candidate=json.loads(json.dumps(manifest)); candidate['program']['tests']=[name] if name else []
                with patch.object(Path,'read_text',return_value=json.dumps(candidate)), self.assertRaises(ValueError):
                    coverage.scenarios()

    def test_missing_child_counters_and_conversion_failure(self):
        with self.assertRaisesRegex(ValueError,'missing child'): coverage.counter_profile(self.root,self.root,'child')
        (self.root/'covmeta.a').touch()
        with self.assertRaisesRegex(ValueError,'missing child'): coverage.counter_profile(self.root,self.root,'child')
        (self.root/'covcounters.a').touch()
        with patch.object(coverage,'recorded',side_effect=subprocess.CalledProcessError(9,['convert'])):
            with self.assertRaises(subprocess.CalledProcessError): coverage.counter_profile(self.root,self.root,'child')

    def test_failed_tests_retain_raw_diagnostics_without_stale_success(self):
        directory=self.root/'.coverage/unit'; directory.mkdir(parents=True)
        (directory/'result.json').write_text('stale PASS')
        def run(directory,label,args,**kwargs):
            if label=='unit':
                (directory/'coverage.out').write_text('partial profile')
                raise subprocess.CalledProcessError(17,args)
        with patch.object(coverage,'CLI',self.root), patch.object(coverage,'inventory',return_value={'packages':['cli/cmd/mch']}), \
             patch.object(coverage,'provenance',return_value=None), patch.object(coverage,'recorded',side_effect=run):
            with self.assertRaises(subprocess.CalledProcessError) as caught: coverage.campaign()
        self.assertEqual(caught.exception.returncode,17)
        self.assertEqual((directory/'coverage.out').read_text(),'partial profile')
        self.assertFalse((directory/'result.json').exists())
        self.assertFalse((directory/'report.txt').exists())
        self.assertEqual(json.loads((directory/'status.json').read_text())['exit'],17)

    def test_below_target_preserves_profile_and_html(self):
        def run(directory,label,args,**kwargs):
            if label=='unit': (directory/'coverage.out').write_text('valid profile fixture')
            return ''
        with patch.object(coverage,'CLI',self.root), patch.object(coverage,'inventory',return_value={'packages':['cli/cmd/mch']}), \
             patch.object(coverage,'provenance',return_value=None), patch.object(coverage,'recorded',side_effect=run), \
             patch.object(coverage,'diagnostics') as diagnostics, patch.object(coverage,'report',return_value=False):
            self.assertEqual(coverage.campaign(html=True),1)
            self.assertTrue(diagnostics.call_args.args[-1])
        directory=self.root/'.coverage/unit'
        self.assertTrue((directory/'coverage.out').exists())
        self.assertTrue(json.loads((directory/'status.json').read_text())['complete'])

    def test_interrupt_or_postprocessing_failure_invalidates_results(self):
        for failure in [KeyboardInterrupt(),ValueError('cleanup failed')]:
            with patch.object(coverage,'CLI',self.root), patch.object(coverage,'inventory',side_effect=failure), \
                 patch.object(coverage,'provenance',return_value=None), patch.object(coverage,'recorded'):
                with self.assertRaises(type(failure)): coverage.campaign()
            status=json.loads((self.root/'.coverage/unit/status.json').read_text())
            self.assertFalse(status['complete'])

    def test_original_subprocess_exit_and_log(self):
        with self.assertRaises(subprocess.CalledProcessError) as caught:
            coverage.recorded(self.root,'failed',['sh','-c','echo diagnostic; exit 19'])
        self.assertEqual(caught.exception.returncode,19)
        self.assertIn('diagnostic',(self.root/'failed.log').read_text())
        self.assertEqual(json.loads((self.root/'commands.jsonl').read_text())['exit'],19)

    def test_output_reaches_log_before_child_exits_including_partial_lines(self):
        log = self.root / 'live.log'
        child = f'''
import pathlib, sys
sys.stdout.write('stdout partial')
sys.stdout.flush()
sys.stderr.write('stderr partial')
sys.stderr.flush()
assert pathlib.Path({str(log)!r}).read_text() == 'stdout partialstderr partial'
'''
        with contextlib.redirect_stdout(io.StringIO()):
            output = coverage.recorded(self.root, 'live', [sys.executable, '-c', child])
        self.assertEqual(output, 'stdout partialstderr partial')
        self.assertEqual(json.loads((self.root/'commands.jsonl').read_text())['exit'], 0)

    def test_command_signal_exit_survives_supervision(self):
        with self.assertRaises(subprocess.CalledProcessError) as caught:
            coverage.recorded(self.root, 'killed', ['sh', '-c', 'kill -KILL $$'])
        self.assertEqual(caught.exception.returncode, -signal.SIGKILL)
        self.assertEqual(json.loads((self.root/'commands.jsonl').read_text())['exit'], -signal.SIGKILL)

    def test_successful_driver_cannot_leave_detached_descendants(self):
        child = '''
import os, subprocess
process = subprocess.Popen(['sleep', '60'], start_new_session=True)
print(process.pid, flush=True)
'''
        with contextlib.redirect_stdout(io.StringIO()):
            output = coverage.recorded(self.root, 'orphan', [sys.executable, '-c', child])
        with self.assertRaises(ProcessLookupError):
            os.kill(int(output), 0)

    def test_sigint_during_child_retains_diagnostics_journal_and_incomplete_status(self):
        self.check_campaign_interruption(signal.SIGINT)

    def test_sigterm_during_child_retains_diagnostics_journal_and_incomplete_status(self):
        self.check_campaign_interruption(signal.SIGTERM)

    def test_campaign_signal_handlers_restore_after_success_and_interruption(self):
        previous = {sig: signal.getsignal(sig) for sig in (signal.SIGINT, signal.SIGTERM)}
        with coverage.campaign_signals():
            self.assertTrue(callable(signal.getsignal(signal.SIGTERM)))
        self.assertEqual({sig: signal.getsignal(sig) for sig in previous}, previous)
        with self.assertRaisesRegex(KeyboardInterrupt, 'SIGTERM'):
            with coverage.campaign_signals():
                signal.raise_signal(signal.SIGTERM)
        self.assertEqual({sig: signal.getsignal(sig) for sig in previous}, previous)

    def check_campaign_interruption(self, interruption):
        (self.root/'go.mod').write_text('module interruption\n\ngo 1.26.0\n')
        (self.root/'interrupt_test.go').write_text(r'''package interruption
import (
    "fmt"
    "os"
    "os/exec"
    "strconv"
    "syscall"
    "testing"
    "time"
)
func TestInterrupt(t *testing.T) {
    // Model socat's setsid child, including an orphan that escapes the Go tree.
    cmd := exec.Command("sh", "-c", "trap '' TERM; sleep 60 & echo $! > orphan.pid; echo $$ > detached.pid; wait")
    cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
    if err := cmd.Start(); err != nil { t.Fatal(err) }
    for _, name := range []string{"orphan.pid", "detached.pid"} {
        for {
            data, err := os.ReadFile(name)
            if err == nil && len(data) > 0 { break }
            time.Sleep(time.Millisecond)
        }
    }
    if err := os.WriteFile("test.pid", []byte(strconv.Itoa(os.Getpid())), 0600); err != nil { t.Fatal(err) }
    if err := os.WriteFile("driver.pid", []byte(strconv.Itoa(os.Getppid())), 0600); err != nil { t.Fatal(err) }
    // Orphan the separate session before interruption to exercise adoption.
    if err := cmd.Process.Kill(); err != nil { t.Fatal(err) }
    _ = cmd.Wait()
    fmt.Println("stdout before interrupt")
    fmt.Fprint(os.Stderr, "stderr partial before interrupt")
    runner, err := strconv.Atoi(os.Getenv("RUNNER_PID"))
    if err != nil { t.Fatal(err) }
    sig, err := strconv.Atoi(os.Getenv("RUNNER_SIGNAL"))
    if err != nil { t.Fatal(err) }
    if err := syscall.Kill(runner, syscall.Signal(sig)); err != nil { t.Fatal(err) }
    time.Sleep(time.Minute)
}
''')
        args = [coverage.GO, 'test', '-count=1', '-v', '-timeout=30s', '.']
        # Signal only an isolated runner, once its real Go test and
        # detached descendant exist. Inspect descendants before lock release.
        runner = f'''
import contextlib, fcntl, os, signal, sys
from pathlib import Path
from unittest.mock import patch
sys.path.insert(0, {str(coverage.CLI/'scripts')!r})
import coverage
root = Path({str(self.root)!r})
recorded, fresh_run = coverage.recorded, coverage.fresh_run
wait = coverage.subprocess.Popen.wait
def checked_wait(process, *args, **kwargs):
    if signal.getsignal(signal.SIGTERM) == signal.SIG_IGN:
        os.kill(os.getpid(), signal.SIGTERM)
        os.kill(os.getpid(), signal.SIGINT)
        (root/'repeated-cancellation').touch()
    return wait(process, *args, **kwargs)
@contextlib.contextmanager
def checked_run(name):
    with fresh_run(name) as directory:
        try:
            yield directory
        finally:
            # Repeated cancellation must not interrupt cleanup or journaling.
            os.kill(os.getpid(), signal.SIGTERM)
            os.kill(os.getpid(), signal.SIGINT)
            for path in root.glob('*.pid'):
                try:
                    os.kill(int(path.read_text()), 0)
                except ProcessLookupError:
                    continue
                raise AssertionError('descendant survived: ' + str(path))
            with (root/'.coverage/unit.lock').open('w') as lock:
                try:
                    fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
                except BlockingIOError:
                    pass
                else:
                    raise AssertionError('lock released before cleanup')
            (root/'cleanup-checked').touch()
def run(directory, label, args, **kwargs):
    if label == 'unit':
        return recorded(directory, label, {args!r}, env=dict(os.environ, RUNNER_PID=str(os.getpid()), RUNNER_SIGNAL={str(int(interruption))!r}))
    return ''
with patch.object(coverage, 'CLI', root), \\
     patch.object(coverage, 'provenance', return_value=None), \\
     patch.object(coverage, 'inventory', return_value={{'packages': ['interruption']}}), \\
     patch.object(coverage, 'fresh_run', side_effect=checked_run), \\
     patch.object(coverage.subprocess.Popen, 'wait', checked_wait), \\
     patch.object(coverage, 'recorded', side_effect=run):
    coverage.campaign()
'''
        unrelated = subprocess.Popen(['sleep', '60'], start_new_session=True)
        try:
            result = subprocess.run([sys.executable, '-B', '-c', runner],
                                    capture_output=True, text=True, timeout=45)
            self.assertIsNone(unrelated.poll(), 'cleanup killed unrelated work')
        finally:
            unrelated.kill()
            unrelated.wait()
        self.assertNotEqual(result.returncode, 0)
        self.assertTrue((self.root/'cleanup-checked').exists(), result.stderr)
        self.assertTrue((self.root/'repeated-cancellation').exists(), result.stderr)
        directory = self.root / '.coverage/unit'
        output = (directory/'unit.log').read_text()
        self.assertIn('stdout before interrupt\n', output)
        self.assertIn('stderr partial before interrupt', output)
        entry = json.loads((directory/'commands.jsonl').read_text())
        self.assertEqual(entry['label'], 'unit')
        self.assertEqual(entry['args'], args)
        self.assertEqual(entry['exit'], -signal.SIGTERM)
        self.assertIn('KeyboardInterrupt', entry['error'])
        self.assertIn(interruption.name, entry['error'])
        status = json.loads((directory/'status.json').read_text())
        self.assertFalse(status['complete'])
        self.assertNotEqual(status['exit'], 0)
        self.assertFalse((directory/'result.json').exists())
        self.assertFalse((directory/'report.txt').exists())
        with patch.object(coverage, 'CLI', self.root), coverage.fresh_run('unit') as fresh:
            self.assertEqual(list(fresh.iterdir()), [])

    def test_campaign_requires_socat_and_both_suites_preserving_first_failure(self):
        manifest={'program':{'package':'./integration','tests':['TestCLIProgramFixture']},
                  'pty':{'package':'./integration/terminal','tests':['TestShellFixture']}}
        meta={'packages':['cli/cmd/mch']}
        with patch.object(coverage,'scenarios',return_value=manifest), patch.object(coverage.shutil,'which',return_value=None):
            with self.assertRaisesRegex(ValueError,'socat is required'):
                coverage.integration(self.root,meta)
        executed=[]
        def run(directory,label,args,**kwargs):
            executed.append(label)
            if label=='build': (directory/'mch').write_bytes(b'covered fixture')
            if label=='program': raise subprocess.CalledProcessError(17,args)
            if label=='pty': raise subprocess.CalledProcessError(18,args)
        with patch.object(coverage,'scenarios',return_value=manifest), patch.object(coverage.shutil,'which',return_value='/fixture/socat'), \
             patch.object(coverage,'recorded',side_effect=run):
            with self.assertRaises(subprocess.CalledProcessError) as caught: coverage.integration(self.root,meta)
        self.assertEqual(caught.exception.returncode,17)
        self.assertIn('pty',executed)
        self.assertFalse((self.root/'coverage.out').exists())

    def test_campaign_arguments_never_import_unit_counters(self):
        manifest={'program':{'package':'./integration','tests':['TestCLIProgramFixture']},
                  'pty':{'package':'./integration/terminal','tests':['TestShellFixture']}}
        seen=[]
        def run(directory,label,args,**kwargs):
            seen.append((label,args,kwargs))
            if label=='build': (directory/'mch').write_bytes(b'covered fixture')
            if label in manifest:
                counters=Path(kwargs['env']['MCH_COVER_DIR'])
                self.assertEqual(counters.parent,directory)
                self.assertEqual(kwargs['env']['MCH_COVER_BINARY'],str(directory/'mch'))
                self.assertNotIn('-race',args) # both binary forms use atomic counters
                if label=='program':
                    self.assertIn('-coverpkg=cli/cmd/mch',args)
                    (directory/'program.out').write_text('mode: atomic\ncli/cmd/mch/main.go:1.1,2.1 1 0\n')
                return json.dumps({'Action':'pass','Test':manifest[label]['tests'][0]})
            return ''
        def counters(directory,unused,label):
            path=directory/(label+'.out')
            path.write_text('mode: atomic\ncli/cmd/mch/main.go:1.1,2.1 1 1\n')
            return path
        with patch.object(coverage,'scenarios',return_value=manifest), patch.object(coverage.shutil,'which',return_value='/socat'), \
             patch.object(coverage,'recorded',side_effect=run), patch.object(coverage,'counter_profile',side_effect=counters), \
             patch.object(coverage,'command',return_value=subprocess.CompletedProcess([],0,'cli/cmd/mch\n')):
            profile=coverage.integration(self.root,{'packages':['cli/cmd/mch']})
        self.assertEqual(coverage.parse_profile(profile),{'cli/cmd/mch/main.go:1.1,2.1':(1,1)})
        for label,args,_ in seen:
            self.assertFalse(any('/unit/' in a for a in args))
