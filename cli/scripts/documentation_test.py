"""Keep the durable route/reuse ledger and local command documentation honest."""
from pathlib import Path
import re
import unittest

import coverage


class DocumentationTest(unittest.TestCase):
    def test_local_links_and_documented_targets(self):
        paths=[coverage.CLI/'agents/cli-contracts.md',coverage.CLI/'agents/cli-rebuild-checkpoint.md',
               coverage.CLI.parent/'AGENTS.md',coverage.CLI.parent/'agent/cli-rebuild-plan.md']
        make=(coverage.CLI/'Makefile').read_text()
        for path in paths:
            text=path.read_text()
            for target in re.findall(r'make -C cli ([a-z][a-z_-]+)',text):
                self.assertRegex(make,rf'(?m)^{re.escape(target)}:')
            for link in re.findall(r'\]\(([^)]+)\)',text):
                if '://' not in link and not link.startswith('#'):
                    self.assertTrue((path.parent/link.split('#')[0]).exists(),str(path)+': '+link)

    def test_backend_route_inventory_and_integration_assertion_ledger(self):
        ledger=(coverage.CLI/'agents/cli-contracts.md').read_text()
        documented=set(re.findall(r'\| (GET|POST) \| `([^`]+)`',ledger))
        actual=set()
        for group in ['project','epic','change','testcase','doc','config','health']:
            source=(coverage.CLI.parent/f'backend/internal/{group}/api.go').read_text()
            prefix='test-case' if group=='testcase' else group
            for method,route in re.findall(r'a\.g\.(GET|POST)\("([^"]*)"',source):
                actual.add((method,'/api/v1/'+prefix+route))
            actual.update(re.findall(r'e\.(GET|POST)\("([^"]*)"',source))
        self.assertEqual(documented,actual)
        for path in (coverage.CLI/'integration').rglob('*_test.go'):
            for name in re.findall(r'^func (Test\w+)\(',path.read_text(),re.M):
                self.assertIn('`'+name+'`',ledger)

    def test_verification_runner_is_not_ignored(self):
        # The repository-wide coverage.* rule must not hide the executable
        # runner from publication or its own provenance inventory.
        import subprocess
        result = subprocess.run(['git','check-ignore','-q','scripts/coverage.py'],cwd=coverage.CLI)
        self.assertEqual(result.returncode,1)
