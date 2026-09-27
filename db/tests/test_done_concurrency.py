"""Test testcase completion and view counts in a disposable PostgreSQL cluster.

Run: python3 -B db/tests/test_done_concurrency.py -v
"""

import unittest

from test_foreign_keys import DatabaseTestCase


class DoneConcurrencyTests(DatabaseTestCase):
    def setUp(self):
        self.project = self.create_project()
        self.epic = self.sql(f"insert into epic(project_id,name) values({self.project},'done race') returning id;")
        self.change = self.sql(f"insert into change(project_id,epic_id) values({self.project},{self.epic}) returning id;")
        self.case = self.create_case(self.change)

    def test_same_case_true_then_false(self):
        with self.session() as first, self.session() as second:
            self.command(first, f"update testcase set done=true where id={self.case};")
            pid = self.command(second, "select pg_backend_pid();")
            self.send(second, f"update testcase set done=false where id={self.case};")
            self.wait_for_lock(pid)
            self.command(first, "commit;")
            self.receive(second)
            self.command(second, "commit;")
        self.assertEqual(self.sql(f"select done from testcase where id={self.case};"), "f")
        self.assert_counts(self.change, 0, 1, self.epic)

    def test_different_cases_update_without_parent_serialization(self):
        for placement in ("same change", "shared epic", "standalone"):
            with self.subTest(placement=placement):
                if placement == "same change":
                    change = self.change
                else:
                    epic = self.epic if placement == "shared epic" else "null"
                    change = self.sql(f"insert into change(project_id,epic_id) values({self.project},{epic}) returning id;")
                other = self.create_case(change)
                self.sql(f"update testcase set done=false where id={self.case};")
                with self.session() as first, self.session() as second:
                    self.command(first, f"update testcase set done=true where id={self.case};")
                    # A different testcase can be updated before the first commits.
                    self.command(second, f"update testcase set done=true where id={other};")
                    self.command(first, "commit;")
                    self.command(second, "commit;")
                self.assertEqual(self.sql(f"select count(*) from testcase where id in ({self.case},{other}) and done;"), "2")
                self.assert_counts(change, 2 if change == self.change else 1, 2 if change == self.change else 1)
                self.sql(f"delete from testcase where id={other};")
        self.assert_counts(self.change, 1, 1, self.epic)

    def test_done_and_scenario_edit_both_orders(self):
        for done_first in (True, False):
            with self.subTest(done_first=done_first):
                case = self.create_case(self.change)
                statements = [f"update testcase set done=true where id={case};",
                              f"update testcase set scenario='edited' where id={case};"]
                if not done_first:
                    statements.reverse()
                with self.session() as first, self.session() as second:
                    self.command(first, statements[0])
                    pid = self.command(second, "select pg_backend_pid();")
                    self.send(second, statements[1])
                    self.wait_for_lock(pid)
                    self.command(first, "commit;")
                    self.receive(second)
                    self.command(second, "commit;")
                self.assertEqual(self.sql(f"select done,scenario from testcase where id={case};"), "t|edited")
        self.assert_counts(self.change, 2, 3, self.epic)

    def test_done_and_delete_both_orders(self):
        for done_first in (True, False):
            with self.subTest(done_first=done_first):
                case = self.create_case(self.change)
                statements = [f"update testcase set done=true where id={case} returning id;",
                              f"delete from testcase where id={case} returning id;"]
                if not done_first:
                    statements.reverse()
                with self.session() as first, self.session() as second:
                    self.command(first, statements[0])
                    pid = self.command(second, "select pg_backend_pid();")
                    self.send(second, statements[1])
                    self.wait_for_lock(pid)
                    self.command(first, "commit;")
                    self.assertEqual(self.receive(second), case if done_first else "")
                    self.command(second, "commit;")
                self.assertEqual(self.sql(f"select count(*) from testcase where id={case};"), "0")
                self.assert_counts(self.change, 0, 1, self.epic)

    def test_rolled_back_done_is_not_counted(self):
        with self.session() as writer:
            self.command(writer, f"update testcase set done=true where id={self.case};")
            self.assert_counts(self.change, 0, 1, self.epic)
        self.assert_counts(self.change, 0, 1, self.epic)


if __name__ == "__main__":
    unittest.main()
