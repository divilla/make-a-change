# CLI rebuild plan

Status: authorized execution. P0–P3 merged; latest actual dev is P3
`80c67871aaa29ed92a967aa70b449b29bd88c406`. P4/spec023 implementation resumed
from snapshot `3ad7aa7`; acceptance inspection and the implementation log are
finished. Fresh unit/race/program/PTY/architecture/tooling/dependency scenarios
pass. P4 review fixes01 clear canceled create subfields and bound mutation feedback;
review fixes02 return edited briefs to the create form before saving and let
identity rows scroll in short detail panes. Review fixes03 preserve the edited
field when Ctrl+C clears an update prompt, preventing accidental spec insertion.
Fresh unit3028/3530 (85.7790%) and terminal2550/3530 (72.2380%) measurements
still fail strict gates, alongside the tracked static baseline. P4 awaits
caller-owned publication, further native review and dev merge. See the
[checkpoint](../cli/agents/cli-rebuild-checkpoint.md) for exact evidence/continuation.
Supporting document current/insert and testcase reads moved into P4 as required
to preserve editors while removing embedded change fields; full P5/P6 scope
remains. Continue the authorized factory through all remaining passes; no
stage or production promotion.

Reference: [backend refactor plan](backend-refactor-plan.md), especially its
six-step specification → branch → implementation → verification/review → dev
merge → checkpoint sequence. Its historical backend APIHydra campaigns and its
later diagnostic-only API coverage policy do not define CLI integration testing.

## Confirmed decisions

| ID | Question | Effect on this plan |
| --- | --- | --- |
| D1 — resolved | Full [CLI architecture](../docs/cli-architecture.md) scope is confirmed. Retain the relevant prompts, temp storage, and configuration under `.mch/`; remove Flow configuration. | This user clarification overrides the architecture's CLI-owned prompt location. Preserve `brief-rewrite`, `brief-resolve`, `spec-write`, `spec-review`, and `spec-fix` prompts for the fixed workflow. |
| D2 — resolved | Integration coverage combines complete-program tests with injected terminal I/O and fake backend/agent collaborators with real PTY tests. | Reuse the existing complete-program harness and PTY infrastructure. Count production statements exercised through those program boundaries; HTTP-client-only tests do not count. |
| D3 — resolved | Numerical coverage targets apply to the final result. | Continue intermediate dev merges with passing tests and honestly reported numerical shortfalls. Keep strict Make gates and run both suites throughout; coverage shortfalls alone do not stop the sequence. |
| D4 — resolved | Keep prompts in `.mch/default/prompts/`, leaving only `brief-rewrite`, `brief-resolve`, `spec-write`, `spec-review`, and `spec-fix`. Use existing scripts and Makefile as migration references. | Preserve/adapt relevant prompt content and general execution mechanics; remove obsolete prompt files and Flow behavior. Keep `.mch/tmp/`. |
| D5 — resolved | Remove all YAML under `.mch/default/`; retain `.mch/config.yaml`. | Remove Flow settings from retained configuration. Do not extend YAML deletion to unrelated repository files. |

Confirmed requirements are to reuse existing tests as much as possible, test the
terminal application through integration tests, and achieve **strictly >95% unit
statement coverage** and **strictly >90% integration statement coverage**. Exactly
95% or 90% does not pass. Ask about unresolved product behavior, conflicting
contracts, or scope changes before writing dependent acceptance criteria; do not
invent requirements to unblock a pass.

## Scope and authorities

Rebuild the existing `cli/` module to the CLI architecture and the user's
resource-location override, keeping the executable name `mch` and module/import
prefix `cli`. This is a contract and
workflow migration followed by cleanup, not a behavior-preserving refactor of
the old Flow system.

- Use the CLI architecture for target behavior and package boundaries. Use current
  backend handlers, request/response definitions, and domain types as the HTTP
  contract. Record conflicts and ask for a decision rather than changing the
  backend or preserving obsolete client assumptions.
- Keep implementation, tests, tooling, generated reports, and execution logs
  under `cli/`, with the explicitly retained `.mch/` resources below. Proposed
  records are `cli/agents/cli-contracts.md`,
  `cli/agents/cli-rebuild-checkpoint.md`, and `cli/implementation-log.md`.
