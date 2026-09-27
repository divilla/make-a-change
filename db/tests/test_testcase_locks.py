"""Test testcase row and foreign-key locks in an isolated PostgreSQL cluster.

Run: python3 -B db/tests/test_testcase_locks.py -v
"""

import unittest

from test_foreign_keys import DatabaseTestCase


class TestcaseLockTests(DatabaseTestCase):
    def create_change(self, epic=None):
        if epic is None:
            project = self.create_project()
            epic = self.sql(f"insert into epic(project_id,name) values({project},'locks') returning id;")
        change = self.sql(f"insert into change(project_id,epic_id) "
                          f"select project_id,id from epic where id={epic} returning id;")
        return change, epic

    def test_batch_delete_and_concurrent_insert_preserve_counts(self):
        change, epic = self.create_change()
        first, second = self.create_case(change), self.create_case(change)
        with self.session() as deleter, self.session() as writer:
            self.command(deleter, f"delete from testcase where id={first};")
            inserted = self.command(writer, f"insert into testcase(change_id,scenario) values({change},'new') returning id;")
            self.command(deleter, f"delete from testcase where id={second};")
            self.command(deleter, "commit;")
            self.command(writer, "commit;")
        self.assertEqual(self.sql(f"select id,scenario from testcase where change_id={change};"), f"{inserted}|new")
        self.assert_counts(change, 0, 1, epic)

    def test_delete_waiting_for_edit_deletes_current_row(self):
        change, epic = self.create_change()
        case = self.create_case(change)
        with self.session() as first, self.session() as second:
            self.command(first, f"update testcase set scenario='edited' where id={case};")
            pid = self.command(second, "select pg_backend_pid();")
            self.send(second, f"delete from testcase where id={case} returning scenario;")
            self.wait_for_lock(pid)
            self.command(first, "commit;")
            self.assertEqual(self.receive(second), "edited")
            self.command(second, "commit;")
        self.assert_counts(change, 0, 0, epic)

    def test_concurrent_inserts_count_same_change_and_shared_epic(self):
        for same_change in (True, False):
            with self.subTest(same_change=same_change):
                change, epic = self.create_change()
                other = change if same_change else self.create_change(epic)[0]
                with self.session() as first, self.session() as second:
                    self.command(first, f"insert into testcase(change_id,scenario) values({change},'first');")
                    self.command(second, f"insert into testcase(change_id,scenario) values({other},'second');")
                    self.assert_counts(change, 0, 0, epic)
                    self.command(first, "commit;")
                    self.command(second, "commit;")
                self.assert_counts(change, 0, 2 if same_change else 1)
                self.assert_counts(other, 0, 2 if same_change else 1)
                self.assertEqual(self.sql(f"select total_tc from vw_epic where id={epic};"), "2")

    def test_missing_cases_are_noops(self):
        self.assertEqual(self.sql("update testcase set scenario='missing' where id=-1 returning id; "
                                  "update testcase set done=true where id=-1 returning id; "
                                  "delete from testcase where id=-1 returning id;"), "")

    def test_insert_after_parent_deletion_reports_foreign_key_violation(self):
        change, _ = self.create_change()
        with self.session() as deleter, self.session() as creator:
            self.command(deleter, f"delete from change where id={change};")
            pid = self.command(creator, "select pg_backend_pid();")
            self.send(creator, f"insert into testcase(change_id,scenario) values({change},'missing parent');")
            self.wait_for_lock(pid)
            self.command(deleter, "commit;")
            creator.wait(timeout=10)
            self.assertNotEqual(creator.returncode, 0)
            self.assertIn("23503", creator.stderr.read())
        self.assertEqual(self.sql(f"select count(*) from testcase where change_id={change};"), "0")

    def test_mutations_after_concurrent_parent_deletion_are_noops(self):
        for operation in ("scenario", "done", "delete"):
            with self.subTest(operation=operation):
                change, _ = self.create_change()
                case = self.create_case(change)
                statement = {
                    "scenario": f"update testcase set scenario='edited' where id={case} returning id;",
                    "done": f"update testcase set done=true where id={case} returning id;",
                    "delete": f"delete from testcase where id={case} returning id;",
                }[operation]
                with self.session() as deleter, self.session() as writer:
                    self.command(deleter, f"delete from testcase where id={case}; delete from change where id={change};")
                    pid = self.command(writer, "select pg_backend_pid();")
                    self.send(writer, statement)
                    self.wait_for_lock(pid)
                    self.command(deleter, "commit;")
                    self.assertEqual(self.receive(writer), "")
                    self.command(writer, "commit;")
                self.assertEqual(self.sql(f"select count(*) from testcase where id={case};"), "0")

    def test_concurrent_deletes_remove_row_once(self):
        change, epic = self.create_change()
        case = self.create_case(change)
        with self.session() as first, self.session() as second:
            self.assertEqual(self.command(first, f"delete from testcase where id={case} returning id;"), case)
            pid = self.command(second, "select pg_backend_pid();")
            self.send(second, f"delete from testcase where id={case} returning id;")
            self.wait_for_lock(pid)
            self.command(first, "commit;")
            self.assertEqual(self.receive(second), "")
            self.command(second, "commit;")
        self.assert_counts(change, 0, 0, epic)

    def test_schema_can_be_reinitialized(self):
        change, _ = self.create_change()
        self.create_case(change)
        self.sql(f"call sp_change_doc_set({change},'brief','old',false);")
        self.run_file(self.db / "init.sql")
        self.run_file(self.db / "seed.sql")
        self.assertEqual(self.sql("select count(*) from doc; select count(*) from testcase;"), "0\n0")
        project = self.create_project()
        change = self.sql(f"select fn_change_insert({project},gen_random_uuid(),'new','new brief');")
        self.assertEqual(self.sql(f"select body from doc where ref_table='change' and ref_id={change};"), "new brief")


if __name__ == "__main__":
    unittest.main()
