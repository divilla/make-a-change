# Verification foundation checkpoint — P0 reviewed; sequence resumed

2026-09-28; branch `change/005-backend-verification-foundation`; base `7dc3aa0`.
Infrastructure and the bounded compilation repairs are implemented. **Backend
readiness is not achieved: neither coverage target is met, and check fails.**
The factory committed and published the specification (`b76be3d`) and bounded
implementation (`a238ecf`), supervisor fixes (`e5659f6`), and review fix (`313702a`). No merge, P1 migration, dependency or schema change
was performed; dev remains at `7dc3aa0`. Implementation writes stayed in backend;
the specification is the authorized documentation exception. The next action is
merge P0 to dev and execute P1, per the user's clarified final-result coverage policy; later contract passes remain queued. Native review completed successfully with no remaining
actionable findings on `313702a`. The user authorizes intermediate merges with
recorded coverage shortfalls; the measured gates remain visibly failed.

## Implementation and contract scope

- `change.Repo` now satisfies its interface. Details scans current-view fields
  into flat `domain.ChangeDetails`; it no longer promises obsolete entity/version/
  document/testcase placeholders. Completion is derived from real view counts.
  List's total counter scan now uses a pointer. The exact limited contract
  correction and future replacements are in [backend-contracts.md](backend-contracts.md).
- Type updates are error-only in repository/service, with missing-row detection
  and empty HTTP 204. Normalization/filtering remains; the obsolete lookup read
  still blocks the real operation and belongs to P3, not a fabricated success.
- Main handles SIGINT/SIGTERM, shuts down HTTP, waits for Serve, closes owned
  resources and returns normally. Startup/serve/shutdown errors remain failures.
  Config/environment/flag precedence, route registration and middleware remain.
  No shutdown endpoint or coverage-specific server branch was added.
- `coverage` uses fresh short uncached race tests and structural Go block metadata
  across cmd/internal/pkg, exact integer gating (>95%), package/file/block audits
  and stale artifact removal. `coverage-html` retains valid below-gate HTML.
- `api-test` builds with atomic coverage, owns a private Unix-socket PostgreSQL
  cluster and server, loads authoritative SQL, runs all APIHydra suites, stops and
  waits before conversion, audits linked metadata, counts unlinked source as
  zero, and enforces >=90%. Readiness/commands/cleanup have deadlines. Failures,
  signals, startup errors, occupancy and failed conversion cannot establish a
  passing result. Locked artifact directories prevent concurrent-run deletion.
- Legacy Go suites remain, with a separate private runner and artifact directory.
  The old runner that could reset an externally supplied test database was
  replaced. Legacy output never supplies APIHydra coverage.

## Final validation evidence

Commands were executed from repository root. Make uses exit 2 to report a failed
recipe; inner coverage gates return 1.

| Command actually run | Exit | Evidence |
| --- | ---: | --- |
| `make -C backend test` | 0 | Fresh short tests, including the new repair/lifecycle/health contracts |
| `make -C backend tooling-test` | 0 | 44 Python tooling regressions plus 3 Go YAML-validator tests |
| `make -C backend check` | 2 | Stops at pre-existing formatting differences |
| `make -k -C backend check` | 2 | Exposes lint too; vet, race and tooling passed |
| `make -C backend coverage` | 2 | Valid **576/1209 = 47.6427%**, strict >95% gate fails |
| `make -C backend deps-audit` | 0 | No vulnerabilities found |
| `make -C backend api-test` | 2 | Real APIHydra and server exit 0; **168/1209 = 13.8958%**, >=90% gate fails |
| `git diff --check` | 0 | No whitespace errors |

Only modified Go files were formatted with pinned golangci-lint's gofumpt and
goimports formatters. Broad pre-existing formatting changes were deliberately
left outside this pass. The final expanded check reports 20 existing lint issues:
6 errcheck, 11 revive, 3 unused. They include unchecked legacy HTTP body closes
and transaction rollbacks; missing comments/error naming; the existing request
logger's unused context argument; unused change helpers. New test lint findings
were fixed before final checks. Existing schema-mock tests pass but do not prove
old SQL matches the database (see the ledger).

