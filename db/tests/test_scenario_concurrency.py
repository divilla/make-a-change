"""Test ordinary testcase edits in a disposable PostgreSQL cluster.

Run: python3 -B db/tests/test_scenario_concurrency.py -v
"""

import unittest

from test_foreign_keys import DatabaseTestCase


class ScenarioConcurrencyTests(DatabaseTestCase):
    def setUp(self):
        project = self.create_project()
        self.change = self.sql(f"insert into change(project_id) values({project}) returning id;")
        self.case = self.create_case(self.change)

    def test_concurrent_edits_apply_in_lock_order(self):
        for second_body in ("first", "second"):
            with self.subTest(second_body=second_body), self.session() as first, self.session() as second:
                self.command(first, f"update testcase set scenario='first' where id={self.case};")
                pid = self.command(second, "select pg_backend_pid();")
                self.send(second, f"update testcase set scenario='{second_body}' where id={self.case} returning scenario;")
                self.wait_for_lock(pid)
                self.command(first, "commit;")
                self.assertEqual(self.receive(second), second_body)
                self.command(second, "commit;")
            self.assertEqual(self.sql(f"select scenario from testcase where id={self.case};"), second_body)
        self.assert_counts(self.change, 0, 1)

    def test_edit_waiting_for_delete_updates_no_rows(self):
        with self.session() as deleter, self.session() as editor:
            self.command(deleter, f"delete from testcase where id={self.case};")
            pid = self.command(editor, "select pg_backend_pid();")
            self.send(editor, f"update testcase set scenario='late edit' where id={self.case} returning id;")
            self.wait_for_lock(pid)
            self.command(deleter, "commit;")
            self.assertEqual(self.receive(editor), "")
            self.command(editor, "commit;")
        self.assertEqual(self.sql(f"select count(*) from testcase where id={self.case};"), "0")
        self.assert_counts(self.change, 0, 0)

    def test_edit_proceeds_after_competing_edit_rolls_back(self):
        with self.session() as first, self.session() as second:
            self.command(first, f"update testcase set scenario='rolled back' where id={self.case};")
            pid = self.command(second, "select pg_backend_pid();")
            self.send(second, f"update testcase set scenario='committed' where id={self.case} returning scenario;")
            self.wait_for_lock(pid)
            self.command(first, "rollback;")
            self.assertEqual(self.receive(second), "committed")
            self.command(second, "commit;")
        self.assertEqual(self.sql(f"select scenario from testcase where id={self.case};"), "committed")


if __name__ == "__main__":
    unittest.main()
