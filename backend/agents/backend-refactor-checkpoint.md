# Backend refactor checkpoint — 016 final failure integration

## API error-suite refresh (2026-10-02)

User requested missing errors that can be tested with APIHydra. This test-only
refresh adds487 requests (479 error assertions plus eight state-preservation
reads) across the existing19 files. Cases isolate omitted/null/blank/incorrect
fields, malformed numeric/UUID/array input, top-level JSON mismatches, wrong
methods for all45 routes, obsolete-route404, forbidden normal comment insertion
and each deleted-parent comment insertion. Fourteen NUL-text requests exercise
real PostgreSQL input errors and verify masked500 responses without outages;
reads prove existing state is unchanged. Captured mutation targets and cleanup
remain mandatory. No production, dependency, database/schema or root-URL edit.

| Command actually run | Exit and result |
| --- | --- |
| Focused serial `apih` doc/config lifecycles, private URL19081 | 0; current-source owned server, existing designated DB. |
| `make -C backend tooling-test` | 0;35 Python tests and Go suite validator. |
| `make -C backend api-test` | 0; **1060/1060 requests**,45/45 operations; **1090/1204 (90.5316%)**, strict90% API gate PASS. |
| `git diff --check` | 0. |

| Production package | API covered/total | API gaps |
| --- | ---: | ---: |
| cmd/server | 77/93 | 16 |
| internal/app | 41/51 | 10 |
| internal/change | 296/315 | 19 |
| internal/config | 96/100 | 4 |
| internal/doc | 188/210 | 22 |
| internal/domain | 0/0 | 0 |
| internal/epic | 131/143 | 12 |
| internal/health | 16/21 | 5 |
| internal/project | 117/125 | 8 |
| internal/testcase | 97/105 | 8 |
| pkg/config | 23/31 | 8 |
| pkg/markdown | 8/10 | 2 |
| **Total** | **1090/1204 (90.5316%)** | **114** |

This fresh API result supersedes the below-target API result in the original
schema-alignment section below. The denominator and production sources are
unchanged. Check, unit coverage and dependency audit were not rerun for this
YAML/documentation-only refresh; their preceding production-source results remain
recorded below, including independent unit1190/1204 (98.8372%). Bare `apih` against
the configured8080 server was not rerun; that server was left running.

Two initial tooling runs failed honestly on forbidden fixed-positive malformed
numbers and the extra health filename; cases were moved into existing files and
use negative fractions/underflow. No validator/assertion was weakened. The final
focused and full HTTP campaigns both pass. Final artifacts/logs are preserved in
`.coverage/verification/api-error-scenarios-20261002/`.

Missing-document mutation404, health outage503, scan/iteration failures, failed
deactivation, timed concurrent-parent deletion and inaccessible selected-config
states still lack live HTTP scenarios. Service guards rejected earlier by HTTP
remain coverage gaps rather than separate API error conditions. See the
[suite guide](../apih-tests/coverage.md) for the exact scope and limitations.
Passing the numerical gate does not establish exhaustive error coverage.

## Backend schema alignment (2026-10-02)

User-authorized backend-only alignment uses the updated checked-in `db/init.sql`
and this branch's backend changes as authority. All initial ambiguities were
resolved in chat before implementation: successful FK-delete deactivation returns
204; projects list both activity states; epic/change have inactive list endpoints;
details/updates/child creation support inactive parents; config_slug and
update-active replace old names; doc list-active, separate comment operations
(including explicitly confirmed comment-insert) and soft deletion are supported.
The user designated and updated localhost:15432/changes. No database/schema file,
CLI/frontend file, dependency or deployment was changed by this task.

| Contract | Meaningful unit evidence | APIHydra evidence |
| --- | --- | --- |
| Schema-backed project active/config_slug and all-state order | project repo/API matrices; TestRepositoryReads | project details/list and selected config reads |
| DELETE FK fallback; only FK triggers; errors/missing rows preserved;204 empty | TestDeleteDeactivatesOnlyOnDependencyConflict and TestRepositoryDeactivationPreservesFailuresAndMissingRows in project/epic; central IsError wrapping test | project/epic DELETE followed by active=false details and lists; physical deletes after children removed |
| Active/inactive epic/change lists; full counters/completion; inactive details and allowed writes/children | expanded repo/service/API matrices; TestInactiveChangeServiceValidatesAndDerivesCompletion; TestInactiveChangeAPIValidationAndErrors | empty/populated inactive lists, validation, inactive details/updates/children and reactivation |
| update-active and obsolete route absence | retained activity handler/validation tests; TestUpdateOpenRouteIsAbsent | both bool values plus inactive/active reads |
| list-active uses doc_id; history includes soft-deleted docs with nullable deleted_at | TestDocRepositoryReads; TestDocRepositoryReadsNullableAndDeletedTimestamps; TestDocumentSerializationHasNullableDeletionTimestamp; TestRenamedDocRouteIsAbsent | regular-doc history, active selection, deletion and independent retained-history reads |
| Separate comment list/insert/update; configured regular insert cannot select comment; explicit empty update supported | TestCommentAndDeleteAPIContracts; TestNewDocServiceValidationAndFailurePropagation; TestRegularInsertRejectsCommentsEvenWhenConfigured; TestCommentUpdateAcceptsExplicitEmptyBody; TestCommentsRenderRetainedHistory | each reference type; no comment active selection; edits including empty; non-comment404; soft-deleted comments retained |
| SQL function signatures and correct ID/null results; config DELETE FK enforcement | TestDocRepositoryInsertAndProject; TestNewDocRepositoryMutations; TestConfigRepositoryMutations | cast-signature calls against the updated DB; direct config delete/missing/conflict lifecycle |
| Complete operation inventory, contexts, response/status/error contracts | TestAPIConstructorRouteInventory; source/ledger audit; affected API/service/error tests | all573 requests and45 registered operations pass |

