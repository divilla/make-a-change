# Backend verification foundation (P0)

Branch: `change/005-backend-verification-foundation`.
Base: `origin/dev` at `7dc3aa0` (compressed preparation commit).
Authority: user-authorized execution of `agent/backend-refactor-plan.md`, P0.

## Purpose and boundaries

Make the backend buildable enough to establish honest unit and APIHydra code
coverage baselines, and implement repeatable enforcement and an owned server/DB
lifecycle. This is the verification foundation, not the later error, project,
epic, change/document, testcase or cleanup migrations. Genuine remaining failures
are a deliverable; never manufacture a green baseline or implement unrelated
contract migrations merely to increase a percentage.

The user's explicit instructions override skill defaults: ignore CLI, frontend,
PRD and unrelated skeleton material; do not read or change PRD to resolve backend
contracts. `AGENTS.md`, `docs/backend-architecture.md`, `db/init.sql`, `db/seed.sql`
and the official refactor plan govern this task. Database files and architecture
documentation are read-only. All implementation, tests, scripts, fixtures,
reports and implementation logs belong under `backend/`. This spec is the sole
additional write exception for the factory. In particular, the change-code log
MUST be `backend/implementation-log.md`, preserving the skill's block format and
counting rules while overriding its repository-root location. Do not change root
AGENTS.md, root scripts, dependencies, CLI, frontend or deployment files.

Do not commit or push from inside the implementation subprocess: the enclosing
`scripts/codex-code-spec.pl` owns those operations. Do not merge or start P1.
Do not spawn concurrent checkout writers. Naming is make-a-change, backend
executable `mch-server`, CLI `mch` (CLI remains out of scope).

## Known starting evidence

- `change.Repo.UpdateChangeTypes` lacks a return. Its declared entity-returning
  contract is itself obsolete; a placeholder return or new enrichment read is
  prohibited. Prefer the narrow architecture-consistent repair: error-only
  repository/service and empty HTTP 204 for this operation, with focused tests.
- `change.Repo.Details` returns a slice unlike its interface's
  `domain.ChangeDetails`; SQL column/scan layouts disagree. Resolve the compiler
  blocker against current database structure, not old schema hidden by mocks.
  Document the exact limited contract correction and leave independent P3 work
  queued. No artificial zero-valued compatibility fields or success responses.
- Existing format/lint, SQL and architecture drift will remain. Do not disable
  checks or perform unrelated broad reformatting merely to hide baseline errors.
- Current `coverage` measures without enforcing; `api-test` runs legacy Go HTTP
  tests without server coverage. Legacy Go tests cannot supply APIHydra coverage.
- Main currently has no graceful signal-driven shutdown. `go build -cover` only
  records counters reliably when the instrumented process exits normally.
- PostgreSQL binaries `initdb`, `pg_ctl`, `postgres` are installed; an existing
  port 5432 service belongs to the user. Prefer a private disposable cluster and
  private socket/port; do not reset or stop the existing service.

## Acceptance criteria and required evidence

Each numbered criterion needs named meaningful unit tests or tooling regression
tests where the subject is tooling; retain the acceptance-to-test mapping in
the checkpoint. Tooling tests never contribute to production Go coverage.

### P0-01: real compilation repairs

Repair actual compiler blockers and tightly coupled scan defects with the
smallest current-schema implementation. Preserve supported behavior except the
explicit narrow mutation correction described above. Add tests demonstrating
return/error propagation, interface compatibility, correctly bound/scanned data,
missing records and response status/body where the fix affects them. No new
Go-managed transactions, error masking, placeholder success, or new dependency.
Record any unavoidable deferred runtime schema failure with its future pass.

### P0-02: owned graceful server lifecycle

Introduce only the injectable startup/shutdown seams needed to start the real
server, handle SIGINT/SIGTERM, stop accepting requests, wait for shutdown, close
owned resources and return from main. Preserve normal flags/config precedence,
routes and middleware. Startup/serve/shutdown failures must remain failures.
Test normal cancellation, startup/serve failure, shutdown failure/resource
cleanup and relevant ordering. No public shutdown/coverage endpoint or special
coverage-only behavior. Demonstrate actual instrumented server counter flush.

### P0-03: unit statement denominator and strict gate

`make -C backend coverage` runs fresh, uncached, short race-enabled unit tests
over every production package under `cmd`, `internal`, and `pkg`, excluding only
test harnesses/dependencies. Produce per-package and aggregate covered/total
statement counts and percentage. Gate uses exact integer arithmetic and requires
strictly greater than 95%; rounded 95.0% is not proof. Enumerate all production
packages and verify they are represented, including packages without tests and
unlinked packages. A package with no executable statements can be identified
explicitly as such rather than falsely missing. Missing, stale, partial,
malformed or failed test coverage cannot pass. HTML investigation should remain
possible for a valid below-threshold profile. Test exact threshold boundaries,
denominator/missing-package handling, stale cleanup and subprocess failures.

