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
| POST | `/api/v1/project/list` | Projects P2: list |
| POST | `/api/v1/project/config` | Projects P2: config |
| POST | `/api/v1/project/details` | Projects P2: details |
| POST | `/api/v1/project/create` | Projects P2: create |
| POST | `/api/v1/project/update` | Projects P2: update |
| POST | `/api/v1/project/delete` | Projects P2: delete |
| POST | `/api/v1/epic/list` | Epics P3: list |
| POST | `/api/v1/epic/details` | Epics P3: details |
| POST | `/api/v1/epic/create` | Epics P3: create |
| POST | `/api/v1/epic/update` | Epics P3: update |
| POST | `/api/v1/epic/delete` | Epics P3: delete |
| POST | `/api/v1/change/list` | Changes P4: list |
| POST | `/api/v1/change/details` | Changes P4: details |
| POST | `/api/v1/change/create` | Changes P4: create |
| POST | `/api/v1/change/update-epic` | Changes P4: update-epic |
| POST | `/api/v1/change/update-after-change` | Changes P4: update-after-change |
| POST | `/api/v1/change/update-phase` | Changes P4: update-phase |
| POST | `/api/v1/change/update-open` | Changes P4: update-open |
| POST | `/api/v1/change/update-types` | Changes P4: update-types |
| POST | `/api/v1/change/update-title` | Changes P4: update-title |
| POST | `/api/v1/change/update-pr-url` | Changes P4: update-pr-url |
| POST | `/api/v1/change/delete` | Changes P4: delete |
| POST | `/api/v1/test-case/list` | Testcases P5: list |
| POST | `/api/v1/test-case/create` | Testcases P5: create |
| POST | `/api/v1/test-case/update` | Testcases P5: update |
| POST | `/api/v1/test-case/update-done` | Testcases P5: update-done |
| POST | `/api/v1/test-case/delete` | Testcases P5: delete |
| POST | `/api/v1/doc/list` | Documents P6; workflow P8–P9: list |
| POST | `/api/v1/doc/current` | Documents P6; workflow P8–P9: current |
| POST | `/api/v1/doc/details` | Documents P6; workflow P8–P9: details |
| POST | `/api/v1/doc/insert` | Documents P6; workflow P8–P9: insert |
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

## P1 startup and Flow-removal migration (020)

P0 rows above are the historical inventory. P1 supersedes their future dispositions
as follows; current HTTP fixtures still characterize the retained legacy adapter,
not completed P2 backend compatibility.

| Criterion | Current assertions |
| --- | --- |
| P1-01 | `TestStartupConfigIndependentOfAgentAndFlow`, `TestConfigRequiredAndMalformedValues`, retained root-resolution/config tests; `TestCLIProgramStartupNavigationAndSelection`, `TestCLIStartupWithoutFlowResources` |
| P1-02 | `TestRemovedCommandsCannotDispatchProcesses`, retained exact commands/help/render/navigation assertions; `TestUnavailableEpicActionsNeverClaimSuccess`; full-program navigation and PTY |
| P1-03 | `TestAtomicConfigReplacementAndFailureCauses`, `TestReplaceFileAtomicallyPreservesExistingFileAfterPartialWriteFailure`, `TestProjectSelectionSaveIsAsynchronousAndKeepsPartialSuccess`, `TestProjectSelectionSerializesOverlappingSaves`; program successful/failed config save |
| P1-04/P1-05 | `TestFixedPromptInventoryAndResponsibilities`, startup absence of default resources, existing-workspace sentinel in program selection test; scoped tracked-asset diff |
| P1-06 | Retained ordinary project/change CRUD, parsing, editor errors, selection, rendering, atomic-write and clipboard unit assertions; `TestCLIProgramEditorSaveAndFailure` covers real editor handoff/redraw and failed save |
| P1-07 | `TestShellNavigationEditorAndScrolling`, `TestCLIProgramStartupNavigationAndSelection`, `TestCLIProgramEditorSaveAndFailure`, `TestCLIStartupWithoutFlowResources`; explicit manifest and eligibility/completeness fixtures |

Prompt inventory inspected before removal: def-rewrite preserved intent, concrete
examples, fenced code and draft scope; spec-write supplied the implementation-ready
sections, clarification gate and evidence requirements; spec-review supplied bounded
review, actionable findings and explicit success. These responsibilities now belong
to brief-rewrite, brief-resolve, spec-write, spec-review and spec-fix respectively.
The external spec-file-structure content is incorporated into spec-write; no shared
change-types file is generated. Session-new, def-review, PR/code/polish/review/fix,
merge/stage/master and change-types prompts were obsolete workflows or dependencies.
The inspected Makefile/scripts performed stage dispatch, prompt substitutions,
session restoration and branch/commit/push/deploy automation. None is a general
adapter needed by ordinary API/editor startup, so all tracked default scripts,
Makefile and YAML were removed. Root factory scripts and dependencies are unchanged.
No `.mch/config.yaml` edit was needed: it had no Flow settings. Existing `.mch/tmp/`
and unrelated files remain untouched.

