# Backend change/document alignment checkpoint — P3 reviewed

2026-09-28; `change/008-backend-change-doc-alignment`, P2 base `22624ca`.
Factory implementation `13e762e` is published. Native review against the pinned
P2 base completed with no actionable findings or fixes. The supervisor is
publishing this checkpoint before the authorized squash merge to dev. No
transient retry was needed; no stage or production promotion occurred.
**Overall backend verification remains incomplete**: known baseline
formatting/lint failures and both final coverage gates remain visible. The
user's intermediate-pass policy explicitly permits this merge.

Next action after merge: create P4 `change/009-backend-testcase-alignment` from
fresh origin/dev. Replace old public.test_case/history/procedure SQL, Go
transactions and composite mutation responses. Remove domain.Change plus
RenderChange/RenderMutation after their last testcase consumers migrate, and
update TestChangeArchitectureAndP4Boundary's transitional consumer assertions.
No P4 behavior is claimed fixed here. Factory/review logs are
`/tmp/mch-p3-code-spec.log` and `/tmp/mch-p3-review-loop.log`.

## Implemented contract

All 14 change routes are retained; documents and set-document bring the route
inventory to 34. Current views supply nullable fields and int64 counts;
completion is service-derived. Creates return only {id}; updates/deletes return
empty 204 without mutation rereads. Stored procedures own create/title/phase/
epic/document workflows; direct updates and deletion use affected-row checks.
The selected project configuration supplies phase/type/document validation, with
no fallback; unrelated operations and reads do not require it. Caller UUIDs
are preserved, absent UUIDs use v7, duplicate UUIDs are safe centralized 409s.

Documents expose current raw content and sanitized HTML through explicit reads.
Repeated writes append; deleting a parent retains historical/current documents,
but subsequent public reads/set reject the absent parent. Real testcase FK
conflicts block change deletion without a cascade. The [ledger](backend-contracts.md)
records the exact preflight/CALL deletion race, same-project epic validation race,
and document read race; none is claimed atomic. SQL/decoder/validation/UUID
causes remain inspectable through the central error package.

P2 fixture rows and configuration assertions remain intact. Its initial global
project-list expectation was extended only with the three additional P3 fixture
projects; all six original entries/values/order remain asserted. P3 executes in
child stage p3/, after P2, and reserves IDs 2001–2407 plus testcase IDs
230001–280000. 50,000 actual testcase rows prove counts beyond int16. No root
schema, dependency, production fixture switch or Go transaction was added.

## Final verification

Commands run from repository root unless a backend working directory is stated.
Logs are `/tmp/mch-p3-*.log`; final command logs use `-final`. Coverage runs
succeeded as measurements, then their strict gates failed; those are not test
failures or passing gates. Source inventories and structural denominators passed.

| Command actually run | Exit / evidence |
| --- | --- |
| Backend: `go test -short ./internal/change ./internal/project ./internal/error` before edits | 0; baseline focused tests, not SQL compatibility |
| Backend: `go build ./cmd/...` | 0 |
| Backend: `golangci-lint fmt --no-config --enable gofumpt --enable goimports` on touched change/domain/error/server and change HTTP packages | 0; diff inspected, unrelated testcase formatting untouched |
| Backend: `go test -short ./internal/change ./internal/error ./cmd/server` and later changed-package focused runs | 0 |
| `GOLANGCI_LINT_CACHE=/tmp/mch-p3-lint-cache make -C backend check` | 2; existing internal/testcase/schema_test.go formatting blocks ordinary target |
| `GOLANGCI_LINT_CACHE=/tmp/mch-p3-lint-cache make -k -C backend check` | 2; formatting/lint fail, vet/race/tooling pass; 45 Python tooling tests plus Go suite validator pass |
| Backend: `GOLANGCI_LINT_CACHE=/tmp/mch-p3-lint-cache golangci-lint run --no-config --default standard --enable revive --timeout 5m ./internal/change/... ./internal/domain/... ./internal/error/... ./cmd/server/...` | 0; touched production packages clean |
| `make -C backend coverage` | 2 (recipe 1); valid 933/1021 = 91.3810%, strict >95% gate fails |
| `make -C backend deps-audit` | 0; no vulnerabilities found |
| `make -C backend api-test` | 2 (recipe 1); all 312 requests and owned server exit 0; valid 766/1021 = 75.0245%, >=90% gate fails |
| Backend: `python3 -B /tmp/mch-p3-db-evidence.py` | 0; owned legacy lifecycle on port 19081 executes `go test -count=1 -v ./api-tests/change ./api-tests/project ./api-tests/epic`; all seven tests pass, profiles kept separate |
| `scripts/codex-review-loop.pl agent/specs/008-backend-change-doc-alignment.md --base origin/dev` | 0; native pass 1 clean against pinned `22624ca`; no fixes, unit/tooling/APIHydra checks repeated successfully |
| `git diff --check` | 0 |

