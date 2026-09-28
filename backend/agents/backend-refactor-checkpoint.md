# Backend refactor checkpoint — 015 compact handlers (R6 reviewed)

2026-09-28; `change/015-backend-compact-handlers` over freshly fetched R5
`bbde3fee7476bb05267fd778611228b47d27bb67`. Factory implementation `7446652`
and fresh contract/coverage report `ae1318f` are published. Native review against
the pinned R5 base passed on its first pass without findings or fixes; the
reviewer repeated backend unit and whitespace checks. All five validation targets
and both strict coverage gates pass. The supervisor is publishing this checkpoint
before the authorized squash merge to dev. No stage/production promotion occurred.

The initial branch/spec creation was rejected before process creation because
automatic approval review timed out. The supervisor verified clean dev, no015
branch/spec/factory log, then one permitted retry succeeded. No command was
repeated after a partial Git mutation and no duplicate factory process started.
Transcripts: `/tmp/mch-r6-code-spec.log` and `/tmp/mch-r6-review-loop.log`.

Next after merge: separate016 final verification, owned-DB-outage campaign and
combined architecture review. No additional R6 compression is included.

## Bounded audit and acceptance evidence

| Criterion | Scope and evidence |
| --- | --- |
| R6-01 | Fresh source inspection found exactly14 separate single-error service assignments, each read only by the immediately following conditional/error return. Actual handlers: project.updateProject; epic.updateEpic; change.updateEpic, updateChangeTypes, updateTitle, updateBrief, updateSpec, updatePR, updatePRUrl, updatePhase, updateOpen; testcase.updateTestCase, updateTestCaseDone, deleteTestCase. Base SHA is above. Adjacent delete/set-document handlers already use this idiom. |
| R6-02 | Exact call, context/request expressions, validation/binding, centralized error return and empty204 success return remain unchanged. Existing meaningful tests are mapped below; missing API-boundary request/context/call/cause assertions were added to existing matrices. No new public/shared type or contract. |
| R6-03 | Manually inspected all14 production hunks: only two-line declaration/conditional pairs became scoped if initializers. Four api.go files, +14/-28 production lines. Result-returning handlers, routes, lower layers, SQL and configuration untouched. No helper or formatting sweep. |
| R6-04 | Affected baseline/final tests, normal check, fresh independent unit/APIHydra measurement, vulnerability audit and complete owned legacy campaign all pass. Same production inventory, strict thresholds and retained scenarios; no tooling, runner, fixture or ownership edits. |
| R6-05 | This checkpoint, only the official plan's top status, and backend/implementation-log.md record the handoff. The log location follows spec015's explicit backend-only scope. |

### Behavioral test mapping

- Project/epic: TestAPIRegisteredContracts proves exact one repository call,
  context identity and truly empty204; its update case now also checks the
  domain request. TestProjectHandlerErrorContracts/TestEpicHandlerErrorContracts
  retain direct/wrapped error identity and exact status/body messages.
- Change: TestChangeAPIContracts retains all original success, malformed,
  wrong-type, invalid, missing and failure scenarios. It now checks returned
  cause identity and, for the nine edited mutations, exact successful repository
  operation sequences, final domain requests and context identity. Failed calls
  retain the original first-operation failure injection and now assert no extra
  call and context identity. Existing service tests cover collaborator failure
  at later steps and specialized document mapping. Empty204 assertions remain.
- Testcase: TestTestCaseAPIExactContracts retains exact requests, one call,
  wide IDs, explicit/omitted/null booleans and empty204 bodies; it now checks
  request-context identity. TestTestCaseHandlerErrorContracts retains centralized
  error/cause and exact response assertions for all three edited mutations.
- TestAPIValidationCauses in all four modules retains actual validator field/rule
  causes, bind wrapping, public400 bodies and no collaborator calls on rejection.
  TestUpdatePRURLValidationCauses retains URL parser cause identity. The server's
  TestAPIConstructorRouteInventory and tooling ledger checks retain all34 routes.
- Reference/diff inspection establishes R6-01/R6-03 scope; no syntax/AST or
  prose-only tests were added. All assertions are behavioral and use existing
  repository fakes and real services; no test seam was added to production.

