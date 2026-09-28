# CLI rebuild checkpoint

## Supervisor handoff after final P0 review

Native review pass 04 of `06042fd` against pinned dev `178b8b0` exited 0 with no
findings. Review evidence: `/tmp/mch-cli-019-review.log`. Reviewer independently
passed tooling and completed both campaigns: unit **3060/4021**, terminal
**1796/4021**, with the strict numerical gates failing honestly. Production Go
is unchanged. Baseline static/legacy failures and Docker mounted-file permissions
remain documented below; this tooling pass does not establish overall CLI readiness.
The supervisor verified allowed-path scope and clean published branch state.
The implementation log intentionally retains its skill-required blank separator.
Next: merge 019 with the factory helper, verify actual dev SHA, then create the
P1 startup/Flow-removal specification. No future squash hash is asserted.

P0 [specification 019](../../agent/specs/019-cli-verification-foundation.md),
2026-09-28, branch `change/019-cli-verification-foundation`.
Base `origin/dev` at specification: `178b8b0`; tested working tree based on
`fadb0a8dce9575f983f939d3caa5673800ba28a8` (specification commit). No implementation
commit, push, merge or promotion performed by this child. The supervisor owns
review/publication. The [authorized plan](../../agent/cli-rebuild-plan.md) and
[contracts/reuse ledger](cli-contracts.md) remain the scope authorities.

## Review fixes 03

The SIGTERM P2 finding is fixed on working tree HEAD
`9335e7832842a6178719de1e4e4fe6d3c64da7ef`. The campaign installs SIGINT/SIGTERM
handlers before acquiring artifact ownership or launching detached commands.
Both signals enter the existing exception cleanup and journaling path; repeated
cancellation is ignored until descendant reaping, command/status journals and
lock release finish. Previous handlers are restored when the campaign unwinds.

The existing real-Go process-tree regression now covers both signals, including
TERM-ignoring detached/orphan descendants, lock ownership before release,
unrelated-process survival, partial diagnostics, incomplete status and immediate
lock reacquisition afterward. It also sends repeated SIGTERM/SIGINT during the
cleanup wait and before lock release. Handler restoration is tested on normal
exit and interruption. The three focused regression tests and `git diff --check`
pass; no production Go or backend contract changes were needed.

Final commands: `make -C cli tooling-test` exits 0 (36 Python tests plus
architecture and terminal harness fixtures). `make -C cli check` exits 2 for
37 pre-existing formatting files, 12 lint findings in unchanged Go files and the
existing agent-to-changes architecture violation; vet, unit race and tooling pass.
The linter's package-comment samples in this run are cmd/mch, agent and dto.
`make -C cli deps-audit` exits 0 with no vulnerabilities. `make -C cli coverage`
exits 2: **3060/4021 (76.1005%)**. `make -C cli integration-coverage` exits 2:
**1796/4021 (44.6655%)**. Both campaigns complete, all six program/startup scenarios
and the real PTY scenario pass without skips, and both strict numerical gates
remain failing. Package counts match review fixes 02; uncovered totals remain
961 unit / 2225 terminal, with full package/block/function gaps and provenance in
`cli/.coverage/{unit,integration}/`. Command logs/exits are retained under
`cli/.coverage/review-fixes-03/`. Checkpoint/log updates after measurement are
documentation-only. Docker was not rerun. No commits or pushes performed.

## Review fixes 02

The descendant-cleanup P2 finding is fixed on working tree HEAD
`7eb7b0fd33b96170c89c38c05938991ce97ad6dd`. Recorded commands now run under an
isolated Linux subreaper (`scripts/owned_command.py`). It adopts and kills/reaps
owned descendants even when Go tests or socat children create separate sessions
or orphan children. The caller waits for cleanup before releasing the campaign
lock, ignores repeated Ctrl-C during cleanup, and retains interruption diagnostics.
Normal command exits and signal exits are preserved; successful drivers cannot
leave orphaned descendants behind. This supervision requires Linux with procfs
and fails visibly before launching commands on unsupported hosts. Cleanup exceeding
five seconds is reported as a failure; the lock stays held until killed processes
are reaped, rather than permitting a contaminated rerun.

