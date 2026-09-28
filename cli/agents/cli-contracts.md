# CLI contracts and assertion reuse ledger

Authority: [repository instructions](../../AGENTS.md),
[CLI architecture](../../docs/cli-architecture.md),
[authorized plan](../../agent/cli-rebuild-plan.md), and
[P0 specification](../../agent/specs/019-cli-verification-foundation.md).
P0 changes verification and test infrastructure only. No application behavior,
HTTP contract, dependency version, Flow asset or prompt was migrated.

## Verification acceptance mapping

| Criterion | Meaningful tooling assertions |
| --- | --- |
| P0-01 inventory/interface | `MakefileTest.test_default_help_and_all_documented_phony_targets`, `test_package_discovery_includes_cmd_untested_and_extra`, `test_failed_or_empty_discovery_cannot_run_checks`, `test_check_runs_unit_once_and_no_program_or_pty_campaign`; `CoverageTest.test_discovery_empty_failures_and_unusual_production_locations` |
| P0-02 setup/read-only checks | `MakefileTest.test_read_only_static_checks_include_tests_and_propagate_failures`, `test_pinned_init_and_docker_no_tty_and_quoted_checkout` |
| P0-03 unit coverage | `CoverageTest.test_exact_boundaries_not_rounded`, `test_inventory_uses_all_go_packages_and_structural_metadata`, `test_denominator_missing_unlinked_and_partial_packages`, `test_malformed_and_mismatched_profiles`, `test_failed_tests_retain_raw_diagnostics_without_stale_success`, `test_below_target_preserves_profile_and_html` |
| P0-04 eligibility | `CoverageTest.test_scenario_manifest_rejects_scripts_empty_and_unmatched`, `test_events_fail_closed_and_require_every_scenario`, `test_campaign_arguments_never_import_unit_counters` |
| P0-05 child instrumentation/ownership | `TestHarnessBinarySelection`, `TestHarnessTimeoutAndOriginalExit`, `TestHarnessOwnedCleanup`, `TestHarnessCleanupRejectsUnownedPIDAndTimeout`; `CoverageTest.test_missing_child_counters_and_conversion_failure`, `test_campaign_requires_socat_and_both_suites_preserving_first_failure`; actual `TestRewriteScreenUsesColoredBlackScrollableViewport` |
| P0-06 accounting | `CoverageTest.test_overlapping_profiles_union_hits_never_sum_statements`, `test_exact_boundaries_not_rounded`, `test_linked_package_metadata_cannot_be_silently_zeroed`, `test_campaign_arguments_never_import_unit_counters` |
| P0-07 lifecycle | `CoverageTest.test_original_subprocess_exit_and_log`, `test_fresh_run_removes_stale_data_and_locks_concurrent_runs`, `test_interrupt_or_postprocessing_failure_invalidates_results`, `test_source_change_rejected`, `test_changed_production_file_set_rejected`, `test_provenance_records_staged_unstaged_and_untracked_backend_inputs` |
| P0-08 structure | `TestCLIPackageBoundariesFixtures` tests forbidden and allowed edges with a different module name and test source; `TestCLIPackageBoundaries` audits real source and exposes the existing agent→changes violation |
| P0-09 durable evidence | `DocumentationTest` checks local links, documented targets, route inventory and the complete integration reuse ledger; checkpoint records real commands separately from fixture checks |

Python assertions are in [coverage_test.py](../scripts/coverage_test.py) and
[makefile_test.py](../scripts/makefile_test.py). Go harness assertions are in
[harness_test.go](../integration/terminal/harness_test.go) and architecture
fixtures in [architecture_test.go](../integration/architecture_test.go).

## Terminal eligibility

The [manifest](../scripts/terminal-scenarios.json) is an explicit allowlist of
complete-program boundaries, not a pattern that selects every integration-named
test. `RunProgramWithIO`/`RunWithIO` drive the event loop with fake HTTP servers,
controlled editor/agent executables or injected collaborators. The startup child
and PTY child use the campaign's freshly built atomic-coverage executable and
separate private counter directories. The PTY wrapper records its owned process
group and actual application exit; orderly `/return`, `/quit` navigation flushes
counters. Forced cleanup is bounded and cannot convert a failed assertion into
success. Standalone PTY runs build their own executable.