The fallback consists of two independent statements. Repository code handles
pgx operations; service code owns the FK fallback decision through the central
error API. No Go transaction, child cascade, parent activity restriction or
postmutation reload is introduced. fn_doc_insert's obsolete null-parent return
handling was removed because the current function always returns its inserted
ID; the service retains explicit parent validation. Calls cast parameters to
select the checked-in signatures in a DB with retained older overloads.

| Final command | Exit/result |
| --- | --- |
| `GOLANGCI_LINT_CACHE=/tmp/backend-align-lint-cache make -C backend check` | 0; formatting, lint, vet, race and tooling pass;35 Python tests plus Go validator. |
| `make -C backend coverage` | 0; **1190/1204 (98.8372%)**;95% gate passes. |
| `make -C backend deps-audit` | 0; no vulnerabilities found. |
| `make -C backend api-test` | 2 (runner1); all573 HTTP requests pass, valid **1082/1204 (89.8671%)**;90% statement gate FAIL. |
| `git diff --check` and scope inspection | 0; task changes confined to backend/. |

| Production package | Unit covered/total | Unit gaps | API covered/total | API gaps |
| --- | ---: | ---: | ---: | ---: |
| cmd/server | 79/93 | 14 | 76/93 | 17 |
| internal/app | 51/51 | 0 | 39/51 | 12 |
| internal/change | 315/315 | 0 | 296/315 | 19 |
| internal/config | 100/100 | 0 | 95/100 | 5 |
| internal/doc | 210/210 | 0 | 187/210 | 23 |
| internal/domain | 0/0 | 0 | 0/0 | 0 |
| internal/epic | 143/143 | 0 | 130/143 | 13 |
| internal/health | 21/21 | 0 | 16/21 | 5 |
| internal/project | 125/125 | 0 | 116/125 | 9 |
| internal/testcase | 105/105 | 0 | 96/105 | 9 |
| pkg/config | 31/31 | 0 | 23/31 | 8 |
| pkg/markdown | 10/10 | 0 | 8/10 | 2 |
| **Total** | **1190/1204** | **14** | **1082/1204** | **122** |

Remaining unit gaps are14 server startup/main statements. API gaps are122
statements, mostly startup/external DB/scan/iteration failures and service
validation intercepted at HTTP boundaries. Missing-document deletion remains
unit-tested because only soft deletion exists; no unused mutation IDs are
guessed. Final completion is **incomplete solely on the API numerical gate**:
1084/1204 would meet it; actual1082/1204 does not. The final gate stays strict.
See [the suite guide](../apih-tests/coverage.md) for limitations and all new
operation assertions. Unit and API profiles are fresh and independent.

Initial baseline `make -C backend test` exited2 with obsolete ChangeListItem.Active
and Doc.Current references plus project/epic body assertion failures. During
implementation, lint found two new test issues and the tooling audit found its
obsolete39 route count; these were repaired. Failed API campaigns (make2, runner3
or101) did not establish coverage. Raw failures and final logs are under
`backend/.coverage/verification/schema-alignment-20261002/`; original diagnostic
copies also remain in /tmp/backend-align*. Failed terminal campaigns may have left
their own records before post-file execution; no unproven ownership was inferred
for cleanup. Complete campaigns perform their captured-ID cleanup, retaining
document history as required. No stage/production promotion was performed.

## Change prerequisite display name (2026-09-30)

Change details now return nullable `after_change_name` from
`vw_change_details` alongside `after_change_id`. In that view, `epic_name` is
the plain epic name without an ID suffix. The prerequisite display name is the
prerequisite title followed by ` (#<id>)`; the association ID remains the edit
value. Repository and API unit tests cover the additional field, and the
APIHydra detail assertion expects null for an unassociated change.

| Command | Exit and result |
| --- | --- |
| `make -C backend check` | 0; format, lint, vet, race, tooling, route inventory, and APIHydra suite validator pass. |
| `make -C backend coverage` | 0; **1051/1065 (98.6854%)**, at least 95% gate passes. `cmd/server` is 79/93 (84.9462%); every other executable package is 100%. |
| `make -C backend deps-audit` | 0; no vulnerabilities found. |
| Focused isolated PostgreSQL details view tests | 0; nullable and associated `after_change_name` values and plain `epic_name` pass. |
| `make -C backend api-test` | Not run; no development/test database designated in this session. |

## Change `ref_slug` contract (2026-09-30)

`public.change.slug` stores only the editable suffix. Both change views expose
nullable `ref_slug`: `ref <= 99` is padded to three digits, larger refs are
unchanged, and `-` plus the stored suffix follows the reference. The change
list/details responses now expose this field instead of separate `ref` and
`slug` fields. `/change/update-slug` accepts only the suffix. The demo seed
also stores only the suffix.