The SIGINT regression now executes a real Go test, creates a TERM-ignoring detached
session and an orphan, and verifies that the Go driver, test executable and all
recorded descendants are gone while the lock is still held. An unrelated process
survives, diagnostics/status remain incomplete, and an immediate fresh campaign
can acquire the lock. Additional regressions cover signal-exit preservation and
orphan cleanup after successful driver exit.

Final commands: `make -C cli tooling-test` exits 0 (34 Python tests plus architecture
and terminal harness fixtures); `make -C cli check` exits 2 for the same baseline
formatting, 12 lint findings and agent-to-changes architecture violation. Vet,
unit race and tooling pass. `make -C cli deps-audit` exits 0 (no vulnerabilities).
`make -C cli coverage` exits 2: **3060/4021 (76.1005%)**; terminal
`make -C cli integration-coverage` exits 2: **1796/4021 (44.6655%)**. Both campaigns
complete with failing numerical gates; all six program/startup scenarios and the
real PTY scenario pass without skips. Every package count matches review fixes 01;
terminal app remains 1037/2542 (1505 uncovered), and total uncovered counts are
961 unit / 2225 terminal. Detailed fresh package/block/function gaps and separate
profiles are in `cli/.coverage/{unit,integration}/`. Command logs and exits are in
`cli/.coverage/review-fixes-02/`. `git diff --check` passes. Docker was not rerun;
the prior mount-permission blocker remains. Checkpoint and implementation-log
updates after measurement are documentation-only. No commits or pushes performed.

## Review fixes 01

Both P2 findings validated and fixed on working tree HEAD
`92d7377161c78639fd6ae51a09158df65115d862`. Docker now mounts the repository root
and runs in `/project/cli`, exposing instructions, backend contracts and Git
metadata. Campaign subprocesses write stdout/stderr directly to disk while
running; interruption terminates/reaps the child and journals its exit and error
in cleanup. Regression tests verify root mount/workdir selection with spaces,
live partial-line output, and real SIGINT preserving logs and incomplete status.

Fresh verification: `make -C cli tooling-test` exits 0 (32 Python tests plus
architecture/harness fixtures); `make -C cli check` exits 2 with existing
formatting, 12 lint findings and the agent-to-changes architecture violation.
Vet, unit race and tooling pass. `make -C cli deps-audit` exits 0.
`make -C cli coverage` exits 2: **3060/4021 (76.1005%)**.
`make -C cli integration-coverage` exits 2: **1796/4021 (44.6655%)**; all six
program/startup scenarios and the PTY scenario pass without skips. Both campaigns
are complete with honestly failing numerical gates. Fresh terminal app coverage
is 1037/2542 (1505 uncovered), one statement below the initial measurement below;
all other package counts are unchanged. Unit and terminal counters stay separate.

`make -C cli test_version` exits 2 after starting Go 1.26.0: the container cannot
read the mounted Makefile (`Permission denied`). A separate read-only root-mount
probe reproduces the denial; Docker checks are blocked, not passing. Recipe
fixtures pass. Logs/exits are in `cli/.coverage/review-fixes-01/`; fresh profiles,
provenance and package/function gaps remain in the unit/integration directories.
`git diff --check` passes. No commits or pushes performed. The initial P0 evidence
below remains historical; these review results supersede its verification totals.

## Outcome

Verification tooling implemented; overall CLI readiness remains incomplete.
All 30 Python tooling assertions, 15 architecture fixture cases, and four harness
tests pass. The complete eligible terminal campaign passes all six top-level
program/startup scenarios (including their subtests) and its real PTY scenario,
with fresh counters from both child runs. No required scenarios were skipped.
Unit race tests and vet pass. No production behavior/dependencies were changed.
Strict final coverage targets are not met. Baseline static/Flow failures are
visible; this is not a green `check` or a passing retained integration suite.

## Exact independent coverage

