# 014 — Backend local patterns

## Authorized execution

Implement this specification after R4 merges to dev. Expected branch: `change/014-backend-local-patterns`; specification: `agent/specs/014-backend-local-patterns.md`. The supervisor confirms a clean checkout and fresh origin/dev before creating the branch. The user authorized the full sequential backend refactor factory. R5 repairs concrete remaining local lint/format conventions; it is not a behavior/API migration or a general documentation rewrite.

Read the current AGENTS.md, docs/backend-architecture.md, this specification, official plan, backend contract ledger and latest checkpoint. Apply change-refactor and change-code skills. User scope overrides skill defaults: do not read PRD, skeleton, CLI, frontend or unrelated architecture references. Implementation, tests, fixtures, artifacts and logs remain under backend/. This committed specification and the top status of agent/backend-refactor-plan.md are authorized exceptions. No edits to root scripts, root AGENTS.md, db/, docs/, dependencies or tool thresholds/configuration. Append the prescribed change-code log to backend/implementation-log.md, excluding it from counts. Factory owns commits/pushes; implementation child must not perform them. No stage/production promotion.

Unit >95% and APIHydra >=90% production statement coverage are FINAL refactor goals, not intermediate blockers. Run and measure both suites honestly; strict numerical gate failures remain visible but do not stop this pass. No artificial tests, excluded production packages, defensive-code removal, profile mixing or assertions weakened for coverage. Repair all introduced failures. This pass should finish with `make -C backend check` green after the previously tracked baseline issues are resolved; a still-failing check must be diagnosed and reported, never hidden behind the numerical-coverage exception.

## Mandatory fresh post-R4 audit

The known debt was observed before P4/R1/R3/R4 and may already be gone. Capture current `format-check` and full lint/check output before editing. The normal linter caps repeated messages: enumerate the complete inventory using its unchanged selection plus `--max-same-issues=0 --max-issues-per-linter=0` and a writable cache; 11 displayed findings are not necessarily all unique debt. Reconcile each actual finding with the checkpoint instead of applying a stale list. Candidate remaining issues were:

- unchecked response Body.Close errors in backend/api-tests/shared/client.go NewClient/Get/Post;
- missing concise exported comments on the shared legacy HTTP test client and its public helpers;
- missing package comments in domain/markdown where still reported (P3 already added a domain package comment—do not duplicate it across files);
- formatting differences remaining after earlier touched-file formatting.

Only fix actual current findings or directly relevant inaccurate comments in touched symbols. No broad conversion of all "executes behavior" comments, naming churn, blanket comment additions to already documented packages, service/repository refactor, generic helper framework or dependency/tool update. If the audit exposes a concrete leftover architecture violation, establish whether it is a small directly related R5 issue or record a distinct follow-up; do not silently widen scope. Recheck known Go-managed transactions, prohibited service imports and centralized error ownership using existing audits, but do not invent new business semantics.

## Acceptance criteria

R5-01 Baseline reconciliation and clean check: record exact current lint/format findings and their owners before editing. Fix legitimate remaining backend formatting, errcheck and exported/package-documentation findings under the unchanged pinned Makefile checks. Final `make -C backend check` succeeds with lint, vet, uncached race tests and tooling checks enabled. No nolint directives, disabled checks, ignored errors, relaxed thresholds, excluded files/packages or swallowed exit codes. If an external blocker prevents a valid run, report its actual evidence rather than claiming green; coverage shortfall alone is not such a blocker.

R5-02 HTTP body cleanup: every successful response acquired by shared NewClient/Get/Post is closed exactly once on all subsequent paths, including response decode/read/assertion failures. Close failures must mark the test failed with useful method/path or health-probe context; do not silence them with `_ =`, ignore a returned error, or merely log without test failure. Keep the original request/status/body assertions and decode diagnostics. Prefer nonfatal error reporting during deferred cleanup so a close error does not discard the primary failure or prevent other registered cleanup from running. Do not lose an owned response on early return. Preserve client timeout, base-URL environment override, payload/status behavior, existing cleanup ordering and isolated-server ownership.

R5-03 Meaningful failure verification: retain existing useful legacy HTTP scenarios and run the complete migrated legacy HTTP suite against its owned disposable cluster/server using the existing legacy target. It supplies harness validation, never APIHydra coverage. Where body-close handling introduces a helper or changes control flow, add a focused harness unit test proving a genuine failing Close records test failure and useful context and that ordinary cleanup closes once. A narrow private helper using the existing testify testing interface plus a recording test double may suffice; do not build a generic testing framework or alter public Client behavior just to inject coverage. If choosing simple deferred assertions without a helper, provide equivalent honest failure-propagation evidence. Tests for shared harness code remain outside backend production coverage and must not be relabeled to increase the percentage. Preserve real test failures and primary assertions when cleanup also fails.

