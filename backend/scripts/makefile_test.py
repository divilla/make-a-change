"""Exercise Make recipes and the legacy runner without external services."""

import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
import unittest


BACKEND = Path(__file__).resolve().parents[1]
UNIT = ["./cmd/...", "./internal/...", "./pkg/..."]
ALL = UNIT + ["./api-tests/..."]
STUB = r'''
import json, os, pathlib, sys, time
name = pathlib.Path(sys.argv[0]).name
args = sys.argv[1:]
root = pathlib.Path(os.environ["FIXTURE_ROOT"])
with open(os.environ["TOOL_LOG"], "a") as stream:
    stream.write(json.dumps([name, args, os.getcwd()]) + "\n")
if " ".join([name] + args).startswith(os.environ.get("FAIL_COMMAND", "__none__")):
    sys.exit(17)
if name == "go" and args[0] == "build":
    binary = pathlib.Path(args[args.index("-o") + 1])
    binary.write_text("#!" + sys.executable + "\n" + r"""
import os, pathlib, signal, time
root = pathlib.Path(os.environ['FIXTURE_ROOT'])
if os.environ.get('FAIL_SERVER'):
    raise SystemExit('server startup failed')
(root / 'started').write_text(str(os.getpid()))
def stop(*_):
    (root / 'stopped').touch()
    raise SystemExit(0)
signal.signal(signal.SIGTERM, stop)
while True:
    time.sleep(0.01)
""")
    binary.chmod(0o755)
if name == "go" and args[0] == "test":
    for arg in args:
        if arg.startswith("-coverprofile="):
            pathlib.Path(arg.split("=", 1)[1]).write_text(
                "mode: atomic\nmch_api/pkg/example.go:1.1,2.1 1 1\n")
if name == "curl":
    if os.environ.get("OCCUPIED"):
        sys.exit(0)
    sys.exit(0 if (root / "started").exists() and not os.environ.get("NOT_READY") else 7)
if name == "sleep":
    time.sleep(0.01)
'''


class MakefileTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix="backend-make-test-")
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.backend = self.root / "repo with spaces" / "backend"
        self.backend.mkdir(parents=True)
        (self.backend / "scripts").mkdir()
        shutil.copy2(BACKEND / "Makefile", self.backend)
        shutil.copy2(BACKEND / "scripts/run-api-tests.sh", self.backend / "scripts")
        (self.backend / "go.mod").write_text("module mch_api\n\ngo 1.26.0\n")
        self.bin = self.root / "bin"
        self.bin.mkdir()
        for tool in ["go", "golangci-lint", "govulncheck", "psql", "curl", "docker", "python3", "sleep"]:
            path = self.bin / tool
            path.write_text("#!" + sys.executable + "\n" + STUB)
            path.chmod(0o755)
        self.log = self.root / "tools.jsonl"
        self.scratch = self.root / "scratch with spaces"
        self.scratch.mkdir()
        self.env = dict(os.environ, PATH=str(self.bin) + os.pathsep + os.environ["PATH"],
                        TOOL_LOG=str(self.log), FIXTURE_ROOT=str(self.root), TMPDIR=str(self.scratch))
        # Parent make's options/jobserver must not affect the isolated child make.
        for key in ["MAKEFLAGS", "MFLAGS", "MAKELEVEL", "FAIL_COMMAND", "OCCUPIED", "NOT_READY", "FAIL_SERVER"]:
            self.env.pop(key, None)

    def make(self, *args, failure=False, **env):
        result = subprocess.run(["make", "--no-print-directory", *args], cwd=self.backend,
                                env=dict(self.env, **env), text=True, capture_output=True, timeout=20)
        if failure:
            self.assertNotEqual(result.returncode, 0, result.stdout + result.stderr)
        else:
            self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        return result

    def calls(self):
        if not self.log.exists():
            return []
        return [json.loads(line) for line in self.log.read_text().splitlines()]

    def commands(self):
        return [(name, args) for name, args, _ in self.calls()]

    def test_help_is_phony_and_does_not_resolve_packages_or_run_tools(self):
        (self.backend / "help").touch()
        result = self.make()
        self.assertEqual(result.stdout, self.make("help").stdout)
        for target in ["format-check", "coverage", "deps-audit", "tooling-test", "api-test"]:
            self.assertIn(target, result.stdout)
        self.assertEqual(self.calls(), [])

    def test_install_pins_both_tools_without_editing_module(self):
        original = (self.backend / "go.mod").read_bytes()
        self.make("init")
        self.assertEqual(self.commands(), [
            ("go", ["install", "github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.13.1"]),
            ("go", ["install", "golang.org/x/vuln/cmd/govulncheck@v1.7.0"])])
        self.assertEqual((self.backend / "go.mod").read_bytes(), original)

    def test_read_only_check_runs_all_checks_even_with_target_files(self):
        for target in ["check", "format-check", "lint", "vet", "race", "tooling-test"]:
            (self.backend / target).touch()
        self.make("-j4", "check")
        commands = self.commands()
        self.assertEqual(len(commands), 5)
        self.assertIn(("go", ["vet"] + ALL), commands)
        self.assertIn(("go", ["test", "-short", "-count=1", "-race"] + UNIT), commands)
        fmt = next(args for name, args in commands if name == "golangci-lint" and args[0] == "fmt")
        self.assertIn("--diff", fmt)
        self.assertIn(("python3", ["-B", "-m", "unittest", "discover", "-s", "scripts", "-p", "*_test.py", "-v"]), commands)
        self.assertFalse(any(name in ["psql", "curl"] or "--fix" in args for name, args in commands))

    def test_format_is_explicit_and_includes_backend_test_harness(self):
        self.make("format")
        self.assertEqual(self.commands(), [("golangci-lint", ["fmt", "--no-config", "--enable", "gofumpt", "--enable", "goimports"] + ALL)])

    def test_unit_and_benchmark_targets_do_not_run_integration_tests(self):
        self.make("test", "benchmark")
        self.assertEqual(self.commands(), [
            ("go", ["test", "-short", "-count=1"] + UNIT),
            ("go", ["test", "-short", "-run=^$", "-bench=.", "-benchmem"] + UNIT)])

    def test_audit_uses_all_backend_packages(self):
        self.make("deps-audit")
        self.assertEqual(self.commands(), [("govulncheck", ALL)])

    def test_every_check_propagates_tool_failure(self):
        for target, command in [("format-check", "golangci-lint fmt"), ("lint", "golangci-lint run"),
                                ("vet", "go vet"), ("race", "go test"), ("tooling-test", "python3"),
                                ("test", "go test"), ("deps-audit", "govulncheck")]:
            with self.subTest(target=target):
                self.make(target, failure=True, FAIL_COMMAND=command)
        self.make("check", failure=True, FAIL_COMMAND="go vet")

    def test_coverage_instruments_all_production_packages_and_generates_html_last(self):
        self.make("coverage-html")
        commands = self.commands()
        self.assertEqual(commands[0], ("go", ["test", "-short", "-count=1", "-race", "-covermode=atomic",
            "-coverpkg=./cmd/...,./internal/...,./pkg/...", "-coverprofile=.coverage/unit/coverage.out"] + UNIT))
        self.assertEqual(commands[1], ("go", ["tool", "cover", "-func=.coverage/unit/coverage.out"]))
        self.assertEqual(commands[2], ("go", ["tool", "cover", "-html=.coverage/unit/coverage.out", "-o", ".coverage/unit/coverage.html"]))

    def test_failed_coverage_removes_stale_reports_and_never_reports_success(self):
        directory = self.backend / ".coverage/unit"
        directory.mkdir(parents=True)
        for name in ["coverage.out", "coverage.html"]:
            (directory / name).write_text("stale success")
        self.make("coverage-html", failure=True, FAIL_COMMAND="go test")
        self.assertEqual(len(self.calls()), 1)
        self.assertEqual(list(directory.iterdir()), [])

    def test_docker_version_follows_module_and_volume_path_is_one_argument(self):
        self.make("test_version")
        name, args = self.commands()[0]
        self.assertEqual(name, "docker")
        self.assertIn(str(self.backend) + ":/project", args)
        self.assertIn("golang:1.26.0", args)
        self.assertNotIn("-it", args)
        self.assertIn("command -v python3", args[-1])
        self.make("test_version", "goversion=1.27")
        self.assertIn("golang:1.27", self.commands()[-1][1])

    def test_import_quotes_connection_and_stops_after_failed_schema(self):
        url = "postgres://localhost/custom?application_name=two words"
        self.make("import-db", "DATABASE_URL=" + url)
        self.assertEqual(self.commands(), [("psql", [url, "-v", "ON_ERROR_STOP=1", "-f", "../db/" + name])
                                         for name in ["init.sql", "seed.sql", "seed-demo.sql"]])
        self.log.unlink()
        self.make("import-db", failure=True, FAIL_COMMAND="psql")
        self.assertEqual(len(self.calls()), 1)

    def test_api_runner_rejects_non_test_database_and_database_query_override(self):
        for url in ["postgres://localhost/changes", "postgres://localhost/changes_test_other",
                    "postgres://localhost/changes_test?dbname=changes", "postgres://localhost/changes_test?db%6eame=changes"]:
            with self.subTest(url=url):
                self.make("api-test", "API_TEST_DB_URL=" + url, failure=True)
                self.assertEqual(self.calls(), [])

    def test_api_build_failure_precedes_database_reset(self):
        self.make("api-test", failure=True, FAIL_COMMAND="go build")
        self.assertEqual([name for name, _, _ in self.calls()], ["go"])
        self.assertEqual(list(self.scratch.iterdir()), [])

    def test_api_occupied_service_is_not_killed_or_reset(self):
        self.make("api-test", failure=True, OCCUPIED="1")
        self.assertEqual([name for name, _, _ in self.calls()], ["go", "curl"])
        self.assertFalse((self.root / "started").exists())

    def test_api_schema_failure_prevents_server_and_test_start(self):
        self.make("api-test", failure=True, FAIL_COMMAND="psql")
        self.assertEqual([name for name, _, _ in self.calls()], ["go", "curl", "psql"])
        self.assertFalse((self.root / "started").exists())
        self.assertEqual(list(self.scratch.iterdir()), [])

    def test_api_success_owns_process_and_preserves_existing_binary(self):
        binary = self.backend / "mch-server"
        binary.write_text("user binary")
        self.make("api-test", "API_TEST_DB_URL=postgres://localhost/changes_test?sslmode=disable")
        self.assertIn(("go", ["test", "-count=1", "./api-tests/..."]), self.commands())
        self.assertEqual(sum(name == "psql" for name, _, _ in self.calls()), 2)
        self.assertTrue((self.root / "stopped").exists())
        self.assertEqual(binary.read_text(), "user binary")
        self.assertEqual(list(self.scratch.iterdir()), [])

    def test_api_test_failure_still_cleans_up_and_fails(self):
        self.make("api-test", failure=True, FAIL_COMMAND="go test")
        self.assertTrue((self.root / "stopped").exists())
        self.assertEqual(list(self.scratch.iterdir()), [])

    def test_api_server_startup_failure_is_reported_without_running_tests(self):
        result = self.make("api-test", failure=True, FAIL_SERVER="1")
        self.assertIn("server startup failed", result.stderr)
        self.assertFalse(any(name == "go" and args[0] == "test" for name, args in self.commands()))
        self.assertEqual(list(self.scratch.iterdir()), [])

    def test_api_timeout_fails_without_running_tests(self):
        result = self.make("api-test", failure=True, NOT_READY="1")
        self.assertIn("Timed out", result.stderr)
        self.assertFalse(any(name == "go" and args[0] == "test" for name, args in self.commands()))
        self.assertTrue((self.root / "stopped").exists())
        self.assertEqual(list(self.scratch.iterdir()), [])


if __name__ == "__main__":
    unittest.main()