All tests from default_flow_test.go and the two Flow-only make tests from
existing_artifact_program_test.go are removed alongside their explicitly removed
Flow assets/behavior (including already missing root helper expectations). The five
UUID/artifact/agent complete-program tests are replaced by the three shell/startup
manifest tests above. Their synchronized driver, editor subprocess handoff, failure
reporting, user-file ownership and orderly shutdown assertions survive. Legacy
workspace/session reuse, stage hooks, branch automation and agent-edit follow-up
ordering are no longer product behavior. Fixed-controller draft/save ordering and
revision/error recovery must be added with P8–P9 rather than retain those protocols.

The PTY driver retains its instrumented binary, owned process-group cleanup,
application exit/counter checks, color, dropdown, editor redraw, and scrolling
assertions. It now scrolls a real changes list instead of an agent viewport.
Scripted progress, cancellation, bounded readers and interrupted-agent cleanup must
return with the general P8 process adapter and P9 controller. The old runner was
entirely coupled to legacy workspaces/session prompts; it is removed, including the
pipe/Wait race, not retained as an unused broken adapter. P8 must prove readers finish
before Wait closes owned pipes with a deterministic subprocess regression.

Agent unit tests for session extraction, workflow environment, workspace replacement,
restoration, generated-spec testcase creation and event rendering are removed with
those behaviors. ParseGeneratedChange/QA extraction had no ordinary caller; its
wrapper and obsolete generated-testcase behavior are removed. Ordinary pure
ParseSpecStructure/ParseDefStructure/ParseArtifactTypes assertions remain in changes.
Definition preview syntax coloring had no surviving UI consumer and is removed
with its preview; ordinary list/details color and wrapping tests remain. Branch
reconciliation unit tests and assign-flow transport assertions are removed with
/reference and its obsolete endpoint. Placeholder epic navigation/save assertions
are replaced by explicit unavailable-action checks, with CRUD owned by P4.

Startup catalog assertions now check in-memory options without shared file generation;
config Flow option/validation assertions are replaced by local config errors and
absence-of-resource checks. Ordinary artifact editor assertions now use document text
without session files. Project editor failures preserve the edited text for recovery.

`TestCLIProgramOrdinaryDocumentEditor` rehomes the old def/spec/PR program editor
and save-order assertions without agent runs or session files. Each document runs
success and follow-up type-save failure, checks the committed text survives, and
proves user writes never carry agent_edit=true.
`TestOrdinaryDocumentSaveRetainsCommittedTextAfterFollowUpFailure` also covers
refresh failures and underlying causes. This preserves ordinary partial-success
semantics while the transport/document ownership migration remains with P2/P7.

P1 review fixes 01 retain ordinary creation partial success in
`TestChangeCreateRetainsCommittedChangeAfterTypeFailure`: the committed change ID
survives type-update failure, details remain reachable, and repeating save cannot
create a duplicate. `TestEditorSubmissionPreservesSlashPrefixedData` distinguishes
interactive commands from literal editor data for testcase/project forms and title
edits, including `/cancel`. The existing manifest-selected
`TestCLIProgramEditorSaveAndFailure/slash-prefixed_testcase` exercises Ctrl+E,
exact saved scenario content, redraw and orderly navigation through the complete
program. No existing assertions were removed and no manifest expansion was needed.

### P1 review fixes 02 — artifact editor fidelity and unchanged exits

P1-06/P1-07 retain original document ownership across the editor handoff.
`TestArtifactEditorSeedsOriginalDocument` checks def/spec/PR temp-file bytes for
tabs and documents beyond the textarea line limit.
`TestArtifactEditorUnchangedExitSkipsPersistence` checks empty and nonempty
unchanged results, independent selected types, redraw-only commands, and cleared
edit state. No unchanged result is persisted or parsed for type updates.
`TestCLIProgramOrdinaryDocumentEditor` retains its save-order and partial-failure
assertions, adding real editor input capture, newline-only edits, unchanged exits,
redraw and temporary-file cleanup. It remains eligible through the existing
manifest entry. Current backend document-contract migration still belongs to P6;
these fixes change no HTTP route or payload contract and remove no assertions.

### P1 review fixes 03 — failed editor save recovery

P1-06/P1-07 now cover repeated failed saves without draft loss.
`TestArtifactDraftSurvivesFailedSaveAndRetry` exercises def/spec/PR with literal
tabs and more than 10,000 lines, failed persistence, raw Ctrl+E seeds, unchanged
reopens, another failed attempt, and exact Enter retry bytes.
`TestEditorRetryKeepsLiteralData` checks failed testcase create/update, project
create/update and title submissions with `/api/v1/health returns 200` and literal
`/cancel`. `TestEditorDraftEditingAndDiscard` checks editable short drafts,
protection of lossy previews, empty drafts, and explicit discard restoring command
semantics. Successful saves clear draft ownership through the normal prompt reset.

