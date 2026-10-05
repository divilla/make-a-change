"""Exercise actual Make recipes against isolated executables, never a backend."""
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest

import coverage


class MakefileTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix='cli tooling with spaces ')
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name) / 'cli'
        self.root.mkdir()
        shutil.copy(coverage.CLI/'Makefile', self.root)
        shutil.copy(coverage.CLI/'go.mod', self.root)
        (self.root/'scripts').mkdir()
        (self.root/'scripts/fixture_test.py').write_text('import unittest\nclass Fixture(unittest.TestCase):\n    def test_isolated(self): self.assertTrue(True)\n')
        for name in ('coverage.py', 'verify.py'):
            shutil.copy(coverage.CLI/'scripts'/name, self.root/'scripts'/name)
        (self.root/'bin').mkdir()
        self.log = self.root/'calls.jsonl'
        stub = '''#!/usr/bin/python3
import json, os, pathlib, sys
name=pathlib.Path(sys.argv[0]).name
args=sys.argv[1:]
with open(os.environ['CALLS'], 'a') as f: f.write(json.dumps([name,*args])+'\\n')
if name=='go' and args[:2]==['list','-json']:
    if os.environ.get('DISCOVERY')=='fail': sys.exit(23)
    if os.environ.get('DISCOVERY')!='empty':
        for p in ['cmd/mch','internal/app','pkg/client','extra','integration']:
            print(json.dumps(dict(ImportPath='cli/'+p,Dir=str(pathlib.Path.cwd()/p),GoFiles=['source.go'])))
if name=='golangci-lint' and args[0]=='fmt' and '--diff' not in args:
    pathlib.Path('formatted').write_text('explicit mutation')
if name==os.environ.get('FAIL_TOOL'): sys.exit(31)
'''
        for name in ('go','golangci-lint','govulncheck','docker'):
            path=self.root/'bin'/name; path.write_text(stub); path.chmod(0o700)
        self.env=dict(os.environ,PATH=str(self.root/'bin')+os.pathsep+os.environ['PATH'],
                      GO='go',GOLANGCI_LINT='golangci-lint',GOVULNCHECK='govulncheck',CALLS=str(self.log))

    def make(self,*args):
        return subprocess.run(['make','--no-print-directory',*args],cwd=self.root,env=self.env,
                              text=True,stdout=subprocess.PIPE,stderr=subprocess.STDOUT)

    def calls(self):
        return [json.loads(line) for line in self.log.read_text().splitlines()] if self.log.exists() else []

    def test_default_help_and_all_documented_phony_targets(self):
        result=self.make()
        self.assertEqual(result.returncode,0,result.stdout)
        names='init format format-check lint vet test race check coverage coverage-html integration-test terminal-test integration-coverage deps-audit tooling-test benchmark test_version'.split()
        for name in names:
            self.assertIn(name,result.stdout)
            self.assertIn(name,(self.root/'Makefile').read_text().split('.PHONY:')[1].split('\n')[0])
        self.assertEqual(self.calls(),[])

    def test_package_discovery_includes_cmd_untested_and_extra(self):
        for target, option in [('test','-short'),('race','-race'),('benchmark','-run=^$')]:
            result=self.make(target)
            self.assertEqual(result.returncode,0,result.stdout)
            args=self.calls()[-1]
            self.assertIn(option,args)
            for package in ['cli/cmd/mch','cli/internal/app','cli/pkg/client','cli/extra']:
                self.assertIn(package,args)
            self.assertNotIn('cli/integration',args)
            if target!='benchmark': self.assertIn('-count=1',args)

    def test_failed_or_empty_discovery_cannot_run_checks(self):
        for mode in ['fail','empty']:
            self.env['DISCOVERY']=mode
            self.log.unlink(missing_ok=True)
            self.assertNotEqual(self.make('test').returncode,0)
            self.assertEqual(len(self.calls()),1)
            self.assertEqual(self.calls()[0][1:3],['list','-json'])

    def test_read_only_static_checks_include_tests_and_propagate_failures(self):
        before=(self.root/'go.mod').read_bytes()
        for name in ['format-check','lint','vet','deps-audit']:
            result=self.make(name)
            self.assertEqual(result.returncode,0,result.stdout)
            self.assertIn('./...',self.calls()[-1])
        self.assertFalse((self.root/'formatted').exists())
        self.assertEqual((self.root/'go.mod').read_bytes(),before)
        self.env['FAIL_TOOL']='golangci-lint'
        self.assertNotEqual(self.make('lint').returncode,0)
        del self.env['FAIL_TOOL']
        self.assertEqual(self.make('format').returncode,0)
        self.assertTrue((self.root/'formatted').exists())

    def test_pinned_init_and_docker_no_tty_and_quoted_checkout(self):
        self.assertEqual(self.make('init').returncode,0)
        self.assertEqual(self.calls(),[
            ['go','install','github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.13.1'],
            ['go','install','golang.org/x/vuln/cmd/govulncheck@v1.7.0']])
        for override,version in [([], '1.26.0'), (['goversion=1.26.8'],'1.26.8')]:
            self.assertEqual(self.make('test_version',*override).returncode,0)
            args=self.calls()[-1]
            mount = args[args.index('-v')+1]
            source, destination = mount.rsplit(':', 1)
            self.assertEqual(Path(source).resolve(), self.root.parent)
            self.assertEqual(destination, '/project')
            self.assertEqual(args[args.index('-w')+1], '/project/cli')
            self.assertIn('golang:'+version,args)
            self.assertNotIn('-it',args)
            self.assertNotIn('-t',args)

    def test_docker_provisions_bat_before_check(self):
        self.assertEqual(self.make('test_version').returncode,0)
        recipe=self.calls()[-1][-1]
        # Run the actual container shell recipe with an isolated PATH. Debian's
        # bat package exposes batcat; the check needs an executable named bat.
        stub='''#!/usr/bin/python3
import json, os, pathlib, sys
name=pathlib.Path(sys.argv[0]).name
args=sys.argv[1:]
with open(os.environ['CALLS'], 'a') as f: f.write(json.dumps([name,*args])+'\\n')
root=pathlib.Path(sys.argv[0]).parent
if name=='apt-get' and args[0]=='install':
    assert 'python3' in args and 'bat' in args
    (root/'batcat').symlink_to('/bin/true')
if name=='ln':
    assert args==['-s','/usr/bin/batcat','/usr/local/bin/bat']
    (root/'bat').symlink_to(root/'batcat')
if name=='make':
    assert args==['init','check']
    assert (root/'bat').exists(), 'check requires bat on PATH'
'''
        for name in ('apt-get','ln','go','make'):
            path=self.root/'bin'/name; path.write_text(stub); path.chmod(0o700)
        env=dict(self.env,PATH=str(self.root/'bin'))
        result=subprocess.run(['/bin/sh','-eu','-c',recipe],env=env,text=True,
                              stdout=subprocess.PIPE,stderr=subprocess.STDOUT)
        self.assertEqual(result.returncode,0,result.stdout)
        self.assertIn(['apt-get','install','-y','--no-install-recommends','python3','bat'],self.calls())
        self.assertEqual(self.calls()[-1],['make','init','check'])

    def test_check_runs_unit_once_and_no_program_or_pty_campaign(self):
        # Python discovery is empty in this fixture; Go tooling/checker fixtures
        # still run through actual recipes and all calls can be audited.
        result=self.make('check')
        self.assertEqual(result.returncode,0,result.stdout)
        calls=self.calls()
        race=[c for c in calls if '-race' in c]
        self.assertEqual(len(race),1)
        for args in calls:
            if './integration' in args or './integration/terminal' in args:
                self.assertIn('-run',args)
                self.assertTrue(args[args.index('-run')+1].startswith(('^TestCLIPackageBoundaries','^TestHarness')))
        self.env['FAIL_TOOL']='golangci-lint'
        self.log.unlink()
        self.assertNotEqual(self.make('check').returncode,0)
        self.assertTrue(any('-race' in c for c in self.calls()),'check must retain all useful results after a format failure')
