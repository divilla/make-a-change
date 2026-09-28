# Backend error contracts (P1)

Branch: `change/006-backend-error-contracts`; base: `origin/dev` at `d3c2235`.

## Purpose and authority

Centralize backend error ownership before business/schema alignment. Follow
`AGENTS.md`, `docs/backend-architecture.md`, the P1 row of
`agent/backend-refactor-plan.md`, and the current backend checkpoint/contract
ledger. Read database SQL only as needed to understand real error fixtures.
The user excludes CLI, frontend and PRD (do not read them to resolve contracts).
All implementation, tests, scripts, reports, fixture and documentation writes
belong under `backend/`; this already-authored spec is the sole exception.
Root AGENTS, scripts, architecture docs, SQL and dependencies are read-only.
Override change-code's log default: append its exact implementation block to
`backend/implementation-log.md`, never a root log. The enclosing factory owns
commits and pushes: do not commit, push, merge or start P2 in the subprocess.
Do not spawn concurrent checkout writers.

The user clarified that >95% unit and >=90% real API integration statement
coverage are FINAL refactor-result goals. Measure both now, improve meaningful
coverage and record exact counts; a numerical shortfall must keep the gate
failing honestly but does not block intermediate completion/merge. New test or
runtime regressions must be repaired. Existing format/lint debt may be deferred
explicitly; fix formatting/lint in touched files where appropriate. Never fake
coverage, weaken assertions or merge unit/legacy profiles into integration.

## Existing evidence and bounded scope

Module service files define their own sentinels. API files independently create
Echo errors and classify sentinels. Repositories classify pgx no-rows and SQLSTATE
errors directly, discarding some causes. The error package is only ExampleErr
and ExampleError. Startup classifies server closure and builds JSON errors;
config/db connectors panic raw errors; health logs raw ping failures. These
all fall under backend-wide centralized ownership. Retained HTTP contracts are
400 invalid payload/reference, 404 missing entity, 409 blocked parent removal,
500 generic Internal Server Error without DB details, and health's existing
503 degraded response. Preserve module/operation-specific existing messages.

P2/P3/P4 own broken legacy SQL, mutation response migrations, options removal,
domain request alignment, transactions and obsolete entities/documents. Do not
expand into those passes to obtain a green integration suite. Existing valid
health and change details paths plus validation/missing/error operations can
exercise errors now. Do not invent public failure endpoints or damage a shared
database. Use only the existing owned disposable integration cluster.

## Acceptance criteria

Each numbered criterion must have meaningful named unit tests, with a concise
criterion-to-test mapping in the checkpoint.

1. **P1-01 central definitions and causes.** Replace ExampleErr/ExampleError and
   module-local error definitions with useful error-package definitions and
   public creation/wrapping/classification functions. Preserve distinct module
   invalid-input/not-found/reference/conflict semantics. Handle nil sensibly,
   preserve underlying errors through errors.Is/As, and avoid repeated wrapping
   when no context is added. Test every exported error function, direct/wrapped
   cases, unknown errors, nil, and nested external causes. Do not introduce a
   generic framework or unused prospective error APIs.
2. **P1-02 database boundary.** Repositories handle external pgx errors through
   the central package, including no rows and the existing foreign-key mapping;
   preserve semantic and underlying causes. Remove local classifiers and
   wrappers. Retain intentional missing-row/affected-row/conflict distinctions
   and existing parameter/query behavior. Test representative query, scan,
   iteration, constraint and missing-row errors and all changed mappings.
3. **P1-03 HTTP contract.** API handlers use central functions for bind/validation
   failures and service error interpretation; remove local error switches.
   The central interpreter owns consistent status/message interpretation;
   response writing remains at the HTTP boundary. Keep existing JSON envelope
   and supported messages/statuses, including wrapped errors and Echo router
   errors, and mask unexpected internal error details. Preserve causes even
   through HTTP translation. Test exact status/body/envelope and original cause
   for invalid input, missing resource, conflict, unknown and wrapped errors.
   Check both handler-return and installed server error-handler behavior.
4. **P1-04 backend-wide ownership.** Startup, configuration, connectors,
   Markdown and health handle external errors through the central package.
   Preserve existing lifecycle, config precedence, health output and cleanup;
   config injection/global-state removal belongs to later cleanup. Preserve
   useful causes on panic paths until that lifecycle is refactored. Tests must
   exercise changed startup/connector/health paths and service validation.
   Services must not import pgx, Echo or validate. Audit production code for
   error definitions, fmt.Errorf, errors.Is/As and Echo error creation outside
   internal/error; remove independent interpretation without relocating HTTP
   response writing into services or repositories.
5. **P1-05 genuine integrations.** Extend backend/apih-tests with representative
   400, 404, 409 and 500 contracts where real current-schema endpoints support
   them. Use explicit status and stable expected_body assertions. If a status
   cannot be reached without deferred SQL migration, retain strong unit proof
   and record the exact missing integration scenario rather than manufacturing
   success. Keep health assertions and route denominator; distinguish routes
   reached only for errors from successful operation coverage. APIHydra drives
   the requests; real instrumented server coverage stays separate from units.
6. **P1-06 verification and handoff.** Run targeted unit tests then required
   `make -C backend check`, `coverage`, `deps-audit`, `api-test`; use expanded
   `make -k ... check` if baseline formatting prevents the other checks.
   Record real exits, exact covered/total counts, package gaps and blocked
   scenarios. Update the compact backend checkpoint and contract ledger with
   current results and P2 next action; replace stale P0 execution history.
   Keep failed gates visible and distinguish baseline lint/schema failures
   from new regressions. No merge, dependency change or promotion here.

## Completion evidence

Review the whole diff for paths outside scope, public-status drift, leaked
internal messages, lost error causes, new transactions, service layer imports,
unnecessary abstractions and weakened tests. Append the implementation log using
the skill's count rules. Report bounded implementation and exact limitations,
without claiming that the entire backend is ready or final coverage passed.