Two misleadingly named `TestCLIProgram` tests run `make` against Flow scripts
without starting the CLI: definition review and artifact chat. They are retained
in `integration-test`, explicitly rejected from the manifest, and contribute no
coverage. The startup scenario remains eligible because its real child is now
instrumented. Direct controller/unit/client checks, architecture fixtures and
harness helper tests remain outside the terminal campaign. No existing assertion
is removed in P0.

## Backend route inventory and action migration

Observed directly from `NewAPI` registrations in backend `internal/*/api.go`.
The current client still assumes removed routes and entity responses. These
fixtures prove the legacy behavior only; P2 must revise transport assertions to
current typed requests/statuses before feature migrations. Create returns 201
with an ID (config insertion returns a slug); update/delete return 204. Document
history/current/insert replaces change artifact fields; selected-project config
replaces global option routes. No SQL or backend internals belong in CLI tests.

<!-- routes:start -->
| Method | Route | CLI action / owning pass |
| --- | --- | --- |
| POST | `/api/v1/project/list` | Projects P3: list |
| POST | `/api/v1/project/config` | Projects P3: config |
| POST | `/api/v1/project/details` | Projects P3: details |
| POST | `/api/v1/project/create` | Projects P3: create |
| POST | `/api/v1/project/update` | Projects P3: update |
| POST | `/api/v1/project/delete` | Projects P3: delete |
| POST | `/api/v1/epic/list` | Epics P4: list |
| POST | `/api/v1/epic/details` | Epics P4: details |
| POST | `/api/v1/epic/create` | Epics P4: create |
| POST | `/api/v1/epic/update` | Epics P4: update |
| POST | `/api/v1/epic/delete` | Epics P4: delete |
| POST | `/api/v1/change/list` | Changes P5: list |
| POST | `/api/v1/change/details` | Changes P5: details |
| POST | `/api/v1/change/create` | Changes P5: create |
| POST | `/api/v1/change/update-epic` | Changes P5: update-epic |
| POST | `/api/v1/change/update-after-change` | Changes P5: update-after-change |
| POST | `/api/v1/change/update-phase` | Changes P5: update-phase |
| POST | `/api/v1/change/update-open` | Changes P5: update-open |
| POST | `/api/v1/change/update-types` | Changes P5: update-types |
| POST | `/api/v1/change/update-title` | Changes P5: update-title |
| POST | `/api/v1/change/update-pr-url` | Changes P5: update-pr-url |
| POST | `/api/v1/change/delete` | Changes P5: delete |
| POST | `/api/v1/test-case/list` | Testcases P6: list |
| POST | `/api/v1/test-case/create` | Testcases P6: create |
| POST | `/api/v1/test-case/update` | Testcases P6: update |
| POST | `/api/v1/test-case/update-done` | Testcases P6: update-done |
| POST | `/api/v1/test-case/delete` | Testcases P6: delete |
| POST | `/api/v1/doc/list` | Documents P7; workflow P8–P9: list |
| POST | `/api/v1/doc/current` | Documents P7; workflow P8–P9: current |
| POST | `/api/v1/doc/details` | Documents P7; workflow P8–P9: details |
| POST | `/api/v1/doc/insert` | Documents P7; workflow P8–P9: insert |
| POST | `/api/v1/config/list` | Configurations P7: list |
| POST | `/api/v1/config/details` | Configurations P7: details |
| POST | `/api/v1/config/insert` | Configurations P7: insert |
| POST | `/api/v1/config/update` | Configurations P7: update |
| POST | `/api/v1/config/delete` | Configurations P7: delete |
| GET | `/api/v1/health` | Health P7: health check |
| GET | `/api/health` | Health P7: compatibility health check |
<!-- routes:end -->

## Existing integration assertions