| Command | Exit and result |
| --- | --- |
| `make -C backend check` | 0; format, lint, vet, race, tooling, route inventory, and APIHydra suite validator pass. |
| `make -C backend coverage` | 0; **1051/1065 (98.6854%)**, at least 95% gate passes. `cmd/server` is 79/93 (84.9462%); every other executable package is 100%. |
| `make -C backend deps-audit` | 0; no vulnerabilities found. |
| Focused isolated PostgreSQL `ref_slug` and demo seed tests | 0; both tests pass. |
| Full `db/tests/test_foreign_keys.py` | 1; 13 tests run, with three assertion failures: one expects four foreign keys when the schema exposes five, and two treat project/epic documents as orphaned change documents. The same failures occur against a clean archive of committed `HEAD` (11 tests). |
| `make -C backend api-test` | Not run; no development/test database has been designated in this session. API statement coverage remains unmeasured. |
| `git diff --check` | 0. |

## Historical change slug editor API (2026-09-29)

Added `POST /api/v1/change/update-slug`. The service preserves the stored Ref
prefix when present and uses the change ID until Ref is assigned; the suffix
must be nonempty lowercase `[a-z0-9_-]`. The repository updates only the slug
and timestamp. The standalone APIHydra change suite now has success, invalid
prefix, and read-back assertions using its own API-created change. Unit tests
cover route binding, service validation, and database statement outcomes.

| Command | Exit and result |
| --- | --- |
| `make -C backend check` | 0; format, lint, vet, race, tooling, route inventory, and APIHydra suite validator pass. |
| `make -C backend coverage` | 0; **1057/1071 (98.6928%)**, at least 95% gate passes. `cmd/server` is 79/93 (84.9462%); every other executable package is 100%. |
| `make -C backend deps-audit` | 0; no vulnerabilities found. |
| `make -C backend api-test` | Not run yet; no development/test database has been designated in this session. API statement coverage is unmeasured, so the at least 90% integration gate remains unverified. |
| `git diff --check` | 0. |

An initial `check` run exposed the new route missing from the route inventory;
the next run exposed the standalone suite's success-case requirement. Both
were corrected, and the final `check` run passed. No database was reset or
existing record changed during this work.

## Final supervisor verification

Final implementation is `21f9db5`, native recorder-cleanup repair `e9f4fbe`,
plus the reviewed emergency-command reap/signal repair in this branch. Native
review pass 02 was clean. Final native review of `da7481f` against pinned
origin/dev `62d9f4f` also exited 0 with no findings or file changes; it independently
passed tooling, the complete phased APIHydra campaign (864/925), and legacy.
Review log: `/tmp/mch-final-review-final.log`. Independent read-only review confirms every reported
failure-preservation, cleanup, and manifest finding is resolved. The root log
accidentally created by the review helper was moved into backend; cumulative
scope is backend plus this specification and the official plan only.

On 2026-09-28 all final-source commands exited **0**:

| Command | Result |
| --- | --- |
| `make -C backend check` | format, lint, vet, race, 62 Python tooling tests and Go YAML validator tests pass |
| `make -C backend coverage` | 911/925, 98.4865%; strict >95% passes |
| `make -C backend deps-audit` | no vulnerabilities |
| `make -C backend api-test` | 864/925, 93.4054%; >=90% passes |
| `make -C backend legacy-api-test` | six HTTP/SQL packages pass; separate counters |

Logs: `/tmp/mch-final-verified-{check,coverage,deps-audit,api-test,legacy-api-test}.log`.
The final real campaign completed 408 normal + 28 outage + 22 recovery requests
on the same server PID `1302702`, binary SHA256
`985bb47273813443fc1efd57cc8f8da535e92c16cf573876e9140bf7a51cf8da`.
All three phases, SQL postconditions, two database starts/stops, normal server
shutdown and cleanup passed. First recovery requests passed without warmup or
retries. Failed writes left original values and timestamps unchanged, proven by
SQL before recovery writes. Owned cluster directories were removed. No failed,
skipped or blocked required scenarios remain. Profiles remain independent;
production inventory is unchanged at 28 files, ten packages, 925 statements.

Real Path.write_text failure injection proves evidence persistence cannot block
server or nested database cleanup, including emergency stop. Secondary kill/reap
failures preserve the original exception and remain reported; bounded reap runs
even after kill failure, with termination handlers shielded then restored.
Native and independent reviews did not request any production metric changes.
The exact remaining uncovered blocks are in [final-coverage-gaps.md](final-coverage-gaps.md).

One initial authorization attempt timed out before any process started; verified
clean state and one bounded retry completed it. The first real campaign's
assertion failure is preserved below as historical evidence, not a measurement.
No future squash hash is asserted; final dev equality is verified after merge.


## Historical review fix 01 — evidence persistence and cleanup

Validated P2 against FV-05/FV-06: a failed campaign.json write could prevent
both cleanup entry and nested database stop commands. Cleanup now defers those
OSError failures until owned server/database shutdown, including emergency stop,
finishes. Persistence errors remain visible on stderr and fail the campaign;
secondary evidence errors cannot replace an original command or signal failure.
No application, schema, API, dependency or coverage contract changed.