## Commands and results

Commands run from root unless marked backend. All final validation exits are0.
Logs: `/tmp/mch-r6-{baseline,targeted,check,coverage,deps-audit,api-test,legacy-api-test}.log`.

| Command actually run | Exit/result |
| --- | --- |
| `git ls-remote origin refs/heads/dev` | 128: host SSH configuration ownership error; no remote operation executed |
| `GIT_SSH_COMMAND='ssh -F /dev/null' git ls-remote origin refs/heads/dev` | 0; remote dev equals the R5 base above, using a process-local config override |
| Backend: `go test -short -count=1 ./internal/project ./internal/epic ./internal/change ./internal/testcase` | 0 baseline and final; all four affected modules pass |
| Backend: `golangci-lint fmt --no-config --enable gofumpt --enable goimports internal/project/api.go internal/epic/api.go internal/change/api.go internal/testcase/api.go internal/project/api_test.go internal/epic/api_test.go internal/change/api_test.go internal/testcase/api_test.go` | 0; touched-file diff inspected; final change/api_test.go adjustment formatted again with the same flags |
| `GOLANGCI_LINT_CACHE=/tmp/mch-r6-lint-cache make -C backend check` | 0: formatting, zero lint findings, vet, race tests, all45 Python tooling tests and Go suite-validator tests |
| `make -C backend coverage` | 0; 911/925 statements, >95% gate passes |
| `make -C backend deps-audit` | 0; no vulnerabilities |
| `make -C backend api-test` | 0; all five unchanged APIHydra suites pass, 837/925 statements, >=90% gate passes |
| `make -C backend legacy-api-test` | 0; change/epic/health/project/shared/testcase HTTP/SQL packages pass |
| `scripts/codex-review-loop.pl agent/specs/015-backend-compact-handlers.md --base origin/dev` | 0; native pass1 clean, no fixes |
| `git diff --check` | 0; tracked/untracked scope inspected |

No failing, skipped or blocked required scenarios remain. APIHydra retains
all408 requests/34 operations. Both integration runners used their own disposable
PostgreSQL cluster and server, then stopped them. Legacy counters never enter
APIHydra coverage. APIHydra version remains c3947513e2a4. No benchmark or Docker
check is required for this local change without performance/toolchain effects.

## Fresh production statement counts and gaps

| Production package | Unit covered/total | APIHydra covered/total |
| --- | --- | --- |
| cmd/server | 77/91 | 75/91 |
| internal/change | 389/389 | 369/389 |
| internal/domain | 0/0 | 0/0 |
| internal/epic | 114/114 | 106/114 |
| internal/error | 48/48 | 36/48 |
| internal/health | 21/21 | 16/21 |
| internal/project | 116/116 | 109/116 |
| internal/testcase | 105/105 | 96/105 |
| pkg/config | 31/31 | 22/31 |
| pkg/markdown | 10/10 | 8/10 |
| **Aggregate** | **911/925 (98.4865%)** | **837/925 (90.4865%)** |

Normal Go instrumentation counts one fewer statement for each scoped initializer:
14 fewer total/covered statements than R5 in both suites. No production package,
behavior or defensive branch was removed to affect the denominator. Unit gaps
remain14 main.go process-orchestration statements at lines15–33. Domain has no
executable statements; every other production package has full unit coverage.

APIHydra retains88 uncovered statements: server16, change20, epic8, error12,
health5, project7, testcase9, config9, markdown2. These cover startup/shutdown and
HTTP error-write failures, configuration load/decode/default/override paths,
SQL query/scan/iteration/Exec failures, entropy/concurrent parent failures,
degraded health, Markdown conversion errors and central nil helpers. Final016
will reassess these retained failures; neither R6 aggregate target is unmet.

Fresh reports, structural denominators, raw counters and provenance remain in
`backend/.coverage/unit` and `backend/.coverage/api`; legacy evidence remains in
`backend/.coverage/legacy`. All three provenance records' Go source/test hashes
match the final implementation. Documentation is updated after measurement.
Exact uncovered blocks are also saved under `/tmp/mch-r6-{unit,api}-gaps.txt`;
unit repeated profile blocks were combined before classifying uncovered code.
