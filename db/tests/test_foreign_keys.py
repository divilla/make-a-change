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


class DatabaseTestCase(unittest.TestCase):
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
        self.fail("expected a transaction to wait for a row lock")


    def create_project(self):
        return self.sql("insert into project(name) values('test project') returning id;")

    def create_case(self, change):
        return self.sql(f"insert into testcase(change_id,scenario) values({change},'original') returning id;")

    def assert_counts(self, change, done, total, epic=None):
        for view in ("vw_change_list", "vw_change_details"):
            self.assertEqual(self.sql(f"select done_tc,total_tc from {view} where id={change};"),
                             f"{done}|{total}")
        if epic is not None:
            self.assertEqual(self.sql(f"select done_tc,total_tc from vw_epic where id={epic};"),
                             f"{done}|{total}")


class ForeignKeyTests(DatabaseTestCase):
    def test_project_configuration_has_no_foreign_key(self):
        project = self.create_project()
        self.assertEqual(self.sql(f"select config from project where id={project};"), "default")
        # init.sql intentionally permits config slugs without a matching row.
        self.assertEqual(self.sql(f"begin; truncate config; "
                                  f"update project set config='missing' where id={project}; "
                                  f"select config from project where id={project}; rollback;"), "missing")

    def test_foreign_key_constraints(self):
        self.run_file(self.db / "tests" / "foreign_keys.sql")
        self.assertEqual(self.sql("select count(*) from vw_foreign_key;"), "4")

    def test_change_epic_assignment_and_counts(self):
        self.run_file(self.db / "tests" / "change_epic.sql")

    def test_document_history_and_missing_parents(self):
        self.run_file(self.db / "tests" / "change_history.sql")

    def test_testcase_lifecycle(self):
        self.run_file(self.db / "tests" / "test_case.sql")

    def test_seed_can_replace_existing_data_twice(self):
        project = self.create_project()
        change = self.sql(f"select fn_change_insert({project},gen_random_uuid(),'existing','initial');")
        self.create_case(change)
        self.sql(f"call sp_project_doc_set({project},'brief','old project',false);")
        for run in range(2):
            with self.subTest(seed_run=run + 1):
                maxima = {table: int(self.sql(f"select coalesce(max(id),0) from {table};"))
                          for table in ("project", "change", "testcase", "doc")}
                self.run_file(self.db / "seed.sql")
                self.run_file(self.db / "seed-demo.sql")
                self.run_file(self.db / "tests" / "seed.sql")
                for table, previous in maxima.items():
                    self.assertGreater(int(self.sql(f"select min(id) from {table};")), previous)

    def test_seed_demo_stores_suffix_and_view_builds_ref_slug(self):
        self.run_file(self.db / "seed-demo.sql")
        for view in ("vw_change_list", "vw_change_details"):
            self.assertEqual(
                self.sql(f"select ref_slug from {view} where title='Respect q=0 in gzip content negotiation';"),
                "201-respect-q-0-in-gzip-content-negotiation")
        self.assertEqual(
            self.sql("select slug from change where title='Respect q=0 in gzip content negotiation';"),
            "respect-q-0-in-gzip-content-negotiation")

    def test_ref_slug_padding_and_suffix_in_both_views(self):
        project = self.create_project()
        for ref, expected in ((6, "006-some-slug"), (99, "099-some-slug"),
                              (100, "100-some-slug"), (1116, "1116-some-slug")):
            change = self.sql(f"select fn_change_insert({project},gen_random_uuid(),'title','brief');")
            self.sql(f"update change set ref={ref},slug='some-slug' where id={change};")
            for view in ("vw_change_list", "vw_change_details"):
                self.assertEqual(self.sql(f"select ref_slug from {view} where id={change};"), expected)

        missing_ref = self.sql(f"select fn_change_insert({project},gen_random_uuid(),'unassigned','brief');")
        for view in ("vw_change_list", "vw_change_details"):
            self.assertEqual(self.sql(f"select ref_slug is null from {view} where id={missing_ref};"), "t")

    def test_after_change_name_in_details_view(self):
        project = self.create_project()
        prior = self.sql(f"select fn_change_insert({project},gen_random_uuid(),'First change','brief');")
        current = self.sql(f"select fn_change_insert({project},gen_random_uuid(),'Current change','brief');")
        self.assertEqual(self.sql(f"select after_change_name is null from vw_change_details where id={current};"), "t")
        self.sql(f"update change set after_change_id={prior} where id={current};")
        self.assertEqual(self.sql(f"select after_change_name from vw_change_details where id={current};"),
                         f"First change (#{prior})")

    def test_details_epic_name_is_only_the_name(self):
        project = self.create_project()
        epic = self.sql(f"insert into epic(project_id,name) values({project},'Roadmap') returning id;")
        change = self.sql(f"select fn_change_insert({project},gen_random_uuid(),'Current change','brief');")
        self.sql(f"update change set epic_id={epic} where id={change};")
        self.assertEqual(self.sql(f"select epic_name from vw_change_details where id={change};"), "Roadmap")

    def test_change_creation_is_atomic_and_uuid_unique(self):
        project = self.create_project()
        change = self.sql(f"select fn_change_insert({project},gen_random_uuid(),'title','initial');")
        self.assertEqual(self.sql(f"select title,open,change_phase from change where id={change};"),
                         "title|t|backlog")
        self.sql(f"""
            do $$ begin
                begin
                    perform fn_change_insert({project}, (select ref_uuid from change where id={change}), 'duplicate', 'brief');
                    raise exception 'duplicate UUID accepted';
                exception when unique_violation then null;
                end;
                begin
                    perform fn_change_insert({project}, gen_random_uuid(), 'invalid body', null);
                    raise exception 'null body accepted';
                exception when not_null_violation then null;
                end;
            end $$;
        """)
        self.assertEqual(self.sql(f"select count(*) from change where project_id={project};"), "1")
        self.assertEqual(self.sql(f"select count(*) from doc where ref_table='change' and ref_id={change};"), "1")

    def test_change_title_and_phase_updates(self):
        project = self.create_project()
        change = self.sql(f"insert into change(project_id,updated_at) values({project},'2000-01-01') returning id;")
        self.sql(f"call sp_change_title_update({change}, E'  spaced  title\\nwith\\t tab  ');")
        self.assertEqual(self.sql(f"select title,updated_at > '2000-01-01' from change where id={change};"),
                         "spaced title with tab|t")
        self.sql(f"update change set updated_at='2000-01-01' where id={change}; "
                 f"call sp_change_phase_update({change},'todo');")
        self.assertEqual(self.sql(f"select change_phase,updated_at > '2000-01-01' from change where id={change};"),
                         "todo|t")
        self.sql("call sp_change_title_update(-1,'missing'); call sp_change_phase_update(-1,'todo');")

    def test_concurrent_project_delete_cannot_orphan_change(self):
        project = self.create_project()
        with self.session() as creator, self.session() as deleter:
            self.command(creator, f"select fn_change_insert({project},gen_random_uuid(),'race','brief');")
            pid = self.command(deleter, "select pg_backend_pid();")
            self.send(deleter, f"delete from project where id={project};")
            self.wait_for_lock(pid)
            self.command(creator, "commit;")
            deleter.wait(timeout=10)
            self.assertNotEqual(deleter.returncode, 0)
            self.assertIn("23503", deleter.stderr.read())
        self.assertEqual(self.sql(f"select count(*) from change c join project p on p.id=c.project_id where p.id={project};"), "1")

    def test_document_updates_after_concurrent_parent_deletion(self):
        for kind in ("project", "epic", "change"):
            with self.subTest(parent=kind):
                project = self.create_project()
                if kind == "project":
                    parent = project
                elif kind == "epic":
                    parent = self.sql(f"insert into epic(project_id,name) values({project},'race') returning id;")
                else:
                    parent = self.sql(f"insert into change(project_id) values({project}) returning id;")
                self.sql(f"call sp_{kind}_doc_set({parent},'brief','initial',false);")
                with self.session() as deleter, self.session() as editor:
                    self.command(deleter, f"delete from {kind} where id={parent};")
                    pid = self.command(editor, "select pg_backend_pid();")
                    self.send(editor, f"call sp_{kind}_doc_set({parent},'brief','edited',false);")
                    self.wait_for_lock(pid)
                    self.command(deleter, "commit;")
                    self.receive(editor)
                    self.command(editor, "commit;")
                self.assertEqual(self.sql(f"select body from doc where ref_table='{kind}' and ref_id={parent};"), "initial")

    def test_concurrent_document_submissions_preserve_both_bodies(self):
        project = self.create_project()
        epic = self.sql(f"insert into epic(project_id,name) values({project},'documents') returning id;")
        change = self.sql(f"insert into change(project_id) values({project}) returning id;")
        for kind, parent in (("project", project), ("epic", epic), ("change", change)):
            with self.subTest(parent=kind), self.session() as first, self.session() as second:
                self.command(first, f"call sp_{kind}_doc_set({parent},'brief','first',true);")
                pid = self.command(second, "select pg_backend_pid();")
                self.send(second, f"call sp_{kind}_doc_set({parent},'brief','second',false);")
                self.wait_for_lock(pid)
                self.command(first, "commit;")
                self.receive(second)
                self.command(second, "commit;")
            self.assertEqual(self.sql(f"select body,agent_edit from doc where ref_table='{kind}' "
                                      f"and ref_id={parent} order by id;"), "first|t\nsecond|f")


if __name__ == "__main__":
    unittest.main()
