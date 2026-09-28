# 019 — CLI verification foundation

## Authorized execution and scope

The user authorized the entire sequential CLI rebuild factory in
agent/cli-rebuild-plan.md. This specification is P0 only, on
change/019-cli-verification-foundation, based on origin/dev 178b8b0. Implement
through the existing code-spec helper; it owns commits/pushes. Do not invoke
another factory, commit, push, merge, or promote from the implementation child.

Read AGENTS.md, the CLI plan, docs/cli-architecture.md, this spec, and relevant
existing CLI implementation/tests. User scope overrides skill defaults: do not
read PRD or invent the missing skeleton/ contracts. Actual CLI architecture and
backend handlers/types are the contracts. The implementation log is
cli/implementation-log.md, not repository-root implementation-log.md. Use the
change-code log format there. Preserve dependencies and production behavior.

Allowed writes: cli/; a new CLI completion-policy section in root AGENTS.md;
the status/execution notes in agent/cli-rebuild-plan.md; this specification only
for an evidence-backed clarification that does not weaken acceptance. Backend,
db, frontend, cli-proto, root scripts, .mch, other docs and dependency versions
are out of scope. Read backend/Makefile and backend/scripts as tooling references.
Use existing dependencies and installed pinned tools where appropriate.

This is verification tooling and useful test-infrastructure repair. Do not
implement the Flow removal, backend migration, feature screens or agent workflow.
Do not reformat all production code simply to make the baseline green. Record
pre-existing format/lint/application gaps accurately for their owning passes;
repair any new failures introduced by this change.

## Goal and observed baseline

Make CLI validation select the real module, preserve existing useful tests, and
produce trustworthy independent unit and terminal-program statement coverage.
Existing PKG=mch discovers no packages; lint modifies files; setup uses unpinned
tools; Docker defaults to Go 1.25 despite go.mod 1.26.0. Integration includes
complete-program tests, Flow script tests, a stale import-boundary test, and a
PTY test that currently builds an uninstrumented child and skips without socat.

Installed socat, codex, golangci-lint and govulncheck were found by the supervisor.
No application test or coverage baseline has yet been established. Run and
record it as part of this pass. Do not claim a dry run is passing validation.

## Acceptance criteria and test mapping

- P0-01 Package inventory and Make interface: discover/include every production
  Go package under cmd/, internal/, pkg/, and audit for production code elsewhere.
  Packages without tests remain in coverage denominators. Test-only integration
  packages and tooling are not production. Fail visibly on package-discovery
  errors and empty discovery. Bare make displays help. Add documented phony
  init, format, format-check, lint, vet, test, race, check, coverage,
  coverage-html, integration-test, terminal-test, integration-coverage,
  deps-audit, tooling-test, benchmark and test_version targets. Use uncached
  runs; check includes read-only formatting, lint, vet, unit race, architecture,
  and isolated tooling verification. Static checks include test source.
  Meaningful tooling tests assert target selection, failures, empty inventories,
  non-mutating checks, and no duplicate integration run inside check.
- P0-02 Safe tooling setup: pin lint/formatter and vulnerability tool versions,
  separate explicit formatting from checks, and preserve application dependencies.
  Derive the Docker Go version from go.mod unless overridden; quote checkout paths
  and work without a TTY. Follow proven backend patterns, without inheriting
  backend/API lifecycle behavior. Test recipe behavior and explicit formatting
  versus read-only checks with isolated tools. Report real installed-tool results.
- P0-03 Independent unit measurement: execute only unit tests, with race detection,
  across the complete production inventory, including cmd and packages with no
  test files. Produce fresh profiles, exact covered/total counts, per-package
  results and useful uncovered-function/source reports. Enforce strict >95%
  through integer statement counts before display rounding. Exactly 95% fails.
  Preserve a valid below-target profile for diagnosis; remove stale success
  artifacts before execution and never accept failed/incomplete runs. Test
  exact-boundary arithmetic, missing-package accounting, duplicate blocks,
  empty/malformed profiles and test-failure preservation with known fixtures.
- P0-04 Terminal integration eligibility: reuse existing complete-program
  RunProgramWithIO tests with injected I/O/fake backend/agent collaborators and
  the real PTY suite. The required integration result combines these eligible
  executions only. Standalone HTTP-client/adapter tests, direct controller tests,
  architecture checks and Flow shell-script tests are not eligible counters.
  Keep their useful behavioral checks separately runnable; do not drop existing
  scenarios merely because they do not count. Make the eligible scenario manifest
  explicit and auditable, reject an empty or unmatched selection, and document
  why any existing integration-named test is outside the measured campaign.
  Tests verify selection and prevent non-program counters from entering results.
