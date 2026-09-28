# Backend project/epic alignment checkpoint — P2 reviewed

2026-09-28; branch `change/007-backend-project-epic-alignment`, P1 base
`131e075`. Factory implementation `9a4cef3` and coherent custom-phase fixture fix
`df3fd34` are published. Native review against the pinned P1 base completed with
no actionable findings. The supervisor is publishing this checkpoint before the
authorized squash merge to dev. No transient retry was needed. No stage or
production promotion occurred. **Overall backend verification remains incomplete**
because tracked baseline debt and final-result coverage targets remain unmet;
the user's intermediate-pass policy explicitly permits this merge.

Next action after merge: create P3 `change/008-backend-change-doc-alignment`
from fresh origin/dev and align change/document behavior. For P3 creation use
default-config project 1002, or another creation-capable fixture: custom project
1001 deliberately supports discover/ship and analysis/decision, not backlog/brief.
Existing change 1202 is coherently seeded in discover. Preserve the six-array
selected-config assertions. The factory/review transcripts are
`/tmp/mch-p2-code-spec.log` and `/tmp/mch-p2-review-loop.log`.

## Implemented contract

Project details expose stored config and last_ref alongside view counts and
metadata. Epic details remove version, widen counters and derive completed in
the service. Creates return only {id}; updates/deletes return empty 204. All
mutations use one SQL statement without rereads, transactions or history calls.
Names and IDs are independently validated in services, with API format validation
through Echo/validate. Domain requests and contexts reach repositories intact.

Project config resolves exactly the stored slug and returns all six ordered
arrays; no fallback. Missing project or selected config returns the centralized
404 `project configuration not found`. Direct deletes let actual FKs enforce
409 conflicts and retain append-only documents. Options routes/module/DTOs and
startup wiring are removed; route inventory is 32 (33 + 1 - 2). No unused
project/epic document API was invented. The [contract ledger](backend-contracts.md)
records these choices and the complete inventory.

## Verification on final implementation

Commands below run from repository root unless a directory is specified. Make
failure exits are 2; valid below-threshold coverage recipes exit 1. Commands were
rerun after the final test fixes. Logs are `/tmp/mch-p2-*.log`; profiles, source
hashes, block inventories and package reports are `.coverage/unit/` and
`.coverage/api/`. Documentation/log writes after measurement change no production
statements. No failed/incomplete campaign supplies the reported coverage.

| Command actually run | Exit | Evidence |
| --- | ---: | --- |
| In backend: `golangci-lint fmt --no-config --enable gofumpt --enable goimports` with touched Go paths | 0 | Touched files only; diff inspected |
| In backend: `go test -count=1 ./internal/project ./internal/epic ./internal/error` | 0 | Boundary, service and exact HTTP contracts pass |
| `make -C backend tooling-test` | 0 | 45 Python tests plus Go suite-validator tests |
| `make -C backend check` | 2 | Baseline formatting in two untouched files |
| `GOLANGCI_LINT_CACHE=/tmp/mch-p2-lint-cache make -k -C backend check` | 2 | 17 baseline lint issues; vet, race and tooling pass |
| `make -C backend coverage` | 2 | Valid **853/1164 = 73.2818%**; strict >95% gate fails |
| `make -C backend deps-audit` | 0 | No vulnerabilities found |
| `make -C backend api-test` | 2 | All 100 requests and owned server pass; valid **414/1164 = 35.5670%**; >=90% gate fails |
| In backend: `python3 -B /tmp/mch-p2-legacy-check.py` | 0 | Owned lifecycle runs `go test -count=1 ./api-tests/project ./api-tests/epic`; timestamp and document-retention contracts pass |
| `make -C backend api-test` after fixture correction | 2 | All 100 requests pass again; unchanged 414/1164; log `/tmp/mch-p2-fixture-api.log` |
| `scripts/codex-review-loop.pl agent/specs/007-backend-project-epic-alignment.md --base origin/dev` | 0 | Native review pass 1 clean; no review fixes |
| `git diff --check` | 0 | No whitespace errors |

Baseline formatting remains in internal/testcase/schema_test.go and
api-tests/change/change_test.go (the third P1 file belonged to removed options).
Lint debt is 6 errcheck issues in shared HTTP client/body closes and testcase
rollbacks, 8 revive comments, and 3 unused change helpers. Two initial P2 test
lint issues were repaired; the final run introduces none. All checks that can
run beyond those baseline failures pass; no failing test was skipped or weakened.

The focused legacy helper reuses scripts/api_coverage.py's isolated ownership
and cleanup, selecting only the two migrated HTTP packages; its artifacts live
in `.coverage/legacy-p2/`. No legacy or unit counters enter APIHydra coverage.
The entire legacy change/testcase campaign, Docker and benchmarks were not run;
no compatibility or performance contract changed in this pass.

## Package statement gaps

