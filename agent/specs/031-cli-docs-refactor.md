# Spec: CLI Docs Refactor

## Goal

Align the CLI with backend change 030 and replace the change details document
presentation with a configured document list, a brief preview, and editable
comments. Add document history browsing and access to inactive changes. Exclude
inactive epics from the ChangeDetailsScreen Epic selection list.
Build the new document presentation on changes first; use it as the template for
project and epic document presentation in later changes.

**Status: ready for review.** All clarification questions are resolved.

## Scope

- In scope: CLI transport and DTO alignment with the changed and added HTTP
  contracts in backend commit `a34809f81647140ed0ae67289d667ec0233cc011`;
  change details Brief / Docs / Comments presentation; immediate document and
  comment deletion; comment creation and editing; document history using `bat`;
  inactive-change list access; exclusion of inactive epics from the
  ChangeDetailsScreen Epic selection list; migration of affected tests and
  documentation.
- Out of scope: the new document layout on project and epic details screens;
  inactive epic browsing or reactivation; unrelated redesign, new agent workflow
  phases, frontend changes, database
  access from the CLI, Git publication, deployment, and Codex/Konsole shortcuts.
  The user's follow-ups explicitly authorize the backend comment-undelete and
  active-set endpoints described below. No additional backend or database changes
  are required by the inactive epic requirement.

## Reference contract

The brief is [031-cli-docs-refactor.md](../briefs/031-cli-docs-refactor.md), with
the user's subsequent clarifications recorded in this specification.
Use [CLI architecture](../../docs/cli-architecture.md),
[CLI rebuild plan](../cli-rebuild-plan.md), and
[repository instructions](../../AGENTS.md). Current backend handlers and domain
DTOs are the HTTP authority. Historical specifications and architecture text
that still name removed routes are migration references, not API contracts.

Contract sources:

- [Document routes](../../backend/internal/doc/api.go),
  [document DTOs](../../backend/internal/domain/doc.go), and
  [document service](../../backend/internal/doc/service.go).
- [Project routes](../../backend/internal/project/api.go) and
  [project DTOs](../../backend/internal/domain/project.go).
- [Epic routes](../../backend/internal/epic/api.go) and
  [epic DTOs](../../backend/internal/domain/epic.go).
- [Change routes](../../backend/internal/change/api.go) and
  [change DTOs](../../backend/internal/domain/change.go).
- [Configuration routes](../../backend/internal/config/api.go) and
  [configuration requests](../../backend/internal/domain/config.go).

All operations below are `POST /api/v1/<operation>`. Owner requests contain
`ref_table` (`project`, `epic`, or `change`) and positive `ref_id`. ID requests
contain a positive `id`.

| Operation | Request | Successful response / semantics |
| --- | --- | --- |
| `doc/list` | Owner | 200 document array, descending ID, including retained and deleted history. |
| `doc/list-active` | Owner | 200 selected active documents, descending ID. Replaces removed `doc/current`. |
| `doc/active-set` | ID | 204; resolve stored owner/type and call `sp_doc_active_set` to select the same historical non-comment ID, restoring deletion metadata as the procedure defines; missing ID 404, comment ID 400. |
| `doc/details` | ID | 200 document, including historical or deleted documents. |
| `doc/insert` | Owner, `doc_type`, `body`, explicit `agent_edit` | 201 `{id}`; appends and selects a configured non-comment document. |
| `doc/comment-list` | Owner | 200 comment array, descending ID, including deleted comments. |
| `doc/comment-insert` | Owner, `body`, explicit `agent_edit` | 201 `{id}`; adds an independent comment without active-document selection. |
| `doc/comment-update` | ID, explicit `body` | 204; edits the same comment, allowing an empty string. Does not undelete it. |
| `doc/comment-undelete` | ID | 204; clears comment deletion metadata without changing ID, owner, content, provenance, or creation/edit timestamps. An already live comment succeeds; missing/non-comment IDs return 404. |
| `doc/delete` | ID | 204; soft-deletes the document or comment and removes any active selection for that ID. |
| `change/list-inactive` | `project_id` | 200 inactive change list items; no `active` field in individual list rows. |
| `change/update-active` | ID, explicit boolean `active` | 204; replaces removed `change/update-open`; false is valid. |