Existing manifest-selected program tests add first-save failure followed by Enter
and unchanged Ctrl+E retries for all three document fields, and literal testcase
retries including `/cancel`. They retain exact HTTP content, editor seed, redraw,
cleanup, save ordering and committed follow-up failure assertions. No tests or
manifest scenarios were removed; no backend contract or route was changed.

### P1 review fixes 04 — clipboard retries and config permissions

P1-06 `TestEditorDraftAsyncPasteRetry` delivers the real textarea asynchronous
paste result using owned clipboard executables. It verifies failed-save drafts
include pasted content and further typing on retry, slash-prefixed data stays
literal, and delayed paste preserves an unrepresentable raw draft.
P1-03 `TestAtomicConfigReplacementPreservesPermissions` checks 0600, 0640 and 0644
replacement, new-file defaults, and lookup failure without replacing the path.
P1-07 `TestCLIProgramStartupNavigationAndSelection` now verifies successful
project selection preserves a private config's 0600 permissions. Existing save
failure, in-memory selection and user-file preservation assertions remain.
No tests or manifest entries were removed; no backend API contract changed.

### P1 review fixes 05 — drain configuration saves before exit

P1-03 `TestQuitDrainsProjectSelectionSaves` holds asynchronous save completion
until after Escape, `/quit`, or Ctrl+C requests exit. It checks single and queued
saves, final project 9 persistence after project 8, and ignored repeated exit/editor
input during the drain. `TestQuitSaveFailureRemainsVisible` proves save errors
cancel automatic exit without dropping the latest queued save or in-memory
selection, and that a subsequent explicit exit remains possible.
P1-07 `TestCLIProgramStartupNavigationAndSelection` now sends selection and exit
in one input burst and checks the persisted project after full program shutdown.
These subtests remain eligible through the existing manifest entry. No HTTP
contract, tests or manifest entries were removed or changed.

### P1 review fixes 06 — require loaded documents before editing

P1-06 `TestDocumentEditorRequiresSuccessfulDetailLoad` holds the detail result,
checks all four retained editable text fields before and after failure, then
verifies exact loaded bytes and rejects readiness from a prior selection.
`TestArtifactEditorSeedsOriginalDocument` now includes successfully loaded empty
documents. P1-07 `TestCLIProgramOrdinaryDocumentEditor` adds delayed and failed
fake-server detail responses, `/edit-spec` while unavailable, recovery through
reselection, exact editor seeds and no writes on unchanged exit. Its existing
manifest entry includes these subtests. No assertions or scenarios were removed;
current HTTP transport migration remains assigned to its existing owning passes.

### P1 review fixes 07 — ordinary slash-prefixed form data

P1-06 `TestPromptSubmissionPreservesSlashPrefixedData` verifies ordinary Enter
submits exact slash-prefixed testcase/project/title data, including command-like
sentences and commands unavailable on that screen.
`TestPromptSubmissionDispatchesRecognizedFormCommands` retains exact `/cancel`
dispatch in forms and unknown-command errors on the main screen. P1-07's existing
`TestCLIProgramEditorSaveAndFailure` adds bracketed paste of a slash-prefixed
scenario and Enter retry after save failure, asserting both HTTP payloads.
Editor literal-data cases remain intact. No assertions or manifest scenarios
were removed, and no HTTP route or payload contract changed.


## P2 projects and typed transport (021)

[Specification](../../agent/specs/021-cli-projects-typed-transport.md). Backend
project handlers and domain DTOs were re-read: every operation is POST, list is
an unfiltered array, details use `/api/v1/project/details`, create returns only
`{id}` with 201, update/delete return empty 204, and config returns its slug and
six ordered string arrays. Project IDs/counts/last_ref and timestamps are typed;
selectors and date strings are constructed only in presentation. There is no
project configuration-assignment write. Transport follows no alternate route or
envelope, accepts caller context and injected clients, bounds HTTP lifetime to
15 seconds, and preserves status/causes including response-close failures.

Project validation, drafts, operation generations, sequencing and result handling
now belong to `internal/projects`; the root routes screens, prompts/editor I/O,
selection and injected capabilities. Mutations execute once and retain created
IDs or saved/deleted outcomes when subsequent reads fail. `/retry` reads only.
Selected-project catalogs are per-model, cleared on selection changes, and loaded
through project/config. Late same-project generations and different-project
results are ignored. No absent color or global/default config is manufactured.
Atomic asynchronous selection persistence, permissions, failure visibility and
quit draining remain covered by the retained P1 regressions.