Fresh verification on HEAD `21f9db5f70c96d423e87184c99f772f802172e30` plus
the uncommitted review fix (2026-09-28):

| Command from repository root | Exit and result |
| --- | --- |
| `python3 -B -m unittest discover -s backend/scripts -p api_coverage_test.py -v` | 0; 36 runner tests |
| `GOLANGCI_LINT_CACHE=/tmp/mch-016-review-lint-cache make -C backend check` | 0; formatting, lint, vet, race, 61 Python tooling tests and Go validator tests |
| `make -C backend coverage` | 0; 911/925 (98.4865%), strict >95% passes |
| `make -C backend deps-audit` | 0; no vulnerabilities |
| `make -C backend api-test` | 0; all three phases, SQL postconditions and cleanup pass; 864/925 (93.4054%), >=90% passes |
| `make -C backend legacy-api-test` | 0; all six HTTP/SQL packages pass independently |
| `git diff --check` | 0 |

Logs: `/tmp/mch-016-review-{focused,check,coverage,deps,api,legacy}.log`.
Fresh artifacts: `backend/.coverage/{unit,api,legacy}`. All API phases used
server PID 4289, binary SHA256
`f697853da6138ab5bd3d8dd543354e9a52f5e6c57d65d973c49bf62523b01482`.
The owned cluster was stopped and removed. Per-package counts and uncovered
blocks remain exactly as recorded in [final-coverage-gaps.md](final-coverage-gaps.md).
The three new regression tests inject disk-full errors at each nested cleanup
write boundary, exercise fast/emergency shutdown, preserve APIHydra 101/102/103
and signal failures, and verify invalid success artifacts are removed. Campaign
doubles now retain the actual command evidence wrapper. No failed, skipped or
blocked verification remains. No commit or push performed; caller owns both.

## Initial implementation evidence (before review fix 01)

2026-09-28. P0–P4, R1, separate011 validation-cause repair, R3–R6 and016
implementation/local review are completed; R2 had no actionable consolidation.
Current branch `change/016-backend-failure-integration`, specification HEAD
`b8c38e3`, refreshed origin/dev `62d9f4f5ff2fd55629b913b50e06932397902585`.
No production Go changes in016. The implementation working diff and source
hashes are recorded in fresh provenance. Factory owns commits/pushes, native
review and dev merge; this implementation pass has performed none of those
Git mutations and does not invent a future squash hash. No stage/prod promotion.

## Result and contract review

One covered server process and one initialized/seeded private PostgreSQL data
directory execute NORMAL → DB stop → OUTAGE → same DB restart → RECOVERY.
All408 retained normal assertions plus28 outage and22 recovery requests pass.
DB transitions, three SQL postconditions, normal server exit and final cleanup
pass before report conversion. Same server PID4313/binary/counters throughout;
no pool replacement, fixture reset or HTTP warmup after restart. Both health
aliases degrade503 and recover200. DB-dependent reads and direct writes fail500
safely while input validation remains400. Recovery independently proves write
non-effects and successful201/204 writes, reads and fixture restoration.

Whole-tree Go YAML validation emits the manifest before setup and rejects
unclassified/unknown/malformed/Debug definitions, missing/empty phases and
root-level steps. Root/defaults remain nonexecutable and typed/validated.
Ownership survives failed stops/restarts and interruption; confirmed-stopped
cleanup skips a redundant stop, uncertain shutdown retains diagnostics. Primary
command/signal failure survives secondary cleanup/provenance errors. Invalid
campaigns remove profile/report/result success artifacts; valid numerical
shortfalls retain their data and fail the unchanged gate honestly.

Reviewed final diff, full retained contract ledger, DB authority and
`/tmp/mch-final-architecture-audit.md` (pinned to merged R6). Production source
is byte-identical to origin/dev, so that source audit remains applicable:
API/service/repository boundaries, domain requests, centralized causes/errors,
ID-only201/empty204, exact project config, existing SQL procedures, append-only
documents and documented preflight races remain aligned. No Go transaction,
public hook/route, driver/schema fault, dependency upgrade or denominator
exclusion. All implementation files stay under backend; the official plan is
the only changed root-level exception. Specification remains unchanged.

## Acceptance and named meaningful tests

