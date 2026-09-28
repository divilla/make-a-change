# Backend dead-code checkpoint — R1 reviewed; R2 no action

2026-09-28; `change/010-backend-dead-code` over freshly fetched P4
`b9f630f0466d93381cf8e8935563672c92c0f6b1`. Specification `12ae35a` and factory
implementation `146346b` are published. Native review against the pinned P4
base passed on its first pass without findings or fixes. The supervisor is
publishing this checkpoint before the authorized squash merge to dev.
Both coverage gates pass; baseline lint remains the only overall-check failure.
No transient retry was needed, and no stage/production promotion occurred.
The implementation child's sandbox-only remote-query failure did not block
supervisor freshness/publication checks, which succeeded with authorized access.

R2 reassessment found no actionable duplicate paths after P3/P4/R1, so no empty
R2 implementation branch is created. Next is the separately scoped
`011-backend-validation-causes` error-contract repair, then `012-backend-startup-boundaries`
(R3), R4 config, R5 lint/local patterns and R6 compact-code reassessment.
Factory/review logs: `/tmp/mch-r1-code-spec.log` and
`/tmp/mch-r1-review-loop.log`. No production change followed final measurements.

## Removal evidence and scope

The [ledger](backend-contracts.md#r1-removal-audit-and-retained-contracts) records
each removal, current callers and retained candidates. Before editing,
production/test symbol searches and Go package/test/server dependency inventories
confirmed five unread private API.e fields and an unlinked pkg/db.Pool wrapper
with only its own test calls. No ignored files/build tags, generators, linkname,
registration, production reflection or documented supported backend consumer
uses those candidates. Removed only those fields/assignments, the obsolete
config debug comment, and db.go/db_test.go. API.g and every route stay intact.

Already resolved by P3/P4: options, legacy domain.Change, mutation renderer
adapters, and obsolete testcase transaction/history/reload paths. No R1 credit
is claimed for those migrations. Retained: parser/sanitizer and renderer,
repository pool and lifecycle seams, defensive errors and independent SQL.
No dependency/schema/transaction/shared API/configuration behavior changed.
Only backend paths and the official plan's top status are changed.

Two meaningful parity gaps are addressed: an exact runtime 34-route inventory
and pgconn.ParseConfigError/unwrap assertions on the real startup error path.
All existing retained unit and integration assertions remain. The deleted
wrapper test covered only its obsolete panic contract, not server startup.

## Final verification

Logs are `/tmp/mch-r1-*.log`; fresh profiles, block/source/package inventories,
provenance and results are under `backend/.coverage/unit/` and `.coverage/api/`.
Final production source hashes match both inventories. Unit and legacy profiles
were never merged into APIHydra coverage. The runner owns/removes its disposable
PostgreSQL cluster and server; no external database was reset.

| Command actually run | Exit and evidence |
| --- | --- |
| Backend: `go list -json ./cmd/... ./internal/... ./pkg/...`; `go list -deps -test ./cmd/... ./internal/... ./pkg/...`; `go list -deps ./cmd/server` | 0 each before editing; db has no importer and is absent from server closure |
| Backend: `go test -short -count=1 ./cmd/server ./internal/project ./internal/epic ./internal/change ./internal/testcase ./internal/health ./pkg/config ./pkg/db` | 0 baseline |
| `GOLANGCI_LINT_CACHE=/tmp/mch-r1-lint-cache make -k -C backend check` before editing | 2; baseline lint11, formatting/vet/race/tooling pass |
| Backend: `golangci-lint fmt --no-config --enable gofumpt --enable goimports cmd/server/main_test.go cmd/server/routes_test.go internal/change/api.go internal/testcase/api.go internal/project/api.go internal/epic/api.go internal/health/api.go pkg/config/config.go` | 0; only touched files; diff inspected |
| Backend: `go test -short -count=1 ./cmd/server ./internal/project ./internal/epic ./internal/change ./internal/testcase ./internal/health ./pkg/config` | 0 final targeted tests |
| `make -C backend check` | 2; formatting passes, baseline lint11; default lint cache also reports read-only writes |
| `GOLANGCI_LINT_CACHE=/tmp/mch-r1-lint-cache make -k -C backend check` final | 2; writable cache resolves cache warnings; formatting/vet/race pass; 45 Python tooling tests and Go validator pass; only baseline lint fails |
| `make -C backend coverage` | 0; 910/928 = 98.0603%, strict >95% passes |
| `make -C backend deps-audit` | 0; no vulnerabilities found |
| `make -C backend api-test` | 0; all 408 requests / 34 operations; 843/928 = 90.8405%, >=90% passes |
| `python3 -B -m unittest discover -s backend/scripts -p 'contracts_test.py' -v` | 0; both contract checks pass after documentation updates |
| `git diff --check` | 0 |

Final lint reports three unchecked shared HTTP Body.Close calls, the shared
package comment and five exported comments, and config/markdown package
comments (3 errcheck, 8 revive). All predate R1; revive report limiting changes
which missing package comments appear (baseline included db/markdown). No new
lint issue is introduced or suppressed. Repair this known debt in R5.
No final application tests/scenarios failed, skipped or were blocked.
Legacy SQL/HTTP was not rerun in R1; its retained P4 evidence remains historical.
Docker/benchmarks and the later outage campaign were not run in this pass.

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
| pkg/markdown | 10/10 | 8/10 |
| **Aggregate** | **910/928** | **843/928** |

The only denominator change from P4's 932 is the four executable statements in
the proven unused db wrapper. API fields and a comment have no statements.
Coverage was not the basis for deletion; no retained source/package is excluded.
Unit gaps remain 18 server main/start/error-handler statements. API gaps total85:
server15, change20, epic8, error10, health5, project7, testcase9, config9,
markdown2. They cover startup/config/shutdown/error branches, SQL query/scan/
iteration/Exec failures, entropy, concurrent FK/parent failures, degraded health
and Markdown failures. Domain has no executable statements. No artificial
HTTP trigger, runtime switch, weakened assertion or profile mixing was added.

## Acceptance and handoff

| Criterion | Evidence for retained behavior |
| --- | --- |
| R1-01 evidence-backed removal | Ledger audit, pre-edit Go dependency inventories, retained startup and API tests, final compile/race checks; comment deletion has no executable test |
| R1-02 exact routes, service/context/status/JSON/middleware parity | New TestAPIConstructorRouteInventory; retained project/epic TestAPIRegisteredContracts, TestChangeAPIContracts, TestDocumentAPIShapeAndExplicitBooleans, TestTestCaseAPIExactContracts, TestHealthAliasExactContracts and handler error tests; unchanged route-ledger tooling test and full APIHydra campaign |
| R1-03 startup/config/database ownership and causes | Strengthened TestStartFailures; retained TestLifecycle, TestStartCancelRealServer, TestHTTPServerPreservesReadTimeout, TestConfigurationPanicCausesAndPrecedence, TestInstalledJSONErrorContracts and central error tests; unchanged startup/flag/middleware code and owned real-server run |
| R1-04 honest checks and bounded scope | Actual exits/counts/package gaps above; final diff and source-hash audit; no tooling/denominator/dependency changes |
| R1-05 compact handoff | This checkpoint, ledger audit, refreshed API measurement and plan top status; parity/coverage evidence supports the documentation, with no prose-only unit test |

## R2 duplicate-path reassessment

Read-only review after P4/R1 inspected change/testcase services/repositories and
rendering, project/epic CRUD, central errors and process runners. P1 centralized
error handling; P3/P4 removed the duplicate testcase change scans, mutation
completion/recalculation, transaction/state paths and renderer coupling. Change's
specialized document writes already delegate to SetDocument. No remaining
maintenance defect justifies a shared abstraction.

Repeated Query/Scan/Close/Err and Exec/RowsAffected paths retain distinct SQL,
DTOs, ordering and error semantics. Local test fakes pin those separate contracts;
three-line completion arithmetic is clear in its owning services. Runner.execute
owns process groups/timeouts/cleanup while coverage.command handles short checked
tool commands: these are not equivalent operations. Preserve the architecture's
explicit simple SQL instead of inventing generic CRUD/helpers or a fake framework.
R2 outcome: **no actionable findings; no production diff or branch**.

Supervisor next action: publish this checkpoint, run `scripts/merge-to-dev.pl`,
verify clean local/remote dev, then proceed to the explicit011 validation-cause
repair already scoped separately. No stage or production promotion.