| Existing test | P0 disposition and future assertion owner |
| --- | --- |
| `TestCLIPackageBoundaries` | Revised module/dto/feature/adapter enforcement; retained structural check outside terminal coverage, all passes. |
| `TestDefaultFlowPromptAndStageIntegration` | Retained outside coverage: Flow script/prompt/automation assertion, no CLI program execution. Remove obsolete assertion with Flow P1; transfer relevant process, prompt and ownership checks P8–P9. |
| `TestPromptsReadStartupGeneratedChangeTypes` | Retained outside coverage: Flow script/prompt/automation assertion, no CLI program execution. Remove obsolete assertion with Flow P1; transfer relevant process, prompt and ownership checks P8–P9. |
| `TestWorkflowPromptsRequireExplicitDocumentationScope` | Retained outside coverage: Flow script/prompt/automation assertion, no CLI program execution. Remove obsolete assertion with Flow P1; transfer relevant process, prompt and ownership checks P8–P9. |
| `TestChangeFileInitPromptPreservesOrdinaryIdeaWording` | Retained outside coverage: Flow script/prompt/automation assertion, no CLI program execution. Remove obsolete assertion with Flow P1; transfer relevant process, prompt and ownership checks P8–P9. |
| `TestDefaultFlowSessionScriptIntegration` | Retained outside coverage: Flow script/prompt/automation assertion, no CLI program execution. Remove obsolete assertion with Flow P1; transfer relevant process, prompt and ownership checks P8–P9. |
| `TestDefaultMakefileAcceptsConfiguredStageNames` | Retained outside coverage: Flow script/prompt/automation assertion, no CLI program execution. Remove obsolete assertion with Flow P1; transfer relevant process, prompt and ownership checks P8–P9. |
| `TestDefaultFlowSessionScriptsRenderPromptPaths` | Retained outside coverage: Flow script/prompt/automation assertion, no CLI program execution. Remove obsolete assertion with Flow P1; transfer relevant process, prompt and ownership checks P8–P9. |
| `TestDefaultFlowSpecReviewChatResumesInReviewWorkspace` | Retained outside coverage: Flow script/prompt/automation assertion, no CLI program execution. Remove obsolete assertion with Flow P1; transfer relevant process, prompt and ownership checks P8–P9. |
| `TestChangeBranchInitializationGuards` | Retained outside coverage: Flow script/prompt/automation assertion, no CLI program execution. Remove obsolete assertion with Flow P1; transfer relevant process, prompt and ownership checks P8–P9. |
| `TestDefaultFlowCommitScriptGuardsSlugAndPushes` | Retained outside coverage: Flow script/prompt/automation assertion, no CLI program execution. Remove obsolete assertion with Flow P1; transfer relevant process, prompt and ownership checks P8–P9. |
| `TestDefaultFlowPRPublishCreatesPRAndPushesURL` | Retained outside coverage: Flow script/prompt/automation assertion, no CLI program execution. Remove obsolete assertion with Flow P1; transfer relevant process, prompt and ownership checks P8–P9. |
| `TestChangeSlugExtractionGrammar` | Retained outside coverage: Flow script/prompt/automation assertion, no CLI program execution. Remove obsolete assertion with Flow P1; transfer relevant process, prompt and ownership checks P8–P9. |
| `TestChangeSlugGrammarIsSharedByWorkflowEntryPoints` | Retained outside coverage: Flow script/prompt/automation assertion, no CLI program execution. Remove obsolete assertion with Flow P1; transfer relevant process, prompt and ownership checks P8–P9. |
| `TestChangeDefScriptCommitsAndPushesDefinition` | Retained outside coverage: Flow script/prompt/automation assertion, no CLI program execution. Remove obsolete assertion with Flow P1; transfer relevant process, prompt and ownership checks P8–P9. |
| `TestChangeNewScriptInitializesLocalBranchWithoutPublishing` | Retained outside coverage: Flow script/prompt/automation assertion, no CLI program execution. Remove obsolete assertion with Flow P1; transfer relevant process, prompt and ownership checks P8–P9. |
| `TestChangeNewScriptReusesExistingBranchesWithoutPublishing` | Retained outside coverage: Flow script/prompt/automation assertion, no CLI program execution. Remove obsolete assertion with Flow P1; transfer relevant process, prompt and ownership checks P8–P9. |
| `TestMasterPromotionRefusesUnsafeState` | Retained outside coverage: Flow script/prompt/automation assertion, no CLI program execution. Remove obsolete assertion with Flow P1; transfer relevant process, prompt and ownership checks P8–P9. |
| `TestMasterPromotionRefusesStageThatMovesDuringPromotion` | Retained outside coverage: Flow script/prompt/automation assertion, no CLI program execution. Remove obsolete assertion with Flow P1; transfer relevant process, prompt and ownership checks P8–P9. |
| `TestCLIProgramExistingSpecUsesUUIDArtifactWorkspaceAndSpecWrite` | Retained, measured program boundary. Revise obsolete Flow/routes P1–P2; retain editor, save ordering and failure assertions P5/P8–P9. |
| `TestCLIProgramExistingDefinitionUsesUUIDArtifactWorkspaceAndDefWrite` | Retained, measured program boundary. Revise obsolete Flow/routes P1–P2; retain editor, save ordering and failure assertions P5/P8–P9. |
| `TestCLIProgramExistingPRUsesUUIDArtifactWorkspaceAndPRWrite` | Retained, measured program boundary. Revise obsolete Flow/routes P1–P2; retain editor, save ordering and failure assertions P5/P8–P9. |
| `TestCLIProgramDefReviewUsesDefinitionPromptAndSharedArtifactSession` | Retained outside coverage: Flow script/prompt/automation assertion, no CLI program execution. Remove obsolete assertion with Flow P1; transfer relevant process, prompt and ownership checks P8–P9. |
| `TestCLIProgramArtifactChatResumesSharedArtifactSession` | Retained outside coverage: Flow script/prompt/automation assertion, no CLI program execution. Remove obsolete assertion with Flow P1; transfer relevant process, prompt and ownership checks P8–P9. |
| `TestCLIProgramExistingSpecKeepsPersistedArtifactAfterFollowUpFailure` | Retained, measured program boundary. Revise obsolete Flow/routes P1–P2; retain editor, save ordering and failure assertions P5/P8–P9. |
| `TestCLIProgramNewChangeScenarios` | Retained, measured program boundary. Revise obsolete Flow/routes P1–P2; retain editor, save ordering and failure assertions P5/P8–P9. |
| `TestCLIStartupRebuildsChangeTypeSlugsPrompt` | Revised child build for coverage; retain startup assertion P1/P3 with project-scoped catalogs, remove shared prompt generation. |
| `TestRewriteScreenUsesColoredBlackScrollableViewport` | Revised instrumented child/owned exit; retain color, scrolling, dropdown, redraw and editor restoration assertions P8–P10. |