| Criterion | Test/evidence |
| --- | --- |
| FV-01 | Go `TestCompleteDecodedManifest`, `TestRejectIncompleteOrMalformedWholeSuite`, `TestRejectDebugYAMLForms`, `TestRepositoryManifestRetainsNormalAndAllPhases`; Python `test_suite_rejects_debug_and_copies_complete_tree`. Every decoded file hash/phase/stage is recorded; actual408 normal requests retained. Four moved files byte-identical; P2 differs in one extended list assertion only. |
| FV-02 | `test_three_phase_lifecycle_order_and_identity`, `test_success_stops_before_conversion_and_uses_private_cluster`, existing occupied-port/path-quoting/prerequisite/environment tests; real phase/PID/binary/counters and init/seed/fixture command evidence. |
| FV-03 | `outage/steps.yaml`28 real HTTP assertions, no transport errors/retries; existing module `Test*HandlerErrorContracts`, API validation-cause and health degraded tests prove exact public envelopes and early validation. |
| FV-04 | `recovery/steps.yaml`22 independent reads/mutations; normal/outage/recovery SQL files prove exact retained state/timestamps and absent inserts. Existing API ID/empty-body tests preserve APIHydra limitations. `test_postcondition_failures_block_reporting` checks all three SQL boundaries. |
| FV-05 | `test_failed_partial_restart_keeps_ownership_until_cleanup`, `test_false_success_pid_and_timeout_states_require_cleanup`, `test_every_phase_failure_preserves_exit_and_blocks_later_phases`, `test_interrupt_at_each_transition_and_phase_cleans_resources`, `test_deferred_spawn_interrupt_retains_server_handle`, `test_executed_child_does_not_inherit_blocked_termination_signal`, `test_uncertain_cluster_without_pid_retains_diagnostics`, `test_cleanup_failure_invalidates_success_and_preserves_primary_error`. |
| FV-06 | Phase records/manifest/input hashes/production inventory and actual binary identities; `test_provenance_failure_cannot_leave_success_or_replace_primary`, `test_failed_campaign_removes_partial_or_stale_success_reports`, missing-counter/conversion/server-crash and below-threshold tests; existing coverage source/inventory/lock/stale-profile tests. |
| FV-07 |58 Python tooling tests plus Go validator tests pass. Phase101/102/103/9 exits, stop/start timeout/false PID success, server crashes, both signals, secondary cleanup and provenance failures are covered. Full actual three-phase and separate legacy campaigns pass. |
| FV-08 | All commands and independent totals below; complete file/package/function/block inventory in [final-coverage-gaps.md](final-coverage-gaps.md). Source hash audit matches final tested implementation. No required failure/skipped/blocked scenario remains. |

## Commands actually run

Logs under `/tmp/mch-016-*.log`; fresh ignored artifacts under
`backend/.coverage/{unit,api,legacy}`. Each final command exits0.

| Command | Exit and result |
| --- | --- |
| `git fetch origin dev` | Initial sandbox attempt255 (read-only FETCH_HEAD); authorized escalation0, origin/dev verified against R6 |
| `apih --help`, `command -v curl jq git`, pinned upstream manual/type inspection |0; installed syntax/identity checked without upgrades |
| `gofmt -w backend/scripts/validate-apih-suite/{main.go,main_test.go}` (individual file invocations) |0; scoped helper formatting |
| Backend: `go run ./scripts/validate-apih-suite apih-tests` |0; decoded manifest |
| `make -C backend tooling-test` | Final0,58 Python tests and Go helper tests; initial2 exposed relocated health-suite test path, repaired before verification |
| `GOLANGCI_LINT_CACHE=/tmp/mch-016-lint-cache make -C backend check` |0; format-check, zero lint issues, vet, race, tooling tests |
| `make -C backend coverage` |0;911/925, strict >95% passes |
| `make -C backend deps-audit` |0; no vulnerabilities |
| `make -C backend api-test` | Final0;458 requests,864/925, >=90% passes. Initial Make2/apih101 due timezone-dependent expected timestamp; corrected to exact SQL equality before recovery mutations; failed profile discarded. Second and final campaigns pass. |
| `make -C backend legacy-api-test` |0; complete change/epic/health/project/shared/testcase HTTP/SQL packages; separate output/counters, no new phase selections |
| `git diff --check`, source-hash/normal-preservation/scope audit |0; implementation unchanged from measured inputs; documentation updated after measurement |

No benchmark or Docker check is required for this tooling/test change without
performance/toolchain changes. The early test failures above are repaired, not
suppressed. Final lint debt is zero; preparation-era compile/lint findings in
the official plan are explicitly historical. Local review moved cleanup evidence inside interruption masking; the new `test_cleanup_evidence_and_shutdown_ignore_interrupts` regression passes, and all five required targets were rerun on that final fix. Final resource cleanup is confirmed.

## Independent production counts

| Package | Unit covered/total | APIHydra covered/total |
| --- | --- | --- |
| cmd/server |77/91|76/91|
| internal/change |389/389|376/389|
| internal/domain |0/0|0/0|
| internal/epic |114/114|110/114|
| internal/error |48/48|39/48|
| internal/health |21/21|21/21|
| internal/project |116/116|112/116|
| internal/testcase |105/105|100/105|
| pkg/config |31/31|22/31|
| pkg/markdown |10/10|8/10|
| **Total** |**911/925 (98.4865%)**|**864/925 (93.4054%)**|

All28 production files/ten packages remain in the denominator; domain is
structural0/0. Unit gaps are14 main process-orchestration statements. The61 API
gaps are15 server,13 change,4 epic,9 error,4 project,5 testcase,9 config and2
Markdown statements: startup/abnormal shutdown/config choices, HTTP error-writer
failure, direct-service guards already rejected by HTTP, correct-schema scans,
second-query/preflight races, entropy/converter errors and unused helper inputs.
The [gap inventory](final-coverage-gaps.md) names every exact block, function,
behavior and practical obstacle; no defensive path was removed to improve counts.

All34 operations have successful normal assertions (100%, a separate diagnostic).
APIHydra subset matching and empty-body limitations remain covered by exact unit
contracts. SQL postconditions get zero Go credit. Readiness probes contribute
only initial synchronization; no invented per-phase statement percentages.

## Handoff