The initial check attempt also reported read-only golangci cache warnings;
final lint uses writable `/tmp/mch-p3-lint-cache`. Remaining 14 reported lint
issues are baseline: three unchecked testcase Rollback calls, three shared HTTP
client Body.Close calls, five client export comments, and package comments in
pkg/config, pkg/db and pkg/markdown. New/touched production lint is clean.
No failures are suppressed. Full legacy testcase scenarios remain deferred and
were not run; the existing database testcase test is skipped in short mode.
Docker and benchmarks were not applicable to this contract migration.

## Fresh statement counts and gaps

| Production package | Unit covered/total | APIHydra covered/total |
| --- | --- | --- |
| cmd/server | 70/88 | 73/88 |
| internal/change | 405/405 | 376/405 |
| internal/domain | 0/0 | 0/0 |
| internal/epic | 115/115 | 107/115 |
| internal/error | 43/43 | 33/43 |
| internal/health | 21/21 | 16/21 |
| internal/project | 117/117 | 110/117 |
| internal/testcase | 118/188 | 22/188 |
| pkg/config | 30/30 | 21/30 |
| pkg/db | 4/4 | 0/4 (unlinked structural zero) |
| pkg/markdown | 10/10 | 8/10 |
| **Aggregate** | **933/1021** | **766/1021** |

The first draft measured 1023 production statements. Final handler review
removed a newly introduced no-op `validate.Struct` check from rendered-artifacts:
that request has no validation tags and intentionally allows empty IDs, so its
failure arm was unreachable. JSON binding still rejects malformed/type-invalid
input, and service normalization still rejects every nonpositive ID. Removing
these two new scaffolding statements explains 1023 → 1021; no existing defensive
behavior was removed and no coverage exclusions changed.

Unit gaps: server main/start/error-handler paths (18 statements), old testcase
repository operations and UpdateTestCaseDone error handling (70). P3 change and
central errors have 100% unit statement coverage. Integration gaps include
query/scan/iteration failures, entropy failure, concurrent FK/parent removal,
legacy rendering adapter, startup/config/health/Markdown failure branches, and
most unmigrated testcase behavior. None receives fabricated HTTP triggers or
merged unit/legacy counters. P4 and later passes should address their legitimate
behavior and testing; final residual failures need explicit discussion.

Fresh ignored data: `backend/.coverage/unit/` and `backend/.coverage/api/`
contain profiles, source/block inventories, provenance and result.json; API
includes server/runner/PostgreSQL logs. `backend/.coverage/legacy-p3/` contains
separate HTTP/SQL evidence and unused counters, never merged into API coverage.

## Acceptance → named test evidence

