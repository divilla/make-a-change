"""Run with python3 db/tests/test_foreign_keys.py; requires PostgreSQL binaries.

Creates and removes an isolated PostgreSQL cluster. No existing database is used.
"""

import contextlib
import os
from pathlib import Path
import subprocess
import tempfile
import time
import unittest


class ForeignKeyTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.directory = tempfile.TemporaryDirectory(prefix="foreign-key-tests-")
        cls.addClassCleanup(cls.directory.cleanup)
        root = Path(cls.directory.name)
        cls.data = str(root / "data")
        subprocess.run(["initdb", "-D", cls.data, "-A", "trust", "--no-locale"],
                       check=True, capture_output=True)
        subprocess.run(["pg_ctl", "-D", cls.data, "-l", str(root / "server.log"),
                        "-o", f"-k {root} -p 5432 -c listen_addresses=''", "-w", "start"],
                       check=True, capture_output=True)
        cls.addClassCleanup(subprocess.run,
                            ["pg_ctl", "-D", cls.data, "-m", "immediate", "-w", "stop"],
                            check=True, capture_output=True)
        cls.psql = ["psql", "-X", "-qAt", "-h", str(root), "-p", "5432", "-d", "postgres",
                    "-v", "ON_ERROR_STOP=1", "-v", "VERBOSITY=verbose"]
        cls.env = {key: value for key, value in os.environ.items() if not key.startswith("PG")}
        cls.env["PGOPTIONS"] = "-c statement_timeout=5000"
        cls.db = Path(__file__).resolve().parents[1]
        cls.run_file(cls.db / "init.sql")
        cls.run_file(cls.db / "seed.sql")

    @classmethod
    def run_file(cls, path):
        result = subprocess.run(cls.psql + ["-f", str(path)], env=cls.env,
                                text=True, capture_output=True, timeout=20)
        if result.returncode:
            raise AssertionError(result.stderr)

    def sql(self, statement):
        result = subprocess.run(self.psql, input=statement, env=self.env,
                                text=True, capture_output=True, timeout=10)
        self.assertEqual(result.returncode, 0, result.stderr)
        return result.stdout.strip()

    @contextlib.contextmanager
    def session(self):
        process = subprocess.Popen(self.psql, env=self.env, text=True,
                                   stdin=subprocess.PIPE, stdout=subprocess.PIPE,
                                   stderr=subprocess.PIPE)
        try:
            self.command(process, "begin;")
            yield process
        finally:
            if process.poll() is None:
                process.communicate("rollback;\n\\q\n", timeout=10)
            else:
                process.communicate(timeout=10)

    def send(self, process, statement):
        process.stdin.write(statement + "\n\\echo READY\n")
        process.stdin.flush()

    def receive(self, process):
        lines = []
        while True:
            line = process.stdout.readline()
            self.assertTrue(line, process.stderr.read() if not line else "")
            if line.strip() == "READY":
                return "\n".join(lines)
            lines.append(line.strip())

    def command(self, process, statement):
        self.send(process, statement)
        return self.receive(process)

    def wait_for_lock(self, pid):
        deadline = time.monotonic() + 3
        while time.monotonic() < deadline:
            if self.sql(f"select wait_event_type = 'Lock' from pg_stat_activity where pid = {pid}") == "t":
                return
            time.sleep(0.02)
        self.fail("expected a transaction to wait for its parent-row lock")

    def test_constraints_and_history(self):
        self.run_file(self.db / "tests" / "foreign_keys.sql")

    def test_change_and_epic_must_share_project(self):
        self.run_file(self.db / "tests" / "change_epic.sql")

    def test_existing_seed_and_test_case_flows(self):
        for run in range(2):
            with self.subTest(seed_run=run + 1):
                self.run_file(self.db / "seed.sql")
                self.run_file(self.db / "seed-demo.sql")
                self.run_file(self.db / "tests" / "seed.sql")
        self.run_file(self.db / "tests" / "test_case.sql")

    def test_concurrent_project_delete_cannot_orphan_change(self):
        project = self.sql("insert into project(name) values('race') returning id;")
        with self.session() as creator, self.session() as deleter:
            self.command(creator, f"select fn_change_insert({project},gen_random_uuid(),'race','brief');")
            pid = self.command(deleter, "select pg_backend_pid();")
            self.send(deleter, f"delete from project where id={project} "
                      f"and not exists(select from change where project_id={project}) "
                      f"and not exists(select from epic where project_id={project});")
            self.wait_for_lock(pid)
            self.command(creator, "commit;")
            deleter.wait(timeout=10)
            self.assertNotEqual(deleter.returncode, 0)
            self.assertIn("23503", deleter.stderr.read())
        self.assertEqual(self.sql(f"select count(*) from change c join project p on p.id=c.project_id where p.id={project}"), "1")

    def test_concurrent_test_case_creation_counts_without_deadlock(self):
        project = self.sql("insert into project(name) values('test case race') returning id;")
        epic = self.sql(f"insert into epic(project_id,name) values({project},'race') returning id;")
        change = self.sql(f"insert into change(project_id,epic_id) values({project},{epic}) returning id;")
        with self.session() as first, self.session() as second:
            # Direct SQL inserts also hold FK locks when counter updates begin.
            first_id = self.command(first, f"insert into test_case(change_id,scenario) values({change},'first') returning id;")
            second_id = self.command(second, f"insert into test_case(change_id,scenario) values({change},'second') returning id;")
            self.command(first, f"call sp_test_case_update_done({first_id},false);")
            # Insert/delete procedures must also tolerate the other session's FK lock.
            self.command(first, f"select fn_test_case_insert({change},'replacement');")
            self.command(first, f"call sp_test_case_delete({first_id});")
            pid = self.command(second, "select pg_backend_pid();")
            self.send(second, f"call sp_test_case_update_done({second_id},false);")
            self.wait_for_lock(pid)
            self.command(first, "commit;")
            self.receive(second)
            self.command(second, "commit;")
        self.assertEqual(self.sql(f"select count(*) from test_case where change_id={change}"), "2")
        self.assertEqual(self.sql(f"select total_tc from change where id={change}"), "2")
        self.assertEqual(self.sql(f"select total_tc from epic where id={epic}"), "2")

    def test_change_history_missing_and_existing_rows(self):
        self.run_file(self.db / "tests" / "change_history.sql")

    def test_change_document_updates_after_concurrent_deletion(self):
        project = self.sql("insert into project(name) values('document deletion race') returning id;")
        for doc_type in ["brief", "spec", "pr"]:
            with self.subTest(doc_type=doc_type):
                change = self.sql(f"select fn_change_insert({project},gen_random_uuid(),'race','initial');")
                with self.session() as deleter, self.session() as editor:
                    self.command(deleter, f"delete from change where id={change};")
                    pid = self.command(editor, "select pg_backend_pid();")
                    self.send(editor, f"call sp_change_{doc_type}_update({change},'edited',false);")
                    self.wait_for_lock(pid)
                    self.command(deleter, "commit;")
                    self.receive(editor)
                    self.command(editor, "commit;")
                self.assertEqual(self.sql(f"select count(*) from change_history where id={change}"), "1")

    def test_concurrent_epic_edits_archive_distinct_versions(self):
        project = self.sql("insert into project(name) values('epic history race') returning id;")
        epic = self.sql(f"insert into epic(project_id,name) values({project},'initial') returning id;")
        with self.session() as first, self.session() as second:
            self.command(first, f"call sp_epic_to_history({epic},false); "
                         f"update epic set name='first',version=version+1 where id={epic};")
            pid = self.command(second, "select pg_backend_pid();")
            self.send(second, f"call sp_epic_to_history({epic},false); "
                      f"update epic set name='second',version=version+1 where id={epic};")
            self.wait_for_lock(pid)
            self.command(first, "commit;")
            self.receive(second)
            self.command(second, "commit;")
        self.assertEqual(self.sql(f"select version,name from epic_history where id={epic} order by version"),
                         "0|initial\n1|first")
        self.assertEqual(self.sql(f"select version,name from epic where id={epic}"), "2|second")

    def test_concurrent_changes_can_recalculate_shared_epic(self):
        project = self.sql("insert into project(name) values('epic race') returning id;")
        epic = self.sql(f"insert into epic(project_id,name) values({project},'race') returning id;")
        with self.session() as first, self.session() as second:
            # Both uncommitted changes hold foreign-key locks on the same epic.
            first_change = self.command(first, f"insert into change(project_id,epic_id) values({project},{epic}) returning id;")
            second_change = self.command(second, f"insert into change(project_id,epic_id) values({project},{epic}) returning id;")
            self.command(first, f"select fn_test_case_insert({first_change},'first');")
            pid = self.command(second, "select pg_backend_pid();")
            self.send(second, f"select fn_test_case_insert({second_change},'second');")
            self.wait_for_lock(pid)
            self.command(first, "commit;")
            self.receive(second)
            self.command(second, "commit;")
        self.assertEqual(self.sql(f"select total_tc from epic where id={epic}"), "2")


if __name__ == "__main__":
    unittest.main()