| Criterion | Named unit assertions | Manifest-selected program/PTY assertions |
| --- | --- | --- |
| P2-01 | `TestP201ProjectRoutesShapesAndExactlyOneRequest` | `TestCLIProgramProjectCRUDAndPartialSuccess` exercises all six routes, name inputs and displayed detail/config values |
| P2-02 | `TestP202ContractErrorsPreserveStatusAndCause`, `TestP202TransportFailuresCancellationAndFiniteDeadline`, `TestP202EmptyListAndInvalidIDs`, `TestP202RedirectIsStatusErrorWithoutAlternateRequest` | `TestCLIProgramShutdownCancelsProjectHTTP`; adapter-only counters remain unit only |
| P2-03 | `TestP203ProjectActionsStatesAndAllDisplayedFields`, retained project forms/rendering/navigation tests | `TestCLIProgramProjectCRUDAndPartialSuccess`, `TestCLIProgramEditorSaveAndFailure`, `TestShellNavigationEditorAndScrolling` |
| P2-04 | `TestP204MutationSuccessFailurePartialSuccessAndReadOnlyRetry`, `TestP204DeleteSelectedProjectClearsScopedCatalogAndPersists`, retained literal editor/draft tests | `TestCLIProgramProjectCRUDAndPartialSuccess` success and refresh-failure cases; retained editor program scenarios |
| P2-05 | `TestP205PhaseStyleUsesOnlyProjectColors`, `TestP205OrderedProjectCatalogsWithoutFallback`, `TestP205SelectionCatalogScopeAndPersistenceFailure`, `TestP205ConfigReadRecoversSelectedProjectSelectors`, retained atomic-save/config-drain tests | `TestCLIProgramProjectSwitchWithPendingConfig`, `TestCLIProgramStartupNavigationAndSelection` |
| P2-06 | `TestP206DelayedResultsCannotOverwriteNewOperation`, `TestP206SelectionIdentityRejectsDelayedCatalogAndDetail`, `TestP206ConfigReadPreservesCatalogOnUnrelatedOrStaleResults`, `TestP206ShellReadNavigationCancelsObsoleteProjectResult`, `TestP206ProgramContextReachesProjectCommands` | `TestCLIProgramProjectSwitchWithPendingConfig`, `TestCLIProgramShutdownCancelsProjectHTTP` |
| P2-07 | `TestP207ProjectReadFailuresAndEmptyResults`, real architecture and manifest tooling checks | All preceding scenarios plus retained `TestCLIProgramOrdinaryDocumentEditor` and `TestCLIStartupWithoutFlowResources` |

Assertion migrations: the old adapter tests `TestHTTPClientListsProjectRows`,
`TestHTTPClientProjectCreateUpdateAndGetPayloads`, and
`TestHTTPClientProjectMutationValidationAndBackendErrors` are replaced by the
P2-01/P2-02 cases above. Their exact route/payload/status/field/error assertions
now cover current contracts instead of envelopes, string IDs, alternate timestamp
fields and entity-returning mutations. `TestHTTPClientPostsToSelectorEndpoints`
retains its epic assertion; project/config and presentation catalog assertions
replace global option endpoints. All program/PTY fakes now return complete current
project values and 201 IDs. Existing app tests use typed project values; the old
invalid-string ID case becomes a zero-ID validation case, with wrong-wire-type
rejection tested at the adapter. Timestamp display still tests a typed zero-time
fallback; malformed wire timestamps are contract errors. Project delete is now
advertised and tested. Update refresh failure now expects committed details,
retained name and explicit refresh-failed status instead of a failed-save form.
Editor retry tests execute the originally scheduled command sequence, preserving
literal bytes and exactly-once attempts rather than scheduling a second save.

Remaining legacy adapter ownership (no compatibility aliases were added):
`ListEpics` and permissive epic options → P3; `ListChangeRows`, `GetChange`,
`CreateChange`, `DeleteChange`, title/types/phase/open/epic/PR-URL updates → P4;
`CreateTestCase`, `UpdateTestCase`, `UpdateTestCaseDone`, `DeleteTestCase` → P5;
`UpdateChangeDef`, `UpdateChangeSpec`, `UpdateChangePR` → P6 document migration.
Their legacy map decoders remain only for those unported operations. Backend
config management and health are P7. No live backend/database, Flow resources,
agent process, dependency or backend source was required or changed in P2.

The downstream `TestPhaseStyleUsesOptionColorOrGreyFallback` assertion is migrated
to `TestP205PhaseStyleUsesOnlyProjectColors`: missing colors now retain terminal
styling instead of inventing a gray or built-in phase color. This is the only
change-rendering behavior changed by P2; the now-unused built-in catalog is removed.
Touched changes model/view/test files also receive required baseline formatting.

### P2 review fixes 02 — configuration ordering and list reload selection

P2-05/P2-06 `TestP206ManualConfigSupersedesPendingSelectionCatalog` delays both
startup and selection catalog results until after an explicit configuration read,
covering old successes and failures. Recovered phase/type selectors remain usable,
and the independent pending project-name read remains valid. Configuration has
its own generation, advanced by selection, deletion and accepted successful manual
results; feature operation generations still guard manual results. No HTTP contract
changes.

