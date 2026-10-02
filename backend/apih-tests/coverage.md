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
fresh IDs and a fresh config slug. Document history remains retained. The document group uses the supported
/doc/delete operation to soft-delete its own documents; no direct database
cleanup is used. Failed or interrupted runs may leave their own records, and
records without captured ownership are never cleaned up by guessing IDs.

## Verified API error-suite refresh — 2026-10-02

The user requested additional missing errors that can be exercised using APIHydra.
The suite adds **487 requests**: **479 error assertions** and eight independent
reads checking that failed writes leave existing state unchanged. Production Go,
dependencies, database/schema, root URL and application lifecycle remain unchanged.

Added scenarios cover each applicable field independently with otherwise valid
request fields: omitted/null values, wrong types, required blank/whitespace text,
unsupported reference tables, fractional IDs, signed integer underflow, required
booleans, UUID input and all six configuration arrays (including an invalid second
entry). JSON array/string bodies are rejected by the typed request binders.
All45 registered operations now have wrong-method405 assertions; removed
`/doc/current` and `/change/update-open` remain404. Normal document insert rejects
comments for all three parent kinds, and deleted projects/epics/changes reject new
comments with the exact404 parent error. Existing conflict tests remain intact.

Fourteen requests exercise real database-rejected NUL text in project/epic names,
change titles/briefs, normal documents/comments, configuration array entries and
testcase scenarios. They assert500 `Internal Server Error` with only the `message`
key, and independent reads verify retained state. These cases document current
behavior: the backend accepts the JSON input and PostgreSQL rejects the text;
the backend does not validate this character into a400. No outage or SQL fixture
is needed. Input and writes use only this campaign's captured IDs/slugs; malformed
numeric inputs are fractional negatives or below the signed64-bit range, never
fixed positive mutation targets. Document history remains soft-deleted as designed.

| Group | Passing requests |
| --- | ---: |
| change | 291 |
| config | 132 |
| doc | 279 |
| epic | 121 |
| health | 3 |
| project | 97 |
| testcase | 137 |
| **Total** | **1060** |

All19 executable files finish, with no failures or skips. Every registered
method/path pair still has passing success assertions: **45/45 (100%) operation
coverage**. Requests across groups also exercise other groups' operations, so
operation counts must be deduplicated across the whole suite.

| Command actually run for this test-only refresh | Exit and result |
| --- | --- |
| Focused `apih --parallelism 0 <private-suite>/doc <private-suite>/config` | 0; both complete lifecycles against an owned current-source backend on19081 and the designated existing development database. |
| `make -C backend tooling-test` | 0;35 Python tests and Go suite/capture/ownership/route validator pass. |
| `make -C backend api-test` | 0;1060 requests pass; **1090/1204 (90.5316%)**, strict90% statement gate PASS. |
| `git diff --check` | 0. |

The private suite uses an owned instrumented server on19080; the running8080
server is not stopped or restarted. Root URL stays8080. Bare `apih` on that server
was not rerun in this refresh and would not measure statements. Unit/check/dependency
audit were not repeated for these YAML/documentation-only changes. The preceding
unchanged production implementation passed check and dependency audit, and measured
unit coverage at1190/1204 (98.8372%); that unit measurement remains separate.

| Production package | API covered/total | API gaps |
| --- | ---: | ---: |
| cmd/server | 77/93 | 16 |
| internal/app | 41/51 | 10 |
| internal/change | 296/315 | 19 |
| internal/config | 96/100 | 4 |
| internal/doc | 188/210 | 22 |
| internal/domain | 0/0 | 0 |
| internal/epic | 131/143 | 12 |
| internal/health | 16/21 | 5 |
| internal/project | 117/125 | 8 |
| internal/testcase | 97/105 | 8 |
| pkg/config | 23/31 | 8 |
| pkg/markdown | 8/10 | 2 |
| **Total** | **1090/1204 (90.5316%)** | **114** |

The integration result covers eight more statements than the previous573-request
campaign:1082/1204 (89.8671%), whose strict gate correctly failed. The fresh result
passes without changing the1204-statement denominator, merging unit hits, altering
production source or weakening any gate. Raw reports, source/suite hashes and
provenance remain in `.coverage/api/`; this refresh also preserves final logs and
profiles under `.coverage/verification/api-error-scenarios-20261002/`.

The initial tooling check rejected literal positive fractional/overflow probes
under its captured-target rule and an extra group filename under its fixed layout.
Those cases now use negative malformed numbers and the existing19 files; the
validator and ownership rule were not relaxed. A second tooling run still exposed
the filename issue before it was removed. Neither failed tooling check was recorded
as passing; no HTTP campaign failed during this refresh.

### Remaining HTTP error gaps

This is broader error coverage, not a claim that every possible API error is tested:

- Missing-document `/doc/delete`404 and truly nonexistent-ID
  `/doc/comment-update`404 remain unit-only. The suite exercises comment-update404
  using an owned ordinary document, but soft deletion cannot produce an absent doc.
  No guessed positive mutation IDs are used.
- Database outage503 on both health aliases, query/scan/iteration failures,
  failed FK deactivation and timed concurrent deletion after successful preflight
  remain unit-only; no fault injection or database lifecycle operations are allowed.
- Missing selected configurations and change creation without configured
  `backlog`/`brief` cannot be established through the current HTTP API without
  changing shared configuration or assigning a custom config to a project.
- Direct-service invalid-ID guards are intercepted by HTTP validation, so they are
  statement gaps rather than additional untested public input-error scenarios.
  Startup and renderer failure paths also remain outside this healthy-server suite.

The application has no authentication/session middleware, so no401/403 contract
is assumed. Arbitrary combinations of malformed input are not enumerated; the
new cases isolate the individual field conditions rather than hiding them behind
another rejection.

See the [route ledger](../agents/backend-contracts.md) and
[checkpoint](../agents/backend-refactor-checkpoint.md) for current contracts and
previous implementation validation. Historical failed schema-alignment campaigns
remain preserved in `.coverage/verification/schema-alignment-20261002/`.
