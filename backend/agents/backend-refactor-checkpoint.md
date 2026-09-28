# Backend testcase alignment checkpoint — P4 reviewed

2026-09-28; `change/009-backend-testcase-alignment` over freshly fetched merged
P3 `b626dd528362c355ebc33d5c8134069a88c23377` (branch specification commit
`8f057b9`). Factory implementation `5583ded` is published. Native review against
the pinned P3 base completed with no actionable findings or fixes. The supervisor
is publishing this checkpoint before the authorized squash merge to dev.
Both numerical coverage gates pass. No transient retry was needed.
**Overall verification remains incomplete because baseline lint still fails.**
No R1 work or stage/production promotion occurred.

Next: supervisor merge-to-dev, then R1 deadcode audit under the authorized sequence.
Factory/review transcripts: `/tmp/mch-p4-code-spec.log` and
`/tmp/mch-p4-review-loop.log`. The
[ledger](backend-contracts.md) records current contracts, precise non-atomic
read limitations and every removed obsolete assertion's replacement evidence.

## Implemented contract

All five testcase routes remain. List returns current six-column rows ordered
by ID after a live-parent check. Create performs one FK-backed INSERT and returns
only {id}/201. Scenario/done updates and delete each perform one statement and
return empty204, with zero affected rows404. Direct services validate positive
IDs and trim scenarios; omitted/null done remains false. Errors preserve causes
through central APIs. No transaction, procedure, history/version, parent count
write, mutation reread or renderer coupling remains. Legacy domain.Change and
RenderChange/RenderMutation are gone; explicit document rendering stays intact.

The new serial APIHydra flow creates its own project/epic/change and captures
all IDs. Independent reads prove 0/0/0 → 0/2/0 → 1/2/50 → 2/2/100 → 1/2/50 →
0/1/0 → 0/0/0 counts, text edits, defaults, order and both bool values. FK409
preserves the parent/children/documents; explicit child removal allows parent
deletion, followed by read/write404. Negative cases include all applicable
invalid/blank/malformed/wrong-type/large-missing/missing/repeated-delete paths.
P2/P3 fixtures are unchanged, including all 50,000 wide-count testcase rows.

The complete retained HTTP campaign passes, including seven project/epic/change
tests, health and two migrated testcase tests. It additionally proves same-value
timestamp advancement, stable created/unrelated fields and unchanged parent
modified timestamps. Retained SQL proves three append-only document rows with
two historic/one current, exact IDs/bodies/flags after deletion, and actual parent
and child absence. Its counters never enter APIHydra coverage.

## Verification and failures

Logs: `/tmp/mch-p4-*.log`. Required commands were run on the final source;
coverage is based on real statement counts, not package percentage averages.

| Command actually run | Exit and evidence |
| --- | --- |
| Backend: `golangci-lint fmt --no-config --enable gofumpt --enable goimports` on touched testcase/domain/change/server and testcase/change HTTP packages | 0; diff inspected |
| Backend: `go test -short ./internal/testcase ./internal/change ./cmd/server` | 0; targeted tests; final testcase additions also pass |
| Backend: `go test -run '^$' ./api-tests/...` | 0; migrated harness compiles |
| Backend: `GOLANGCI_LINT_CACHE=/tmp/mch-p4-lint-cache golangci-lint run --no-config --default standard --enable revive --timeout 5m ./internal/testcase/... ./internal/change/... ./internal/domain/... ./cmd/server/...` | 0; touched production packages clean |
| `GOLANGCI_LINT_CACHE=/tmp/mch-p4-lint-cache make -C backend check` | 2; format passes, baseline lint fails |
| `GOLANGCI_LINT_CACHE=/tmp/mch-p4-lint-cache make -k -C backend check` | 2; format/vet/race/tooling pass, baseline lint fails; 45 Python tooling tests and Go validator pass |
| `make -C backend coverage` | 0; 914/932 = 98.0687%, strict >95% passes |
| `make -C backend deps-audit` | 0; no vulnerabilities found |
| `make -C backend api-test` | 0; 408 requests and owned server pass; 843/932 = 90.4506%, >=90% passes |
| `make -C backend legacy-api-test` | 0; all retained HTTP/SQL tests pass against its owned disposable cluster |
| `git diff --check` | 0 |

First APIHydra attempt failed (make2, runner101): the new P4 file omitted its
JSON Content-Type default. Added that header; all assertions were retained and
the complete campaign rerun successfully. Failed-run diagnostics are preserved
in `/tmp/mch-p4-api-runner-first.log`; that run establishes no valid coverage.
A focused lint run exposed a missing testcase package comment; repaired before
final verification. No application scenarios remain failed/skipped/blocked.
Docker/benchmarks and the later owned-DB outage campaign are outside this pass.