P2-03/P2-06 `TestP203ProjectReloadCannotSelectHiddenCachedRow` proves loading
clears cached rows and their selection, rejects Enter without canceling the list
request, and selects the refreshed project after completion. Manifest-selected
`TestCLIProgramProjectReloadBlocksCachedSelection` exercises return navigation,
Enter during a held HTTP list response, continued refresh and the resulting detail
request through the complete program. The manifest inventory assertion includes
the new scenario; no retained test or assertion was removed.

### P2 review fixes 03 — canceled manual configuration reads

P2-05/P2-06 `TestP206CanceledManualConfigPreservesPendingCatalog` keeps startup
and selection catalog responses pending while opening `/project-config`, then
returns to the project list and discards the late manual result. Both response
orderings (shared result before/after cancellation) preserve phase/type selectors.
The shared generation advances only when a successful manual result is accepted;
starting a screen-scoped read cannot invalidate shared loading. Existing manual
success supersession and unrelated/stale-result tests remain unchanged. Existing
manifest-selected program/PTY scenarios are retained; no new terminal coverage
claim is made for these unit-only permutations. No HTTP contract changed.

### P2 review fixes 04 — deletion with failed local selection clear

P2-04/P2-05 `TestP204DeleteSelectedProjectPreservesOutcomeOnConfigFailure` checks
local atomic-write failure after committed deletion with and without list-refresh
failure, including a queued clear after an earlier selection save. It preserves
deletion status, refresh diagnostics and `/retry` guidance, uses accurate cleared
selection wording, keeps in-memory selection empty, cancels automatic quit on save
failure, preserves local data and proves retries never repeat the delete.
Manifest-selected `TestCLIProgramProjectCRUDAndPartialSuccess` adds both failure
combinations using keyboard-driven deletion and a temporary config destination
that cannot be replaced. Visible diagnostics and exactly one delete plus the
follow-up list and read retry are asserted. Existing selection-save, queue/drain,
CRUD, program and PTY assertions remain; no HTTP contract or manifest changed.

## P3 epic management (022)

[Specification](../../agent/specs/022-cli-epic-management.md). Current backend
handlers and `domain/epic.go` confirm five POST operations: list(project_id),
details(id), create(project_id,name), update(id,name), delete(id). Create returns
201 with ID only; updates/deletes return empty 204. The required response fields
are id, project_id, name, done_tc, total_tc, completed, change_count, created_at,
updated_at. Counts/completion are numeric, timestamps are typed and none are
nullable. Server completion is displayed verbatim, including values inconsistent
with locally calculated ratios. Empty arrays succeed; missing/null/wrong-shaped
values fail with retained HTTP status and contract causes.

`internal/epics` owns validation, raw name drafts, scoped generations, mutation/read
sequencing, no-op detection and results. The shell composes the screens and shared
editor/prompt. A committed ID/name or deletion survives refresh failure; `/retry`
only reads. Obsolete requests cancel and their results cannot overwrite a new
screen, project, draft or error. Change association/filter selectors use the same
typed list, with display options converted in the epic presentation boundary.

| Criterion | Named unit assertions | Manifest-selected program/PTY assertions |
| --- | --- | --- |
| P3-01 | `TestP301EpicRoutesShapesAndExactlyOneOperation`, `TestP301EpicRequiredFieldsAndMalformedResponses`, `TestP301EpicFailuresStatusesCancellationAndInvalidIDs` | `TestCLIProgramEpicCRUDAndPartialSuccess` uses all five current routes; adapter-only counters stay unit-only |
| P3-02 | `TestP302EpicOperationsValidationFormsAndPresentation`, `TestP302EpicKeyboardCRUDHelpConfirmationAndScope`, `TestP302EpicActionsRequireRealSelection` | `TestCLIProgramEpicCRUDAndPartialSuccess` exercises help, forms, all displayed fields, validation and failures; `TestShellNavigationEditorAndScrolling` renders server completion and edits an epic |
| P3-03 | `TestP303EpicMutationsExactlyOnceFailureRecoveryAndNoOp`, `TestP303EpicEditorRawDraftRetryCancelAndNoOp`, confirmation assertions in `TestP302EpicKeyboardCRUDHelpConfirmationAndScope` | `TestCLIProgramEpicCRUDAndPartialSuccess` verifies exact editor bytes, failed writes and retries, unchanged editor exit, cancellation, retained committed outcomes, unrelated rows and exactly-once committed writes |
| P3-04 | `TestP304EpicIdentityCancellationAndHiddenRows`, `TestP304EpicObsoleteReadCannotChangeShellOrDraft`, `TestP304EpicSelectorReopenRejectsOlderSameProjectResult`; retained P1 save-drain and P2 catalog-generation regressions | `TestCLIProgramEpicDelayedScopeAndShutdown` covers leaving detail, project switching and program shutdown with held HTTP work; `TestCLIProgramEpicReloadBlocksCachedRows` rejects Enter during held list refresh; retained startup selection/save-drain programs |
| P3-05 | `TestP305EpicSelectorUsesTypedValuesAndObsoleteScope`, retained `TestFilterSelectorsReturnToChangesList` and current-contract `TestHTTPClientPostsToSelectorEndpoints` | `TestCLIProgramEpicCRUDAndPartialSuccess` returns to changes and selects the remaining epic filter; architecture checks prohibit sibling feature imports |
| P3-06 | All named P3 unit assertions above; `CoverageTest.test_scenario_manifest_rejects_scripts_empty_and_unmatched` audits the explicit campaign selection | All three new epic program tests are manifest-selected; existing `TestShellNavigationEditorAndScrolling` adds epic list/detail, completion, edit/editor redraw, delete cancellation and creation with the real PTY child |