The document JSON fields are `id`, `ref_table`, `ref_id`, `doc_type`, `body`,
`html`, `agent_edit`, `created_at`, `updated_at`, and nullable `deleted_at`.
There is no `current` field. Active selection comes from `doc/list-active`, not
from choosing the greatest historical ID or merely checking `deleted_at`.
Configuration catalogs retain the six existing arrays and explicit empty-array
semantics. Non-comment inserts require a configured type for the correct owner;
`comment` is excluded from ordinary insertion and comment insertion is independent
of those catalogs. The backend trims inserted bodies; comment updates preserve
the submitted body, including an explicit empty string.

Project responses replace `config` with `config_slug` and add `active`. Project
lists include both active and inactive projects. Epic responses add `active`.
Change details replace `open` with `active`; change list items contain neither.
Project and epic deletion can return 204 after deactivating a referenced record
instead of physically deleting it. There are no project/config inactive-list
routes or project/epic reactivation routes in the inspected backend.

## Requirements

Every numbered requirement below needs meaningful unit coverage. Names after
each requirement are proposed acceptance tests to add or adapt, not tests claimed
to exist or pass. Program tests and PTY tests supplement those unit tests.

- **031-01 — Transport migration.** Decode document rows using the current
  fields, including explicit null and non-null `deleted_at`, and replace every
  use of `doc/current` with `doc/list-active`. Reject malformed required fields,
  wrong requested document identity/owner, null arrays, and invalid response
  types while accepting valid empty arrays and backend history ordering. Do not
  require or manufacture a wire `current` flag. Unit:
  `Test031DocumentWireContract`, `Test031DocumentReadRoutesAndOwnerValidation`.
- **031-02 — Comment and delete transport.** Provide the current comment list,
  insert, update, and shared delete operations with exact owner/ID payloads and
  status handling. Human comment inserts send `agent_edit: false`; updates send
  the same comment ID and an explicit body, including `""`, without an insert.
  Validate positive IDs and owner tables before sending requests. Unit:
  `Test031CommentRoutesPayloadsAndStatuses`, `Test031DeleteDocumentContract`.
- **031-03 — Other changed backend contracts.** Decode project `config_slug` and
  `active`, epic `active`, and change-detail `active`. Accept change list items
  without `open` or `active`. Send `change/update-active` with an explicit true
  or false value; remove requests to `change/update-open`. Rename the change
  command from `/open` to `/active` and its displayed label from `Open` to
  `Active`, including menus, help text, and documentation. Audit the actual 030
  endpoint/DTO diff and migrate each affected production caller and fixture,
  retaining unchanged configuration, testcase, and health behavior. Unit:
  `Test031ProjectEpicChangeWireContracts`, `Test031UpdateActiveRouteAndFalse`,
  `Test031ActiveCommandLabelAndHelp`,
  plus the retained configuration, testcase, and health contract tests.
- **031-04 — Load configured types.** Loading or switching the selected project
  loads its configuration and retains the ordered `project_docs`, `epic_docs`,
  and `change_docs` catalogs. Change document slots use only `change_docs` from
  that project. An empty catalog creates no invented slots; configuration
  failures are visible and do not enable writes using an old project's catalog.
  Existing project and epic document actions retain their own catalogs. Unit:
  `Test031OwnerCatalogsAndProjectSwitch`, adapting
  `TestP602OwnerCatalogsAndEmptyReadAccess`.
- **031-05 — Active documents are unique by type.** The change details screen
  displays at most one active non-comment document for each configured type.
  Read selection through `doc/list-active`; history and comments do not become
  active by inference. Duplicate active types or mismatched owners produce a
  visible contract error instead of selecting an arbitrary row. Unit:
  `Test031ActiveDocumentSelectionAndDuplicateTypes`.
- **031-06 — Brief preview.** Keep the existing change-details `Brief` preview
  and editing interaction: at most 16 rendered lines, with the existing
  truncation behavior (15 content lines plus `...` when longer); Return opens
  edit mode with the full brief body. The Docs list excludes `brief`. Unit:
  `Test031BriefPreviewAndFullEditorSeed`, adapting the existing detail preview
  and editor tests.
