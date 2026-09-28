from pathlib import Path
import re
import unittest

BACKEND=Path(__file__).resolve().parents[1]


class ContractsTest(unittest.TestCase):
    def test_all_registered_routes_remain_in_ledger_denominator(self):
        routes=set()
        for file in (BACKEND/'internal').glob('*/api.go'):
            text=file.read_text()
            groups=re.findall(r'\.Group\("([^"]*)"\)',text)
            prefix=''.join(groups)
            for receiver, method, path in re.findall(r'(a\.g|e)\.(GET|POST)\("([^"]*)"',text):
                routes.add((method,(prefix if receiver=='a.g' else '')+path))
        ledger=(BACKEND/'agents/backend-contracts.md').read_text()
        documented=set(re.findall(r'^\| (GET|POST) \| (\S+) \|',ledger,re.M))
        self.assertEqual(len(routes),33)
        self.assertEqual(routes,documented)

    def test_health_suite_has_explicit_contracts_for_both_aliases(self):
        text=(BACKEND/'apih-tests/health-steps.yaml').read_text()
        self.assertEqual(re.findall(r'path: (\S+)',text),['/api/v1/health','/api/health'])
        self.assertEqual(text.count('expected_status: 200'),2)
        self.assertEqual(text.count('expected_body:'),2)
        self.assertNotIn('debug:',text)