Intentional assertion migration: the old selector endpoint test keeps its exact
POST/path/payload assertions but now expects the complete current DTO array,
rather than an `epics` envelope with `title`. Invalid selector project IDs now
use typed zero/negative values; wrong wire types have explicit transport tests.
The retained filter test uses numeric project/epic fixtures instead of synthetic
`project-1`/`epic-1` IDs, preserving its selection/clear assertions. Placeholder
unavailable-action assertions now prove no fake selection or save without a
project, and advertise real creation. Removed permissive option decoders had no
remaining caller; unported change/testcase decoders remain owned by P4/P5/P6.
P1 exact editor bytes, config-save queue draining, P2 catalog recovery ordering,
manual-read cancellation and deletion/config-write diagnostics remain intact.
No dependency, backend, database, Flow resource or live service was used or changed.

P3 self-review also closes the shared selector generation gap: every filter and
detail selector opening gets its own operation identity. A canceled epic read
cannot replace reopened options (including a new change entity), clear loading
or install a stale error. The named selector-reopen regression failed before the
fix and passes for late success and failure on both selector paths.

### P3 review fixes 01 — viewport, literal names and refresh attribution

P3-02 `TestP302EpicViewportKeepsSelectionVisible` covers every selection in 40
rows at five viewport heights, including a single visible row and no space.
`TestP302EpicListFitsTerminalWhileMovingAndResizing` drives arrows and Enter,
checking the complete shell fits resized terminals with error/footer wrapping.
The feature receives available height and centers the viewport on selection.

P3-03 `TestP303LoadedEpicNamesRemainLiteralOnEnter` checks loaded `/save`,
`/cancel`, `/editor` and `/return` names: unchanged Enter is a no-op, and prompt
edits save exact literal bytes. `TestP303EpicOutcomeBelongsOnlyToPendingRefresh`
covers create/edit/delete with successful and failed refreshes, read-only retries,
recovery, navigation and unrelated entity failures without repeated writes.
Only a failed committed refresh retains an outcome, bound to its read operation
and entity; recovery or unrelated work clears it.

The existing manifest-selected `TestCLIProgramEpicCRUDAndPartialSuccess` adds a
literal `/save` editor-name case, unchanged Enter after reopening edits, and
unrelated detail failure/recovery after deletion in all four modes. Existing
request-count, exact-byte, cancellation and partial-success assertions remain.
The unchanged real PTY scenario also passes; viewport permutations are unit
assertions, not additional PTY coverage claims. No HTTP contract changed.

### P3 review fixes 02 — bounded details and linear truncation

P3-02 `TestP302EpicDetailsFitsTerminalAndScrollsEveryField` checks the measured
shell plus details at 80×24, 60×16 and 120×40, all 40 name lines and returned
metadata, arrow/page navigation and prompt focus. Feature test
`TestDetailsViewportScrollResizeAndReset` covers clamping, empty/loading output,
zero space, resize and offset reset after selection or accepted detail/mutation
results. `TestCLIProgramEpicCRUDAndPartialSuccess` now pages through a multiline
epic and back to its IDs using actual keyboard input and the fake backend.
It remains selected by the existing terminal manifest; all former CRUD assertions
remain intact.

`TestTruncateBlockTerminalCells` covers ASCII, wide glyphs, combining marks,
emoji graphemes, ANSI resets, empty lines and default width. The shared helper
uses the already-pinned ANSI library's linear scan, without dependency changes.
`TestLongEpicRenderingRemainsResponsive` checks 50,000-character names through
both table and details with a broad two-second regression ceiling;
`BenchmarkEpicLongNames` measures 5k/50k/100k inputs separately for both paths.
No backend DTO, form/editor byte handling or mutation contract changes.

### P3 review fixes 03 — empty find read recovery

P3-04 `TestP304EpicEmptyFindRestartsCanceledRead` covers list and detail reads,
empty and whitespace-only input, retained find validation feedback, fresh read
results and rejection of stale results before and after recovery. It failed on
the original implementation because empty submission returned no read command.
`submitFindValue` now returns through `arrive`, resuming normal read sequencing.
The manifest-selected `TestCLIProgramEpicDelayedScopeAndShutdown` adds empty
find list/detail scenarios with pending HTTP requests, cancellation, successful
keyboard-driven recovery and orderly exit. No API contract or existing assertion
changed; the same complete-program driver and fake backend are reused.