- **031-07 — Docs rows.** Under `Docs`, display each configured change document
  type except `brief` and `comment` as a selectable row in catalog order.
  An existing active document shows `<type> [✓] YYYY-MM-DD HH:mm`, with a green
  check; a missing document shows `<type> [ ]` without a fabricated date.
  Replace standalone full-body Spec/PR/other-document previews with these slots;
  retain the independent PR URL field. Use that document's `updated_at`, displayed
  in local time as `yyyy-mm-dd hh:mm`.
  Unit: `Test031ConfiguredDocumentSlotsAndCheckColor`.
- **031-08 — Edit or create a non-comment document.** Return on a Docs slot
  opens edit mode seeded with the active document's entire body, or an empty
  body when the slot is missing. Saving uses one `doc/insert` with the configured
  type and `agent_edit: false`, selecting the newly appended version. Preserve
  existing editor cancellation, unchanged-content no-ops, nonblank body
  validation, and terminal restoration. Unit:
  `Test031DocumentSlotCreateEditCancelAndNoOp`.
- **031-09 — Immediate document deletion.** Delete on an existing Docs slot
  sends exactly one `doc/delete` for its active ID immediately, without a
  confirmation dialog, then refreshes the active selection. The slot becomes
  empty; the CLI must not select an older historical version automatically.
  Delete on an empty slot sends no request and changes nothing. Unit:
  `Test031ImmediateDocumentDeleteAndMissingSlotNoOp`.
- **031-10 — Comments presentation.** The change details screen uses the label
  `Comments` and displays multiple independent comment entries. Each preview
  has at most three body lines without wrapping long lines. For more than two
  body lines, show the date on line four; shorter previews show it after their
  body without padding to line four. Show each comment's `updated_at` in local
  time as `yyyy-mm-dd hh:mm`. Hide soft-deleted comments from the normal section.
  Preview truncation never changes the stored or editor body. Unit:
  `Test031CommentPreviewNoWrapAndFullBody`, `Test031CommentTimestampLayout`.
- **031-11 — New comment.** `/new-comment` is the first change-details menu
  item. It opens comment editing and creates exactly one independent comment
  through `doc/comment-insert`; it never uses `doc/insert`, changes an active
  document, or requires `comment` in `change_docs`. Blank creation is rejected;
  cancellation sends no write. Unit:
  `Test031NewCommentFirstCommandAndIndependentInsert`.
- **031-12 — Comment editing and deletion.** Return on any selected comment
  opens edit mode seeded with its entire body, including text beyond the
  preview. Save uses `doc/comment-update` on that same ID; clearing a comment
  saves an explicit empty body. Delete sends one immediate `doc/delete` for
  that ID without confirmation. Other comments retain their identities and
  bodies. Remove deleted comments from the normal section, retaining them for
  history browsing and undelete. Unit:
  `Test031CommentEditEmptyBodyAndImmediateDelete`.
- **031-13 — History selection and navigation.** Ctrl+H on a particular
  non-comment document opens that type's history for the same change. Use
  `doc/list` and filter by owner and selected type; newest ID is shown first,
  even when another version is active. Right shows the next older version;
  Left shows the next newer version. Neither arrow wraps past the oldest or
  newest boundary. The history source includes soft-deleted versions. For each
  displayed document or comment, show all three timestamp fields with their
  labels: `created_at`, `updated_at`, and `deleted_at`. Format each non-null value
  in local time as `yyyy-mm-dd hh:mm`; show a null `deleted_at` as absent rather
  than inventing a date. Unit:
  `Test031DocumentHistoryTypeScopeOrderAndArrowBounds`,
  `Test031HistoryShowsCreatedUpdatedAndDeletedTimestamps`.