## Existing unit assertion families

| Tests | P0 and later disposition |
| --- | --- |
| `internal/app/run_test.go` | Retain all navigation, rendering, forms, save sequencing, cancellation and editor checks; revise startup/Flow P1, API orchestration P2–P7 and move ownership into features/controller P8–P9. Direct model/controller calls are unit tests. |
| `internal/app/config_test.go` | Retain repository resolution/errors and project persistence; replace Flow configuration assertions P1. |
| `internal/app/change_reference_test.go` | Retain input/identity safety until P2/P5 aligns persisted backend identity; remove obsolete branch automation expectations with P1. |
| `internal/agent/model_test.go` | Retain pure parsing, output/error handling and resource assertions where useful; replace Flow/session transitions with fixed brief/spec controller P8–P9. |
| `internal/changes/model_test.go`, `view_test.go` | Retain list filtering, artifact parsing/rendering where applicable; revise typed DTO/document ownership P2/P5/P7. |
| `pkg/client/http_test.go` | Revise exact route/payload/status assertions P2 for every route above; retain transport errors, malformed-data and decoding checks where valid. Adapter-only unit coverage. |
| cmd, epics, projects, testcases, help, navigation, ui | Existing testless production packages stay in both denominators. Add meaningful command/feature/error unit tests in P1/P3–P7/P10; zero-statement dto/styles packages are still reported. |

## Measurement and limitations

[coverage.py](../scripts/coverage.py) uses `go list -json ./...` to audit the full
module, excludes only the explicit integration/scripts trees, and includes any
production package outside cmd/internal/pkg. Per-file `go tool cover` metadata
provides the statement denominator without executing unit tests. Atomic profiles
must match that inventory; wholly unlinked integration packages become structural
zeros, while partial or missing linked-package metadata is an error. Duplicate
blocks and overlapping program/child profiles union hits by statement identity.

Unit uses only production-package tests with race detection. Terminal uses only
the manifest's program tests and covered children. Its two child directories must
both contain metadata and counters. Fresh campaigns use an exclusive lock, remove
prior success artifacts, and retain command logs, raw profiles, hashes and exact
package totals. Failure or interruption leaves `status.json` incomplete and no
success report. The valid below-target gate remains nonzero. `complete.out` adds
structural zeros for source/function/HTML diagnosis; it adds no hits.

See the [checkpoint](cli-rebuild-checkpoint.md) for real measured results, baseline
failures and next action. No live backend or real-agent compatibility is claimed.