R5-04 Documentation/formatting accuracy: concise exported comments describe actual responsibilities; package comments occur in the appropriate single package source location and match the package name. Formatting uses the pinned tools on actual affected files and changes no runtime behavior. Do not write tests that assert comments exist or formatting text: formatter/linter and retained behavioral tests are the meaningful evidence for these mechanical fixes. Do not edit historical docs or unrelated code just to make prose consistent.

R5-05 Contract/scope parity: production API/domain/SQL behavior, status/envelope shapes, ID-only201/empty204, selected configuration, retained append-only documents, ordering/nullability, resource ownership and centralized errors remain unchanged. Run the existing meaningful unit suite and full APIHydra campaign. No owned-PostgreSQL outage campaign, new server test hook, broader fixture migration, startup/config redesign or R6 compaction belongs to this pass. Any legitimate coverage gain comes from preserved behavior tested through the established suites, not denominator games.

R5-06 Compact delivery: update the backend checkpoint with actual repaired findings, retained behavior/test mapping, clean check result, separate unit/APIHydra covered/total counts and package gaps, legacy harness run, native review status and next R6 action. Update only the top official-plan status. Document still-unmet final numerical targets honestly without stopping the agreed intermediate merge. No additional prose-only unit tests are required.

## Implementation guidance

The inspected shared client performs an initial health request and GET/POST helpers with `defer res.Body.Close()`; each owns its response. Handling the three errors at their ownership points is the goal. A tiny local close-reporting helper is acceptable only if it makes those repeated ownership/error semantics clearer and meaningfully testable; use existing dependencies. Do not move HTTP transport code into production packages or impose a new generic client abstraction. Keep t.Helper diagnostics on the caller where applicable.

The current test client cleanup and all post-P4 retained scenarios must continue to respect explicit child/parent deletion contracts. Do not restore old cascades, swallow409 cleanup errors or accept arbitrary statuses merely so the legacy suite passes. If cleanup fixture assumptions are now obsolete, first establish the retained contract and fix the harness within scope without changing the backend API. Existing correct assertions remain intact.

Format only files with current formatting debt or semantic/comment edits. Because this pass explicitly clears known baseline formatting, the scope may include previously untouched backend files with demonstrated formatter diffs; inspect each diff for mechanical-only changes. No CLI/frontend/root formatting.

## Validation and factory workflow

Run the relevant shared harness unit checks when added, formatting/lint checks, then on final code and after any review fix:

```sh
make -C backend check
make -C backend coverage
make -C backend deps-audit
make -C backend api-test
make -C backend legacy-api-test
```

Use `GOLANGCI_LINT_CACHE=/tmp/mch-r5-lint-cache` if the environment requires a writable lint cache. The final check must be a normal successful `make check`; `make -k check` can diagnose multiple failures during repair but does not excuse any. Keep legacy and APIHydra profiles separate. APIHydra remains under backend/apih-tests and uses the installed tool with existing honest assertions and owned DB/server lifecycle. Fresh reports use the unchanged structural denominator and thresholds. Run git diff --check and inspect tracked/untracked file scope. Logs belong under /tmp/mch-r5* or backend/.coverage/verification.

Supervisor sequence: author/commit this spec on the matching014 branch from fresh origin/dev; invoke `scripts/codex-code-spec.pl agent/specs/014-backend-local-patterns.md`; inspect implementation and evidence; run `scripts/codex-review-loop.pl agent/specs/014-backend-local-patterns.md --base origin/dev`; repair valid findings and revalidate; publish compact checkpoint; run `scripts/merge-to-dev.pl`; verify clean local/remote dev; continue to R6 audit. Inspect state before retrying partial tool/Git actions; transient failures may be retried at most twice, code/test failures require repair. No promotion beyond dev.

Confirmed fresh R4 base: origin/dev `d35ea6852fecc0d6a518bdb37ab188ffa127e2a7`. Supervisor baseline logs `/tmp/mch-r5-baseline-uncapped.log` and `/tmp/mch-r5-baseline-format.log` record actual full findings; format passes. Use those and verify current source before editing. Keep the final normal Makefile lint/check command unchanged.