- **031-14 — History printing.** Print the selected historical document using
  Linux `bat -pp --color=always <file>`, with its full body in an operation-owned
  Markdown file, capturing its colored output inside the CLI history viewport.
  **Don't lose `bat -pp` colors.** Preserve the syntax-color ANSI sequences from
  `bat` through rendering, scrolling, clipping, resize and version changes;
  measure visible width without counting ANSI bytes. Force colors when stdout
  is piped; do not strip or replace `bat` colors with a uniform text style.
  Changing the selection prints the selected version, preserving navigation.
  Invoke the executable with separate arguments rather than interpolating body
  text into a shell command. A missing `bat`, failed process, or file failure is
  visible and retains the selected history record. Restore the terminal and
  clean up owned resources on return, failure, cancellation, and shutdown.
  Esc or Ctrl+C exits history and returns to the originating details screen
  with selection restored. Up/Down and PgUp/PgDown scroll the history viewport;
  Left/Right select versions. Ctrl+H enters history rather than acting as an
  active/inactive toggle within the viewer. Unit:
  `Test031HistoryBatArgumentsAndExactBody`,
  `Test031HistoryBatFailureCancellationAndCleanup`,
  `Test031HistoryPreservesBatANSIColorsAndVisibleWidth`,
  `Test031HistoryEscapeCtrlCAndSelectionRestoration`.
- **031-15 — History activation and comment restoration.** Space makes the
  displayed historical non-comment document active; for a deleted comment,
  Space undeletes that same comment. This action must retain the selected
  record's identity and must not simulate restoration by inserting a new copy.
  Comment restoration uses `doc/comment-undelete` with that same comment ID.
  Non-comment activation uses `doc/active-set` with that same historical ID.
  The footer identifies Space's active-selection or undelete action while
  browsing history. After success refresh through the backend without inserting
  a copy; no-op Space on an already active document/live comment changes no body.
  `Test031HistoricalDocumentActivation`, `Test031DeletedCommentRestoration`.
- **031-16 — Inactive changes.** Ctrl+H opens the inactive list from the change
  list, using `change/list-inactive` and the selected
  `project_id`. Preserve ordinary filters, keyboard selection, visible loading,
  empty/error states, and stale-project isolation. Space on an inactive change
  makes it active through `change/update-active` with `active: true`; show that
  action in the footer while browsing inactive changes and refresh after success.
  Esc/Ctrl+C returns to the originating list.
  Unit:
  `Test031InactiveListRoutesKeyboardAndScope`,
  `Test031InactiveChangeSpaceActivationAndFooter`.
- **031-17 — Entity deletion outcomes.** A 204 project/epic deletion can mean
  physical removal or deactivation. Refresh using actual backend responses;
  do not reject a retained inactive record as malformed or claim it was
  physically removed without evidence. Respect the project list's returned
  ordering and active state. Preserve committed-write/read-failure feedback.
  Unit: `Test031ProjectEpicDeleteDeactivationOutcomes`.
- **031-18 — Existing document and brief capabilities.** Keep `/documents`
  available and migrate its current-document reads and history presentation to
  the new contracts. Preserve existing project/epic document access without
  applying the new change-details layout. Migrate ordinary brief editing and
  brief clarification to active-document reads; retain exact input/drafts,
  provenance, version invalidation, and no duplicate write on read-only retry.
  Unit: `Test031RetainedOwnerHistoryAndBriefWorkflow`, adapting
  `TestP602ProjectEpicChangeNavigationAndScope` and the existing agent tests.
- **031-19 — Failure, concurrency, and focus behavior.** Reads and writes retain
  bounded context-aware HTTP handling. Failed writes preserve useful drafts and
  errors. Committed mutations followed by failed reads remain visibly committed;
  retries refresh reads without replaying inserts, updates, deletes, or Space
  activation. Busy input cannot duplicate a write. Late HTTP/editor/process
  results cannot update another project, change, selected document, or revision.
  History/Delete/Space keys act on selected rows only when a text editor or menu
  does not own the key; ordinary prompt Backspace/Delete remain text editing.
  Unit: `Test031MutationPartialSuccessAndReadOnlyRetry`,
  `Test031LateResultIsolationAndBusyDeduplication`,
  `Test031DocumentHistoryAndDeleteKeyFocus`.