## P4 change management (023)

Review fixes 03 retain P4-03/05 field-update contracts:
`TestP403ClearingUpdatePromptPreservesField` drives Ctrl+C, replacement typing and
Enter for PR URL and prerequisite fields, asserts the original specification is
preserved with no document insertion, and checks Escape/second-Ctrl+C cancellation.
The existing `TestCLIProgramChangeCRUDAndPartialSuccess` now clears those two
prompts through keyboard input before saving; exact update payloads, normal and
failed-refresh outcomes, and rejection of unexpected document routes remain
asserted. The prompt hint distinguishes clearing input from canceling the editor.

Review fixes 02 retain P4-03/04/05 contracts:
`TestP403EditorReturnsToCreateFormBeforeFirstSave` verifies valid Markdown returns
from the initial editor without writing, title override, UUID supplied/omitted,
unchanged editor reopening and exact brief bytes through explicit confirmation.
The canceled-subfield regression now confirms before asserting creation.
`TestShortDetailsViewportScrollsIdentityAndBody` covers three-, four- and five-line
panes, every identity/body field, reverse paging and selection/copy row identity.
`TestP404ShortDetailShellKeepsAllFieldsReachable` drives PageUp/PageDown and resize
with three/four-line detail space and wrapped error feedback. Fixed rows become
scrollable only when they would consume all body space.
The existing manifest-selected `TestCLIProgramChangeCRUDAndPartialSuccess` now
starts with a valid Markdown brief in the normal case and retains a plain brief
in the partial-success case. It asserts zero creates after editor completion,
then changes title/UUID and verifies the exact single create payload. Existing
CRUD, retry and raw-byte assertions remain; the real PTY scenario is unchanged.

Review fixes 01 retain P4-03/04/05 contracts:
`TestP403CanceledCreateSubfieldDoesNotConsumeNextBrief` covers title/UUID cancel
via Esc and `/cancel`, reopening creation, and exact new brief submission.
`TestP404SavedLongTitleKeepsDetailViewportUsable` covers saved 3,300-character
titles with successful or failed refresh, terminal height and paging to later rows.
`TestP405LongSavedValuesKeepBoundedFeedbackAndFullDetail` covers long title,
PR URL, phase and type feedback, escaped controls, full retained values and failed
read-only retries. These are unit regressions; the existing complete-program and
real PTY campaign remains unchanged and passed independently.

[Specification](../../agent/specs/023-cli-change-management.md). Change transport
now uses all eleven current POST routes, one cancellable operation per method.
List/details decode required typed fields directly; create returns only its 201
ID; writes and deletion accept empty 204 responses. Nullable ref/slug/epic and
prerequisite distinguish explicit null from absent fields. Counts/completion are
int64, timestamps are time.Time, and server identity/completion are never rebuilt
from document titles or testcase ratios. `Change` is the wire/business value;
`ChangeView` is a separate screen projection of change, document and testcase reads.

`changes` owns operation state, validation, create drafts, ordered effect sequencing,
revision/project/entity matching and committed outcomes. The shell routes keys,
shared prompt/editor I/O, catalogs and feature results. Every committed create,
field save, document insertion and optional metadata type update is recorded
before dependent reads/writes. Read-only `/retry` cannot replay those writes.
Failed editor saves keep the exact literal bytes; loaded slash-shaped titles and
unchanged editor output stay data/no-ops. A committed outcome is retained only for
its associated failed refresh; recovery or unrelated navigation removes it.

Dependency-driven P6 support is intentionally included: `documents.Access` validates
selected-project configured types and exposes ordinary current/insert capabilities.
Change details use `/doc/current`'s array and separate `/test-case/list`; `/brief`,
`/edit-spec` and `/document` append document versions with human provenance.
Configured custom document types are also selectable. Generic owner navigation,
history/details and full document management remain P6. Testcase mutation contracts
and feature ownership remain P5; their legacy adapter tests are explicitly retained
and are not evidence of current backend mutation compatibility.

