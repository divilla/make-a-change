# 022 — CLI epic management

P2/spec021 merged as actual dev `1658624c17451c678a72279f0b49e8baf1cb754b`
after clean native review pass05. This is P3 in the approved
CLI plan: deliver full epic management using P2's typed transport and feature
boundaries, preserving P1 editor/config and P2 project contracts. Read AGENTS,
CLI plan/architecture/checkpoint, current epic handlers and domain types.
Allowed writes cli/, numbered spec and plan status. No backend/DB/frontend/
cli-proto/root-helper/.mch/dependency changes. Log cli/implementation-log.md;
implementation child does not commit/push or invoke factories. User authorized
the sequential implementation/review/dev-merge sequence, no stage/prod.

## Acceptance

- P3-01 Implement all five typed POST operations under /api/v1/epic: list(project_id),
  details(id), create(project_id,name), update(id,name), delete(id). Reconfirm
  current DTOs. Creation201 returns ID; update/delete204 empty. Model returned
  id/project_id/name, done_tc/total_tc/completed/change_count and timestamps
  accurately, preserving server completion values rather than inventing history
  or recalculating business state. Every transport method makes one operation;
  reuse P2's finite-timeout/context/status/error and malformed-response handling.
  Unit tests assert exact payloads, statuses, DTOs, errors and cancellation.
  Current backend epic fields are not nullable; do not invent nullability or
  default missing required fields. Empty arrays are valid lists, malformed or
  null responses are contract errors, as in P2.
- P3-02 Reachable list/detail/create/edit/delete actions within the selected project
  have discoverable help, real backend effects, loading/empty/error/success
  feedback and every supported input/display field. Remove former navigation
  placeholders. Require a valid selected project for scoped operations; show
  actionable errors without silently selecting a different project or fake row.
  Own epic state, validation, forms, operation sequencing and results in epics;
  root composes navigation/focus and injects capabilities. No forbidden imports.
- P3-03 Preserve committed mutation outcomes on refresh failure, retaining created IDs
  and explicit saved/deleted plus refresh-failed feedback. Retry only the read;
  never duplicate creates or deletes automatically. Preserve exact raw form/editor
  content and dirty draft retry semantics from P1. Confirm deletion through the
  existing interaction pattern, respect cancellation, and leave unrelated rows
  untouched. Tests verify exactly-once writes, failures, no-op/cancel and recovery.
- P3-04 Carry selected project, entity and operation identity through async requests;
  ignore/cancel obsolete reads and prevent older responses changing current
  selection, form, loading or errors. Preserve program cancellation and orderly
  local config-save shutdown. Unit and complete-program delayed-response tests
  cover project switching, leaving a screen mid-read and shutting down with work.
- P3-05 Replace legacy epic selector transport with the typed list and presentation
  conversion at the feature boundary. Keep change association selectors usable
  without importing the epic implementation into changes. No HTTP envelope/field
  guessing, hidden extra reads or old entity-returning mutation assumptions.
  Update old test fixtures for the recorded current-contract migration and retain
  valuable existing assertions.
- P3-06 Extend reused full-program fake backend/drivers and the terminal manifest to
  cover all five operations with keyboard-driven outcomes, supported fields,
  validation, backend failures and partial success. Include real PTY epic
  navigation/rendering/editor behavior through the existing harness. Pure client
  tests are not terminal integration. Each acceptance bullet has meaningful
  named unit tests plus explicit program/PTY mapping in the contract ledger.

## Verification

Run check, coverage, deps-audit, integration-test and integration-coverage via
make -C cli on final source and after review fixes. All production packages count
in separate unit/terminal profiles. Strict >95% unit and >90% terminal are final
rebuild targets; honestly measured numerical shortfalls alone do not stop
intermediate dev merge. Repair introduced regressions and touched baseline
format/lint; keep unrelated baseline gaps visible with owners. No live backend
is designated; use fixtures aligned to current backend contracts.

Compact checkpoint: actual revision, commands/exits, exact counts/package gaps,
all scenarios/failures/skips, review status and next P4 changes. Record test reuse
and intentional contract changes in ledger; update implementation log and plan
status. Do not predict a future squash hash or declare the full rebuild complete.

## Current baseline and retained contracts

The current epic model is an empty navigation shell; its API exposes a legacy
string-project-ID to selector-options list only. Replace that path and its
permissive decoding with typed current-contract values, converting labels at the
presentation boundary. Do not remove change epic selectors or alter the backend.
Existing epic api.go/model.go formatting baseline is owned by this pass. Preserve
P2 catalog recovery semantics: successful current-project configuration reads
refresh shared selectors, obsolete results cannot overwrite newer selection.
Prevent hidden cached-row selection while lists are loading, and order all
configuration read paths so older startup reads cannot overwrite successful
manual recovery. Canceling manual reads must leave shared catalog loading
available. Preserve deletion and refresh diagnostics if clearing selected-project
configuration fails, including read-only retry behavior.
Keep P1 exact editor bytes and literal slash content, asynchronous atomic local
config writes, queued-save draining on exit and partial write outcomes intact.

Implementation children use $change-code for this supplied specification, with
AGENTS.md's authorized CLI path/reference overrides. Do not invoke another
factory, commit or push; the caller runs native review and the dev merge helper.
No unrelated files or dependency changes. Update the ledger with named tests
for P3-01 through P3-06 and record why any legacy assertion changes.

Run on final code and on each production review fix:

```sh
make -C cli check
make -C cli coverage
make -C cli deps-audit
make -C cli integration-test
make -C cli integration-coverage
```

Compact the durable checkpoint to current contracts, final command results and
next P4; preserve history in Git and temporary logs rather than appending every
review transcript. No predicted dev squash hash or claim that the whole rebuild
is complete. P2 baseline: unit2365/2852 (82.9243%), terminal1655/2852 (58.0295%).
Both strict gates fail honestly; tests/race/program/PTY/architecture/tooling/deps
pass. Six untouched formatting files and baseline package-comment lint remain;
epics/api.go and model.go are owned by this pass. See the compact checkpoint
for exact evidence and remaining owners.
