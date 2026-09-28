# APIHydra final 016 campaign

Final supervisor verification on 2026-09-28 includes both native recorder-cleanup
and independently reviewed emergency command fixes. All five required/separate
checks pass; unit **911/925** and APIHydra **864/925** remain unchanged. Final
source, command exits and review evidence are in the
[current checkpoint](../agents/backend-refactor-checkpoint.md#final-supervisor-verification).
Production source is unchanged from merged R6 `62d9f4f`; tooling and suite
provenance records the tested branch diff. No future squash hash is claimed.

`make -C backend api-test` passes **864/925 = 93.4054%** production statements,
above the >=90% gate. Independent unit coverage passes **911/925 = 98.4865%**,
above strict >95%. All required checks and the complete legacy HTTP/SQL suite
pass. No production code, dependency, database schema or public API changed.

## Full campaign and evidence

The authoritative command from the repository root is:

```sh
make -C backend api-test
```

Do not select the suite root with apih: phases require different database states.
A manual `apih --parallelism 0 backend/apih-tests/normal` selection against an
independently prepared owned healthy server is diagnostic only, and cannot
establish full-campaign coverage. The Make target validates all YAML before
setup, then selects the three copied subtrees explicitly, serially.

| Phase | Requests | Exit | Server PID | Database state |
| --- | --- | --- | --- | --- |
| normal | 408 | 0 | 1302702 | Original initialized/seeded cluster |
| outage | 28 | 0 | 1302702 | Confirmed stopped, no postmaster PID file |
| recovery | 22 | 0 | 1302702 | Same data directory/socket restarted with pg_ctl readiness |

All **458 requests** pass. Server binary SHA256:
`985bb47273813443fc1efd57cc8f8da535e92c16cf573876e9140bf7a51cf8da`.
One covered build, one server, one cluster initialization, one init/seed/fixture
load and one counters directory serve the campaign. No pool replacement or
HTTP warmup occurs after database restart. Both health aliases' first recovery
requests pass200. Original server shutdown exits0; final DB stop and cleanup
pass before counter conversion/reporting.

Normal, post-outage/pre-write, and post-recovery SQL postconditions all pass
with ON_ERROR_STOP. They prove exact retained rows, timestamps, absent attempted
inserts, restored testcase state, and append-only P3 documents. SQL receives no
Go coverage credit. Initial HTTP health probes only synchronize startup and can
contribute counters. No per-phase statement percentage is inferred from the one
process. Unit and legacy profiles are never merged into APIHydra data.

Fresh ignored artifacts live under `backend/.coverage/api/`:

- `suite-manifest.json`: every decoded YAML document, source hash, kind,
  executable phase/selection, stage depth and request count.
- `suite-inputs.json`: hashes of all copied files, including the port-substituted
  root and SQL; `suite/` retains the exact executed input tree.
- `campaign.json`: ordered phase command/start/end/exit/error, PID/binary hash,
  DB transitions, SQL postconditions, shutdown and cleanup results. Incomplete
  runs retain failed/not-reached evidence and remove success reports/profiles.
- `provenance.json`: HEAD/diff, backend input hashes (including untracked files),
  actual apih identity/hash, commands and campaign evidence; DB source hashes
  are in campaign evidence. `denominator.json`/`packages.json`, native counters,
  `coverage.out`, `report.txt`, `result.json` and runner/server/PG logs preserve
  the independently audited statement measurement.

The [checkpoint](../agents/backend-refactor-checkpoint.md) records exact command
exits, criterion/test mapping and factory handoff. The
[gap inventory](../agents/final-coverage-gaps.md) lists every production file,
package total and uncovered function/source block with its practical obstacle.
Implementation source hashes still match all three measured provenance records;
reporting Markdown was written after measurement.

## Operation inventory and behavioral assertions

The [contract ledger](../agents/backend-contracts.md) and route-constructor/tooling
tests retain **34/34 = 100%** successfully exercised method/path operations:
16 change, 6 project, 5 epic, 5 testcase, 2 health. Operation coverage is a
separate diagnostic, not the statement gate. No operations were omitted.

| Selection | Retained or added contract |
| --- | --- |
| normal/error-steps.yaml | Seven malformed/missing/unknown-route contracts |
| normal/health-steps.yaml | Both registered healthy aliases |
| normal/p2-steps.yaml | 91 project/config/epic assertions; original nine global-list rows remain in order with two reserved projects appended |
| normal/p3/steps.yaml | 212 change/document assertions: UUIDs, config, current/history reads, explicit mutations, FK failures, sanitization and missing/invalid cases |
| normal/p4/steps.yaml | 96 testcase lifecycle/count/validation/deletion assertions |
| outage/steps.yaml | Both503 safe degraded bodies; valid project/epic/change/documents/artifacts/testcase requests return live500 generic envelopes; direct insert/update/delete failures; procedure-preflight failures; invalid request still400 |
| recovery/steps.yaml | Both health aliases200; independent retained parents/counts/names/scenarios/document reads; update204 followed by read/restore; create201 with phase-local capture, list/count read, delete204 and empty list/count read |

Four original normal files are byte-identical after relocation. P2 changes only
its initial project-list expectation; every previous row/order/value remains.
Stage ancestry is preserved beneath normal. Fixture IDs3001/3002,3101/3102,
3201/3202,3301/3302 and3401 are independent of normal dynamic records. Existing
P2/P3 fixtures are unchanged. Explicit reserved IDs do not advance identity
sequences; the bounded campaign's generated identities remain below1000.
Captures reset per invocation; only recovery-local IDs are consumed in recovery.
No fixture reload/reset occurs between phases.

## Limits and historical evidence

Installed apih remains `v0.1.1-0.20260907061551-c3947513e2a4`, revision
`c3947513e2a4b948ce732dec60e090a5e83be744`; help and the pinned local manual were
inspected. Requests use a five-second bound; phase commands have a300-second
bound. Mutation/error files use `retries: -1`; zero would inherit retries.
Transport errors/timeouts cannot satisfy expected500. No Debug directives,
assertion retries, schema corruption, production fault hooks or hidden skipped
phases were introduced.

APIHydra compares object subsets and skips empty expected bodies. Existing
`TestAPIRegisteredContracts`, `TestChangeAPIContracts`,
`TestDocumentAPIShapeAndExplicitBooleans` and `TestTestCaseAPIExactContracts`
remain authoritative for exact ID-only201 and truly empty204 contracts.
Separate preflight/query/procedure races and append-only polymorphic document
semantics remain as documented in the ledger; no atomicity claim was added.

The first016 real run passed normal/outage but failed101 on a timezone-specific
HTTP timestamp expectation. It produced no valid profile. That assertion was
replaced with SQL timestamp equality before recovery writes; all subsequent
complete runs pass. Historical R6 measurement was837/925; the additional27
covered statements are genuine outage/error/health behavior. The61 API-uncovered
statements and14 unit-uncovered main statements are listed exactly in the gap
report. Both targets pass; no required scenario remains failed, skipped or blocked.
