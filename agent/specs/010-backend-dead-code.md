# 010 — Backend dead code

## Authorized execution

Implement this specification only after P4 has completed and merged to dev. The expected branch is `change/010-backend-dead-code`, with this specification at `agent/specs/010-backend-dead-code.md`; the supervisor must confirm the next available number and fresh origin/dev before branch creation. The user approved the complete sequential backend factory/refactor workflow. This is R1, the first behavior-preserving cleanup pass, not another API migration.

Read the current AGENTS.md, docs/backend-architecture.md, this specification, agent/backend-refactor-plan.md, backend/agents/backend-contracts.md and the latest backend checkpoint. Apply change-refactor and change-code skills. User scope overrides skill defaults: do not read PRD, skeleton, CLI, frontend or unrelated architecture references. db/ and docs/ are read-only authorities. All implementation, tests, scripts, artifacts and logs stay under backend/. This committed specification and the top status of agent/backend-refactor-plan.md are authorized exceptions. No root scripts, root AGENTS.md, dependencies, schema changes or Go-managed transactions. Append the prescribed change-code implementation log to backend/implementation-log.md, excluding that log from line counts. The implementation child must not commit/push; the factory owns those steps.

Coverage >95% unit and >=90% APIHydra production statement counts are FINAL refactor goals. Measure both suites honestly on each pass and retain strict failing exits when below target. A genuine intermediate shortfall does not stop implementation/review/merge. No coverage-driven deletion, production exclusion, artificial tests, profile mixing or weakened assertions. Repair introduced failures; leave unrelated formatting/lint debt for R5 and report it. No stage or production promotion.

## Mandatory fresh audit before any deletion

This specification was prepared while P3/P4 were pending. Its candidates are hypotheses to verify against the final merged P4 tree, not instructions to delete declarations merely because an earlier snapshot looked unused. Inspect status and preserve unrelated work. Record the actual base SHA and current symbols/callers before editing. Use references/imports, route registration, build variants/tags, generators, reflection/dynamic use and relevant documented backend entrypoints to distinguish dead code from supported behavior. Inspect all backend production and tests; application package dependency information from `go list` is useful supporting evidence. Do not assume a name search alone proves an exported API dead.

Record each verified removal and the supporting evidence in the existing backend checkpoint/contract ledger; no extra permanent audit document is needed. Candidates already removed by P3/P4 are completed migrations, not new R1 changes. If no actionable candidates remain, report an explicit no-action result so the supervisor skips an empty implementation branch; do not manufacture work to satisfy the code-spec script's changed-files requirement.

## Bounded candidates

1. Private `API.e *echo.Echo` fields in business/health API structs that are assigned in NewAPI and never read. Stable P2 project/epic/health modules had this pattern; recheck change/testcase after their migrations. Remove only the unread field and its assignment. Preserve the constructor Echo parameter, service injection, every route and HTTP behavior, including the direct `/api/health` alias. `API.g` is actually used during registration: do not label it dead or rewrite it in R1 merely to store less state.

2. The obsolete commented config debug print referencing `config.Data()["db-ws"]`, if still present in pkg/config/config.go. Remove this dead comment without inventing a test for comment removal. The global config design remains R4 scope.

3. `pkg/db.Pool` is a candidate obsolete wrapper, not an automatic deletion. The inspected server calls pgxpool.New directly and owns returned errors/resources; pkg/db was unlinked and referenced only by its own tests. Remove the wrapper/package and tests specific solely to it only if the fresh audit establishes no live importer, registered use, build-tag variant, generator/linkname/reflection use or documented supported backend consumer. Preserve tests for retained startup/database behavior. If any supported consumer remains, leave it and record why. Do not introduce a new main import merely to cover its statements; do not filter the retained package from the coverage denominator. Its four uncovered integration statements are not evidence for removal.

Any additional candidate must be directly evidenced as unused/unreachable/obsolete under the same proof standard and remain small within this dead-code category. In particular, do not remove defensive errors, legitimate error branches, independent SQL operations, parser/sanitizer interfaces, repository pool test seams or lifecycle seams because they are hard to integrate-test. Do not remove or change routes, DTO fields, current document/history retention, request validation, constructor side effects or supported configuration behavior. If a candidate needs a behavioral/API decision, leave it for a separately recorded follow-up instead of widening R1.

