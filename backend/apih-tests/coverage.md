# Standalone APIHydra tests

Start the development application separately (`make run` from the repository
root), then run the suite directly:

```sh
cd backend/apih-tests
apih
```

From the repository root, run the same suite manually or with statement coverage:

```sh
apih backend/apih-tests
make -C backend api-test
```

`root.yaml` selects the server, currently `http://127.0.0.1:8080`. Edit its
`base_url` to target another development/test backend. The server must already
be healthy and initialized with the current application schema and default
configuration. Change creation requires `backlog` and `brief`; doc and type
cases capture configured values from `project/config` (nonempty project/epic
doc types and change types). No specific demo records or identity values are
required. Do not target production.

The suite always uses HTTP endpoints and captures for setup and cleanup. Neither
invocation performs SQL setup, seeding, truncation or database shutdown.

Manual `apih` uses the already-running server selected by `root.yaml`.
`make -C backend api-test` instead builds an instrumented backend, starts it on
`API_TEST_PORT` (default 19080) against the existing `DATABASE_URL` (default
`postgres://postgres:postgres@localhost:15432/changes`), and runs a private copy
of the unchanged suite against that server. The existing application server is
left running. Override `DATABASE_URL` or `API_TEST_PORT` through Make/environment
when necessary; an occupied test port is rejected. No database tools are needed
by the runner; the database must already exist with the application schema and
configuration.

The Make target streams standard APIHydra output, gracefully stops only its own
server to flush Go counters, then prints a blank line and actual statement
coverage by package and in total. A complete campaign must cover at least 90%
of backend production statements to pass.
Test, build, shutdown and invalid-profile failures still fail the command and
remove invalid success reports. Readiness checks use TCP, not extra HTTP calls;
the measured counters include server startup, APIHydra traffic and shutdown.
Fresh artifacts in `backend/.coverage/api/` include `coverage.out`, `report.txt`,
`result.json`, the source denominator, suite hashes, provenance and server/build
logs. Unit profiles are never merged into the API measurement. Bare `apih` on an
ordinary server does not collect statement counters.

## Setup, captures and cleanup

Seven independent endpoint groups run concurrently in APIHydra's default mode1.
Within each group the numbered files run serially:

```text
apih-tests/
  root.yaml
  change/{01-init,02-main,03-post}.yaml
  config/{01-init,02-main,03-post}.yaml
  doc/{01-init,02-main,03-post}.yaml
  epic/{01-init,02-main,03-post}.yaml
  health/02-main.yaml
  project/{01-init,02-main,03-post}.yaml
  testcase/{01-init,02-main,03-post}.yaml
```

`01-init.yaml` creates cross-module prerequisites through endpoints.
`02-main.yaml` exercises behavior and checks it through independent reads.
`03-post.yaml` deletes the group's remaining records through endpoints, in
foreign-key dependency order. Some deletions happen in the main workflow to
verify missing-record and conflict behavior.

Response captures supply all successful entity IDs. Missing mutation targets
are captured IDs deleted earlier by the same group, never guesses at unused
IDs. The only literal positive-ID probes are read-only oversized not-found
cases. Config CRUD uses a unique slug derived from its own API-created project
ID, so repeated and concurrent runs do not collide or modify shared configs.
Global project/config list assertions allow other application data to coexist.
The suite never updates or deletes the application's default configuration.

For example, an ID captured as `.id` is reused unquoted in a JSON request:

```yaml
response:
  expected_status: 201
  capture:
    project_id: .id
# A later request in the same group:
request:
  method: POST
  path: /api/v1/project/details
  body: '{"id":${project_id}}'
```

Captures are run-scoped and write-once. Each group has unique variable names and
uses only its own earlier producers. Captured JSON strings retain their quotes,
so requests interpolate them as JSON values as well. Interpolation is supported
in request/expected bodies, not jq selectors. These rules follow the
[APIHydra manual](https://github.com/divilla/apihydra/blob/master/docs/user-manual/apih.md#variables-and-captures).
Tooling tests validate capture ordering/uniqueness, mutation ownership, all
registered operations and the absence of SQL fixtures.

Select a whole group for a focused run so its setup and cleanup are included:

```sh
apih backend/apih-tests/doc
```

Use mode0 or the default mode1; mode2 would run dependent init/main/post files
concurrently. A terminal failure or interruption may prevent post files from
running. Such partial runs can leave their own records; a new run still creates
fresh IDs and a fresh config slug. Docs and history retained by normal API
deletion semantics remain retained: no direct database cleanup or unsupported
delete endpoint is used.

## Verified results — 2026-09-28

The standalone suite previously passed repeated manual runs on the development
server. The restored coverage target also passes all the same HTTP requests
against its own instrumented server, using the existing database without SQL
setup. `make -C backend tooling-test` passed 33 Python tests plus the Go validator
tests. `make -C backend coverage` passed with fresh 1031/1045 (98.6603%) unit
statements; the only unit gap is 14 startup/main statements in cmd/server.
The standalone YAML and its port 8080 URL remain unchanged.

| Group | Requests | Successful registered operations |
| --- | ---: | ---: |
| change | 143 | 11 |
| config | 50 | 5 |
| doc | 88 | 4 |
| epic | 50 | 5 |
| health | 3 | 2 |
| project | 46 | 6 |
| testcase | 89 | 5 |
| **Total** | **469** | **38/38 (100%)** |

Each operation has a successful case; malformed payloads, validation failures,
missing records, reference conflicts, independent mutation reads, doc history,
HTML sanitization and testcase completion counters are also exercised. Every
request declares an explicit expected status. Nineteen YAML steps files pass.

The prior API **statement coverage was 943/1045 (90.2392%)**; it was recorded
before the current 90% gate was enabled and is not a fresh measurement of this change.
The previous 980/1045 result belongs to the removed SQL/outage campaign; it is
not reused here. Operation coverage above is a separate metric.

| Production package | Covered/total statements | Uncovered |
| --- | ---: | ---: |
| cmd/server | 76/93 | 17 |
| internal/app | 40/52 | 12 |
| internal/change | 261/275 | 14 |
| internal/config | 96/101 | 5 |
| internal/doc | 112/127 | 15 |
| internal/domain | 0/0 | 0 |
| internal/epic | 106/114 | 8 |
| internal/health | 16/21 | 5 |
| internal/project | 109/116 | 7 |
| internal/testcase | 96/105 | 9 |
| pkg/config | 23/31 | 8 |
| pkg/markdown | 8/10 | 2 |
| **Total** | **943/1045** | **102** |

No production package is excluded. Domain has no executable statements. Gaps
include startup errors, database failures, scan/iteration failures and service
validation not reached through already-validated HTTP requests.

Cases requiring database outages, missing selected configs, assigning a custom
config to a project, artificial wide counters or direct history-table inspection
are outside the standalone suite: the public API cannot set up those states
without altering shared configuration or accessing the database. The removed
SQL-fixture and SQL-postcondition assertions are not claimed as passing here.
Application unit tests retain the corresponding error and validation contracts.
See the [route ledger](../agents/backend-contracts.md) for individual operations.