- This plan and `agent/specs/NNN-cli-*.md` are factory documentation. Migration
  of `.mch/` resources is an explicit additional path exception: retain the five
  named prompts under `.mch/default/prompts/`, temp storage at `.mch/tmp/`, and
  configuration at `.mch/config.yaml` with Flow settings removed.
  Remove all YAML files under `.mch/default/` and all other prompt files from
  `.mch/default/prompts/`. Inspect the existing `.mch/default/Makefile` and scripts
  as references before removing obsolete dispatch and automation. Remove obsolete
  Flow assets without deleting retained prompts, user documents, or existing
  workspaces. Do not treat all of `.mch/` or `.mch/default/` as disposable before
  completing the retained-resource inventory.
- Keep backend, database, frontend, and `cli-proto/` unchanged. The prototype is
  neither a second implementation to extend nor a new dependency. Root `scripts/`
  are the maintained mini-factory tools; they are distinct from obsolete Flow
  scripts under `.mch/default/` and must remain intact.
- Follow [AGENTS.md](../AGENTS.md). Plan any necessary CLI completion-policy update
  there explicitly in the tooling specification before execution. Do not silently
  expand a CLI-only specification into root documentation or tooling edits.
  The resource-location clarification also requires a narrowly scoped alignment
  of `docs/cli-architecture.md` during the relevant specification; record that
  documentation exception explicitly so the target architecture remains accurate.
- Do not infer requirements from historical backend specifications or stale
  template references. The current spec template mentions `skeleton/`, which is
  absent; specify the actual CLI/backend contracts rather than inventing a
  skeleton. Surface any conflict with implementation-skill defaults when preparing
  the first specification.
- Preserve the current framework and dependency choices unless a concrete need
  requires a separately agreed change. No stage or production promotion belongs
  to this factory sequence.

## Inspected baseline

Inspection: commit `79d34cf`, branch `change/cli-rebuild-api-connector`.
These observations describe source inspection, not passing application tests.

| Evidence | Planning consequence |
| --- | --- |
| `cli/Makefile` selects `mch/...`, while `cli/go.mod` declares `cli`. A dry run warns that no packages match. | Repair package discovery before trusting checks or coverage. Fail on discovery errors or an empty production inventory. |
| `lint` runs modifying `goimports -w`; setup uses `@latest`; Docker defaults to Go 1.25 while the module requires 1.26.0. | Separate formatting from verification, pin tools, and derive the default toolchain from `go.mod`. |
| No CLI unit/integration statement-coverage gates exist in the Makefile. | Establish independent measurement and failure propagation in the first pass. |
| `pkg/client/http.go` uses `/project/get`, `/change/get`, global options, `/change/assign-flow`, and entity-returning mutations. | Migrate to current routes, project configuration, typed DTOs, ID-only creation and empty successful mutations. |
| The root app owns feature forms, HTTP orchestration, editor/process effects, and Flow coordination. Epic/testcase feature models are incomplete. | Deliver coherent feature ownership with each rebuilt capability; retain only shell responsibilities in `internal/app`. |
| Startup and existing integration fixtures depend on Flow files and generated change-type prompts. | Remove those dependencies while preserving ordinary configuration, terminal startup, and useful test mechanics. |
| `integration/new_change_program_test.go` drives `app.RunProgramWithIO` with input/output and controlled collaborators. | Reuse its program driver, synchronization, and partial-failure scenarios; migrate assertions to the target workflow. |
| `integration/existing_artifact_program_test.go`, `default_flow_test.go`, and `startup_change_types_test.go` mix retained intent with obsolete Flow contracts. | Classify assertions individually before changing or removing them. Test reuse does not mean keeping removed behavior. |
| `integration/terminal/rewrite_view_test.go` builds `mch`, drives a PTY through `socat`, and checks redraw, colors, scrolling, and subprocess output. It currently skips without `socat`. | Reuse the PTY harness and instrument its child executable. Missing PTY prerequisites must remain visible and cannot establish a passing complete campaign. |
| `integration/architecture_test.go` checks `mch/internal/` and `domain`, while the implementation uses `cli/internal/` and `dto`. | Repair import enforcement and prove it rejects a deliberately forbidden import. Its present success would not prove architectural compliance. |

Planning checks actually run:

