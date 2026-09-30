# Repository Instructions

Follow these instructions when working on code in this repository.

## Authorized CLI rebuild factory

The user authorized execution of `agent/cli-rebuild-plan.md`, including sequential
specification, implementation, review, and dev-merge passes. For CLI rebuild
specifications, the approved user scope explicitly overrides generic skill
defaults: use `docs/cli-architecture.md`, current backend HTTP contracts, and the
CLI plan rather than missing `skeleton/` or unrelated PRD references. Write the
change-code/change-fix-findings implementation log to
`cli/implementation-log.md`, using the skill's block format; do not create a
repository-root implementation log. This is an authorized exception, not an
unresolved contract conflict requiring another user confirmation. The factory
helpers own implementation/review commits and pushes; implementation children
must not invoke another factory or perform Git publication themselves.

Final CLI coverage targets are at least 80% unit and 70% terminal integration
production statements. Integration combines complete-program tests using injected
I/O and fake collaborators with real PTY tests; HTTP-client-only tests do not
count. Keep separate profiles and strict gates, but continue intermediate dev
merges with passing tests and honestly reported numerical shortfalls. Record
baseline issues and repair new regressions. Do not promote to stage or production.

## Development Commands

Backend commands belong to `backend/Makefile`. From the repository root, use
`make -C backend <target>`; root-level Make targets do not run backend validation.
From inside `backend/`, omit `-C backend`.

| Command from repository root | Purpose |
| --- | --- |
| `make -C backend init` | Install pinned golangci-lint and govulncheck tools; setup only. |
| `make -C backend format` | Apply gofumpt/goimports formatting; explicitly modifies Go files. |
| `make -C backend format-check` | Check formatting without modifying files. |
| `make -C backend lint` | Run standard golangci-lint checks plus revive, including staticcheck. |
| `make -C backend vet` | Vet backend Go packages. |
| `make -C backend test` | Run uncached short unit tests. |
| `make -C backend race` | Run uncached short unit tests with the race detector. |
| `make -C backend tooling-test` | Run isolated tests for backend build/test tooling. |
| `make -C backend check` | Run format-check, lint, vet, race, and tooling-test. |
| `make -C backend coverage` | Run unit tests with race detection and report production statement coverage. |
| `make -C backend coverage-html` | Rerun coverage and generate an HTML report for investigating gaps. |
| `make -C backend deps-audit` | Run govulncheck; requires access to the vulnerability database. |
| `make -C backend api-test` | Run APIHydra on an instrumented backend using the existing database; report statement coverage. |
| `make -C backend benchmark` | Run benchmarks without ordinary/integration tests. |
| `make -C backend test_version` | Run checks in Docker using the Go version from go.mod; override with `goversion=X`. |
| `make -C backend help` | Show the current target list; default when no target is supplied. |

## Backend Definition of Done

For backend production-code changes, run all of these commands on the final
implementation, including any review fixes:

```sh
make -C backend check
make -C backend coverage
make -C backend deps-audit
make -C backend api-test
```

Successful commands are necessary but not sufficient:

- Every acceptance criterion, including each individual bullet, must be
  implemented and have a meaningful unit test. Verify architecture and database
  contracts as well as test results.
- Unit statement coverage must be **at least 95%** across all backend
  production Go packages (`cmd`, `internal`, and `pkg`), including packages with
  no tests. Report package-level gaps as well as the aggregate. Evaluate actual
  statement counts, not a rounded percentage or an average of package percentages.
- API integration tests must use **APIHydra (`apih`)**, with suites under
  `backend/apih-tests/`. A plain `apih` invocation there must pass against a
  configured, running development backend. Setup, assertions and cleanup use
  endpoints and captured response values only: no direct database access, SQL
  fixtures, forced outages or database lifecycle management. Plain `apih` uses
  the configured running server; `make api-test` may start and stop its own
  instrumented backend against the existing development database to collect
  coverage. It must not stop another server. Test-created records are isolated
  from existing data.
  API statement coverage must be **at least 90%** across all backend production
  Go packages. The Make target reports fresh measured statements after normal
  APIHydra output and a blank line. Bare `apih` against an uninstrumented server
  does not measure coverage; endpoint coverage is a separate metric. Never merge
  unit profiles into the API result or report failed/incomplete runs as passing.
- Record the commands actually run, their exit results, measured coverage
  totals (or explicitly unmeasured API coverage), and any failing, skipped, or
  blocked scenarios. Use fresh coverage data
  for the tested revision; failed or incomplete runs cannot establish a passing
  coverage result. Keep generated artifacts under `backend/` or temporary storage.
- Run API tests only against a user-designated development/test server. Never
  mutate unrelated records. `import-db` resets data and is not an API test step
  or routine completion check.
- Missing tools, unavailable services, vulnerability findings, compiler errors,
  and unmet thresholds must remain visible. Do not suppress failures, weaken
  assertions, omit difficult production code, or add artificial tests to make
  validation pass. Finish the safe, reviewable work and discuss genuine blockers
  with the user; describe the result as incomplete rather than declaring it done.

**Refactor execution policy (user clarification, 2026-09-28):** the numerical
coverage targets apply to the final refactor result, not to each intermediate
branch. Implement meaningful tests and run both suites in every pass; measure
unit and API statement coverage when available; continue the agreed
specification/review/merge sequence when a valid measurement is below target.
Make coverage gates must still return failure honestly. Aim for at least 95%
unit and 90% API integration at the end. If legitimate testing falls short,
finish the remaining work and report actual counts, uncovered behavior and
options for discussion. A coverage shortfall alone does not stop implementation
or merging to dev. This is not permission to hide failing tests, introduce
regressions, manipulate coverage or claim unmet targets passed. Known baseline
format/lint issues are tracked for repair in the relevant passes; new failures
must be repaired. Do not promote to stage or production.