| Production package | Unit covered/total | Terminal covered/total | Uncovered unit / terminal |
| --- | ---: | ---: | ---: |
| `cli/cmd/mch` | 0/3 | 1/3 | 3 / 2 |
| `cli/internal/agent` | 249/385 | 173/385 | 136 / 212 |
| `cli/internal/app` | 1946/2542 | 1038/2542 | 596 / 1504 |
| `cli/internal/changes` | 481/601 | 390/601 | 120 / 211 |
| `cli/internal/dto` | 0/0 | 0/0 | 0 / 0 |
| `cli/internal/epics` | 4/5 | 4/5 | 1 / 1 |
| `cli/internal/help` | 6/6 | 6/6 | 0 / 0 |
| `cli/internal/navigation` | 31/40 | 1/40 | 9 / 39 |
| `cli/internal/projects` | 105/116 | 6/116 | 11 / 110 |
| `cli/internal/styles` | 0/0 | 0/0 | 0 / 0 |
| `cli/internal/testcases` | 5/6 | 5/6 | 1 / 1 |
| `cli/internal/ui` | 12/13 | 8/13 | 1 / 5 |
| `cli/pkg/client` | 221/304 | 165/304 | 83 / 139 |
| **Total** | **3060/4021 (76.1005%)** | **1797/4021 (44.6904%)** | **961 / 2224** |

Both runners completed valid measurements and exited 1 for their strict integer
threshold gates; Make exposes exit 2. The two zero-statement packages remain in
the inventory. All 13 production packages are audited; there is no production
package outside cmd/internal/pkg today. cmd/mch's three statements remain in
both denominators. No unit counters enter terminal coverage.

Largest gaps: app state/command/form branches (596 unit / 1504 terminal), agent
error/process handling (136 / 212), changes (120 / 211), client error/decoding
paths (83 / 139), project screens (11 / 110) and navigation (9 / 39). Individual
uncovered source blocks and functions are in each campaign's `uncovered.txt`
and `functions.log`. These are useful starting points for P1–P10, not a reason
to add artificial execution paths or remove defensive code.

## Commands and artifacts

Commands below actually ran. Fixtures test recipes separately and are not a
claim that setup, Docker, explicit whole-tree formatting, or HTML campaigns ran.

| Command | Exit | Result |
| --- | ---: | --- |
| `go test -short -count=1 ./cmd/... ./internal/... ./pkg/...` (cli, baseline) | 0 | Unit baseline passes, including explicit cmd inventory |
| `go test -count=1 ./integration/...` (cli, baseline) | 1 | Twelve existing Flow-related top-level failures; PTY passes |
| `make -C cli tooling-test` | 0 | 30 Python tests, boundary and harness fixtures pass |
| `make -C cli check` | 2 | format-check, lint and real architecture fail; vet, unit race and tooling pass |
| `make -C cli coverage` | 2 | Valid 3060/4021, >95% gate fails |
| `make -C cli deps-audit` | 0 | No vulnerabilities found |
| `make -C cli integration-coverage` | 2 | All eligible scenarios pass, valid 1797/4021, >90% gate fails |
| `make -C cli integration-test` | 2 | Retained baseline Flow failures plus one intermittent spec-write failure described below |
| `go test -count=3 -timeout=1m ./integration -run '^TestCLIProgramExistingSpecUsesUUIDArtifactWorkspaceAndSpecWrite$'` (cli) | 0 | Three focused reruns pass; original failure retained |
| `go test -count=1 ./integration/terminal` (cli, initial harness revision) | 0 | Standalone build works; final covered PTY execution is in the campaign |

Evidence is under ignored `cli/.coverage/verification/` (command exit ledger and
complete baseline/final logs), `cli/.coverage/unit/` and
`cli/.coverage/integration/`. Campaign `provenance.json` records HEAD, working diff,
source hashes, Go/build environment; `commands.jsonl` records build/test/tool exits.
`status.json` says measurement complete with failing gate exit 1. Profiles are
`coverage.out`; `complete.out` adds only structural zeros for diagnosis.
Terminal artifacts also include `scenarios.json`, `binary.sha256`, `mch`,
`program.out`, `program-child.out`, `pty-child.out`, private counter directories,
and test JSON logs. Future campaigns replace same-suite artifacts under a lock.
The checkpoint/log edits after measurement are documentation-only.