- P0-05 Instrument the real child: reuse the PTY harness but allow the campaign
  to supply its freshly built covered mch executable and a private counters
  directory. Standalone terminal-test still works. Drive an orderly application
  exit after assertions so counters flush; use bounded owned-process cleanup on
  failures. Do not stop unrelated processes. Real PTY execution remains required
  for a complete campaign; missing socat or unexpected skips must fail the gate
  visibly, not produce a passing reduced suite. Unit/tooling tests exercise child
  build selection, missing tools/counters, timeout/cleanup failures and ownership;
  run the actual terminal test to demonstrate instrumentation works.
- P0-06 Coverage aggregation: instrument all CLI production packages for both
  in-process complete-program tests and child executables, using compatible
  Go coverage modes. Normalize blocks by statement identity and union coverage
  hits; never sum repeated statements or average percentages. Compare emitted
  metadata with the independent complete production inventory, accounting for
  unlinked/unexecuted packages without importing unit hits. Enforce strictly
  >90%, so exactly 90% fails. Exclude only harness/dependency/backend code. Test
  overlapping profiles, untouched cmd packages, mismatched statement inventories,
  malformed data, strict thresholds and the separation of unit/integration inputs.
- P0-07 Honest run lifecycle: isolate every campaign's fresh artifacts and record
  revision/source provenance, binary hash where applicable, Go/tool versions,
  scenario inventory, subprocess/test exits, raw profiles and package totals.
  An assertion failure, crash, missing counters, required skip, interrupted run
  or cleanup failure cannot establish passing coverage. Preserve original failure
  exits even if cleanup succeeds; retain diagnostics without stale success reports.
  Keep outputs under cli/ or temporary storage and ignore generated files locally.
  Tooling tests prove failure propagation, stale-artifact handling and cleanup.
- P0-08 Architecture enforcement: repair the real cli/internal/ prefix and shared
  dto ownership in the existing checker. Enforce target dependencies relevant to
  current packages: features cannot import app/concrete adapters/other feature
  implementations; shared packages cannot import higher layers; adapters cannot
  import app/features. Prefer actual module imports over stale hardcoded prefixes.
  A deliberate forbidden-import fixture must fail, and allowed fixtures pass.
  Keep this structural check separate from terminal coverage. Record any existing
  violations without silently permitting forbidden edges.
- P0-09 Durable evidence and reuse ledger: create cli/agents/cli-contracts.md and
  cli/agents/cli-rebuild-checkpoint.md. Map architecture/API actions and existing
  tests to retained, revised, replaced or removed assertions and future passes.
  Record the current route inventory from backend registrations; no backend edits.
  Reuse program/PTY synchronization and meaningful assertions. Do not remove
  obsolete Flow tests yet: their removal belongs to P1. Update the plan status
  from planning-only to authorized execution, with this pass's actual results
  and next P1 action. Add canonical CLI commands/completion rules to AGENTS.md
  without changing backend policy. Verify doc links and documented targets.

## Coverage and intermediate merge policy

Final goals are strictly >95% unit and >90% integration production statements.
The user explicitly allows intermediate dev merges with passing tests and valid,
honestly reported numerical shortfalls. Keep strict gates returning failures;
do not lower targets, omit packages, merge unit hits into integration, delete
defensive production behavior, introduce test-only production execution paths,
or add artificial tests. P0 is not expected to complete the feature rebuild or
reach final coverage. Baseline format/lint issues are tracked for repair during
the rebuild; new regressions require repair. A failed/incomplete suite has no
passing coverage result and must remain visibly incomplete.

## Verification and handoff

Run baseline useful tests, focused new tooling/architecture tests, then these
real targets on the final tooling revision, preserving every exit result:

```sh
make -C cli tooling-test
make -C cli check
make -C cli coverage
make -C cli deps-audit
make -C cli integration-coverage
```

Run retained integration checks excluded from the measured campaign through
integration-test as needed; document what each target actually executes. Reuse
campaign evidence rather than unnecessarily rerunning the same PTY scenario.
No live backend or database is required or authorized for P0; existing fake
servers and local subprocesses are the integration collaborators. Run Docker
only if available and warranted; report unexecuted compatibility checks.

Record exact covered/total production statements, per-package gaps, baseline
issues, passing/failing/skipped scenarios, artifact paths, and every command exit.
Update the implementation log and compact checkpoint. Supervisor then inspects
the diff/results, runs native review against origin/dev, revalidates review fixes,
merges the bounded change to dev, and continues with P1. No stage/prod promotion.

## Notes

Only tooling/helper tests are needed for tooling acceptance; do not invent
production APIs or duplicate application tests to satisfy documentation bullets.
The full-rebuild prompt/resource decisions are recorded in the plan and apply to
later passes; this pass leaves .mch/ unchanged.
