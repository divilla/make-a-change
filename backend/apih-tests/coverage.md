# APIHydra P3 measurement

Measured 2026-09-28 on `change/008-backend-change-doc-alignment`, working-tree
implementation over `8722e1d`. **All 312 requests pass**: 212 P3 requests in
[p3/steps.yaml](p3/steps.yaml), 91 retained P2, 7 errors and 2 health aliases.
P3 is a child stage so P2 completes before P3 mutates its independent fixtures.
APIHydra and the owned instrumented server exit 0. The isolated runner loads
unchanged root init/seed followed by [fixtures.sql](fixtures.sql), all with
ON_ERROR_STOP, then stops/removes its own private PostgreSQL cluster and server.
No external database is reset.

`make -C backend api-test` exits **2** (coverage recipe **1**): valid
**766/1021 = 75.0245%** production statement coverage, below the final >=90%
target. Independent unit coverage is **933/1021 = 91.3810%**, below strict >95%.
Change has 405/405 unit statements and 376/405 APIHydra statements. Neither unit
nor legacy profiles are merged into integration. Unlinked pkg/db's 4 statements
remain structural zero; domain has zero executable statements. The
[checkpoint](../agents/backend-refactor-checkpoint.md) lists every package,
command exit, source gaps and named criterion evidence.

## Operation inventory and scenarios

The [ledger](../agents/backend-contracts.md) retains all **34 method/path pairs**.
**29/34 = 85.2941%** have passing successful contracts: all 16 change routes,
6 project, 5 epic, 2 health. **30/34** are reached including testcase list's
error-only scenario. Five testcase successful operations remain blocked by old
SQL pending P4 and stay in the denominator. Removed options/unknown routes are
404 checks, not additional registered operations. Operation coverage is a
separate diagnostic and never substitutes for statement coverage.

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

`TestChangeDeletionRetainsAppendOnlyDocuments` uses a separate owned campaign
and proves identical writes create three distinct IDs, two historic/one current;
a testcase FK prevents deletion without removing the testcase; after explicit
child deletion, change deletion retains every doc ID/body/current flag. A public
set on the removed parent cannot resurrect documents. The separate
`TestChangeIdentityOrderingAndSameValueTimestamps` proves generated UUIDv7,
modified ordering and same-value title/open/types/pr-url timestamp advancement.
These checks and retained project/epic HTTP/SQL tests pass but contribute **zero**
APIHydra coverage. The seven-test campaign lives in `.coverage/legacy-p3/`.
Obsolete change version/history/inline-testcase assertions were replaced only
after the corresponding retained contracts passed APIHydra. Testcase's own
legacy tests are deferred to P4 and were not claimed passing.

Installed `/home/vito/go/bin/apih` is module
`v0.1.1-0.20260907061551-c3947513e2a4`, revision
`c3947513e2a4b948ce732dec60e090a5e83be744`. `apih --help` and that exact revision's
manual were inspected locally with git show; the newer checkout is not authority
for syntax. Serial invocation is `apih --parallelism 0 <private-suite-copy>`.
Mutations/errors use `retries: -1` to omit retries; zero inherits retry defaults.
There are no credentials/auth middleware, Debug breakpoints or partial selections.

APIHydra's subset comparison ignores extra object keys; an empty expected body
skips comparison. Thus it cannot prove exact {id} or truly empty204 bodies.
`TestChangeAPIContracts` and `TestDocumentAPIShapeAndExplicitBooleans` pin exact
keys/nulls/empty bodies. Tests separately preserve bind/validation/PG causes.
Captured IDs and explicit expected_status/body/types use supported syntax.

## Remaining gaps and rerun

Query/scan/iteration failures, entropy failure, concurrent FK/parent failures,
legacy adapter rendering, degraded health and startup/config/Markdown error
branches have no fabricated HTTP trigger. Most deferred testcase behavior remains
unexecuted. Procedure preflight/CALL and epic association checks are not atomic;
see the ledger for exact races. Both final coverage goals remain unmet.

Fresh ignored profiles, provenance, source/block inventories, result.json and
runner/server/PostgreSQL logs live under `backend/.coverage/api/`; unit and
legacy evidence remain in separate directories. Rerun from repository root:
`make -C backend api-test`. Its strict gate remains a failure until >=90% is
actually measured. No failing/blocked scenario receives successful coverage credit.