- `go list ./...` from `cli/`: exit 0; enumerated the CLI production and test
  packages under the real module prefix.
- `make -C cli -n check`: exit 0, but emitted the no-matching-packages warning and
  showed mutating formatting plus empty package arguments. This was a dry run,
  not successful backend or CLI validation.

Application tests, vulnerability scans, and coverage have not been run for this
documentation task. Both CLI coverage totals are **unmeasured**.

## Mini-factory operation

When execution is requested, use the existing
[repository helpers](../scripts/README.md) sequentially. Complete one bounded
specification before starting another; do not stack unmerged change branches.
Inspect state before retrying a partially completed helper operation.

1. **Author one specification.** Reassess the next pass against the latest
   checkpoint and actual source. Prepare `agent/specs/NNN-cli-slug.md` with scope,
   stable acceptance IDs, current/target contracts, per-bullet unit-test mapping,
   terminal scenarios, baseline failures, coverage rules, and allowed paths.
   Direct implementation and review-fix logs to `cli/implementation-log.md`.
   Resolve material questions before implementation. Prepare the spec outside
   the checkout until its branch exists if it is not already on dev.
2. **Create its branch from freshly fetched `origin/dev`.** Check a clean tree
   and `bash scripts/git-auth.sh --check` first. Use `scripts/branch-create.sh`
   when the spec already exists on `origin/dev`; otherwise create the equivalent
   `change/NNN-cli-slug` branch from that base and install/commit the prepared spec
   there. Choose unused numbers at execution time. Ensure dev contains the agreed
   architecture and plan first. Do not absorb unrelated work into helper commits.
3. **Implement through `scripts/codex-code-spec.pl agent/specs/NNN-cli-slug.md`.**
   The helper invokes `$change-code`, commits, and pushes. Inspect its actual diff,
   final response, tests, and coverage evidence; process success alone does not
   establish acceptance. Keep fixes within the bounded specification.
4. **Verify and review.** Run the required CLI checks and both suites, then
   `scripts/codex-review-loop.pl agent/specs/NNN-cli-slug.md --base origin/dev`.
   The explicit base avoids the helper's default remote branch. Revalidate final
   code after review fixes, refreshing coverage when production code or test
   scenarios change. Resolve findings and document remaining baseline gaps.
5. **Merge to dev through `scripts/merge-to-dev.pl`.** Numerical coverage shortfalls
   alone do not block intermediate merges; record actual counts and outstanding
   gaps. Failing tests and new regressions must be repaired. The helper requires
   a clean published change branch containing fetched dev. If dev advanced,
   reconcile and rerun affected verification before merging. Verify the actual
   published dev SHA afterward; never predict the squash hash.
6. **Continue from a compact checkpoint.** Include the checkpoint in the reviewed
   branch before merge, then read it from dev for the next specification. Record
   completed acceptance IDs, commands and exits, separate coverage counts,
   remaining gaps, decisions, and the exact next action. Record post-merge SHA
   evidence in the next checkpoint or handoff without inventing it beforehand.

The factory implements this rebuild. The terminal application's own agent
workflow, by contrast, ends at a reviewed spec and performs no code implementation,
Git publishing, merge, or deployment.

## Proposed bounded rebuild passes

This sequence uses the confirmed full scope and resource override. Pass labels
are planning identifiers, not reserved specification numbers. Split a pass further if its contract or diff
cannot be reviewed coherently. Tests ship with every pass.

