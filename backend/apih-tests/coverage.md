# APIHydra R3 measurement

Measured 2026-09-28 on `change/012-backend-startup-boundaries`, published
implementation `8bc82d4` over merged dev `4c25d70`; native review passed clean.
**All 408 requests pass**: 96 P4 in
[p4/steps.yaml](p4/steps.yaml), 212 P3, 91 P2, 7 errors and 2 health aliases.
APIHydra and the owned instrumented server exit0. The isolated runner loads
unchanged root init/seed and [fixtures.sql](fixtures.sql), then stops/removes
its own private PostgreSQL cluster and server. No external database is reset.

`make -C backend api-test` exits **0**: **851/939 = 90.6283%** production
statement coverage, passing >=90%. Independent unit coverage is
**924/939 = 98.4026%**, passing strict >95%. Testcase is 108/108 unit and 99/108
API statements. Unit and legacy profiles are never merged into integration.
R1 previously removed the audited unused pkg/db wrapper; 011 added seven real
validation-cause statements and R3 adds four composition-boundary statements.
The production denominator is 939; domain has no executable statements. No
retained production package is excluded. All router/lifecycle statements are
unit-covered; process-only main statements remain honestly uncovered.
The [checkpoint](../agents/backend-refactor-checkpoint.md) lists every package,
command exit, named acceptance evidence and remaining baseline lint failures.

## Operation inventory and scenarios

The [ledger](../agents/backend-contracts.md) inventories **34 method/path pairs**,
recalculated from route registration by a passing tooling test.
**34/34 = 100%** have passing successful contracts: 16 change, 6 project,
5 epic, 5 testcase, 2 health. Negative-only scenarios do not earn successful
operation credit. Removed options/move/unknown routes are404 checks, not extra
registered operations. Operation coverage is separate from statement coverage.

P4 creates its own project/epic/change through ID-only201, associates via204,
and independently lists cases and reads counters. Two captured testcase IDs
prove trimmed scenarios, default false and ID ordering. Toggle/edit/same-value
writes, omitted/null false, deletes and repeated-delete404 are followed by
independent state/count reads: 0/0/0, 0/2/0, 1/2/50, 2/2/100, 1/2/50, 0/1/0,
and delete-last0/0/0. Parent deletion409 preserves children and current docs;
after explicit child deletion, parent delete204 succeeds, then change
get/documents/set-document and testcase list/create return404. All applicable
invalid IDs, blank/null scenario, malformed/wrong-type payloads, representable
large missing IDs and missing parent/case paths assert stable status/envelopes.
Retries remain -1 to prevent mutation/error replay. No mutation side data is
used as a read.

Every retained change operation and the two explicit document operations have
successful contracts with substantive reads. Each successful mutation is followed
by a separate read. The campaign covers generated and supplied v4 UUIDs,
duplicate UUID409, trimmed create, ID-only create status, title procedure
whitespace normalization, phase/epic/open/types/pr-url updates, same-value direct
updates, ordered type filtering/clearing, specialized brief/spec/pr and generic
configured documents, repeated same-body writes, raw+sanitized reads, bulk
artifacts in input order, empty/missing reads, and direct deletion/FK409.
Malformed JSON, wrong field types, negative/zero IDs, absent targets, empty/null
content, omitted/null/wrong-type booleans, invalid URLs, cross-project/missing
epics, unknown phases/kinds and missing selected configuration are exercised.

P2's six original fixture projects, config arrays and relationships remain
unchanged. The exact initial global list expectation is extended with the three
P3 fixture projects, retaining all original entries and assertions. P3 projects
2001/2002/2003 select default/custom/missing config. Their independent changes
2201–2207 support writes, no-config reads/writes, deletion, empty docs, wide
counts, and malformed duplicate-current artifacts. Epic2101 matches project2001;
2102 intentionally belongs to2002. Testcase IDs230001–280000 give change2206
40000 done/50000 total and completed80, with actual FK conflict on deletion.
Docs2401–2407 cover retained history, missing-config rendering, duplicate current
spec rows, a wrong ref_table PR and a noncurrent PR. Dynamic identities remain
small and are captured; no generated ID is assumed.

