# 015 — Backend compact handlers

## Authorized execution and fresh decision

This is the tiny R6 compact-code pass, to be considered only after R5 is merged. Expected branch `change/015-backend-compact-handlers`, specification `agent/specs/015-backend-compact-handlers.md`. First reassess the candidate against fresh post-R5 origin/dev. If the scoped-if pattern below is already consistently applied, record R6 **no actionable findings** in the checkpoint and skip an empty implementation branch/code-spec invocation. Do not invent another shortening task to produce a commit. The supervisor confirms the actual clean base and candidate before branch creation.

The user authorized the complete sequential backend refactor factory. Read current AGENTS.md, docs/backend-architecture.md, the official plan, backend contract ledger/checkpoint and this spec. Apply change-refactor/change-code skills. User scope overrides skill defaults: do not read PRD, skeleton, CLI, frontend or unrelated architecture references. All implementation, tests, artifacts and logs stay under backend/. This committed spec and the top status of agent/backend-refactor-plan.md are authorized exceptions. No root scripts, root AGENTS.md, db/, docs/, dependencies, tool configuration or Go-managed transaction changes. Append the prescribed change-code log to backend/implementation-log.md, excluding the log from counts. The implementation child does not commit/push; the factory owns those steps.

Final goals remain >95% unit and >=90% APIHydra production statement coverage. Measure both honestly; a genuine numerical shortfall alone does not stop the agreed intermediate sequence. Keep strict gates, all production packages, assertions, scenarios and independent profiles. No artificial tests, defensive-code removal, coverage filtering or threshold changes. There is no line-count or denominator-reduction objective. Keep the separate016 final-verification/owned-DB-outage pass out of R6. No stage/production promotion.

## Only candidate in scope

Some error-only mutation handlers currently declare `err := a.s.Operation(...)` and use it solely in the immediately following `if err != nil`. Adjacent delete/set-document handlers already use `if err := a.s.Operation(...); err != nil`. Scoping these one-use errors to the conditional narrows their lifetime and removes unnecessary local scaffolding while preserving the explicit handler structure.

The preparation snapshot identified project.updateProject, epic.updateEpic, change.updateEpic/updateChangeTypes/updateTitle/updateBrief/updateSpec/updatePR/updatePRUrl/updatePhase/updateOpen, and testcase update/update-done/delete handlers. Recheck actual names/code after R5; do not mechanically trust the list. Change only separate single-error assignments whose variable is not used afterward, keeping the exact call/context/request expression, error branch and success return.

Do not touch result-returning handlers where `res` or another value is needed later. Do not move service/business logic, change validation/binding/central error calls, combine operations, reorder evaluation, change messages/statuses/bodies, add generic middleware/CRUD helpers or compress SQL. Do not change normalization, rendering, optional IDs, completion arithmetic, constructor storage, configuration or test harness logic; those are other categories. Preserve all newly fixed validation/URL error causes from specification011.

## Acceptance criteria

R6-01 Fresh bounded audit: record the actual remaining handlers and base SHA. Each changed error variable is used only by its immediately following conditional. If no candidates remain, report no action and skip an empty branch. Reference inspection establishes the scope; do not add a syntax/AST test merely proving an `if` initializer was used.

R6-02 Exact behavior parity: every changed handler invokes the same service exactly once with the same domain request/context after unchanged binding/validation; failed calls return the same centralized error/cause; successful calls return the same truly empty204. Public routes, IDs/DTOs, response envelopes, error identity, request semantics and side effects remain unchanged. Existing meaningful exact status/body/error-cause and no-extra-call tests prove these criteria; add a new test only for an actual missing behavioral assertion, not to mirror the edit.

R6-03 Narrow implementation: source changes are limited to the scoped conditional declarations in verified mutation handlers and required formatting of those lines. No result-variable lifetime changes, hidden helper, reordered call, business mapping, broad naming/comment/formatting sweep, repository/query or test weakening. Inspect each diff hunk manually. Statements/coverage may be unchanged or reflect normal Go instrumentation; report measured counts without optimizing them.

R6-04 Genuine verification: retain and run the meaningful affected unit tests, full backend checks, fresh unit/APIHydra coverage and complete owned legacy HTTP campaign. All application tests and normal check must remain green from R5. Numerical gate outcomes are reported honestly and separately. No syntax-only or duplicate mirror test is needed for this reversible local refactor. No ownership, cleanup or APIHydra runner behavior changes.

R6-05 Compact handoff: update checkpoint with scoped handler list, existing parity-test mapping, exact command exits and separate covered/total statements/package gaps; update only the official-plan top status. The next action is the separately scoped016 final verification/owned-outage campaign and combined architecture review, not additional R6 compression. No prose-only unit tests are required.

## Validation and factory workflow

Apply the pinned formatter to touched Go files and inspect the diff. Run affected API tests, then all of the following on final code and after any review fix:

```sh
make -C backend check
make -C backend coverage
make -C backend deps-audit
make -C backend api-test
make -C backend legacy-api-test
```

Use `GOLANGCI_LINT_CACHE=/tmp/mch-r6-lint-cache` when a writable cache is needed; do not suppress linter findings. Existing APIHydra suites remain under backend/apih-tests and retain all scenarios against the owned disposable cluster/server; legacy counters never enter APIHydra coverage. Fresh reports use the complete unchanged production inventory policy and strict thresholds. Run git diff --check, inspect tracked/untracked scope and preserve evidence under /tmp/mch-r6* or backend/.coverage/verification.

Supervisor sequence when candidates remain: author/commit this spec on matching015 branch from fresh origin/dev; run `scripts/codex-code-spec.pl agent/specs/015-backend-compact-handlers.md`; inspect the tiny implementation diff and behavioral evidence; run `scripts/codex-review-loop.pl agent/specs/015-backend-compact-handlers.md --base origin/dev`; repair valid findings and revalidate; publish compact checkpoint; invoke `scripts/merge-to-dev.pl`; verify clean local/remote dev and hand off016. Inspect state before retrying partial tool/Git actions; transient failures may be retried at most twice, code/test failures require repair. No promotion beyond dev.