| Production package | Unit covered/total | APIHydra covered/total |
| --- | ---: | ---: |
| cmd/server | 70/88 | 73/88 |
| internal/change | 332/555 | 33/555 |
| internal/domain | 0/0 (no executable statements) | 0/0 |
| internal/epic | 115/115 | 107/115 |
| internal/error | 36/36 | 28/36 |
| internal/health | 21/21 | 16/21 |
| internal/project | 117/117 | 110/117 |
| internal/testcase | 118/188 | 22/188 |
| pkg/config | 30/30 | 21/30 |
| pkg/db | 4/4 | 0/4 (unlinked structural zero) |
| pkg/markdown | 10/10 | 4/10 |
| **Aggregate** | **853/1164** | **414/1164** |

Project and epic unit coverage is 100%. Unit gaps remain in startup/main/error
paths and deferred change/testcase repositories. API gaps include database query,
scan and iteration failures, concurrent parent-removal FK classification,
startup failures and connector/Markdown error branches, plus most deferred P3/P4
business behavior. These have no fabricated HTTP triggers or coverage credit.
Both final coverage targets remain unmet. Next passes should migrate and exercise
change/document and testcase workflows, then assess legitimate residual error
paths and the final thresholds with the user.

## Acceptance-to-test mapping

Named module tests exist in both project and epic unless qualified otherwise.

| Criterion / individual contracts | Meaningful evidence |
| --- | --- |
| P2-01 all project fields, stored config/last_ref, count source, ordered/empty list | `TestRepositoryReads` asserts full scan results, large counts, current view/base join, SQL ordering and closure; `TestAPIRegisteredContracts` pins JSON keys; `TestAPIEmptyList` proves []; APIHydra initial tied-timestamp list and project 1001 details |
| P2-02 epic fields, no version, widened counts, service-derived zero/nonzero completion, scoped absent/empty lists and ordering | `TestRepositoryReads`, exact `TestAPIRegisteredContracts`, `TestServiceDerivesCompletion` (0, 50, truncated 66 and large 70%), `TestAPIEmptyList`; APIHydra 1102/1103 details/list and absent project list |
| P2-03 ID-only 201, truly empty 204, error-only mutation methods, no reread/history/transaction, same-name modified | `TestAPIRegisteredContracts`, `TestServiceRequestsAndErrors`, `TestRepositorySingleStatementMutations` enforce one call and mutation SQL; APIHydra independent post-write reads; legacy `TestProjectCRUD` / `TestEpicCRUDAndProjectScopedList` prove same-name timestamp advancement |
| P2-04 independent positive IDs/trimmed nonblank names, same requests/contexts, central bind/validation/DB/missing/conflict errors | `TestServiceRejectsInvalidProjectInput`, `TestServiceRejectsInvalidEpicInput`, `TestServiceRequestsAndErrors`, `TestAPIRejectsMalformedAndInvalidRequests`, handler error-contract tests, repository tests preserving pgx/FK causes; retained `TestBackendErrorOwnership` audit; APIHydra malformed/types/invalid/missing cases |
| P2-05 selected slug/all arrays/order, no hardcoding/fallback, missing selected slug while default exists, stored selection, shared unavailable 404, default create/name-only update | `TestRepositorySelectedConfig`, `TestServiceConfigNeverSubstitutes`, exact config API/error tests, `TestHTTPContractsAndCauses`, one-statement mutation tests; APIHydra distinct six-array custom/default fixtures and missing selected/project rows |
| P2-06 direct deletes, project epic/change conflicts, epic change conflicts, missing/success, safe conflict message, retained docs | `TestRepositorySingleStatementMutations` checks one direct DELETE, zero/one rows and preserved FK cause; handler/central error tests; APIHydra fixtures 1004/1005/1006/1101/1102 and independent conflict-survivor reads; legacy SQL `TestDeletionRetainsAppendOnlyDocuments` proves historical/current rows survive both parent deletes |
| P2-07 options 404/removal, config ownership, buildable startup, 33+1-2=32 inventory | `TestOptionsRoutesUnregistered`, retained startup lifecycle tests, `ContractsTest.test_all_registered_routes_remain_in_ledger_denominator`; APIHydra former options paths and project/config; build/vet and reference audit confirm no stale options DTOs/wiring |
| P2-08 every operation, independent successful reads, negatives, genuine coverage and owned fixtures | All 11 P2 operations in 91 serial P2 requests; full 100-request campaign; runner `test_success_stops_before_conversion_and_uses_private_cluster` checks init/seed/fixture order and ON_ERROR_STOP, `test_fixture_failure_stops_before_server_and_suite` verifies failure propagation; existing strict coverage/tooling tests and actual command results above |

The [APIHydra report](../apih-tests/coverage.md) documents exact-key/204-body
assertion limits, operation diagnostics, installed tool revision and reserved
fixture IDs. Numerical shortfalls do not stop this authorized intermediate
merge sequence; they remain honest failing gates and are not final readiness.