- **031-20 — Terminal acceptance.** Exercise configured slots, brief preview,
  comment creation/edit/delete, immediate document delete, history order and
  arrow boundaries, `bat` execution/failure, and inactive change activation
  through keyboard-driven complete-program tests with fake HTTP collaborators
  and owned local processes. Extend the real PTY scenario to prove Ctrl+H,
  Left/Right/Space/Delete, footer feedback, redraw and restoration at the terminal
  boundary. Verify `bat` colors on highlighted Markdown in the real PTY, including
  captured stdout rendered inside the app, scrolling and version changes.
  Include new program scenarios in the explicit terminal manifest; keep direct
  adapter and harness tests outside terminal counters. Unit:
  `Test031HistoryNavigationModelAndFooter`,
  `Test031ScenarioManifestIncludesDocsAndInactiveLists`.
- **031-21 — Backend comment undelete.** Provide
  `POST /api/v1/doc/comment-undelete` accepting a positive `id`, returning empty
  204 for deleted or already live comments, 400 for malformed/invalid IDs, and
  404 for missing IDs or non-comment documents. Clear only `deleted_by` and
  `deleted_at`; preserve identity, owner, body, HTML derivation, provenance,
  `created_at`, and `updated_at`. Never select a comment as an active document.
  Use one parameterized repository mutation and retain error causes and request
  cancellation. Unit: `TestCommentAndDeleteAPIContracts`,
  `TestCommentUndeleteAPIRequiresPositiveID`,
  `TestCommentUndeleteValidationAndRepositoryErrors`, and
  `TestCommentUndeleteRepositoryRestrictsTypeAndPreservesContent`. APIHydra:
  owned project/epic/change comment restoration and repeated requests in
  `backend/apih-tests/doc/02-main.yaml`, with endpoint-only verification/cleanup.
- **031-22 — Backend historical-document selection.** Provide
  `POST /api/v1/doc/active-set` with positive document `id`, returning empty204.
  Resolve the stored document's own reference/type before calling the existing
  `sp_doc_active_set(ref_table, ref_id, doc_type, doc_id)` procedure with explicit
  SQL parameter casts. A historical or soft-deleted non-comment becomes active
  with the same ID, owner, body, provenance and creation time; procedure-defined
  restoration updates the deletion/edit metadata. Repeated selection succeeds.
  Do not create a new version or change the SQL schema. Reject comment IDs400,
  missing IDs404 and malformed/nonpositive IDs400; retain errors and context.
  Unit: `TestActiveSetAPIContract`,
  `TestActiveSetUsesStoredOwnerAndRejectsComments`,
  `TestActiveSetRepositoryCallsExistingProcedure`. APIHydra: select older and
  deleted versions for all three owners, verify selected IDs and preserved
  fields through independent reads, restore the expected selection for cleanup,
  and verify comment rejection and bad input/method behavior.
- **031-23 — Comment history access.** Ctrl+H from the Comments section opens
  the owner's retained comments, including deleted records, newest ID first;
  Left/Right move newer/older with the same boundaries as document history.
  History remains accessible when the normal section has no live comments.
  Comments remain independent mutable records: do not invent historical body
  revisions that the backend does not store. Unit:
  `Test031DeletedCommentsHistoryAndEmptySectionAccess`.
- **031-24 — Local timestamps.** Display all CLI dates/times in the user's local
  timezone, including document/comment rows, history metadata and retained entity
  screens, consistently formatted as `yyyy-mm-dd hh:mm` (24-hour time, no seconds
  or timezone suffix; Go layout `2006-01-02 15:04`). On ChangeDetailsScreen,
  `updated_at` is the only displayed timestamp field, including the change's own
  metadata and each document/comment row; do not show `created_at` or
  `deleted_at` there. History browsing displays all three fields as specified
  in031-13. Convert backend timestamps to local time before formatting; do not
  merely print a UTC or server-offset timestamp unchanged. Test differing field
  values, different input offsets and local daylight-saving boundaries without
  changing saved timestamps. Unit: `Test031AllDisplayedTimestampsUseLocalTime`,
  `Test031ChangeDetailsShowsOnlyUpdatedAt`.