Current doc ordering is doc_type,id. Historical records are absent from current
reads; bulk artifacts select latest current rows deterministically if duplicate
current data exists. Sanitization leaves raw bodies intact. After deleting a
parent, get/documents/set return404 and bulk artifacts omit it. Missing selected
configuration blocks config-dependent writes/create but not plain reads or
unrelated title/open/pr-url/epic/delete operations. No fallback is credited.

## Separate SQL/HTTP evidence and tool limits

`TestChangeDeletionRetainsAppendOnlyDocuments` runs in the complete separate
owned legacy campaign and proves identical writes append three increasing IDs,
two historic/one current. A testcase created through the new API blocks deletion
without removing child/current documents. After API child deletion and parent
delete204, SQL confirms parent/child absence and exact retained document
IDs/bodies/current flags. Public writes cannot resurrect removed-parent docs.
No equivalent runner SQL hook existed in merged P3, so this useful SQL evidence
was retained and adapted rather than discarded.

`TestChangeIdentityOrderingAndSameValueTimestamps` still proves UUIDv7 and
change timestamp ordering; `TestTestCaseCurrentStateAndSameValueTimestamps`
adds testcase same-value modified advancement with preserved created, unrelated
rows, parent timestamps and independently read counters. All retained
project/epic/change/health/testcase HTTP and SQL tests pass in the complete
R3 `make -C backend legacy-api-test` campaign, independently of APIHydra.
Separate `.coverage/legacy/` counters receive
**zero APIHydra coverage credit**. The ledger records why nonexistent testcase
history/version/procedure/cascade and composite-response assertions were removed,
and maps every retained guarantee to current tests.

Installed `/home/vito/go/bin/apih` is module
`v0.1.1-0.20260907061551-c3947513e2a4`, revision
`c3947513e2a4b948ce732dec60e090a5e83be744`. `apih --help` and that exact revision's
manual were inspected locally with git show; the newer checkout is not authority
for syntax. Serial invocation is `apih --parallelism 0 <private-suite-copy>`.
Mutations/errors use `retries: -1` to omit retries; zero inherits retry defaults.
There are no credentials/auth middleware, Debug breakpoints or partial selections.

APIHydra's subset comparison ignores extra object keys; an empty expected body
skips comparison. Thus it cannot prove exact {id} or truly empty204 bodies.
`TestTestCaseAPIExactContracts`, `TestTestCaseListExactCurrentFields`,
`TestChangeAPIContracts` and `TestDocumentAPIShapeAndExplicitBooleans` pin exact
keys/nulls/empty bodies. Tests separately preserve bind/validation/PG causes.
Captured IDs and explicit expected_status/body/types use supported syntax.

## Remaining gaps and rerun

Query/scan/iteration and Exec failures, entropy failure, concurrent FK/parent
races, degraded health and startup/config/Markdown error branches remain gaps;
no fabricated HTTP triggers or exclusions were introduced. Testcase's nine API
uncovered statements are DB failure arms, all unit tested. Both numerical goals
pass, but overall verification still reports 11 unrelated baseline lint issues.
The two-read testcase existence/list race is documented without an atomic claim.

First P4 API attempt failed101 because the new file lacked JSON Content-Type;
that header was fixed and the entire campaign rerun with unchanged assertions.
Only completed successful runs establish the counts above. No scenarios remain
failed/skipped/blocked; no outage campaign was added.

Fresh ignored profiles, provenance, source/block inventories, result.json and
runner/server/PostgreSQL logs live under `backend/.coverage/api/`; unit and
legacy evidence stay separate. Rerun: `make -C backend api-test`.
