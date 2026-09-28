# APIHydra P2 measurement

Measured 2026-09-28 on `change/007-backend-project-epic-alignment`, working-tree
P2 implementation over `131e075`. The complete suite passes **100 requests**:
91 in `p2-steps.yaml`, 7 retained errors, 2 health aliases. APIHydra and the owned
instrumented server exit 0. The private PostgreSQL cluster loads unchanged
`db/init.sql`, `db/seed.sql`, then `apih-tests/fixtures.sql`, each with
ON_ERROR_STOP. It is stopped and removed after the run. No external database is
reset. Runner unit tests prove fixture ordering and abort-before-server behavior
when fixture SQL fails.

`make -C backend api-test` exits **2** (coverage recipe **1**): valid
**414/1164 = 35.5670%** production statement coverage, below the final >=90%
target. Unit coverage independently measures **853/1164 = 73.2818%**, below
strict >95%. Project and epic each have 100% unit coverage; their real server
coverage is 110/117 and 107/115 respectively. No unit or legacy counters are
merged. Structural metadata keeps unlinked pkg/db's 4 statements at zero;
internal/domain has no executable statements. Source/block audits pass.
The [checkpoint](../agents/backend-refactor-checkpoint.md) gives every package's
actual numerator/denominator, command exits and remaining failures.

Operation diagnostics are separate: **15/32** registered method/path pairs are
reached. **13/32 = 40.625%** have passing successful contracts: all 6 project,
all 5 epic, both health operations. Change get and testcase list are reached only
for errors. The [ledger](../agents/backend-contracts.md) includes all 32 routes;
P3/P4 successes remain untested or blocked by their old SQL. Removed options and
unknown routes prove 404 without adding to the registered-route denominator.

## Contracts exercised

The P2 serial workflow creates a project, captures its ID, independently gets,
updates and gets it, then does the same for a captured epic. Same-name updates
also succeed. Project delete returns 409 while the epic exists; explicit epic
delete returns 204 followed by get 404, then project delete 204/get 404. Bodies
interpolate captured IDs; no generated identity ID is assumed. Exact response
statuses and substantive read/error expectations are declared throughout.

Reserved SQL fixture IDs >=1000 cannot collide with the small dynamically
created identities in this bounded campaign. Custom config has all six arrays
different from default, including ordered custom phases/colors. Projects 1001,
1002 and 1003 select custom/default/absent slugs; default remains present during
the absent-slug 404. Details expose the stored selection and last_ref 42. Empty
project 1004 deletes; 1005 conflicts with empty epic 1101, then deletes after its
epic; 1006 conflicts with change 1201. Epic 1102 has linked change 1202 and two
testcases (one done), proving 1/2 counters and completed 50; 1103 proves zero
completion. Tied timestamps prove explicit list tie-breaking. Empty scoped
lists include an absent project. Conflict-survivor reads verify unchanged counts.
Malformed JSON, wrong field types, invalid IDs, missing IDs/parents and whitespace
names assert their 400/404 envelopes. Both former options routes assert 404.

No P2 fixture calls the still-broken P3 mutation API. P1's broken project/epic
500-create fixtures were removed. Unknown-error masking remains unit-tested.
Project/epic delete-conflict, missing details and selected-config errors now run
against real current SQL. No new project/epic document endpoint was invented.
The separate retained HTTP/SQL `TestDeletionRetainsAppendOnlyDocuments` verifies
historic/current records survive deletion; it contributes no APIHydra coverage.

## Tool capabilities and assertion limits

Installed `/home/vito/go/bin/apih` is module
`v0.1.1-0.20260907061551-c3947513e2a4`, build revision
`c3947513e2a4b948ce732dec60e090a5e83be744`, confirmed with `go version -m`.
Its exact-revision manual was read using `git show`; the local checkout HEAD is
newer and is not assumed to describe installed capabilities. `apih --help`
confirms serial invocation: `apih --parallelism 0 <private-complete-suite-copy>`.
The runner's XDG cache lives in the owned artifact directory. No Debug directive,
selection, credential, secret file or profile mixing is used. The API has no
authentication middleware.

Mutation/error files use `retries: -1`; zero inherits retries in this installed
version, whereas a negative value omits curl's retry option. Captures work in
request/expected bodies, so POST IDs need no path interpolation.

APIHydra's JSON subset matcher ignores extra object keys. Empty expected bodies
skip comparison. Thus API unit tests (`TestAPIRegisteredContracts`) independently
prove exact create key set {id}, exact read/config JSON keys, absence of obsolete
epic version, and genuinely zero-length 204 bodies. Handler/central tests prove
exact error envelopes and causes. Retained Go HTTP CRUD separately proves that
same-name updates advance modified; this is not claimed from subset comparisons.

## Remaining gaps and artifacts

Real query/scan/iteration failures and concurrent epic-create FK failure have
unit evidence, without artificial outage endpoints. Health degraded 503, startup
failures and several connector/Markdown branches remain outside this campaign.
Most change/testcase success behavior awaits P3/P4; both global coverage goals
remain unmet. No failing/blocked case was credited as a success. Full legacy
change/testcase, Docker and benchmark campaigns were not run.

Fresh ignored artifacts live under `../.coverage/api/`: runner/server/PostgreSQL
logs, source/block inventories, provenance, counters, coverage.out, report.txt
and result.json. Legacy P2 checks live separately under `.coverage/legacy-p2/`.
Run the authoritative full isolated suite from repository root with
`make -C backend api-test`; its strict coverage gate intentionally remains red
until the final target is attained.