**Current migration status:** `coverage` retains its strict unit statement-count
gate. `api-test` instruments an owned backend (default port19080) against the
existing `DATABASE_URL`, runs a private copy of the same standalone suite, and
enforces the 90% integration gate. Manual `apih` uses the unchanged
`backend/apih-tests/root.yaml` URL (default port8080). SQL fixtures, outage/recovery
suites and database lifecycle management remain removed, as do the Go HTTP
harness and `legacy-api-test` target. APIHydra mode1 runs endpoint groups in
parallel and each group's numbered init/main/post files serially. See
`agent/backend-refactor-plan.md` and the backend checkpoint for current results.

`check` already includes formatting, lint, vet, race tests, and tooling tests;
there is no need to run each constituent again without a new change or failure.
Use `format` when needed before verification and inspect its diff. Run benchmarks
when a change affects a performance contract, and Docker checks when changing
toolchain compatibility; neither replaces the required commands above.

For documentation-only changes, verify the documented commands and links without
rerunning unrelated application suites. For changes confined to backend build or
test tooling, run `tooling-test` plus the affected real targets where feasible.
Report pre-existing application failures separately: passing tooling tests proves
the tooling change, not overall backend readiness. These scoped checks do not
waive the production-code completion requirements.

Keep the completion policy here, executable checks in `backend/Makefile` and
`backend/scripts/`, and iteration-specific progress in the refactor plan or
backend checkpoint. Keep these rules and the Make targets in sync when tooling
changes; do not maintain conflicting copies in each specification.

## Unit Test Requirements

- Implement and maintain at least one unit test for every acceptance criterion,
  including every individual acceptance criteria bullet.
- Write unit tests for all production code and keep unit test coverage at least
  95%.
- Avoid tests that do not increase coverage unless they prove a specific
  acceptance criterion bullet.

## Architecture

### Backend

The backend follows Screaming Architecture:

- `pkg/*` contains loaders and abstractions.
- Follow `docs/backend-architecture.md` for the target backend architecture.
- Business modules under `internal/*` use three layers. `domain`, `error`, and
  `health` are exceptions:

  1. **API (`api`)** imports `echo`. Its methods should contain only code that
     requires `echo` or API request validation using `validate`; move business
     logic to the service layer.
  2. **Service (`service`)** contains business logic and must never import
     `echo`, `validate`, or `pgx`.
  3. **Repository (`repo`)** imports `pgx` and contains only code that cannot
     work without `pgx`.

## CLI commands and completion policy

CLI commands belong to `cli/Makefile`; use `make -C cli <target>` from the root.
Bare `make -C cli` shows help. `init` installs pinned tools. `format` explicitly
writes source; `format-check`, `lint`, and `vet` include test source and are
read-only. `test` and `race` run uncached unit tests over the audited production
inventory; `benchmark` runs benchmarks only. `check` runs formatting, lint, vet,
unit race, architecture, and isolated tooling checks, retaining failures while
continuing the other checks. It does not run the terminal campaign.

For CLI production changes and review fixes, run:

```sh
make -C cli check
make -C cli coverage
make -C cli deps-audit
make -C cli integration-coverage
```

`coverage-html` adds HTML to the unit campaign, including valid below-target
results. `integration-test` retains complete-program, standalone startup, and
legacy Flow checks outside coverage; it excludes architecture checks.
`terminal-test` runs the real PTY scenario plus its harness tests. Missing socat
fails visibly. `integration-coverage` selects the explicit manifest at
`cli/scripts/terminal-scenarios.json`: complete-program tests plus the covered
startup child and real PTY child. Flow script tests, architecture, harness unit
tests and direct adapter tests contribute no terminal counters. Use the campaign
as evidence for its constituent tests without repeating the same PTY run.
`architecture` and `tooling-test` are independently runnable. `test_version` uses
the go.mod Go version in Docker without a TTY; override with `goversion=X`.

Keep unit and terminal profiles independent. Both denominators contain all
production statements under cmd/internal/pkg and any additional production
packages discovered by the inventory audit, including packages without tests.
Only integration and scripts are classified as harness/tooling trees. The final
gates use integer counts: `100*covered >= 80*total` for unit and
`100*covered >= 70*total` for terminal integration.
A campaign must finish all required scenarios without skips, missing counters,
assertion failures, crashes, timeouts or cleanup failures to establish coverage.
A package with no tests is not a skipped unit scenario and still counts in the
denominator. Preserve valid below-target profiles and raw failed-run diagnostics;
never present incomplete runs as passing or combine unit hits with terminal hits.

Artifacts live in ignored `cli/.coverage/unit` and `cli/.coverage/integration`;
each campaign replaces its previous artifacts under an exclusive lock. Record
revision/source hashes, tool versions, commands/exits, scenarios, child binary
hash, raw profiles, exact package totals and uncovered statements/functions.
Use `cli/agents/cli-rebuild-checkpoint.md` for current evidence and
`cli/agents/cli-contracts.md` for the assertion reuse/API ledger. Implementation
blocks belong to `cli/implementation-log.md`. For tooling-only passes, run
`tooling-test` and affected real targets. Do not mutate production source merely
to make an existing format/lint baseline green; track those issues by owning pass.

The authorized intermediate-merge policy above applies: numerical shortfalls
alone may continue to dev with passing tests and honest gates. Repair new
regressions; expose pre-existing failures. Final rebuild completion still requires
all contracts, tests, checks, at least 80% unit and 70% terminal coverage. Do not promote
to stage or production. No live backend/database use is implied by CLI tests;
P0 uses fake servers and owned local processes only.

Frontend coverage targets are to be decided; no numerical gate is set yet.