### P0-04: isolated APIHydra runner

Replace `make -C backend api-test` with an APIHydra-driven run. Keep legacy Go
HTTP tests until their retained scenarios have passing replacements; if keeping
a separate legacy target, clearly label it and exclude its output from APIHydra
coverage. Use a newly owned disposable PostgreSQL cluster/database and private
server; load unchanged `db/init.sql` and `db/seed.sql`, fail on SQL errors. Never
mutate an unrelated database or stop an unrelated process. Check prerequisites
and occupancy before setup, propagate failures, bound readiness/cleanup waits,
and clean up only owned resources on success, failure and signals. Avoid secret
logging and keep generated artifacts within backend or temporary storage.
Set APIHydra cache location to private writable storage if needed.

Run installed `apih --parallelism 0` against suites under `backend/apih-tests/`.
Inspect its installed-version manual (local checkout at
`/home/vito/go/src/apihydra`, commit `c3947513e2a4`) before selecting syntax; do not
invent flags/interpolation. No Debug breakpoints or skipped/selected subsets
masquerading as a full campaign. Tests must prove failures 101/102/103 and other
nonzero commands propagate, occupied services survive, SQL/startup failure is
visible, and owned resources clean up. Fake tooling tests do not replace a real
APIHydra smoke run against real PostgreSQL and the instrumented current server.

### P0-05: real integration statement coverage and gate

Build `mch-server` with `go build -cover -covermode=atomic` and all production
package patterns, run with a fresh absolute GOCOVERDIR, execute APIHydra, then
gracefully stop/wait before `go tool covdata textfmt`. Require normal successful
server exit, completed suites, metadata and counters. Report >=90% using exact
statement counts separately from unit profiles and endpoint counts. Never merge
unit/legacy HTTP profiles into this result. A valid below-threshold profile is
useful evidence but must fail the gate. Audit all production packages against
metadata; go build can omit unlinked code. Such code must be counted as zero
using defensible structural metadata or cause a visible incomplete-denominator
failure, never be silently excluded. Do not call otherwise-unused code from the
server to instrument it. Test 90% boundaries, malformed/missing data, stale data,
suite/server failure preservation, coverage conversion failure and denominator
auditing. Reports record revision, commands/tool provenance and raw artifacts.

### P0-06: health smoke and route/schema ledger

Create APIHydra health scenarios with explicit status and stable body assertions
for both registered health aliases. Inventory every registered method/path
(currently 33 pairs), mapping existing behavior and database source to intended
architecture, including obsolete operations and future replacements. Keep
untested/blocked routes in the denominator. Publish this inventory as
`backend/agents/backend-contracts.md`; publish real suite results and separate
operation/statement coverage in `backend/apih-tests/coverage.md`. This pass's
health smoke does not claim all operations or 90% statement coverage. Record
APIHydra exact-object/empty-body assertion limitations honestly, using API unit
tests for exact response contracts where applicable. No invented auth behavior.

### P0-07: reviewable checkpoint and final validation

Keep bare `make -C backend` as help. Extend Makefile/tooling tests for new
behavior, preserving visible lint/vet/race/audit failures and tool versions.
Run `make -C backend check`, `make -C backend coverage`,
`make -C backend deps-audit`, and `make -C backend api-test` on the final code;
run focused meaningful tests first. Finish safe bounded work and record exact
exit results, valid measured covered/total statement counts (or precise reason
measurement is unavailable), per-package gaps, acceptance mapping and blockers
in `backend/agents/backend-refactor-checkpoint.md`. Distinguish implemented P0
infrastructure from whole-backend readiness. The root AGENTS migration note
cannot be updated in this scope; note its stale description in the checkpoint
for coordinator handling instead of editing it without authorization.

If >95% unit or >=90% integration cannot be achieved legitimately in this
bounded pass, do not expand into unrelated refactors, hide failures or weaken
assertions. Preserve the implementation and honest results, explicitly report
P0 incomplete / coverage target not met, and stop for coordinator/user discussion
before merge. Never claim all done just because factory commits successfully.

## User clarification after P0 review — 2026-09-28

The user explicitly authorizes continuing the entire sequence and intermediate
dev merges despite numerical coverage shortfalls. Targets apply to the final
refactor; pursue them through meaningful tests, preserve failing gate results,
and report any remaining shortfall at the end. This supersedes the earlier
stop-before-merge instruction above. No tests or coverage counters are weakened.