| Criterion | Meaningful named unit coverage | Complete-program / PTY evidence |
| --- | --- | --- |
| P4-01 eleven typed operations, exact payload/status, one request, malformed data, cancellation | `TestP401AllChangeOperationsExactTypedPayloadsAndOneRequest`, `TestP401MalformedChangeFieldsAndStatusCauses`, `TestP401ChangeCancellationAndUUIDOmission`, `TestP401InvalidIDsNeverRequest`; retained P2 transport deadline/status/cause/redirect tests | `TestCLIProgramChangeCRUDAndPartialSuccess`, `TestCLIProgramChangeMalformedReadRecovery`; adapter counters remain unit-only |
| P4-02 every returned field, nullable identity/associations, int64 completion, separate presentation | `TestP402EveryReturnedFieldAndLiteralNoOp`, `TestP401MalformedChangeFieldsAndStatusCauses`, `TestP404ChangeViewportsFitAndExposeEveryField`, retained change table/detail assertions | `TestCLIProgramChangeCRUDAndPartialSuccess`, `TestShellNavigationEditorAndScrolling` show actual server completion and nullable fields |
| P4-03 list/details/create/delete and seven field updates, help, forms, explicit title/brief/UUID, catalogs, clears/false/empty, HTTP(S) URL | `TestP403EachChangeActionAndValidation`, `TestP403InvalidFormsAndAbsentCatalogsNeverWrite`, `TestP403ExplicitCreateFieldsRetainRawBrief`, `TestP403NullableAssociationForm`; retained phase/epic/type/open/title keyboard tests and exact command lists | `TestCLIProgramChangeCRUDAndPartialSuccess` drives all eleven routes, ordered catalogs, null clears, false open, empty type set, UUID and plain brief creation; retained document editor program |
| P4-04 ownership, viewports, literal values, scoped outcomes, empty find, stale/project/entity/revision identity and shutdown | `TestP404ChangeEmptyFindAndObsoleteResults`, `TestP404ChangeReadCannotOverwriteDraftOrSelectedProject`, `TestP404StaleResultsCanceledWorkAndInvisibleRows`, `TestP404ChangeViewportsFitAndExposeEveryField`, `TestP404ChangeLongRenderingAndLiteralTitleNoOp`; package boundary checks | `TestCLIProgramChangeDelayedScopeAndShutdown`, `TestCLIProgramChangeReloadBlocksCachedRows`, CRUD recovery program and real PTY |
| P4-05 independently committed steps, failed later writes/reads, read-only retry, busy duplicate prevention, raw/literal editor bytes and unchanged no-op | `TestP405CommittedStepsSurviveLaterFailureAndRetryOnlyReads`, `TestP405CancellationAfterCommitRetainsStepAndLiteralValue`; retained `TestOrdinaryDocumentSaveRetainsCommittedTextAfterFollowUpFailure`, `TestChangeCreateRetainsCommittedChangeAfterTypeFailure`, `TestArtifactDraftSurvivesFailedSaveAndRetry`, `TestEditorRetryKeepsLiteralData`, P1 configuration drain and P2/P3 partial-success regressions | `TestCLIProgramChangeCRUDAndPartialSuccess`, `TestCLIProgramOrdinaryDocumentEditor`, retained project/epic/save-drain programs |
| P4-06 current/insert document DTOs, configured ordinary editors, separate testcase read, no successful insert replay | `TestP406DocumentCurrentInsertAndSeparateTestCases`, `TestP406MalformedDocumentAndTestcaseReads`, `TestP406ConfiguredDocumentAccessAndExactBytes`, `TestP406ConfiguredDocumentSelectorAndNoCatalog`; retained editor exact bytes/tabs/long-document and metadata tests | `TestCLIProgramOrdinaryDocumentEditor` now asserts current document reads and exact insert payloads/201 IDs; follow-up type failures never repeat insertion |
| P4-07 keyboard effects/results, malformed response, lifecycle, test reuse, manifests | All preceding P4 tests, existing manifest inventory and architecture tooling | Four new manifest-selected change programs plus the extended `TestShellNavigationEditorAndScrolling` use fake servers/owned processes only |

Field mapping: the P4-01 route test asserts IDs/project IDs, UUID omission/value,
phase/types/title/open/PR URL, nullable epic/prerequisite and create brief payloads.
The P4-02 test verifies returned ref/slug/epic identity and completion separately
from derived values; malformed-field cases cover **each** list/detail wire field,
including required nullable presence, timestamp types and wide counts. The viewport
test traverses multiline titles and all identity/count/timestamp/relationship fields.

Assertion migration: `TestHTTPClientChangeListCreateUpdateAndGetPayloads` is replaced
by P4-01/P4-02/P4-06 exact-route/typed-wire tests. Obsolete change document fields,
agent-edit state, guessed envelopes and entity-returning mutations are replaced by
current document DTO assertions; `TestRetainedTestCaseMutationPayloads` preserves
its four still-unmigrated testcase request assertions. Brief replaces def in active
DTOs/screens/parsers/tests. Creation can infer a Markdown heading as a convenience,
but explicit title and plain brief need no Markdown structure. Document edits no
longer rename a change from a heading; explicit title action tests preserve rename
coverage. Unchanged document editing now performs no type rewrite or detail read.
Backend reference values display without zero-padding an obsolete identifier.
Loaded details reject stale old success after a failed generation: the editor-read
regression now schedules a fresh `/retry`. List/PTY row counts reflect measured
shell height (seven visible rows at 100x20); scrolling and selection assertions
remain. Four existing format files/package-comment baselines remain separately
owned by P5/P10. No backend, database, dependency, Flow resource or live agent changed.