| Pass | Scope and completion evidence |
| --- | --- |
| P0 — Verification foundation | Repair CLI Make targets, pin development tools, establish all-production package inventory, separate unit/program/PTY/tooling suites, repair architecture checks, and produce independent fresh coverage reports and strict gates. Build the contract/test-reuse ledger and checkpoint. Characterize retained terminal behavior; measure and report baseline failures without weakening tests. |
| P1 — Startup and Flow removal | Inspect existing scripts and Makefile as migration references. Remove legacy Flow loading, commands, hooks, generated change-type prompts, all YAML under `.mch/default/`, obsolete assets, and obsolete workflow tests. Leave only the five specified prompts in `.mch/default/prompts/`, preserving/adapting relevant content; retain `.mch/tmp/` and `.mch/config.yaml` without Flow settings. Retain useful adapters and harness helpers. Ordinary startup/navigation must work without Flow resources or an agent executable. Make project selection/configuration failures explicit and local persistence atomic. Align the architecture's resource-location text with the user clarification. Record temporarily unavailable target capabilities rather than retaining a hidden old engine. |
| P2 — Projects and typed transport | Establish context-aware finite-timeout HTTP mechanics and typed error/status handling; implement project list/details/create/update/delete/config as reachable actions. Load project-scoped catalogs, preserve selection after local-save failure, and isolate stale results. Reuse client tests with corrected routes and mutation contracts. |
| P3 — Epics | Complete epic list/details/create/update/delete screens and fields, with loading, errors, success feedback, nullable values and returned completion data. Own behavior in the epic feature and test keyboard-driven operations. |
| P4 — Changes | Complete change list/details/create/delete, associations, and every supported field update, including prerequisite changes and permitted clears. Use `brief`, read backend identity, and remove obsolete field/envelope guesses. Distinguish committed writes from failed refreshes; never retry creates automatically. |
| P5 — Testcases | Complete testcase list/create/update/done/undone/delete. Refresh separate reads as required; remove assumptions that testcase mutations return a change. Keep testcase behavior in its feature even when embedded in change details. |
| P6 — Documents | Implement list/current/details/insert for project, epic, and change owners using configured document types. Display history and append versions with correct provenance; expose all request fields. No invented update/delete routes. Provide the document capabilities needed by the fixed workflow. |
| P7 — Configuration and health | Complete configuration list/details/insert/update/delete, immutable slug identity, all six catalog arrays including explicit empties, project-catalog refresh after edits, and both health routes with degraded status. All operations must have discoverable terminal access. |
| P8 — Brief clarification | Implement editable brief intake, original-input preservation, repeated concrete questions and answers, explicit unresolved-blocker state, and document persistence for new/existing changes. Introduce the fixed controller, dedicated prompts, and injected runner; verify missing configured brief/spec types, cancellation, and partial persistence. |
| P9 — Spec write, review, fix | Write from the clarified brief and answers, review the current revision, fix findings or ask questions, and always return fixes to review. Only an explicit successful review of the current spec completes the workflow. Test multiple loops, malformed output, stale results, edits invalidating downstream state, failures and cancellation. |
| P10 — Combined acceptance | Reconcile every architecture requirement and backend operation with a reachable terminal scenario and named unit tests. Finish cross-feature lifecycle, editor handoff, ownership, and failure coverage. Produce fresh independent coverage and a remaining-gap report; do not postpone all coverage work until this pass. |

P0 precedes all production migrations. P1–P9 proceed sequentially so each branch
can be reviewed and merged independently. P8 depends on project catalogs, change
creation, and document persistence; P9 depends on P8's explicit blocker and
revision state. Reassess the sequence when a concrete dependency emerges.

## Six cleanup iterations after contract alignment

Repeat the backend plan's six categories on the rebuilt CLI, preserving its new
contracts. Reassess candidates first; record “no actionable findings” and skip an
empty branch rather than manufacturing changes.

| Iteration | Candidate and parity requirement |
| --- | --- |
| R1 — Dead code | Remove proven unused declarations and obsolete Flow remnants after reference/build-path inspection. Preserve registered actions and supported configuration. Never delete behavior to improve a coverage denominator. |
| R2 — Duplicated paths | Consolidate equivalent surviving form, selector, mutation-result, and transport paths within their owner. Preserve ordering, nullable/empty distinctions, errors and partial success; avoid a generic CRUD framework. |
| R3 — Oversized modules | Remeasure the root app, client, and agent controller. Move coherent responsibilities to the architecture's owners. Preserve navigation, rendering, cancellation, and terminal handoff. |
| R4 — Stale abstractions | Remove obsolete wrappers, globals and overbroad interfaces while retaining useful injection seams. Prove project/model isolation, bounded progress delivery, and owned-process/file cleanup. |
| R5 — Local patterns | Align naming, comments, context propagation, and package boundaries with the established CLI conventions. Verify actual imports and retain ordinary API access independent of agent resources. |
| R6 — Compact code and final review | Simplify only where readability improves, preserving evaluation order and side effects. Run final acceptance, both complete suites, independent coverage, vulnerability checks, and combined architecture review on the reviewed revision. |