Known untouched baseline lint: three unchecked shared HTTP Body.Close calls,
shared package/five exported comments and other existing package comments
(11 reported issues: 3 errcheck, 8 revive). Final `check` reports epic/markdown;
final `make -k check` reports health/markdown; an earlier run reported db/markdown.
Revive report limiting varies the selected existing package-comment diagnostics. No failure is suppressed or gate
weakened. Refactor policy permits review/merge with documented baseline debt.

## Fresh statement counts and gaps

| Production package | Unit covered/total | APIHydra covered/total |
| --- | --- | --- |
| cmd/server | 70/88 | 73/88 |
| internal/change | 396/396 | 376/396 |
| internal/domain | 0/0 | 0/0 |
| internal/epic | 115/115 | 107/115 |
| internal/error | 43/43 | 33/43 |
| internal/health | 21/21 | 16/21 |
| internal/project | 117/117 | 110/117 |
| internal/testcase | 108/108 | 99/108 |
| pkg/config | 30/30 | 21/30 |
| pkg/db | 4/4 | 0/4 |
| pkg/markdown | 10/10 | 8/10 |
| **Aggregate** | **914/932** | **843/932** |

P3 denominator1021 → P4 denominator932 reflects removal of obsolete testcase
transactions/reloads/rendering and the legacy renderer, plus API validation and
current SQL. No packages, defensive paths or source statements were excluded.
Unit gaps are the existing 18 server main/start/error-handler statements. API
gaps (89 statements) include query/scan/iteration/Exec failures, UUID entropy,
concurrent FK/parent races, startup/config, degraded health and Markdown failure
branches; pkg/db is unlinked structural zero. Testcase's nine API gaps are DB
failure arms. No fault endpoints, outage campaign, artificial tests or merged
profiles were used. Domain has zero executable statements.

Fresh ignored artifacts are `backend/.coverage/unit/`, `.coverage/api/` and
`.coverage/legacy/`; each retains separate provenance/logs/profiles. The
[API report](../apih-tests/coverage.md) records 34/34 successful operations,
installed manual revision, tool limitations and rerun command.

## Acceptance evidence

| Criterion and individual behavior | Named unit and real integration evidence |
| --- | --- |
| P4-01 six fields, domain list request, parent/empty/order/closure/query-scan-iterator causes, read race | `TestRepositoryCurrentSixColumnList`, `TestCurrentTestcaseArchitecture`, `TestTestCaseListExactCurrentFields`, `TestServiceNormalizesAndDelegatesOnce`; P4 ordered/empty/missing independent lists; non-atomic race documented, empty-after-positive test |
| P4-02 trim/positive create, one INSERT/defaults/ID-only, FK404 and unknown cause | `TestRepositoryCreateOnlyID`, `TestRepositoryTranslationKeepsExternalCauses`, `TestServiceRejectsInvalidTestCaseInput`, `TestServiceNormalizesAndDelegatesOnce`, `TestTestCaseAPIExactContracts`, `TestTestCaseHandlerErrorContracts`; P4 creates/default false/trimmed rows and missing-parent404 |
| P4-03 one statement/error-only/affected rows/204/same-value/no parent or unrelated writes | `TestRepositorySingleStatementMutations`, service and API exact tests; P4 lifecycle; `TestTestCaseCurrentStateAndSameValueTimestamps` checks actual timestamps, created/unrelated/parent invariance |
| P4-04 central bind/400/404/500 and causes, direct validation, zero/negative/large IDs, done false/true/omitted/null, route removal | `TestTestCaseHandlerErrorContracts`, `TestTestCaseAPIInvalidPayloads`, `TestTestCaseAPIExactContracts`, service tests, `TestRepositoryTranslationKeepsExternalCauses`, `TestTestCaseMoveRouteRemoved`; P4 negative cases and retained HTTP invalid/missing test |
| P4-05 remove legacy types/helpers/transactions/renderer, retain explicit sanitization | `TestCurrentTestcaseArchitecture`, `TestChangeArchitectureAndP4Boundary` absence checks; `TestRendererExplicitSource` and retained P3 `TestServiceSanitizesExplicitReads`; P3 current-document/artifact API suite unchanged |
| P4-06 independent workflow, ordered state, full count progression/delete-last, repeated same-value and negative operations | All repository/service/API tests above; retained P2/P3 completion unit tests; 96-request dependent P4 suite and migrated timestamp/lifecycle HTTP test |
| P4-07 FK409/readable survivor, explicit child removal/delete204, removed-parent404, historic+current docs retained | Retained P3 `TestRepositorySingleStatementMutations`, central `TestChangeDatabaseContracts`, testcase single-statement tests; P4 FK/document workflow; adapted `TestChangeDeletionRetainsAppendOnlyDocuments` actually runs SQL retention and parent/child absence assertions |
| P4-08 meaningful SQL/call/scan/cause tests, legacy migration, full route denominator, fresh validation/handoff | Tests above plus `ContractsTest.test_all_registered_routes_remain_in_ledger_denominator`; all 408 API requests and complete owned legacy campaign pass; ledger records obsolete assertion reasons and preserved guarantees |
