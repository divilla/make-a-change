# Backend refactor plan

Status: P0–P4, R1, the separate011 validation-cause repair, R3 startup boundaries and R4 configuration isolation are implemented and reviewed; R2 remains no actionable work. R4 preserves408 APIHydra requests/34 operations and full legacy HTTP/SQL parity. Both gates pass: unit925/939 (98.5091%), integration851/939 (90.6283%). Required check remains incomplete for displayed baseline lint diagnostics; R5 will enumerate uncapped findings and repair them without weakening checks. Next are014 local patterns (R5),015 compact-handler reassessment (R6) and separate016 final failure-integration verification. See [the current checkpoint](../backend/agents/backend-refactor-checkpoint.md).
Inspected on 2026-09-28 at commit `7a89905`, branch `change/004-refactor-backend`.

## Recommendation and scope

Treat this as two kinds of work: first restore a working backend aligned with the current database and architecture; then conduct six separate behavior-preserving cleanup iterations. The current backend does not compile, and several queries and tests describe the previous database. A single large “refactor” would obscure necessary contract changes and make regressions difficult to identify.

Use small, sequential branches with tests delivered alongside each change. Start APIHydra and coverage infrastructure early; do not postpone integration testing until the final branch. Complete one reviewable pass before starting the next. No parallel agent work is needed for the initial shared-contract changes.

- All implementation, tests, scripts, fixtures, reports, local ignore rules, and backend documentation must stay under `backend/`.
- The official plan is `agent/backend-refactor-plan.md`. This file and future `agent/specs/NNN-slug.md` specifications are the authorized documentation exceptions. The user also authorized updating root `AGENTS.md` with backend completion rules in this preparation branch. Do not change root-level scripts, `docs/`, `.github/`, or create root-level `apih-tests/`.
- Ignore the CLI, frontend, and `prd.md`. Do not update their consumers to compensate for backend changes.
- Read `docs/backend-architecture.md`, `db/init.sql`, and `db/seed.sql` as authorities; do not edit them. A database defect or missing capability that cannot be resolved in the backend becomes a reported blocker.
- No framework migration, dependency upgrade, new dependency, speculative API, compatibility placeholder, or new Go-managed transaction is included.
- Preserve retained behavior. Explicit architecture-required API changes belong in the contract-alignment branches, not hidden inside cleanup commits.

## Preparation on change/004-refactor-backend

The user authorized the Makefile modernization before the specification sequence. The supplied `/home/vito/apihydra/Makefile` path was absent; the user confirmed `/home/vito/go/src/apihydra/Makefile` as the reference, with improvements where appropriate.

Implemented in `backend/Makefile` and its backend-local helpers:

