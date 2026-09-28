import contextlib
import io
import json
import hashlib
from pathlib import Path
import subprocess
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
                (94999,100000,False,False), (90,100,True,True), (89999,100000,True,False)]:
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
            self.assertTrue(coverage.report(meta,self.profile,self.root,True))
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
        result=subprocess.CompletedProcess([],0,json.dumps(dict(ImportPath='m/p',GoFiles=['a.go','new.go'])))
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
        with patch.object(coverage,'BACKEND',self.root):
            directory=self.root/'.coverage/unit'; directory.mkdir(parents=True)
            (directory/'result.json').write_text('stale PASS')
            with coverage.fresh_run('unit') as fresh:
                self.assertEqual(list(fresh.iterdir()),[])
                with self.assertRaises(BlockingIOError), coverage.fresh_run('unit'):
                    pass

    def test_failed_tests_delete_partial_profile_and_no_report(self):
        def fail(args, **kwargs):
            if 'test' in args:
                (self.root/'.coverage/unit/coverage.out').write_text('partial')
                raise subprocess.CalledProcessError(17,args)
        with patch.object(coverage,'BACKEND',self.root), patch.object(coverage,'inventory',return_value={}), \
             patch.object(coverage,'provenance'), patch.object(coverage,'command',side_effect=fail):
            with self.assertRaises(subprocess.CalledProcessError): coverage.unit(True)
            self.assertFalse((self.root/'.coverage/unit/coverage.out').exists())
            self.assertFalse((self.root/'.coverage/unit/report.txt').exists())

    def test_html_generated_for_valid_below_threshold(self):
        with patch.object(coverage,'BACKEND',self.root), patch.object(coverage,'inventory',return_value={}), \
             patch.object(coverage,'provenance'), patch.object(coverage,'report',return_value=False), \
             patch.object(coverage,'command') as command:
            self.assertEqual(coverage.unit(True),1)
            args=command.call_args_list[0].args[0]
            for option in ['-short','-count=1','-race','-covermode=atomic']:
                self.assertIn(option,args)
            self.assertIn('cover', command.call_args_list[1].args[0])
            self.assertTrue(any(a.startswith('-html=') for a in command.call_args_list[1].args[0]))

    def test_inventory_uses_all_go_packages_and_structural_metadata(self):
        source=self.root/'file.go'; source.write_text('package p\nfunc F() {}\n')
        def command(args,**kwargs):
            if args[1]=='list':
                self.assertEqual(args[3:],coverage.PATTERNS)
                return subprocess.CompletedProcess(args,0,json.dumps(dict(ImportPath='m/p',Dir=str(self.root),GoFiles=['file.go'])))
            output=Path(args[args.index('-o')+1])
            output.write_text('Pos: [3 * 1]uint32{\n1, 2, 0x20001,\n}\nNumStmt: [1]uint16{\n1,\n}')
        with patch.object(coverage,'command',side_effect=command):
            meta=coverage.inventory(self.root)
        self.assertEqual(meta['blocks'],{'m/p/file.go:1.1,2.2':1})
        with patch.object(coverage,'command',side_effect=command):
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
        with patch.object(coverage, 'BACKEND', backend):
            coverage.provenance(output)
        provenance = json.loads((output / 'provenance.json').read_text())
        self.assertIn('+unstaged edit', provenance['diff'])
        self.assertIn('-initial', provenance['diff'])
        self.assertEqual(provenance['inputs_sha256']['new.go'],
                         hashlib.sha256(b'new source\n').hexdigest())
        self.assertEqual(provenance['inputs_sha256']['source.go'],
                         hashlib.sha256(b'unstaged edit\n').hexdigest())
