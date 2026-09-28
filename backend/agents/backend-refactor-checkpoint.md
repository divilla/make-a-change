# Backend refactor checkpoint — 016 final failure integration

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