All implementation, final-source validations and final native review are complete.
The authorized factory now merges this reviewed branch to dev; the root supervisor will
verify actual local/remote dev equality and rerun checks after merge. Neither
numerical target is unmet. No stage or production promotion is authorized.


## 2026-09-28 — shared doc/config modules and schema naming

User-authorized follow-up on the current branch; no commit, merge or deployment.
All business reads formerly named `get` are now `details`. Fields, scans, SQL,
fixtures and tests use `created_at`/`updated_at` (`CreatedAt`/`UpdatedAt` in Go).
Project/epic/testcase/change API handlers and layer methods follow the same
list/details/create/delete naming. Existing user table/seed remain untouched;
no user service, session, token or attribution behavior was implemented.

Current endpoints and behavior:

- POST `doc/list`: `{ref_id,ref_table}`; all matching history, ID descending.
- POST `doc/current`: same reference, additionally `current=true`, ID descending.
- POST `doc/details`: `{id}`; exactly one stored doc, including historical docs.
- POST `doc/insert`: `{ref_id,ref_table,doc_type,body,agent_edit}`; returns201
  `{id}`. PostgreSQL locks the live parent, retires prior current rows of that
  parent/type and appends a new row even for identical content. Existing SQL doc
  procedures share `fn_doc_insert`. Reads do not perform parent preflights.
  Each ref_table uses its project's `project_docs`, `epic_docs` or `change_docs`.
  Raw stored content and sanitized HTML are returned together on explicit reads.
- No doc operations, including rendered-artifacts, remain in the change group.
- POST `change/update-after-change`: `{id,after_change_id}`; null clears the
  prerequisite, success204, invalid FK400, absent change404. `change/details`
  returns the nullable reference. No scheduling/completion enforcement is added.
- POST `config/list`, `details`, `insert`, `update`, `delete`; slug is the key,
  insert201 returns `{slug}`, update/delete204, missing404, duplicate409.
  Insert/update require all six arrays (empty arrays accepted). Update never
  changes the slug. Deletion locks project writes and rejects referenced slugs409.
- Change phases and normalized change types must occur in the selected config.
  Unknown change types now fail400 without mutating state instead of being
  silently discarded. Doc kinds must occur in the respective configured array.

`seed-demo.sql` now includes project/epic docs and nullable prerequisite examples.
It was executed twice in an owned disposable PostgreSQL cluster; neither the
application database nor the user's server was used. SQL assertions cover exact
retained history, returned IDs, repeated appends, current-row uniqueness,
attribution defaults, missing parents, and config deletion protection.

### Final commands and outcomes

| Command actually run | Exit | Result |
| --- | --- | --- |
| `make -C backend format` | 0 | Applied formatting; diff inspected |
| `GOLANGCI_LINT_CACHE=/tmp/mch-golangci-cache make -C backend check` | 0 | Format, lint (0 issues), vet, uncached race tests, 62 Python tooling tests and Go suite-validator tests pass |
| `make -C backend coverage` | 0 | Fresh unit profile: **1031/1045 = 98.6603%**, strict >95% gate passes |
| `make -C backend deps-audit` | 0 | No vulnerabilities found |
| `make -C backend api-test` | 0 | Fresh APIHydra-only profile: **980/1045 = 93.7799%**, >=90% gate passes |
| `python3 /tmp/verify_seed.py` | 0 | Owned cluster: init, seed, demo seed, `db/tests/doc_config.sql`, `change_history.sql`, `test_case.sql`, repeated demo seed all pass |
| `git diff --check` | 0 | No whitespace errors |

The seed verification script uses the existing Runner/private_cluster helpers,
loads only the named SQL files with ON_ERROR_STOP, and shuts down its owned
cluster. Its output is in `.coverage/seed/runner.log` and `/tmp/backend-seed.log`.
The API campaign ran 521 normal, 37 outage and 22 recovery requests (580 total),
with all SQL postconditions and cleanup checks passing. No unit or legacy HTTP
profiles were merged into the integration profile. Production source hashes in
both coverage inventories were checked against the final files with no changes.

| Package (mch_api/) | Unit covered/total | APIHydra covered/total |
| --- | ---: | ---: |
| cmd/server | 79/93 | 78/93 |
| internal/app | 52/52 | 43/52 |
| internal/change | 275/275 | 267/275 |
| internal/config | 101/101 | 100/101 |
| internal/doc | 127/127 | 119/127 |
| internal/domain | 0/0 (no executable statements) | 0/0 |
| internal/epic | 114/114 | 110/114 |
| internal/health | 21/21 | 21/21 |
| internal/project | 116/116 | 112/116 |
| internal/testcase | 105/105 | 100/105 |
| pkg/config | 31/31 | 22/31 |
| pkg/markdown | 10/10 | 8/10 |
| **Total** | **1031/1045** | **980/1045** |

The only unit gap is 14 startup/main statements in cmd/server. Integration gaps
are distributed as shown, including scan/iteration failures, direct-caller
validation unreachable after API validation, and startup/error branches. No
production package was excluded. Earlier development runs failed on stale
removed-route types, timestamp identifier formatting/lint, old error-message
assertions and the old suite request count; all were corrected. Initial lint
cache warnings were resolved by using a writable /tmp cache. No failing,
skipped or blocked scenario remains in the required final checks. Legacy HTTP,
benchmarks and Docker toolchain checks were not run; there was no toolchain or
performance-contract change.