## Reuse existing tests deliberately

P0 creates an acceptance → existing test → retained/revised/replaced/removed
assertion → target unit test → terminal scenario ledger. Preserve meaningful
assertions and reusable fixtures rather than replacing whole suites by default.

- Reuse complete-program input/output drivers, synchronized output waiting,
  controlled editor/agent execution, isolated directories, and failure scenarios.
- Reuse PTY setup, keyboard interaction, rendering capture, scrolling, and
  terminal restoration checks. Correct obsolete route and Flow fixtures.
- Adapt client tests to assert exact current routes, typed payloads, status codes,
  empty 204 handling, malformed responses, cancellation, and preserved causes.
  These remain adapter tests; they do not establish terminal integration coverage.
- Remove an old assertion only with a documented architecture/contract reason.
  Replace retained behavior coverage in the same pass. Assertions exclusively
  about removed Flow scripts, PR generation, branch automation or generic chat
  disappear with those features; useful process/cleanup checks transfer to the
  new adapters and controller.
- Preserve and adapt useful prompt-content assertions for the five retained
  prompts. Test their `.mch/default/prompts/` paths and fixed-workflow responsibilities,
  configuration without Flow fields, and operation-owned temp storage. Keeping
  these resources does not retain Flow dispatch, hooks, or the old session protocol.
- Keep structural architecture tests and tooling tests separately identified.
  Their presence under `integration/` does not make them terminal scenarios or
  entitle their execution to integration coverage credit.

Every acceptance bullet needs a meaningful named unit test, reusing one where it
already proves the criterion. Add tests for uncovered behavior or a specific
criterion, not merely to mirror implementation or increase test counts.

## Terminal integration strategy

Integration tests exercise application startup, user input, the event loop,
navigation, forms, rendering, adapter orchestration, and shutdown. Scenarios must
assert visible results and meaningful effects. A direct HTTP call, client method,
feature method, or controller transition in isolation is not terminal integration.

Eligible executions include complete-program `RunProgramWithIO` scenarios and
PTY-driven executable runs. Both suites are required; retain real PTY coverage for
terminal-specific behavior. Controlled backend/agent collaborators in existing
tests are valuable for deterministic failures, but do not prove live backend or
real-agent compatibility. Record which collaborators each scenario uses.

Required scenario families under the full architecture scope:

- Startup, version/argument handling, configuration errors, help, navigation,
  selection persistence, project switching and ordinary use without Flow/agent
  resources.
- Every operation in the architecture's API inventory through a discoverable
  terminal action, including all supported editable fields and permitted clears.
- Empty/loading/error/success states, malformed server data, cancelled reads,
  delayed stale results, mutation-success/refresh-failure, and safe read retries.
- Document history/version inspection and append semantics, missing catalogs,
  explicit empty configuration arrays, and both health routes.
- Complete brief-question-answer and spec-review-fix loops, unresolved blockers,
  revision invalidation, malformed review results, missing output, partial saves,
  retained drafts, cancellation, and explicit successful completion.
- Editor handoff and restoration, redraw, resize, color, scrolling, keyboard
  focus, bounded agent output, and clean exit with owned subprocesses reaped.
- Scratch-directory ownership, refusal to overwrite unrelated files, isolated
  test data, and cleanup on success/failure/cancellation.

Use existing `cli/integration/` and `cli/integration/terminal/` infrastructure.
Backend APIHydra suites do not satisfy this CLI requirement. If live compatibility
validation needs a server, obtain its designation before running it; the
repository's configured URL alone is not permission to mutate unrelated records.
Do not add database access, resets, fixtures, or backend lifecycle management to
the terminal suite. Record any required live validation that remains unconfigured.

## Independent statement coverage

Both denominators include all production Go packages under CLI `cmd/`,
`internal/`, and `pkg/`, plus any production code found elsewhere during inventory.
Include packages without tests and report package/function gaps. Exclude test
harnesses, tooling tests, backend code and dependencies explicitly, never difficult
CLI production code. Count Go executable statements, not endpoints, commands,
test cases, source lines, or averages of package percentages.

- Unit gate: `100 × covered > 95 × total` using only unit-suite execution.
- Integration gate: `100 × covered > 90 × total` using only complete-program and PTY
  program executions. Unit and adapter-only profiles never enter this result.
