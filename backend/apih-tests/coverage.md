# APIHydra P1 measurement

Measured 2026-09-28 on `change/006-backend-error-contracts`, working-tree P1
implementation over `d3c2235`. The complete campaign contains
`health-steps.yaml` and `error-steps.yaml`: **13 requests pass**, APIHydra exits
0, and the owned instrumented server exits 0 following SIGTERM. Covmeta and
covcounters were collected before conversion. A private PostgreSQL cluster
loaded unchanged init.sql and seed.sql with ON_ERROR_STOP, then shut down and
removed its temporary files. No external database was reset.

`make -C backend api-test` exits **2** (gate recipe **1**): **298/1209 = 24.6485%**
of backend production statements, below the required **90%** final-result goal.
Structural metadata includes unlinked `pkg/db` (4 statements at zero coverage).
`internal/domain` has no executable statements. Linked-package and block audits
pass; no unit or legacy profiles contribute. Unit coverage separately measures
**786/1209 = 65.0124%**, below its strict >95% final goal.

Operation diagnostics are separate: **9/33** registered method/path pairs are
reached, with successful operation coverage still **2/33** (health aliases).
Seven routes are reached only for errors: project list/get/create/delete,
epic create, change get and testcase list. An additional unregistered route
checks router 404 without changing the denominator. See the
[ledger](../agents/backend-contracts.md) for all 33 routes.

The error campaign covers malformed bind 400, service-validation 400, missing
change/testcase-parent/project/epic-parent 404, generic SQL failure 500, and
project child-conflict 409. The initial database contains config only. Its first
project/epic inserts succeed before their obsolete reload projections fail;
explicit 500 assertions establish the disposable fixture for the 409. They do
not prove successful create contracts and will change with P2's SQL repairs.

Remaining real error scenarios: epic-delete 409 is blocked by its obsolete
projection before the child guard; project-get 404 is blocked by `last_ref`;
testcase insertion FK mapping is blocked by removed `fn_test_case_insert`;
change phase-reference checks depend on removed `change_phase`. Unit tests
retain these semantics and underlying causes. Degraded health 503 has exact
unit proof; this campaign does not induce a database outage.

APIHydra provenance: installed binary `/home/vito/go/bin/apih`, module
`v0.1.1-0.20260907061551-c3947513e2a4`, build revision
`c3947513e2a4b948ce732dec60e090a5e83be744`. Its exact revision's local manual and
`pkg/runner/runner.go` were read with `git show c3947513e2a4:...`. Invoked as
`apih --parallelism 0 <private-complete-suite-copy>` with `XDG_CACHE_HOME` inside
the owned artifact directory. The copy changes only the root base URL's port.
No Debug directives, request selections or profile merging are used.

The first run failed (APIHydra 101) because inherited curl retries concatenate
multiple 500 bodies and replay inserts. No valid measurement was accepted from
that run. The error file now sets `retries: -1`: the installed resolver treats
zero as inheritance, accepts a negative override, and the runner only emits
`--retry` for positive values. This executes each asserted request once without
changing any expected status/body or altering the installed tool.

APIHydra compares declared JSON fields and ignores extra object keys. Empty
expected bodies disable comparison, so exact error envelopes and empty 204s
remain unit-tested in `TestInstalledJSONErrorContracts`, module handler tests,
`TestHealthAliasExactContracts` and `TestChangeAPIContracts`.

Current ignored artifacts in `../.coverage/api/`: `runner.log`, `server.log`,
`postgres.log`, `provenance.json`, `packages.json`, `denominator.json`, counters,
`coverage.out`, `report.txt` and `result.json`. Provenance records commands,
versions and backend changes; denominator metadata hashes production source.
The [checkpoint](../agents/backend-refactor-checkpoint.md) records exact package
totals and all required command exits. Legacy Go HTTP scenarios remain separate
and were not run or counted as passing APIHydra evidence.
