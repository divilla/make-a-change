"""Exercise backup/restore scripts against an isolated PostgreSQL cluster."""

import gzip
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile

from test_foreign_keys import DatabaseTestCase


class BackupRestoreTests(DatabaseTestCase):
    def setUp(self):
        directory = tempfile.TemporaryDirectory(prefix="backup-restore-tests-")
        self.addCleanup(directory.cleanup)
        self.root = Path(directory.name)
        self.bin = self.root / "bin"
        self.bin.mkdir()
        self.env = dict(self.env, PATH=str(self.bin) + os.pathsep + os.environ["PATH"],
                        TMPDIR=str(self.root))
        # Redirect only connection arguments; preserve the scripts' error and
        # transaction flags so tests exercise their actual restore behavior.
        for tool in ("psql", "pg_dump"):
            executable = shutil.which(tool)
            self.write_tool(tool, f"""#!{sys.executable}
import os
import sys
args = []
pending = iter(sys.argv[1:])
for arg in pending:
    if arg in ('-h', '--host', '-p', '--port', '-U', '--username', '-d', '--dbname'):
        next(pending)
    elif arg not in ('changes', 'changes_test'):
        args.append(arg)
os.execv({executable!r}, [{executable!r}, *args,
    '-h', {str(Path(self.data).parent)!r}, '-p', '5432', '-d', 'postgres'])
""")

    def write_tool(self, name, body):
        path = self.bin / name
        path.write_text(body)
        path.chmod(0o755)

    def run_script(self, name, *args):
        return subprocess.run(["bash", str(self.db / "backup" / name), *map(str, args)],
                              cwd=self.root, env=self.env, text=True,
                              capture_output=True, timeout=20)

    def assert_no_temporary_files(self):
        self.assertEqual(sorted(p.name for p in self.root.iterdir()), ["bin"])

    def test_failed_dump_does_not_publish_archive(self):
        self.write_tool("pg_dump", "#!/bin/sh\nprintf 'partial dump'\nexit 1\n")
        for script in ("backup.sh", "backup-test.sh"):
            with self.subTest(script=script):
                result = self.run_script(script)
                self.assertNotEqual(result.returncode, 0)
                self.assert_no_temporary_files()

    def test_failed_compression_does_not_publish_archive(self):
        self.write_tool("pg_dump", "#!/bin/sh\nprintf 'dump'\n")
        self.write_tool("gzip", "#!/bin/sh\ncat >/dev/null\nprintf 'partial gzip'\nexit 1\n")
        for script in ("backup.sh", "backup-test.sh"):
            with self.subTest(script=script):
                self.assertNotEqual(self.run_script(script).returncode, 0)
                self.assert_no_temporary_files()

    def test_backup_round_trip(self):
        self.sql("create table restore_round_trip(id integer primary key); "
                 "insert into restore_round_trip values(1);")
        for suffix in ("", "-test"):
            with self.subTest(suffix=suffix):
                result = self.run_script(f"backup{suffix}.sh")
                self.assertEqual(result.returncode, 0, result.stderr)
                archives = list(self.root.glob("*.sql.gz"))
                self.assertEqual(len(archives), 1)
                self.assertIn(b"restore_round_trip", gzip.decompress(archives[0].read_bytes()))
                # Restoration must also succeed when an object from the dump
                # is absent, despite ON_ERROR_STOP being enabled.
                self.sql("drop table restore_round_trip;")
                archive = self.root / "backup with spaces.sql.gz"
                archives[0].rename(archive)
                result = self.run_script(f"restore{suffix}.sh", archive)
                self.assertEqual(result.returncode, 0, result.stderr)
                self.assertEqual(self.sql("select id from restore_round_trip;"), "1")
                archive.unlink()
                self.assert_no_temporary_files()

    def test_sql_failure_rolls_back_entire_restore(self):
        self.sql("create table restore_atomic(id integer primary key); "
                 "insert into restore_atomic values(1);")
        archive = self.root / "invalid.sql.gz"
        archive.write_bytes(gzip.compress(b"delete from restore_atomic;\n"
                                          b"insert into restore_atomic values(2);\n"
                                          b"insert into restore_atomic values(2);\n"
                                          b"insert into restore_atomic values(3);\n"))
        for script in ("restore.sh", "restore-test.sh"):
            with self.subTest(script=script):
                result = self.run_script(script, archive)
                self.assertNotEqual(result.returncode, 0)
                self.assertIn("duplicate key", result.stderr)
                self.assertEqual(self.sql("select id from restore_atomic;"), "1")
                self.assertEqual(sorted(p.name for p in self.root.iterdir()),
                                 ["bin", "invalid.sql.gz"])

    def test_invalid_archive_never_starts_restore(self):
        self.write_tool("psql", "#!/bin/sh\ntouch psql-called\ncat >/dev/null\n")
        archive = self.root / "corrupt.sql.gz"
        # A truncated gzip stream can emit SQL before reporting its error.
        archive.write_bytes(gzip.compress(b"select 1;\n")[:-4])
        for script in ("restore.sh", "restore-test.sh"):
            for args in ([], [self.root / "missing.sql.gz"], [archive]):
                with self.subTest(script=script, args=args):
                    self.assertNotEqual(self.run_script(script, *args).returncode, 0)
                    self.assertEqual(sorted(p.name for p in self.root.iterdir()),
                                     ["bin", "corrupt.sql.gz"])


if __name__ == "__main__":
    import unittest
    unittest.main()