- Explicit production package patterns replace the stale module discovery. Formatting/lint/vet include the legacy HTTP harness; unit/race/coverage commands exclude that test-only harness and use short mode to avoid DB integration tests.
- `make init` installs pinned golangci-lint v2.13.1 and govulncheck v1.7.0 without changing application dependencies. `make format` explicitly applies gofumpt/goimports; `make format-check` reports differences without writing. `make lint` runs standard golangci-lint checks plus revive, including staticcheck. These replace the older standalone lint pipeline using the [supported formatters](https://golangci-lint.run/docs/formatters/) and [linters](https://golangci-lint.run/docs/linters/).
- Bare `make` displays help without running checks or installing tools. `make check` runs formatting checks, lint, vet, short race tests and isolated tooling tests. It does not reformat code, clear the global test cache, reset a DB, or run the legacy HTTP suite. `make deps-audit` is a separate network-dependent vulnerability check.
- `make coverage` and `make coverage-html` produce fresh unit profiles/reports across all production packages. These are measurement targets; numeric coverage enforcement and APIHydra instrumentation remain P0 work. A build/test failure stops report generation and is not accepted as a coverage result.
- All action targets are phony. Docker checks derive their default Go version from `go.mod`, quote the checkout path, work without a TTY, and provide Python 3 for the Makefile tests.
- The retained `make api-test` delegates to `backend/scripts/run-api-tests.sh`: build before resetting `changes_test`, reject ambiguous DB query overrides, reject an already responding service, use private temporary files, stop only its own server and preserve failures. Only absent query parameters or `sslmode=disable/require` are accepted by this legacy runner. It remains a Go HTTP test runner until the planned APIHydra replacement.
- `backend/scripts/makefile_test.py` exercises recipe selection, parallel read-only checks, failure propagation, coverage report lifecycle, DB guards and owned-process cleanup with isolated test doubles. These tests validate tooling behavior and are never counted as backend/APIHydra code coverage.

Verification: 19 tooling regression tests pass; a real golangci-lint formatter smoke test on disposable Go files proved that checking does not write and explicit formatting resolves the differences. Shell syntax and diff whitespace checks pass. Running `make -k check coverage deps-audit` against the real backend exposes pre-existing formatting differences and the missing return in `change/repo.go`; the vulnerability scanner also reports the existing `Details` interface mismatch. Full backend checks, unit coverage and vulnerability analysis therefore do not pass yet. No production Go files were reformatted or repaired. Docker execution, tool installation and real database integration were not run; their command/runner behavior was tested in isolation.

The user also authorized repository-wide naming cleanup on this preparation branch: the solution is `make-a-change`, the backend executable is `mch-server`, and the CLI is `mch`. Build recipes, frontend branding/storage keys, and tooling tests use these names. The obsolete checked-in executable was removed; generated `backend/mch-server` builds are ignored. Existing browser project selections under the previous storage namespace reset once. Future refactor specifications remain backend-only.

## Initial findings that determine the order

| Evidence | Consequence |
| --- | --- |
| `internal/change/repo.go:UpdateChangeTypes` discards the `Exec` result and has no return. | The baseline build fails. Repair this before relying on coverage or test results. |
| `change.Repository.Details` promises `domain.ChangeDetails`, while `Repo.Details` returns `[]domain.Change`; its query and scan layouts also disagree. `Repo.List` passes `change.TotalTC` without a pointer. | Expect additional build/runtime failures after the first compiler error is fixed. Repair and characterize these explicitly. |
| Change/epic queries expect `version` and `completed`; testcase queries use `public.test_case` and old procedures. Current SQL defines `public.testcase`, view-derived counters, and `public.doc`. | Map every field/query to the current schema. Remove unsupported history/version contracts. Assess derived fields individually: absence of a stored column alone does not prove a database-backed derivation is obsolete. |
| Project queries select `last_ref` from `vw_project`, which does not expose it, although `project.last_ref` exists. | Distinguish a wrong query from a genuinely obsolete field. Do not remove valid fields merely to make scans pass. |
| `internal/options` reads removed lookup tables and hardcodes colors. `domain.Config` lacks several configuration arrays. | Replace options with project-scoped configuration resolved through `project.config = config.slug`, including document types, phases, colors, and change types. No fallback when the selected slug is missing. |
| Project/epic/change mutations return entities; testcase mutations return the testcase, change, and refreshed list, with rendering. | Apply ID-only create responses and empty update/delete responses. Remove follow-up reads and testcase-to-change rendering coupling. |
| `change`, `epic`, and `testcase` repositories use `Begin`, `Commit`, `Rollback`, transaction helpers, and obsolete history/recalculation calls. | Remove Go-managed transactions while preserving supported database atomicity through existing functions/procedures or individual SQL statements. Do not equate independent calls with one atomic operation. |
| Error definitions/classifiers are spread across services and APIs; `internal/error` contains only `ExampleErr`/`ExampleError`. | Establish central error ownership before converting modules, preserving causes and explicit HTTP behavior. |
| `pkg/config` has global mutable state and panics; startup constructs dependencies and handles errors in `main`. | Plan injectable loading and testable startup/shutdown responsibilities without changing configuration precedence accidentally. |
| At initial inspection, `backend/Makefile` used an outdated module name, Go 1.25 for Docker, and a mutating lint target. | Corrected in the preparation above. P0 should use these targets and extend the coverage/integration infrastructure, not repeat the Makefile migration. |
| Existing HTTP tests are Go tests under `api-tests/`; no APIHydra suite was found. The API test target starts an uninstrumented server and kills it during cleanup. | Port retained scenarios to APIHydra and add reliable server coverage collection. Existing Go HTTP test coverage cannot establish the requested APIHydra coverage. |
| Several “current schema” unit fixtures assert old columns; the repository integration test asserts old testcase history. | Audit tests against the database before treating them as specifications. Replace obsolete assertions with traceable current-contract assertions. |

Paths in this and subsequent tables are relative to `backend/` unless stated otherwise. Size is supporting context, not proof that a file needs splitting: `change/repo.go` currently has 761 lines, much of which may disappear during contract alignment.

### Baseline actually attempted

From `backend/`:

```sh
go test -short -coverprofile=/tmp/backend-refactor-unit-baseline.out ./cmd/... ./internal/... ./pkg/...
```

Result: **failed**, first compiler error `internal/change/repo.go:254:1: missing return`. Some independent packages ran: epic 12.7%, project 15.4%, health 47.6%, markdown 88.9%; error, options, config, and db reported 0.0%. These are partial package observations, **not a valid overall baseline**. Change, testcase, and server did not build. Integration coverage is unmeasured. No database was initialized, server started, or API integration suite executed during planning.

## Branches and dependencies

### Official six-step implementation workflow

After the user resumes execution, conduct the entire sequence autonomously using the repository's existing factory scripts. Read `scripts/README.md` before starting and check Git authentication with `bash scripts/git-auth.sh --check`. A successful implementation subprocess does not by itself prove that coverage or acceptance criteria passed.

1. **Author one specification** for the next bounded pass as `agent/specs/NNN-slug.md`. Include scope, acceptance IDs, current/target contracts, concrete tests, coverage requirements, known baseline failures, and explicit backend-only write restrictions. Prepare the text outside the checkout until the matching branch exists. Each specification must reiterate the user's exclusions: ignore frontend, CLI, and PRD; use the current DB and backend architecture. Direct implementation records to `backend/implementation-log.md`, overriding the change-code skill's root-level log default to preserve scope.
2. **Create `change/NNN-slug` from freshly fetched `origin/dev`**, then add and commit the specification there so implementation starts with a clean tree. Use `scripts/branch-create.sh` when the specification is already committed on `origin/dev`; otherwise create the equivalently named branch directly from `origin/dev` before installing the prepared specification. The helper requires a clean tree and an existing spec, and checking out its `origin/dev` base would lose a spec committed only on another branch. Never hide unrelated work in an automated commit.
3. **Run `scripts/codex-code-spec.pl agent/specs/NNN-slug.md`.** The script invokes `$change-code`, then commits and pushes implementation. Inspect its final result and actual diff; resolve failures within the same bounded specification rather than moving on.
4. **Verify and review**: run required backend checks, unit coverage and APIHydra coverage appropriate to the pass; then run `scripts/codex-review-loop.pl agent/specs/NNN-slug.md --base origin/dev`. This explicit base avoids the helper's default `origin/HEAD` (master). Recheck affected validation after review fixes, including coverage when code changed. Record baseline gaps honestly. Numerical coverage gate failures do not block intermediate merges under the user clarification; failing tests and introduced regressions still require repair.
5. **Run `scripts/merge-to-dev.pl`** after the bounded specification is implemented, reviewed and tested, with actual coverage results and tracked baseline gaps recorded. It requires a clean published change branch containing fetched dev, squashes and pushes to dev, deletes the remote change branch, and leaves dev checked out. If dev advanced, reconcile/rebase and rerun relevant verification before retrying. Do not promote to stage or production as part of this sequence.
6. **Update the backend checkpoint and continue with the next specification.** Include the checkpoint in the completed branch before its merge, then read it from dev for the next pass. Report completed specifications, coverage and blockers as work proceeds. Continue sequentially despite coverage shortfalls; improve tests throughout and discuss any remaining numerical gap after the full sequence.

The user will integrate the current Makefile/plan branch and return to dev before requesting execution. Confirm that the resulting `origin/dev` contains this preparation before creating the first implementation branch; merely having it on stage is insufficient because the factory branches from dev. Do not begin these runs, commit/push, or promote this preparation on the user's behalf during the current task.

The existing pattern is `change/NNN-description`: `001-skills-and-archon`, `002-feature-cutoff-db`, `003-db-redesign`, `004-refactor-backend`. Keep `004` for this plan/current work. The following names are proposed next numbers, not created branches; check for newly reserved numbers before execution.

| Order | Proposed branch | Scope and exit evidence |
| --- | --- | --- |
| P0 | `change/005-backend-verification-foundation` | Minimal build repair; use the modernized backend checks; behavior/DB/route inventory; unit coverage denominator and gate; APIHydra health smoke suite; isolated DB runner and coverage lifecycle. Expose remaining failures without fabricating a green baseline. |
| P1 | `change/006-backend-error-contracts` | Central error definitions, wrapping/classification and HTTP interpretation; unit tests for each public function; adapt modules without silently changing existing supported status/body contracts. |
| P2 | `change/007-backend-project-epic-alignment` | Project configuration endpoint, removal of options, project/epic SQL/domain alignment, explicit mutation responses, no Go transactions in these modules. Unit tests and APIHydra workflows in the same branch. Split project and epic into consecutive branches if review becomes unwieldy. |
| P3 | `change/008-backend-change-doc-alignment` | Change reads and specialized mutations mapped to current views/functions and `doc`; explicit document reads; current project configuration; remove old history paths and convenience responses. Split reads/documents and mutations if necessary, assigning the next available branch numbers. |
| P4 | `change/009-backend-testcase-alignment` | Current testcase CRUD, view-derived counts, independent read/mutation contracts, remove rendered mutation responses and remaining transaction-dependent workflows. Confirm cross-module deletion and integrity behavior. |
| R1 | `change/010-backend-dead-code` | Dedicated audit/removal of proven dead remnants after alignment. |
| R2 | `change/011-backend-duplicate-paths` | Dedicated consolidation of equivalent surviving paths. |
| R3 | `change/012-backend-module-boundaries` | Dedicated responsibility/size cleanup based on what remains. |
| R4 | `change/013-backend-stale-abstractions` | Dedicated removal of obsolete indirection and hidden dependency lookup. |
| R5 | `change/014-backend-local-patterns` | Dedicated consistency pass with installed dependencies/toolchain unchanged. |
| R6 | `change/015-backend-compact-code` | Dedicated readability pass and final combined review. |

Dependencies are P0 → P1 → P2 → P3 → P4 → R1 → R2 → R3 → R4 → R5 → R6. Complete and merge each specification to dev using the six-step workflow before branching for the next; do not stack unmerged implementation branches. Intermediate dev merges may retain documented baseline/coverage gaps; do not promote to stage or production. Repair introduced failures immediately and remove tracked baseline issues during the planned passes. Do not create empty implementation branches when an iteration has no actionable findings; record that result in the checkpoint and move on (the code-spec helper requires actual implementation changes). Coverage work belongs in every branch, with a final gap review after R6, rather than a separate last-minute test rewrite.

## Prerequisites and contract-alignment passes

P0 must produce a compact `backend/agents/backend-contracts.md` containing acceptance IDs, route/field-to-database mappings, and old → target behavior. Reuse existing useful tests as fixtures; record any intentionally removed assertion with its database/architecture reason. Maintain `backend/agents/backend-refactor-checkpoint.md` for progress, decisions, test results, and the exact next pass. These are proposed future artifacts, not files created by this planning task.

Inventory currently finds 33 registered method/path pairs: project 5, epic 5, change 14, testcase 5, options 2, health 2 (including `/api/health`). Recalculate after contract changes. Account explicitly for removed/replaced routes; do not silently drop difficult routes from the inventory.

| Pass, scope, evidence | Behavior to preserve or explicit target | Bounded change | Validation |
| --- | --- | --- | --- |
| P0 build and checks: `change/repo.go`, `cmd/server`, `Makefile` | Compile retained contracts; checks must execute production packages and propagate failures. | Repair compiler blockers without placeholder successes; correct module discovery; introduce separate unit/APIHydra coverage targets. Extract only startup/shutdown seams needed to run and stop the real server reliably. | Build all backend packages; rerun unit baseline; prove health smoke requests execute and coverage counters appear after clean exit; deliberately failing test/suite must fail its target. Record pre-existing failures separately. |
| P1 errors: `internal/error`, API classifiers, services, connectors, startup | Invalid input, missing resource, conflict and unexpected failure remain distinguishable; underlying errors remain inspectable; internal failures do not leak DB details. | Centralize definitions/creation/wrapping/interpretation. Keep HTTP response writing at the API boundary. | Table-driven unit cases for direct/wrapped errors and cause preservation; APIHydra checks for representative 400/404/409/500 responses, with exact mappings entered in the contract ledger. |
| P2 project/configuration and epic | Project-selected configuration; missing slug errors even when `default` exists; DB-backed fields/counts/order; creates return 201 + ID; updates/deletes return 204. | Align DTOs/queries; expose one explicit project configuration operation; remove options routes/wiring; use current SQL and remove old epic history/transactions. | Direct-service validation; scan/row/error tests; two projects with distinct configs; missing config; create/read/update/read/delete; missing IDs; blocked parent deletion; no follow-up mutation reads. |
| P3 change and documents | Retain specialized update operations and supported ID ordering/nullability; use `fn_change_insert`, applicable `sp_change_*` and `sp_change_doc_set`; render stored content safely on explicit reads. | Separate document reads from mutations; map current columns; align document types/phase/type validation with selected configuration; remove old schema fields only with evidence. | Unit tests for each handler/service/repo operation; APIHydra create → details, each specialized update → read, document update → explicit read, invalid inputs, missing rows, cross-project epic association, unsafe Markdown, unknown types/phases, repeated writes. |
| P4 testcase and deletion | Use `public.testcase`; retain scenario/done state and database-derived counts; explicit mutation results; preserve supported parent/child integrity. | Remove obsolete history/procedure calls, duplicate change scans/rendering, and transactions. Model reads independently. Coordinate deletion only as the agreed current contract permits. | APIHydra testcase lifecycle and count changes through change/epic reads; missing parent/child, unchanged values, delete with children, constraint conflicts; unit row/scan/query error paths and service validation. |

For P2–P4, domain request types must pass through API → service → repository wherever possible. API code performs binding/format validation with Echo/validate; services enforce business validity independently; repositories contain only pgx-dependent work. All cross-layer business types live in `internal/domain`. Introduce small interfaces only for real mocking needs; do not build a generic repository/CRUD framework.

### Decisions to settle from concrete evidence during P0–P4

- **Deletion and atomicity:** the current SQL has child foreign keys and no general change-delete procedure. Decide the supported separate child-delete workflow and parent-conflict response before removing transaction wrappers. Do not silently turn all-or-nothing deletion into partial deletion. If a required atomic workflow cannot be expressed using existing DB facilities or one sound SQL statement, report the missing capability; do not edit `db/` or restore a transaction.
- **Missing rows and concurrent updates:** several stored procedures return normally when the target does not exist. Establish the intended missing-resource response and race behavior; a preflight existence query alone does not guarantee atomicity. Document any unresolved constraint rather than claiming parity.
- **History, completion, and documents:** old entity version/history storage is gone; append-only document storage remains. Distinguish these. Counters are backed by views; a derived completion percentage needs an explicit retained-contract decision. Stored document content is not obsolete just because it moved out of `change` columns.
- **Project/epic documents and new routes:** configuration advertises document types for all three entities. Determine the minimum explicit read/write contract required by the architecture; do not expand this cleanup into implementing every possible database operation.
- **Missing database constraints:** the schema deliberately omits the config foreign key and composite project/epic relationship constraint. Service validation and honest concurrency limits are necessary; do not claim PostgreSQL enforces what it does not.
- **UUID/configuration behavior:** preserve intentional UUID generation, accepted formats, precedence (file/defaults/environment/flags), CORS values, and startup failures unless a separately documented alignment decision changes them.

## Six separate cleanup iterations

These follow contract alignment because removing old contracts first should substantially reduce the cleanup surface. Preserve the now-established target behavior in all six. Reassess evidence at each boundary; “no actionable findings” is a valid outcome.

| Iteration and candidate scope/evidence | Behavior to preserve | Structural improvement | Parity validation |
| --- | --- | --- | --- |
| R1 dead code: `ExampleErr`/`ExampleError` have no references outside their definitions; `change.API.e` appears assigned but unused; old DTOs/helpers may remain after P2–P4. These are candidates until references, registration, build variants, and consumers are checked. | Registered routes, error identity, constructor effects, supported build paths. | Remove only proven unused declarations, obsolete comments and unreachable branches. Do not classify registered options endpoints as dead code before their explicit P2 removal. | Compile/vet/staticcheck; route registration tests and APIHydra inventory parity; focused tests for affected constructors. No coverage-driven deletion. |
| R2 duplicates: change/testcase currently duplicate change column scans, listing, mutation completion and rendering; API modules duplicate error classifiers. Much should already disappear during alignment. | Column order/types/null handling, error identity, deterministic result ordering, sanitization, side effects. | Consolidate surviving equivalent paths within their owning layer. Keep distinct business rules and simple repeated SQL where extraction adds indirection. | Null/non-null scans, iterator/scan/query failures, empty arrays, ordering and Markdown fixtures; module APIHydra parity. |
| R3 oversized modules: remeasure `change/repo.go` and `cmd/server/main.go` after removals. Startup mixes flag/config loading, dependency wiring, middleware, serving and error handling. | Routes/middleware order, request context, configuration precedence, shutdown/resource ownership and mutation boundaries. | Separate coherent startup functions; move business mapping out of repositories/API files; split change files only if responsibilities remain distinct. Keep the architecture's layer ownership clear. | Startup/config unit scenarios; cancellation/shutdown/resource-close tests; route/CORS/health APIHydra cases; full module workflow comparisons. |
| R4 stale abstractions: transaction-oriented `queryer`/`state`/`finishMutation` helpers, global `config.Get`, renderer dependencies crossing business modules. | Required injection seams, external behavior, error causes and ownership. | Remove remaining obsolete wrappers; inject only used collaborators; replace global lookup with explicit values. Keep interfaces justified by tests. | Independent config instances; collaborator failure/cancellation tests; race checks; APIHydra full suite. Do not remove a test seam merely to reduce interface count. |
| R5 legacy patterns: generic “executes behavior” comments, inconsistent naming/imports, raw error handling, inconsistent pgx iteration; current Makefile drift is handled in P0. | Supported method/route names, response shapes, validation rules, toolchain and dependency versions. | Apply established local conventions and accurate comments. Audit prohibited service imports, error creation outside `internal/error`, and Go transaction use. | Formatting/staticcheck/golint/vet/race; architecture checks plus focused unit/APIHydra cases where control flow changes. No mechanical renaming for symmetry. |
| R6 compact code: repeated validation/return scaffolding and nullable-value handling in surviving service/repo code. | Evaluation order, nil-versus-empty JSON, errors, short-circuiting and side effects. | Simplify only where readability improves. Preserve useful domain names and explicit steps; no line-count target or clever one-liners. | Boundary tables, affected unit tests, complete APIHydra suite and both final coverage reports; review combined diff against the contract ledger. |

## Unit-test strategy and coverage

The user requires 95% unit coverage; repository instructions say **greater than 95%**. Use the stricter **>95% backend-wide statement coverage** gate, report each package/function, and aim above 95% in each production package so low-coverage boundaries cannot hide behind larger services. Go's standard metric is executable statement coverage, not branch or source-line coverage. Test meaningful branches separately.

1. Give every acceptance bullet a stable ID and at least one named unit test. Maintain an acceptance → unit test → APIHydra scenario mapping; do not count one vague happy path as proof of all bullets.
2. Include all production Go packages under `cmd/`, `internal/`, and `pkg/`, including startup, connectors, health and errors. Include untested packages. Test-only clients under `api-tests/` are harness code, not production; document that distinction once. Audit the inventory for production files elsewhere.
3. Isolate unit runs from PostgreSQL/HTTP integration tests. The existing repository integration test skips in short mode; preserve explicit suite separation as new tests are added. A database-backed test must not be relabeled a unit test.
4. API unit tests use Echo with request/response recorders and prove malformed input, status, exact response keys, truly empty 204 bodies, and error envelopes. Service tests call methods directly with narrow collaborator fakes, proving business validation cannot be bypassed. Repository tests prove parameter binding, scan layouts, row closure, affected-row handling, and realistic pgx failures; integration tests prove actual SQL validity.
5. Use existing dependencies and small handwritten fakes where necessary. Do not add a mocking library without approval. A fake must assert meaningful interactions/results, not simply reproduce implementation logic to execute lines.
6. Test config precedence, renderer sanitization, startup/shutdown and resource cleanup through ordinary injectable collaborators. Do not introduce production test-only endpoints, switches, fake errors, or unreachable calls to inflate coverage.

Proposed unit measurement, from `backend/` after P0 makes the target runnable:

```sh
mkdir -p .coverage/unit
go test -short -count=1 -race -covermode=atomic \
  -coverpkg=./cmd/...,./internal/...,./pkg/... \
  -coverprofile=.coverage/unit/coverage.out \
  ./cmd/... ./internal/... ./pkg/...
go tool cover -func=.coverage/unit/coverage.out
go tool cover -html=.coverage/unit/coverage.out -o .coverage/unit/coverage.html
```

The gate must use covered/total statement counts before display rounding, not an arithmetic average of package percentages. Verify that all inventoried production packages are represented. A failed/partial run never passes the gate.

## APIHydra integration tests and genuine 90% code coverage

### Tool and suite layout

Installed binary: `/home/vito/go/bin/apih`, built from APIHydra commit `c3947513e2a4` (`v0.1.1-0.20260907061551-c3947513e2a4`). `apih --help` supports `--parallelism` and directory/steps selections. The installed-version manual was inspected through the local upstream checkout; its current HEAD differs, so use the installed commit's documentation when implementing. `curl`, `jq`, and `git` are available.

Use `backend/apih-tests/root.yaml`, scoped defaults, and steps files organized by health, project/configuration, epic, change/documents, testcase, and negative scenarios. This location intentionally overrides the apih-build skill's root-level default to obey the user's backend-only scope. Add runner scripts under `backend/scripts/` and evidence in `backend/apih-tests/coverage.md`. Ignore generated coverage/binaries locally through `backend/.gitignore` if needed; do not change the root ignore file.

```sh
# From backend/, after the isolated service is ready:
apih --parallelism 0 ./apih-tests
```

Use `app: apihydra`, `kind: root/defaults/steps`, `spec.steps`, explicit `response.expected_status`, stable JSON assertions, and dynamic type assertions. Capture IDs once and reuse them in request bodies. Captures are run-scoped and write-once; interpolation works in request/expected bodies, not URLs/headers. Keep dependent producers and consumers in one steps file or explicitly ordered stages; serial mode alone is not a substitute for declaring the intended dependency order. Remove all Debug breakpoints before verification.

Create an isolated test database and dedicated port; load the unchanged `../db/init.sql` and `../db/seed.sql` only into that disposable database. Validate the target before destructive setup, stop only the process started by the runner, and clean up that run's resources. A pre-existing `mch-server` binary is not evidence that current source works. Generate fixtures against the current schema, including distinct project configs and missing-config references, without editing root DB files. Fail setup on SQL errors rather than altering fixtures to conceal schema failures.

### Scenario design and assertion limits

- Port retained intent from existing `api-tests/`, then replace obsolete entity/history assumptions using the contract ledger. Remove an old Go HTTP test only after its retained scenario has a passing APIHydra replacement; direct DB contract checks may remain separately labeled and are not a substitute for APIHydra.
- Cover every retained operation with success, malformed/type-invalid input where applicable, invalid/missing IDs, relevant constraint errors, empty results, and state transitions. Verify writes through independent subsequent reads. Include both health aliases and degraded DB behavior. Do not invent authentication/pagination requirements when the backend has none.
- Exercise project config selection/missing config, project/epic deletion conflicts, every specialized change update, current documents and sanitization, testcase done/undone flows, and changes in DB-derived counts. Verify same-value updates according to the actual current database contract rather than old version/history assumptions.
- APIHydra's expected-object comparison ignores extra actual fields. An omitted/empty expected body skips body comparison. Therefore `expected_body: '{"id":...}'` alone does not prove an ID-only response, and `expected_body: ''` does not prove an empty response. Prove exact keys/empty bodies in API unit tests; investigate supported assertions in the installed tool and record any remaining integration assertion gap. Do not claim unsupported checks passed or modify APIHydra outside this repository's scope.
- Keep deterministic cleanup and error scenarios scoped to disposable fixtures. Meaningful constraint violations or an isolated DB outage can test failure paths. Do not corrupt unrelated databases or fabricate application branches solely to raise coverage.

### Measure server execution separately from endpoint coverage

**Required integration gate: at least 90% of backend production Go statements executed by APIHydra-driven integration runs.** Counting endpoints is a separate diagnostic; it does not satisfy the gate. Aim to exercise every retained operation and report the actual covered/total route count, including blocked routes in the denominator.

Build and run the real backend with Go coverage instrumentation. This is supported by [Go's integration coverage tooling](https://go.dev/doc/build-cover). The proposed commands below describe the build/report portions; the P0 runner must implement isolated DB setup, readiness, failure propagation, graceful stop and process waiting between them.

```sh
# From backend/; runner creates a fresh directory for each measured campaign.
mkdir -p .coverage/integration/raw .coverage/bin
go build -cover -covermode=atomic \
  -coverpkg=./cmd/...,./internal/...,./pkg/... \
  -o .coverage/bin/server ./cmd/server

# Runner launches this binary with an absolute GOCOVERDIR pointing to raw/,
# waits for readiness, executes apih --parallelism 0 ./apih-tests,
# then requests graceful shutdown and waits for normal process completion.

go tool covdata textfmt -i=.coverage/integration/raw \
  -o=.coverage/integration/coverage.out
go tool cover -func=.coverage/integration/coverage.out
go tool cover -html=.coverage/integration/coverage.out \
  -o .coverage/integration/coverage.html
```

The current server lacks an explicit graceful shutdown lifecycle. P0 must prove that shutdown allows `main` to return and coverage files to flush; simply sending a terminating signal is insufficient. Treat missing metadata/counters, crashed processes, skipped suites, and incomplete runs as failures. Do not add a public coverage/shutdown endpoint.

Use fresh profiles for each commit/build. Combining valid APIHydra scenarios from the same measured revision is acceptable, with run provenance. Never merge unit profiles or legacy Go HTTP-test profiles into the APIHydra result. Audit the production package inventory against emitted coverage metadata: `go build` only instruments linked packages, so unlinked production code must be reported, not silently excluded to improve the percentage. No test harness, third-party library, or SQL procedure statement belongs in the Go production denominator; DB correctness needs its own assertions.

APIHydra failure codes (101 assertions, 102 configuration, 103 execution, plus other external-command failures) must fail the backend target. Preserve the test failure even if cleanup succeeds. Record the suite exit result, selected scenarios, commit, tool version, DB fixture source, raw profile location, total statement counts, per-package coverage, and route mapping.

Reference: [APIHydra manual at the installed commit](https://github.com/divilla/apihydra/blob/c3947513e2a4b948ce732dec60e090a5e83be744/docs/user-manual/apih.md).

## How to conduct each pass

1. Read the latest checkpoint and inspect the working tree. State one bounded purpose, its acceptance IDs, affected modules, and any intentional old → new contract change.
2. Run the affected baseline checks and record failures. Add meaningful characterization tests for retained behavior before transforming it; use target-contract tests for explicit alignment work.
3. Make the smallest coherent backend-only change and its tests. Preserve useful assertions; do not fix unrelated formatting or change DB/frontend/CLI files.
4. Run focused tests, then backend `make check` after its P0 repair, unit coverage, and APIHydra against the instrumented server. Use fresh profiles and inspect uncovered statements. Extend tests only when they add coverage or prove a specific acceptance bullet.
5. Review the diff for layer violations, hidden reads after mutations, raw error definitions, Go-managed transactions, unsupported SQL references, and edits outside the allowed path. Check untracked files too.
6. Update the backend checkpoint with actual results, deviations, outstanding gaps, and next action. At each of the six cleanup boundaries, condense the checkpoint rather than carrying a long transcript. Reassess remaining candidates before proceeding.

Keep PRs understandable by a reviewer without this conversation: concrete problem, resulting behavior, affected contracts, tests actually run, measured coverage, and known limitations. Split a pass when it mixes an API migration with cleanup, changes several unrelated responsibilities, or cannot be reviewed/reverted coherently. Revert a complete pass and its tests together if it regresses retained behavior.

## Completion and honest shortfall policy

The canonical backend completion commands and their current migration limits are in root `AGENTS.md`, under “Backend Definition of Done”. Keep the future verification-foundation work aligned with that policy; individual specifications must repeat the user clarification that coverage is a final-result goal and does not stop intermediate work.

The final target is backend checks passing, a unit test for every acceptance bullet, unit statement coverage **>95%**, APIHydra-driven integration statement coverage **≥90%**, verified architecture/database alignment, and no implementation edits outside `backend/`. Report the two percentages independently and provide reproducible backend commands.

If a numerical target remains unreachable through meaningful testing, finish the entire remaining safe, reviewable sequence and report a clear **target not met** result at the end; do not stop or request permission between specifications solely because of coverage. Do not keep rewriting code/tests merely to make a number green. Provide:

- Actual covered/total statement counts and percentages for both suites, plus package/function gaps and uncovered source locations.
- The observable behavior each gap represents, attempts made to exercise it, and the concrete obstacle: infrastructure, tool assertion limits, required DB change, naturally unreachable external failure, or an unresolved product contract.
- Passing/failing/skipped scenarios and preserved raw evidence; no rounded-up or combined-suite claims.
- Concrete options for discussion, such as a justified test seam, improved isolated fixtures, an APIHydra capability enhancement, a separately authorized database task, or an explicitly agreed exception.

Never lower thresholds silently, filter difficult production files/packages, mark live code generated, delete defensive behavior, replace real integrations with mocks, introduce no-op tests, or weaken assertions. Legitimate removal of obsolete behavior requires a database/architecture reason and a recorded contract change independent of its effect on coverage. Keep any unresolved gap visible and discuss it together before treating the work as complete.
