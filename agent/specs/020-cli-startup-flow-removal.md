# 020 — CLI startup and Flow removal

P1 on change/020-cli-startup-flow-removal, from fresh origin/dev e46bf64.
P0 is implemented and merged after native review pass04 was clean. Execute only
this bounded specification; the caller owns the rest of the factory sequence.

## Contract and scope

Use AGENTS.md, agent/cli-rebuild-plan.md, docs/cli-architecture.md and the current
CLI checkpoint. Full factory execution is authorized; helper owns commits/pushes.
User resource override: retain .mch/config.yaml without Flow fields, .mch/tmp/,
and exactly .mch/default/prompts/{brief-rewrite,brief-resolve,spec-write,
spec-review,spec-fix}.md. Remove YAML under .mch/default/ and obsolete prompts.
Inspect existing .mch/default/Makefile and scripts as migration references before
removing Flow dispatch and obsolete automation. Root scripts/ remain unchanged.
Allowed writes: cli/, scoped .mch/default/ tracked assets and .mch/config.yaml if
Flow settings actually exist, docs/cli-architecture.md for user override/current
migration status, the plan status, and this pass's numbered spec. Preserve user
workspaces, unrelated documents, backend/db/frontend/cli-proto and dependencies.

## Bounded purpose

Remove legacy Flow startup and reachable workflow families while retaining a
working ordinary terminal shell and general editor/process/HTTP/config adapters.
Do not implement the new fixed controller, full HTTP migration or feature CRUD
in this pass. Those have explicit P2–P9 owners. Do not retain a hidden legacy
runner or expose fake saves/placeholder commands claiming unfinished capabilities.

## Acceptance criteria

- P1-01 Startup loads local backend URL/project configuration independently of Flow
  YAML/help files, agent executable, shared prompt generation, workspace creation
  or Git branch state. Keep repository-root lookup separate from API transport;
  use supplied root in program tests. Existing config precedence and actionable
  errors remain documented. Unit tests cover required/malformed config, missing
  Flow files, missing agent executable, and ordinary program startup/navigation.
- P1-02 Remove Flow types, catalogs, task steps/statuses/modes, stage dispatch/hooks,
  old def/spec/PR/review/chat command families and restoration protocols from
  startup and visible UI. Keep normal API reads/edits that later passes migrate.
  Characterize retained help/navigation/rendering. Tests prove obsolete commands
  cannot invoke shell or process runners and ordinary navigation remains usable.
- P1-03 Preserve local project selection in memory when saving fails; persist config
  atomically without losing the previous valid file on failure. Read/write local
  config outside View/synchronous Update and report failure explicitly. Unit tests
  prove atomic replacement, error causes and partial success; program test drives
  selection plus successful/failed save. Avoid new project catalog semantics here.
- P1-04 Inventory retained prompt content before cleanup. Rework def-rewrite into
  brief-rewrite and create/adapt brief-resolve and spec-fix; adapt spec-write and
  spec-review. Prompts preserve intent, ask material questions, use explicit
  supplied input/output/context paths, and prohibit implementation/Git automation.
  No old Flow variables/session protocol/shared change-types file dependencies.
  Brief resolve incorporates answers without silently resolving unanswered
  blockers. Spec review returns findings or explicit success; spec fix returns
  to review rather than declaring completion. Keep prompt serialization protocol
  details for P8/P9 unless required now; do not invent alternate workflows.
  Tests check retained prompt responsibilities and exact five-file inventory.
- P1-05 Remove all YAML under .mch/default/ and obsolete Flow Makefile/scripts after
  extracting general behaviors needed by retained adapters. Preserve .mch/config.yaml
  and .mch/tmp/ contents. Do not delete unrelated existing files or touch root
  factory scripts. Update the architecture only to align prompt/resource ownership
  and migration facts, not to weaken the fixed workflow/product requirements.
- P1-06 Migrate test coverage as behavior moves: remove assertions only for explicitly
  removed Flow behavior, recording reason in the reuse ledger. Keep useful config,
  editor handoff, process cleanup, selection, rendering and file-ownership tests;
  rehome helpers before deleting old files. Obsolete shell tests for already
  missing root helpers may be removed as Flow-history assertions, while retained
  root scripts stay untouched. Do not preserve forbidden behavior to keep old
  tests passing or drop valuable assertions to make new tests pass.
- P1-07 Keep eligible complete-program and PTY suites meaningful after the old agent
  screen is removed: reuse their drivers and replace obsolete scenario specifics
  with retained startup/navigation/editor/redraw/scrolling behavior. Do not
  replace PTY coverage with direct model tests or let selection manifests become
  stale/no-op. Update the P0 manifest and suite completeness checks explicitly.
  Scripted agent progress scenarios return with P8/P9, tracked in the ledger.

## Recorded baseline and required repairs

P0 final unit coverage is 3060/4021 (76.1005%); terminal coverage is 1796/4021
(44.6655%). Unit race, vet, tooling and deps-audit pass. Check exposes 37 files
with baseline formatting, 12 lint issues, and the forbidden agent/parser.go ->
changes dependency. Retained Flow tests include twelve baseline missing/stale
asset failures, plus an intermittent `read |0: file already closed` from the old
agent subprocess path. Preserve raw P0 evidence; the checkpoint lists ownership.

This pass removes the forbidden parser dependency by moving retained pure
parsing to its proper owner or removing truly obsolete parsing with its behavior.
If the general agent process adapter remains, repair the observed pipe ownership
race with a deterministic subprocess test proving readers finish before Wait
closes their pipes. If it is removed with the obsolete workflow, record that
reason and the future P8 general-process coverage requirement; do not retain an
unused broken runner. Remove obsolete Flow assertions only alongside explicit
behavior removal, so the retained suite is meaningful and passing. Format and
repair lint in touched/rebuilt code. Keep remaining unrelated baseline findings
assigned to future passes, never introduce new ones.

P0's terminal manifest has legacy-specific test names and its validation currently
restricts allowed prefixes. Adapt both explicitly as scenarios migrate. Keep the
real instrumented child, complete program execution, source inventory, strict
coverage gates, interruption cleanup, and truthful completeness checks intact.

## Verification

Run meaningful unit tests per acceptance bullet, real full program + PTY suites,
check, unit coverage, integration-coverage, and deps-audit. Reuse existing tests.
Final targets >95% unit, >90% integration; valid numerical shortfalls do not stop
intermediate dev merge, but gates remain strict and failures visible. Fix new
regressions. Record actual source revision/counts/package gaps/command exits and
removed-test reasons in cli/agents/cli-rebuild-checkpoint.md and contract ledger;
use cli/implementation-log.md. Next pass P2 projects/typed transport.

Run final commands on the implementation and after any review fixes:

```sh
make -C cli check
make -C cli coverage
make -C cli deps-audit
make -C cli integration-test
make -C cli integration-coverage
```

Use injected fake HTTP/agent/editor collaborators and isolated resources. No
live backend/database access is authorized or required. Record all command
exits and actual package counts; below-target coverage alone does not stop the
factory. Test every acceptance bullet with meaningful unit tests and terminal
scenarios where applicable. Do not edit root helpers, backend, DB, frontend,
cli-proto, dependencies, or unrelated files. Do not commit/push from the child.

## Notes

The approved plan/user override governs prompt location despite the old target
architecture saying CLI-owned prompts. The scoped architecture edit records this
already resolved decision; no new permission is needed. Implementation logs stay
under cli/ per AGENTS.md, overriding generic skill defaults. No missing skeleton,
PRD or old Flow behavior is a binding contract. Preserve temporary directories
and existing user files; remove only obsolete tracked assets in scope.