## 2026-09-28 — APIHydra-only parallel endpoint groups

User-requested follow-up removes `backend/api-tests/` entirely, the
`legacy-api-test` target, `scripts/run-api-tests.sh`, legacy campaign branches,
and Go harness package/environment references in Make. Application Go code was
not changed in this follow-up. Earlier entries describing the legacy harness
are historical, superseded by this entry and `apih-tests/coverage.md`.

Normal HTTP cases are organized as change/config/doc/epic/health/project/testcase,
with `01-init.yaml`, `02-main.yaml`, and optional `03-post.yaml`. Mode1 runs
independent directories concurrently and their files serially. Cross-module
parent setup uses each group's init file; dependent count/history assertions
remain alongside the mutations they verify. Fixture identity ranges and capture
names are isolated; the validator tests local producer/consumer ordering and
rejects duplicate captures. SQL outage/recovery phases remain separately ordered
around the database lifecycle. A normal selection cannot include outage tests.

Timestamp comparisons from the removed Go harness are preserved using a
fixture-only update audit plus SQL postconditions, including every same-value
write. UUIDv7, list ordering after mutation and retained docs after parent deletion
are asserted through APIHydra. Public application schema/functions were not
changed for test coverage. SQL fixture assertions contribute no Go counters.

Final validation (all exits0):

- `make -C backend tooling-test`: 60 Python tooling tests and Go validator tests.
- `GOLANGCI_LINT_CACHE=/tmp/mch-golangci-cache make -C backend check`: formatting,
  lint (zero issues), vet, race, tooling tests; no Go harness packages remain.
- `make -C backend coverage`: fresh **1031/1045 (98.6603%)** unit statements.
- `make -C backend deps-audit`: no vulnerabilities.
- `make -C backend api-test`: fresh **980/1045 (93.7799%)** APIHydra statements;
  540 normal +37 outage +22 recovery =599 successful requests, 38/38 operations.
- `git diff --check`: no whitespace errors.

Both complete parallel campaigns passed; the second includes the added
UUID/order/history/timestamp assertions. No failed, skipped or blocked scenario
remains. Unit and API package gaps are unchanged from the preceding table:
cmd/server is the only unit gap (14 statements); API gaps remain startup/errors,
scan/iteration and direct-service validation branches. No profiles were merged
with unit or legacy data. Artifacts are under `.coverage/api/` and `.coverage/unit/`.
The suite guide records exact package counts and the manual selection commands.

## 2026-09-28 — Standard APIHydra output

`api-test` now inherits terminal stdout/stderr for APIHydra and prints a blank
line before the coverage report. Setup commands still write to `runner.log`;
APIHydra failures point to the displayed output. Exit and cleanup behavior is
unchanged. This follow-up changes only tooling and documentation.

Validation: `make -C backend tooling-test` exited0 (62 Python tests plus Go
validator tests); `make -C backend api-test` exited0 with all three phases
passing and fresh **980/1045 (93.7799%)** API statement coverage. The terminal
output showed standard APIHydra results, then the blank line and package report.
Unit coverage was not rerun for this tooling-only change; the preceding
**1031/1045 (98.6603%)** measurement still describes the unchanged production Go
code. Package gaps remain as recorded above. No failed, skipped or blocked
scenarios.

## 2026-09-28 — Local Docker database setup

Docker PostgreSQL now publishes host port15432 and declares `changes` as its
initial database. Root and backend database targets and the development config
use that port; root Make exports `DATABASE_URL` to the server. Removed an unused
frontend binding that caused the development ESLint error. Restored the APIHydra
suite's canonical port19080 for the isolated runner.

With explicit user authorization, created the missing Docker `changes` database
and ran `make db` (init.sql, seed.sql, seed-demo.sql), exit0. The first creation
attempt encountered PostgreSQL startup; after readiness succeeded, creation and
refresh passed. Result: 3 projects, 5 epics, 200 changes, 590 docs, 600 testcases.
Backend health and frontend HTTP checks both returned200. No existing database
was reset; the three scripts ran against the newly created `changes` database.

Validation: frontend ESLint and typecheck exited0; backend tooling-test exited0
(62 Python tests and Go validator tests); the final `make -C backend api-test`
exited0 with **980/1045 (93.7799%)** API statement coverage. Unit coverage was
not rerun because production Go code is unchanged; its preceding measurement is
**1031/1045 (98.6603%)**, with package gaps recorded above. API tests used their
own disposable cluster, separate from Docker development data.

## 2026-09-28 — Standalone APIHydra suite (current)

The user's latest requirement supersedes the owned-database runner and API
statement gate: manual `apih` must work against the configured running backend,
with setup and cleanup performed through endpoints and captured response values.
Removed API SQL fixtures/postconditions, forced outage/recovery files and
`scripts/api_coverage.py` with its obsolete lifecycle tests. `api-test` now runs
`apih apih-tests` directly and reports statement coverage as unmeasured after a
blank line. No database client, server build, shutdown or direct SQL runs in
that target. Unit coverage tooling and its strict >95% gate remain unchanged.