Latest command logs are `backend/.coverage/verification/review-01-*.log`; unit reports under
`.coverage/unit/`; API reports/counters under `.coverage/api/`. These ignored
artifacts are fresh working-tree measurements, not claimed as clean-base results.
The checked-in summary preserves the counts when local artifacts are absent.
A repeat initially rejected port 19080 during address reuse; no foreign process
was stopped. The occupancy probe now permits TIME_WAIT reuse like Go's listener,
with a regression proving recently closed owned sockets can be reused while an
occupied non-HTTP listener survives. The final full API run completed successfully
before its expected coverage failure.

## Package statement gaps

| Production package | Unit covered/total | APIHydra covered/total |
| --- | ---: | ---: |
| cmd/server | 71/101 | 75/101 |
| internal/change | 336/562 | 19/562 |
| internal/domain | 0/0 (no executable statements) | 0/0 |
| internal/epic | 26/142 | 9/142 |
| internal/error | 0/1 | 0/1 (unlinked structural zero) |
| internal/health | 20/21 | 16/21 |
| internal/options | 6/38 | 6/38 |
| internal/project | 24/104 | 9/104 |
| internal/testcase | 77/197 | 9/197 |
| pkg/config | 8/30 | 21/30 |
| pkg/db | 0/4 | 0/4 (unlinked structural zero) |
| pkg/markdown | 8/9 | 4/9 |
| **Aggregate** | **576/1209** | **168/1209** |

Every package with executable statements still has unit and integration gaps.
API operation coverage is separately 2/33, not a substitute for these statement
counts. No profiles from unit tests or the legacy HTTP harness were merged.

## Acceptance-to-test mapping

| Criterion | Named tests and evidence |
| --- | --- |
| P0-01 current compilation, bound/scanned values, errors, missing records, HTTP contracts | `TestDetailsCurrentViewAndMissingRecords`, `TestTypeMutationErrorOnly`, `TestListScansTotalPointer`, `TestServiceReturnsCurrentChangeDetails`, existing service normalization/error cases, `TestChangeAPIContracts`, compile-time Repository assertion |
| P0-02 cancellation, startup/serve/shutdown failure, cleanup/order | `TestLifecycle` (cancel/startup/serve/shutdown), `TestStartFailures`, `TestStartCancelRealServer`, `TestHTTPServerPreservesReadTimeout`; actual API server counters after SIGTERM and exit 0 |
| P0-03 strict boundaries, denominator, malformed/missing/stale/failed unit profiles, HTML | `CoverageTest.test_exact_boundaries_not_rounded`, `test_denominator_missing_unlinked_and_partial_packages`, `test_malformed_and_mismatched_profiles`, `test_source_change_rejected`, `test_changed_production_file_set_rejected`, `test_fresh_run_removes_stale_data_and_locks_concurrent_runs`, `test_failed_tests_delete_partial_profile_and_no_report`, `test_html_generated_for_valid_below_threshold`, `test_inventory_uses_all_go_packages_and_structural_metadata` |
| P0-04 isolated lifecycle, prerequisite/environment/SQL/command errors, ownership, signals, bounded waits | `APICoverageTest.test_missing_prerequisite_prevents_setup`, `test_suite_exit_codes_preserved_and_owned_resources_stop` (101/102/103/7), `test_execute_preserves_actual_external_exit_codes`, `test_success_stops_before_conversion_and_uses_private_cluster`, `test_sql_startup_and_conversion_failures_remain_visible`, `test_occupied_non_http_service_survives`, `test_recently_closed_owned_listener_can_be_reused`, `test_readiness_timeout_cleans_up_without_running_suite`, `test_timeout_kills_owned_child_and_shutdown_is_bounded`, `test_interrupt_cleans_owned_command_group`, `test_environment_ignores_external_pg_settings_and_uses_private_cache`; real PostgreSQL/APIHydra smoke |
| P0-05 >=90 boundary, conversion, metadata/counters, linked/unlinked audit, failure preservation, separation | `test_exact_boundaries_not_rounded`, `test_linked_package_metadata_cannot_be_silently_zeroed`, `test_denominator_missing_unlinked_and_partial_packages`, `test_server_crash_and_missing_counters_rejected`, `test_unsuccessful_server_exit_cannot_supply_coverage`, `test_valid_below_threshold_result_is_failure`, `test_failed_campaign_removes_partial_or_stale_success_reports`, `test_legacy_output_never_enters_api_profile`, SQL/conversion and stale-directory tests above; real covdata output |
| P0-06 registered aliases, stable/exact response, full route denominator | `TestHealthAliasExactContracts`; both health YAML steps; `ContractsTest.test_all_registered_routes_remain_in_ledger_denominator`, `test_health_suite_has_explicit_contracts_for_both_aliases`; ledger and coverage report |
| P0-07 help/versions/check wiring/failure visibility/checkpoint | `MakefileTest.test_help_is_phony_and_does_not_resolve_packages_or_run_tools`, `test_install_pins_both_tools_without_editing_module`, `test_every_check_propagates_tool_failure`, `test_coverage_targets_delegate_to_strict_runner`, `test_api_target_runs_instrumented_apih_driver`, `test_legacy_runner_uses_private_lifecycle_and_separate_output`; command table above |

