# Backend error contracts checkpoint — P1 reviewed

2026-09-28; branch `change/006-backend-error-contracts`; base `origin/dev`
`d3c2235`. Bounded P1 implementation passed native review at `8832dc5`. **Overall
backend verification remains incomplete:** baseline check failures and both
coverage shortfalls remain visible. The user's intermediate-pass policy permits
continuing the sequence with these measurements; final coverage has not passed.
The factory published implementation `6a75bde` and review fix `8832dc5`.
Review pass 2 found no actionable regressions; required checks were rerun after
the test-only review fix. The supervisor is publishing this checkpoint before
the authorized squash merge to dev. No dependency/SQL change, P2 implementation
or promotion occurred. Implementation writes stayed in backend; the specification
is the authorized exception. Factory transcripts: `/tmp/mch-p1-code-spec.log`
and `/tmp/mch-p1-review-loop.log`. No transient retry was needed.

## Implemented contract

- `internal/error` owns distinct module sentinels, cause-preserving diagnostic
  wrapping, explicit no-row/foreign-key mappings, invalid payload creation,
  HTTP interpretation/translation and normal shutdown classification. Unknown
  database errors pass through without losing their identity. Nil/no-context
  wrapping and repeated database/HTTP interpretation do not add wrappers.
- Repositories retain their SQL, parameters, affected-row and conflict behavior.
  Missing rows and testcase foreign-key errors retain both business semantics
  and original pgx causes. Private project/epic pool interfaces enable boundary
  tests; public constructors remain unchanged. Existing transactions remain for
  P2–P4; none was added.
- APIs retain operation-specific bind messages and the JSON `message` envelope;
  central HTTP translation keeps wrapped causes and masks unexpected internals.
  The installed server handler also interprets wrapped and router errors.
  Response writing stays at the HTTP boundary. Health retains its separate
  healthy/degraded JSON and 200/503 behavior.
- Startup, config/db panic paths, health and Markdown use the central package.
  Startup/shutdown cleanup and config precedence remain. Markdown still returns
  an empty string on conversion failure and now logs the contextual cause.
  Services retain independent validation and do not import Echo, pgx or validate.
- The real APIHydra campaign asserts 400/404/409/500 plus both health aliases.
  Current project/epic inserts precede their broken reloads: explicit 500 checks
  establish disposable fixtures for the real project-delete 409. These requests
  count as error-path reach, never successful create operations.

## Final verification

Commands were run from the repository root on the final production code.
Make failure exits are 2; coverage gate recipes return 1.

| Command actually run | Exit | Evidence |
| --- | ---: | --- |
| `make -C backend test` | 0 | All uncached short production-package unit tests pass |
| `make -C backend check` | 2 | Stops on baseline formatting in three untouched files |
| `GOLANGCI_LINT_CACHE=/tmp/mch-p1-lint-cache make -k -C backend check` | 2 | 17 baseline lint issues; vet, race, 44 Python tooling tests and Go validator tests pass |
| `make -C backend coverage` | 2 | Valid **786/1209 = 65.0124%**, strict >95% gate fails |
| `make -C backend deps-audit` | 0 | No vulnerabilities found |
| `make -C backend api-test` | 2 | APIHydra and owned server exit 0; valid **298/1209 = 24.6485%**, >=90% gate fails |
| `git diff --check` | 0 | No whitespace errors |

Modified Go files were formatted with pinned gofumpt/goimports. Baseline format
failures remain in `internal/options/service.go`, `internal/testcase/schema_test.go`
and `api-tests/change/change_test.go`. Baseline lint reports 6 errcheck issues
(legacy HTTP body closes and existing transaction rollbacks), 8 revive issues
(legacy client comments and package comments), and 3 unused change helpers.
No new lint findings remain. An earlier expanded check also reported sandbox
cache-write warnings; the final command uses a writable temporary lint cache.

The first API attempt failed with APIHydra exit 101: curl retries concatenated
500 response bodies and replayed inserts. It established no valid coverage.
The complete suite now uses `retries: -1` for its error file, verified against
the installed APIHydra revision: negative values override inherited defaults
and omit curl's retry argument. Exact expected status/body assertions remain.
The final campaign passes all 13 requests and owns/cleans its disposable cluster
and server. No unit or legacy counters enter the API result.

Current ignored logs: `.coverage/verification/p1-*.log`; fresh profiles, block
inventories, source hashes and package reports: `.coverage/unit/` and
`.coverage/api/`. Documentation/log updates after measurement do not change
production statements. No Docker check, benchmark, legacy HTTP campaign or
full business workflow validation is claimed.

## Review fix 01 — Echo ownership audit

The P2 review finding was valid: the AST audit resolved the unaliased Echo v5
import as `v5`, allowing `echo.NewHTTPError` and `echo.StatusCode` through.
The audit now maps the supported import to its declared name, `echo`, while
honoring explicit aliases. Source fixtures run through the same audit helper:
`TestErrorOwnershipEchoCalls` rejects both calls with default and aliased imports;
`TestErrorOwnershipAllowsEchoSetupAndCentralErrors` permits ordinary Echo setup
and central error handling. Both default-import negative fixtures failed before
the fix and pass afterward. No production behavior or public contract changed.

Final review-fix commands and exits:

- From `backend/`, `golangci-lint fmt --no-config --enable gofumpt --enable goimports --diff internal/error/ownership_test.go` and `go test -short -count=1 ./internal/error`: 0.
- `make -C backend check`: 2, the same three baseline formatting failures.
- `GOLANGCI_LINT_CACHE=/tmp/mch-p1-review-01-lint-cache make -k -C backend check`: 2, the same 17 baseline lint issues; vet, race, 44 Python tooling tests and Go validator tests pass.
- `make -C backend coverage`: 2; fresh valid 786/1209 (65.0124%), below >95%.
- `make -C backend deps-audit`: 0, no vulnerabilities found.
- `make -C backend api-test`: 2; APIHydra and owned server pass, fresh valid 298/1209 (24.6485%), below >=90%.
- `git diff --check`: 0.

Logs are under `.coverage/verification/p1-review-01-*.log`. Fresh profiles are
under `.coverage/unit/` and `.coverage/api/`; the package counts below and
deferred scenarios remain unchanged. Overall verification remains incomplete
because of the recorded baseline debt and coverage shortfalls. The finding is
fixed; the caller retains ownership of commits and pushes.

## Package statement gaps

| Production package | Unit covered/total | APIHydra covered/total |
| --- | ---: | ---: |
| cmd/server | 73/91 | 77/91 |
| internal/change | 332/555 | 33/555 |
| internal/domain | 0/0 (no executable statements) | 0/0 |
| internal/epic | 75/137 | 33/137 |
| internal/error | 36/36 | 28/36 |
| internal/health | 21/21 | 16/21 |
| internal/options | 6/38 | 6/38 |
| internal/project | 81/99 | 58/99 |
| internal/testcase | 118/188 | 22/188 |
| pkg/config | 30/30 | 21/30 |
| pkg/db | 4/4 | 0/4 (unlinked structural zero) |
| pkg/markdown | 10/10 | 4/10 |
| **Aggregate** | **786/1209** | **298/1209** |

Unit gaps remain in startup/main/write-error paths, repository success and
transaction branches, options and deferred change/testcase behavior. Every
executable package has integration gaps; failure-only config/connector/Markdown
branches and broken legacy SQL cannot be credited as covered. Successful
operation coverage remains **2/33** (health); **9/33** registered pairs were
reached, seven only for error contracts. An unknown router path is additional
HTTP-error evidence, excluded from the registered-route denominator.

## Acceptance-to-test mapping

| Criterion | Named unit tests and integration evidence |
| --- | --- |
| P1-01 definitions, nil, idempotence, direct/wrapped/unknown and nested causes | `TestWrapAndShutdownCauses`, `TestDatabaseMappingsAndCauses`, `TestHTTPContractsAndCauses`, `TestInvalidPayload`; all exported error functions covered, 36/36 statements |
| P1-02 database no-row/FK/query/scan/iteration/affected-row/conflict contracts | `TestProjectRepositoryMissingAndConflictContracts`, `TestEpicRepositoryMissingCauses`, `TestChangeMissingRowKeepsCause`, `TestChangeHelperMissingCauses`, `TestRepositoryTranslationKeepsExternalCauses`, `TestMutationChangeMissingCause`, `TestCreateTestCaseErrors`, `TestDeleteTestCasesForChange`, `TestTypeMutationErrorOnly`, retained schema/parameter tests |
| P1-03 exact HTTP status/body/envelope and original cause, handler-return and installed handler | `TestProjectHandlerErrorContracts`, `TestEpicHandlerErrorContracts`, `TestTestCaseHandlerErrorContracts`, `TestChangeHandlerReturnCauses`, `TestChangeAPIContracts`, `TestInstalledJSONErrorContracts`, `TestHTTPContractsAndCauses` |
| P1-04 startup/connector/config/Markdown/health ownership and service validation | `TestLifecycle`, `TestStartFailures`, `TestStartCancelRealServer`, `TestHTTPServerPreservesReadTimeout`, `TestConfigurationPanicCausesAndPrecedence`, `TestPoolCreationAndPanicCause`, `TestPingPreservesCancellation`, `TestMarkdownFailureKeepsEmptyOutputAndLogsCause`, `TestHealthAliasExactContracts`, module `TestServiceRejectsInvalid*Input` tests, `TestBackendErrorOwnership` AST audit |
| P1-05 genuine error integrations and full route denominator | `error-steps.yaml` and `health-steps.yaml`; exact unit proofs above supplement APIHydra's partial-object matcher; `ContractsTest.test_all_registered_routes_remain_in_ledger_denominator`, `test_health_suite_has_explicit_contracts_for_both_aliases` |
| P1-06 verification and handoff | Command exits/counts above; retained coverage/tooling regression suite; updated ledger, integration report and backend implementation log |

## Deferred scenarios and next action

Epic-delete 409 cannot reach its child guard because `getEpic` projects obsolete
`version`/`completed` first. Project-get 404 is blocked by absent `vw_project.last_ref`.
Testcase insertion's FK-to-404 mapping requires the removed `fn_test_case_insert`;
change phase-reference checks require the removed `change_phase` table. These
mappings have unit proof, not passing real integration claims. Health degraded
503 has exact unit proof; no artificial outage endpoint was introduced.

Next: finish the authorized P1 merge, then P2 aligns project/epic SQL and mutation
responses and replaces options with project-selected configuration. P2 must
replace the temporary schema-failure 500 expectations in `error-steps.yaml`
with successful current-schema operations; preserve unknown-error masking unit
proof and retain real error scenarios supported by the repaired backend.
P3/P4 own remaining change/testcase migrations.
Retain final >95% unit / >=90% API goals and honest failing gates throughout.