Tools actually used: Go `go1.26.8-X:nodwarf5 linux/amd64`, golangci-lint `2.13.1`
(built with go1.26.7-X:nodwarf5), govulncheck `v1.7.0`, installed socat (version in
campaign `socat-version.log`). Pinned tool setup was not repeated. The first lint
run could not write its default sandbox-external cache; final tooling uses ignored
`cli/.coverage/lint-cache` and no longer reports that environmental failure.
Docker compatibility was not run: this pass changes recipe selection, not the
module toolchain or dependencies; quoted paths/version/no-TTY behavior is tested
with isolated tools. No live backend/database or real-agent campaign was run.

## Visible baseline issues and next ownership

- **Formatting:** 37 existing files need gofumpt/goimports formatting. They span
  cmd, app, agent, feature/client source and older integration tests. P0 formats
  only the edited/new harness/checker files. Repair in their owning P1–P9 passes.
- **Lint:** 12 existing findings: unchecked close in startup test and HTTP client;
  ineffectual prefix assignment in agent runner; builtin `clear` parameter and
  unused result in app agent flow; three unused context parameters in reference
  tests; missing dto/help/testcases package comments; unused HTTP-test request.
  Repair with owning feature/adapter migration. No new lint findings were added.
- **Architecture:** `internal/agent/parser.go` imports `cli/internal/changes`.
  Previously invisible because the checker used `mch/internal/`; now fails openly.
  Resolve shared parsing/DTO ownership in P1/P8, without whitelisting the edge.
- **Legacy Flow:** old stage count, removed prompt paths, renamed temp UUID
  variables and unresolved prompt placeholders conflict with retained assertions.
  Missing root scripts include `change-slug.pl`, `change-def.pl`, `change-new.pl`.
  Twelve top-level baseline failures are listed below. P1 removes obsolete Flow
  assertions with the removed behavior and transfers useful process/prompt checks.
- **Intermittent unchanged runner failure:** one retained-suite spec-write run
  saved the user artifact, then reported `read |0: file already closed` before the
  agent save. The same unchanged test passed in the baseline, all measured
  campaigns, and three focused reruns. This is consistent with the existing
  concurrent stdout-reader/process-wait path; that cause is not yet confirmed.
  Preserve the failed log and repair process ownership/read completion in P1/P8.
  Do not label the failed retained run as passing because reruns passed.

- `TestDefaultFlowPromptAndStageIntegration`
- `TestPromptsReadStartupGeneratedChangeTypes`
- `TestWorkflowPromptsRequireExplicitDocumentationScope`
- `TestChangeFileInitPromptPreservesOrdinaryIdeaWording`
- `TestDefaultFlowSessionScriptIntegration`
- `TestDefaultFlowSessionScriptsRenderPromptPaths`
- `TestChangeSlugExtractionGrammar`
- `TestChangeSlugGrammarIsSharedByWorkflowEntryPoints`
- `TestChangeDefScriptCommitsAndPushesDefinition`
- `TestChangeNewScriptInitializesLocalBranchWithoutPublishing`
- `TestChangeNewScriptReusesExistingBranchesWithoutPublishing`
- `TestCLIProgramDefReviewUsesDefinitionPromptAndSharedArtifactSession`

Initial tooling verification caught and repaired helper issues: the isolated
Make fixture needed an actual Python fixture test (empty discovery exits 5), and
Go's no-test-files package events must not be treated as skipped unit test cases.
A CLI-local ignore exception also keeps the new coverage runner publishable and
in its own source provenance (the root ignore rule excludes `coverage.*`).
Final tooling assertions and both fresh campaigns verify those repairs.

Next: supervisor review against origin/dev and revalidate any review fixes.
Continue with P1 legacy Flow removal under the plan's resource constraints:
retain `.mch/config.yaml`, `.mch/tmp/`, and the five approved prompts; remove
obsolete Flow dispatch/YAML/automation without deleting user workspaces. Replace
useful assertions in the same pass, address the recorded runner/static baseline,
and keep both strict coverage gates and measurements throughout. No stage/prod.