## Deferred work and limitations

P1 owns error centralization, including existing module/startup/connector error
handling. P2 owns project/epic current-schema responses and replacement of
options by project-selected config. P3 owns remaining change state/mutation/doc
contracts and old counter widths. P4 owns testcase table/procedure/history and
composite mutation contracts. Existing Go-managed transactions remain only in
those deferred paths; none was introduced here. These are real runtime blockers,
not skipped tests being counted as success.

The APIHydra exact-object/empty-body limitations and binary/manual provenance
are documented in [coverage.md](../apih-tests/coverage.md). No real legacy HTTP
campaign, Docker toolchain check, benchmark or broad business workflow campaign
was run; those are not claimed. Do not expand into P1–P4 simply to raise coverage.

The coordinator updated root AGENTS.md with the user's clarified coverage
policy and current runner status under the previously authorized rules update. Root PRD, skeleton and
agent/architecture.md are absent and were not used to override backend contracts.
The spec directs the implementation log to `backend/implementation-log.md`.

## Supervision and review outcome

The implementation factory completed once; no transient retry was needed.
Supervisor review repaired provenance (backend-relative staged/unstaged diffs and
untracked input hashes), private PostgreSQL path quoting/URL encoding, interrupted
server shutdown ownership, and YAML breakpoint detection. A real API run with
spaces, ampersands and hash signs in TMPDIR passed both health assertions and
cleaned up its server/database. Structural coverage still includes every
production package; the YAML validator is test tooling using an existing module.

Additional regressions: `test_private_socket_handles_spaces_and_query_characters`,
`test_interrupted_wait_preserves_server_for_final_cleanup`,
`test_provenance_records_staged_unstaged_and_untracked_backend_inputs`,
`TestRejectDebugYAMLForms`, `TestAcceptBodiesAndRejectMalformedYAML`, and
`TestDirectoryValidation`.

`scripts/codex-review-loop.pl agent/specs/005-backend-verification-foundation.md
--base origin/dev` exited 0 after two review passes and one committed/pushed fix.
The base was pinned to `7dc3aa0`. The fix restores Echo's 30-second HTTP read
and inherited idle timeout while retaining the header timeout; its regression
failed before the fix and passed afterward. Required checks and both coverage
measurements were rerun on that production code. Review pass 2 found no further
actionable regressions. Transcript: `/tmp/mch-p0-review-loop.log`.

Current reviewed implementation: `313702a`. Subsequent checkpoint/report edits
are documentation only. No merge-to-dev or promotion ran; dev remains `7dc3aa0`.
The user clarified that >95% unit / >=90% integration apply to the final refactor
result, not intermediate merges. Continue all bounded specifications, reviews
and dev merges while improving meaningful coverage. Keep actual failed gates
visible; do not stop solely for coverage, even if the final result needs a
shortfall discussion. No test exclusions or combined unit/API profiles were used.
The next action is merge-to-dev, then specification 006 backend error contracts.