## Acceptance criteria

R1-01 Evidence-backed removal: every removed declaration/package/comment is backed by the fresh audit and classified above. No active P4 behavior or unknown exported consumer is removed. Record already-resolved and retained candidates explicitly. Existing relevant unit tests, import/build checks and reference evidence prove retained startup/API behavior; removal-only comments do not require empty tests.

R1-02 HTTP parity: retained API constructors still register exactly the current merged-P4 method/path inventory, including health aliases and explicit project config/change-document operations. Service dependencies, error/status/JSON contracts, ID-only201, empty204, request contexts and middleware behavior remain unchanged. Run existing meaningful API constructor/handler tests and the full APIHydra suite. If a gap in route parity is discovered, add a real unit assertion of the retained registered routes; do not add a test that merely asserts an unused field was deleted. Update the inventory audit only if needed to keep it accurately observing unchanged registrations, never to relax the denominator.

R1-03 Startup/config/database parity: preserve the normal server resource ownership, pgx errors and causes, pool/listener cleanup, config/default/environment/flag precedence, timeouts, normal shutdown and centralized errors. If the unused pkg/db wrapper is removed, map existing retained startup failure/lifecycle/config unit tests to this criterion and retain those assertions. If it remains, keep its legitimate unit tests and include it honestly in coverage. No new dependency, transaction, global lookup, connector or runtime test switch is introduced.

R1-04 Honest verification/scope: run the required final commands, fresh coverage inventories and APIHydra campaign; report actual command exits, covered/total counts, package gaps and any known baseline issues. Existing relevant unit tests may satisfy unchanged behavior criteria; add tests only for a missing meaningful acceptance assertion, not coverage-free removal trivia. No unrelated formatting sweep, cleanup category mixing, test weakening or denominator manipulation. Only allowed paths are changed.

R1-05 Compact handoff: refresh the backend checkpoint with removal evidence, preserved contract/test mapping, final review/check/coverage state and the next exact action. Update only the top status of the official plan. Explicitly hand off R2 duplicate-path reassessment after P3/P4/R1; a supported result is "no actionable duplicate paths remain after contract alignment," with inspected evidence. Do not create speculative generic repositories, helpers or a shared fake framework just to ensure R2 has a commit. This criterion is supported by the retained parity/coverage checks and documented audit; do not invent a unit test of prose.

## Validation and delivery

Run focused retained tests for touched packages first; apply pinned formatting only to touched Go files and inspect its diff. Run these commands from repository root on the final implementation, including review fixes:

```sh
make -C backend check
make -C backend coverage
make -C backend deps-audit
make -C backend api-test
```

If check stops on known unrelated baseline formatting, additionally run `GOLANGCI_LINT_CACHE=/tmp/mch-r1-lint-cache make -k -C backend check` to expose all check components. Run tooling-test when tooling/audits change; check already includes it. Keep APIHydra under backend/apih-tests, all scenarios against the owned disposable DB/server, and installed-tool limitations documented. Tests and coverage must remain separate: do not count legacy Go HTTP profiles as APIHydra execution. Run git diff --check and inspect changed/untracked paths. Store logs under /tmp/mch-r1* or backend/.coverage/verification; fresh reports under existing backend/.coverage paths.

The supervisor's factory sequence is: author/commit this spec on the correctly named branch from fresh origin/dev; run `scripts/codex-code-spec.pl agent/specs/010-backend-dead-code.md`; inspect the implementation; run `scripts/codex-review-loop.pl agent/specs/010-backend-dead-code.md --base origin/dev`; repair valid findings and rerun applicable final checks; publish the compact checkpoint; invoke `scripts/merge-to-dev.pl`; verify clean local/remote dev and continue to the R2 reassessment. Never retry a partial Git/merge action blindly. Inspect state first; transient tool failures may be retried at most twice, while code/test failures require repairs. No promotion beyond dev.
