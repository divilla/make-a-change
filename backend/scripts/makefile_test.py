"""Exercise Make recipes without external services."""

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
ALL = UNIT
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
        (self.backend / "go.mod").write_text("module mch_api\n\ngo 1.26.0\n")
        self.bin = self.root / "bin"
        self.bin.mkdir()
        for tool in ["go", "golangci-lint", "govulncheck", "psql", "curl", "docker", "python3", "sleep", "apih"]:
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
        self.assertEqual(len(commands), 6)
        self.assertIn(("go", ["vet"] + ALL), commands)
        self.assertIn(("go", ["test", "-short", "-count=1", "-race"] + UNIT), commands)
        self.assertIn(("go", ["test", "-count=1", "./scripts/validate-apih-suite"]), commands)
        fmt = next(args for name, args in commands if name == "golangci-lint" and args[0] == "fmt")
        self.assertIn("--diff", fmt)
        self.assertIn(("python3", ["-B", "-m", "unittest", "discover", "-s", "scripts", "-p", "*_test.py", "-v"]), commands)
        self.assertFalse(any(name in ["psql", "curl"] or "--fix" in args for name, args in commands))

    def test_format_is_explicit_and_includes_all_backend_go(self):
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

    def test_coverage_targets_delegate_to_strict_runner(self):
        self.make("coverage")
        self.make("coverage-html")
        self.assertEqual(self.commands(), [
            ("python3", ["-B", "scripts/coverage.py"]),
            ("python3", ["-B", "scripts/coverage.py", "--html"])])
        self.make("coverage", failure=True, FAIL_COMMAND="python3")

    def test_api_target_runs_coverage_driver_without_database_setup(self):
        self.make("api-test")
        self.assertEqual(self.commands(), [("python3", ["-B", "scripts/api_coverage.py"])])
        self.make("api-test", failure=True, FAIL_COMMAND="python3")

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



if __name__ == "__main__":
    unittest.main()