- **031-25 — Epic selection.** Inactive epics must not appear in the
  ChangeDetailsScreen Epic selection list. Use the selected project's active
  epic list; an epic with `active: false` must never become a selectable option.
  This is the only inactive epic requirement for this change. Unit:
  `Test031ChangeDetailsEpicSelectionExcludesInactiveEpics`, covering mixed active
  and inactive responses and no active epics. Include the exclusion in a
  keyboard-driven complete-program test.

## Notes

### Implementation boundaries

Production and tests belong under `cli/`. Own document/comment/history behavior
in the document feature, entity lifecycle behavior in the corresponding feature,
and HTTP/process mechanics in adapters. Keep the root application responsible
for navigation, focus and delegation rather than business rules. Introduce only
the abstractions necessary for this behavior; project/epic layout rollout is a
later change, not a second implementation in this pass.

Narrow documentation edits may align `docs/cli-architecture.md` with the 030 API
inventory and agreed keyboard contracts, and update the CLI assertion ledger and
checkpoint. Implementation/review blocks belong to `cli/implementation-log.md`.
The explicitly requested backend comment-undelete and active-set code, tests, APIHydra suite,
route ledger and validation records are an additional allowed path exception.
Backend/database work is limited to those two endpoints. Inactive epic exclusion
uses the existing epic contract and does not require a new route.
Use the CLI coverage policy in `AGENTS.md`; older specifications' >95%/>90%
CLI targets are historical and do not override the current 80%/70% policy.

### Test reuse and verification

Reuse and migrate the existing tests in `cli/pkg/client/documents_test.go`,
`cli/internal/documents/`, `cli/internal/app/documents_test.go`, and
`cli/integration/documents_program_test.go`. Keep useful owner-scope,
history-selection, provenance, committed-write recovery, cancellation, viewport,
and malformed-response assertions. Replace assertions about removed
`doc/current`, wire `current`, change `open`, and old full-body Spec/PR previews
with the current contract. Migrate affected fake-server responses across all
retained scenarios, including brief clarification and the real PTY suite.

For implementation and review fixes, run from the repository root:

```sh
make -C cli check
make -C cli coverage
make -C cli deps-audit
make -C cli integration-coverage
```

Report actual exits, source provenance, package totals, failed/skipped scenarios,
and independent covered/total production statements. Unit coverage uses the
80% integer gate; terminal coverage uses the 70% integer gate. The integration
campaign includes complete-program and real PTY execution; do not repeat its
constituent PTY run merely to count it twice. Use fake servers and owned local
processes for CLI tests. For the separately requested backend endpoints, the
user identified the existing development database in `backend/config/dev.yaml`;
the owned API coverage backend uses that target explicitly. No database reset or
lifecycle management, direct-SQL test fixtures, or unrelated-record mutation is
authorized. `changes_test` appears in backup scripts but is not selected by the
current API runner and must not be assumed to exist or be configured for this suite.
Coverage shortfalls alone follow the authorized intermediate-dev policy; failed
scenarios and new regressions remain visible and require repair.

The last recorded CLI checkpoint reports passing 80%/70% coverage gates and
pre-existing formatting/package-comment lint failures. These are historical
results, not fresh measurements for this specification. Establish the actual
implementation baseline before changing production code and distinguish newly
introduced failures from existing ones.

The CLI specification is ready for review; no CLI production source has changed.
The separately requested backend endpoints are implemented and verified in the
working tree. `make -C backend check`, `coverage`, `deps-audit`, and `api-test`
exit 0 on the final Go source. Fresh independent backend coverage is unit
1224/1238 (98.8691%) and API 1121/1238 (90.5493%); both gates pass. Doc unit
coverage is244/244. Unit's remaining gap is cmd/server79/93; exact API package
counts and uncovered functions are recorded under `backend/.coverage/api/` and
in the backend checkpoint. All1131 APIHydra requests in19 executable files pass
against an owned backend on19080 using the existing configured development DB.
The first API run failed on an activation fixture that did not restore its
expected cleanup state; the fixture was repaired and the complete campaign rerun.
That failed run established no API coverage and its raw diagnostics are retained
under `/tmp/031-api-failed-01`. No Git publication occurred.