The default suite URL is port8080. Each of the seven parallel groups uses its
own init/main/post files, captured IDs and deletion order. Config slugs derive
from an API-created project ID. No shared default config is mutated. Missing
mutation targets come from records deleted by that group's earlier requests.
Only read-only oversized-ID probes retain literal positive IDs. Global lists
allow existing data; docs retained by normal API deletion remain as history.
Cases that require unexposed database state are explicitly excluded in the suite
guide, not relabeled as passing. Seven duplicate negative requests produced by
replacing fixed IDs with captures were removed.

Final validation (temporary writable XDG_CACHE_HOME for APIHydra in the sandbox):

- `make -C backend tooling-test`: exit0, 24 Python tests plus Go suite-validator
  tests. Tests check direct Make invocation/failure propagation, local ordered
  captures, API-only mutation targets, absence of SQL and the route inventory.
- `make -C backend api-test`: exit0, **469 requests** in19 steps files, covering
  **38/38 registered operations** with successful cases.
- Plain `apih` from `backend/apih-tests/`: exit0 on the same server after the
  Make run. Repeated manual runs also passed without resets or SQL setup.
- Before/after full project/config responses compared equal during a repeat
  run, proving that existing active records were preserved and owned active
  records were cleaned up. Older unrelated partial-run records were untouched.
- `git diff --check`: exit0.

Initial authoring runs exposed incorrect new assertions about create-time title
spacing, missing epic status, last_ref and project-list ordering; corrected
against the implementation. An initial sandbox run needed a writable APIHydra
cache, and an invocation from the repository root without a suite selection
was corrected to the documented suite directory. Final runs have no failed,
skipped or blocked requests. No application Go code changed, so unrelated
production checks were not rerun for this test/tooling-only follow-up.

API statement coverage is **unmeasured**, explicitly accepted by the user.
The earlier 980/1045 value is historical and does not describe this suite; its
obsolete generated `.coverage/api/` report was removed. Unit coverage was not
rerun here; the previous production measurement is **1031/1045 (98.6603%)**,
with the unchanged cmd/server gap recorded above. Operation coverage is not
substituted for statement coverage. `AGENTS.md`, Make and the suite guide now
agree on the standalone contract.

## 2026-09-28 — Restore measured API coverage without DB control (current)

The user clarified that removing the API coverage threshold did not mean
removing reporting. `make -C backend api-test` now builds an instrumented backend
on an owned port (default 19080), runs a copied standalone suite against it, and
stops only that server to collect counters. It uses the existing DATABASE_URL;
there are no database clients, SQL fixtures, resets, outage phases, container
commands or DB lifecycle actions. The checked-in suite still targets port 8080
and remains directly runnable with `apih`. TCP readiness does not add HTTP
traffic. Counters include the owned server's normal startup and shutdown.

Standard APIHydra output is followed by a blank line and the full package/total
statement report. The source inventory includes every production package, even
if unlinked or unexecuted. Threshold enforcement is explicitly disabled only
for this diagnostic report; the strict unit gate is unchanged. Test failures,
missing counters, invalid profiles and abnormal shutdown remain errors; failed
runs discard coverage/report/result files. The runner logs provenance and the
binary and suite hashes without recording DATABASE_URL credentials.

Validation (all final commands exit 0):

- `make -C backend tooling-test`: 33 Python tests plus Go suite-validator tests.
  Added coverage tests for below-threshold reporting with actual counts, native
  output/blank-line ordering, copied-suite isolation, occupied ports, process
  signal masks, child timeouts, failed runs and cleanup/report invalidation.
- `make -C backend api-test`: 469 requests, 38/38 operations; fresh
  **943/1045 (90.2392%)** API statements, diagnostic with no coverage gate.
- `make -C backend coverage`: fresh **1031/1045 (98.6603%)** unit statements,
  strict >95% gate passes. Production Go source is unchanged.
- Existing development server health still returns 200 after the coverage run.
- `git diff --check`:exit 0.

API package covered/total counts: cmd/server 76/93; internal/app 40/52;
change 261/275; config 96/101; doc 112/127; domain 0/0; epic 106/114; health 16/21;
project 109/116; testcase 96/105; pkg/config 23/31; pkg/markdown 8/10.
The 102 uncovered statements include startup/error paths, database failures and
scan/iteration branches. Unit coverage's only gap remains 14 cmd/server statements.
No failed, skipped or blocked scenario remains. Artifacts are under
`.coverage/api/` and `.coverage/unit/`; no profiles were combined. AGENTS.md,
Make, the suite guide and the refactor plan reflect the restored reporting.

## 2026-09-29 — Coverage gate policy update

The current user policy is at least 95% backend unit statements and 90% APIHydra
integration statements. The fresh API runner now returns failure for a complete
below-threshold profile while preserving its valid report and raw counters;
failed or incomplete scenarios still invalidate success artifacts. The current
suite uses an owned instrumented backend against the existing development
database and does not manage database lifecycle.

`make -C backend tooling-test` exits 0, including exact boundary and runner exit
tests. `make -C backend coverage` exits 0 with **1031/1045 (98.6603%)** fresh unit
statements. `make -C backend api-test` was not run for this tooling change because
no user-designated development database was provided in this session. The prior
**943/1045 (90.2392%)** API result predates this gate change and is historical
evidence, not a fresh passing campaign for this checkout. No backend production
Go source was changed.