- Instrument child `mch` processes explicitly; coverage of the Go test harness
  does not measure an uninstrumented subprocess. Collect process counters in
  private run directories and flush them through orderly owned-process exit.
- For eligible in-process program tests, collect production-package coverage
  separately from unit and non-program tests. Normalize compatible profiles by
  statement identity and union hits; do not sum repeated coverage or average
  percentages. P0 must verify this accounting against known fixtures.
- Compare emitted metadata with the full production inventory. Report unlinked
  or unrepresented packages as gaps; never silently shrink the denominator.
- Use fresh profiles tied to the tested source revision, binary/build settings,
  tool versions, suite/scenario inventory and command exit results. Never combine
  revisions, stale runs, unit counters, or incomplete campaigns.
- Missing tools, unexpected skips, absent counters, crashes, failed assertions,
  timeouts, or failed cleanup cannot establish a passing full campaign. Preserve
  diagnostics and failure exits. A complete below-threshold run is valid evidence
  with a failing numerical gate; that numerical shortfall alone does not block
  intermediate dev merges under the confirmed final-result policy.

Keep generated artifacts under `cli/` or temporary storage. Reports must show
covered/total statements and percentages separately for each suite, package gaps,
failed/skipped/blocked scenarios, and provenance. Rounding cannot change a gate.

## Verification interface

P0 implements these interfaces in `cli/Makefile`, using the backend tooling
patterns for inventory and accounting while keeping the terminal campaign
independent of backend/API lifecycle tooling. Current failures remain visible in
the checkpoint.

| Command from repository root | Intended contract |
| --- | --- |
| `make -C cli init` | Install pinned development tools; setup only. |
| `make -C cli format` | Explicitly apply formatting; inspect the diff. |
| `make -C cli check` | Read-only format check, lint, vet, uncached unit race tests, architecture and tooling checks; include integration source in static checks. |
| `make -C cli coverage` | Fresh unit coverage across all CLI production packages, strict >95% gate. |
| `make -C cli integration-test` | Retained complete-program, startup and legacy Flow checks, excluding architecture; no coverage claim. |
| `make -C cli terminal-test` | Real PTY behavioral suite; missing prerequisites visibly prevent full validation. |
| `make -C cli integration-coverage` | Complete program and PTY campaign with instrumented production code, independent strict >90% gate, and explicit scenario/PTY completeness checks. |
| `make -C cli deps-audit` | Vulnerability scan of CLI dependencies with visible findings/failures. |
| `make -C cli tooling-test` | Isolated verification-runner and measurement-accounting tests. |
| `make -C cli test_version` | Docker checks using the module's Go version by default; run when toolchain compatibility changes. |

Define the coverage runner's constituent suites in P0 so execution does not repeat
the same full behavioral campaign unnecessarily. For every production pass run
`check`, `coverage`, `deps-audit`, and the complete integration campaign, including
real PTY tests. Recheck final implementation after review fixes. The integration
coverage campaign runs both retained program tests and real PTY tests, combining
only their eligible production counters. Unit and integration measurements remain
independent.

## Completion and checkpoint policy

Completion requires the agreed architecture scope implemented, every acceptance
bullet mapped to meaningful unit coverage, all reachable terminal actions and
workflow invariants verified, checks passing, **>95% unit** and **>90% integration**
statement coverage, and final review findings resolved. These numerical targets
apply to the final rebuild result, not each intermediate branch. Run both suites,
measure coverage, and improve meaningful tests in every pass. Continue the agreed
review/merge sequence when only the numerical target is unmet; keep Make gates
failing honestly and repair failing tests or new regressions before merging.

At each pass, record the specification/base/tested revision, retained and changed
contracts, reused/replaced tests, actual command exits, fresh independent counts,
package gaps, failures/skips/blockers, reviewed result, and next action. Condense
the checkpoint at cleanup boundaries rather than accumulating a transcript.

If meaningful testing still leaves a final numerical gap, report **target not
met**, the exact statements and observable behavior left uncovered, attempted
scenarios, and concrete options for discussion. Never lower a threshold, merge
unit counters into integration results, remove live behavior to improve numbers,
add artificial execution paths, hide missing PTY tests, or claim completion from
a successful factory subprocess alone.