| Criterion / individual behaviors | Unit evidence and real integration evidence |
| --- | --- |
| P3-01 explicit current columns, nulls, ordered/empty/missing reads, wide counts, service completion, no config | `TestRepositoryCurrentReads` enforces scan positions/types/SQL/order/closure; `TestServiceDerivesWideCompletion` proves zero/50/truncated66/wide70; `TestChangeAPIContracts` pins exact flat keys/nulls; `TestServiceCollaboratorFailuresAndContext` proves no config; API fixtures prove 40000/50000=80 and tied list order, absent project [], missing404 |
| P3-02 positive/trimmed create, UUIDv7 and preserved v4, ID-only function result, selected-config defaults/errors, duplicate409, no reread | `TestServiceCreateIdentityDefaultsAndFailures`, `TestServiceRejectsInvalidDirectInput`, `TestServiceCollaboratorFailuresAndContext`, `TestRepositoryCreateOnlyID`, exact `TestChangeAPIContracts`, central `TestChangeDatabaseContracts`; API default/custom/missing/duplicate cases; separate HTTP generated UUID check |
| P3-03 specialized SQL, empty204/error-only, title whitespace, phase/config, same-project/nil epic, explicit bool/URL, same-value modified, no unrelated config | `TestRepositorySingleStatementMutations`, `TestServicePreflightAndAssociation`, `TestServiceRejectsInvalidDirectInput`, `TestServiceCollaboratorFailuresAndContext`, `TestChangeAPIContracts`, `TestDocumentAPIShapeAndExplicitBooleans`; each API mutation has independent read; `TestChangeIdentityOrderingAndSameValueTimestamps` proves DB normalization/timestamp ordering with APIHydra normalization and separate HTTP timestamps |
| P3-04 ordered trim/dedup/intersection, null/empty/unknown clears, two configs, missing config, rejecting phase/doc kinds | `TestServiceSelectedConfigurationFiltering`, `TestServiceCustomPhaseAndDocuments`, `TestServiceRejectsUnknownPhaseAndDocumentKind`, `TestServiceCollaboratorFailuresAndContext`; retained P2 `TestRepositorySelectedConfig` proves stored-slug resolution; API default/custom/missing selection and clears |
| P3-05 specialized domain mapping, generic write validation/SQL, explicit boolean, repeated append, live/current ordered raw+safe documents and [] | `TestServiceDocumentMappingAndRepeatedWrites`, `TestServiceRejectsInvalidDirectInput`, `TestRepositorySingleStatementMutations`, `TestRepositoryCurrentReads`, `TestServiceSanitizesExplicitReads`, `TestDocumentAPIShapeAndExplicitBooleans`; API four kinds/current-only/repeated/sanitized reads; SQL `TestChangeDeletionRetainsAppendOnlyDocuments` verifies three increasing row IDs, two historic and one current |
| P3-06 bulk wrapper/positive ordered dedup, missing IDs/docs, live/current joins, deterministic duplicate-current choice, no config, safe rendering/raw separation | `TestServiceSanitizesExplicitReads`, `TestServiceRejectsInvalidDirectInput`, `TestRepositoryCurrentReads`, `TestChangeAPIContracts`; API mixed absent/duplicate IDs, empty list, malformed duplicate-current SQL fixture, wrong table/historical exclusion, spec+PR rendering |
| P3-07 one DELETE, FK409/missing404/success204, no cascade/purge, retained orphan history inaccessible publicly | `TestRepositorySingleStatementMutations` pins exact single DELETE, affected rows and FK cause; `TestChangeDatabaseContracts` pins safe409; `TestChangeArchitectureAndP4Boundary`; API FK conflict survivor plus post-delete get/documents/set404 and artifacts omission; separate SQL `TestChangeDeletionRetainsAppendOnlyDocuments` proves testcase remains on conflict and all docs survive deletion |
| P3-08 targeted preflight, no full reload/config, direct affected rows, honest non-atomic CALL, central causes | `TestRepositoryTargetedContext`, `TestServicePreflightAndAssociation`, `TestServiceCollaboratorFailuresAndContext`, `TestRepositorySingleStatementMutations` (including CALL zero-row success), `TestChangeHandlerReturnCauses`, `TestDocumentAPIShapeAndExplicitBooleans`, `TestServiceCreateIdentityDefaultsAndFailures`, central `TestChangeDatabaseContracts`, retained `TestBackendErrorOwnership`; API already-missing404 and ledger's exact race limitations |
| P3-09 obsolete paths removed, minimal P4 adapter retained, no testcase migration/ID rename | `TestChangeArchitectureAndP4Boundary`, `TestP4RenderingAdapterRetainsLegacyContract`, existing testcase unit tests and full build/vet/race; ledger names exact P4 consumers |
| P3-10 meaningful layered tests, 14+2 route contracts, negatives/independent reads, migrated legacy assertions, 34-operation inventory, honest gates | All tests above; `ContractsTest.test_all_registered_routes_remain_in_ledger_denominator`; 212 P3 + 100 existing APIHydra requests; seven separate migrated HTTP/SQL tests; exact create/204 assertions in unit tests; measured commands and counts above |

The [APIHydra report](../apih-tests/coverage.md) records operation coverage,
installed tool revision, assertion limits and fixture provenance. Review-ready
implementation does not imply final backend readiness or atomic race guarantees.
